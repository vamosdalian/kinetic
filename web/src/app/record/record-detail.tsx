import * as React from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  ReactFlow,
  type Edge,
  type Node,
  Background,
  Controls,
  useNodesState,
  useEdgesState,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { LoaderCircle, ArrowLeft } from "lucide-react";

import { apiClient } from "@/lib/api";
import { type WorkflowRunDetail } from "./types";
import { RunNode } from "./run-node";
import { Button } from "@/components/ui/button";

const nodeTypes = {
  runNode: RunNode,
};

export function RecordDetail() {
  const { runId } = useParams();
  const navigate = useNavigate();
  const [loading, setLoading] = React.useState(true);
  const [runData, setRunData] = React.useState<WorkflowRunDetail | null>(null);
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  React.useEffect(() => {
    if (!runId) return;

    setLoading(true);
    apiClient<WorkflowRunDetail>(`/api/workflow_runs/${runId}`)
      .then((data) => {
        setRunData(data);
        
        // Transform taskNodes to ReactFlow Nodes
        const flowNodes: Node[] = data.taskNodes.map((task) => ({
          id: task.task_id, // Use task_id to match edge source/target? Wait, edges usually use valid UUIDs.
          // In the backend, edges connect task IDs or node IDs. 
          // Let's verify edge source/target in existing workflow. 
          // Usually edges connect Node.id. 
          // In DTO: TaskNodeRun has TaskID. EdgeRun has Source/Target.
          // Assuming Source/Target in EdgeRun correspond to TaskID. 
          
          type: "runNode",
          position: task.position || { x: 0, y: 0 },
          data: { 
            name: task.name, 
            type: task.type,
            status: task.status,
            exit_code: task.exit_code
          },
          draggable: false, 
          selectable: true,
        }));

        const flowEdges: Edge[] = data.edges.map((edge) => ({
          id: edge.edge_id,
          source: edge.source,
          target: edge.target,
          sourceHandle: edge.sourceHandle,
          targetHandle: edge.targetHandle,
        }));

        setNodes(flowNodes);
        setEdges(flowEdges);
      })
      .catch((err) => {
        console.error("Failed to fetch run detail:", err);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [runId, setNodes, setEdges]);

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
                 <h2 className="text-lg font-semibold">{runData.name} <span className="text-muted-foreground text-sm font-normal">v{runData.version}</span></h2>
                <div className="text-sm text-muted-foreground font-mono">{runData.run_id}</div>
            </div>
         </div>
         <div className="flex gap-4 text-sm">
             <div><span className="text-muted-foreground">Status:</span> <span className="font-medium">{runData.status}</span></div>
             <div><span className="text-muted-foreground">Started:</span> <span className="font-medium">{runData.started_at}</span></div>
             <div><span className="text-muted-foreground">Finished:</span> <span className="font-medium">{runData.finished_at}</span></div>
         </div>
      </div>
      <div className="flex-1 min-h-0 bg-muted/20">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
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
      </div>
    </div>
  );
}
