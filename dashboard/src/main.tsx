import { StrictMode } from 'react'
import ReactDOM from 'react-dom/client'
import { RouterProvider, createRouter } from '@tanstack/react-router'
import "./index.css"
// Import the generated route tree
import { routeTree } from './routeTree.gen'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

// Create a new router instance
const router = createRouter({ routeTree })

// Register the router instance for type safety
declare module '@tanstack/react-router' {
   interface Register {
      router: typeof router
   }
}

const queryClient = new QueryClient({
   defaultOptions: {
      queries: {
         staleTime: 1000 * 60, // 1 minute
         gcTime: 1000 * 60 * 5, // 5 minutes
         refetchOnWindowFocus: false,
         retry: 1,
      },
   },
})

// Render the app
const rootElement = document.getElementById('root')!
if (!rootElement.innerHTML) {
   const root = ReactDOM.createRoot(rootElement)
   root.render(
      <StrictMode>
         <QueryClientProvider client={queryClient}>               
                  <RouterProvider router={router} />
         </QueryClientProvider>
      </StrictMode>,
   )
}