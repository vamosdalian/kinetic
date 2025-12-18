import { ScrollArea } from "@/components/ui/scroll-area";
import { Taskform } from "./task-form";
import { Workflowform } from "./workflow-form";
import { useSelection } from "./selection-context";

export function WorkflowRight() {
  const { selectedTaskId } = useSelection();
  return (
    <ScrollArea className="h-[calc(100vh-var(--header-height))]">
      {selectedTaskId.length > 0 ? (
        <Taskform />
      ) : (
        <Workflowform />
      )}
    </ScrollArea>
  );
}
