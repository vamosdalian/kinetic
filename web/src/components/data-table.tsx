import * as React from "react"
import { IconDotsVertical } from "@tabler/icons-react"
import type { ColumnDef } from "@tanstack/react-table"
import { LoaderCircle } from "lucide-react"

import type {
  DashboardNodeActivityRow,
  DashboardScheduledRunRow,
  DashboardTables,
  DashboardWorkflowRow,
} from "@/lib/dashboard"
import {
  formatDashboardDateTime,
  formatDashboardDuration,
  formatNodeRole,
  formatPercentage,
} from "@/lib/dashboard"
import { getNodeStatusBadgeClassName } from "@/lib/node-status"
import { getStatusBadgeClassName } from "@/app/record/status"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { CommonTable } from "@/components/common-table"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { cn } from "@/lib/utils"

function WorkflowRowActions({
  runID,
  onView,
  onRerun,
  rerunning,
}: {
  runID: string
  onView: (runID: string) => void
  onRerun: (runID: string) => void
  rerunning: boolean
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          className="data-[state=open]:bg-muted text-muted-foreground flex size-8"
          size="icon"
        >
          <IconDotsVertical />
          <span className="sr-only">Open menu</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-36">
        <DropdownMenuItem onSelect={() => onView(runID)}>View details</DropdownMenuItem>
        <DropdownMenuItem disabled={rerunning} onSelect={() => onRerun(runID)}>
          {rerunning ? (
            <>
              <LoaderCircle className="size-4 animate-spin" />
              Rerunning...
            </>
          ) : (
            "Rerun"
          )}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function NodeRowActions({ onOpenNodes }: { onOpenNodes: () => void }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          className="data-[state=open]:bg-muted text-muted-foreground flex size-8"
          size="icon"
        >
          <IconDotsVertical />
          <span className="sr-only">Open menu</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-36">
        <DropdownMenuItem onSelect={onOpenNodes}>Open nodes</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

const workflowColumns = ({
  onViewWorkflowRun,
  onRerunWorkflowRun,
  rerunningRunID,
}: {
  onViewWorkflowRun: (runID: string) => void
  onRerunWorkflowRun: (runID: string) => void
  rerunningRunID: string | null
}): ColumnDef<DashboardWorkflowRow>[] => [
  {
    accessorKey: "workflow_name",
    header: "Workflow",
    cell: ({ row }) => (
      <div className="min-w-[180px]">
        <div className="font-medium">{row.original.workflow_name}</div>
        <div className="font-mono text-xs text-muted-foreground">
          {row.original.run_id}
        </div>
      </div>
    ),
  },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) => (
      <Badge
        variant="outline"
        className={`capitalize ${getStatusBadgeClassName(row.original.status)}`}
      >
        {row.original.status}
      </Badge>
    ),
  },
  {
    accessorKey: "started_at",
    header: "Started At",
    cell: ({ row }) => formatDashboardDateTime(row.original.started_at),
  },
  {
    accessorKey: "duration_seconds",
    header: "Duration",
    cell: ({ row }) => formatDashboardDuration(row.original.duration_seconds),
  },
  {
    accessorKey: "node_label",
    header: "Node",
  },
  {
    id: "actions",
    header: "",
    cell: ({ row }) => (
      <WorkflowRowActions
        runID={row.original.run_id}
        onView={onViewWorkflowRun}
        onRerun={onRerunWorkflowRun}
        rerunning={rerunningRunID === row.original.run_id}
      />
    ),
    enableHiding: false,
  },
]

const scheduledRunColumns = ({
  onOpenNodes,
}: {
  onOpenNodes: () => void
}): ColumnDef<DashboardScheduledRunRow>[] => [
  {
    accessorKey: "workflow_name",
    header: "Workflow",
  },
  {
    accessorKey: "mode",
    header: "Mode",
    cell: ({ row }) => (
      <Badge variant="outline" className="text-muted-foreground px-1.5">
        {row.original.mode}
      </Badge>
    ),
  },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) => (
      <Badge
        variant="outline"
        className={`capitalize ${getStatusBadgeClassName(row.original.status)}`}
      >
        {row.original.status}
      </Badge>
    ),
  },
  {
    accessorKey: "next_run_at",
    header: "Next Run",
    cell: ({ row }) => formatDashboardDateTime(row.original.next_run_at),
  },
  {
    accessorKey: "last_run_at",
    header: "Last Run",
    cell: ({ row }) => formatDashboardDateTime(row.original.last_run_at),
  },
  {
    accessorKey: "target_node",
    header: "Target Node",
  },
  {
    id: "actions",
    header: "",
    cell: () => <NodeRowActions onOpenNodes={onOpenNodes} />,
    enableHiding: false,
  },
]

const nodeActivityColumns = ({
  onOpenNodes,
}: {
  onOpenNodes: () => void
}): ColumnDef<DashboardNodeActivityRow>[] => [
  {
    accessorKey: "node_name",
    header: "Node",
    cell: ({ row }) => (
      <div className="min-w-[180px]">
        <div className="font-medium">{row.original.node_name}</div>
        <div className="font-mono text-xs text-muted-foreground">
          {row.original.node_id}
        </div>
      </div>
    ),
  },
  {
    accessorKey: "role",
    header: "Role",
    cell: ({ row }) => (
      <Badge variant="outline" className="text-muted-foreground px-1.5">
        {formatNodeRole(row.original.role)}
      </Badge>
    ),
  },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) => (
      <Badge
        variant="outline"
        className={cn("capitalize", getNodeStatusBadgeClassName(row.original.status))}
      >
        {row.original.status}
      </Badge>
    ),
  },
  {
    accessorKey: "today_runs",
    header: "Today's Runs",
  },
  {
    accessorKey: "success_rate",
    header: "Success Rate",
    cell: ({ row }) => formatPercentage(row.original.success_rate, 1),
  },
  {
    accessorKey: "running_count",
    header: "Running Count",
  },
  {
    accessorKey: "last_heartbeat_at",
    header: "Last Heartbeat",
    cell: ({ row }) => formatDashboardDateTime(row.original.last_heartbeat_at),
  },
  {
    id: "actions",
    header: "",
    cell: () => <NodeRowActions onOpenNodes={onOpenNodes} />,
    enableHiding: false,
  },
]

const tabItems = [
  { value: "today-workflows", label: "Today's Workflows" },
  { value: "scheduled-runs", label: "Scheduled Runs" },
  { value: "failed-workflows", label: "Failed Workflows" },
  { value: "node-activity", label: "Node Activity" },
] as const

export function DataTable({
  tables,
  rerunningRunID,
  onViewWorkflowRun,
  onRerunWorkflowRun,
  onOpenNodes,
}: {
  tables: DashboardTables
  rerunningRunID: string | null
  onViewWorkflowRun: (runID: string) => void
  onRerunWorkflowRun: (runID: string) => void
  onOpenNodes: () => void
}) {
  const [activeTab, setActiveTab] = React.useState("today-workflows")

  const renderViewSelector = () => (
    <>
      <Select value={activeTab} onValueChange={setActiveTab}>
        <SelectTrigger className="flex w-fit @4xl/main:hidden" size="sm">
          <SelectValue placeholder="Select a view" />
        </SelectTrigger>
        <SelectContent>
          {tabItems.map((tab) => (
            <SelectItem key={tab.value} value={tab.value}>
              {tab.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <TabsList className="hidden @4xl/main:flex">
        <TabsTrigger value="today-workflows">
          Today's Workflows
          <Badge variant="secondary">{tables.today_workflows.count}</Badge>
        </TabsTrigger>
        <TabsTrigger value="scheduled-runs">
          Scheduled Runs
          <Badge variant="secondary">{tables.scheduled_runs.count}</Badge>
        </TabsTrigger>
        <TabsTrigger value="failed-workflows">
          Failed Workflows
          <Badge variant="secondary">{tables.failed_workflows.count}</Badge>
        </TabsTrigger>
        <TabsTrigger value="node-activity">
          Node Activity
          <Badge variant="secondary">{tables.node_activity.count}</Badge>
        </TabsTrigger>
      </TabsList>
    </>
  )

  return (
    <Tabs
      value={activeTab}
      onValueChange={setActiveTab}
      className="w-full flex-col justify-start"
    >
      <TabsContent value="today-workflows" className="flex flex-col px-4 lg:px-6">
        <CommonTable
          columns={workflowColumns({
            onViewWorkflowRun,
            onRerunWorkflowRun,
            rerunningRunID,
          })}
          data={tables.today_workflows.items}
          renderToolbarActions={renderViewSelector}
        />
      </TabsContent>

      <TabsContent value="scheduled-runs" className="flex flex-col px-4 lg:px-6">
        <CommonTable
          columns={scheduledRunColumns({ onOpenNodes })}
          data={tables.scheduled_runs.items}
          renderToolbarActions={renderViewSelector}
        />
      </TabsContent>

      <TabsContent value="failed-workflows" className="flex flex-col px-4 lg:px-6">
        <CommonTable
          columns={workflowColumns({
            onViewWorkflowRun,
            onRerunWorkflowRun,
            rerunningRunID,
          })}
          data={tables.failed_workflows.items}
          renderToolbarActions={renderViewSelector}
        />
      </TabsContent>

      <TabsContent value="node-activity" className="flex flex-col px-4 lg:px-6">
        <CommonTable
          columns={nodeActivityColumns({ onOpenNodes })}
          data={tables.node_activity.items}
          renderToolbarActions={renderViewSelector}
        />
      </TabsContent>
    </Tabs>
  )
}
