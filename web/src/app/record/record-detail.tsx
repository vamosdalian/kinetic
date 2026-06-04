import * as React from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  ReactFlow,
  type Edge,
  type Node,
  Background,
  Controls,
  type ColorMode,
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
import { formatDashboardDateTime } from "@/lib/dashboard";
import { buildEventSourceURL } from "@/lib/auth";
import { cn } from "@/lib/utils";
import {
  type EdgeRun,
  type TaskNodeRun,
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
import { DETAIL_PANEL_LAYOUT_STYLE } from "@/components/detail-panel-layout";
import { SiteHeader } from "@/components/site-header";
import { useTheme } from "next-themes";
import { toast } from "sonner";

const nodeTypes = {
  runNode: RunNode,
};

function getTaskSortValue(task: TaskNodeRun) {
  return `${task.created_at || ""}:${task.task_run_id || ""}`;
}

function pickLatestTask(tasks: TaskNodeRun[]) {
  return [...tasks].sort((a, b) => {
    const loopCompare = (b.loop_index ?? -1) - (a.loop_index ?? -1);
    if (loopCompare !== 0) {
      return loopCompare;
    }
    return getTaskSortValue(b).localeCompare(getTaskSortValue(a));
  })[0];
}

function getLoopOptions(tasks: TaskNodeRun[], forTaskID: string) {
  const optionMap = new Map<number, number>();
  for (const task of tasks) {
    if (task.loop_id !== forTaskID || task.loop_index === undefined || task.loop_index < 0) {
      continue;
    }
    optionMap.set(task.loop_index, task.loop_value ?? task.loop_index);
  }
  return [...optionMap.entries()]
    .sort(([a], [b]) => a - b)
    .map(([index, value]) => ({ index, value }));
}

function getForBodyTaskIDs(forTaskID: string, edges: EdgeRun[]) {
  const outgoing = new Map<string, EdgeRun[]>();
  for (const edge of edges) {
    const list = outgoing.get(edge.source) ?? [];
    list.push(edge);
    outgoing.set(edge.source, list);
  }

  const bodyTaskIDs = new Set<string>();
  const queue = edges
    .filter((edge) => edge.source === forTaskID && edge.sourceHandle === "body")
    .map((edge) => edge.target);

  while (queue.length > 0) {
    const taskID = queue.shift();
    if (!taskID || bodyTaskIDs.has(taskID)) {
      continue;
    }
    bodyTaskIDs.add(taskID);

    for (const edge of outgoing.get(taskID) ?? []) {
      queue.push(edge.target);
    }
  }

  return bodyTaskIDs;
}

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
  const { resolvedTheme } = useTheme();
  const [loading, setLoading] = React.useState(true);
  const [colorMode, setColorMode] = React.useState<ColorMode>("light");
  const [runData, setRunData] = React.useState<WorkflowRunDetail | null>(null);
  const [selectedTaskId, setSelectedTaskId] = React.useState("");
  const [selectedLoopByForTaskID, setSelectedLoopByForTaskID] = React.useState<Record<string, number>>({});
  const [rerunning, setRerunning] = React.useState(false);
  const [cancelling, setCancelling] = React.useState(false);
  const runDataRef = React.useRef<WorkflowRunDetail | null>(null);
  const runStatus = runData?.status;

  React.useEffect(() => {
    runDataRef.current = runData;
  }, [runData]);

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

  React.useEffect(() => {
    setColorMode(resolvedTheme === "dark" ? "dark" : "light");
  }, [resolvedTheme]);

  const applyTaskUpdate = React.useCallback(
    (
      taskID: string,
      taskRunID: string | undefined,
      updater: (task: WorkflowRunDetail["taskNodes"][number]) => WorkflowRunDetail["taskNodes"][number]
    ) => {
      setRunData((prev) => {
        if (!prev) {
          return prev;
        }
        return {
          ...prev,
          taskNodes: prev.taskNodes.map((task) =>
            (taskRunID ? task.task_run_id === taskRunID : task.task_id === taskID) ? updater(task) : task
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

    const source = new EventSource(buildEventSourceURL(`/api/workflow_runs/${runId}/events`));

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
      const hasTarget = runDataRef.current?.taskNodes.some((task) =>
        payload.task_run_id ? task.task_run_id === payload.task_run_id : task.task_id === payload.task_id
      );
      if (!hasTarget) {
        void fetchRunDetail(false);
        return;
      }
      applyTaskUpdate(payload.task_id, payload.task_run_id, (task) => ({
        ...task,
        status: payload.status ?? task.status,
        assigned_node_id: payload.assigned_node_id ?? task.assigned_node_id,
        effective_tag: payload.effective_tag ?? task.effective_tag,
        assigned_at: payload.assigned_at ?? task.assigned_at,
        started_at: payload.started_at ?? task.started_at,
        finished_at: payload.finished_at ?? task.finished_at,
        result: payload.result ?? task.result,
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
      const hasTarget = runDataRef.current?.taskNodes.some((task) =>
        payload.task_run_id ? task.task_run_id === payload.task_run_id : task.task_id === payload.task_id
      );
      if (!hasTarget) {
        void fetchRunDetail(false);
        return;
      }
      applyTaskUpdate(payload.task_id, payload.task_run_id, (task) => ({
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

  const visibleTasks = React.useMemo(() => {
    if (!runData) {
      return [];
    }

    const tasksByID = new Map<string, TaskNodeRun[]>();
    for (const task of runData.taskNodes) {
      const list = tasksByID.get(task.task_id) ?? [];
      list.push(task);
      tasksByID.set(task.task_id, list);
    }

    const selectedLoopScopes = runData.taskNodes
      .filter((task) => task.type === "for")
      .map((task) => {
        const options = getLoopOptions(runData.taskNodes, task.task_id);
        return {
          taskID: task.task_id,
          bodyTaskIDs: getForBodyTaskIDs(task.task_id, runData.edges),
          loopIndex: selectedLoopByForTaskID[task.task_id] ?? options[options.length - 1]?.index,
        };
      })
      .filter((scope) => scope.loopIndex !== undefined);

    const visible: TaskNodeRun[] = [];
    for (const [taskID, tasks] of tasksByID.entries()) {
      const scope = selectedLoopScopes.find((item) => item.bodyTaskIDs.has(taskID));
      if (scope) {
        const scopedTask = pickLatestTask(
          tasks.filter((task) => task.loop_id === scope.taskID && task.loop_index === scope.loopIndex)
        );
        if (scopedTask) {
          visible.push(scopedTask);
          continue;
        }
      }

      const baseTask = pickLatestTask(tasks.filter((task) => (task.loop_index ?? -1) < 0));
      const fallbackTask = baseTask ?? pickLatestTask(tasks);
      if (fallbackTask) {
        visible.push(fallbackTask);
      }
    }

    return visible;
  }, [runData, selectedLoopByForTaskID]);

  const selectedTask = React.useMemo(() => {
    return visibleTasks.find((task) => task.task_id === selectedTaskId) ?? null;
  }, [selectedTaskId, visibleTasks]);

  const selectedLoopOptions = React.useMemo(() => {
    if (!runData || selectedTask?.type !== "for") {
      return [];
    }
    return getLoopOptions(runData.taskNodes, selectedTask.task_id);
  }, [runData, selectedTask]);

  const nodes = React.useMemo<Node[]>(() => {
    if (!runData) {
      return [];
    }

    return visibleTasks.map((task) => ({
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
  }, [runData, selectedTaskId, visibleTasks]);

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
    <div className="flex h-full w-full flex-col">
      <SiteHeader
        breadcrumbs={[
          { label: "Record", href: "/record" },
          { label: runData.name, href: null },
        ]}
      />
      <div className="flex min-h-(--header-height) shrink-0 items-center justify-between border-b bg-background px-4 py-3 lg:px-6">
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
            <span className="font-medium">{formatDashboardDateTime(runData.create_at)}</span>
          </div>
          <div>
            <span className="text-muted-foreground">Started:</span>{" "}
            <span className="font-medium">{formatDashboardDateTime(runData.started_at || "")}</span>
          </div>
          <div>
            <span className="text-muted-foreground">Finished:</span>{" "}
            <span className="font-medium">{formatDashboardDateTime(runData.finished_at || "")}</span>
          </div>
        </div>
      </div>

      <div
        className="relative min-h-0 flex-1 overflow-hidden bg-muted/20"
        style={DETAIL_PANEL_LAYOUT_STYLE}
      >
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodeClick={(_, node) => setSelectedTaskId(node.id)}
          onPaneClick={() => setSelectedTaskId("")}
          nodeTypes={nodeTypes}
          colorMode={colorMode}
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
            selectedTask ? "-translate-x-[var(--detail-panel-button-shift)]" : "translate-x-0"
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
            "absolute top-2 right-2 bottom-2 w-[var(--detail-panel-width)]",
            "shadow-2xl z-20 p-0 border-border",
            "transition-transform duration-300 ease-in-out",
            selectedTask ? "translate-x-0" : "translate-x-[calc(100%+1rem)]"
          )}
        >
          {selectedTask ? (
            <RecordRight
              task={selectedTask}
              workflowTag={runData.tag}
              loopOptions={selectedLoopOptions}
              selectedLoopIndex={
                selectedTask.type === "for"
                  ? selectedLoopByForTaskID[selectedTask.task_id] ?? selectedLoopOptions[selectedLoopOptions.length - 1]?.index
                  : undefined
              }
              onLoopIndexChange={(loopIndex) => {
                if (selectedTask.type !== "for") {
                  return;
                }
                setSelectedLoopByForTaskID((prev) => ({
                  ...prev,
                  [selectedTask.task_id]: loopIndex,
                }));
              }}
            />
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
