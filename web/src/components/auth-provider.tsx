import * as React from "react"
import { useLocation, useNavigate } from "react-router-dom"

import { apiClientPublic } from "@/lib/api"
import {
  type AuthUser,
  type LoginRequest,
  type LoginResponse,
  clearStoredAuthToken,
  getStoredAuthToken,
  setStoredAuthToken,
} from "@/lib/auth"

type AuthContextValue = {
  user: AuthUser | null
  isReady: boolean
  isAuthenticated: boolean
  login: (payload: LoginRequest) => Promise<AuthUser>
  logout: () => void
  refresh: () => Promise<void>
}

const AuthContext = React.createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate()
  const location = useLocation()
  const [user, setUser] = React.useState<AuthUser | null>(null)
  const [isReady, setIsReady] = React.useState(false)

  const logout = React.useCallback(() => {
    clearStoredAuthToken()
    setUser(null)
    navigate("/login", { replace: true })
  }, [navigate])

  const refresh = React.useCallback(async () => {
    const token = getStoredAuthToken()
    if (!token) {
      setUser(null)
      setIsReady(true)
      return
    }

    try {
      const nextUser = await apiClientPublic<AuthUser>("/api/auth/me", {
        skipAuthRedirect: true,
      })
      setUser(nextUser)
    } catch {
      clearStoredAuthToken()
      setUser(null)
    } finally {
      setIsReady(true)
    }
  }, [])

  React.useEffect(() => {
    void refresh()
  }, [refresh])

  React.useEffect(() => {
    if (!isReady) {
      return
    }
    if (location.pathname === "/login") {
      return
    }
    if (!user && !getStoredAuthToken()) {
      navigate(`/login?redirect=${encodeURIComponent(`${location.pathname}${location.search}`)}`, {
        replace: true,
      })
    }
  }, [isReady, location.pathname, location.search, navigate, user])

  const login = React.useCallback(async (payload: LoginRequest) => {
    const response = await apiClientPublic<LoginResponse>("/api/auth/login", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
      skipAuthRedirect: true,
      skipAuthToken: true,
    })
    setStoredAuthToken(response.token)
    setUser(response.user)
    return response.user
  }, [])

  const value = React.useMemo<AuthContextValue>(
    () => ({
      user,
      isReady,
      isAuthenticated: Boolean(user),
      login,
      logout,
      refresh,
    }),
    [isReady, login, logout, refresh, user]
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const context = React.useContext(AuthContext)
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider")
  }
  return context
}
