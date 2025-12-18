import * as React from "react";
import { type LucideIcon } from "lucide-react";
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { useLocation, useNavigate } from "react-router-dom";
import { useDirtyStore } from "@/app/workflow/dirty-store";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";

export function NavMain({
  items,
}: {
  items: {
    title: string;
    url: string;
    icon?: LucideIcon;
  }[];
}) {
  const location = useLocation();
  const navigate = useNavigate();
  const { isDirty, markClean } = useDirtyStore();
  const [showDialog, setShowDialog] = React.useState(false);
  const [pendingUrl, setPendingUrl] = React.useState<string | null>(null);

  const handleNavClick = (
    e: React.MouseEvent<HTMLAnchorElement>,
    url: string
  ) => {
    // If already on this page, do nothing
    if (location.pathname === url || location.pathname.startsWith(url + "/")) {
      e.preventDefault();
      return;
    }

    // If dirty, show confirmation dialog
    if (isDirty) {
      e.preventDefault();
      setPendingUrl(url);
      setShowDialog(true);
    }
    // Otherwise, let the default Link behavior handle navigation
  };

  const handleConfirmLeave = () => {
    markClean();
    setShowDialog(false);
    if (pendingUrl) {
      navigate(pendingUrl);
      setPendingUrl(null);
    }
  };

  const handleCancelLeave = () => {
    setShowDialog(false);
    setPendingUrl(null);
  };

  return (
    <>
      <SidebarGroup>
        <SidebarGroupContent className="flex flex-col gap-2">
          <SidebarMenu>
            {items.map((item) => (
              <SidebarMenuItem key={item.title}>
                <SidebarMenuButton
                  asChild
                  isActive={
                    item.url === "/"
                      ? location.pathname === item.url
                      : location.pathname.startsWith(item.url)
                  }
                  tooltip={item.title}
                >
                  <a
                    href={item.url}
                    onClick={(e) => {
                      e.preventDefault();
                      handleNavClick(e, item.url);
                      if (!isDirty) {
                        navigate(item.url);
                      }
                    }}
                  >
                    {item.icon && <item.icon />}
                    <span>{item.title}</span>
                  </a>
                </SidebarMenuButton>
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>

      <AlertDialog open={showDialog} onOpenChange={setShowDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Unsaved Changes</AlertDialogTitle>
            <AlertDialogDescription>
              You have unsaved changes. Are you sure you want to leave? Your
              changes will be lost.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={handleCancelLeave}>
              Stay
            </AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmLeave}>
              Leave
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
