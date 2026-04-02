import { Navigate, Outlet, useLocation } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"

export function RequireAuth() {
  const location = useLocation()
  const { isReady, isAuthenticated } = useAuth()

  if (!isReady) {
    return (
      <div className="flex min-h-screen items-center justify-center text-sm text-muted-foreground">
        Loading application...
      </div>
    )
  }

  if (!isAuthenticated) {
    return (
      <Navigate
        replace
        to={`/login?redirect=${encodeURIComponent(`${location.pathname}${location.search}`)}`}
      />
    )
  }

  return <Outlet />
}
