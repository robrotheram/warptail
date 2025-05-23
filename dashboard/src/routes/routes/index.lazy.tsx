import { RouteList } from '@/components/cards/ServiceList'
import { useAuth } from '@/context/AuthContext'
import { useConfig } from '@/context/ConfigContext'
import { Role } from '@/lib/api'
import ProtectedRoute from '@/Protected'
import { createLazyFileRoute } from '@tanstack/react-router'

export const Route = createLazyFileRoute('/routes/')({
    component: () => {
        const { user } = useAuth()
        const { read_only } = useConfig()
        return <ProtectedRoute><RouteList read_only={read_only || user?.role !== Role.ADMIN} /></ProtectedRoute>
    }
})
