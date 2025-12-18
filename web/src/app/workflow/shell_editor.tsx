import React from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogFooter,
  DialogClose,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useWorkflowStore } from "./workflow-store";

// CodeMirror 6 imports
import { EditorState } from "@codemirror/state";
import { EditorView, keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { syntaxHighlighting, defaultHighlightStyle, StreamLanguage } from "@codemirror/language";
import { shell } from "@codemirror/legacy-modes/mode/shell";
import { oneDark } from "@codemirror/theme-one-dark";

export function ShellEditor({
  children,
  ...props
}: {
  children: React.ReactNode;
}) {
  const [open, setOpen] = React.useState(false);
  const [scriptContent, setScriptContent] = React.useState("");
  const editorRef = React.useRef<HTMLDivElement>(null);
  const viewRef = React.useRef<EditorView | null>(null);

  const { selectedTaskId, taskNodes, updateTaskNode } = useWorkflowStore();
  const taskNode = taskNodes[selectedTaskId];

  // Track dark mode
  const [isDark, setIsDark] = React.useState(() =>
    document.documentElement.classList.contains("dark")
  );

  React.useEffect(() => {
    const updateTheme = () => {
      setIsDark(document.documentElement.classList.contains("dark"));
    };

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

    observer.observe(document.documentElement, { attributes: true });

    return () => {
      observer.disconnect();
    };
  }, []);

  // Initialize and manage CodeMirror editor
  React.useEffect(() => {
    if (!open) return;

    // Small delay to ensure Dialog content is fully rendered
    const timer = setTimeout(() => {
      if (!editorRef.current) {
        console.error("Editor container not found");
        return;
      }

      // Destroy previous editor instance if exists
      if (viewRef.current) {
        viewRef.current.destroy();
        viewRef.current = null;
      }

      // Clear any previous content
      editorRef.current.innerHTML = "";

      // Get initial content from config.script
      const initialContent = taskNode?.config?.script || "#!/bin/bash\n";
      setScriptContent(initialContent);

      const extensions = [
        lineNumbers(),
        highlightActiveLine(),
        highlightActiveLineGutter(),
        history(),
        keymap.of([...defaultKeymap, ...historyKeymap]),
        StreamLanguage.define(shell),
        syntaxHighlighting(defaultHighlightStyle),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            setScriptContent(update.state.doc.toString());
          }
        }),
        EditorView.theme({
          "&": { height: "100%" },
          ".cm-scroller": { overflow: "auto" },
        }),
      ];

      // Add dark theme if needed
      if (isDark) {
        extensions.push(oneDark);
      }

      const state = EditorState.create({
        doc: initialContent,
        extensions,
      });

      const view = new EditorView({
        state,
        parent: editorRef.current,
      });

      viewRef.current = view;
    }, 50); // Small delay for Dialog animation

    return () => {
      clearTimeout(timer);
      if (viewRef.current) {
        viewRef.current.destroy();
        viewRef.current = null;
      }
    };
  }, [open, isDark, taskNode]); // Recreate editor when dialog opens or theme changes

  const handleSave = () => {
    if (selectedTaskId && taskNode) {
      updateTaskNode(selectedTaskId, {
        config: { ...taskNode.config, script: scriptContent },
      });
      setOpen(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen} {...props}>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent className="sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Edit Script</DialogTitle>
        </DialogHeader>
        <div
          ref={editorRef}
          className="border rounded-md overflow-hidden"
          style={{ height: "480px" }}
        />
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
