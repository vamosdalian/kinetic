import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { ShellEditor } from "./shell_editor";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import type { TaskNode, TaskType, ShellConfig } from "./types";

interface TaskFormProps {
  taskId: string;
  node: TaskNode;
  onUpdate: (id: string, data: Partial<Omit<TaskNode, "id">>) => void;
}

export function Taskform({ taskId, node, onUpdate }: TaskFormProps) {
  if (!node) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground">
        Select a task to edit
      </div>
    );
  }

  return (
    <div className="grid gap-6 m-4">
      <div className="grid gap-2">
        <h1 className="text-xl">Task Node</h1>
        <Separator style={{ margin: "0" }}></Separator>
      </div>
      <div className="grid gap-2">
        <Label htmlFor="task_name">Task Name</Label>
        <Input
          id="task_name"
          placeholder="Enter task name..."
          value={node.name}
          onChange={(e) => {
            onUpdate(taskId, { name: e.target.value });
          }}
        />
      </div>
      <div className="grid gap-2">
        <Label htmlFor="description">Description</Label>
        <Textarea
          id="description"
          placeholder="Describe what this task does..."
          value={node.description}
          onChange={(e) => {
            onUpdate(taskId, { description: e.target.value });
          }}
        />
      </div>
      <div className="grid gap-2">
        <Label htmlFor="task_type">Task Type</Label>
        <Select
          value={node.type}
          onValueChange={(value: TaskType) => {
            // 切换类型时，重置 config
            let newConfig = {};
            if (value === "shell") {
              newConfig = { script: "" } as ShellConfig;
            } else if (value === "http") {
              newConfig = { url: "", method: "GET" };
            } else if (value === "python") {
              newConfig = { script: "", requirements: [] };
            } else if (value === "condition") {
              newConfig = { expression: "" };
            }
            onUpdate(taskId, { type: value, config: newConfig });
          }}
        >
          <SelectTrigger id="task_type" className="w-full">
            <SelectValue placeholder="Select task type" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="shell">Shell</SelectItem>
            <SelectItem value="http">HTTP</SelectItem>
            <SelectItem value="python">Python</SelectItem>
            <SelectItem value="condition">Condition</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Shell Config */}
      {node.type === "shell" && (
        <div className="grid gap-2">
          <Label>Script</Label>
          <ShellEditor
            value={node.config?.script}
            onChange={(val) =>
              onUpdate(taskId, { config: { ...node.config, script: val } })
            }
          >
            <Button variant="outline" className="w-full">
              Edit Script
            </Button>
          </ShellEditor>
        </div>
      )}

      {/* HTTP Config */}
      {node.type === "http" && (
        <>
          <div className="grid gap-2">
            <Label htmlFor="http_url">URL</Label>
            <Input
              id="http_url"
              placeholder="https://api.example.com/endpoint"
              value={node.config?.url || ""}
              onChange={(e) => {
                onUpdate(taskId, {
                  config: { ...node.config, url: e.target.value },
                });
              }}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="http_method">Method</Label>
            <Select
              value={node.config?.method || "GET"}
              onValueChange={(value) => {
                onUpdate(taskId, {
                  config: { ...node.config, method: value },
                });
              }}
            >
              <SelectTrigger id="http_method" className="w-full">
                <SelectValue placeholder="Select method" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="GET">GET</SelectItem>
                <SelectItem value="POST">POST</SelectItem>
                <SelectItem value="PUT">PUT</SelectItem>
                <SelectItem value="DELETE">DELETE</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </>
      )}

      {/* Python Config */}
      {node.type === "python" && (
        <div className="grid gap-2">
          <Label>Python Script</Label>
          <Textarea
            placeholder="Enter Python code..."
            value={node.config?.script || ""}
            onChange={(e) => {
              onUpdate(taskId, {
                config: { ...node.config, script: e.target.value },
              });
            }}
            className="font-mono min-h-[200px]"
          />
        </div>
      )}

      {/* Condition Config */}
      {node.type === "condition" && (
        <div className="grid gap-2">
          <Label htmlFor="condition_expr">Condition Expression</Label>
          <Input
            id="condition_expr"
            placeholder="e.g., {{ input.status }} == 'success'"
            value={node.config?.expression || ""}
            onChange={(e) => {
              onUpdate(taskId, {
                config: { ...node.config, expression: e.target.value },
              });
            }}
          />
        </div>
      )}


    </div>
  );
}
