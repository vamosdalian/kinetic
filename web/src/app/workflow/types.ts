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
export type TaskConfig = any;

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

export const defaultTaskNode: Omit<TaskNode, "id" | "position"> = {
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

// ============ Workflow Detail (API response) ============

import { type Edge } from "@xyflow/react";

export interface WorkflowDetail {
  id: string;
  name: string;
  description: string;
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
