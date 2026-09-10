import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogTrigger } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { TableHeader, TableRow, TableHead, TableBody, TableCell, Table } from '@/components/ui/table'
import { useConfig } from '@/context/ConfigContext'
import { MenuIcon } from '@/Icons'
import { createUser, deleteUser, getUsers, User, updateUser, Role } from '@/lib/api'
import ProtectedRoute from '@/Protected'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { PencilIcon, Plus, TrashIcon } from 'lucide-react'
import { useState } from 'react'

import { AlertDialog, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { UserEditForm } from '@/components/forms/UserForm'
import { RequestError } from '@/components/RequestError'
import { Loader } from '@/context/AuthContext'





type UserModelProps = {
  mode: 'create' | 'edit';
  user?: User; // Required for 'edit', optional for 'create'
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export const UserModel = ({ mode, user, open, onOpenChange }: UserModelProps) => {
  const queryClient = useQueryClient();

  const mutationFn = mode === 'create' ? createUser : updateUser;
  const mutation = useMutation({
    mutationFn,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['userList'] });
      await queryClient.invalidateQueries({ queryKey: ['profile'] });
      onOpenChange(false);
    },
  });

  const defaultUser: User = { name: '', email: '', password: '', type: "internal", role: Role.USER };

  return (
    <Dialog open={open} onOpenChange={value => { if (!mutation.isPending) { onOpenChange(value); mutation.reset(); } }}>
      {mode === 'create' && <DialogTrigger asChild>
        <Button variant="outline"> <Plus className="mr-2 h-4 w-4" />Create User</Button>
      </DialogTrigger>}
      <DialogContent className="sm:max-w-[800px]">
        <DialogHeader>
          <DialogTitle>{mode === 'create' ? 'Create User' : `Edit User ${user?.name}`}</DialogTitle>
          <DialogDescription>Manage this account’s profile and access.</DialogDescription>
        </DialogHeader>
        {mutation.isError && <RequestError title="Unable to save user" error={mutation.error} />}
        <UserEditForm
          mode={mode}
          user={mode === 'create' ? defaultUser : user!}
          onSubmit={mutation.mutateAsync}
          onCancel={() => onOpenChange(false)}
        />
      </DialogContent>
    </Dialog>
  );
};


type UserActionsProps = {
  user: User
}
const UserActions = ({ user }: UserActionsProps) => {
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false)
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false)
  const queryClient = useQueryClient();
  const remove = useMutation({
    mutationFn: deleteUser,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['userList'] })
      setIsDeleteDialogOpen(false)
    },
  })

  const handleDelete = () => {
    remove.mutate(user)
  }

  return <>
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger asChild><Button variant="ghost" size="icon" aria-label={`Actions for ${user.name}`}><MenuIcon /></Button></DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem onClick={() => setIsEditDialogOpen(true)}>
          <PencilIcon className="mr-2 h-4 w-4" />
          <span>Edit</span>
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => setIsDeleteDialogOpen(true)}>
          <TrashIcon className="mr-2 h-4 w-4" />
          <span>Delete</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
    <UserModel mode={"edit"} user={user} open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen} />
    <AlertDialog open={isDeleteDialogOpen} onOpenChange={open => { if (!remove.isPending) setIsDeleteDialogOpen(open) }}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Are you sure you want to delete?</AlertDialogTitle>
          <AlertDialogDescription>
            This action cannot be undone. This will permanently delete the item.
          </AlertDialogDescription>
        </AlertDialogHeader>
        {remove.isError && <RequestError title="Unable to delete user" error={remove.error} />}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={remove.isPending}>Cancel</AlertDialogCancel>
          <Button variant="destructive" disabled={remove.isPending} onClick={handleDelete}>{remove.isPending ? 'Deleting…' : 'Delete user'}</Button>

        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </>
}

const UserComponent = () => {
  const { read_only } = useConfig()
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false)
  const { isPending, error, data, isLoading, refetch } = useQuery({
    queryKey: ['userList'],
    queryFn: getUsers,
  })




  if (isPending || isLoading) {
    return <Loader/>
  } else if (error) {
    return <RequestError title="Unable to load users" error={error} retry={() => void refetch()} />
  }

  return <Card className="container mx-auto p-2 space-y-6">
    <CardHeader className='flex flex-row justify-between'>
      <div className='space-y-1.5 flex flex-col'>
        <CardTitle>Users</CardTitle>
        <CardDescription>Manage access to your applications</CardDescription>
      </div>
      {!read_only && <UserModel mode={"create"} open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen} />}
    </CardHeader>
    <CardContent>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Email</TableHead>
            <TableHead>Role</TableHead>
            <TableHead>Type</TableHead>
            {!read_only && <TableHead><span className="sr-only">Actions</span></TableHead>}
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.length === 0 && <TableRow><TableCell colSpan={5} className="py-8 text-center text-muted-foreground">No users yet.</TableCell></TableRow>}
          {[...data].sort((a, b) => {
            return a.name.toLowerCase().localeCompare(b.name.toLowerCase());
          }).map(svc => {
            return <TableRow key={svc.id}>
              <TableCell className="font-medium">{svc.name}</TableCell>
              <TableCell className="font-medium">{svc.email}</TableCell>
              <TableCell className="font-medium">{svc.role}</TableCell>
              <TableCell className="font-medium">{svc.type}</TableCell>
              {!read_only && <TableCell className=""><UserActions user={svc} /></TableCell>}
            </TableRow>
          })}
        </TableBody>
      </Table>
    </CardContent>
  </Card>

  // onClick={() => navigate({ to: `/routes/${svc.id}` })} className='cursor-pointer'
}



export const Route = createFileRoute('/users/')({
  component: () => <ProtectedRoute admin><UserComponent /></ProtectedRoute>,
})


