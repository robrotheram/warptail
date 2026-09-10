import { describe, expect, it } from 'vitest';
import { safeRedirect } from './redirect';

const origin = 'https://warp.example.com';
describe('safeRedirect', () => {
  it.each([
    '//evil.example', '/\\evil.example', '/%5cevil.example', '%2f%2fevil.example',
    'https://evil.example/path', 'https://warp.example.com@evil.example',
    'javascript:alert(1)', 'data:text/html,test', 'https://warp.example.com:444/',
    '/login', '/auth/logout', '/password-reset', '%invalid', '\n//evil.example',
  ])('rejects unsafe destination %s', value => expect(safeRedirect(value, origin)).toBeNull());
  it('preserves a valid local destination, query and fragment', () => {
    expect(safeRedirect('/routes/example?tab=traffic#chart', origin)).toBe('/routes/example?tab=traffic#chart');
    expect(safeRedirect(`${origin}/settings?tab=logs`, origin)).toBe('/settings?tab=logs');
  });
  it('removes tokens from redirect destinations', () => {
    expect(safeRedirect('/routes?token=secret&tab=details#chart', origin)).toBe('/routes?tab=details#chart');
  });
});
