import { createRootRoute, Outlet } from '@tanstack/react-router'
import { HeaderNav, SideNav } from "../Nav"
import { AuthProvider } from '@/context/AuthContext';

import { ConfigProvider } from '@/context/ConfigContext';


export const Route = createRootRoute({
    component: () => {
        return <ConfigProvider>
            <AuthProvider>
                <div className="flex min-h-screen w-full">
                    <SideNav />
                    <div className="flex flex-1 flex-col sm:gap-4 sm:py-4 sm:pl-14">
                        <HeaderNav />
                        <main className="flex-1 p-4 sm:px-6 sm:py-0">
                            <Outlet />
                        </main>
                    </div>
                </div> 
            </AuthProvider>
        </ConfigProvider>
    }
})


