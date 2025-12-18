import * as React from "react";
import {
  ReactFlow,
  applyEdgeChanges,
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
    selectTask,
    addTaskNode,
    updateTaskNode,
    removeTaskNode,
    setEdges,
  } = useWorkflowStore();

  const { isDirty, markClean } = useDirtyStore();

  const nodeTypes = {
    baseNodeFull: ActionBarNodeDemo,
  };

  // 从 taskNodes 派生 ReactFlow nodes
  const nodes: Node[] = React.useMemo(() => {
    return Object.values(taskNodes).map((task) => ({
      id: task.id,
      position: task.position,
      type: task.nodeType,
      data: { name: task.name, type: task.type },
    }));
  }, [taskNodes]);

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
      // For example, fetch from an API and setNodes, setEdges accordingly
      // This is a placeholder for actual data fetching logic
      console.log(`Load workflow data for: ${workflowName}`);
    }
  }, [workflowName, createNewNode, isReady, setEdges]);

  React.useEffect(() => {
    // Function to update color mode based on the presence of 'dark' class on <html>
    const updateColorMode = () => {
      const isDarkMode = document.documentElement.classList.contains("dark");
      setColorMode(isDarkMode ? "dark" : "light");
    };

    // Set the initial color mode when the component mounts
    updateColorMode();

    // Create an observer to watch for class changes on the <html> element
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

    // Start observing the <html> element
    observer.observe(document.documentElement, { attributes: true });

    // Cleanup function to disconnect the observer when the component unmounts
    return () => {
      observer.disconnect();
    };
  }, []);

  const onNodesChange = React.useCallback(
    (changes: NodeChange[]) => {
      // 处理位置变化，同步到 taskNodes
      for (const change of changes) {
        if (change.type === "position" && change.position) {
          updateTaskNode(change.id, { position: change.position });
        }
        if (change.type === "remove") {
          removeTaskNode(change.id);
        }
      }

      // 对于 select 和其他变化，ReactFlow 内部处理即可
      console.log("onNodesChange", changes);
    },
    [updateTaskNode, removeTaskNode]
  );

  const onEdgesChange = React.useCallback(
    (changes: EdgeChange[]) => {
      const newEdges = applyEdgeChanges(changes, edges);
      console.log("onEdgesChange", changes);
      setEdges(newEdges);
    },
    [edges, setEdges]
  );

  const onConnect = React.useCallback(
    (params: Connection) => {
      const newEdges = addEdge(params, edges);
      console.log("onConnect", params);
      setEdges(newEdges);
    },
    [edges, setEdges]
  );

  const onSelectionChange = React.useCallback(
    ({ nodes: selectedNodes }: { nodes: Node[] }) => {
      if (selectedNodes.length === 1) {
        const selectedNode = selectedNodes[0];
        selectTask(selectedNode.id);
      } else {
        selectTask("");
      }
    },
    [selectTask]
  );

  const onSave = React.useCallback(() => {
    console.log("Saving workflow...");
    console.log("Workflow:", workflowData);
    console.log("TaskNodes:", taskNodes);
    console.log("Edges:", edges);
    // TODO: Implement actual save logic (API call)
    markClean();
  }, [edges, taskNodes, markClean, workflowData]);

  return (
    <div
      style={{ width: "100%", height: "100%" }}
      className="position: relative;"
    >
      <ReactFlow
        nodes={nodes}
        nodeTypes={nodeTypes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onSelectionChange={onSelectionChange}
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
