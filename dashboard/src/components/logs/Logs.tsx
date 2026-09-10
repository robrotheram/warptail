import { getLogs } from '@/lib/api';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Loader } from '@/context/AuthContext';
import { RequestError } from '../RequestError';
import { LogViewer, logLines } from './LogViewer';

export function Logs({ type }: { type: 'server' | 'access' | 'error' }) {
  const queryClient = useQueryClient();
  const queryKey = ['logs', type];
  const { data, isPending, error, refetch } = useQuery({
    queryKey,
    queryFn: async ({ signal }) => {
      const incoming = await getLogs(type, signal);
      // Server logs are snapshots; access/error endpoints drain their buffer.
      const previous = type === 'server' ? [] : queryClient.getQueryData<string[]>(queryKey) ?? [];
      return logLines([...previous, ...incoming]);
    },
    refetchInterval: 5000,
    refetchIntervalInBackground: false,
    retry: false,
  });
  return <div className="min-w-0">
    {error && <RequestError title="Unable to refresh logs" error={error} retry={() => void refetch()} />}
    {isPending ? <Loader label="Loading logs…" /> : <LogViewer logs={data ?? []} />}
  </div>;
}
