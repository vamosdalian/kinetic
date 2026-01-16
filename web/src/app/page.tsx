import { AppSidebar } from "@/components/app-sidebar";

// import { SiteHeader } from "@/components/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { Routes, Route } from "react-router-dom";

import { Workflow } from "./workflow/workflow";
import { Dashboard } from "./dashboard/dashboard";
import { Admin } from "./admin/admin";
import { Node } from "./node/node";
import { WorkflowDetail } from "./workflow/workflow-detail";
import { Record } from "./record/record";
import { RecordDetail } from "./record/record-detail";
import { Toaster } from "@/components/ui/sonner"

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
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/workflow" element={<Workflow />} />
          <Route path="/workflow/:workflowId" element={<WorkflowDetail />} />
          <Route path="/record" element={<Record />} />
          <Route path="/record/:runId" element={<RecordDetail />} />
          <Route path="/node" element={<Node />} />
          <Route path="/admin" element={<Admin />} />
        </Routes>
      </SidebarInset>
      <Toaster position="top-center" richColors/>
    </SidebarProvider>
  );
}
