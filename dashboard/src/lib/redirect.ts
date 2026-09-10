/** Accept only browser-normalized, same-origin destinations. Never forward credentials. */
export function safeRedirect(value: unknown, origin = window.location.origin): string | null {
  if (typeof value !== 'string' || !value.trim()) return null;
  try {
    const decoded = decodeURIComponent(value);
    if (/[\\\s]/.test(value) || /[\\\s]/.test(decoded) || decoded.startsWith('//')) return null;
    const url = new URL(value, origin);
    if (url.origin !== origin || !['http:', 'https:'].includes(url.protocol) || url.username || url.password) return null;
    if (/^\/(auth(?:\/|$)|login\/?$|password-reset\/?$)/.test(url.pathname)) return null;
    url.searchParams.delete('token');
    return url.pathname + url.search + url.hash;
  } catch {
    return null;
  }
}
