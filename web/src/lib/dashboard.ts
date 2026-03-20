import { apiClient } from "@/lib/api"

export type DashboardRange = "7d" | "30d" | "90d"

export interface DashboardResponse {
  summary: {
    workflow_runs: number
    total_workflows: number
    total_nodes: number
    success_rate: number
  }
  chart: {
    range: DashboardRange
    timezone: string
    points: DashboardChartPoint[]
  }
  tables: {
    today_workflows: DashboardWorkflowTable
    scheduled_runs: DashboardScheduledRunTable
    failed_workflows: DashboardWorkflowTable
    node_activity: DashboardNodeActivityTable
  }
}

export interface DashboardTables {
  today_workflows: DashboardWorkflowTable
  scheduled_runs: DashboardScheduledRunTable
  failed_workflows: DashboardWorkflowTable
  node_activity: DashboardNodeActivityTable
}

export interface DashboardChartPoint {
  date: string
  success: number
  failure: number
}

export interface DashboardWorkflowTable {
  count: number
  items: DashboardWorkflowRow[]
}

export interface DashboardWorkflowRow {
  run_id: string
  workflow_id: string
  workflow_name: string
  status: string
  started_at: string
  duration_seconds: number | null
  node_label: string
}

export interface DashboardScheduledRunTable {
  count: number
  items: DashboardScheduledRunRow[]
}

export interface DashboardScheduledRunRow {
  workflow_name: string
  mode: string
  status: string
  next_run_at: string
  last_run_at: string
  target_node: string
}

export interface DashboardNodeActivityTable {
  count: number
  items: DashboardNodeActivityRow[]
}

export interface DashboardNodeActivityRow {
  node_id: string
  node_name: string
  role: string
  status: string
  today_runs: number
  success_rate: number
  running_count: number
  last_heartbeat_at: string
}

export interface DashboardCardMetric {
  label: string
  value: string
  badge: string
  summary: string
  detail: string
}

const numberFormatter = new Intl.NumberFormat("en-US")

export function formatDashboardRangeLabel(range: DashboardRange) {
  switch (range) {
    case "7d":
      return "Last 7 days"
    case "90d":
      return "Last 90 days"
    case "30d":
    default:
      return "Last 30 days"
  }
}

export function buildDashboardPath(range: DashboardRange, timezone: string) {
  const params = new URLSearchParams({
    range,
    tz: timezone,
  })

  return `/api/dashboard?${params.toString()}`
}

export async function fetchDashboard(range: DashboardRange, timezone: string) {
  return apiClient<DashboardResponse>(buildDashboardPath(range, timezone))
}

export function getDashboardTimezone() {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC"
  } catch {
    return "UTC"
  }
}

export function createEmptyDashboard(range: DashboardRange, timezone: string): DashboardResponse {
  return {
    summary: {
      workflow_runs: 0,
      total_workflows: 0,
      total_nodes: 0,
      success_rate: 0,
    },
    chart: {
      range,
      timezone,
      points: [],
    },
    tables: {
      today_workflows: {
        count: 0,
        items: [],
      },
      scheduled_runs: {
        count: 0,
        items: [],
      },
      failed_workflows: {
        count: 0,
        items: [],
      },
      node_activity: {
        count: 0,
        items: [],
      },
    },
  }
}

export function buildDashboardCardMetrics(
  summary: DashboardResponse["summary"],
  range: DashboardRange
): DashboardCardMetric[] {
  const rangeLabel = formatDashboardRangeLabel(range)

  return [
    {
      label: "Workflow Runs",
      value: numberFormatter.format(summary.workflow_runs),
      badge: rangeLabel,
      summary: "Workflow executions in range",
      detail: "Selected dashboard time range",
    },
    {
      label: "Total Workflows",
      value: numberFormatter.format(summary.total_workflows),
      badge: "Published",
      summary: "Current workflow inventory",
      detail: "Includes all saved workflows",
    },
    {
      label: "Total Nodes",
      value: numberFormatter.format(summary.total_nodes),
      badge: "Cluster",
      summary: "Available execution nodes",
      detail: "Includes controller and worker nodes",
    },
    {
      label: "Success Rate",
      value: formatPercentage(summary.success_rate, 1),
      badge: rangeLabel,
      summary: "Execution success ratio",
	      detail: "Terminal workflow runs in range",
    },
  ]
}

export function formatDashboardDateTime(value: string) {
  if (!value) {
    return "-"
  }

  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value))
}

export function formatDashboardDuration(value: number | null) {
  if (value == null) {
    return "-"
  }

  if (value < 60) {
    return `${value}s`
  }

  const totalMinutes = Math.floor(value / 60)
  const seconds = value % 60

  if (totalMinutes < 60) {
    if (seconds === 0) {
      return `${totalMinutes}m`
    }
    return `${totalMinutes}m ${seconds}s`
  }

  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60

  if (minutes === 0) {
    return `${hours}h`
  }

  return `${hours}h ${minutes}m`
}

export function formatPercentage(value: number, fractionDigits = 1) {
  return `${new Intl.NumberFormat("en-US", {
    maximumFractionDigits: fractionDigits,
  }).format(value)}%`
}

export function formatNodeRole(role: string) {
  if (!role) {
    return "-"
  }
  return role.charAt(0).toUpperCase() + role.slice(1)
}

export function parseDashboardBucketDate(value: string) {
  return new Date(`${value}T00:00:00`)
}
