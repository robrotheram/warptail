import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Progress } from '@/components/ui/progress'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { TableHeader, TableRow, TableHead, TableBody, TableCell, Table } from '@/components/ui/table'
import { calculateStrength, getStrengthColor, getStrengthLabel } from '@/components/utils/PasswordStrengthMeter'
import { useConfig } from '@/context/ConfigContext'
import { MenuIcon } from '@/Icons'
import { createUser, deleteUser, getUsers, Role, User, updateUser } from '@/lib/api'
import ProtectedRoute from '@/Protected'
import { Label } from '@radix-ui/react-label'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { AlertCircle, PencilIcon, Plus, TrashIcon } from 'lucide-react'
import { useState } from 'react'
import { Formik, Form, Field, ErrorMessage, useFormikContext } from 'formik';
import * as yup from 'yup';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'


const ErrorWithIcon = ({ name }: { name: string }) => (
  <ErrorMessage name={name}>
    {(message) => (
      <div className="flex items-center text-red-500 text-sm mt-1">
        <AlertCircle className="w-4 h-4 mr-1" />
        {message}
      </div>
    )}
  </ErrorMessage>
);

const RoleField = () => {
  const { setFieldValue, values } = useFormikContext<{ role: string }>();
  return (
    <div className='flex flex-col gap-1'>
      <Label htmlFor="role">Role</Label>
      <Select
        onValueChange={(value) => setFieldValue('role', value)}
        value={values.role}
      >
        <SelectTrigger className="w-full">
          <SelectValue placeholder="Select a role" />
        </SelectTrigger>
        <SelectContent>
          {Object.values(Role).map((role) => (
            <SelectItem key={role} value={role}>
              {role}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <ErrorWithIcon name="role" />
    </div>
  );
};

const PasswordField = () => {
  const { setFieldValue, values } = useFormikContext<{ password: string }>();
  return (
    <div className='flex flex-col gap-1'>
      <Label htmlFor="role">Password</Label>
      <Input
        value={values.password}
        onChange={(e) => { setFieldValue("password", e.target.value, true) }}
        type="password"
        id="password"
        placeholder="Enter your password"
      />
      {values.password &&
        <div className="space-y-2">
          <div className="flex justify-between text-sm">
            <span>Strength:</span>
            <span className="font-medium">{getStrengthLabel(calculateStrength(values.password))}</span>
          </div>
          <Progress value={calculateStrength(values.password)} className={getStrengthColor(calculateStrength(values.password))} />
        </div>
      }

      <ErrorWithIcon name="password" />
    </div>
  );
};

const CreateUserValidationSchema = yup.object({
  name: yup.string()
    .required('name is required')
    .min(3, 'name must be at least 3 characters long')
    .max(200, 'name must be at most 200 characters long'),
  password: yup.string()
    .required("Passsword is required") // password is optional
    .min(8, 'Password must be at least 8 characters long')
    .max(50, 'Password must be at most 50 characters long')
    .matches(/[A-Z]/, 'Password must contain at least one uppercase letter')
    .matches(/[a-z]/, 'Password must contain at least one lowercase letter')
    .matches(/[0-9]/, 'Password must contain at least one number')
    .matches(/[@$!%*?&#]/, 'Password must contain at least one special character (@, $, !, %, *, ?, &, #)'),
  email: yup.string()
    .required('Email is required')
    .email('Invalid email format'),
  role: yup.mixed<Role>()
    .required('Role is required')
    .oneOf(Object.values(Role), `Role must be one of: ${Object.values(Role).join(', ')}`),
});


const EditUserValidationSchema = yup.object({
  name: yup.string()
    .required('name is required')
    .min(3, 'name must be at least 3 characters long')
    .max(200, 'name must be at most 200 characters long'),
  password: yup.string()
    .optional()
    .min(8, 'Password must be at least 8 characters long')
    .max(50, 'Password must be at most 50 characters long')
    .matches(/[A-Z]/, 'Password must contain at least one uppercase letter')
    .matches(/[a-z]/, 'Password must contain at least one lowercase letter')
    .matches(/[0-9]/, 'Password must contain at least one number')
    .matches(/[@$!%*?&#]/, 'Password must contain at least one special character (@, $, !, %, *, ?, &, #)'),
  email: yup.string()
    .required('Email is required')
    .email('Invalid email format'),
  role: yup.mixed<Role>()
    .required('Role is required')
    .oneOf(Object.values(Role), `Role must be one of: ${Object.values(Role).join(', ')}`),
});



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

  const defaultUser: User = { name: '', email: '', password: '' };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {mode === 'create' && <DialogTrigger asChild>
        <Button variant="outline"> <Plus className="mr-2 h-4 w-4" />Create User</Button>
      </DialogTrigger>}
      <DialogContent className="sm:max-w-[800px]">
        <DialogHeader>
          <DialogTitle>{mode === 'create' ? 'Create User' : `Edit User ${user?.id}`}</DialogTitle>
        </DialogHeader>
        <Formik
          initialValues={ mode === 'create' ? defaultUser : user!}
          validationSchema={mode === "create" ? CreateUserValidationSchema : EditUserValidationSchema}
          onSubmit={mutation.mutate}
        >
          <Form className="flex flex-col space-y-4">
            <div className="flex flex-col gap-1">
              <label htmlFor="username">Name:</label>
              <Field name="name">
                {({ field }: { field: any }) => (
                  <Input {...field} id="name" placeholder="Enter your name" />
                )}
              </Field>
              <ErrorWithIcon name="username" />
            </div>
            <div className="flex flex-col gap-1">
              <Label htmlFor="email">Email</Label>
              <Field name="email">
                {({ field }: { field: any }) => (
                  <Input {...field} type="email" id="email" placeholder="Enter your email" />
                )}
              </Field>
              <ErrorWithIcon name="email" />
            </div>
            <PasswordField />
            <RoleField />
            <div className="flex gap-2 justify-end">
              <Button className="w-full" type="submit">
                {mode === 'create' ? 'Create' : 'Edit'}
              </Button>
              <Button className="w-full" variant={'secondary'} onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
            </div>
          </Form>
        </Formik>
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
  const { read_only: canEdit } = useConfig()
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

  return <Card>
    <CardHeader className='flex flex-row justify-between'>
      <div className='space-y-1.5 flex flex-col'>
        <CardTitle>Users</CardTitle>
        <CardDescription>Manage access to your applications</CardDescription>
      </div>
      {canEdit && <UserModel mode={"create"} open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen} />}
    </CardHeader>
    <CardContent>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Email</TableHead>
            <TableHead>Role</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map(svc => {
            return <TableRow key={svc.id}>
              <TableCell className="font-medium">{svc.name}</TableCell>
              <TableCell className="font-medium">{svc.email}</TableCell>
              <TableCell className="font-medium">{svc.role}</TableCell>
              <TableCell className=""><UserActions user={svc} /></TableCell>
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


