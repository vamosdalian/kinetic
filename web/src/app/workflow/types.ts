import { type Edge } from "@xyflow/react";

// ============ Task Type Configs ============

export type TaskType = "shell" | "http" | "python" | "condition";

export interface TaskPolicy {
  timeout_seconds?: number;
  retry_count?: number;
  retry_backoff_seconds?: number;
}

export interface ShellConfig extends TaskPolicy {
  script: string;
}

export interface HttpConfig extends TaskPolicy {
  url: string;
  method: "GET" | "POST" | "PUT" | "DELETE";
  headers?: Record<string, string>;
  body?: string;
}

export interface PythonConfig extends TaskPolicy {
  script: string;
  requirements?: string[];
}

export interface ConditionConfig extends TaskPolicy {
  expression: string;
}

export type TaskConfig = ShellConfig | HttpConfig | PythonConfig | ConditionConfig;

export function createTaskConfig(type: TaskType): TaskConfig {
  switch (type) {
    case "shell":
      return { script: "" };
    case "http":
      return { url: "", method: "GET", headers: {} };
    case "python":
      return { script: "", requirements: [] };
    case "condition":
      return { expression: "" };
  }
}

// ============ TaskNode (merged Node + Task) ============

export interface TaskNode {
  id: string;

  // 业务数据
  name: string;
  description: string;
  type: TaskType;
  config: TaskConfig;
  tag: string;

  // 视觉数据
  position: { x: number; y: number };
  nodeType: string; // ReactFlow node type, e.g., "baseNodeFull"
}

export const defaultTaskNode: Omit<TaskNode, "id" | "position"> = {
  name: "New Task",
  description: "",
  type: "shell",
  config: createTaskConfig("shell"),
  tag: "",
  nodeType: "baseNodeFull",
};

// ============ Workflow Data ============

export interface WorkflowData {
  name: string;
  description: string;
  tag: string;
}

// ============ Workflow Detail (API response) ============

export interface WorkflowDetail {
  id: string;
  name: string;
  description: string;
  tag?: string;
  taskNodes: TaskNode[];
  edges: Edge[];
  version?: string;
  enable?: boolean;
  create_at?: string;
  update_at?: string;
}

export interface WorkflowListItem {
  id: string;
  name: string;
  enable: boolean;
  version: string;
  create_at: string;
  update_at: string;
}
