import { Config, getConfig } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { createContext, useContext, useEffect, ReactNode } from 'react';
import { Loader } from './AuthContext';
import { RequestError } from '@/components/RequestError';

const ConfigContext = createContext<Config | undefined>(undefined);

export function ConfigProvider({ children }: { children: ReactNode }) {
  const { data: config, isPending, error, refetch } = useQuery({ queryKey: ['config'], queryFn: getConfig, staleTime: 300_000 });
  useEffect(() => {
    document.title = config?.site_name || 'WarpTail';
    if (config?.site_logo) {
      document.querySelectorAll<HTMLLinkElement>('link[rel="icon"], link[rel="apple-touch-icon"]').forEach(link => { link.href = config.site_logo!; });
    }
  }, [config?.site_name, config?.site_logo]);
  if (isPending) return <Loader label="Loading WarpTail…" />;
  if (error) return <div className="mx-auto max-w-md p-4"><RequestError title="Unable to load WarpTail" error={error} retry={() => void refetch()} /></div>;
  return <ConfigContext.Provider value={config}>{children}</ConfigContext.Provider>;
}

export function useConfig(): Config {
  return useContext(ConfigContext) ?? { read_only: true, auth_type: '', auth_name: '' };
}
