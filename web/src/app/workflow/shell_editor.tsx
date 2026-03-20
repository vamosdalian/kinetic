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

// CodeMirror 6 imports
import { EditorState } from "@codemirror/state";
import { EditorView, keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { syntaxHighlighting, defaultHighlightStyle, StreamLanguage } from "@codemirror/language";
import { shell } from "@codemirror/legacy-modes/mode/shell";
import { oneDark } from "@codemirror/theme-one-dark";
import { useTheme } from "next-themes";

interface ShellEditorProps {
  children: React.ReactNode;
  value?: string;
  onChange: (value: string) => void;
}

export function ShellEditor({
  children,
  value = "",
  onChange,
  ...props
}: ShellEditorProps) {
  const [open, setOpen] = React.useState(false);
  const [scriptContent, setScriptContent] = React.useState("");
  const editorRef = React.useRef<HTMLDivElement>(null);
  const viewRef = React.useRef<EditorView | null>(null);
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";

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

      // Get initial content from value
      const initialContent = value || "#!/bin/bash\n";
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
  }, [open, isDark, value]);

  const handleSave = () => {
    onChange(scriptContent);
    setOpen(false);
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
