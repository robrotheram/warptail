import React, { useEffect, useState, useCallback } from 'react'
import { useAuth } from './context/AuthContext'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Button, buttonVariants } from '@/components/ui/button'
import { useMutation } from '@tanstack/react-query'
import { login as api, AUTH_URL, Login, Role } from "./lib/api"
import { useNavigate, useSearch } from '@tanstack/react-router'
import { AlertCircle, Fingerprint, Loader2, Lock, Mail } from 'lucide-react'
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
  const { login, isAuthenticated, requiresPasswordReset } = useAuth()
  const navigate = useNavigate()
  
  // Redirect to home if already authenticated, or to password reset if needed
  useEffect(() => {
    if (isAuthenticated) {
      if (requiresPasswordReset) {
        navigate({ to: '/password-reset' })
      } else {
        navigate({ to: '/' })
      }
    }
  }, [isAuthenticated, requiresPasswordReset, navigate])

  const authenticate = useMutation({
    mutationFn: api,
    onSuccess: (data) => {
      const safeNext = getSafeRedirectUrl(searchParams.next ?? null);
      if (safeNext !== null) {
        // Safe redirect - only to same-origin paths
        window.location.href = `${safeNext}${safeNext.includes('?') ? '&' : '?'}token=${encodeURIComponent(data.authorization_token)}`
      } else if (data.role === Role.ADMIN) {
        // Just set the token - the useEffect above will handle navigation
        // once the AuthProvider's profile query completes
        login(data.authorization_token)
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
      // Token from URL (e.g., from OpenID callback) - just login with it
      // The isAuthenticated useEffect will handle navigation
      login(tokenQuery)
    }
  }, [searchParams.token, login]);


  return (
    <div className="w-full max-w-md mx-auto px-4">
      <Card className="border-0 shadow-lg">
        <CardHeader className="space-y-4 pb-6">
          <div className="flex flex-col items-center space-y-4">
            <div className="p-3 rounded-full bg-primary/10">
              <img 
                alt={site_name ? site_name : "WarpTail"} 
                src={site_logo ? site_logo : '/logo.png'} 
                className='w-16 h-16 object-contain' 
              />
            </div>
            <div className="text-center space-y-1">
              <CardTitle className="text-2xl font-bold">{site_name ? site_name : "WarpTail"}</CardTitle>
              <CardDescription>Sign in to your account to continue</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className='space-y-6'>
          {alert && (
            <Alert className='rounded-lg bg-red-700 border-red-800 text-red-50'>
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>Authentication Failed</AlertTitle>
              <AlertDescription>{alert}</AlertDescription>
            </Alert>
          )}

          <form onSubmit={handleSubmit} className='space-y-4'>
            <div className="space-y-2">
              <Label htmlFor="email" className="text-sm font-medium">Email</Label>
              <div className="relative">
                <Mail className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  id="email"
                  type="text"
                  placeholder="Enter your email"
                  value={userLogin.username}
                  onChange={handleUsernameChange}
                  className="pl-10"
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="password" className="text-sm font-medium">Password</Label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  id="password"
                  type="password"
                  placeholder="Enter your password"
                  value={userLogin.password}
                  onChange={handlePasswordChange}
                  className="pl-10"
                />
              </div>
            </div>

            <Button 
              type="submit" 
              className="w-full h-11 font-medium"
              disabled={authenticate.isPending}
            >
              {authenticate.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Signing in...
                </>
              ) : (
                'Sign In'
              )}
            </Button>
          </form>

          {auth_type === "openid" && (
            <>
              <div className="relative">
                <div className="absolute inset-0 flex items-center">
                  <span className="w-full border-t" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-background px-2 text-muted-foreground">
                    Or continue with
                  </span>
                </div>
              </div>
              <OpenIDButton />
            </>
          )}
        </CardContent>
      </Card>
      
      <p className="text-center text-xs text-muted-foreground mt-6">
        Secure access to your infrastructure
      </p>
    </div>
  )
}


const OpenIDButton = () => {
  const { auth_name } = useConfig()
  const searchParams = useSearch({ strict: false }) as { next?: string }
  
  // Only use safe redirect URLs for the next parameter
  const safeNext = getSafeRedirectUrl(searchParams.next ?? null)
  const nextParam = safeNext ?? window.location.pathname

  return (
    <a 
      className={buttonVariants({ variant: "outline" }) + " w-full h-11 font-medium gap-2"} 
      href={`${AUTH_URL}/login?next=${encodeURIComponent(nextParam)}`}
    >
      <Fingerprint className="h-4 w-4" />
      Login with {auth_name}
    </a>
  )
}