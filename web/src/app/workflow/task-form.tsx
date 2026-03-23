import * as React from "react";
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
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import { CircleQuestionMark } from "lucide-react";
import { KeyValueEditor } from "./key-value-editor";
import {
  createTaskConfig,
  type ConditionConfig,
  type HttpConfig,
  type PythonConfig,
  type ShellConfig,
  type TaskConfig,
  type TaskNode,
  type TaskPolicy,
  type TaskType,
} from "./types";

interface TaskFormProps {
  taskId: string;
  node: TaskNode | null;
  tagOptions: string[];
  workflowTag: string;
  onUpdate: (id: string, data: Partial<Omit<TaskNode, "id">>) => void;
}

const INHERIT_TAG_VALUE = "__inherit_workflow_tag__";

function HelpHint({ content }: { content: React.ReactNode }) {
  return (
    <HoverCard openDelay={150} closeDelay={100}>
      <HoverCardTrigger asChild>
        <button
          type="button"
          className="inline-flex h-4 w-4 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
          aria-label="Show help"
        >
          <CircleQuestionMark className="h-4 w-4" />
        </button>
      </HoverCardTrigger>
      <HoverCardContent align="start" className="w-72 text-sm leading-5">
        {content}
      </HoverCardContent>
    </HoverCard>
  );
}

export function Taskform({
  taskId,
  node,
  tagOptions,
  workflowTag,
  onUpdate,
}: TaskFormProps) {
  const config = React.useMemo(() => {
    if (!node) {
      return null;
    }

    return (node.config ?? createTaskConfig(node.type)) as TaskConfig;
  }, [node]);

  const updateConfig = React.useCallback(
    (nextConfig: TaskConfig) => {
      if (!node) {
        return;
      }

      onUpdate(taskId, { config: nextConfig });
    },
    [node, onUpdate, taskId]
  );

  const updatePolicy = React.useCallback(
    (field: keyof TaskPolicy, value: string) => {
      if (!config) {
        return;
      }

      const nextValue = value === "" ? undefined : Number(value);
      updateConfig({
        ...config,
        [field]: Number.isFinite(nextValue) ? nextValue : undefined,
      });
    },
    [config, updateConfig]
  );

  if (!node || !config) {
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
        <Separator style={{ margin: "0" }} />
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
            onUpdate(taskId, { type: value, config: createTaskConfig(value) });
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

      <div className="grid gap-2">
        <div className="flex items-center gap-2">
          <Label htmlFor="task_tag">Tag</Label>
          <HelpHint
            content={
              node.tag
                ? "This task overrides the workflow-level routing tag."
                : workflowTag
                  ? `This task will run on nodes tagged ${workflowTag}.`
                  : "This task can run on any healthy node unless you choose a tag."
            }
          />
        </div>
        <Select
          value={node.tag || INHERIT_TAG_VALUE}
          onValueChange={(value) => {
            onUpdate(taskId, { tag: value === INHERIT_TAG_VALUE ? "" : value });
          }}
        >
          <SelectTrigger id="task_tag" className="w-full">
            <SelectValue placeholder="Select node tag" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={INHERIT_TAG_VALUE}>
              {workflowTag ? `Inherit workflow tag (${workflowTag})` : "Inherit workflow tag"}
            </SelectItem>
            {tagOptions.map((tag) => (
              <SelectItem key={tag} value={tag}>
                {tag}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {node.type === "shell" && (
        <div className="grid gap-2">
          <Label>Script</Label>
          <ShellEditor
            value={(config as ShellConfig).script}
            onChange={(value) =>
              updateConfig({ ...(config as ShellConfig), script: value })
            }
          >
            <Button variant="outline" className="w-full">
              Edit Script
            </Button>
          </ShellEditor>
        </div>
      )}

      {node.type === "http" && (
        <>
          <div className="grid gap-2">
            <Label htmlFor="http_url">URL</Label>
            <Input
              id="http_url"
              placeholder="https://api.example.com/endpoint"
              value={(config as HttpConfig).url || ""}
              onChange={(e) => {
                updateConfig({ ...(config as HttpConfig), url: e.target.value });
              }}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="http_method">Method</Label>
            <Select
              value={(config as HttpConfig).method || "GET"}
              onValueChange={(value) => {
                updateConfig({
                  ...(config as HttpConfig),
                  method: value as "GET" | "POST" | "PUT" | "DELETE",
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

          <div className="grid gap-3">
            <div className="flex items-center justify-between gap-3">
              <div className="flex items-center gap-2">
                <Label>Headers</Label>
                <HelpHint content="Configure request headers in the task config. Keys are unique within this map." />
              </div>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => {
                  const headers = { ...((config as HttpConfig).headers ?? {}) };
                  let index = Object.keys(headers).length + 1;
                  let candidate = `header-${index}`;

                  while (candidate in headers) {
                    index += 1;
                    candidate = `header-${index}`;
                  }

                  headers[candidate] = "";
                  updateConfig({ ...(config as HttpConfig), headers });
                }}
              >
                Add
              </Button>
            </div>
            <KeyValueEditor
              values={(config as HttpConfig).headers ?? {}}
              onChange={(headers) => {
                updateConfig({ ...(config as HttpConfig), headers });
              }}
              emptyText="No custom headers configured."
              keyPlaceholder="Header name"
              valuePlaceholder="Header value"
              keyPrefix="header"
              showAddButton={false}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="http_body">Body</Label>
            <Textarea
              id="http_body"
              placeholder='{"hello":"world"}'
              value={(config as HttpConfig).body || ""}
              onChange={(e) => {
                updateConfig({ ...(config as HttpConfig), body: e.target.value });
              }}
              className="font-mono min-h-[120px]"
            />
          </div>
        </>
      )}

      {node.type === "python" && (
        <div className="grid gap-2">
          <Label>Python Script</Label>
          <Textarea
            placeholder="Enter Python code..."
            value={(config as PythonConfig).script || ""}
            onChange={(e) => {
              updateConfig({ ...(config as PythonConfig), script: e.target.value });
            }}
            className="font-mono min-h-[200px]"
          />
        </div>
      )}

      {node.type === "condition" && (
        <div className="grid gap-2">
          <div className="flex items-center gap-2">
            <Label htmlFor="condition_expr">Condition Expression</Label>
            <HelpHint
              content={
                <>
                  Supported fields: <code>status</code>, <code>exit_code</code>, <code>output</code>, <code>json</code>, <code>json.field</code>
                </>
              }
            />
          </div>
          <Input
            id="condition_expr"
            placeholder='e.g., json.ok == true'
            value={(config as ConditionConfig).expression || ""}
            onChange={(e) => {
              updateConfig({
                ...(config as ConditionConfig),
                expression: e.target.value,
              });
            }}
          />
        </div>
      )}

      <div className="grid gap-2">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <Label>Environment Variables</Label>
            <HelpHint
              content="Task-level variables override workflow-level variables. Names starting with KINETIC_ are reserved for the system. Keys are unique within this map."
            />
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => {
              const env = { ...(config.env ?? {}) };
              let index = Object.keys(env).length + 1;
              let candidate = `env-${index}`;

              while (candidate in env) {
                index += 1;
                candidate = `env-${index}`;
              }

              env[candidate] = "";
              updateConfig({
                ...config,
                env,
              });
            }}
          >
            Add
          </Button>
        </div>
        <KeyValueEditor
          values={config.env ?? {}}
          onChange={(env) => {
            updateConfig({
              ...config,
              env,
            });
          }}
          emptyText="No task environment variables configured."
          keyPlaceholder="Variable name"
          valuePlaceholder="Variable value"
          keyPrefix="env"
          showAddButton={false}
        />
      </div>

      <div className="grid gap-3">
        <div className="flex items-center gap-2">
          <h2 className="text-sm font-medium">Execution Policy</h2>
          <HelpHint content="Timeout and retry settings are applied per task run." />
        </div>
        <div className="grid grid-cols-3 gap-3">
          <div className="grid gap-2">
            <Label htmlFor="timeout_seconds">Timeout (s)</Label>
            <Input
              id="timeout_seconds"
              type="number"
              min="0"
              placeholder="0"
              value={config.timeout_seconds ?? ""}
              onChange={(e) => updatePolicy("timeout_seconds", e.target.value)}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="retry_count">Retry Count</Label>
            <Input
              id="retry_count"
              type="number"
              min="0"
              placeholder="0"
              value={config.retry_count ?? ""}
              onChange={(e) => updatePolicy("retry_count", e.target.value)}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="retry_backoff_seconds">Retry Backoff (s)</Label>
            <Input
              id="retry_backoff_seconds"
              type="number"
              min="0"
              placeholder="0"
              value={config.retry_backoff_seconds ?? ""}
              onChange={(e) => updatePolicy("retry_backoff_seconds", e.target.value)}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
