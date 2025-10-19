import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { useWorkflowStore, useSavedStore } from "./workflow-store";

export function Workflowform() {
  const { workflowData, setWorkflowData } = useWorkflowStore();
  const { setSaved } = useSavedStore();
  return (
    <div className="grid gap-6 m-4">
      <div className="grid gap-2">
        <h1 className="text-xl">Workflow Info</h1>
        <Separator style={{ margin: "0" }}></Separator>
      </div>
      <div className="grid gap-2">
        <Label htmlFor="subject">WorkflowName</Label>
        <Input
          id="workflow_name"
          placeholder="I need help with..."
          value={workflowData.name}
          onChange={(e) => {
            setWorkflowData({ name: e.target.value });
            setSaved(false);
          }}
        />
      </div>
      <div className="grid gap-2">
        <Label htmlFor="description">Description</Label>
        <Textarea
          id="description"
          placeholder="Please include all information relevant to your issue."
          value={workflowData.description}
          onChange={(e) => {
            setWorkflowData({ description: e.target.value });
            setSaved(false);
          }}
        />
      </div>
    </div>
  );
}
