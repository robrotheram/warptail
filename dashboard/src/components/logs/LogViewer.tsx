import { AnsiHtml } from "fancy-ansi/react"
import { useEffect, useRef } from "react";
import "./logviewer.scss"

export const LogViewer = ({ logs }: { logs: string[] }) => {
    const bottomRef = useRef<null | HTMLDivElement>(null);
    
    useEffect(() => {
        bottomRef.current?.scrollIntoView({behavior: 'smooth'});
    }, [logs]);
    
    // Process logs to ensure proper line breaks
    const processedLogs = logs.map(log => 
        log.split('\n').filter(line => line.trim() !== '')
    ).flat();
    
    return (
        <div className="h-[400px] overflow-y-auto noScrollbar font-mono text-sm">
            <div className="space-y-1 p-4">
                {processedLogs.map((log, index) => (
                    <div key={index} className="whitespace-pre-wrap break-words">
                        <AnsiHtml text={log} />
                    </div>
                ))}
            </div>
            <div ref={bottomRef} />
        </div>
    )
}