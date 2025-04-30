
import { UserEditForm } from '@/components/forms/UserForm'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useAuth } from '@/context/AuthContext'
import { updateUser } from '@/lib/api'
import { useMutation } from '@tanstack/react-query'
import { createLazyFileRoute } from '@tanstack/react-router'
import { InfoIcon } from 'lucide-react'




export const ProfilePage: React.FC = () => {
  const { user } = useAuth()
  const mutation = useMutation({
    mutationFn: updateUser,
    onSuccess: () => {

    },
  });
  if (!user) {
    return null
  }
  return <div className='flex px-3 items-center lg:mx-auto lg:w-1/3 w-full'>
    <Card className='w-full'>
      <CardHeader>
        <CardTitle>Edit Profile</CardTitle>
      </CardHeader>
      <CardContent>
        {user.type !== "openid" ?
          <UserEditForm mode='profile' user={user} onSubmit={mutation.mutate} /> :
          <div className="flex flex-col space-y-4">
            <Alert>
              <InfoIcon className="h-4 w-4" />
              <AlertTitle>Heads up!</AlertTitle>
              <AlertDescription>
                Your profile is managed by the external authentication provider
              </AlertDescription>
            </Alert>

            <div className="flex flex-col gap-1">
              <Label htmlFor="name">Name</Label>
              <Input value={user.name} id="name" placeholder="Enter your name" disabled />
            </div>
            <div className="flex flex-col gap-1">
              <Label htmlFor="email">Email</Label>
              <Input value={user.email} id="email" placeholder="Enter your email" disabled />
            </div>
            <div className="flex flex-col gap-1">
              <Label htmlFor="role">Role</Label>
              <Input value={user.role} id="role" disabled />
            </div>
          </div>}
      </CardContent>
    </Card>
  </div>
}


export const Route = createLazyFileRoute('/profile')({
  component: () => <ProfilePage/>,
})