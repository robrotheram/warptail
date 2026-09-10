import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { renderHook, act, waitFor, cleanup } from '@testing-library/react';
import { AuthProvider, useAuth } from './AuthContext';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode } from 'react';
import { getProfile, logout, token, Role } from '@/lib/api';
import { AxiosError } from 'axios';

vi.mock('@/lib/api', async importOriginal => ({
  ...await importOriginal<typeof import('@/lib/api')>(),
  getProfile: vi.fn(),
  logout: vi.fn(),
}));

const user = { id: '1', name: 'Test User', email: 'test@example.com', type: 'internal', role: Role.ADMIN };
const deferred = <T,>() => {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(done => { resolve = done; });
  return { promise, resolve };
};

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}><AuthProvider>{children}</AuthProvider></QueryClientProvider>;
  return { client, ...renderHook(() => useAuth(), { wrapper }) };
}

beforeEach(() => {
  token.remove();
  vi.mocked(getProfile).mockReset().mockResolvedValue(user);
  vi.mocked(logout).mockReset().mockResolvedValue(undefined);
});
afterEach(() => { cleanup(); token.remove(); });

describe('authentication transitions', () => {
  it('does not request private data without a session', () => {
    const { result } = setup();
    expect(result.current.isAuthenticated).toBe(false);
    expect(getProfile).not.toHaveBeenCalled();
  });

  it('keeps auth pending until the stored session is verified', async () => {
    token.set('saved-token');
    const profile = deferred<typeof user>();
    vi.mocked(getProfile).mockReturnValue(profile.promise);
    const { result } = setup();
    expect(result.current.isLoading).toBe(true);
    expect(result.current.isAuthenticated).toBe(false);
    await act(async () => { profile.resolve(user); });
    await waitFor(() => expect(result.current.isAuthenticated).toBe(true));
  });

  it('cancels old requests and clears private caches when switching accounts', async () => {
    token.set('account-a');
    const { client, result } = setup();
    await waitFor(() => expect(result.current.isAuthenticated).toBe(true));
    client.setQueryData(['userList'], [user]);
    client.setQueryData(['logs', 'access'], ['private log']);
    client.setQueryData(['config'], { site_name: 'WarpTail' });
    const pending = deferred<string[]>();
    let signal: AbortSignal | undefined;
    const fetch = client.fetchQuery({ queryKey: ['slow'], queryFn: context => { signal = context.signal; return pending.promise; } }).catch(() => undefined);
    vi.mocked(getProfile).mockResolvedValue({ ...user, id: '2', name: 'Second account' });
    await act(async () => { await result.current.login('account-b'); });
    expect(signal?.aborted).toBe(true);
    expect(client.getQueryData(['userList'])).toBeUndefined();
    expect(client.getQueryData(['logs', 'access'])).toBeUndefined();
    expect(client.getQueryData(['config'])).toBeDefined();
    pending.resolve(['late private data']);
    await fetch;
    await waitFor(() => expect(result.current.user?.id).toBe('2'));
    expect(client.getQueryData(['slow'])).toBeUndefined();
    expect(JSON.stringify(client.getQueryCache().getAll().map(query => query.queryKey))).not.toContain('account-b');
  });

  it('ends the server session and clears all private data on logout', async () => {
    token.set('active-token');
    const { result, client } = setup();
    await waitFor(() => expect(result.current.isAuthenticated).toBe(true));
    client.setQueryData(['repoData'], ['private service']);
    await act(async () => { await result.current.logout(); });
    expect(logout).toHaveBeenCalledOnce();
    expect(token.get()).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
    expect(client.getQueryData(['repoData'])).toBeUndefined();
  });

  it('keeps a failed logout recoverable instead of claiming success', async () => {
    token.set('active-token');
    vi.mocked(logout).mockRejectedValue(new Error('Offline'));
    const { result } = setup();
    await waitFor(() => expect(result.current.isAuthenticated).toBe(true));
    await act(async () => { await result.current.logout(); });
    expect(result.current.isLoggingOut).toBe(false);
    expect(result.current.isAuthenticated).toBe(true);
    expect(token.get()).toBe('active-token');
  });

  it('retains credentials during a network failure and can retry', async () => {
    token.set('active-token');
    vi.mocked(getProfile).mockRejectedValueOnce(new Error('Offline'));
    const { result } = setup();
    await waitFor(() => expect(result.current.error).toBeTruthy());
    expect(token.get()).toBe('active-token');
    act(() => result.current.retry());
    await waitFor(() => expect(result.current.isAuthenticated).toBe(true));
  });

  it('clears an invalid session after a profile 401', async () => {
    token.set('invalid-token');
    vi.mocked(getProfile).mockRejectedValue(Object.assign(new AxiosError('Unauthorized'), { response: { status: 401 } }));
    const { result } = setup();
    await waitFor(() => expect(token.get()).toBeNull());
    expect(result.current.isAuthenticated).toBe(false);
  });
});
