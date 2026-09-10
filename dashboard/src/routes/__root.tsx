import { createRootRoute, Link, Outlet, useRouterState } from '@tanstack/react-router';
import { HeaderNav, SideNav } from '../Nav';
import { AuthProvider, useAuth } from '@/context/AuthContext';
import { ConfigProvider } from '@/context/ConfigContext';
import { Banner } from '@/components/banner';
import { Toaster } from 'sonner';
import { Button } from '@/components/ui/button';

function Layout() {
  const { isAuthenticated, requiresPasswordReset } = useAuth();
  const pathname = useRouterState({ select: state => state.location.pathname });
  const minimal = !isAuthenticated || requiresPasswordReset || pathname === '/login' || pathname === '/password-reset';
  if (minimal) return <main id="main-content" className="flex min-h-svh w-full items-center justify-center p-4"><Outlet /></main>;
  return <div className="min-h-svh w-full">
    <a href="#main-content" className="sr-only focus:not-sr-only focus:fixed focus:z-50 focus:rounded focus:bg-primary focus:p-3 focus:text-primary-foreground">Skip to content</a>
    <SideNav />
    <div className="min-w-0 sm:pl-56">
      <HeaderNav />
      <Banner />
      <main id="main-content" tabIndex={-1} className="mx-auto max-w-screen-2xl p-4 sm:p-8"><Outlet /></main>
    </div>
  </div>;
}

function RootLayout() {
  return <ConfigProvider><AuthProvider><Layout /><Toaster theme="dark" richColors closeButton /></AuthProvider></ConfigProvider>;
}

export const Route = createRootRoute({
  component: RootLayout,
  notFoundComponent: () => <div className="space-y-4 p-8 text-center"><h1 className="text-2xl font-semibold">Page not found</h1><p className="text-muted-foreground">This page may have moved or no longer exists.</p><Button asChild><Link to="/">Back to services</Link></Button></div>,
  errorComponent: ({ reset }) => <div role="alert" className="space-y-4 p-8 text-center"><h1 className="text-2xl font-semibold">Something went wrong</h1><p className="text-muted-foreground">Try loading the page again.</p><Button onClick={reset}>Try again</Button></div>,
});
