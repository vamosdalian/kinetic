import * as React from "react";
import { useLocation, useNavigate } from "react-router-dom";
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
import { useDirtyStore } from "./dirty-store";

export function UnsavedChangesGuard() {
  const { isDirty, markClean } = useDirtyStore();
  const location = useLocation();
  const navigate = useNavigate();
  const [showDialog, setShowDialog] = React.useState(false);
  const [pendingPath, setPendingPath] = React.useState<string | null>(null);
  const isNavigatingRef = React.useRef(false);

  // Handle browser close/refresh
  React.useEffect(() => {
    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (isDirty) {
        e.preventDefault();
        e.returnValue = "";
        return "";
      }
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, [isDirty]);

  // Handle browser back/forward buttons
  React.useEffect(() => {
    const handlePopState = () => {
      if (isDirty && !isNavigatingRef.current) {
        // Push current state back to prevent navigation
        window.history.pushState(null, "", location.pathname);
        setPendingPath("back");
        setShowDialog(true);
      }
    };

    window.addEventListener("popstate", handlePopState);
    return () => {
      window.removeEventListener("popstate", handlePopState);
    };
  }, [isDirty, location.pathname]);

  const handleConfirmLeave = () => {
    isNavigatingRef.current = true;
    markClean();
    setShowDialog(false);

    if (pendingPath === "back") {
      window.history.back();
    } else if (pendingPath) {
      navigate(pendingPath);
    }

    setPendingPath(null);
    // Reset the flag after navigation
    setTimeout(() => {
      isNavigatingRef.current = false;
    }, 100);
  };

  const handleCancelLeave = () => {
    setShowDialog(false);
    setPendingPath(null);
  };

  return (
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
          <AlertDialogCancel onClick={handleCancelLeave}>Stay</AlertDialogCancel>
          <AlertDialogAction onClick={handleConfirmLeave}>
            Leave
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

// Hook to use for programmatic navigation with dirty check
export function useNavigateWithDirtyCheck() {
  const { isDirty } = useDirtyStore();
  const navigate = useNavigate();

  return React.useCallback(
    (path: string, options?: { force?: boolean }) => {
      if (isDirty && !options?.force) {
        // Show confirmation via native confirm for simplicity
        const confirmed = window.confirm(
          "You have unsaved changes. Are you sure you want to leave?"
        );
        if (!confirmed) return;
      }
      navigate(path);
    },
    [isDirty, navigate]
  );
}
