import { getLogs } from "@/lib/api"
import { useQuery } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import { LogViewer } from "./LogViewer"
import { useEffect, useState } from "react"

export const AccessLogs = () => {
    const { data, isLoading } = useQuery({
        queryKey: ['acssess-logs'],
        queryFn: () => getLogs("access"),
        refetchInterval: 2000,
    })
    const [logs, setLogs] = useState<string[]>([])
    useEffect(() => {
        if (data && data.length > 0) {
            setLogs(prev => [...prev, ...data])
        }
    }, [data])

    return (
        <div className="bg-muted/50 rounded-md p-4 font-mono text-sm">
            {isLoading && (
                <div className="flex items-center justify-center">
                    <Loader2 className="h-5 w-5 animate-spin" />
                </div>
            )}
            {!isLoading && logs.length === 0 && (
                <div className="text-center text-muted-foreground py-8">No logs available</div>
            )}
            {!isLoading && logs && logs.length > 0 && (
                <LogViewer logs={logs} />
            )}
        </div>
    )
}