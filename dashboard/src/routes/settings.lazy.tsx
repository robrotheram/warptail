import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useConfig } from '@/context/ConfigContext'
import { getTSConfig, getTSSTATUS, Tailsale, TailsaleNode, TS_STATE, updateTSConfig } from '@/lib/api'
import ProtectedRoute from '@/Protected'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createLazyFileRoute } from '@tanstack/react-router'
import { Save, Edit, Activity, Loader2, Key, Clock, ArrowUp, ArrowDown, ArrowUpDown, Terminal, CogIcon, TerminalIcon } from 'lucide-react'
import { useState } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { ServerLogs } from '@/components/logs/ServerLogs'
import { AccessLogs } from '@/components/logs/AccessLogs'
import { ErrorLogs } from '@/components/logs/ErrorLogs'

const TailScaleForm = () => {

  const { data } = useQuery({
    queryKey: ['tailscale'],
    queryFn: getTSConfig,
  })

  const [editMode, setEditMode] = useState(false)
  const [hasAuthKey, setHasAutKey] = useState(data?.AuthKey !== "")
  const { read_only } = useConfig()
  const [editedRoute, setEditedRoute] = useState<Tailsale>(data!)
  const queryClient = useQueryClient()
  const update = useMutation({
    mutationFn: updateTSConfig,
    onSuccess: () => {
      queryClient.invalidateQueries()
      setEditMode(false)
    },
  })

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target
    setEditedRoute({
      ...editedRoute,
      [name]: value,
    })
  }

  const handleEdit = () => {
    setEditMode(true)
    setEditedRoute(data!)
  }

  const handleSave = () => {
    update.mutate(editedRoute)
  }

  return <TabsContent value="status" className="mt-6">
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <CogIcon className="h-5 w-5" />
          TailScale Settings
        </CardTitle>
        <CardDescription>Edit Tailscale Settings</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <Label htmlFor="Hostname">Tailscale Hostname</Label>
            <Input
              id="Hostname"
              name="Hostname"
              type="text"
              value={editMode ? editedRoute.Hostname : data?.Hostname}
              onChange={handleInputChange}
              disabled={!editMode}
            />
          </div>
          <div className="flex items-center space-x-2">
            <Label htmlFor="airplane-mode">Use AuthKey: </Label>
            <Switch id="airplane-mode" checked={hasAuthKey} onCheckedChange={setHasAutKey} disabled={!editMode} />
          </div>
          {data?.AuthKey || hasAuthKey && (
            <div className="flex flex-col gap-2">
              <Label htmlFor="AuthKey">Tailscale API Key</Label>
              <Input
                id="AuthKey"
                name="AuthKey"
                type="text"
                value={editMode ? editedRoute.AuthKey : data?.AuthKey}
                onChange={handleInputChange}
                disabled={!editMode}
              />
            </div>
          )}
        </div>
      </CardContent>
      <CardFooter className={`flex justify-end`}>
        {editMode && (
          <div className="flex items-center gap-2">
            <Button onClick={() => setEditMode(false)} className="w-24 bg-gray-500 hover:bg-gray-600">
              Cancel
            </Button>
            <Button onClick={handleSave} className="w-24 bg-blue-500 hover:bg-blue-600" disabled={update.isPending}>
              {!update.isPending ? <Save className="mr-2 h-4 w-4" /> : <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
              Save
            </Button>
          </div>
        )}
        {!editMode && !read_only && (
          <Button onClick={handleEdit} className="w-24">
            <Edit className="mr-2 h-4 w-4" />
            Edit
          </Button>
        )}
      </CardFooter>
    </Card>
  </TabsContent>
}

type TailScaleKeyCardProps = {
  key_expiry?: Date
}

const TailScaleKeyCard = ({ key_expiry }: TailScaleKeyCardProps) => {


  // Calculate days until key expiry
  const calculateDaysUntilExpiry = (expiryDate: Date) => {
    const today = new Date()
    const diffTime = expiryDate.getTime() - today.getTime()
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24))
    return diffDays
  }

  // Get color based on days until expiry
  const getExpiryStatusColor = (days: number) => {
    if (days <= 0) return "text-red-500"
    if (days <= 14) return "text-red-500"
    if (days <= 30) return "text-amber-500"
    return "text-green-500"
  }

  // Format expiry date
  const formatExpiryDate = (date: Date) => {
    return new Date(date).toLocaleDateString(undefined, {
      year: "numeric",
      month: "long",
      day: "numeric",
    })
  }

  const daysUntilExpiry = key_expiry ? calculateDaysUntilExpiry(new Date(key_expiry)) : 0
  const expiryStatusColor = getExpiryStatusColor(daysUntilExpiry)


  return <Card>
    <CardHeader className="pb-2">
      <CardTitle className="text-sm font-medium">Key Expiry</CardTitle>
    </CardHeader>
    <CardContent>
      <div className="flex items-center gap-2">
        <Key className={`h-5 w-5 ${expiryStatusColor}`} />
        <span className={`text-xl font-bold ${expiryStatusColor}`}>
          {daysUntilExpiry <= 0 ? "Expired" : `${daysUntilExpiry} days`}
        </span>
      </div>
      <p className="text-xs text-muted-foreground mt-1">
        {key_expiry ? formatExpiryDate(key_expiry) : "No expiry date"}
      </p>
    </CardContent>
  </Card>
}

const formatLastSeen = (dateString: string) => {
  return new Date(dateString).toLocaleDateString(undefined, {
    year: "numeric",
    month: "long",
    day: "numeric",
  })
}

const TailsaleMessages = () => {
  return (
    <TabsContent value="logs" className="mt-6">
      <Tabs defaultValue="server">
        <Card>
          <CardHeader className='flex flex-row gap-2 justify-between'>

            <div>
              <CardTitle className="flex items-center gap-2">
                <Terminal className="h-5 w-5" />
                Server Logs
              </CardTitle>
              <CardDescription>Recent server logs</CardDescription>
            </div>
            <TabsList>
              <TabsTrigger value="server">Server Logs</TabsTrigger>
              <TabsTrigger value="access">Access Logs</TabsTrigger>
              <TabsTrigger value="error">Error Logs</TabsTrigger>
            </TabsList>
          </CardHeader>
          <CardContent>
            <TabsContent value="server"><ServerLogs /></TabsContent>
            <TabsContent value="access"><AccessLogs /></TabsContent>
            <TabsContent value="error"><ErrorLogs /></TabsContent>
          </CardContent>
        </Card>
      </Tabs>
    </TabsContent >
  )
}


type SortField = "hostname" | "ip" | "online" | "os" | "last_seen"
type SortDirection = "asc" | "desc"

type TailScaleNodesProps = {
  nodes: TailsaleNode[]
}
const TailScaleNodes = ({ nodes }: TailScaleNodesProps) => {
  const [sortField, setSortField] = useState<SortField>("online")
  const [sortDirection, setSortDirection] = useState<SortDirection>("desc")

  // Handle sorting
  const handleSort = (field: SortField) => {
    if (field === sortField) {
      // Toggle direction if same field
      setSortDirection(sortDirection === "asc" ? "desc" : "asc")
    } else {
      // Set new field and default to ascending
      setSortField(field)
      setSortDirection("asc")
    }
  }

  // Get sorted nodes
  const getSortedNodes = () => {
    return [...nodes].sort((a, b) => {
      let comparison = 0

      // Handle different field types
      if (sortField === "last_seen") {
        // Date comparison
        comparison = new Date(a[sortField]).getTime() - new Date(b[sortField]).getTime()
      } else if (sortField === "ip") {
        // IP address comparison (simple string comparison for demo)
        comparison = a[sortField].localeCompare(b[sortField])
      } else {
        // String comparison for other fields
        comparison = String(a[sortField]).localeCompare(String(b[sortField]))
      }

      // Apply sort direction
      return sortDirection === "asc" ? comparison : -comparison
    })
  }

  // Get sort icon for column header
  const getSortIcon = (field: SortField) => {
    if (field !== sortField) return <ArrowUpDown className="ml-2 h-4 w-4" />
    return sortDirection === "asc" ? <ArrowUp className="ml-2 h-4 w-4" /> : <ArrowDown className="ml-2 h-4 w-4" />
  }
  const sortedNodes = getSortedNodes()
  return (
    <TabsContent value="table" className="mt-6">
      <Card>
        <CardHeader>
          <CardTitle>TailScale Nodes</CardTitle>
          <CardDescription>Overview of all your TailScale nodes and their current status</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="cursor-pointer hover:bg-muted/50" onClick={() => handleSort("hostname")}>
                  <div className="flex items-center">
                    Name
                    {getSortIcon("hostname")}
                  </div>
                </TableHead>
                <TableHead className="cursor-pointer hover:bg-muted/50" onClick={() => handleSort("ip")}>
                  <div className="flex items-center">
                    IP Address
                    {getSortIcon("ip")}
                  </div>
                </TableHead>
                <TableHead className="cursor-pointer hover:bg-muted/50" onClick={() => handleSort("online")}>
                  <div className="flex items-center">
                    Status
                    {getSortIcon("online")}
                  </div>
                </TableHead>
                <TableHead className="cursor-pointer hover:bg-muted/50" onClick={() => handleSort("os")}>
                  <div className="flex items-center">
                    OS
                    {getSortIcon("os")}
                  </div>
                </TableHead>
                <TableHead className="cursor-pointer hover:bg-muted/50" onClick={() => handleSort("last_seen")}>
                  <div className="flex items-center">
                    Last Seen
                    {getSortIcon("last_seen")}
                  </div>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sortedNodes.map((node) => (
                <TableRow key={node.id}>
                  <TableCell className="font-medium">{node.name}</TableCell>
                  <TableCell>{node.ip}</TableCell>
                  <TableCell>
                    <Badge
                      variant={node.online ? "default" : "secondary"}
                      className={node.online ? "bg-green-500" : "bg-gray-500"}
                    >
                      {node.online ? " Online" : "Offline"}
                    </Badge>
                  </TableCell>
                  <TableCell>{node.os}</TableCell>
                  <TableCell className="flex items-center gap-1">
                    <Clock className="h-3 w-3 text-muted-foreground" />
                    <span>{formatLastSeen(node.last_seen)}</span>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </TabsContent>
  )
}

const SettingComponent = () => {
  const { error, data } = useQuery({
    queryKey: ['settings'],
    queryFn: getTSSTATUS,
  })

  return (
    <div className="container mx-auto py-6 space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-3xl font-bold">TailScale Status</h1>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded relative" role="alert">
          <span className="block sm:inline"></span>
        </div>
      )}

      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">TailScale Version</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data?.version}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Current Hostname</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{data?.hostname || "Unknown"}</div>
          </CardContent>
        </Card>

        <TailScaleKeyCard key_expiry={data?.key_expiry} />

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Network Status</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Activity className={`h-5 w-5 ${data?.state === TS_STATE.RUNNING ? 'text-green-500' : 'text-red-500'}`} />
              <span className="text-xl font-bold">{data?.state}</span>
            </div>
            <p className="text-xs text-muted-foreground mt-1">{data?.state !== TS_STATE.RUNNING && "Encountered some problems"}</p>
          </CardContent>
        </Card>
      </div>
      <Tabs defaultValue="table">
        <TabsList>
          <TabsTrigger value="table">List Nodes</TabsTrigger>
          <TabsTrigger value="logs"><TerminalIcon className='h-4' /> Logs</TabsTrigger>
          <TabsTrigger value="status"><CogIcon className='h-4' /> Settings</TabsTrigger>
        </TabsList>
        <TailScaleNodes nodes={data?.nodes || []} />
        <TailsaleMessages />
        <TailScaleForm />
      </Tabs>
    </div>
  )
}


export const Route = createLazyFileRoute('/settings')({
  component: () => <ProtectedRoute><SettingComponent /></ProtectedRoute>,
})
