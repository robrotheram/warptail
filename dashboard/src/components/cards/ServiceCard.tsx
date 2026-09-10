import { useBlocker, useNavigate } from '@tanstack/react-router'
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

import { lazy, Suspense, useMemo, useRef, useState } from 'react'
import { getServiceHealth } from '@/lib/utils'
import { Input } from '@/components/ui/input'
import { RouteEditCard } from './RouteEditCard'
import { RouteStatusCard } from './RouteStatusCard'
const RouterChart = lazy(() => import('./ChartCard').then(module => ({ default: module.RouterChart })))
import { RequestError } from '../RequestError'
import { Loader } from '@/context/AuthContext'
import { toast } from 'sonner'
import { AlertDialog, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogDescription, AlertDialogFooter, AlertDialogCancel } from '../ui/alert-dialog'
import { useConfig } from '@/context/ConfigContext'
import { Label } from '../ui/label'
import { useAuth } from '@/context/AuthContext'



type ServiceCardProps = {
  id: string
  edit?: boolean
}
export const ServiceCard = (props: ServiceCardProps) => <ServiceContent key={`${props.id}-${!!props.edit}`} {...props} />

const ServiceContent = ({ id, edit }: ServiceCardProps) => {
  const navigate = useNavigate()
  const { read_only } = useConfig()
   const {user} = useAuth()

  const queryClient = useQueryClient()
  const [draft, setService] = useState<Service | null>(null);
  const { isPending, error, data, refetch } = useQuery({
    queryKey: ['route', id],
    retry: false,
    queryFn: ({ signal }) => getService(id, signal),
    refetchInterval: edit ? false : 5000, // Disable polling when editing
    refetchIntervalInBackground: false, // Disable background polling when editing
  })
  const fetchedService = useMemo(() => data ? { ...data, routes: (data.routes ?? []).map((route, key) => ({ ...route, key })) } : null, [data])
  const service = draft ?? fetchedService
  const leaving = useRef(false)
  const blocker = useBlocker({
    shouldBlockFn: () => !!draft && !leaving.current,
    enableBeforeUnload: () => !!draft && !leaving.current,
    withResolver: true,
  })
  const [confirmDelete, setConfirmDelete] = useState(false)
  const refreshServices = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['repoData'] }),
      queryClient.invalidateQueries({ queryKey: ['route', id] }),
    ])
  }

  const updateStatus = useMutation({
    mutationFn: service?.enabled ? stopService : startService,
    onSuccess: async () => { await refreshServices() },
  })

  const modifyService = useMutation({
    mutationFn: updateService,
    onSuccess: async (svc) => {
      await refreshServices()
      leaving.current = true
      toast.success('Service saved.')
      await navigate({ to: '/routes/$service', params: { service: svc.id || id } })
    },
  })
  const deleteFn = useMutation({
    mutationFn: deleteService,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['repoData'] })
      queryClient.removeQueries({ queryKey: ['route', id] })
      leaving.current = true
      toast.success('Service deleted.')
      await navigate({ to: '/routes' })
    },
  })
  const busy = modifyService.isPending || deleteFn.isPending || updateStatus.isPending

  if (error && !data) return <RequestError title="Unable to load service" error={error} retry={() => void refetch()} />
  if (isPending || !service) return <Loader label="Loading service…" />

  const toggleStatus = () => {
    updateStatus.mutate(service.id)
  }

  const handleSave = () => {
    if (!busy) modifyService.mutate(service)
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target
    setService({ ...service, [name]: value })
  }
  const addRoute = () => {
    const routes = [{
      key: Math.max(-1, ...service.routes.map(route => route.key ?? -1)) + 1,
      type: "https",
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

  const serviceHealth = getServiceHealth(service.enabled, service.routes)

  return (
    <div className="mx-auto space-y-6">
      {error && <RequestError title="Unable to refresh service" error={error} retry={() => void refetch()} />}
      {(modifyService.error || updateStatus.error) && <RequestError title="Unable to update service" error={modifyService.error || updateStatus.error} />}
      <div className={`grid grid-cols-1 gap-0 space-y-6 md:gap-6 md:space-y-0 ${user?.role === Role.ADMIN && "md:grid-cols-3 "}`}>
        <Card className="md:col-span-2 flex flex-col justify-between">
          <CardContent className="p-6 sm:p-10">
            {edit ? (
              <div className="space-y-8">
                <h1 className="text-3xl font-bold">Edit Service</h1>
                <div className="flex flex-col gap-3">
                  <Label htmlFor="name">Service Name:</Label>
                  <Input id="name" name="name" disabled={busy} value={service.name} onChange={handleInputChange} />
                </div>
              </div>
            ) : (
              <div className="flex items-center justify-center h-full">
                <div className="text-center space-y-6 max-w-lg mx-auto">
                  <h1 className="text-3xl break-words sm:text-4xl font-bold text-foreground leading-tight tracking-tight">{service.name}</h1>
                  <div className="flex items-center justify-center gap-8 text-muted-foreground">
                    <div className="flex items-center gap-2">
                      <div className="w-2 h-2 rounded-full bg-blue-500"></div>
                      <span className="text-lg font-medium">
                        {service.routes.length} {service.routes.length === 1 ? 'Route' : 'Routes'}
                      </span>
                    </div>
                    <div className="w-px h-6 bg-border"></div>
                    <div className="flex items-center gap-2">
                      <div className={`w-2 h-2 rounded-full ${serviceHealth.bgClass}`}></div>
                      <span className={`text-lg font-medium ${serviceHealth.textClass}`}>
                        {serviceHealth.label}
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
            {!edit && <Button disabled={busy} onClick={toggleStatus} variant={service.enabled ? 'destructive' : 'default'} className="w-full">
              {service.enabled ?
                (<><StopCircle className="mr-2 h-4 w-4" />Stop</>) :
                (<><PlayCircle className="mr-2 h-4 w-4" />Start</>)
              }
            </Button>
            }
            {edit && !read_only && (
              <Button disabled={busy} onClick={() => navigate({ to: `/routes/${id}` })} variant="secondary" className="w-full">
                Cancel
              </Button>
            )}
            {edit && !read_only && (
              <Button disabled={busy} onClick={() => setConfirmDelete(true)} variant="destructive" className="w-full">
                <Trash className="mr-2 h-4 w-4" />
                Delete
              </Button>
            )}
            {edit && !read_only && (
              <Button disabled={busy || !service.name.trim()} onClick={handleSave} className="w-full">
                <Save className="mr-2 h-4 w-4" />
                {modifyService.isPending ? 'Saving…' : 'Save changes'}
              </Button>
            )}
            {!edit && !read_only && (
              <Button disabled={busy} onClick={() => navigate({ to: `/routes/${id}/edit` })} className="w-full">
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
          {edit && !read_only && <Button disabled={busy} onClick={addRoute} className='w-full md:w-auto'>
            <PlusIcon className='mr-2' /> New Route
          </Button>}
        </CardHeader>
        <CardContent><fieldset disabled={busy} className="flex min-w-0 flex-col gap-4">
          {service.routes.map(route => edit ?
            <RouteEditCard route={route} key={route.key} updateRoute={updateRoute} removeRoute={removeRoute} /> :
            <RouteStatusCard route={route} key={route.key} />)}
        </fieldset></CardContent>
      </Card>
      {!edit && <Suspense fallback={<Loader label="Loading traffic chart…" />}><RouterChart service={service} /></Suspense>}
      <AlertDialog open={blocker.status === 'blocked'} onOpenChange={open => { if (!open) blocker.reset?.() }}>
        <AlertDialogContent><AlertDialogHeader><AlertDialogTitle>Discard unsaved changes?</AlertDialogTitle><AlertDialogDescription>Your edits have not been saved. Stay on this page to keep editing.</AlertDialogDescription></AlertDialogHeader><AlertDialogFooter><Button variant="outline" onClick={() => blocker.reset?.()}>Keep editing</Button><Button variant="destructive" onClick={() => blocker.proceed?.()}>Discard changes</Button></AlertDialogFooter></AlertDialogContent>
      </AlertDialog>
      <AlertDialog open={confirmDelete} onOpenChange={open => { if (!deleteFn.isPending) setConfirmDelete(open) }}>
        <AlertDialogContent>
          <AlertDialogHeader><AlertDialogTitle>Delete {service.name}?</AlertDialogTitle><AlertDialogDescription>This permanently removes the service and its routes. This cannot be undone.</AlertDialogDescription></AlertDialogHeader>
          {deleteFn.isError && <RequestError title="Unable to delete service" error={deleteFn.error} />}
          <AlertDialogFooter><AlertDialogCancel disabled={deleteFn.isPending}>Cancel</AlertDialogCancel><Button variant="destructive" disabled={deleteFn.isPending} onClick={() => deleteFn.mutate(service)}>{deleteFn.isPending ? 'Deleting…' : 'Delete service'}</Button></AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
