import { ServiceCard } from '@/components/cards/ServiceCard'
import ProtectedRoute from '@/Protected'
import { createLazyFileRoute } from '@tanstack/react-router'

export const Route = createLazyFileRoute('/routes/$service/')({
    component: function Page() {
        const { service } = Route.useParams()
        return <ProtectedRoute><ServiceCard id={service} /></ProtectedRoute>
      },
})
