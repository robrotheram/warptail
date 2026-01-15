import { AnsiHtml } from "fancy-ansi/react"
import { useEffect, useRef } from "react";
import "./logviewer.scss"

// URL regex pattern to match http, https, and file URLs
const urlPattern = /(https?:\/\/[^\s<>"{}|\\^`[\]]+|file:\/\/[^\s<>"{}|\\^`[\]]+)/g;

// Component to render text with clickable links
const LinkifyText = ({ text }: { text: string }) => {
    const parts = text.split(urlPattern);
    
    return (
        <>
            {parts.map((part, i) => {
                if (urlPattern.test(part)) {
                    // Reset regex lastIndex after test
                    urlPattern.lastIndex = 0;
                    return (
                        <a 
                            key={i} 
                            href={part} 
                            target="_blank" 
                            rel="noopener noreferrer"
                            className="text-blue-400 hover:text-blue-300 underline"
                            onClick={(e) => e.stopPropagation()}
                        >
                            {part}
                        </a>
                    );
                }
                // Reset regex lastIndex
                urlPattern.lastIndex = 0;
                return <AnsiHtml key={i} text={part} />;
            })}
        </>
    );
};

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
                        <LinkifyText text={log} />
                    </div>
                ))}
            </div>
            <div ref={bottomRef} />
        </div>
    )
}