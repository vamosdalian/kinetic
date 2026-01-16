import { ScrollArea } from "@/components/ui/scroll-area";
import { Taskform } from "./task-form";
import { Workflowform } from "./workflow-form";
import { useSelection } from "./selection-context";

export function WorkflowRight() {
  const { selectedTaskId } = useSelection();
  return (
    <ScrollArea className="h-full">
      {selectedTaskId === "ROOT" ? (
        <Workflowform />
      ) : (
        <Taskform />
      )}
    </ScrollArea>
  );
}
