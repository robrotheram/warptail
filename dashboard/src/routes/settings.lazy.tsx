import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useConfig } from '@/context/ConfigContext'
import { getTSConfig, getTSSTATUS, Tailsale, TS_STATE, updateTSConfig } from '@/lib/api'
import ProtectedRoute from '@/Protected'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createLazyFileRoute } from '@tanstack/react-router'
import { Save, Edit, Activity, Loader2 } from 'lucide-react'
import { useState } from 'react'
import {
    Accordion,
    AccordionContent,
    AccordionItem,
    AccordionTrigger,
} from "@/components/ui/accordion"
import { ScrollArea } from '@/components/ui/scroll-area'


type TailScaleFormProps = {
    config: Tailsale
}
const TailScaleForm = ({ config }: TailScaleFormProps) => {
    const [editMode, setEditMode] = useState(false)
    const { read_only } = useConfig()
    const [editedRoute, setEditedRoute] = useState<Tailsale>(config)
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
        setEditedRoute(config)
    }

    const handleSave = () => {
        update.mutate(editedRoute)

    }
    return <>
        <CardContent>
            <div className="grid md:grid-cols-2 gap-4 grid-cols-1">
                <div className="flex flex-col gap-2">
                    <Label htmlFor="AuthKey">Tailscale API Key</Label>
                    <Input
                        id="AuthKey"
                        name="AuthKey"
                        type="text"
                        value={editMode ? editedRoute.AuthKey : config.AuthKey}
                        onChange={handleInputChange}
                        disabled={!editMode}
                    />
                </div>
                <div className="flex flex-col gap-2">
                    <Label htmlFor="Hostname">Tailscale Hostname</Label>
                    <Input
                        id="Hostname"
                        name="Hostname"
                        type="text"
                        value={editMode ? editedRoute.Hostname : config.Hostname}
                        onChange={handleInputChange}
                        disabled={!editMode}
                    />
                </div>
            </div>
            
        </CardContent>
        <CardFooter
            className={`flex justify-end`}
        >
            {editMode && (
                <Button onClick={handleSave} className="w-24">
                    {!update.isPending ? <Save className="mr-2 h-4 w-4" /> : <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
                    Save


                </Button>
            )}
            {!editMode && !read_only && (
                <Button onClick={handleEdit} className="w-24">
                    <Edit className="mr-2 h-4 w-4" />
                    Edit
                </Button>
            )}
        </CardFooter>
    </>
}

const TailScaleStatusComponent = () => {
    const { data } = useQuery({
        queryKey: ['settings', 'tailscale', "status"],
        queryFn: getTSSTATUS,
    })
    if (!data) {
        return null
    } 
    return <CardHeader className='w-full lg:w-[33%] space-y-3'>
        <div className='flex gap-2 items-center'>
            <Activity className={`h-5 w-5 ${data.state === TS_STATE.RUNNING ? 'text-green-500' : 'text-red-500'}`} />
            <p className='text-xl text-nowrap t'>{data.state}</p>
        </div>
        <CardTitle className='text-xl text-ellipsis '>Version: {data.version}</CardTitle>
        
        {data.messages && data.state !== TS_STATE.RUNNING && <Accordion type="single" collapsible>
            <AccordionItem value="item-1">
                <AccordionTrigger>Tailscale Log</AccordionTrigger>
                <AccordionContent>
                    <ScrollArea className="max-h-44 h-44 border p-3 ">
                        {data.messages.map((log, index) => (
                            <div
                                key={index}
                                className="mb-2 p-2 border"
                            >
                                <p className="text-sm">{log}</p>
                            </div>
                        ))}
                    </ScrollArea>
                </AccordionContent>
            </AccordionItem>
        </Accordion>}
    </CardHeader>

}


const SettingComponent = () => {
    const { isPending, error, data, isLoading } = useQuery({
        queryKey: ['settings', 'tailscale'],
        queryFn: getTSConfig,
    })

    if (isPending || isLoading) {
        return "LOADING"
    } else if (error) {
        console.log(error)
        return JSON.stringify(error)
    }

    return <Card className="container mx-auto p-2">
            <div className='flex flex-col lg:flex-row justify-between'>
                <CardHeader>
                    <CardTitle className='text-3xl'>Tailscale</CardTitle>
                    <CardDescription className='text-2xl'>Manage the connection</CardDescription>
                </CardHeader>
                <TailScaleStatusComponent />
            </div>
            <TailScaleForm config={data} />
        </Card>
}


export const Route = createLazyFileRoute('/settings')({
    component: () => <ProtectedRoute><SettingComponent /></ProtectedRoute>,
})
