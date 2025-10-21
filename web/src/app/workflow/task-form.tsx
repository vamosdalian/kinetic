// import * as React from "react";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { ShellEditor } from "./shell_editor";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { useWorkflowStore, useSavedStore } from "./workflow-store";

export function Taskform() {
  const { taskId, nodes, setNodes } = useWorkflowStore();
  const { setSaved } = useSavedStore();
  const taskinfo = nodes[taskId] || {};

  return (
    <div className="grid gap-6 m-4">
      <div className="grid gap-2">
        <h1 className="text-xl">Task Info</h1>
        <Separator style={{ margin: "0" }}></Separator>
      </div>
      <div className="grid gap-2">
        <Label htmlFor="subject">TaskName</Label>
        <Input
          id="task_name"
          placeholder="I need help with..."
          value={taskinfo.name || ""}
          onChange={(e) => {
            if (taskId) {
              setNodes({ [taskId]: { name: e.target.value } });
              setSaved(false);
            }
          }}
        />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="grid gap-2">
          <Label htmlFor="area">Input</Label>
          <Select defaultValue="billing">
            <SelectTrigger id="area" className="w-full">
              <SelectValue placeholder="Select" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="team">Task1</SelectItem>
              <SelectItem value="billing">Task2</SelectItem>
              <SelectItem value="account">Account</SelectItem>
              <SelectItem value="deployments">Deployments</SelectItem>
              <SelectItem value="support">None</SelectItem>
            </SelectContent>
          </Select>
        </div>
        {/* <div className="grid gap-2">
          <Label htmlFor="security-level">Security Level</Label>
          <Select defaultValue="2">
            <SelectTrigger
              id="security-level"
              className="line-clamp-1 w-[160px] truncate"
            >
              <SelectValue placeholder="Select level" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="1">Severity 1 (Highest)</SelectItem>
              <SelectItem value="2">Severity 2</SelectItem>
              <SelectItem value="3">Severity 3</SelectItem>
              <SelectItem value="4">Severity 4 (Lowest)</SelectItem>
            </SelectContent>
          </Select>
        </div> */}
      </div>
      <div className="grid gap-2">
        <Label htmlFor="subject">Script</Label>
        <ShellEditor>
          <Button variant="outline" className="w-full ">
            Edit Script
          </Button>
        </ShellEditor>
      </div>
      <div className="grid gap-2">
        <Label htmlFor="subject">Subject</Label>
        <Input id="subject" placeholder="I need help with..." />
      </div>
      <div className="grid gap-2">
        <Label htmlFor="description">Description</Label>
        <Textarea
          id="description"
          placeholder="Please include all information relevant to your issue."
          value={taskinfo.description || ""}
          onChange={(e) => {
            if (taskId) {
              setNodes({ [taskId]: { description: e.target.value } });
              setSaved(false);
            }
          }}
        />
      </div>
    </div>
  );
}
