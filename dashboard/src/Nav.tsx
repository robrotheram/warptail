import { LogOut } from "lucide-react"
import { Button } from "./components/ui/button"
import { SheetTrigger, SheetContent, Sheet } from "./components/ui/sheet"
import { MenuIcon, NetworkIcon, SettingsIcon } from "./Icons"
import { Link, useNavigate } from "@tanstack/react-router"
import React from "react"
import { useAuth } from "./context/AuthContext"

interface LinksProps {
  to: string;
  label: string;
  icon: (props: React.SVGProps<SVGSVGElement>) => JSX.Element
  className: string
}
const Links = [
  {
    to: "/routes",
    label: "Routes",
    icon: NetworkIcon,
    className: "h-5 w-5"
  },
  {
    to: "/settings",
    label: "Settings",
    icon: SettingsIcon,
    className: "h-5 w-5"
  }
] as LinksProps[]



export const HeaderNav = () => {
  const { logout } = useAuth()
  const navigate = useNavigate()
  const handleLogout = () => {
    logout();
    navigate({ to: '/' })
  }


  return <header className="sticky top-0 z-30 flex h-14 items-center gap-4 border-b bg-background sm:static sm:h-auto sm:border-0 sm:bg-transparent">
    <Sheet>
      <SheetTrigger asChild>
        <Button size="icon" variant="outline" className="sm:hidden">
          <MenuIcon className="h-5 w-5" />
          <span className="sr-only">Toggle Menu</span>
        </Button>
      </SheetTrigger>
      <SheetContent side="left" className="">
        <div className="w-full">
          <Link to="/" className="group w-full flex items-center justify-left gap-4 rounded-full">
            <img src="/logo.png" className="h-14 w-14 transition-all" />
            <h1 className="text-2xl">WarpTail</h1>
          </Link>
        </div>
        <nav className="grid gap-6 text-lg font-medium py-6">
          {Links.map((link: LinksProps) => <Link key={link.to} to={link.to} className="flex items-center gap-4 px-2.5 text-foreground">
            {link.icon({ className: link.className })}
            <span className="sr-only">{link.label}</span>
            {link.label}
          </Link>)}
          <button onClick={()=>handleLogout()} className="flex items-center gap-4 px-2.5 text-foreground">
          <LogOut className="h-5 w-5" />
          Logout
          </button>
        </nav>
        
      </SheetContent>
    </Sheet>
  </header>
}
export const SideNav = () => {
  const { logout } = useAuth()
  const navigate = useNavigate()
  const handleLogout = () => {
    logout();
    navigate({ to: '/' })
  }

  return <aside className="fixed inset-y-0 left-0 z-10 hidden w-14 flex-col border-r bg-background sm:flex justify-between">
    <nav className="flex flex-col items-center gap-4 px-2 sm:py-5">
      <Link to="/" className="group flex h-9 w-9 shrink-0 items-center justify-center gap-2 rounded-full">
        <img src="/logo.png" className="h-full w-full transition-all group-hover:scale-110" />
        <span className="sr-only">Load Balancer</span>
      </Link>
      {Links.map((link: LinksProps) =>
        <Link key={link.to} to={link.to} className="group flex h-9 w-9 shrink-0 items-center justify-center gap-2 rounded-full bg-primary text-lg font-semibold text-primary-foreground md:h-8 md:w-8 md:text-base">
          {link.icon({ className: link.className })}
          <span className="sr-only">{link.label}</span>
        </Link>)}
    </nav>
    <footer className="flex flex-col items-center gap-4 px-2 sm:py-5">
      <button onClick={()=>handleLogout()} className="group flex h-9 w-9 shrink-0 items-center justify-center gap-2 rounded-full bg-primary text-lg font-semibold text-primary-foreground md:h-8 md:w-8 md:text-base">
        <LogOut className="h-5 w-5" />
      </button>
    </footer>
  </aside>
}