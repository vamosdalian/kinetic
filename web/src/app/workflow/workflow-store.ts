import { create } from "zustand";
import { type Edge } from "@xyflow/react";
import { useDirtyStore } from "./dirty-store";

// ============ Task Type Configs ============

export type TaskType = "shell" | "http" | "python" | "condition";

export interface ShellConfig {
  script: string;
}

export interface HttpConfig {
  url: string;
  method: "GET" | "POST" | "PUT" | "DELETE";
  headers?: Record<string, string>;
  body?: string;
}

export interface PythonConfig {
  script: string;
  requirements?: string[];
}

export interface ConditionConfig {
  expression: string;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type TaskConfig = any; // ShellConfig | HttpConfig | PythonConfig | ConditionConfig | any

// ============ TaskNode (merged Node + Task) ============

export interface TaskNode {
  id: string;

  // 业务数据
  name: string;
  description: string;
  type: TaskType;
  config: TaskConfig;

  // 视觉数据
  position: { x: number; y: number };
  nodeType: string; // ReactFlow node type, e.g., "baseNodeFull"
}

const defaultTaskNode: Omit<TaskNode, "id" | "position"> = {
  name: "New Task",
  description: "",
  type: "shell",
  config: { script: "" } as ShellConfig,
  nodeType: "baseNodeFull",
};

// ============ Workflow Data ============

export interface WorkflowData {
  name: string;
  description: string;
}

// ============ Workflow State ============

interface WorkflowState {
  // Workflow 元信息
  workflowId: string;
  workflowData: WorkflowData;

  // 核心数据（单一数据源）
  taskNodes: Record<string, TaskNode>;
  edges: Edge[];

  // UI 状态
  selectedTaskId: string;

  // Actions - Workflow
  clear: () => void;
  setWorkflowId: (id: string) => void;
  setWorkflowData: (data: Partial<WorkflowData>) => void;

  // Actions - TaskNodes
  addTaskNode: (id: string, position: { x: number; y: number }) => void;
  updateTaskNode: (id: string, data: Partial<Omit<TaskNode, "id">>) => void;
  removeTaskNode: (id: string) => void;

  // Actions - Edges
  setEdges: (edges: Edge[]) => void;

  // Actions - UI
  selectTask: (id: string) => void;
}

// Helper to mark dirty
const markDirty = () => useDirtyStore.getState().markDirty();

export const useWorkflowStore = create<WorkflowState>()((set) => ({
  // Initial state
  workflowId: "",
  workflowData: {
    name: "",
    description: "",
  },
  taskNodes: {},
  edges: [],
  selectedTaskId: "",

  // Actions - Workflow
  clear: () => {
    useDirtyStore.getState().markClean();
    set({
      workflowId: "",
      workflowData: { name: "", description: "" },
      taskNodes: {},
      edges: [],
      selectedTaskId: "",
    });
  },

  setWorkflowId: (id: string) => set({ workflowId: id }),

  setWorkflowData: (data: Partial<WorkflowData>) => {
    markDirty();
    set((state) => ({
      workflowData: { ...state.workflowData, ...data },
    }));
  },

  // Actions - TaskNodes
  addTaskNode: (id: string, position: { x: number; y: number }) => {
    markDirty();
    set((state) => ({
      taskNodes: {
        ...state.taskNodes,
        [id]: { ...defaultTaskNode, id, position },
      },
    }));
  },

  updateTaskNode: (id: string, data: Partial<Omit<TaskNode, "id">>) => {
    markDirty();
    set((state) => {
      const existing = state.taskNodes[id];
      if (!existing) return state;
      return {
        taskNodes: {
          ...state.taskNodes,
          [id]: { ...existing, ...data },
        },
      };
    });
  },

  removeTaskNode: (id: string) => {
    markDirty();
    set((state) => {
      const { [id]: _, ...rest } = state.taskNodes;
      return {
        taskNodes: rest,
        edges: state.edges.filter((e) => e.source !== id && e.target !== id),
        selectedTaskId: state.selectedTaskId === id ? "" : state.selectedTaskId,
      };
    });
  },

  // Actions - Edges
  setEdges: (edges: Edge[]) => {
    markDirty();
    set({ edges });
  },

  // Actions - UI
  selectTask: (id: string) => set({ selectedTaskId: id }),
}));

// ============ Exports ============

export type { WorkflowState };
