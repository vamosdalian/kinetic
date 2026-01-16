import { FlowWithProvider } from "./workflow-graph";
import { SiteHeader } from "@/components/site-header";

export function WorkflowDetail() {
  return (
    <>
      <div className="flex h-full flex-col">
        <SiteHeader
          breadcrumbs={[
            { label: "Workflow", href: "/workflow" },
            { label: "Editor", href: null },
          ]}
        />
        <div className="flex-1 overflow-hidden">
          <FlowWithProvider />
        </div>
      </div>
    </>
  );
}
