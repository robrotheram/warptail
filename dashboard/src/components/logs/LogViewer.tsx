import { memo, useEffect, useMemo, useRef, useState } from 'react';
import { Button } from '../ui/button';

export const MAX_LOG_LINES = 1000;
// Render untrusted logs as React text, never as HTML or terminal escape sequences.
export function logLines(logs: string[]) {
  return logs.slice(-MAX_LOG_LINES).flatMap(log => log.split('\n')).filter(line => line.trim()).slice(-MAX_LOG_LINES);
}

const LogLine = memo(function LogLine({ text }: { text: string }) {
  // eslint-disable-next-line no-control-regex
  const plain = text.replace(/\x1b\[[0-?]*[ -/]*[@-~]/g, '');
  return <div className="whitespace-pre-wrap break-all">{plain.split(/(https?:\/\/[^\s<>"{}|\\^`\[\]]+)/g).map((part, index) =>
    /^https?:\/\//.test(part) ? <a key={index} href={part} target="_blank" rel="noopener noreferrer" className="text-primary underline">{part}</a> : part
  )}</div>;
});

export function LogViewer({ logs }: { logs: string[] }) {
  const container = useRef<HTMLDivElement>(null);
  const [following, setFollowing] = useState(true);
  const lines = useMemo(() => logLines(logs), [logs]);
  useEffect(() => {
    if (following && container.current) container.current.scrollTop = container.current.scrollHeight;
  }, [lines, following]);
  return <div className="space-y-2">
    <div className="flex items-center justify-between gap-3 text-xs text-muted-foreground"><span>Latest {lines.length.toLocaleString()} lines · up to {MAX_LOG_LINES.toLocaleString()}</span><Button size="sm" variant="ghost" aria-pressed={following} onClick={() => setFollowing(value => !value)}>{following ? 'Pause scrolling' : 'Follow latest'}</Button></div>
    <div ref={container} tabIndex={0} aria-label="Log output" className="h-[400px] overflow-auto rounded-md bg-background p-4 font-mono text-xs leading-relaxed" onScroll={() => {
      const element = container.current;
      if (element && element.scrollHeight - element.scrollTop - element.clientHeight > 40) setFollowing(false);
    }}>{lines.length ? lines.map((line, index) => <LogLine key={index} text={line} />) : <p className="py-8 text-center text-muted-foreground">No logs available.</p>}</div>
  </div>;
}
