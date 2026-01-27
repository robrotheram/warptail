import { createRootRoute, Outlet, useRouterState } from '@tanstack/react-router'
import { HeaderNav, SideNav } from "../Nav"
import { AuthProvider, useAuth } from '@/context/AuthContext';
import { ConfigProvider } from '@/context/ConfigContext';
import { Banner } from '@/components/banner';

const AuthenticatedLayout = ({ children }: { children: React.ReactNode }) => {
    const { isAuthenticated } = useAuth();
    
    if (!isAuthenticated) {
        // Show a minimal layout for unauthenticated users
        return (
            <div className="flex min-h-screen w-full items-center justify-center">
                {children}
            </div>
        );
    }

    return (
        <div className="flex min-h-screen w-full">
            <SideNav />
            <div className="flex flex-1 flex-col sm:pl-14">
                <HeaderNav />
                <div className="flex flex-1 flex-col">
                    <Banner />
                    <main className="flex-1 p-4 sm:px-6 sm:py-6">
                        {children}
                    </main>
                </div>
                
            </div>
        </div>
    );
};

const RootLayout = () => {
    const routerState = useRouterState();
    const isLoginPage = routerState.location.pathname === '/login';
    const isPasswordResetPage = routerState.location.pathname === '/password-reset';

    // For login page and password reset page, render with AuthProvider but without the full layout
    if (isLoginPage || isPasswordResetPage) {
        return (
            <ConfigProvider>
                <AuthProvider>
                    <div className="flex min-h-screen w-full items-center justify-center">
                        <Outlet />
                    </div>
                </AuthProvider>
            </ConfigProvider>
        );
    }

    // For all other pages, use auth-protected layout
    return (
        <ConfigProvider>
            <AuthProvider>
                <AuthenticatedLayout>
                    <Outlet />
                </AuthenticatedLayout>
            </AuthProvider>
        </ConfigProvider>
    );
};

export const Route = createRootRoute({
    component: RootLayout,
});


