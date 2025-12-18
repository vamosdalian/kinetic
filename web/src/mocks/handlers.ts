import { http, HttpResponse } from "msw";

// ============ Types ============

interface WorkflowListItem {
  id: string;
  name: string;
  enable: boolean;
  version: string;
  last_running_status: "pending" | "running" | "success" | "failed";
  next_running_time: string;
  create_at: string;
}

interface Position {
  x: number;
  y: number;
}

interface TaskNode {
  id: string;
  name: string;
  description: string;
  type: "shell" | "http" | "python" | "condition";
  config: Record<string, unknown>;
  position: Position;
  nodeType: string;
}

interface Edge {
  id: string;
  source: string;
  target: string;
  sourceHandle?: string;
  targetHandle?: string;
}

interface WorkflowDetail {
  id: string;
  name: string;
  description: string;
  taskNodes: TaskNode[];
  edges: Edge[];
  version: string;
  enable: boolean;
  create_at: string;
  update_at: string;
}

// ============ Mock Data ============

const mockWorkflowDetails: Map<string, WorkflowDetail> = new Map([
  [
    "workflow-001",
    {
      id: "workflow-001",
      name: "Data Processing Pipeline",
      description: "A sample workflow that processes data",
      taskNodes: [
        {
          id: "task-001",
          name: "Fetch Data",
          description: "Fetch data from API",
          type: "http",
          config: {
            url: "https://api.example.com/data",
            method: "GET",
          },
          position: { x: 250, y: 50 },
          nodeType: "baseNodeFull",
        },
        {
          id: "task-002",
          name: "Process Data",
          description: "Process the fetched data",
          type: "shell",
          config: {
            script: "#!/bin/bash\necho 'Processing data...'",
          },
          position: { x: 250, y: 200 },
          nodeType: "baseNodeFull",
        },
      ],
      edges: [
        {
          id: "edge-001",
          source: "task-001",
          target: "task-002",
        },
      ],
      version: "1.0.0",
      enable: true,
      create_at: "2025-12-18T08:00:00Z",
      update_at: "2025-12-18T10:30:00Z",
    },
  ],
  [
    "workflow-002",
    {
      id: "workflow-002",
      name: "Daily Report Generator",
      description: "Generates daily reports",
      taskNodes: [
        {
          id: "task-003",
          name: "Generate Report",
          description: "Run Python script to generate report",
          type: "python",
          config: {
            script: "print('Generating report...')",
            requirements: ["pandas>=1.5.0"],
          },
          position: { x: 250, y: 100 },
          nodeType: "baseNodeFull",
        },
      ],
      edges: [],
      version: "1.0.0",
    enable: true,
      create_at: "2025-12-17T09:00:00Z",
      update_at: "2025-12-17T09:00:00Z",
    },
  ],
]);

// Helper: Convert detail to list item
function toListItem(detail: WorkflowDetail): WorkflowListItem {
  return {
    id: detail.id,
    name: detail.name,
    enable: detail.enable,
    version: detail.version,
    last_running_status: "success",
    next_running_time: "",
    create_at: detail.create_at,
  };
}

// ============ Handlers ============

export const handlers = [
  // Health check
  http.get("/api/_hc", () => {
    return HttpResponse.json({ status: "ok" });
  }),

  // GET /api/workflows - List all workflows
  http.get("/api/workflows", () => {
    const workflows = Array.from(mockWorkflowDetails.values()).map(toListItem);
    console.log("[MSW] GET /api/workflows", workflows);
    return HttpResponse.json(workflows);
  }),

  // GET /api/workflows/:id - Get workflow details
  http.get("/api/workflows/:workflowId", ({ params }) => {
    const { workflowId } = params;
    const workflow = mockWorkflowDetails.get(workflowId as string);

    console.log("[MSW] GET /api/workflows/:id", workflowId);

    if (workflow) {
      return HttpResponse.json(workflow);
    } else {
      return HttpResponse.json(
        { code: "WORKFLOW_NOT_FOUND", message: `Workflow '${workflowId}' not found` },
        { status: 404 }
      );
    }
  }),

  // PUT /api/workflows/:id - Create or update workflow (upsert)
  http.put("/api/workflows/:workflowId", async ({ params, request }) => {
    const { workflowId } = params;
    const body = (await request.json()) as Partial<WorkflowDetail>;

    console.log("[MSW] PUT /api/workflows/:id", workflowId, body);

    const existing = mockWorkflowDetails.get(workflowId as string);
    const now = new Date().toISOString();

    const workflow: WorkflowDetail = {
      id: workflowId as string,
      name: body.name || existing?.name || "Untitled",
      description: body.description || existing?.description || "",
      taskNodes: body.taskNodes || existing?.taskNodes || [],
      edges: body.edges || existing?.edges || [],
      version: existing?.version || "1.0.0",
      enable: body.enable ?? existing?.enable ?? true,
      create_at: existing?.create_at || now,
      update_at: now,
    };

    mockWorkflowDetails.set(workflowId as string, workflow);

    return HttpResponse.json(workflow);
  }),

  // DELETE /api/workflows/:id - Delete workflow
  http.delete("/api/workflows/:workflowId", ({ params }) => {
    const { workflowId } = params;

    console.log("[MSW] DELETE /api/workflows/:id", workflowId);

    if (mockWorkflowDetails.has(workflowId as string)) {
      mockWorkflowDetails.delete(workflowId as string);
      return new HttpResponse(null, { status: 204 });
    } else {
      return HttpResponse.json(
        { code: "WORKFLOW_NOT_FOUND", message: `Workflow '${workflowId}' not found` },
        { status: 404 }
      );
    }
  }),

  // POST /api/workflows/:id/run - Run workflow
  http.post("/api/workflows/:workflowId/run", async ({ params, request }) => {
    const { workflowId } = params;

    console.log("[MSW] POST /api/workflows/:id/run", workflowId);

    if (!mockWorkflowDetails.has(workflowId as string)) {
      return HttpResponse.json(
        { code: "WORKFLOW_NOT_FOUND", message: `Workflow '${workflowId}' not found` },
        { status: 404 }
      );
    }

    // Parse optional inputs
    let inputs = {};
    try {
      const body = await request.json();
      inputs = (body as { inputs?: Record<string, unknown> }).inputs || {};
    } catch {
      // No body or invalid JSON, that's fine
    }

    console.log("[MSW] Running workflow with inputs:", inputs);

    // Return mock execution response
    const executionId = `exec-${Date.now()}`;
    return HttpResponse.json(
      {
        execution_id: executionId,
        status: "pending",
        started_at: new Date().toISOString(),
      },
      { status: 202 }
    );
  }),
];
