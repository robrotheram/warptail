
import { Config, getConfig } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { createContext, useContext, ReactNode } from 'react';


const ConfigContext = createContext<Config | undefined>(undefined);

export const ConfigProvider = ({ children }: { children: ReactNode }): ReactNode => {

    const {data:config} = useQuery({
        queryKey: ['config'],
        queryFn: getConfig,
    })

    return (
        <ConfigContext.Provider value={config}>
            {children}
        </ConfigContext.Provider>
    );
};

export const useConfig = (): Config => {
    const context = useContext(ConfigContext);
    if (!context) {
        throw new Error('useConfig must be used within an ConfigProvider');
    }
    return context;
};
