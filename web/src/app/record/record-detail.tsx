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
import { LoaderCircle, ArrowLeft, SquarePen } from "lucide-react";

import { apiClient } from "@/lib/api";
import { cn } from "@/lib/utils";
import { type WorkflowRunDetail } from "./types";
import { RunNode } from "./run-node";
import { RecordRight } from "./record-right";
import { getStatusBadgeClassName } from "./status";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";

const nodeTypes = {
  runNode: RunNode,
};

export function RecordDetail() {
  const { runId } = useParams();
  const navigate = useNavigate();
  const [loading, setLoading] = React.useState(true);
  const [runData, setRunData] = React.useState<WorkflowRunDetail | null>(null);
  const [selectedTaskId, setSelectedTaskId] = React.useState("");

  const fetchRunDetail = React.useCallback(
    async (showLoader: boolean) => {
      if (!runId) return;

      if (showLoader) {
        setLoading(true);
      }

      try {
        const data = await apiClient<WorkflowRunDetail>(`/api/workflow_runs/${runId}`);
        setRunData(data);
      } catch (err) {
        console.error("Failed to fetch run detail:", err);
      } finally {
        if (showLoader) {
          setLoading(false);
        }
      }
    },
    [runId]
  );

  React.useEffect(() => {
    fetchRunDetail(true);
  }, [fetchRunDetail]);

  React.useEffect(() => {
    if (!runId || !runData || !["created", "running"].includes(runData.status)) {
      return;
    }

    const timer = window.setInterval(() => {
      void fetchRunDetail(false);
    }, 2000);

    return () => {
      window.clearInterval(timer);
    };
  }, [fetchRunDetail, runData, runId]);

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
            <span className="font-medium">{runData.started_at}</span>
          </div>
          <div>
            <span className="text-muted-foreground">Finished:</span>{" "}
            <span className="font-medium">{runData.finished_at}</span>
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
            "absolute top-4 right-4 z-10 transition-transform duration-300 ease-in-out",
            selectedTask ? "-translate-x-[510px]" : "translate-x-0"
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
            <RecordRight task={selectedTask} />
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
