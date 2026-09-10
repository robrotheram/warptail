import { ServiceCard } from '@/components/cards/ServiceCard'
import { useConfig } from '@/context/ConfigContext'
import { Redirect } from '@/components/Redirect'
import ProtectedRoute from '@/Protected'
import { createLazyFileRoute } from '@tanstack/react-router'

export const Route = createLazyFileRoute('/routes/$service/edit')({
    component: function Page() {
        const { service } = Route.useParams()
        const { read_only } = useConfig()
        if (read_only) return <Redirect to={`/routes/${encodeURIComponent(service)}`} />
        return <ProtectedRoute admin><ServiceCard id={service} edit/></ProtectedRoute>
      },
})
