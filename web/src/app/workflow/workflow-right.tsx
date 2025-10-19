// import * as React from "react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Taskform } from "./task-form";
import { Workflowform } from "./workflow-form";
import { useWorkflowStore } from "./workflow-store";

export function WorkflowRight() {
  const taskId = useWorkflowStore((state) => state.taskId);
  return (
    <ScrollArea className="h-[calc(100vh-var(--header-height))]">
      {taskId.length > 0 ? (
        <Taskform></Taskform>
      ) : (
        <Workflowform></Workflowform>
      )}
    </ScrollArea>
  );
}
