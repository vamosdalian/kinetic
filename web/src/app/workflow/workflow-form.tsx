import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { WorkflowData } from "./types";

interface WorkflowFormProps {
  data: WorkflowData;
  tagOptions: string[];
  onUpdate: (data: Partial<WorkflowData>) => void;
}

const ANY_NODE_VALUE = "__any_node__";

export function Workflowform({ data, tagOptions, onUpdate }: WorkflowFormProps) {
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
      <div className="grid gap-2">
        <Label htmlFor="workflow_tag">Tag</Label>
        <Select
          value={data.tag || ANY_NODE_VALUE}
          onValueChange={(value) => {
            onUpdate({ tag: value === ANY_NODE_VALUE ? "" : value });
          }}
        >
          <SelectTrigger id="workflow_tag" className="w-full">
            <SelectValue placeholder="Select node tag" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ANY_NODE_VALUE}>Any node</SelectItem>
            {tagOptions.map((tag) => (
              <SelectItem key={tag} value={tag}>
                {tag}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}
