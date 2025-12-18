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
import { useParams } from "react-router-dom";
import { useWorkflowStore } from "./workflow-store";
import { useDirtyStore } from "./dirty-store";
import { useSelection } from "./selection-context";

const initialEdges: Edge[] = [];

function WorkflowGraph() {
  const { workflowName } = useParams();
  const [colorMode, setColorMode] = React.useState<ColorMode>("light");
  const [running, setRunning] = React.useState<boolean>(false);
  const [isReady, setIsReady] = React.useState<boolean>(false);
  const { getViewport } = useReactFlow();

  const {
    workflowData,
    taskNodes,
    edges,
    addTaskNode,
    updateTaskNode,
    removeTaskNode,
    setEdges,
  } = useWorkflowStore();

  const { isDirty, markClean } = useDirtyStore();
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

  React.useEffect(() => {
    console.log("Workflow Name from URL:", workflowName);
    if (!isReady) {
      return;
    }
    if (workflowName === "new") {
      createNewNode();
      setEdges(initialEdges);
    } else {
      // Load existing workflow data based on workflowName
      console.log(`Load workflow data for: ${workflowName}`);
    }
  }, [workflowName, createNewNode, isReady, setEdges]);

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

  const onSave = React.useCallback(() => {
    console.log("Saving workflow...");
    console.log("Workflow:", workflowData);
    console.log("TaskNodes:", taskNodes);
    console.log("Edges:", edges);
    markClean();
  }, [edges, taskNodes, markClean, workflowData]);

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
          {!isDirty ? (
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
