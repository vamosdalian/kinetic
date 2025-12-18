import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { useWorkflowStore } from "./workflow-store";

export function Workflowform() {
  const { workflowData, setWorkflowData } = useWorkflowStore();

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
          value={workflowData.name}
          onChange={(e) => {
            setWorkflowData({ name: e.target.value });
          }}
        />
      </div>
      <div className="grid gap-2">
        <Label htmlFor="description">Description</Label>
        <Textarea
          id="description"
          placeholder="Describe what this workflow does..."
          value={workflowData.description}
          onChange={(e) => {
            setWorkflowData({ description: e.target.value });
          }}
        />
      </div>
    </div>
  );
}
