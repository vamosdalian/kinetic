import { FlowWithProvider } from "./workflow-graph";
import { WorkflowRight } from "./workflow-right";
import { PanelGroup, Panel, PanelResizeHandle } from "react-resizable-panels";
import { UnsavedChangesGuard } from "./unsaved-changes-guard";

export function WorkflowDetail() {
  return (
    <>
      <UnsavedChangesGuard />
      <PanelGroup direction="horizontal" className="h-full min-h-0">
        <Panel>
          <FlowWithProvider />
        </Panel>
        <PanelResizeHandle className="w-[2px] bg-gray-200" />
        <Panel defaultSize={20}>
          <WorkflowRight />
        </Panel>
      </PanelGroup>
    </>
  );
}
