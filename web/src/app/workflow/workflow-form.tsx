import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import type { WorkflowData } from "./types";

interface WorkflowFormProps {
  data: WorkflowData;
  onUpdate: (data: Partial<WorkflowData>) => void;
}

export function Workflowform({ data, onUpdate }: WorkflowFormProps) {
  return (
    <div className="grid gap-6 m-4">
      <div className="grid gap-2">
        <h1 className="text-xl">Workflow Info</h1>
        <Separator style={{ margin: "0" }}></Separator>
      </div>
      <div className="grid gap-2">
        <Label htmlFor="workflow_name">Workflow Name</Label>
        <Input
          id="workflow_name"
          placeholder="Enter workflow name..."
          value={data.name}
          onChange={(e) => {
            onUpdate({ name: e.target.value });
          }}
        />
      </div>
      <div className="grid gap-2">
        <Label htmlFor="description">Description</Label>
        <Textarea
          id="description"
          placeholder="Describe what this workflow does..."
          value={data.description}
          onChange={(e) => {
            onUpdate({ description: e.target.value });
          }}
        />
      </div>
    </div>
  );
}
