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
import { deleteService, getService, Role, Route as RouteInterface, Service, startService, stopService, updateService } from '../../lib/api'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { useEffect, useState } from 'react'
import { Input } from '@/components/ui/input'
import { RouteEditCard } from './RouteEditCard'
import { RouteStatusCard } from './RouteStatusCard'
import { RouterChart } from './ChartCard'
import { ErrorCard } from './ErrorCard'
import { useConfig } from '@/context/ConfigContext'
import { Label } from '../ui/label'
import { useAuth } from '@/context/AuthContext'



type ServiceCardProps = {
  id: string
  edit?: boolean
}
export const ServiceCard = ({ id, edit }: ServiceCardProps) => {
  const navigate = useNavigate()
  const { read_only } = useConfig()
   const {user} = useAuth()

  const queryClient = useQueryClient()
  const [service, setService] = useState<Service | null>(null);
  const { isPending, isError, data, isLoading } = useQuery({
    queryKey: ['route', id],
    retry: false,
    queryFn: () => getService(id),
    refetchInterval: edit ? false : 5000, // Disable polling when editing
    refetchIntervalInBackground: !edit, // Disable background polling when editing
  })
  useEffect(() => {
    if (data && !edit) {
      const updatedData = { ...data, routes: data.routes.map((r, i) => ({ ...r, key: i })) };
      setService(updatedData);
    } else if (data && edit && !service) {
      // Only set initial data when entering edit mode
      const updatedData = { ...data, routes: data.routes.map((r, i) => ({ ...r, key: i })) };
      setService(updatedData);
    }
  }, [data, edit]);

  const updateStatus = useMutation({
    mutationFn: service?.enabled ? stopService : startService,
    onSuccess: (svc) => {
      queryClient.setQueryData(['route', svc.id], svc)
      // Invalidate and refetch to get updated route status
      queryClient.invalidateQueries({ queryKey: ['route', svc.id] })
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
      bot_protect: false,
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
      <div className={`grid grid-cols-1 gap-0 space-y-6 md:gap-6 md:space-y-0 ${user?.role === Role.ADMIN && "md:grid-cols-3 "}`}>
        <Card className="col-span-2 flex flex-col justify-between">
          <CardContent className="p-12">
            {edit ? (
              <div className="space-y-8">
                <h1 className="text-3xl font-bold">Edit Service</h1>
                <div className="flex flex-col gap-3">
                  <Label htmlFor="name">Service Name:</Label>
                  <Input id="name" name="name" value={service.name} onChange={handleInputChange} />
                </div>
              </div>
            ) : (
              <div className="flex items-center justify-center h-full">
                <div className="text-center space-y-6 max-w-lg mx-auto">
                  <h1 className="text-5xl font-bold text-foreground leading-tight tracking-tight">{service.name}</h1>
                  <div className="flex items-center justify-center gap-8 text-muted-foreground">
                    <div className="flex items-center gap-2">
                      <div className="w-2 h-2 rounded-full bg-blue-500"></div>
                      <span className="text-lg font-medium">
                        {service.routes.length} {service.routes.length === 1 ? 'Route' : 'Routes'}
                      </span>
                    </div>
                    <div className="w-px h-6 bg-border"></div>
                    <div className="flex items-center gap-2">
                      <div className={`w-2 h-2 rounded-full ${service.enabled ? 'bg-green-500' : 'bg-red-500'}`}></div>
                      <span className={`text-lg font-medium ${service.enabled ? 'text-green-600' : 'text-red-600'}`}>
                        {service.enabled ? 'Running' : 'Stopped'}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
        {user?.role === Role.ADMIN &&
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
            {edit && !read_only && (
              <Button onClick={() => navigate({ to: `/routes/${data.id}` })} variant="secondary" className="w-full">
                Cancel
              </Button>
            )}
            {edit && !read_only && (
              <Button onClick={handleDelete} variant="destructive" className="w-full">
                <Trash className="mr-2 h-4 w-4" />
                Delete
              </Button>
            )}
            {edit && !read_only && (
              <Button onClick={handleSave} className="w-full">
                <Save className="mr-2 h-4 w-4" />
                Save
              </Button>
            )}
            {!edit && !read_only && (
              <Button onClick={() => navigate({ to: `/routes/${data.id}/edit` })} className="w-full">
                <Edit className="mr-2 h-4 w-4" />
                Edit
              </Button>
            )}
          </CardContent>
        </Card>}
      </div>
      <Card>
        <CardHeader className='flex flex-row justify-between gap-8 items-center '>
          <h2 className='text-2xl'>Routes:</h2>
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
