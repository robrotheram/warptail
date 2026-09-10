import { Badge } from '@/components/ui/badge';
import { Card, CardContent } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { createService, getServices } from '@/lib/api';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useNavigate } from '@tanstack/react-router';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Label } from '../ui/label';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '../ui/dialog';
import { useMemo, useState } from 'react';
import { Plus, Search, Network, RefreshCw } from 'lucide-react';
import { formatDuration, getServiceHealth } from '@/lib/utils';
import { Loader } from '@/context/AuthContext';
import { RequestError } from '../RequestError';

export function CreateServiceModel() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('');
  const create = useMutation({
    mutationFn: createService,
    onSuccess: async data => {
      await queryClient.invalidateQueries({ queryKey: ['repoData'] });
      setOpen(false);
      setName('');
      await navigate({ to: '/routes/$service/edit', params: { service: data.id } });
    },
  });
  return <Dialog open={open} onOpenChange={value => { if (!create.isPending) { setOpen(value); setName(''); create.reset(); } }}>
    <DialogTrigger asChild><Button><Plus aria-hidden="true" className="mr-2 h-4 w-4" />Create service</Button></DialogTrigger>
    <DialogContent className="sm:max-w-lg">
      <DialogHeader><DialogTitle>Create service</DialogTitle><DialogDescription>Give your service a name, then add its routes.</DialogDescription></DialogHeader>
      <form className="space-y-4" onSubmit={event => { event.preventDefault(); if (name.trim() && !create.isPending) create.mutate({ name: name.trim(), routes: [] }); }}>
        <Label htmlFor="service-name">Service name</Label><Input id="service-name" value={name} onChange={event => setName(event.target.value)} autoFocus required maxLength={200} disabled={create.isPending} />
        {create.isError && <RequestError title="Unable to create service" error={create.error} />}
        <Button type="submit" disabled={!name.trim() || create.isPending}>{create.isPending ? 'Creating…' : 'Create service'}</Button>
      </form>
    </DialogContent>
  </Dialog>;
}

export function RouteList({ read_only }: { read_only: boolean }) {
  const [search, setSearch] = useState('');
  const { data, error, isPending, isFetching, refetch } = useQuery({ queryKey: ['repoData'], queryFn: getServices, refetchInterval: 10_000 });
  const services = useMemo(() => [...(data ?? [])].filter(service => service.name.toLowerCase().includes(search.trim().toLowerCase())).sort((a, b) => a.name.localeCompare(b.name)), [data, search]);
  return <div className="space-y-6">
    <div className="flex flex-wrap items-center justify-between gap-4">
      <div><h1 className="text-3xl font-semibold tracking-tight">Services</h1><p className="mt-2 text-sm text-muted-foreground">Manage routes and monitor your network.</p></div>
      {!read_only && <CreateServiceModel />}
    </div>
    {data && <div className="grid grid-cols-3 gap-3">
      {[['Services', data.length], ['Enabled', data.filter(service => service.enabled).length], ['Routes', data.reduce((count, service) => count + (service.routes?.length ?? 0), 0)]].map(([label, value]) => <Card key={label}><CardContent className="p-4 sm:p-5"><p className="text-xs text-muted-foreground">{label}</p><p className="mt-2 text-2xl font-semibold tabular-nums">{value}</p></CardContent></Card>)}
    </div>}
    <Card><CardContent className="p-4 sm:p-6">
      <div className="mb-5 flex items-center justify-between gap-3">
        <div className="relative w-full max-w-sm"><Search aria-hidden="true" className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" /><Input aria-label="Search services" placeholder="Search services…" className="pl-9" value={search} onChange={event => setSearch(event.target.value)} /></div>
        <Button aria-label="Refresh services" variant="outline" size="icon" disabled={isFetching} onClick={() => void refetch()}><RefreshCw className={`h-4 w-4 ${isFetching ? 'animate-spin' : ''}`} /></Button>
      </div>
      {error && <RequestError title={data ? 'Unable to refresh services' : 'Unable to load services'} error={error} retry={() => void refetch()} />}
      {isPending ? <Loader label="Loading services…" /> : data && services.length === 0 ? <div className="space-y-3 py-12 text-center"><Network className="mx-auto h-8 w-8 text-muted-foreground" /><h2 className="font-medium">{search ? 'No matching services' : 'No services yet'}</h2><p className="text-sm text-muted-foreground">{search ? 'Try a different name or clear your search.' : 'Create a service to connect your first application.'}</p>{search && <Button variant="outline" onClick={() => setSearch('')}>Clear search</Button>}</div> : data &&
        <Table><TableHeader><TableRow><TableHead>Service</TableHead><TableHead>Latency</TableHead><TableHead>Status</TableHead></TableRow></TableHeader><TableBody>
          {services.map(service => { const health = getServiceHealth(service.enabled, service.routes ?? []); return <TableRow key={service.id}>
            <TableCell><Link to="/routes/$service" params={{ service: service.id }} className="block py-2 font-medium text-foreground hover:text-primary hover:underline">{service.name}</Link></TableCell>
            <TableCell className="tabular-nums text-muted-foreground">{service.enabled && service.latency ? formatDuration(service.latency) : '—'}</TableCell>
            <TableCell><Badge variant="outline" className={health.color === 'green' ? 'border-emerald-500/30 text-emerald-400' : health.color === 'yellow' ? 'border-amber-500/30 text-amber-400' : 'border-red-500/30 text-red-400'}>{health.label}</Badge></TableCell>
          </TableRow>; })}
        </TableBody></Table>}
    </CardContent></Card>
  </div>;
}
