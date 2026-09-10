import React, { useEffect, useState, useCallback } from 'react'
import { useAuth, Loader } from './context/AuthContext'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Button, buttonVariants } from '@/components/ui/button'
import { useMutation } from '@tanstack/react-query'
import { login as api, AUTH_URL, Login, errorMessage, isUnauthorized } from "./lib/api"
import { useNavigate, useSearch } from '@tanstack/react-router'
import { AlertCircle, Fingerprint, Loader2, Lock, Mail } from 'lucide-react'
import { Alert, AlertTitle, AlertDescription } from './components/ui/alert'
import { useConfig } from './context/ConfigContext'

import { safeRedirect } from './lib/redirect'
import { RequestError } from './components/RequestError'

export const LoginPage: React.FC = () => {
  // Use TanStack Router's useSearch for type-safe URL params
  const searchParams = useSearch({ strict: false }) as { next?: string; token?: string }
  
  const [userLogin, setUserLogin] = useState<Login>({ username: "", password: "" })
  const [alert, setAlert] = useState<string>()
  const { auth_type, site_name, site_logo } = useConfig()
  const { login, isAuthenticated, requiresPasswordReset, isLoading, error, retry } = useAuth()
  const navigate = useNavigate()
  
  // Redirect to home if already authenticated, or to password reset if needed
  useEffect(() => {
    if (isAuthenticated) {
      if (requiresPasswordReset) {
        navigate({ to: '/password-reset', search: { next: safeRedirect(searchParams.next) ?? undefined }, replace: true })
      } else {
        const next = safeRedirect(searchParams.next)
        if (next) window.location.replace(next)
        else void navigate({ to: '/', replace: true })
      }
    }
  }, [isAuthenticated, requiresPasswordReset, navigate, searchParams.next])

  const authenticate = useMutation({
    mutationFn: api,
    onSuccess: async (data) => {
      setUserLogin(previous => ({ ...previous, password: '' }))
      await login(data.authorization_token)
    },
    onError: (error) => {
      setAlert(isUnauthorized(error) ? 'Check your username and password, then try again.' : errorMessage(error, 'Sign-in failed. Please try again.'))
    }
  })

  const handleSubmit = useCallback((e: React.FormEvent) => {
    e.preventDefault()
    if (authenticate.isPending) return
    setAlert(undefined)
    authenticate.mutate(userLogin)
  }, [userLogin, authenticate])

  const handleUsernameChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setUserLogin(prev => ({ ...prev, username: e.target.value }))
  }, [])

  const handlePasswordChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setUserLogin(prev => ({ ...prev, password: e.target.value }))
  }, [])

  if (isLoading) return <Loader label="Checking your session…" />
  if (error) return <div className="w-full max-w-md px-4"><RequestError title="Unable to check your session" error={error} retry={retry} /></div>


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
              <Label htmlFor="email" className="text-sm font-medium">Username or email</Label>
              <div className="relative">
                <Mail className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  id="email"
                  type="text"
                  placeholder="Enter your username or email"
                  autoComplete="username"
                  required
                  disabled={authenticate.isPending}
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
                  autoComplete="current-password"
                  required
                  disabled={authenticate.isPending}
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
  const safeNext = safeRedirect(searchParams.next)
  const nextParam = safeNext ? `/login?next=${encodeURIComponent(safeNext)}` : '/login'

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