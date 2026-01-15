export interface WorkflowRunListItem {
  run_id: string;
  workflow_id: string;
  name: string;
  version: string;
  status: string;
  create_at: string;
  started_at: string;
  finished_at: string;
}

export interface TaskNodeRun {
  run_id: string;
  task_id: string;
  name: string;
  description: string;
  type: string;
  config: unknown;
  position: {
    x: number;
    y: number;
  };
  nodeType: string;
  status: string;
  created_at: string;
  started_at: string;
  finished_at: string;
  exit_code: number;
  output?: string;
}

export interface EdgeRun {
  run_id: string;
  edge_id: string;
  source: string;
  target: string;
  sourceHandle?: string;
  targetHandle?: string;
}

export interface WorkflowRunDetail extends WorkflowRunListItem {
  description: string;
  taskNodes: TaskNodeRun[];
  edges: EdgeRun[];
}
