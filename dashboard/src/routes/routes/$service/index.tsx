import { ServiceCard } from '@/components/cards/ServiceCard'
import ProtectedRoute from '@/Protected'
import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/routes/$service/')({
    component: () => {
        const { service } = Route.useParams()
        return <ProtectedRoute><ServiceCard id={service} /></ProtectedRoute>
      },
})
