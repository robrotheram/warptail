import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { createService, CreateService, getServices } from '@/lib/api'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '../ui/dialog'
import { useState } from 'react'
import { Plus } from 'lucide-react'
import { formatDuration, getServiceHealth } from '@/lib/utils'

export const CreateServiceModel = () => {
  const navigate = useNavigate({ from: `/` })
  const create = useMutation({
    mutationFn: createService,
    onSuccess: (data) => { navigate({ to: `/routes/${data.id}/edit` }) },
  })

  const [service, setService] = useState<CreateService>({ name: "", routes: [] });

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target
    setService({ ...service, [name]: value })
  }

  return <Dialog>
    <DialogTrigger asChild>
      <Button variant="outline"> <Plus className="mr-2 h-4 w-4" />Create Service</Button>
    </DialogTrigger>
    <DialogContent className="sm:max-w-[500px]">
      <DialogHeader>
        <DialogTitle>Create Service</DialogTitle>
      </DialogHeader>

      <div className='flex gap-2'>
        <Input
          id="name"
          name="name"
          value={service.name}
          onChange={handleInputChange}
          placeholder='Service Name'
        />
        <Button type="submit" onClick={() => { create.mutate(service) }} disabled={service.name.length === 0}>
          <Plus className="mr-2 h-4 w-4" /> Create
        </Button>
      </div>
    </DialogContent>
  </Dialog>
}

type RouteListProps = {
  read_only: boolean
}
export const RouteList = ({ read_only }: RouteListProps) => {
  const navigate = useNavigate({ from: '/' })
  const { data, isError, error } = useQuery({
    queryKey: ['repoData'],
    queryFn: getServices,
  })

  // Check if data is empty and error indicates authentication needed
  const needsTailscaleAuth = !data || (data.length === 0 && (
    isError && (
      error?.message?.includes('NeedsLogin') ||
      error?.message?.includes('offline') ||
      error?.message?.includes('Authentication failed')
    )
  ))

  if (needsTailscaleAuth) {
    return (
      <Card className="container mx-auto p-2 space-y-6">
        <CardHeader>
          <CardTitle className='text-3xl text-center font-semibold text-red-600'>Tailscale Authentication Required</CardTitle>
        </CardHeader>
        <CardContent className='text-center'>
          <p className="text-muted-foreground mx-auto">
            The Tailscale client needs to be authenticated to access network services.
            Please authenticate your Tailscale client to continue.
          </p>
        </CardContent>
        <CardFooter className='flex justify-center'>
        <Button
          onClick={() => navigate({ to: '/settings', search: { tab: 'logs' } })}
          variant="destructive"
        >
          View Logs & Settings
        </Button>
        </CardFooter>
      </Card>
    )
  }
  return <Card className="container mx-auto p-2 space-y-6">
    <CardHeader className='flex flex-row justify-between'>
      <div className='space-y-1.5 flex flex-col'>
        <CardTitle>Services</CardTitle>
        <CardDescription>Manage your load balancer routes.</CardDescription>
      </div>
      {!read_only && !needsTailscaleAuth && <CreateServiceModel />}
    </CardHeader>
    <CardContent>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Service Name</TableHead>
            <TableHead>Average Latency</TableHead>
            <TableHead>Status</TableHead>
          </TableRow>
        </TableHeader>
        {data &&
          <TableBody>
            {
              (() => {
                const sorted = [...data].sort((a, b) => {
                  return a.name.toLowerCase().localeCompare(b.name.toLowerCase());
                });
                return sorted.map(svc => {
                  const health = getServiceHealth(svc.enabled, svc.routes)
                  return <TableRow onClick={() => navigate({ to: `/routes/${svc.id}` })} className='cursor-pointer' key={svc.id}>
                    <TableCell className="font-medium">{svc.name}</TableCell>
                    <TableCell>
                      {svc.enabled && svc.latency ? `${formatDuration(svc.latency)}` : "n/a"}
                    </TableCell>
                    <TableCell>
                      <Badge 
                        variant="default" 
                        className={`${health.color === 'green' ? 'bg-green-700' : health.color === 'yellow' ? 'bg-yellow-600' : 'bg-red-700'}`}
                      >
                        {health.label}
                      </Badge>
                    </TableCell>
                  </TableRow>
                });
              })()
            }
          </TableBody>
        }
      </Table>
    </CardContent>
  </Card>
}