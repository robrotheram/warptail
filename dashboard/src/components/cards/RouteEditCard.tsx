import {
  Card,
  CardContent,
} from '@/components/ui/card'
import { Button } from '@/components/ui/button'

import {
  Trash,
} from 'lucide-react'
import { Route, RouterType } from '../../lib/api'

import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '../ui/switch'
import { TailscaleHost } from '../utils/TailscaleHost'
import { AdvancedHttpSettings } from './AdvancedHttpSettings'



type RouteEditProps = {
  route: Route
  updateRoute: (route: Route) => void
  removeRoute: (route: Route) => void
}
export const RouteEditCard = ({ route, updateRoute, removeRoute }: RouteEditProps) => {

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target
    updateRoute({
      ...route,
      [name]: name === 'port' || name === 'listen' ? parseInt(value, 10) || value : value,
    })
  }
  
  const handleSelectChange = (value: string) => {
    updateRoute({ ...route, type: value as RouterType })
  }

  const handleStatusChange = (value: boolean) => {
    updateRoute({ ...route, private: value })
  }
  const handleBotProtectChange = (value: boolean) => {
    updateRoute({ ...route, bot_protect: value })
  }

  return (
    <>
      <Card>
        <CardContent className="py-5 flex gap-4 items-end flex-col md:flex-row">
          <div className='grid grid-cols-1  md:grid-cols-12 gap-4 items-center grow w-full'>
            <div className='w-full col-span-6 md:col-span-2'>
              <Label htmlFor="type">Type</Label>
              <Select name="type" onValueChange={handleSelectChange}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder={route.type} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={RouterType.HTTP}>{RouterType.HTTP}</SelectItem>
                  <SelectItem value={RouterType.HTTPS}>{RouterType.HTTPS}</SelectItem>
                  <SelectItem value={RouterType.TCP}>{RouterType.TCP}</SelectItem>
                  <SelectItem value={RouterType.UDP}>{RouterType.UDP}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {(route.type === RouterType.TCP || route.type === RouterType.UDP) && (
              <div className='w-full col-span-5'>
                <Label htmlFor="listen">Listen</Label>
                <Input
                  id="listen"
                  name="port"
                  type="number"
                  value={route.port}
                  onChange={handleInputChange}
                />
              </div>
            )}
            {(route.type === RouterType.HTTP || route.type === RouterType.HTTPS) && (

              <div className='col-span-5 flex flex-col md:flex-row gap-4'>
                <div className='flex flex-col py-1.5 h-full justify-end items-center'>
                  <Label htmlFor="domain">Private</Label>
                  <Switch className='mt-3'
                    checked={route.private}
                    onCheckedChange={handleStatusChange}
                  />
                </div>
                {route.type === RouterType.HTTPS && (
                <div className='flex flex-col py-1.5 h-full justify-end items-center'>
                  <Label htmlFor="domain">Bot Protection</Label>
                  <Switch className='mt-3'
                    checked={route.bot_protect}
                    onCheckedChange={handleBotProtectChange}
                  />
                </div>
                )}
                <div className='flex-grow'>
                  <Label htmlFor="domain">Domain</Label>
                  <Input
                    id="domain"
                    name="domain"
                    type="text"
                    value={route.domain}
                    onChange={handleInputChange}
                  />
                </div>
                
              </div>

            )}
            <TailscaleHost route={route} updateRoute={updateRoute}/>
          </div>
          <div className='flex flex-col justify-end h-full w-full md:w-auto'>
            <Button variant="destructive" onClick={() => removeRoute(route)}>
              <Trash className="h-4 w-4" />
            </Button>
          </div>

        </CardContent>
      </Card>
      <AdvancedHttpSettings route={route} updateRoute={updateRoute} />
    </>
  )
}