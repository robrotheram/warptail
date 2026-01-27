import { getTSSTATUS, TS_STATE } from "@/lib/api";
import { useQuery } from "@tanstack/react-query";
import { ExternalLink } from "lucide-react";

export const Banner = () => {

    const { data: tsStatus } = useQuery({
        queryKey: ['settings'],
        queryFn: getTSSTATUS,
        refetchOnWindowFocus: true,
    })

    // Check if data is empty and error indicates authentication needed
    const needsTailscaleAuth = tsStatus?.state === TS_STATE.NEEDS_LOGIN

    if (needsTailscaleAuth) {
        return (
            <div className="bg-red-100 border-l-4 border-red-500 text-red-700 p-4 flex items-center justify-between flex-col md:flex-row" role="alert">
                <div>
                    <p className="font-bold">Tailscale Issue</p>
                    <p>
                        Warptail now requires Tailscale authentication to manage services. Please authenticate with Tailscale to continue using all features.
                    </p>
                </div>
                {tsStatus?.auth_url && (
                    <a 
                        href={tsStatus.auth_url} 
                        target="_blank" 
                        rel="noopener noreferrer"
                        className="flex justify-center w-full md:w-auto items-center gap-2 mt-2 px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 transition-colors"
                    >
                        <ExternalLink className="h-4 w-4" />
                        Login to Tailscale
                    </a>
                )}
            </div>
        );
    };
}
