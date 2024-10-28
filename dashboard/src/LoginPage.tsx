import React, { useState } from 'react'
import { useAuth } from './context/AuthContext'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { useMutation } from '@tanstack/react-query'
import { login as api } from "./lib/api"
import { useNavigate } from '@tanstack/react-router'
import { AlertCircle } from 'lucide-react'
import { Alert, AlertTitle, AlertDescription } from './components/ui/alert'



export const LoginPage: React.FC = () => {
  const [password, setPassword] = useState('')
  const [alert, setAlert] = useState<string>()
  const { login } = useAuth()
  const navigate = useNavigate()
  const authenticate = useMutation({
    mutationFn: api,
    onSuccess: (data) => {
      login(data.authorization_token)
      navigate({ to: '/' })
    },
    onError: () => {
      setAlert('Invalid username or password')
    }
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    authenticate.mutate({password})
  }

  return (
    <Card className="col-span-2 my-10 mx-auto max-w-screen-sm">
      <CardHeader className="pb-2 flex flex-row items-center justify-center gap-4">
        <img src='/logo.png' className='w-20' />
        <CardTitle className="text-3xl">WarpTail</CardTitle>
      </CardHeader>
      <CardContent className='space-y-3'>

        {alert&&<Alert className='bg-red-800 border-red-900 rounded-sm'>
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>
            {alert}
          </AlertDescription>
        </Alert>}

        <form onSubmit={handleSubmit} className='flex flex-col space-y-4'>
          <div>
            <Label>Password: </Label>
            <Input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>

          <Button type="submit" className='w-24'>Login</Button>
        </form>
      </CardContent>
    </Card>
  )
}
