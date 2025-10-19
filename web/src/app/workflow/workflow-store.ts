import { create } from "zustand";

interface TaskData {
  name: string;
  description: string;
  input: string;
  script: string;
}

const defaultTaskData: TaskData = {
  name: "",
  description: "",
  input: "",
  script: "",
};

// saved state

interface SavedState {
  saved: boolean;
  setSaved: (saved: boolean) => void;
}

const useSavedStore = create<SavedState>()((set) => ({
  saved: false,
  setSaved: (saved: boolean) => set({ saved }),
}));

// workflow data

interface WorkflowData {
  name: string;
  description: string;
}

interface WorkflowState {
  workflowId: string;
  workflowData: WorkflowData;
  taskId: string;
  nodes: { [key: string]: TaskData }; // key: node ID, value: TaskData

  clear: () => void;
  setWorkflowId: (id: string) => void;
  setWorkflowData: (data: Partial<WorkflowData>) => void;
  setNodes: (data: { [key: string]: Partial<TaskData> }) => void;
  delNode: (id: string) => void;
  setTaskId: (id: string) => void;
}

const useWorkflowStore = create<WorkflowState>()((set) => ({
  workflowId: "",
  taskId: "",
  workflowData: {
    name: "",
    description: "",
  },
  nodes: {},

  clear: () =>
    set({
      workflowId: "",
      workflowData: {
        name: "",
        description: "",
      },
      nodes: {},
    }),
  setWorkflowId: (id: string) => set({ workflowId: id }),
  setWorkflowData: (data: Partial<WorkflowData>) =>
    set((state) => ({
      workflowData: { ...state.workflowData, ...data },
    })),
  setNodes: (data: { [key: string]: Partial<TaskData> }) =>
    set((state) => {
      const newNodes = { ...state.nodes };
      for (const id in data) {
        const existingNode = newNodes[id];
        const providedData = data[id];
        if (existingNode) {
          // Update existing node
          newNodes[id] = { ...existingNode, ...providedData };
        } else {
          // Create new node with defaults
          newNodes[id] = { ...defaultTaskData, ...providedData };
        }
      }
      return { nodes: newNodes };
    }),
  delNode: (id: string) =>
    set((state) => {
      const newNodes = { ...state.nodes };
      delete newNodes[id];
      return { nodes: newNodes };
    }),
  setTaskId: (id: string) => set({ taskId: id }),
}));

export { useSavedStore, useWorkflowStore, type WorkflowData };
