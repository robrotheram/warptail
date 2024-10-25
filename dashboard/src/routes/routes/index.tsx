import { RouteList } from '@/components/cards/ServiceList'
import ProtectedRoute from '@/Protected'
import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/routes/')({
    component: () => <ProtectedRoute><RouteList/></ProtectedRoute>,
})
