import {
  Card,
  CardContent,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'

import {
  Network,
  Earth,
  Activity,
  LockIcon,
} from 'lucide-react'
import { Route, RouterStatus, RouterType } from '../../lib/api'
import { formatDuration } from '@/lib/utils'

type RouteCardProps = {
  route: Route
}

const RouteIcon = ({route}:RouteCardProps)=> {
  if (route.type === RouterType.TCP || route.type === RouterType.UDP){
    return <Network />
  }
  return route.private? <LockIcon />:<Earth/>
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
      <CardContent className="py-5 flex flex-col justify- items-center gap-4 md:grid grid-cols-10">
        <RouteIcon route={route}/>
        <div className='col-span-3'>
          {(route.type === RouterType.TCP || route.type === RouterType.UDP) &&
            <>Listening: {route.port}</>
          }
          {(route.type === RouterType.HTTP || route.type === RouterType.HTTPS) &&
            <a href={`http://${route.domain}`}>http://{route.domain}</a>
          }
        </div>
        <div className='col-span-3'>{route.machine.address}:{route.machine.port}</div>
        <div className="col-span-2 flex flex-col gap-1 text-sm text-muted-foreground group">
          <div className="flex gap-2 grow items-center whitespace-nowrap">
            <Activity className={`h-5 w-5 ${isActive(route) ? 'text-green-500' : 'text-red-500'}`} />
            {isActive(route) && formatDuration(route.latency)}              
          </div>
          <span className="hidden group-hover:block">
          {route.type === RouterType.UDP && (
              <span className="text-xs text-muted-foreground">
                UDP only measures the server ping; it cannot measure service latency.
              </span>
            )}
          </span>
        </div>
        <Badge
          variant={
            route?.status === RouterStatus.RUNNING ? 'default' : 'destructive'
          }
          className={`text-xs px-2 py-1 flex justify-center ${route?.status === RouterStatus.RUNNING ? 'bg-green-700' : 'bg-red-700'}`}
        >
          {route?.status}
        </Badge>
      </CardContent>
    </Card>
  )
}