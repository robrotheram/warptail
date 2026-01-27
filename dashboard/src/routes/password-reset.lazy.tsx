import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/context/AuthContext'
import { updateUser } from '@/lib/api'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { createLazyFileRoute, useNavigate } from '@tanstack/react-router'
import { AlertCircle, KeyRound } from 'lucide-react'
import { useState } from 'react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Progress } from '@/components/ui/progress'
import { calculateStrength, getStrengthColor, getStrengthLabel } from '@/components/utils/PasswordStrengthMeter'
import * as yup from 'yup'
import { useConfig } from '@/context/ConfigContext'

const PasswordResetSchema = yup.object({
  password: yup.string()
    .required('Password is required')
    .min(8, 'Password must be at least 8 characters long')
    .max(50, 'Password must be at most 50 characters long')
    .matches(/[A-Z]/, 'Password must contain at least one uppercase letter')
    .matches(/[a-z]/, 'Password must contain at least one lowercase letter')
    .matches(/[0-9]/, 'Password must contain at least one number')
    .matches(/[@$!%*?&#]/, 'Password must contain at least one special character (@, $, !, %, *, ?, &, #)'),
  confirmPassword: yup.string()
    .required('Please confirm your password')
    .oneOf([yup.ref('password')], 'Passwords must match'),
})

export const PasswordResetPage: React.FC = () => {
  const { user, token } = useAuth()
  const { site_name, site_logo } = useConfig()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [alert, setAlert] = useState<string>()

  const mutation = useMutation({
    mutationFn: updateUser,
    onSuccess: () => {
      // Invalidate and refetch the profile to get updated user data
      queryClient.invalidateQueries({ queryKey: ['profile', token] })
      navigate({ to: '/' })
    },
    onError: () => {
      setAlert('Failed to update password. Please try again.')
    },
  })

  // Redirect if no user or password_reset is not required
  if (!user) {
    return null
  }

  if (!user.password_reset) {
    navigate({ to: '/' })
    return null
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setErrors({})
    setAlert(undefined)

    try {
      await PasswordResetSchema.validate({ password, confirmPassword }, { abortEarly: false })
      
      mutation.mutate({
        ...user,
        password,
        password_reset: false,
      })
    } catch (err) {
      if (err instanceof yup.ValidationError) {
        const validationErrors: Record<string, string> = {}
        err.inner.forEach((error) => {
          if (error.path) {
            validationErrors[error.path] = error.message
          }
        })
        setErrors(validationErrors)
      }
    }
  }

  return (
    <Card className="my-10 mx-auto w-full max-w-md">
      <CardHeader className="pb-2 flex flex-col items-center justify-center gap-4">
        <img alt={site_name ? site_name : "WarpTail"} src={site_logo ? site_logo : '/logo.png'} className='w-20' />
        <div className="text-center">
          <CardTitle className="text-2xl flex items-center justify-center gap-2">
            <KeyRound className="h-6 w-6" />
            Change Your Password
          </CardTitle>
          <CardDescription className="mt-2">
            For security reasons, you must change your password before continuing.
          </CardDescription>
        </div>
      </CardHeader>
      <CardContent className='space-y-4'>
        <form onSubmit={handleSubmit} className='flex flex-col space-y-4'>
          <div className="flex flex-col gap-1">
            <Label htmlFor="password">New Password</Label>
            <Input
              type="password"
              id="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Enter your new password"
            />
            {password && (
              <div className="space-y-2 mt-2">
                <div className="flex justify-between text-sm">
                  <span>Strength:</span>
                  <span className="font-medium">{getStrengthLabel(calculateStrength(password))}</span>
                </div>
                <Progress value={calculateStrength(password)} className={getStrengthColor(calculateStrength(password))} />
              </div>
            )}
            {errors.password && (
              <div className="flex items-center text-red-500 text-sm mt-1">
                <AlertCircle className="w-4 h-4 mr-1" />
                {errors.password}
              </div>
            )}
          </div>

          <div className="flex flex-col gap-1">
            <Label htmlFor="confirmPassword">Confirm Password</Label>
            <Input
              type="password"
              id="confirmPassword"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              placeholder="Confirm your new password"
            />
            {errors.confirmPassword && (
              <div className="flex items-center text-red-500 text-sm mt-1">
                <AlertCircle className="w-4 h-4 mr-1" />
                {errors.confirmPassword}
              </div>
            )}
          </div>

          {alert && (
            <Alert className='bg-red-800 border-red-900 rounded-sm'>
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>Error</AlertTitle>
              <AlertDescription>{alert}</AlertDescription>
            </Alert>
          )}

          <Button type="submit" className="w-full" disabled={mutation.isPending}>
            {mutation.isPending ? 'Updating...' : 'Change Password'}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}

export const Route = createLazyFileRoute('/password-reset')({
  component: () => <PasswordResetPage />,
})
