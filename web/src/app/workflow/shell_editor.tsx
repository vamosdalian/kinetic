import React from "react";
import {
  Dialog,
  DialogContent,
  //   DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogFooter,
  DialogClose,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import Editor from "@monaco-editor/react";
import { useWorkflowStore, useSavedStore } from "./workflow-store";

export function ShellEditor({
  children,
  ...props
}: {
  children: React.ReactNode;
}) {
  const [theme, setTheme] = React.useState("light");
  const [open, setOpen] = React.useState(false);
  const [scriptContent, setScriptContent] = React.useState("");

  const { taskId, nodes, setNodes } = useWorkflowStore();
  const { setSaved } = useSavedStore();

  React.useEffect(() => {
    // Function to update color mode based on the presence of 'dark' class on <html>
    const updateTheme = () => {
      const isDarkMode = document.documentElement.classList.contains("dark");
      setTheme(isDarkMode ? "vs-dark" : "light");
    };

    // Set the initial color mode when the component mounts
    updateTheme();

    // Create an observer to watch for class changes on the <html> element
    const observer = new MutationObserver((mutationsList) => {
      for (const mutation of mutationsList) {
        if (
          mutation.type === "attributes" &&
          mutation.attributeName === "class"
        ) {
          updateTheme();
        }
      }
    });

    // Start observing the <html> element
    observer.observe(document.documentElement, { attributes: true });

    // Cleanup function to disconnect the observer when the component unmounts
    return () => {
      observer.disconnect();
    };
  }, []);

  React.useEffect(() => {
    if (open && taskId && nodes[taskId]) {
      setScriptContent(nodes[taskId].script || "#!/bin/bash");
    }
  }, [open, taskId, nodes]);

  const handleSave = () => {
    if (taskId) {
      setNodes({ [taskId]: { script: scriptContent } });
      setSaved(false);
      setOpen(false); // Close the dialog
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen} {...props}>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent className="sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Edit Script</DialogTitle>
        </DialogHeader>
        <div className="h-120 border">
          <Editor
            height="100%"
            width="100%"
            defaultLanguage="shell"
            value={scriptContent}
            onChange={(value) => setScriptContent(value || "")}
            theme={theme}
          />
        </div>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">Cancel</Button>
          </DialogClose>
          <Button type="submit" onClick={handleSave}>
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
