
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { createService, CreateService, getServices } from '@/lib/api'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '../ui/dialog'
import { useState } from 'react'
import { Plus } from 'lucide-react'
import { useConfig } from '@/context/ConfigContext'


export const CreateServiceModel = () => {
  const navigate = useNavigate({ from: `/` })
  const create = useMutation({
    mutationFn: createService,
    onSuccess: (data) => { navigate({ to: `/routes/${data.id}/edit` }) },
  })

  const [service, setService] = useState<CreateService>({name: "",routes: []});

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
        <Button  type="submit" onClick={() => { create.mutate(service) }}>
          <Plus className="mr-2 h-4 w-4" /> Create
        </Button>
      </div>
    </DialogContent>
  </Dialog>
}

export const RouteList = () => {
    const navigate = useNavigate({ from: '/' })
    const {read_only} = useConfig()
    const { data } = useQuery({
      queryKey: ['repoData'],
      queryFn: getServices,
    })
    
    return <Card>
        <CardHeader className='flex flex-row justify-between'>
          <div className='space-y-1.5 flex flex-col'>
            <CardTitle>Services</CardTitle>
            <CardDescription>Manage your load balancer routes.</CardDescription>
          </div>
          {!read_only &&<CreateServiceModel />}
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Service Name</TableHead>
                <TableHead>Average Latency</TableHead>
                <TableHead>Enabled</TableHead>
              </TableRow>
            </TableHeader>
            {data &&
            <TableBody>
              {
                data.sort((a, b) => {
                  return a.name.toLowerCase().localeCompare(b.name.toLowerCase());
                }).map(svc => {
                  return <TableRow onClick={() => navigate({ to: `/routes/${svc.id}` })} className='cursor-pointer' key={svc.id}>
                    <TableCell className="font-medium">{svc.name}</TableCell>
                    <TableCell>
                      {svc.enabled && svc.latency ? `${svc.latency} ms` : "n/a"}
                    </TableCell>
                    <TableCell>
                      <Badge variant={`${svc.enabled ? "success" : "destructive"}`}>{`${svc.enabled ? "Active" : "Inactive"}`}</Badge>
                    </TableCell>
                  </TableRow>
                })}
            </TableBody>
}
          </Table>
        </CardContent>
      </Card>
  }