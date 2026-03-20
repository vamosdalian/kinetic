import { describe, expect, it, vi } from "vitest"

import {
  buildDashboardPath,
  fetchDashboard,
  formatDashboardDuration,
  getDashboardTimezone,
} from "./dashboard"

describe("dashboard helpers", () => {
  it("builds dashboard API paths with range and timezone", () => {
    expect(buildDashboardPath("30d", "Asia/Shanghai")).toBe(
      "/api/dashboard?range=30d&tz=Asia%2FShanghai"
    )
  })

  it("formats workflow durations for short and long runs", () => {
    expect(formatDashboardDuration(null)).toBe("-")
    expect(formatDashboardDuration(18)).toBe("18s")
    expect(formatDashboardDuration(65)).toBe("1m 5s")
    expect(formatDashboardDuration(3600)).toBe("1h")
  })

  it("falls back to UTC when the browser timezone is unavailable", () => {
    const dateTimeFormat = vi
      .spyOn(Intl, "DateTimeFormat")
      .mockImplementation(
        () =>
          ({
            resolvedOptions: () => ({ timeZone: "" }),
          }) as Intl.DateTimeFormat
      )

    expect(getDashboardTimezone()).toBe("UTC")
    dateTimeFormat.mockRestore()
  })
})

describe("fetchDashboard", () => {
  it("requests the aggregated dashboard endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: vi.fn().mockResolvedValue({
        success: true,
        data: {
          summary: {
            workflow_runs_this_month: 1,
            total_workflows: 2,
            total_nodes: 3,
            success_rate_90d: 50,
          },
          chart: {
            range: "30d",
            timezone: "Asia/Shanghai",
            points: [],
          },
          tables: {
            today_workflows: { count: 0, items: [] },
            scheduled_runs: { count: 0, items: [] },
            failed_workflows: { count: 0, items: [] },
            node_activity: { count: 0, items: [] },
          },
        },
      }),
    })

    vi.stubGlobal("fetch", fetchMock)

    await fetchDashboard("30d", "Asia/Shanghai")

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/dashboard?range=30d&tz=Asia%2FShanghai",
      undefined
    )
  })
})
