import React, { useEffect, useState } from 'react'
import { useAuth } from './context/AuthContext'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { useMutation } from '@tanstack/react-query'
import { login as api, AUTH_URL, getProfile, Login, Role } from "./lib/api"
import { useNavigate } from '@tanstack/react-router'
import { AlertCircle, Fingerprint } from 'lucide-react'
import { Alert, AlertTitle, AlertDescription } from './components/ui/alert'
import { buttonVariants } from "@/components/ui/button"
import { useConfig } from './context/ConfigContext'



export const LoginPage: React.FC = () => {
  const urlParams = new URLSearchParams(window.location.search);
  const [userLogin, setUserLogin] = useState<Login>({ username: "", password: "" })
  const [token, setToken] = useState<string>()
  const [alert, setAlert] = useState<string>()
  const { auth_type, site_name, site_logo } = useConfig()
  const { login } = useAuth()
  const navigate = useNavigate()
  const authenticate = useMutation({
    mutationFn: api,
    onSuccess: (data) => {
      const next = urlParams.get('next');
      if (next !== null) {
        window.location.href = `${next}?token=${data.authorization_token}`
      } else if (data.role === Role.ADMIN) {
        setToken(data.authorization_token)
        profile.mutate(data.authorization_token)
      }
      setAlert("Permission Denied")
    },
    onError: () => {
      setAlert('Invalid username or password')
    }
  })

  const profile = useMutation({
    mutationFn: getProfile,
    onSuccess: () => {
      if (token) {
          login(token)
          navigate({ to: '/' })
      }
    },
    onError: () => {
      setAlert("Permission Denied")
    }
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    authenticate.mutate(userLogin)
  }

  useEffect(() => {
    const tokenQuery = urlParams.get('token');
    if (tokenQuery !== null) {
      setToken(tokenQuery)
      profile.mutate(tokenQuery)
    }
  }, []);


  return (
    <Card className="col-span-2 my-10 mx-auto max-w-screen-sm">
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
              onChange={(e) => setUserLogin({ ...userLogin, username: e.target.value })}
            />
          </div>
          <div>
            <Label>Password: </Label>
            <Input
              type="password"
              value={userLogin.password}
              onChange={(e) => setUserLogin({ ...userLogin, password: e.target.value })}
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
  const urlParams = new URLSearchParams(window.location.search);
  const next = urlParams.get('next') ?? String(window.location);

  return <a className={buttonVariants({ variant: "outline" }) + " w-full"} href={`${AUTH_URL}/login?next=${next}`}>
    <Fingerprint></Fingerprint>
    Login with {auth_name}
  </a>

}