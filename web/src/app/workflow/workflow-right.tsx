import { ScrollArea } from "@/components/ui/scroll-area";
import { Taskform } from "./task-form";
import { Workflowform } from "./workflow-form";
import type { WorkflowData, TaskNode } from "./types";

interface WorkflowRightProps {
  selectedTaskId: string;
  workflowData: WorkflowData;
  tagOptions: string[];
  onUpdateWorkflowData: (data: Partial<WorkflowData>) => void;
  taskNodes: Record<string, TaskNode>;
  onUpdateTaskNode: (id: string, data: Partial<Omit<TaskNode, "id">>) => void;
}

export function WorkflowRight({
  selectedTaskId,
  workflowData,
  tagOptions,
  onUpdateWorkflowData,
  taskNodes,
  onUpdateTaskNode,
}: WorkflowRightProps) {
  return (
    <ScrollArea className="h-full">
      {selectedTaskId === "ROOT" ? (
        <Workflowform
          data={workflowData}
          tagOptions={tagOptions}
          onUpdate={onUpdateWorkflowData}
        />
      ) : (
        <Taskform
          taskId={selectedTaskId}
          node={taskNodes[selectedTaskId]}
          tagOptions={tagOptions}
          workflowTag={workflowData.tag}
          onUpdate={onUpdateTaskNode}
        />
      )}
    </ScrollArea>
  );
}
