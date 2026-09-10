import { ErrorMessage, Field, Form, Formik, useFormikContext, FieldInputProps } from "formik"
import { Input } from "../ui/input"
import { Button } from "../ui/button"
import { Label } from "../ui/label"
import { AlertCircle } from "lucide-react"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../ui/select"
import { Role, User } from "@/lib/api"
import { Progress } from "../ui/progress"
import { calculateStrength, getStrengthColor, getStrengthLabel } from "../utils/PasswordStrengthMeter"
import * as yup from 'yup';

export const CreateUserValidationSchema = yup.object({
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


export const EditUserValidationSchema = yup.object({
    name: yup.string()
        .required('name is required')
        .min(3, 'name must be at least 3 characters long')
        .max(200, 'name must be at most 200 characters long'),
    password: yup.string()
        .transform(value => value === '' ? undefined : value)
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

export const ProfileValidationSchema = yup.object({
    name: yup.string()
        .required('name is required')
        .min(3, 'name must be at least 3 characters long')
        .max(200, 'name must be at most 200 characters long'),
    password: yup.string()
        .transform(value => value === '' ? undefined : value)
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
});


type UserEditFormProps = {
    mode: 'create' | 'edit' | "profile";
    user: User;
    onSubmit: (user: User) => Promise<unknown>;
    onCancel?: () => void;
}

export const UserEditForm = ({ mode, user, onSubmit, onCancel }: UserEditFormProps) => {
    let validationSchema
    switch(mode){
        case "create": validationSchema = CreateUserValidationSchema; break;
        case "edit":validationSchema = EditUserValidationSchema; break;
        case "profile":validationSchema = ProfileValidationSchema; break;
    }

    return <Formik
        initialValues={{ ...user, password: '' }}
        validationSchema={validationSchema}
        onSubmit={async (values, helpers) => {
            try {
                await onSubmit({ ...values, password: values.password || undefined });
                helpers.resetForm({ values: { ...values, password: '' } });
            } catch { /* The owning mutation displays the error and preserves the draft. */ }
        }}
    >
        {({ isSubmitting }) => <Form className="flex flex-col space-y-4"><fieldset disabled={isSubmitting} className="contents">
            <div className="flex flex-col gap-1">
                <label htmlFor="name">Name:</label>
                <Field name="name">
                    {({ field }: { field: FieldInputProps<string> }) => (
                        <Input {...field} id="name" placeholder="Enter your name" disabled={user.type==="openid"} />
                    )}
                </Field>
                <ErrorWithIcon name="name" />
            </div>
            <div className="flex flex-col gap-1">
                <Label htmlFor="email">Email</Label>
                <Field name="email">
                    {({ field }: { field: FieldInputProps<string> }) => (
                        <Input {...field} type="email" id="email" placeholder="Enter your email" disabled={user.type==="openid"} />
                    )}
                </Field>
                <ErrorWithIcon name="email" />
            </div>
            {user.type!=="openid" &&<PasswordField />}
            {mode !== "profile" &&<RoleField />}
            <div className="flex gap-2 justify-end">
                <Button className="w-full" type="submit" disabled={isSubmitting}>
                    {isSubmitting ? 'Saving…' : mode === 'create' ? 'Create user' : 'Save changes'}
                </Button>
                {onCancel&&<Button className="w-full" variant={'secondary'} type="button" disabled={isSubmitting} onClick={() => onCancel()}>
                    Cancel
                </Button>}
            </div>
        </fieldset></Form>}
    </Formik>
}


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
                <SelectTrigger id="role" className="w-full">
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
            <Label htmlFor="password">Password</Label>
            <Input
                value={values.password ?? ''}
                onChange={(e) => { setFieldValue("password", e.target.value, true) }}
                type="password"
                id="password"
                autoComplete="new-password"
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
