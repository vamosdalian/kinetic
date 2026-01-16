import { ScrollArea } from "@/components/ui/scroll-area";
import { Taskform } from "./task-form";
import { Workflowform } from "./workflow-form";

export function WorkflowRight({ selectedTaskId }: { selectedTaskId: string }) {
  return (
    <ScrollArea className="h-full">
      {selectedTaskId === "ROOT" ? (
        <Workflowform />
      ) : (
        <Taskform taskId={selectedTaskId} />
      )}
    </ScrollArea>
  );
}
