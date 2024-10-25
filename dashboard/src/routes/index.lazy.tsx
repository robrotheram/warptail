import { RouteList } from '@/components/cards/ServiceList'
import ProtectedRoute from '@/Protected'
import { createLazyFileRoute } from '@tanstack/react-router'

export const Route = createLazyFileRoute('/')({
  component: () => <ProtectedRoute><RouteList/></ProtectedRoute>,
})

