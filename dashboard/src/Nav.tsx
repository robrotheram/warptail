import { LogOut } from "lucide-react"
import { Button } from "./components/ui/button"
import { SheetTrigger, SheetContent, Sheet } from "./components/ui/sheet"
import { MenuIcon, NetworkIcon, SettingsIcon, UsersIcon } from "./Icons"
import { Link, useNavigate } from "@tanstack/react-router"
import React from "react"
import { useAuth } from "./context/AuthContext"
import { Avatar, AvatarFallback } from "./components/ui/avatar"
import { Role, User } from "./lib/api"
import { useConfig } from "./context/ConfigContext"

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
    to: "/users",
    label: "Users",
    icon: UsersIcon,
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
  const { site_name, site_logo } = useConfig()
  const { logout, user } = useAuth()
  const navigate = useNavigate()
  const handleLogout = () => {
    logout();
    navigate({ to: '/' })
  }


  return <header className="sticky top-0 z-30 flex h-14 items-center gap-4 border-b bg-background sm:static sm:h-auto sm:border-0 sm:bg-transparent px-4">
    <Sheet>
      <SheetTrigger asChild>
        <Button size="icon" variant="outline" className="sm:hidden">
          <MenuIcon className="h-5 w-5" />
          <span className="sr-only">Toggle Menu</span>
        </Button>
      </SheetTrigger>
      <SheetContent side="left" className="flex flex-col">
        <div className="w-full border-b pb-4">
          <Link to="/" className="group w-full flex items-center justify-left gap-4 rounded-full">
            <img alt={site_name ? site_name : "WarpTail"} src={site_logo ? site_logo : '/logo.png'} className="h-14 w-14 transition-all" />
            <h1 className="text-2xl">{site_name ? site_name : "WarpTail"}</h1>
          </Link>
        </div>
        <nav className="flex flex-col gap-6 text-lg font-medium flex-grow">
          {user?.role === Role.ADMIN && Links.map((link: LinksProps) => <Link key={link.to} to={link.to} className="flex items-center gap-4 px-2.5 text-foreground">
            {link.icon({ className: link.className })}
            <span className="sr-only">{link.label}</span>
            {link.label}
          </Link>)}
        </nav>
        <footer className="flex flex-col gap-4 px-2 pt-4 border-t">
          <Link to={"/profile"} className="flex items-center gap-4 text-foreground">
            <ProfileIcon user={user} />
            Your Profile
          </Link>
          <button onClick={() => handleLogout()} className="flex items-center gap-4 px-2.5 text-foreground">
            <LogOut className="h-5 w-5" />
            Logout
          </button>
        </footer>



      </SheetContent>
    </Sheet>
  </header>
}
export const SideNav = () => {
  const { site_name, site_logo } = useConfig()
  const { logout, user } = useAuth()
  const handleLogout = () => {
    logout();
  }

  return <aside className="fixed inset-y-0 left-0 z-10 hidden w-14 flex-col border-r bg-background sm:flex justify-between">
    <nav className="flex flex-col items-center gap-4 px-2 sm:py-5">
      <Link to="/" className="group flex h-9 w-9 shrink-0 items-center justify-center gap-2 rounded-full">
        <img alt={site_name ? site_name : "WarpTail"} src={site_logo ? site_logo : '/logo.png'} className="h-full w-full transition-all group-hover:scale-110" />
        <span className="sr-only">Load Balancer</span>
      </Link>
      {user?.role === Role.ADMIN && Links.map((link: LinksProps) =>
        <Link key={link.to} to={link.to} className="group flex h-9 w-9 shrink-0 items-center justify-center gap-2 rounded-full bg-primary text-lg font-semibold text-primary-foreground md:h-8 md:w-8 md:text-base">
          {link.icon({ className: link.className })}
          <span className="sr-only">{link.label}</span>
        </Link>)}
    </nav>
    <footer className="flex flex-col items-center gap-4 px-2 sm:py-5">
      <Link to={"/profile"} >
        <ProfileIcon user={user} />
      </Link>
      <button onClick={() => handleLogout()} className="group flex h-9 w-9 shrink-0 items-center justify-center gap-2 rounded-full bg-primary text-lg font-semibold text-primary-foreground md:h-8 md:w-8 md:text-base">
        <LogOut className="h-5 w-5" />
      </button>
    </footer>
  </aside>
}

type ProfeIconProps = {
  user?: User
}

const ProfileIcon = ({ user }: ProfeIconProps) => {
  if (!user) {
    return null
  }

  const parts = user.name.trim().split(/\s+/);

  const text = parts.length > 1
    ? (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
    : user.name.slice(0, 2).toUpperCase();

  // Extended color palette
  const colors = [
    "#FF5733", "#33FF57", "#3357FF", "#FF33A1", "#FFD633", "#33FFF6", "#B833FF", "#FF8C33",
    "#75FF33", "#FF3333", "#8E44AD", "#2980B9", "#27AE60", "#E74C3C", "#F39C12", "#16A085",
    "#2C3E50", "#D35400", "#7F8C8D", "#C0392B", "#1ABC9C", "#9B59B6", "#3498DB", "#34495E",
    "#F1C40F", "#E67E22", "#95A5A6", "#D1A7FF", "#8CFFBA", "#FFABAB", "#FFE156", "#6C5B7B",
  ];
  const colorIndex = Array.from(user.name).reduce((acc, char) => acc + char.charCodeAt(0), 0) % colors.length;

  const backgroundColor = colors[colorIndex];

  const hexToRgb = (hex: string) => {
    const bigint = parseInt(hex.slice(1), 16);
    return {
      r: (bigint >> 16) & 255,
      g: (bigint >> 8) & 255,
      b: bigint & 255,
    };
  };

  const rgb = hexToRgb(backgroundColor);
  const brightness = (rgb.r * 0.299 + rgb.g * 0.587 + rgb.b * 0.114) / 255;
  const textColor = brightness > 0.5 ? "#000000" : "#FFFFFF";

  return <Avatar className="h-9 w-9" style={{ backgroundColor }}>
    <AvatarFallback style={{ backgroundColor, color: textColor }}>{text}</AvatarFallback>
  </Avatar>;
}

