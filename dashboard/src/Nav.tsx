import { LogOut, Menu, Network, Settings, Users, UserRound } from 'lucide-react';
import { Button } from './components/ui/button';
import { SheetTrigger, SheetContent, Sheet, SheetTitle, SheetDescription } from './components/ui/sheet';
import { Link } from '@tanstack/react-router';
import { useState } from 'react';
import { useAuth } from './context/AuthContext';
import { Role } from './lib/api';
import { useConfig } from './context/ConfigContext';

const links = [
  { to: '/routes', label: 'Services', icon: Network, admin: false },
  { to: '/users', label: 'Users', icon: Users, admin: true },
  { to: '/settings', label: 'Settings', icon: Settings, admin: true },
];

function Brand() {
  const { site_name, site_logo } = useConfig();
  return <Link to="/" className="flex min-w-0 items-center gap-3 px-2 py-3 font-semibold tracking-tight">
    <img alt="" src={site_logo || '/logo.png'} className="h-9 w-9 object-contain" />
    <span className="truncate text-lg">{site_name || 'WarpTail'}</span>
  </Link>;
}

function Navigation({ close }: { close?: () => void }) {
  const { user, logout, isLoggingOut } = useAuth();
  return <>
    <nav aria-label="Main navigation" className="flex flex-1 flex-col gap-1 py-6">
      {links.filter(link => !link.admin || user?.role === Role.ADMIN).map(({ to, label, icon: Icon }) =>
        <Link key={to} to={to} onClick={close} className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground" activeProps={{ className: 'bg-primary/10 text-primary' }}>
          <Icon aria-hidden="true" className="h-4 w-4" />{label}
        </Link>)}
    </nav>
    <footer className="space-y-2 border-t py-4">
      <Link to="/profile" onClick={close} className="flex items-center gap-3 rounded-lg px-3 py-2 hover:bg-secondary" activeProps={{ className: 'bg-secondary' }}>
        <UserRound aria-hidden="true" className="h-5 w-5 shrink-0" />
        <span className="min-w-0"><span className="block truncate text-sm font-medium">{user?.name || 'Your profile'}</span><span className="block text-xs capitalize text-muted-foreground">{user?.role}</span></span>
      </Link>
      <Button variant="ghost" className="w-full justify-start gap-3 text-muted-foreground" disabled={isLoggingOut} onClick={() => void logout()}><LogOut aria-hidden="true" className="h-4 w-4" />{isLoggingOut ? 'Signing out…' : 'Sign out'}</Button>
    </footer>
  </>;
}

export function HeaderNav() {
  const [open, setOpen] = useState(false);
  return <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b bg-background/95 px-4 backdrop-blur sm:hidden">
    <Brand />
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild><Button size="icon" variant="outline" aria-label="Open navigation"><Menu className="h-5 w-5" /></Button></SheetTrigger>
      <SheetContent side="left" className="flex w-72 flex-col">
        <SheetTitle className="sr-only">Navigation</SheetTitle>
        <SheetDescription className="sr-only">Navigate WarpTail and manage your account.</SheetDescription>
        <Brand /><Navigation close={() => setOpen(false)} />
      </SheetContent>
    </Sheet>
  </header>;
}

export function SideNav() {
  return <aside className="fixed inset-y-0 left-0 z-30 hidden w-56 flex-col border-r bg-card px-4 sm:flex"><Brand /><Navigation /></aside>;
}
