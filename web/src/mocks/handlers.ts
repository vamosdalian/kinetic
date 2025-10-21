import { http, HttpResponse } from "msw";
import {type Workflow} from "../app/workflow/workflow";
// interface Workflow {
//   id: string;
//   name: string;
//   description: string;
//   lastRunTimestamp: string;
//   lastRunStaus: "success" | "failed" | "running" | "pedding";
//   nextRunTimestamp: string;
// }

const mockWorkflows: Workflow[] = [
  {
    id: "1",
    name: "Workflow One",
    enable: true,
    version:"1",
    last_running_status:"success",
    next_running_time:"",
    create_at:"",
  },
  {
    id: "2",
    name: "Workflow Two",
    enable: true,
    version:"1",
    last_running_status:"success",
    next_running_time:"",
    create_at:"",
  },
];

export const handlers = [
  http.get("/api/_hc", ({ request }) => {
    console.log("Health check request received:", request);
    return HttpResponse.json({ status: "ok" });
  }),
  http.get("/api/workflows", ({ request }) => {
    console.log("Fetch workflows request received:", request);
    return HttpResponse.json(mockWorkflows);
  }),
  http.get("/api/workflows/:id", ({ request, params }) => {
    console.log("Fetch workflows request received:", request);
    const workflow = mockWorkflows.find((wf) => wf.id === params.id);
    if (workflow) {
      return HttpResponse.json(workflow);
    } else {
      return HttpResponse.json(
        { error: "Workflow not found" },
        { status: 404 }
      );
    }
  }),
  http.put("/api/workflows/:id", async ({ request, params }) => {
    const updatedWorkflow = (await request.json()) as Partial<Workflow>;
    const index = mockWorkflows.findIndex((wf) => wf.id === params.id);
    if (index !== -1) {
      mockWorkflows[index] = { ...mockWorkflows[index], ...updatedWorkflow };
      console.log("Update workflow request received:", request, updatedWorkflow);
      return HttpResponse.json(mockWorkflows[index]);
    } else {
      return HttpResponse.json(
        { error: "Workflow not found" },
        { status: 404 }
      );
    }
  }),
  http.delete("/api/workflows/:id", ({ request, params }) => {
    const index = mockWorkflows.findIndex((wf) => wf.id === params.id);
    if (index !== -1) {
      mockWorkflows.splice(index, 1);
      console.log("Delete workflow request received:", request);
      return HttpResponse.json({ message: "Workflow deleted" });
    } else {
      return HttpResponse.json(
        { error: "Workflow not found" },
        { status: 404 }
      );
    }
  }),
];
