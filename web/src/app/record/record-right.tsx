import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import type { TaskNodeRun } from "./types";

function getStatusVariant(
  status: string
): "default" | "secondary" | "destructive" | "outline" {
  switch (status) {
    case "success":
      return "default";
    case "running":
      return "secondary";
    case "failed":
      return "destructive";
    default:
      return "outline";
  }
}

function formatConfig(config: unknown) {
  try {
    return JSON.stringify(config, null, 2);
  } catch {
    return String(config);
  }
}

interface RecordRightProps {
  task: TaskNodeRun;
}

export function RecordRight({ task }: RecordRightProps) {
  return (
    <ScrollArea className="h-full">
      <div className="grid gap-6 m-4">
        <div className="grid gap-2">
          <div className="flex items-start justify-between gap-4">
            <div className="min-w-0">
              <h1 className="text-xl truncate">{task.name}</h1>
              <p className="text-sm text-muted-foreground font-mono truncate">
                {task.task_id}
              </p>
            </div>
            <Badge variant={getStatusVariant(task.status)} className="capitalize">
              {task.status}
            </Badge>
          </div>
          <Separator style={{ margin: "0" }} />
        </div>

        {task.description ? (
          <div className="grid gap-2">
            <h2 className="text-sm font-medium">Description</h2>
            <p className="text-sm text-muted-foreground">{task.description}</p>
          </div>
        ) : null}

        <div className="grid gap-3">
          <h2 className="text-sm font-medium">Runtime</h2>
          <div className="grid grid-cols-2 gap-3 text-sm">
            <div className="rounded-lg border bg-muted/30 p-3">
              <div className="text-muted-foreground">Type</div>
              <div className="font-medium capitalize">{task.type}</div>
            </div>
            <div className="rounded-lg border bg-muted/30 p-3">
              <div className="text-muted-foreground">Exit Code</div>
              <div className="font-medium">{task.exit_code}</div>
            </div>
            <div className="rounded-lg border bg-muted/30 p-3">
              <div className="text-muted-foreground">Created</div>
              <div className="font-medium break-words">{task.created_at || "-"}</div>
            </div>
            <div className="rounded-lg border bg-muted/30 p-3">
              <div className="text-muted-foreground">Started</div>
              <div className="font-medium break-words">{task.started_at || "-"}</div>
            </div>
            <div className="rounded-lg border bg-muted/30 p-3 col-span-2">
              <div className="text-muted-foreground">Finished</div>
              <div className="font-medium break-words">{task.finished_at || "-"}</div>
            </div>
          </div>
        </div>

        <div className="grid gap-2">
          <h2 className="text-sm font-medium">Output</h2>
          <pre className="rounded-lg border bg-muted/40 p-3 text-xs whitespace-pre-wrap break-words overflow-x-auto min-h-48">
            {task.output || "No output captured."}
          </pre>
        </div>

        <div className="grid gap-2">
          <h2 className="text-sm font-medium">Config</h2>
          <pre className="rounded-lg border bg-muted/20 p-3 text-xs whitespace-pre-wrap break-words overflow-x-auto">
            {formatConfig(task.config)}
          </pre>
        </div>
      </div>
    </ScrollArea>
  );
}
