
import { UserEditForm } from '@/components/forms/UserForm'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { useAuth } from '@/context/AuthContext'
import { updateUser } from '@/lib/api'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { createLazyFileRoute } from '@tanstack/react-router'
import { CheckCircle2, InfoIcon, Mail, Shield, User, Calendar } from 'lucide-react'
import { useState } from 'react'

const getInitials = (name: string) => {
  return name
    .split(' ')
    .map(word => word[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)
}

const formatDate = (date?: Date) => {
  if (!date) return 'Unknown'
  return new Date(date).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric'
  })
}

export const ProfilePage: React.FC = () => {
  const { user, token } = useAuth()
  const queryClient = useQueryClient()
  const [showSuccess, setShowSuccess] = useState(false)

  const mutation = useMutation({
    mutationFn: updateUser,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['profile', token] })
      setShowSuccess(true)
      setTimeout(() => setShowSuccess(false), 3000)
    },
  })

  if (!user) {
    return null
  }

  return (
    <div className='container max-w-2xl mx-auto py-8 px-4'>
      {/* Profile Header */}
      <div className="flex flex-col items-center mb-8">
        <Avatar className="h-24 w-24 mb-4">
          <AvatarFallback className="text-2xl bg-primary text-primary-foreground">
            {getInitials(user.name)}
          </AvatarFallback>
        </Avatar>
        <h1 className="text-2xl font-bold">{user.name}</h1>
        <p className="text-muted-foreground">{user.email}</p>
        <div className="flex items-center gap-2 mt-2">
          <Badge variant={user.role === 'admin' ? 'default' : 'secondary'}>
            <Shield className="h-3 w-3 mr-1" />
            {user.role}
          </Badge>
          <Badge variant="outline">
            {user.type === 'openid' ? 'SSO' : 'Local'}
          </Badge>
        </div>
      </div>

      {/* Success Message */}
      {showSuccess && (
        <Alert className="mb-6 bg-green-50 border-green-200 text-green-800">
          <CheckCircle2 className="h-4 w-4 text-green-600" />
          <AlertTitle>Success</AlertTitle>
          <AlertDescription>Your profile has been updated successfully.</AlertDescription>
        </Alert>
      )}

      {/* Profile Details Card */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <User className="h-5 w-5" />
            Profile Information
          </CardTitle>
          <CardDescription>
            {user.type === 'openid' 
              ? 'Your profile is managed by your identity provider'
              : 'Update your personal information and password'
            }
          </CardDescription>
        </CardHeader>
        <CardContent>
          {user.type !== "openid" ? (
            <UserEditForm mode='profile' user={user} onSubmit={mutation.mutate} />
          ) : (
            <div className="space-y-6">
              <Alert>
                <InfoIcon className="h-4 w-4" />
                <AlertTitle>Managed by SSO</AlertTitle>
                <AlertDescription>
                  Your profile is managed by the external authentication provider. 
                  Contact your administrator to make changes.
                </AlertDescription>
              </Alert>

              <Separator />

              <div className="grid gap-4">
                <div className="flex flex-col gap-2">
                  <Label htmlFor="name" className="flex items-center gap-2 text-muted-foreground">
                    <User className="h-4 w-4" />
                    Full Name
                  </Label>
                  <Input value={user.name} id="name" disabled className="bg-muted" />
                </div>
                <div className="flex flex-col gap-2">
                  <Label htmlFor="email" className="flex items-center gap-2 text-muted-foreground">
                    <Mail className="h-4 w-4" />
                    Email Address
                  </Label>
                  <Input value={user.email} id="email" disabled className="bg-muted" />
                </div>
                <div className="flex flex-col gap-2">
                  <Label htmlFor="role" className="flex items-center gap-2 text-muted-foreground">
                    <Shield className="h-4 w-4" />
                    Role
                  </Label>
                  <Input value={user.role} id="role" disabled className="bg-muted capitalize" />
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Account Info Card */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Calendar className="h-5 w-5" />
            Account Details
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-6">
            <div className="space-y-1">
              <p className="text-xs uppercase tracking-wide text-muted-foreground font-medium">Account Type</p>
              <p className="text-base font-semibold">{user.type === 'openid' ? 'Single Sign-On' : 'Local Account'}</p>
            </div>
            <div className="space-y-1">
              <p className="text-xs uppercase tracking-wide text-muted-foreground font-medium">Member Since</p>
              <p className="text-base font-semibold">{formatDate(user.created_at)}</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}


export const Route = createLazyFileRoute('/profile')({
  component: () => <ProfilePage/>,
})