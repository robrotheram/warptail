import { AnsiHtml } from "fancy-ansi/react"
import { useEffect, useRef } from "react";
import "./logviewer.scss"

export const LogViewer = ({ logs }: { logs: string[] }) => {
    const bottomRef = useRef<null | HTMLTableElement>(null);
    useEffect(() => {
        bottomRef.current?.scrollIntoView({behavior: 'smooth'});
      }, [logs]);
  return (
    <div className="h-[400px] overflow-y-auto noScrollbar">
        <table className="min-w-full divide-y">
          <tbody className=" divide-y">
            {logs.map((log, index) => (
              <tr key={index}>
                <td className="px-6 py-4 whitespace-nowrap text-sm">
                  <AnsiHtml text={log}/>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        <div ref={bottomRef} />
      </div>
  )
}