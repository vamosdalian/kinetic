import { afterEach, describe, expect, it, vi } from "vitest"

import { apiClient, apiClientFull } from "./api"
import { AUTH_TOKEN_STORAGE_KEY } from "./auth"

describe("apiClient", () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it("returns response data for successful requests", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: vi.fn().mockResolvedValue({
          success: true,
          data: { id: "run-1" },
        }),
      })
    )

    await expect(apiClient<{ id: string }>("/api/workflow_runs/run-1")).resolves.toEqual({
      id: "run-1",
    })
  })

  it("throws parsed API error messages for non-2xx responses", async () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => undefined)

    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: vi.fn().mockResolvedValue({
          error: {
            message: "backend unavailable",
          },
        }),
      })
    )

    await expect(apiClient("/api/workflow_runs/run-1")).rejects.toThrow("backend unavailable")
    expect(consoleError).toHaveBeenCalledWith("backend unavailable")
  })

  it("throws API envelope errors when success is false", async () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => undefined)

    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: vi.fn().mockResolvedValue({
          success: false,
          error: {
            message: "validation failed",
          },
        }),
      })
    )

    await expect(apiClient("/api/workflows")).rejects.toThrow("validation failed")
    expect(consoleError).toHaveBeenCalledWith("API Error:", "validation failed", {
      message: "validation failed",
    })
  })

  it("adds the bearer token when one is stored", async () => {
    window.localStorage.setItem(AUTH_TOKEN_STORAGE_KEY, "token-123")
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: vi.fn().mockResolvedValue({
        success: true,
        data: { ok: true },
      }),
    })
    vi.stubGlobal("fetch", fetchMock)

    await apiClient<{ ok: boolean }>("/api/dashboard")

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [, init] = fetchMock.mock.calls[0]
    expect(init?.headers).toBeInstanceOf(Headers)
    expect(new Headers(init?.headers).get("Authorization")).toBe("Bearer token-123")
  })
})

describe("apiClientFull", () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it("returns the full API envelope for successful requests", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: vi.fn().mockResolvedValue({
          success: true,
          data: ["a", "b"],
          meta: {
            page: 1,
            pageSize: 10,
            total: 2,
            totalPages: 1,
          },
        }),
      })
    )

    await expect(apiClientFull<string[]>("/api/workflows")).resolves.toEqual({
      success: true,
      data: ["a", "b"],
      meta: {
        page: 1,
        pageSize: 10,
        total: 2,
        totalPages: 1,
      },
    })
  })
})
