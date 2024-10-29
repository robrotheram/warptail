import {
  Card,
  CardContent,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'

import {
  Network,
  Earth,
  Activity,
} from 'lucide-react'
import { Route, RouterStatus, RouterType } from '../../lib/api'

type RouteCardProps = {
  route: Route
}



export const RouteStatusCard = ({ route }: RouteCardProps) => {
  const isActive = (route: Route): boolean => {
    if (route.latency && route?.status === RouterStatus.RUNNING) {
      if (route.latency > -1) {
        return true
      }
    }
    return false
  }
  return (
    <Card>
      <CardContent className="py-5 flex flex-col md:flex-row gap-4 justify-around items-center">
        <div>
          {route.type === RouterType.HTTP ? <Earth /> : <Network />}
        </div>
        <div>
          {(route.type === RouterType.TCP || route.type === RouterType.UDP) &&
            <>Listening: {route.port}</>
          }
          {(route.type === RouterType.HTTP) &&
            <a href={`http://${route.domain}`}>http://{route.domain}</a>
          }
        </div>
        <div>{route.machine.address}:{route.machine.port}</div>
        <div className='flex gap-2'>
          <Activity className={`h-5 w-5 ${isActive(route) ? 'text-green-500' : 'text-red-500'}`} />
          {isActive(route) && `${route.latency} ms`}
        </div>
        <Badge
          variant={
            route?.status === RouterStatus.RUNNING ? 'success' : 'destructive'
          }
          className="text-xs px-2 py-1"
        >
          {route?.status}
        </Badge>
      </CardContent>
    </Card>
  )
}