import { test, expect, Page } from '@playwright/test';

const profile = { id: 'admin-1', name: 'Alex Morgan', email: 'alex@example.com', type: 'internal', role: 'admin', password_reset: false };
const service = { id: 'docs', name: 'Documentation', enabled: true, latency: 1200000, routes: [{ type: 'https', domain: 'docs.example.com', private: true, bot_protect: false, machine: { address: '100.64.0.1', port: 3000 }, status: 'Running' }] };

async function mockAPI(page: Page, signedIn = true) {
  if (signedIn) await page.addInitScript(() => sessionStorage.setItem('token', 'test-token'));
  await page.route('**/config', route => route.fulfill({ json: { read_only: false, auth_type: 'basic', auth_name: '', site_name: 'WarpTail' } }));
  await page.route('**/auth/profile', route => route.fulfill({ json: profile }));
  await page.route('**/auth/logout', route => route.fulfill({ status: 200, body: 'Logged out' }));
  await page.route('**/api/**', route => {
    const path = new URL(route.request().url()).pathname;
    if (path === '/api/services') return route.fulfill({ json: [service, { ...service, id: 'metrics', name: 'Metrics', enabled: false }] });
    if (path === '/api/services/docs') return route.fulfill({ json: service });
    if (path === '/api/settings/tailscale/status') return route.fulfill({ json: { version: '1.90.0', hostname: 'warptail', state: 'Running', key_expiry: null, nodes: [] } });
    if (path === '/api/settings/tailscale') return route.fulfill({ json: { Hostname: 'warptail', AuthKey: 'tskey-secret-value' } });
    if (path === '/api/settings/logs') return route.fulfill({ json: ['Request completed', '<img src=x onerror=alert(1)>'] });
    if (path === '/api/user') return route.fulfill({ json: [profile] });
    return route.fulfill({ json: [] });
  });
}

test('checks the session before private requests and supports searching', async ({ page }) => {
  await mockAPI(page);
  const errors: string[] = [];
  page.on('pageerror', error => errors.push(error.message));
  let release!: () => void;
  const gate = new Promise<void>(resolve => { release = resolve; });
  await page.route('**/auth/profile', async route => { await gate; await route.fulfill({ json: profile }); });
  const privateRequests: string[] = [];
  page.on('request', request => { if (request.url().includes('/api/')) privateRequests.push(request.url()); });
  await page.goto('/routes');
  await expect(page.getByRole('status')).toContainText('Checking your session');
  expect(privateRequests).toEqual([]);
  release();
  await expect(page.getByRole('heading', { name: 'Services', exact: true })).toBeVisible();
  await page.getByRole('textbox', { name: 'Search services' }).fill('doc');
  await expect(page.getByRole('link', { name: 'Documentation' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Metrics', exact: true })).toHaveCount(0);
  expect(errors).toEqual([]);
  await page.screenshot({ path: 'test-results/services-desktop.png', fullPage: true });
});

test('a failed save keeps the draft and added routes remain independent', async ({ page }) => {
  await mockAPI(page);
  await page.route('**/api/services/docs', route => route.request().method() === 'PUT' ? route.fulfill({ status: 500, body: 'Save failed' }) : route.fulfill({ json: service }));
  await page.goto('/routes/docs/edit');
  await page.getByRole('button', { name: 'New Route' }).click();
  await page.getByRole('button', { name: 'New Route' }).click();
  const domains = page.getByRole('textbox', { name: 'Domain', exact: true });
  await expect(domains).toHaveCount(3);
  await domains.nth(0).fill('first.example.com');
  await domains.nth(1).fill('second.example.com');
  await page.getByRole('button', { name: 'Remove route', exact: true }).nth(0).click();
  await expect(domains).toHaveCount(2);
  await expect(domains.nth(0)).toHaveValue('second.example.com');
  await page.getByRole('textbox', { name: 'Service Name:' }).fill('Draft name');
  await page.getByRole('button', { name: 'Save changes' }).click();
  await expect(page.getByRole('alert')).toContainText('Unable to update service');
  await expect(page).toHaveURL(/\/routes\/docs\/edit$/);
  await expect(page.getByRole('textbox', { name: 'Service Name:' })).toHaveValue('Draft name');
  await page.getByRole('link', { name: 'Services', exact: true }).click();
  await expect(page.getByRole('alertdialog')).toContainText('Discard unsaved changes?');
  await page.getByRole('button', { name: 'Keep editing' }).click();
  await expect(page).toHaveURL(/\/routes\/docs\/edit$/);
});

test('password reset waits for the updated profile before showing services', async ({ page }) => {
  await mockAPI(page);
  let reset = true;
  await page.route('**/auth/profile', async route => {
    if (route.request().method() === 'POST') reset = false;
    await route.fulfill({ json: { ...profile, password_reset: reset } });
  });
  await page.goto('/routes');
  await expect(page).toHaveURL(/password-reset/);
  await page.locator('#password').fill('NewPassword123!');
  await page.locator('#confirmPassword').fill('NewPassword123!');
  await page.getByRole('button', { name: /Update Password|Reset Password|Change Password/ }).click();
  await expect(page.getByRole('heading', { name: 'Services', exact: true })).toBeVisible();
  await expect(page).not.toHaveURL(/password-reset/);
});

test('settings mask the auth key and tabs follow browser history', async ({ page }) => {
  await mockAPI(page);
  await page.goto('/settings?tab=status');
  await expect(page.getByRole('switch')).toBeChecked();
  await expect(page.locator('body')).not.toContainText('tskey-secret-value');
  await expect(page.getByLabel('Tailscale auth key')).toHaveAttribute('type', 'password');
  await page.getByRole('button', { name: 'Edit', exact: true }).click();
  await expect(page.getByLabel('Tailscale auth key')).toHaveValue('tskey-secret-value');
  await page.getByRole('button', { name: 'Cancel', exact: true }).click();
  await page.getByRole('tab', { name: 'Logs', exact: true }).click();
  await expect(page).toHaveURL(/tab=logs/);
  await page.goBack();
  await expect(page.getByRole('tab', { name: 'Settings', exact: true })).toHaveAttribute('data-state', 'active');
});

test('mobile navigation closes after selecting a page and logout returns to sign in', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await mockAPI(page);
  await page.goto('/routes');
  await page.getByRole('button', { name: 'Open navigation' }).click();
  await page.getByRole('dialog').getByRole('link', { name: 'Settings', exact: true }).click();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await expect(page.getByRole('heading', { name: 'TailScale Status' })).toBeVisible();
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
  await page.screenshot({ path: 'test-results/settings-mobile.png', fullPage: true });
  await page.getByRole('button', { name: 'Open navigation' }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Sign out', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Sign In', exact: true })).toBeVisible();
  expect(await page.evaluate(() => sessionStorage.getItem('token'))).toBeNull();
});

test('login preserves a deep link and never adds credentials to the destination URL', async ({ page }) => {
  await mockAPI(page, false);
  await page.route('**/auth/login', route => route.fulfill({ json: { authorization_token: 'new-login-token', role: 'admin' } }));
  await page.goto('/settings?tab=logs');
  await expect(page.getByRole('button', { name: 'Sign In', exact: true })).toBeVisible();
  await page.getByRole('textbox', { name: 'Username or email' }).fill('alex@example.com');
  await page.getByLabel('Password', { exact: true }).fill('GoodPassword1!');
  await page.getByRole('button', { name: 'Sign In', exact: true }).click();
  await expect(page.getByRole('tab', { name: 'Logs', exact: true })).toHaveAttribute('data-state', 'active');
  await expect(page).toHaveURL(/\/settings\?tab=logs$/);
  expect(page.url()).not.toContain('token');
});

test('OIDC callback credentials are consumed once and removed from browser history', async ({ page }) => {
  await mockAPI(page, false);
  await page.goto('/login?token=callback-token&next=%2Fsettings%3Ftab%3Dstatus');
  await expect(page.getByRole('tab', { name: 'Settings', exact: true })).toHaveAttribute('data-state', 'active');
  expect(page.url()).not.toContain('token');
  expect(await page.evaluate(() => sessionStorage.getItem('token'))).toBe('callback-token');
});

test('a forbidden request does not sign the user out; an expired session does', async ({ page }) => {
  await mockAPI(page);
  let status = 403;
  await page.route('**/api/user', route => route.fulfill({ status, body: 'Denied' }));
  await page.goto('/users');
  await expect(page.getByRole('alert')).toContainText('You do not have permission');
  expect(await page.evaluate(() => sessionStorage.getItem('token'))).toBe('test-token');
  status = 401;
  await page.getByRole('button', { name: 'Try again' }).click();
  await expect(page.getByRole('button', { name: 'Sign In', exact: true })).toBeVisible();
  expect(await page.evaluate(() => sessionStorage.getItem('token'))).toBeNull();
});

test('a successful save navigates only after the server response', async ({ page }) => {
  await mockAPI(page);
  let finish!: () => void;
  const gate = new Promise<void>(resolve => { finish = resolve; });
  let saved = service;
  await page.route('**/api/services/docs', async route => {
    if (route.request().method() === 'PUT') {
      saved = route.request().postDataJSON();
      await gate;
    }
    await route.fulfill({ json: saved });
  });
  await page.goto('/routes/docs/edit');
  await page.getByRole('textbox', { name: 'Service Name:' }).fill('Updated documentation');
  await page.getByRole('button', { name: 'Save changes' }).click();
  await expect(page.getByRole('button', { name: 'Saving…' })).toBeDisabled();
  await expect(page).toHaveURL(/\/edit$/);
  finish();
  await expect(page).toHaveURL(/\/routes\/docs$/);
  await expect(page.getByRole('heading', { name: 'Updated documentation' })).toBeVisible();
});

test('Tailscale authentication clears the warning without reloading the dashboard', async ({ page }) => {
  await mockAPI(page);
  let state = 'NeedsLogin';
  let statusRequests = 0;
  let documentRequests = 0;
  page.on('request', request => {
    if (request.isNavigationRequest() && request.frame() === page.mainFrame()) documentRequests++;
  });
  await page.route('**/api/settings/tailscale/status', route => {
    statusRequests++;
    return route.fulfill({ json: { version: '1.90.0', hostname: 'warptail', state, key_expiry: null, nodes: [], auth_url: 'https://login.tailscale.com/a/test' } });
  });
  await page.goto('/routes');
  const warning = page.getByRole('alert').filter({ hasText: 'Tailscale Issue' });
  await expect(warning).toBeVisible();
  await page.getByRole('textbox', { name: 'Search services' }).fill('doc');

  // Returning while the cached response is fresh must still recheck the status.
  const beforeReturn = statusRequests;
  await page.evaluate(() => window.dispatchEvent(new Event('visibilitychange')));
  await expect.poll(() => statusRequests, { timeout: 1500 }).toBeGreaterThan(beforeReturn);
  await expect(warning).toBeVisible();

  // The backend may finish authenticating after the user has already returned.
  state = 'Starting';
  await expect(warning).toHaveCount(0, { timeout: 5000 });
  await expect(page.getByRole('textbox', { name: 'Search services' })).toHaveValue('doc');
  state = 'Running';
  await page.getByRole('link', { name: 'Settings', exact: true }).click();
  await expect(page.getByText('Running', { exact: true })).toBeVisible({ timeout: 5000 });
  await expect(warning).toHaveCount(0);
  const settledRequests = statusRequests;
  await page.clock.install();
  await page.clock.runFor(6000);
  expect(statusRequests).toBe(settledRequests);
  await page.getByRole('link', { name: 'Services', exact: true }).click();
  await expect(warning).toHaveCount(0);
  expect(documentRequests).toBe(1);
});
