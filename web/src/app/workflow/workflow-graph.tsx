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
import { Play, Save, LoaderCircle, Plus, CloudCheck, Settings } from "lucide-react";
import { useReactFlow } from "@xyflow/react";
import { ReactFlowProvider } from "@xyflow/react";
import { v7 as uuidv7 } from "uuid";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { defaultTaskNode, type WorkflowDetail, type WorkflowData, type TaskNode } from "./types";
import { apiClient } from "@/lib/api";
import { cn } from "@/lib/utils";
import { Card } from "@/components/ui/card";
import { WorkflowRight } from "./workflow-right";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useBlocker } from "react-router-dom";
import { toast } from "sonner";
import { validateWorkflowDefinition } from "./validation";

function WorkflowGraph() {
  // URL 中的 workflowId
  const { workflowId: urlWorkflowId } = useParams();
  const navigate = useNavigate();
  // URL 查询参数，action=create 表示新建，否则为更新
  const [searchParams] = useSearchParams();
  const action = searchParams.get("action");
  const [colorMode, setColorMode] = React.useState<ColorMode>("light");
  const [running, setRunning] = React.useState<boolean>(false);
  const [saving, setSaving] = React.useState<boolean>(false);
  const [loading, setLoading] = React.useState<boolean>(false);
  const [isReady, setIsReady] = React.useState<boolean>(false);
  const { getViewport, setViewport, fitView } = useReactFlow();

  // State Management
  const [workflowId, setWorkflowId] = React.useState<string>("");
  const [workflowData, setWorkflowData] = React.useState<WorkflowData>({
    name: "",
    description: "",
  });
  const [taskNodes, setTaskNodes] = React.useState<Record<string, TaskNode>>({});
  const [edges, setEdges] = React.useState<Edge[]>([]);

  const [isDirty, setIsDirty] = React.useState(false);
  const markDirty = React.useCallback(() => setIsDirty(true), []);
  const markClean = React.useCallback(() => setIsDirty(false), []);

  const [selectedTaskId, setSelectedTaskId] = React.useState<string>("");

  // Actions
  const updateWorkflowData = React.useCallback((data: Partial<WorkflowData>) => {
    setWorkflowData((prev) => ({ ...prev, ...data }));
    markDirty();
  }, [markDirty]);

  const addTaskNode = React.useCallback((id: string, position: { x: number; y: number }) => {
    setTaskNodes((prev) => ({
      ...prev,
      [id]: { ...defaultTaskNode, id, position },
    }));
    markDirty();
  }, [markDirty]);

  const updateTaskNode = React.useCallback((id: string, data: Partial<Omit<TaskNode, "id">>) => {
    setTaskNodes((prev) => {
      const existing = prev[id];
      if (!existing) return prev;
      return {
        ...prev,
        [id]: { ...existing, ...data },
      };
    });
    markDirty();
  }, [markDirty]);

  const removeTaskNode = React.useCallback((id: string) => {
    setTaskNodes((prev) => {
      const next = { ...prev };
      delete next[id];
      return next;
    });
    setEdges((prev) => prev.filter((e) => e.source !== id && e.target !== id));
    markDirty();
  }, [markDirty, setEdges]);
  
  const updateEdges = React.useCallback((newEdges: Edge[]) => {
      setEdges(newEdges);
      markDirty();
  }, [markDirty]);

  const clear = React.useCallback(() => {
    setWorkflowId("");
    setWorkflowData({ name: "", description: "" });
    setTaskNodes({});
    setEdges([]);
    markClean();
  }, [markClean]);

  const loadWorkflowData = React.useCallback((data: WorkflowDetail) => {
    const taskNodesRecord: Record<string, TaskNode> = {};
    if (data.taskNodes) {
      for (const task of data.taskNodes) {
        taskNodesRecord[task.id] = task;
      }
    }

    setWorkflowId(data.id);
    setWorkflowData({
      name: data.name,
      description: data.description,
    });
    setTaskNodes(taskNodesRecord);
    setEdges(data.edges || []);
    markClean();
  }, [markClean]);

  // React Router blocker for navigation protection
  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      isDirty && currentLocation.pathname !== nextLocation.pathname
  );

  // Handle browser close/refresh
  React.useEffect(() => {
    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (isDirty) {
        e.preventDefault();
        e.returnValue = "";
        return "";
      }
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, [isDirty]);

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

  // 初始化新 workflow，使用指定的 id
  const initNewWorkflow = React.useCallback((id: string) => {
    clear();
    // 使用传入的 workflowId
    setWorkflowId(id);
    // 创建初始 task，使用固定位置
    const taskId = uuidv7();
    addTaskNode(taskId, { x: 250, y: 200 });
    const initialEdges: Edge[] = [];
    setEdges(initialEdges);
    // 设置画布视图：zoom=1，居中显示节点
    // 使用 setTimeout 确保节点已渲染
    setTimeout(() => {
      setViewport({ x: 0, y: 0, zoom: 2 }, { duration: 0 });
    }, 0);
    // 新建 workflow 标记为 dirty
    markDirty();
  }, [clear, setWorkflowId, addTaskNode, setEdges, setViewport, markDirty]);

  // 加载已有 workflow 数据
  const fetchWorkflow = React.useCallback(async (id: string) => {
    setLoading(true);
    try {
      const data = await apiClient<WorkflowDetail>(`/api/workflows/${id}`);
      loadWorkflowData(data);
      // 加载完成后调整视图以适应所有节点
      setTimeout(() => {
        fitView({ padding: 0.2, duration: 0 });
      }, 0);
    } catch (error) {
      console.error("Failed to load workflow:", error);
    } finally {
      setLoading(false);
    }
  }, [loadWorkflowData, fitView]);

  React.useEffect(() => {
    if (!isReady || !urlWorkflowId) {
      return;
    }

    if (action === "create") {
      // 新建 workflow
      initNewWorkflow(urlWorkflowId);
    } else {
      // 加载已有 workflow
      fetchWorkflow(urlWorkflowId);
    }
  }, [urlWorkflowId, action, isReady, initNewWorkflow, fetchWorkflow]);

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
      updateEdges(newEdges);
    },
    [edges, updateEdges]
  );

  const onConnect = React.useCallback(
    (params: Connection) => {
      const newEdges = addEdge(params, edges);
      updateEdges(newEdges);
    },
    [edges, updateEdges]
  );

  const onPaneClick = React.useCallback(() => {
    setSelectedTaskId("");
  }, [setSelectedTaskId]);

  const onSave = React.useCallback(async () => {
    const validation = validateWorkflowDefinition(taskNodes, edges);
    if (!validation.valid) {
      toast.error(validation.errors[0] || "Workflow validation failed");
      return;
    }

    setSaving(true);

    try {
      const payload = {
        name: workflowData.name || "Untitled Workflow",
        description: workflowData.description || "",
        taskNodes: Object.values(taskNodes),
        edges: edges,
      };
      await apiClient(`/api/workflows/${workflowId}`, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });

      toast.success("Workflow saved successfully");
      markClean();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to save workflow");
    } finally {
      setSaving(false);
    }
  }, [workflowId, workflowData, taskNodes, edges, markClean]);

  const onRun = React.useCallback(async () => {
    if (!workflowId) {
      toast.error("Workflow ID is missing");
      return;
    }

    setRunning(true);
    try {
      const response = await apiClient<{ run_id: string }>(`/api/workflows/${workflowId}/run`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
      });
      toast.success("Workflow run started successfully");
      navigate(`/record/${response.run_id}`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to start workflow run");
    } finally {
      setRunning(false);
    }
  }, [workflowId, navigate]);

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
      className="relative w-full h-full overflow-hidden"
    >
      <ReactFlow
        nodes={localNodes}
        nodeTypes={nodeTypes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onPaneClick={onPaneClick}
        onInit={() => setIsReady(true)}
        colorMode={colorMode}
        defaultViewport={{ x: 0, y: 0, zoom: 1 }}
      >
        <Background />
        <Controls />
        <div className="absolute left-4 top-4 flex flex-col gap-2 z-10">
          <Button variant="outline" size="icon" onClick={createNewNode}>
            <Plus></Plus>
          </Button>
          <Button
            variant="outline"
            size="icon"
            onClick={() => setSelectedTaskId("ROOT")}
          >
            <Settings></Settings>
          </Button>
        </div>
        <div
          className={cn(
            "absolute top-4 right-4 flex gap-2 z-10 transition-transform duration-300 ease-in-out",
            selectedTaskId ? "-translate-x-[510px]" : "translate-x-0"
          )}
        >
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
            <Button variant="default" onClick={onRun} disabled={isDirty}>
              <Play />
              Run Workflow
            </Button>
          )}
        </div>
      </ReactFlow>

      {/* Drawer Layer */}
      <Card
        className={cn(
          "absolute top-2 right-2 bottom-2 w-[500px]",
          "shadow-2xl z-20 p-0 border-border",
          "transition-transform duration-300 ease-in-out",
          selectedTaskId ? "translate-x-0" : "translate-x-[calc(100%+1rem)]"
        )}
      >
        <WorkflowRight
          selectedTaskId={selectedTaskId}
          workflowData={workflowData}
          onUpdateWorkflowData={updateWorkflowData}
          taskNodes={taskNodes}
          onUpdateTaskNode={updateTaskNode}
        />
      </Card>

      <AlertDialog open={blocker.state === "blocked"}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Unsaved Changes</AlertDialogTitle>
            <AlertDialogDescription>
              You have unsaved changes. Are you sure you want to leave? Your
              changes will be lost.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => blocker.reset?.()}>
              Stay
            </AlertDialogCancel>
            <AlertDialogAction onClick={() => blocker.proceed?.()}>
              Leave
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
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
