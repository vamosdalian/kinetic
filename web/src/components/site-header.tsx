/* eslint-disable react-refresh/only-export-components */
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { SidebarTrigger } from "@/components/ui/sidebar";
import { Github, Sun, Moon } from "lucide-react";
import * as React from "react";
import {
  Breadcrumb as ShadcnBreadcrumb,
  BreadcrumbItem,
  BreadcrumbList,
  BreadcrumbSeparator,
  BreadcrumbPage,
} from "@/components/ui/breadcrumb";
import { Link, useLocation } from "react-router-dom"

export interface BreadcrumbItem {
  label: string
  href: string | null // null 表示不可点击（如动态参数）
}

export const breadcrumbMap: Record<string, { label: string; isDynamic?: boolean }> = {
  "/": { label: "Dashboard" },
  "/workflow": { label: "Workflow" },
  "/record": { label: "Record" },
  "/node": { label: "Node" },
  "/admin": { label: "Admin" },
}

export function SiteHeader() {
  const [isDarkMode, setIsDarkMode] = React.useState<boolean>(false);

  const toggleDarkMode = () => setIsDarkMode((isDark) => !isDark);

  React.useEffect(() => {
    const initialDarkMode =
      !!document.querySelector('meta[name="color-scheme"][content="dark"]') ||
      window.matchMedia("(prefers-color-scheme: dark)").matches;
    setIsDarkMode(initialDarkMode);
  }, []);

  React.useEffect(() => {
    document.documentElement.classList.toggle("dark", isDarkMode);
  }, [isDarkMode]);

  const location = useLocation()

  const breadcrumbs = React.useMemo(() => {
    const pathSegments = location.pathname.split("/").filter(x => x)
    if (pathSegments.length === 0) {
      // Handle root path e.g. /
      const root = breadcrumbMap["/"]
      if (root) {
        return [{ label: root.label, href: null }]
      }
      return []
    }

    return pathSegments.map((segment, index) => {
      const currentPath = `/${pathSegments.slice(0, index + 1).join("/")}`
      const isLast = index === pathSegments.length - 1
      const item = breadcrumbMap[currentPath]

      if (item) {
        return {
          label: item.label,
          href: isLast ? null : currentPath,
        }
      }
      
      return {
        label: segment.replace(/_/g, " ").toUpperCase(),
        href: null, // Dynamic parts are not clickable
      }
    })
  }, [location.pathname])

  return (
    <header className="flex h-(--header-height) shrink-0 items-center gap-2 border-b transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-(--header-height)">
      <div className="flex w-full items-center gap-1 px-4 lg:gap-2 lg:px-6">
        <SidebarTrigger className="-ml-1" />
        <Separator
          orientation="vertical"
          className="mx-2 data-[orientation=vertical]:h-4"
        />
        <ShadcnBreadcrumb className="px-4 py-2">
          <BreadcrumbList>
            {breadcrumbs.map((item, index) => (
              <React.Fragment key={index}>
                {index > 0 && <BreadcrumbSeparator />}
                {item.href ? (
                  <BreadcrumbItem>
                    <Link
                      to={item.href}
                      className="hover:text-foreground font-medium"
                    >
                      {item.label}
                    </Link>
                  </BreadcrumbItem>
                ) : (
                  <BreadcrumbItem>
                    <BreadcrumbPage>{item.label}</BreadcrumbPage>
                  </BreadcrumbItem>
                )}
              </React.Fragment>
            ))}
          </BreadcrumbList>
        </ShadcnBreadcrumb>
        <div className="ml-auto flex items-center gap-2">
          <Button
            variant="ghost"
            onClick={toggleDarkMode}
            asChild
            size="sm"
            className="hidden sm:flex"
          >
            <span>{isDarkMode ? <Sun /> : <Moon />}</span>
          </Button>
          <Button variant="ghost" asChild size="sm" className="hidden sm:flex">
            <a
              href="https://github.com/vamosdalian/kinetic.git"
              rel="noopener noreferrer"
              target="_blank"
              className="dark:text-foreground"
            >
              <Github />
            </a>
          </Button>
        </div>
      </div>
    </header>
  );
}
