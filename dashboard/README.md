# WarpTail dashboard

React 19, TanStack Router and Query, Vite 8, and TypeScript. The dashboard is embedded in the Go server for production.

## Development

Use Node.js 24 LTS (minimum 22.12):

```sh
cd dashboard
npm ci
npm run dev
```

Vite listens on `http://127.0.0.1:5173`. Requests to `/api`, `/auth`, and `/config` are proxied to `http://localhost:8001`, so development uses the same-origin request and cookie behavior as production. Set `WARPTAIL_API_URL` to use another backend. For local OIDC testing, configure the authentication base URL and provider callback for the development origin.

## Validation

```sh
npm run build
npm run lint
npm run test:run
npx playwright install chromium
npm run test:e2e
npm audit
```

Browser tests start Vite automatically and intercept API requests with isolated fixtures. They cover session loading, login/deep links, callback token cleanup, expiry versus permission errors, password reset, save failures/success, route identity, unsaved changes, secret masking, settings history, and mobile navigation. They do not connect to a real tailnet or identity provider.

Run `go test ./pkg/auth ./pkg/api` from the repository root for the server authentication checks. Deploy the dashboard and Go server together: profile editing now uses `POST /auth/profile`, and logout uses `POST /auth/logout`.

## Behavior and maintenance

- The auth provider remains mounted across page changes. Protected queries start after the profile resolves, and private queries and mutations are cleared when the account changes or signs out. Late responses are rejected if they belong to an earlier session. Successful logout also notifies other tabs.
- The existing bearer-token protocol is retained. Tokens live in session storage, are excluded from query keys, and are removed from callback URLs before rendering. They are still accessible to JavaScript; this is not a migration to server-side revocable sessions. HTTPS deployments set Secure, HttpOnly, SameSite=Lax cookies.
- The server enforces password resets, derives profile identity/role from the authenticated user, rejects cross-origin writes, and restricts OIDC return URLs to the dashboard or configured private proxy origins. Profile and authenticated API responses are not cacheable.
- Services, charts, and forms are split into route chunks. Polling pauses in hidden tabs. Log history retains the latest 1,000 lines and does not force scrolling when reading older entries. Access/error logs append batches because the current backend drains these buffers; server logs replace snapshots.
- Tailwind remains on its latest 3.4 release, paired with tailwind-merge 2.6, to preserve the existing component styling API. TypeScript remains on 5.9, compatible with the current ESLint TypeScript tooling. Other UI/runtime and build dependencies were refreshed, including the calendar, resize panel, and chart major-version adaptations.

The generated route tree is maintained by the Vite router plugin. Commit `package-lock.json` with dependency changes and use `npm ci` for reproducible installs.
