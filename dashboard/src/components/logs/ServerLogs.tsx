import { getLogs } from "@/lib/api"
import { useQuery } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import { LogViewer } from "./LogViewer"

export const ServerLogs = () => {
    const { data, isLoading } = useQuery({
        queryKey: ['server-logs'],
        queryFn: () => getLogs("server"),
        refetchInterval: 5000,
    })
    return (
        <div className="bg-muted/50 rounded-md p-4 font-mono text-sm">
            {isLoading && (
                <div className="flex items-center justify-center">
                    <Loader2 className="h-5 w-5 animate-spin" />
                </div>
            )}
            {!isLoading && data && data.length === 0 && (
                <div className="text-center text-muted-foreground py-8">No logs available</div>
            )}
            {!isLoading && data && data.length > 0 && (
                <LogViewer logs={data} />
            )}
        </div>
    )
}