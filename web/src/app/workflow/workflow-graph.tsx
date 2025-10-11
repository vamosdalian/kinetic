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
import type { JSX } from "react/jsx-runtime";
import { v7 as uuidv7 } from "uuid";

const initialNodes: Node[] = [];
const initialEdges: Edge[] = [];

function WorkflowGraph(props: JSX.IntrinsicAttributes) {
  // const { workflowName } = useParams();
  const [nodes, setNodes] = React.useState<Node[]>(initialNodes);
  const [edges, setEdges] = React.useState<Edge[]>(initialEdges);
  const [colorMode, setColorMode] = React.useState<ColorMode>("light");
  const [saved, setSaved] = React.useState<boolean>(true);
  const [running, setRunning] = React.useState<boolean>(false);
  const { getViewport } = useReactFlow();

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
    console.log("Adding new node");
    const viewport = getViewport();
    const centerX = (window.innerWidth / 2 - viewport.x) / viewport.zoom;
    const centerY = (window.innerHeight / 2 - viewport.y) / viewport.zoom;
    const newNode: Node = {
      id: uuidv7(),
      position: { x: centerX, y: centerY },
      data: { label: `New Task` },
      type: "baseNodeFull",
    };
    setNodes((nds) => nds.concat(newNode));
    setSaved(false);
  }, [getViewport]);

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

  const nodeTypes = {
    baseNodeFull: ActionBarNodeDemo,
  };

  const onSave = () => {
    // Implement your save logic here, e.g., send nodes and edges to a server
    console.log("Saving workflow...", { nodes, edges });
    setSaved(true);
  }

  return (
    <div style={{ width: "100%", height: "100%" }} className="position: relative;">
      <ReactFlow
        nodes={nodes}
        nodeTypes={nodeTypes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        colorMode={colorMode}
        fitView
        {...props}
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

export function FlowWithProvider(props: JSX.IntrinsicAttributes) {
  return (
    <ReactFlowProvider>
      <WorkflowGraph {...props} />
    </ReactFlowProvider>
  );
}
