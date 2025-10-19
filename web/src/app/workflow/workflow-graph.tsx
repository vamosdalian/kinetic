import * as React from "react";
import {
  ReactFlow,
  applyNodeChanges,
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
import { useSavedStore, useWorkflowStore } from "./workflow-store";

const initialNodes: Node[] = [];
const initialEdges: Edge[] = [];

function WorkflowGraph() {
  const { workflowName } = useParams();
  const [nodes, setNodes] = React.useState<Node[]>(initialNodes);
  const [edges, setEdges] = React.useState<Edge[]>(initialEdges);
  const [colorMode, setColorMode] = React.useState<ColorMode>("light");
  const { saved, setSaved } = useSavedStore();
  const [running, setRunning] = React.useState<boolean>(false);
  const [isReady, setIsReady] = React.useState<boolean>(false);
  const { getViewport } = useReactFlow();
  const { workflowData, setTaskId } = useWorkflowStore();

  const nodeTypes = {
    baseNodeFull: ActionBarNodeDemo,
  };

  const NewNode = React.useCallback(() => {
    const viewport = getViewport();
    const centerX = (window.innerWidth / 2 - viewport.x) / viewport.zoom;
    const centerY = (window.innerHeight / 2 - viewport.y) / viewport.zoom;
    const newNode: Node = {
      id: uuidv7(),
      position: { x: centerX, y: centerY },
      data: { name: `New Task` },
      type: "baseNodeFull",
    };
    return newNode;
  }, [getViewport]);

  React.useEffect(() => {
    console.log("Workflow Name from URL:", workflowName);
    if (!isReady) {
      return;
    }
    if (workflowName === "new") {
      setNodes(initialNodes.concat(NewNode()));
      setEdges(initialEdges);
      setSaved(false);
    } else {
      // Load existing workflow data based on workflowName
      // For example, fetch from an API and setNodes, setEdges accordingly
      // This is a placeholder for actual data fetching logic
      console.log(`Load workflow data for: ${workflowName}`);
    }
  }, [workflowName, NewNode, isReady, setSaved]);

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

  const addNode = React.useCallback(() => {
    setNodes((nds) => nds.concat(NewNode()));
    setSaved(false);
  }, [NewNode, setSaved]);

  const onNodesChange = React.useCallback(
    (changes: NodeChange[]) =>
      setNodes((nodesSnapshot) => applyNodeChanges(changes, nodesSnapshot)),
    []
  );
  const onEdgesChange = React.useCallback(
    (changes: EdgeChange[]) =>
      setEdges((edgesSnapshot) => applyEdgeChanges(changes, edgesSnapshot)),
    []
  );
  const onConnect = React.useCallback(
    (params: Connection) =>
      setEdges((edgesSnapshot) => addEdge(params, edgesSnapshot)),
    []
  );

  const onSelectionChange = React.useCallback(
    ({ nodes: selectedNodes }: { nodes: Node[] }) => {
      // props.setFocusedNode(selectedNodes.length === 1);
      if (selectedNodes.length === 1) {
        const selectedNode = selectedNodes[0];
        setTaskId(selectedNode.id);
      } else {
        setTaskId("");
      }
    },
    [setTaskId]
  );

  const onSave = React.useCallback(() => {
    console.log("Saving workflow...", { workflowData });
    setSaved(true);
  }, [setSaved, workflowData]);

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
          <Button variant="outline" size="icon" onClick={addNode}>
            <Plus></Plus>
          </Button>
        </div>
        <div className="absolute top-4 right-4 flex gap-2 z-10">
          {saved ? (
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
