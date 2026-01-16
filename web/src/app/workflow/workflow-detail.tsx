import { FlowWithProvider } from "./workflow-graph";
import { WorkflowRight } from "./workflow-right";
import { UnsavedChangesGuard } from "./unsaved-changes-guard";
import { SelectionProvider, useSelection } from "./selection-context";
import { cn } from "@/lib/utils";
import { Card } from "@/components/ui/card";
import { SiteHeader } from "@/components/site-header";

function WorkflowLayout() {
  const { selectedTaskId } = useSelection();
  const isOpen = !!selectedTaskId;

  return (
    <div className="relative h-full w-full overflow-hidden">
      {/* Canvas Layer */}
      <div className="h-full w-full">
        <FlowWithProvider />
      </div>

      {/* Drawer Layer */}
      <Card
        className={cn(
          "absolute top-2 right-2 bottom-2 w-[500px]",
          "shadow-2xl z-20 p-0 border-border",
          "transition-transform duration-300 ease-in-out",
          isOpen ? "translate-x-0" : "translate-x-[calc(100%+1rem)]"
        )}
      >
        <WorkflowRight />
      </Card>
    </div>
  );
}

export function WorkflowDetail() {
  return (
    <SelectionProvider>
      <UnsavedChangesGuard />
      <div className="flex h-full flex-col">
        <SiteHeader
          breadcrumbs={[
            { label: "Workflow", href: "/workflow" },
            { label: "Editor", href: null },
          ]}
        />
        <div className="flex-1 overflow-hidden">
          <WorkflowLayout />
        </div>
      </div>
    </SelectionProvider>
  );
}
