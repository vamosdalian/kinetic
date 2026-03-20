import * as React from "react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"

import { ChartAreaInteractive } from "@/components/chart-area-interactive"
import { DataTable } from "@/components/data-table"
import { SectionCards } from "@/components/section-cards"
import { SiteHeader } from "@/components/site-header"
import {
  buildDashboardCardMetrics,
  createEmptyDashboard,
  fetchDashboard,
  getDashboardTimezone,
  type DashboardRange,
} from "@/lib/dashboard"
import { apiClient } from "@/lib/api"

export function Dashboard() {
  const navigate = useNavigate()
  const [timeRange, setTimeRange] = React.useState<DashboardRange>("30d")
  const [timezone] = React.useState(() => getDashboardTimezone())
  const [dashboard, setDashboard] = React.useState(() => createEmptyDashboard("30d", timezone))
  const [rerunningRunID, setRerunningRunID] = React.useState<string | null>(null)

  React.useEffect(() => {
    let active = true

    void fetchDashboard(timeRange, timezone)
      .then((response) => {
        if (!active) {
          return
        }
        React.startTransition(() => {
          setDashboard(response)
        })
      })
      .catch((error) => {
        if (!active) {
          return
        }
        toast.error(error instanceof Error ? error.message : "Failed to load dashboard")
      })

    return () => {
      active = false
    }
  }, [timeRange, timezone])

  const handleRerunWorkflowRun = React.useCallback(
    async (runID: string) => {
      setRerunningRunID(runID)
      try {
        const response = await apiClient<{ run_id: string }>(
          `/api/workflow_runs/${runID}/rerun`,
          {
            method: "POST",
          }
        )
        toast.success("Workflow run restarted")
        navigate(`/record/${response.run_id}`)
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to rerun workflow")
      } finally {
        setRerunningRunID(null)
      }
    },
    [navigate]
  )

  const handleViewWorkflowRun = React.useCallback(
    (runID: string) => {
      navigate(`/record/${runID}`)
    },
    [navigate]
  )

  const handleOpenNodes = React.useCallback(() => {
    navigate("/node")
  }, [navigate])

  const cardMetrics = buildDashboardCardMetrics(dashboard.summary)

  return (
    <div className="flex flex-1 flex-col">
      <SiteHeader breadcrumbs={[{ label: "Dashboard", href: null }]} />
      <div className="@container/main flex flex-1 flex-col gap-2">
        <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
          <SectionCards metrics={cardMetrics} />
          <div className="px-4 lg:px-6">
            <ChartAreaInteractive
              data={dashboard.chart.points}
              timeRange={timeRange}
              onTimeRangeChange={setTimeRange}
            />
          </div>
          <DataTable
            tables={dashboard.tables}
            rerunningRunID={rerunningRunID}
            onViewWorkflowRun={handleViewWorkflowRun}
            onRerunWorkflowRun={handleRerunWorkflowRun}
            onOpenNodes={handleOpenNodes}
          />
        </div>
      </div>
    </div>
  )
}
