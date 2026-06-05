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

export interface WorkflowRunEvent {
  type: "snapshot" | "run_status" | "task_status" | "task_output" | "keepalive";
  run_id?: string;
  task_run_id?: string;
  task_id?: string;
  status?: string;
  assigned_node_id?: string;
  effective_tag?: string;
  assigned_at?: string;
  started_at?: string;
  finished_at?: string;
  output?: string;
  result?: unknown;
  exit_code?: number;
}

export interface TaskNodeRun {
  task_run_id: string;
  run_id: string;
  task_id: string;
  ref: string;
  name: string;
  description: string;
  type: string;
  config: unknown;
  tag?: string;
  effective_tag?: string;
  assigned_node_id?: string;
  position: {
    x: number;
    y: number;
  };
  nodeType: string;
  status: string;
  created_at: string;
  assigned_at?: string;
  started_at: string;
  finished_at: string;
  exit_code?: number;
  output?: string;
  result?: unknown;
  loop_id?: string;
  loop_index?: number;
  loop_value?: number;
}

export interface EdgeRun {
  edge_run_id: string;
  run_id: string;
  edge_id: string;
  source: string;
  target: string;
  sourceHandle?: string;
  targetHandle?: string;
  loop_id?: string;
  loop_index?: number;
  loop_value?: number;
}

export interface WorkflowRunDetail extends WorkflowRunListItem {
  description: string;
  tag?: string;
  taskNodes: TaskNodeRun[];
  edges: EdgeRun[];
}
