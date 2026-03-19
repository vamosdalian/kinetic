import * as React from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  ReactFlow,
  type Edge,
  type Node,
  Background,
  Controls,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import {
  LoaderCircle,
  ArrowLeft,
  SquarePen,
  RotateCcw,
  SquareX,
} from "lucide-react";

import { apiClient } from "@/lib/api";
import { cn } from "@/lib/utils";
import {
  type WorkflowRunDetail,
  type WorkflowRunEvent,
} from "./types";
import { RunNode } from "./run-node";
import { RecordRight } from "./record-right";
import {
  getStatusBadgeClassName,
  isTerminalRunStatus,
} from "./status";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { toast } from "sonner";

const nodeTypes = {
  runNode: RunNode,
};

function parseEventData<T>(event: MessageEvent<string>) {
  try {
    return JSON.parse(event.data) as T;
  } catch {
    return null;
  }
}

export function RecordDetail() {
  const { runId } = useParams();
  const navigate = useNavigate();
  const [loading, setLoading] = React.useState(true);
  const [runData, setRunData] = React.useState<WorkflowRunDetail | null>(null);
  const [selectedTaskId, setSelectedTaskId] = React.useState("");
  const [rerunning, setRerunning] = React.useState(false);
  const [cancelling, setCancelling] = React.useState(false);
  const runStatus = runData?.status;

  const fetchRunDetail = React.useCallback(
    async (showLoader: boolean) => {
      if (!runId) {
        return;
      }

      if (showLoader) {
        setLoading(true);
      }

      try {
        const data = await apiClient<WorkflowRunDetail>(`/api/workflow_runs/${runId}`);
        setRunData(data);
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to load workflow run");
      } finally {
        if (showLoader) {
          setLoading(false);
        }
      }
    },
    [runId]
  );

  React.useEffect(() => {
    void fetchRunDetail(true);
  }, [fetchRunDetail]);

  const applyTaskUpdate = React.useCallback(
    (
      taskID: string,
      updater: (task: WorkflowRunDetail["taskNodes"][number]) => WorkflowRunDetail["taskNodes"][number]
    ) => {
      setRunData((prev) => {
        if (!prev) {
          return prev;
        }
        return {
          ...prev,
          taskNodes: prev.taskNodes.map((task) =>
            task.task_id === taskID ? updater(task) : task
          ),
        };
      });
    },
    []
  );

  React.useEffect(() => {
    if (!runId || !runStatus || isTerminalRunStatus(runStatus)) {
      return;
    }

    const source = new EventSource(`/api/workflow_runs/${runId}/events`);

    source.addEventListener("snapshot", (event) => {
      const payload = parseEventData<WorkflowRunDetail>(
        event as MessageEvent<string>
      );
      if (!payload) {
        return;
      }
      setRunData(payload);
      setLoading(false);
      if (isTerminalRunStatus(payload.status)) {
        source.close();
      }
    });

    source.addEventListener("run_status", (event) => {
      const payload = parseEventData<WorkflowRunEvent>(
        event as MessageEvent<string>
      );
      if (!payload?.status) {
        return;
      }
      setRunData((prev) => {
        if (!prev) {
          return prev;
        }
        return {
          ...prev,
          status: payload.status || prev.status,
          started_at: payload.started_at ?? prev.started_at,
          finished_at: payload.finished_at ?? prev.finished_at,
        };
      });
      if (isTerminalRunStatus(payload.status)) {
        source.close();
      }
    });

    source.addEventListener("task_status", (event) => {
      const payload = parseEventData<WorkflowRunEvent>(
        event as MessageEvent<string>
      );
      if (!payload?.task_id) {
        return;
      }
      applyTaskUpdate(payload.task_id, (task) => ({
        ...task,
        status: payload.status ?? task.status,
        assigned_node_id: payload.assigned_node_id ?? task.assigned_node_id,
        effective_tag: payload.effective_tag ?? task.effective_tag,
        assigned_at: payload.assigned_at ?? task.assigned_at,
        started_at: payload.started_at ?? task.started_at,
        finished_at: payload.finished_at ?? task.finished_at,
        exit_code: payload.exit_code ?? task.exit_code,
      }));
    });

    source.addEventListener("task_output", (event) => {
      const payload = parseEventData<WorkflowRunEvent>(
        event as MessageEvent<string>
      );
      if (!payload?.task_id || payload.output === undefined) {
        return;
      }
      applyTaskUpdate(payload.task_id, (task) => ({
        ...task,
        output: `${task.output || ""}${payload.output || ""}`,
      }));
    });

    source.onerror = () => {
      if (source.readyState === EventSource.CLOSED) {
        void fetchRunDetail(false);
      }
    };

    return () => {
      source.close();
    };
  }, [applyTaskUpdate, fetchRunDetail, runId, runStatus]);

  const selectedTask = React.useMemo(() => {
    return runData?.taskNodes.find((task) => task.task_id === selectedTaskId) ?? null;
  }, [runData, selectedTaskId]);

  const nodes = React.useMemo<Node[]>(() => {
    if (!runData) {
      return [];
    }

    return runData.taskNodes.map((task) => ({
      id: task.task_id,
      type: "runNode",
      position: task.position || { x: 0, y: 0 },
      data: {
        name: task.name,
        type: task.type,
        status: task.status,
        exit_code: task.exit_code,
        assigned_node_id: task.assigned_node_id,
        effective_tag: task.effective_tag,
      },
      draggable: false,
      selectable: true,
      selected: task.task_id === selectedTaskId,
    }));
  }, [runData, selectedTaskId]);

  const edges = React.useMemo<Edge[]>(() => {
    if (!runData) {
      return [];
    }

    return runData.edges.map((edge) => ({
      id: edge.edge_id,
      source: edge.source,
      target: edge.target,
      sourceHandle: edge.sourceHandle,
      targetHandle: edge.targetHandle,
    }));
  }, [runData]);

  React.useEffect(() => {
    if (!selectedTaskId || selectedTask) {
      return;
    }

    setSelectedTaskId("");
  }, [selectedTask, selectedTaskId]);

  const handleRerun = React.useCallback(async () => {
    if (!runData) {
      return;
    }

    setRerunning(true);
    try {
      const response = await apiClient<{ run_id: string }>(
        `/api/workflow_runs/${runData.run_id}/rerun`,
        {
          method: "POST",
        }
      );
      toast.success("Workflow run restarted");
      navigate(`/record/${response.run_id}`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to rerun workflow");
    } finally {
      setRerunning(false);
    }
  }, [navigate, runData]);

  const handleCancel = React.useCallback(async () => {
    if (!runData) {
      return;
    }

    setCancelling(true);
    try {
      await apiClient(`/api/workflow_runs/${runData.run_id}/cancel`, {
        method: "POST",
      });
      toast.success("Workflow run cancellation requested");
      await fetchRunDetail(false);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to cancel workflow");
    } finally {
      setCancelling(false);
    }
  }, [fetchRunDetail, runData]);

  if (loading) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoaderCircle className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!runData) {
    return <div>Failed to load workflow run data.</div>;
  }

  return (
    <div className="flex flex-col h-full w-full">
      <div className="border-b bg-background p-4 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate("/record")}>
            <ArrowLeft className="w-4 h-4" />
          </Button>
          <div>
            <h2 className="text-lg font-semibold">
              {runData.name}{" "}
              <span className="text-muted-foreground text-sm font-normal">
                v{runData.version}
              </span>
            </h2>
            <div className="text-sm text-muted-foreground font-mono">{runData.run_id}</div>
          </div>
        </div>
        <div className="flex gap-4 text-sm">
          <div className="flex items-center gap-2">
            <span className="text-muted-foreground">Status:</span>
            <Badge
              variant="outline"
              className={`capitalize ${getStatusBadgeClassName(runData.status)}`}
            >
              {runData.status}
            </Badge>
          </div>
          <div>
            <span className="text-muted-foreground">Created:</span>{" "}
            <span className="font-medium">{runData.create_at}</span>
          </div>
          <div>
            <span className="text-muted-foreground">Started:</span>{" "}
            <span className="font-medium">{runData.started_at || "-"}</span>
          </div>
          <div>
            <span className="text-muted-foreground">Finished:</span>{" "}
            <span className="font-medium">{runData.finished_at || "-"}</span>
          </div>
        </div>
      </div>

      <div className="flex-1 min-h-0 bg-muted/20 relative overflow-hidden">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodeClick={(_, node) => setSelectedTaskId(node.id)}
          onPaneClick={() => setSelectedTaskId("")}
          nodeTypes={nodeTypes}
          fitView
          attributionPosition="bottom-right"
          nodesConnectable={false}
          nodesDraggable={false}
          className="bg-background"
        >
          <Background />
          <Controls />
        </ReactFlow>

        <div
          className={cn(
            "absolute top-4 right-4 z-10 flex gap-2 transition-transform duration-300 ease-in-out",
            selectedTask ? "-translate-x-[660px]" : "translate-x-0"
          )}
        >
          <Button
            variant="outline"
            size="sm"
            onClick={() => navigate(`/workflow/${runData.workflow_id}`)}
          >
            <SquarePen className="w-4 h-4" />
            Edit
          </Button>

          <Button
            variant="outline"
            size="sm"
            disabled={rerunning}
            onClick={() => void handleRerun()}
          >
            {rerunning ? (
              <LoaderCircle className="w-4 h-4 animate-spin" />
            ) : (
              <RotateCcw className="w-4 h-4" />
            )}
            Rerun
          </Button>

          {!isTerminalRunStatus(runData.status) ? (
            <Button
              variant="destructive"
              size="sm"
              disabled={cancelling}
              onClick={() => void handleCancel()}
            >
              {cancelling ? (
                <LoaderCircle className="w-4 h-4 animate-spin" />
              ) : (
                <SquareX className="w-4 h-4" />
              )}
              Cancel
            </Button>
          ) : null}
        </div>

        <Card
          className={cn(
            "absolute top-2 right-2 bottom-2 w-[500px]",
            "shadow-2xl z-20 p-0 border-border",
            "transition-transform duration-300 ease-in-out",
            selectedTask ? "translate-x-0" : "translate-x-[calc(100%+1rem)]"
          )}
        >
          {selectedTask ? (
            <RecordRight task={selectedTask} workflowTag={runData.tag} />
          ) : (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
              Select a task to inspect its runtime details.
            </div>
          )}
        </Card>
      </div>
    </div>
  );
}
