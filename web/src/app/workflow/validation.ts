import { type Edge } from "@xyflow/react";
import {
  type ConditionConfig,
  type ForConfig,
  type HttpConfig,
  type ShellConfig,
  type TaskConfig,
  type TaskNode,
} from "./types";

export interface WorkflowValidationResult {
  valid: boolean;
  errors: string[];
}

function getTaskLabel(task: TaskNode) {
  return task.name.trim() || task.id;
}

function validatePolicy(config: TaskConfig, task: TaskNode, errors: string[]) {
  const policyFields = [
    ["timeout_seconds", config.timeout_seconds],
    ["retry_count", config.retry_count],
    ["retry_backoff_seconds", config.retry_backoff_seconds],
  ] as const;

  for (const [field, value] of policyFields) {
    if (value === undefined || value === null) {
      continue;
    }
    if (!Number.isFinite(Number(value)) || Number(value) < 0) {
      errors.push(`${getTaskLabel(task)} has an invalid ${field} value.`);
    }
  }
}

function isValidEnvName(value: string) {
  return /^[A-Za-z_][A-Za-z0-9_]*$/.test(value) && !value.startsWith("KINETIC_");
}

export function validateWorkflowDefinition(
  taskNodes: Record<string, TaskNode> | TaskNode[],
  edges: Edge[]
): WorkflowValidationResult {
  const tasks = Array.isArray(taskNodes) ? taskNodes : Object.values(taskNodes);
  const errors: string[] = [];
  const taskMap = new Map<string, TaskNode>();
  const inbound = new Map<string, Edge[]>();
  const outbound = new Map<string, Edge[]>();

  for (const task of tasks) {
    if (!task.id) {
      errors.push("Each task must have an ID.");
      continue;
    }

    taskMap.set(task.id, task);
    inbound.set(task.id, []);
    outbound.set(task.id, []);

    const config = (task.config ?? {}) as TaskConfig;
    validatePolicy(config, task, errors);

    switch (task.type) {
      case "shell":
        if (!((config as ShellConfig).script || "").trim()) {
          errors.push(`${getTaskLabel(task)} requires a shell script.`);
        }
        break;
      case "http":
        if (!((config as HttpConfig).url || "").trim()) {
          errors.push(`${getTaskLabel(task)} requires a request URL.`);
        }
        break;
      case "condition":
        if (!((config as ConditionConfig).expression || "").trim()) {
          errors.push(`${getTaskLabel(task)} requires a condition expression.`);
        }
        break;
      case "for": {
        const forConfig = config as ForConfig;
        if (!Number.isFinite(Number(forConfig.start)) || !Number.isFinite(Number(forConfig.end))) {
          errors.push(`${getTaskLabel(task)} requires numeric start and end values.`);
        }
        const variable = (forConfig.var || "").trim();
        if (variable && !isValidEnvName(variable)) {
          errors.push(`${getTaskLabel(task)} has an invalid loop variable name.`);
        }
        break;
      }
      default:
        errors.push(`${getTaskLabel(task)} has an unsupported task type.`);
        break;
    }
  }

  for (const edge of edges) {
    if (!edge.source || !edge.target) {
      errors.push("Every edge must have both a source and a target.");
      continue;
    }
    if (!taskMap.has(edge.source) || !taskMap.has(edge.target)) {
      errors.push(`Edge ${edge.id || `${edge.source}->${edge.target}`} references a missing task.`);
      continue;
    }

    inbound.get(edge.target)?.push(edge);
    outbound.get(edge.source)?.push(edge);
  }

  for (const task of tasks) {
    if (task.type === "condition") {
      const inEdges = inbound.get(task.id) ?? [];
      const outEdges = outbound.get(task.id) ?? [];
      const handles = new Set(outEdges.map((edge) => edge.sourceHandle).filter(Boolean));

      if (inEdges.length !== 1) {
        errors.push(`${getTaskLabel(task)} must have exactly one inbound edge.`);
      }
      if (outEdges.length !== 2) {
        errors.push(`${getTaskLabel(task)} must have exactly two outbound edges.`);
      }
      if (!handles.has("true") || !handles.has("false") || handles.size !== 2) {
        errors.push(`${getTaskLabel(task)} must connect both true and false branches.`);
      }
      continue;
    }

    if (task.type !== "for") {
      continue;
    }

    const outEdges = outbound.get(task.id) ?? [];
    const bodyEdges = outEdges.filter((edge) => edge.sourceHandle === "body");
    const doneEdges = outEdges.filter((edge) => edge.sourceHandle === "done");
    const unsupportedEdges = outEdges.filter(
      (edge) => edge.sourceHandle !== "body" && edge.sourceHandle !== "done"
    );

    if (bodyEdges.length !== 1) {
      errors.push(`${getTaskLabel(task)} must connect exactly one body branch.`);
    }
    if (doneEdges.length > 1) {
      errors.push(`${getTaskLabel(task)} can connect at most one done branch.`);
    }
    if (unsupportedEdges.length > 0) {
      errors.push(`${getTaskLabel(task)} only supports body and done branches.`);
    }
  }

  const indegree = new Map<string, number>();
  const adjacency = new Map<string, string[]>();

  for (const task of tasks) {
    indegree.set(task.id, 0);
    adjacency.set(task.id, []);
  }

  for (const edge of edges) {
    if (!taskMap.has(edge.source) || !taskMap.has(edge.target)) {
      continue;
    }
    indegree.set(edge.target, (indegree.get(edge.target) ?? 0) + 1);
    adjacency.get(edge.source)?.push(edge.target);
  }

  const queue = tasks
    .map((task) => task.id)
    .filter((id) => (indegree.get(id) ?? 0) === 0);
  let visited = 0;

  while (queue.length > 0) {
    const taskID = queue.shift();
    if (!taskID) {
      continue;
    }
    visited += 1;
    for (const next of adjacency.get(taskID) ?? []) {
      const nextDegree = (indegree.get(next) ?? 0) - 1;
      indegree.set(next, nextDegree);
      if (nextDegree === 0) {
        queue.push(next);
      }
    }
  }

  if (visited !== tasks.length) {
    errors.push("Workflow graph contains a cycle.");
  }

  return {
    valid: errors.length === 0,
    errors,
  };
}
