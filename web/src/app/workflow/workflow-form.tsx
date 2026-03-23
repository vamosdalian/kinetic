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
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import { Button } from "@/components/ui/button";
import { CircleQuestionMark } from "lucide-react";
import { KeyValueEditor } from "./key-value-editor";
import type { WorkflowData } from "./types";

interface WorkflowFormProps {
  data: WorkflowData;
  tagOptions: string[];
  onUpdate: (data: Partial<WorkflowData>) => void;
}

function HelpHint({ content }: { content: React.ReactNode }) {
  return (
    <HoverCard openDelay={150} closeDelay={100}>
      <HoverCardTrigger asChild>
        <button
          type="button"
          className="inline-flex h-4 w-4 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
          aria-label="Show help"
        >
          <CircleQuestionMark className="h-4 w-4" />
        </button>
      </HoverCardTrigger>
      <HoverCardContent align="start" className="w-72 text-sm leading-5">
        {content}
      </HoverCardContent>
    </HoverCard>
  );
}

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
          value={data.tag || "node-default"}
          onValueChange={(value) => {
            onUpdate({ tag: value });
          }}
        >
          <SelectTrigger id="workflow_tag" className="w-full">
            <SelectValue placeholder="Select node tag" />
          </SelectTrigger>
          <SelectContent>
            {tagOptions.map((tag) => (
              <SelectItem key={tag} value={tag}>
                {tag}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="grid gap-2">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <Label>Environment Variables</Label>
            <HelpHint content="These values are stored under workflow config and inherited by tasks unless overridden. Keys are unique within this map." />
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => {
              const env = { ...(data.config.env ?? {}) };
              let index = Object.keys(env).length + 1;
              let candidate = `env-${index}`;

              while (candidate in env) {
                index += 1;
                candidate = `env-${index}`;
              }

              env[candidate] = "";
              onUpdate({
                config: {
                  ...data.config,
                  env,
                },
              });
            }}
          >
            Add
          </Button>
        </div>
        <KeyValueEditor
          values={data.config.env ?? {}}
          onChange={(env) => {
            onUpdate({
              config: {
                ...data.config,
                env,
              },
            });
          }}
          emptyText="No workflow environment variables configured."
          keyPlaceholder="Variable name"
          valuePlaceholder="Variable value"
          keyPrefix="env"
          showAddButton={false}
        />
      </div>
    </div>
  );
}
