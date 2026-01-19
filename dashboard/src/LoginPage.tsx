import React, { useEffect, useState, useCallback, useRef } from 'react'
import { useAuth } from './context/AuthContext'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Button , buttonVariants} from '@/components/ui/button'
import { useMutation } from '@tanstack/react-query'
import { login as api, AUTH_URL, getProfile, Login, Role } from "./lib/api"
import { useNavigate, useSearch } from '@tanstack/react-router'
import { AlertCircle, Fingerprint } from 'lucide-react'
import { Alert, AlertTitle, AlertDescription } from './components/ui/alert'
import { useConfig } from './context/ConfigContext'

/**
 * Validates and sanitizes a redirect URL to prevent open redirect attacks.
 * Only allows relative paths or same-origin URLs.
 */
function getSafeRedirectUrl(url: string | null): string | null {
  if (!url) return null
  
  try {
    // Check if it's a relative path (starts with /)
    if (url.startsWith('/') && !url.startsWith('//')) {
      // Ensure it doesn't contain protocol-relative URLs or other tricks
      const decoded = decodeURIComponent(url)
      if (decoded.startsWith('/') && !decoded.startsWith('//') && !decoded.includes('://')) {
        return url
      }
      return null
    }
    
    // Parse as absolute URL and check if same origin
    const parsed = new URL(url, window.location.origin)
    if (parsed.origin === window.location.origin) {
      return parsed.pathname + parsed.search + parsed.hash
    }
    
    return null
  } catch {
    return null
  }
}

export const LoginPage: React.FC = () => {
  // Use TanStack Router's useSearch for type-safe URL params
  const searchParams = useSearch({ strict: false }) as { next?: string; token?: string }
  
  const [userLogin, setUserLogin] = useState<Login>({ username: "", password: "" })
  const [alert, setAlert] = useState<string>()
  const { auth_type, site_name, site_logo } = useConfig()
  const { login } = useAuth()
  const navigate = useNavigate()
  
  // Use ref for token to avoid stale closure issues in mutation callbacks
  const tokenRef = useRef<string>()

  const profile = useMutation({
    mutationFn: getProfile,
    onSuccess: () => {
      if (tokenRef.current) {
        login(tokenRef.current)
        navigate({ to: '/' })
      }
    },
    onError: () => {
      setAlert("Permission Denied")
    }
  })

  const authenticate = useMutation({
    mutationFn: api,
    onSuccess: (data) => {
      const safeNext = getSafeRedirectUrl(searchParams.next ?? null);
      if (safeNext !== null) {
        // Safe redirect - only to same-origin paths
        window.location.href = `${safeNext}${safeNext.includes('?') ? '&' : '?'}token=${encodeURIComponent(data.authorization_token)}`
      } else if (data.role === Role.ADMIN) {
        tokenRef.current = data.authorization_token
        profile.mutate(data.authorization_token)
      } else {
        setAlert("Permission Denied")
      }
    },
    onError: () => {
      setAlert('Invalid username or password')
    }
  })

  const handleSubmit = useCallback((e: React.FormEvent) => {
    e.preventDefault()
    setAlert(undefined)
    authenticate.mutate(userLogin)
  }, [userLogin, authenticate])

  const handleUsernameChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setUserLogin(prev => ({ ...prev, username: e.target.value }))
  }, [])

  const handlePasswordChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setUserLogin(prev => ({ ...prev, password: e.target.value }))
  }, [])

  useEffect(() => {
    const tokenQuery = searchParams.token;
    if (tokenQuery) {
      tokenRef.current = tokenQuery
      profile.mutate(tokenQuery)
    }
  }, [searchParams.token]);


  return (
    <Card className="col-span-2 my-10 mx-auto max-w-screen-2xl w-full ">
      <CardHeader className="pb-2 flex flex-row items-center justify-center gap-4">
        <img alt={site_name?site_name:"WarpTail"} src={site_logo?site_logo:'/logo.png'} className='w-20' />
        <CardTitle className="text-3xl">{site_name?site_name:"WarpTail"}</CardTitle>
      </CardHeader>
      <CardContent className='space-y-4'>

        <form onSubmit={handleSubmit} className='flex flex-col space-y-4'>
          <div>
            <Label>Email: </Label>
            <Input
              type="text"
              value={userLogin.username}
              onChange={handleUsernameChange}
            />
          </div>
          <div>
            <Label>Password: </Label>
            <Input
              type="password"
              value={userLogin.password}
              onChange={handlePasswordChange}
            />
          </div>

          {
            alert && <Alert className='bg-red-800 border-red-900 rounded-sm'>
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>Error</AlertTitle>
              <AlertDescription>
                {alert}
              </AlertDescription>
            </Alert>
          }

          <Button type="submit" className="w-full">Login</Button>
        </form>
        {auth_type === "openid" && <>
          <div className="relative text-center text-sm after:absolute after:inset-0 after:top-1/2 after:z-0 after:flex after:items-center after:border-t after:border-border">
            <span className="relative z-10 bg-background px-2 text-muted-foreground">
              Or continue with
            </span>
          </div>
          <OpenIDButton/>
        </>
        }
      </CardContent>
    </Card>
  )
}


const OpenIDButton = () => {
  const { auth_name } = useConfig()
  const searchParams = useSearch({ strict: false }) as { next?: string }
  
  // Only use safe redirect URLs for the next parameter
  const safeNext = getSafeRedirectUrl(searchParams.next ?? null)
  const nextParam = safeNext ?? window.location.pathname

  return <a className={buttonVariants({ variant: "outline" }) + " w-full"} href={`${AUTH_URL}/login?next=${encodeURIComponent(nextParam)}`}>
    <Fingerprint></Fingerprint>
    Login with {auth_name}
  </a>

}