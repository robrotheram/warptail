import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { TableHeader, TableRow, TableHead, TableBody, TableCell, Table } from '@/components/ui/table'
import { useConfig } from '@/context/ConfigContext'
import { MenuIcon } from '@/Icons'
import { createUser, deleteUser, getUsers, User, updateUser } from '@/lib/api'
import ProtectedRoute from '@/Protected'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { PencilIcon, Plus, TrashIcon } from 'lucide-react'
import { useState } from 'react'

import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { UserEditForm } from '@/components/forms/UserForm'





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
    onSuccess: () => {
      onOpenChange(false);
      queryClient.invalidateQueries();
    },
  });

  const defaultUser: User = { name: '', email: '', password: '', type: "" };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {mode === 'create' && <DialogTrigger asChild>
        <Button variant="outline"> <Plus className="mr-2 h-4 w-4" />Create User</Button>
      </DialogTrigger>}
      <DialogContent className="sm:max-w-[800px]">
        <DialogHeader>
          <DialogTitle>{mode === 'create' ? 'Create User' : `Edit User ${user?.name}`}</DialogTitle>
        </DialogHeader>
        <UserEditForm
          mode={mode}
          user={mode === 'create' ? defaultUser : user!}
          onSubmit={mutation.mutate}
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
    onSuccess: () => {
      queryClient.invalidateQueries()
    },
  })

  const handleDelete = () => {
    remove.mutate(user)
    setIsDeleteDialogOpen(false)
  }

  return <>
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger><MenuIcon /></DropdownMenuTrigger>
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
    <AlertDialog open={isDeleteDialogOpen} onOpenChange={setIsDeleteDialogOpen}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Are you sure you want to delete?</AlertDialogTitle>
          <AlertDialogDescription>
            This action cannot be undone. This will permanently delete the item.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={handleDelete}>Delete</AlertDialogAction>

        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </>
}

const UserComponent = () => {
  const { read_only } = useConfig()
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false)
  const { isPending, error, data, isLoading } = useQuery({
    queryKey: ['userList'],
    queryFn: getUsers,
  })

  if (isPending || isLoading) {
    return "LOADING"
  } else if (error) {
    console.log(error)
    return JSON.stringify(error)
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
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.sort((a, b) => {
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
  component: () => <ProtectedRoute><UserComponent /></ProtectedRoute>,
})


