import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { SidebarTrigger } from "@/components/ui/sidebar";
import { Github, Sun, Moon } from "lucide-react";
import * as React from "react";
import { useTheme } from "next-themes";
import {
  Breadcrumb as ShadcnBreadcrumb,
  BreadcrumbItem as ShadcnBreadcrumbItem,
  BreadcrumbList,
  BreadcrumbSeparator,
  BreadcrumbPage,
} from "@/components/ui/breadcrumb";
import { Link } from "react-router-dom";

export interface BreadcrumbItem {
  label: string
  href: string | null // null 表示不可点击（如动态参数）
}

export interface SiteHeaderProps {
  breadcrumbs: BreadcrumbItem[];
  actions?: React.ReactNode;
}

export function SiteHeader({ breadcrumbs, actions }: SiteHeaderProps) {
  const { resolvedTheme, setTheme } = useTheme();
  const isDarkMode = resolvedTheme === "dark";

  const toggleDarkMode = React.useCallback(() => {
    setTheme(isDarkMode ? "light" : "dark");
  }, [isDarkMode, setTheme]);

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
                  <ShadcnBreadcrumbItem>
                    <Link
                      to={item.href}
                      className="hover:text-foreground font-medium"
                    >
                      {item.label}
                    </Link>
                  </ShadcnBreadcrumbItem>
                ) : (
                  <ShadcnBreadcrumbItem>
                    <BreadcrumbPage>{item.label}</BreadcrumbPage>
                  </ShadcnBreadcrumbItem>
                )}
              </React.Fragment>
            ))}
          </BreadcrumbList>
        </ShadcnBreadcrumb>
        <div className="ml-auto flex items-center gap-2">
          {actions}
          <Button
            variant="ghost"
            onClick={toggleDarkMode}
            size="sm"
            className="hidden sm:flex"
          >
            {isDarkMode ? <Sun /> : <Moon />}
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
