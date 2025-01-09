
import { Config, getConfig } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { createContext, useContext, ReactNode } from 'react';


const ConfigContext = createContext<Config | undefined>(undefined);

export const ConfigProvider = ({ children }: { children: ReactNode }): ReactNode => {

    const {data:config} = useQuery({
        queryKey: ['config'],
        queryFn: getConfig,
    })

    if (config?.site_name){
        document.title = config?.site_name;
    }
    
    if (config?.site_logo){
        let link = document.querySelector("link[rel~='icon'][sizes='16x16']") as HTMLLinkElement;
        link.href = config.site_logo;
        link = document.querySelector("link[rel~='icon'][sizes='32x32']") as HTMLLinkElement;
        link.href = config.site_logo;
        link = document.querySelector("link[rel~='apple-touch-icon']") as HTMLLinkElement;
        link.href = config.site_logo;
    }

    return (
        <ConfigContext.Provider value={config}>
            {children}
        </ConfigContext.Provider>
    );
};

export const useConfig = (): Config => {
    const context = useContext(ConfigContext);
    if (!context) {
        return {read_only: true, auth_type: "", auth_name: ""}
    }
    return context;
};
