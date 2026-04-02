import * as React from "react";
import {
  Workflow,
  Gauge,
  StretchHorizontal,
  HardDrive,
  User,
  Settings,
  FileQuestionMark,
  ClipboardClock,
} from "lucide-react";
import { NavMain } from "@/components/nav-main";
import { NavSecondary } from "@/components/nav-secondary";
import { NavUser } from "@/components/nav-user"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";

const data = {
  user: {
    name: "shadcn",
    email: "m@example.com",
    avatar: "/avatars/shadcn.jpg",
  },
  navMain: [
    {
      title: "Dashboard",
      url: "/",
      icon: Gauge,
    },
    {
      title: "WorkFlow",
      url: "/workflow",
      icon: StretchHorizontal,
    },
    {
      title: "Record",
      url: "/record",
      icon: ClipboardClock,
    },
    {
      title: "Node",
      url: "/node",
      icon: HardDrive,
    },
    {
      title: "Admin",
      url: "/admin",
      icon: User,
    },
  ],
  navSecondary: [
    {
      title: "Settings",
      url: "#",
      icon: Settings,
    },
    {
      title: "Get Help",
      url: "/docs",
      icon: FileQuestionMark,
    },
  ],
};

export function AppSidebar({
  user,
  onLogout,
  ...props
}: React.ComponentProps<typeof Sidebar> & {
  user: { username: string } | null
  onLogout: () => void
}) {
  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              asChild
              className="data-[slot=sidebar-menu-button]:!p-1.5"
            >
              <a href="/">
                <Workflow className="!size-5" />
                <span className="text-base font-semibold">Kinetic</span>
              </a>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={data.navMain} />
        <NavSecondary items={data.navSecondary} className="mt-auto" />
      </SidebarContent>
      {user ? (
        <SidebarFooter>
          <NavUser
            onLogout={onLogout}
            user={{
              name: user.username,
              email: "Administrator",
              avatar: "",
            }}
          />
        </SidebarFooter>
      ) : null}
    </Sidebar>
  );
}
