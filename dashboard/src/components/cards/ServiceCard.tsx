import { useNavigate } from '@tanstack/react-router'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Button } from '@/components/ui/button'

import {
  Edit,
  StopCircle,
  PlayCircle,
  Trash,
  Save,
  PlusIcon,
} from 'lucide-react'
import { deleteService, getService, Route as RouteInterface, Service, startService, stopService, updateService } from '../../lib/api'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { useEffect, useState } from 'react'
import { Input } from '@/components/ui/input'
import { RouteEditCard } from './RouteEditCard'
import { RouteStatusCard } from './RouteStatusCard'
import { RouterChart } from './ChartCard'
import { ErrorCard } from './ErrorCard'
import { useConfig } from '@/context/ConfigContext'
import { Label } from '../ui/label'



type ServiceCardProps = {
  id: string
  edit?: boolean
}
export const ServiceCard = ({ id, edit }: ServiceCardProps) => {
  const navigate = useNavigate()
  const { read_only: canEdit } = useConfig()

  const queryClient = useQueryClient()
  const { isPending, isError, data, isLoading } = useQuery({
    queryKey: ['route', id],
    retry: false,
    queryFn: () => getService(id),
  })
  const [service, setService] = useState<Service | null>(null);
  useEffect(() => {
    if (data) {
      data.routes = data.routes.map((r, i) => { return { ...r, key: i } })
      setService(data);
    }
  }, [data]);

  const updateStatus = useMutation({
    mutationFn: service?.enabled ? stopService : startService,
    onSuccess: (svc) => {
      queryClient.setQueryData(['route', svc.id], svc)
    },
  })

  const modifyService = useMutation({
    mutationFn: updateService,
    onSuccess: (svc) => {
      if (svc.id === id) {
        queryClient.invalidateQueries({ queryKey: ['route', svc.id] });
        return
      }
      navigate({ to: `/routes/${svc.id}` })
    },
  })
  const deleteFn = useMutation({
    mutationFn: deleteService,
    onSuccess: () => { navigate({ to: `/` }) }
  })


  if (isError) {
    return <ErrorCard />
  }
  if (isPending || isLoading || !service) {
    return 'LOADING'
  }

  const toggleStatus = () => {
    updateStatus.mutate(service.id)
  }

  const handleSave = () => {
    modifyService.mutate(service)
    navigate({ to: `/routes/${data.id}` })
  }

  const handleDelete = () => {
    deleteFn.mutate(service)
    queryClient.invalidateQueries()
    navigate({ to: `/` })
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target
    setService({ ...service, [name]: value })
  }
  const addRoute = () => {
    const routes = [{
      type: "",
      private: false,
      machine: {
        address: "",
        port: 0,
      }
      
    }, ...service.routes]
    setService({ ...service, routes })
  }

  const updateRoute = (route: RouteInterface) => {
    const routes = service.routes.map(r => {
      if (r.key === route.key) {
        return route
      }
      return r
    })
    setService({ ...service, routes })
  }

  const removeRoute = (route: RouteInterface) => {
    const routes = service.routes.filter(r => r.key !== route.key)
    setService({ ...service, routes })
  }

  if (!service) {
    return null
  }

  return (
    <div className="container mx-auto p-2 space-y-6">
      <div className="grid grid-cols-1 gap-0 space-y-6 md:grid-cols-3  md:gap-6  md:space-y-0">
        <Card className="col-span-2 flex flex-col justify-between">
          <CardContent>
            <CardHeader className='flex flex-col px-0 space-y-4'>
              {edit &&<h1 className='text-3xl'>Edit the service</h1>}
              {!edit && <CardTitle>{service.name}</CardTitle>}
              {edit &&<div className="flex flex-col gap-2">
                <Label htmlFor="name">Service Name:</Label>
                <Input id="name" name="name" value={service.name} onChange={handleInputChange} />
              </div>}
            </CardHeader>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className='py-4'>
            <CardTitle className="text-lg">Actions</CardTitle>
          </CardHeader>
          <CardContent className=" flex flex-col items-center space-y-4 justify-center">
            {!edit && <Button onClick={toggleStatus} variant={service.enabled ? 'destructive' : 'default'} className="w-full">
              {service.enabled ?
                (<><StopCircle className="mr-2 h-4 w-4" />Stop</>) :
                (<><PlayCircle className="mr-2 h-4 w-4" />Start</>)
              }
            </Button>
            }
            {edit && canEdit && (
              <Button onClick={() => navigate({ to: `/routes/${data.id}` })} variant="secondary" className="w-full">
                Cancel
              </Button>
            )}
            {edit && canEdit && (
              <Button onClick={handleDelete} variant="destructive" className="w-full">
                <Trash className="mr-2 h-4 w-4" />
                Delete
              </Button>
            )}
            {edit && canEdit && (
              <Button onClick={handleSave} className="w-full">
                <Save className="mr-2 h-4 w-4" />
                Save
              </Button>
            )}
            {!edit && canEdit && (
              <Button onClick={() => navigate({ to: `/routes/${data.id}/edit` })} className="w-full">
                <Edit className="mr-2 h-4 w-4" />
                Edit
              </Button>
            )}
          </CardContent>
        </Card>
      </div>
      <Card>
        <CardHeader className='flex flex-row justify-between gap-8 items-center '>
          <h2 className='text-2xl'>Routes</h2>
          {edit && <Button onClick={addRoute} className='w-full md:w-auto'>
            <PlusIcon className='mr-2' /> New Route
          </Button>}
        </CardHeader>
        <CardContent className='flex flex-col gap-4'>
          {service.routes.map(route => edit ?
            <RouteEditCard route={route} key={route.key} updateRoute={updateRoute} removeRoute={removeRoute} /> :
            <RouteStatusCard route={route} key={route.key} />)}
        </CardContent>
      </Card>
      {!edit && <RouterChart service={service} />}
    </div>
  )
}
