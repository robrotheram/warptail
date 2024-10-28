import { createRootRoute, Outlet } from '@tanstack/react-router'
import { HeaderNav, SideNav } from "../Nav"
import { useAuth } from '@/context/AuthContext';
import { LoginPage } from '@/LoginPage';


export const Route = createRootRoute({
    component: () => {
        const { isAuthenticated } = useAuth();
        if(!isAuthenticated) return <LoginPage />
        return <>
            <div className="flex min-h-screen w-full">
                <SideNav />
                <div className="flex flex-1 flex-col sm:gap-4 sm:py-4 sm:pl-14">
                    <HeaderNav />
                    <main className="flex-1 p-4 sm:px-6 sm:py-0">
                        <Outlet />
                    </main>
                </div>
            </div>
        </>
    }
})


