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
  const handleInputMachineChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target
    updateRoute({
      ...route, machine: {
        ...route.machine,
        [name]: name === 'Port' || name === 'Listen' ? parseInt(value, 10) || value : value,
      }
    })
  }
  const handleSelectChange = (value: string) => {
    updateRoute({ ...route, type: value as RouterType })
  }
  const handleStatusChange = (value: boolean) => {
    updateRoute({ ...route, private: value })
  }

  return (
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
              <div className='flex flex-col py-1.5 h-full justify-end'>
                <Label htmlFor="domain">Private</Label>
                <Switch className='mt-3'
                  checked={route.private}
                  onCheckedChange={handleStatusChange}
                />
              </div>
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
          <div className="md:col-span-5 grid md:grid-cols-2 col-span-6 gap-4 w-full">
            <div>
              <Label htmlFor="host">Tailscale Host</Label>
              <Input
                id="host"
                name="address"
                value={route?.machine.address}
                onChange={handleInputMachineChange}
              />
            </div>
            <div>
              <Label htmlFor="port">Tailscale Port</Label>
              <Input
                id="port"
                name="port"
                type="number"
                value={route?.machine.port}
                onChange={handleInputMachineChange}
              />
            </div>
          </div>
        </div>
        <div className='flex flex-col justify-end h-full w-full md:w-auto'>
          <Button variant="destructive" onClick={() => removeRoute(route)}>
            <Trash className="h-4 w-4" />
          </Button>
        </div>

      </CardContent>
    </Card>
  )
}