import * as React from "react";
import { AppSidebar } from "@/components/app-sidebar";

// import { SiteHeader } from "@/components/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { Routes, Route } from "react-router-dom";

const Workflow = React.lazy(async () => {
  const mod = await import("./workflow/workflow");
  return { default: mod.Workflow };
});
const Dashboard = React.lazy(async () => {
  const mod = await import("./dashboard/dashboard");
  return { default: mod.Dashboard };
});
const Admin = React.lazy(async () => {
  const mod = await import("./admin/admin");
  return { default: mod.Admin };
});
const Docs = React.lazy(async () => {
  const mod = await import("./docs/docs");
  return { default: mod.Docs };
});
const Node = React.lazy(async () => {
  const mod = await import("./node/node");
  return { default: mod.Node };
});
const WorkflowDetail = React.lazy(async () => {
  const mod = await import("./workflow/workflow-detail");
  return { default: mod.WorkflowDetail };
});
const Record = React.lazy(async () => {
  const mod = await import("./record/record");
  return { default: mod.Record };
});
const RecordDetail = React.lazy(async () => {
  const mod = await import("./record/record-detail");
  return { default: mod.RecordDetail };
});
import { Toaster } from "@/components/ui/sonner"

function RouteLoader() {
  return (
    <div className="flex h-full min-h-[320px] items-center justify-center text-sm text-muted-foreground">
      Loading page...
    </div>
  );
}

export default function Page() {
  return (
    <SidebarProvider
      style={
        {
          "--sidebar-width": "calc(var(--spacing) * 64)",
          "--header-height": "calc(var(--spacing) * 12)",
        } as React.CSSProperties
      }
    >
      <AppSidebar />
      <SidebarInset>
        {/* <SiteHeader /> */}
        <React.Suspense fallback={<RouteLoader />}>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/workflow" element={<Workflow />} />
            <Route path="/workflow/:workflowId" element={<WorkflowDetail />} />
            <Route path="/record" element={<Record />} />
            <Route path="/record/:runId" element={<RecordDetail />} />
            <Route path="/node" element={<Node />} />
            <Route path="/admin" element={<Admin />} />
            <Route path="/docs" element={<Docs />} />
          </Routes>
        </React.Suspense>
      </SidebarInset>
      <Toaster position="top-center" richColors/>
    </SidebarProvider>
  );
}
