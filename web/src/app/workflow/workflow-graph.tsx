import * as React from "react";
import {
  ReactFlow,
  applyEdgeChanges,
  applyNodeChanges,
  addEdge,
  type NodeChange,
  type EdgeChange,
  type Node,
  type Edge,
  type Connection,
  Background,
  Controls,
  type ColorMode,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Button } from "@/components/ui/button";
import { ActionBarNodeDemo } from "./custom-node";
import { Play, Save, LoaderCircle, Plus, CloudCheck } from "lucide-react";
import { useReactFlow } from "@xyflow/react";
import { ReactFlowProvider } from "@xyflow/react";
import { v7 as uuidv7 } from "uuid";
import { useParams, useNavigate } from "react-router-dom";
import { useWorkflowStore, type WorkflowDetail } from "./workflow-store";
import { useDirtyStore } from "./dirty-store";
import { useSelection } from "./selection-context";

const initialEdges: Edge[] = [];

function WorkflowGraph() {
  // URL 中的 workflowId，可能是 "new" 或者实际的 UUID
  const { workflowId: urlWorkflowId } = useParams();
  const navigate = useNavigate();
  const [colorMode, setColorMode] = React.useState<ColorMode>("light");
  const [running, setRunning] = React.useState<boolean>(false);
  const [saving, setSaving] = React.useState<boolean>(false);
  const [loading, setLoading] = React.useState<boolean>(false);
  const [isReady, setIsReady] = React.useState<boolean>(false);
  const { getViewport } = useReactFlow();

  const {
    workflowId,
    workflowData,
    taskNodes,
    edges,
    setWorkflowId,
    loadWorkflow,
    clear,
    addTaskNode,
    updateTaskNode,
    removeTaskNode,
    setEdges,
  } = useWorkflowStore();

  const { isDirty, markDirty, markClean } = useDirtyStore();
  const { selectedTaskId, setSelectedTaskId } = useSelection();

  // 本地 nodes state，用于实时拖动
  const [localNodes, setLocalNodes] = React.useState<Node[]>([]);

  const nodeTypes = React.useMemo(
    () => ({
      baseNodeFull: ActionBarNodeDemo,
    }),
    []
  );

  // 当 taskNodes 或 selectedTaskId 变化时，同步到 localNodes
  React.useEffect(() => {
    const newNodes = Object.values(taskNodes).map((task) => ({
      id: task.id,
      position: task.position,
      type: task.nodeType,
      data: { name: task.name, type: task.type },
      selected: task.id === selectedTaskId,
    }));
    setLocalNodes(newNodes);
  }, [taskNodes, selectedTaskId]);

  const createNewNode = React.useCallback(() => {
    const viewport = getViewport();
    const centerX = (window.innerWidth / 2 - viewport.x) / viewport.zoom;
    const centerY = (window.innerHeight / 2 - viewport.y) / viewport.zoom;
    const id = uuidv7();
    addTaskNode(id, { x: centerX, y: centerY });
  }, [getViewport, addTaskNode]);

  // 加载已有 workflow 数据
  const fetchWorkflow = React.useCallback(async (id: string) => {
    setLoading(true);
    try {
      const response = await fetch(`/api/workflows/${id}`);
      if (!response.ok) {
        if (response.status === 404) {
          console.error("Workflow not found:", id);
          navigate("/workflow");
          return;
        }
        throw new Error(`Failed to load workflow: ${response.statusText}`);
      }
      const data: WorkflowDetail = await response.json();
      console.log("Workflow loaded:", data);
      loadWorkflow(data);
    } catch (error) {
      console.error("Failed to load workflow:", error);
    } finally {
      setLoading(false);
    }
  }, [loadWorkflow, navigate]);

  // 初始化新 workflow
  const initNewWorkflow = React.useCallback(() => {
    clear();
    // 生成新的 workflowId
    const newId = uuidv7();
    setWorkflowId(newId);
    // 创建初始 task
    createNewNode();
    setEdges(initialEdges);
    // 新建 workflow 标记为 dirty
    markDirty();
  }, [clear, setWorkflowId, createNewNode, setEdges, markDirty]);

  React.useEffect(() => {
    console.log("Workflow ID from URL:", urlWorkflowId);
    if (!isReady) {
      return;
    }

    if (urlWorkflowId === "new") {
      // 新建 workflow
      initNewWorkflow();
    } else if (urlWorkflowId) {
      // 使用 URL 中的 ID 加载已有 workflow
      fetchWorkflow(urlWorkflowId);
    }
  }, [urlWorkflowId, isReady, initNewWorkflow, fetchWorkflow]);

  React.useEffect(() => {
    const updateColorMode = () => {
      const isDarkMode = document.documentElement.classList.contains("dark");
      setColorMode(isDarkMode ? "dark" : "light");
    };

    updateColorMode();

    const observer = new MutationObserver((mutationsList) => {
      for (const mutation of mutationsList) {
        if (
          mutation.type === "attributes" &&
          mutation.attributeName === "class"
        ) {
          updateColorMode();
        }
      }
    });

    observer.observe(document.documentElement, { attributes: true });

    return () => {
      observer.disconnect();
    };
  }, []);

  const onNodesChange = React.useCallback(
    (changes: NodeChange[]) => {
      // 实时应用所有变化到本地 state（包括拖动）
      setLocalNodes((nds) => applyNodeChanges(changes, nds));

      // 处理特定类型的变化
      for (const change of changes) {
        // 处理选中变化
        if (change.type === "select") {
          if (change.selected) {
            setSelectedTaskId(change.id);
          } else if (selectedTaskId === change.id) {
            setSelectedTaskId("");
          }
        }
        // 拖动结束时，同步位置到 store
        if (change.type === "position" && change.dragging === false && change.position) {
          updateTaskNode(change.id, { position: change.position });
        }
        // 处理删除
        if (change.type === "remove") {
          removeTaskNode(change.id);
        }
      }
    },
    [updateTaskNode, removeTaskNode, setSelectedTaskId, selectedTaskId]
  );

  const onEdgesChange = React.useCallback(
    (changes: EdgeChange[]) => {
      const newEdges = applyEdgeChanges(changes, edges);
      setEdges(newEdges);
    },
    [edges, setEdges]
  );

  const onConnect = React.useCallback(
    (params: Connection) => {
      const newEdges = addEdge(params, edges);
      setEdges(newEdges);
    },
    [edges, setEdges]
  );

  const onSave = React.useCallback(async () => {
    setSaving(true);

    try {
      const payload = {
        name: workflowData.name || "Untitled Workflow",
        description: workflowData.description || "",
        taskNodes: Object.values(taskNodes),
        edges: edges,
      };

      console.log("Saving workflow...", workflowId, payload);

      const response = await fetch(`/api/workflows/${workflowId}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        throw new Error(`Failed to save workflow: ${response.statusText}`);
      }

      const savedWorkflow = await response.json();
      console.log("Workflow saved:", savedWorkflow);

      // 如果是从 /workflow/new 保存的，更新 URL 为实际 ID
      if (urlWorkflowId === "new") {
        navigate(`/workflow/${workflowId}`, { replace: true });
      }

      markClean();
    } catch (error) {
      console.error("Failed to save workflow:", error);
      // TODO: Show error toast
    } finally {
      setSaving(false);
    }
  }, [workflowId, urlWorkflowId, workflowData, taskNodes, edges, markClean, navigate]);

  // 显示加载状态
  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <LoaderCircle className="w-8 h-8 animate-spin text-muted-foreground" />
        <span className="ml-2 text-muted-foreground">Loading workflow...</span>
      </div>
    );
  }

  return (
    <div
      style={{ width: "100%", height: "100%" }}
      className="position: relative;"
    >
      <ReactFlow
        nodes={localNodes}
        nodeTypes={nodeTypes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onInit={() => setIsReady(true)}
        colorMode={colorMode}
        fitView
      >
        <Background />
        <Controls />
        <div className="absolute left-4 top-4 flex flex-col gap-2 z-10">
          <Button variant="outline" size="icon" onClick={createNewNode}>
            <Plus></Plus>
          </Button>
        </div>
        <div className="absolute top-4 right-4 flex gap-2 z-10">
          {saving ? (
            <Button variant="outline" disabled>
              <LoaderCircle className="mr-2 animate-spin" />
              Saving...
            </Button>
          ) : !isDirty ? (
            <Button variant="outline" disabled>
              <CloudCheck></CloudCheck>
              Saved
            </Button>
          ) : (
            <Button variant="outline" onClick={onSave}>
              <Save></Save>
              Save
            </Button>
          )}
          {running ? (
            <Button variant="default" disabled>
              <LoaderCircle className="mr-2 animate-spin" />
              Running...
            </Button>
          ) : (
            <Button variant="default" onClick={() => setRunning(true)}>
              <Play />
              Run Workflow
            </Button>
          )}
        </div>
      </ReactFlow>
    </div>
  );
}

export function FlowWithProvider() {
  return (
    <ReactFlowProvider>
      <WorkflowGraph />
    </ReactFlowProvider>
  );
}
