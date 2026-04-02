export const AUTH_TOKEN_STORAGE_KEY = "kinetic.auth.token"

export interface AuthUser {
  id: string
  username: string
  permission: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user: AuthUser
}

export interface AdminUserListItem {
  id: string
  username: string
  permission: string
  is_bootstrap_admin: boolean
  created_at: string
  updated_at: string
}

export function getStoredAuthToken(): string {
  if (typeof window === "undefined") {
    return ""
  }
  return window.localStorage.getItem(AUTH_TOKEN_STORAGE_KEY) ?? ""
}

export function setStoredAuthToken(token: string) {
  if (typeof window === "undefined") {
    return
  }
  window.localStorage.setItem(AUTH_TOKEN_STORAGE_KEY, token)
}

export function clearStoredAuthToken() {
  if (typeof window === "undefined") {
    return
  }
  window.localStorage.removeItem(AUTH_TOKEN_STORAGE_KEY)
}

export function getCurrentRelativeURL() {
  if (typeof window === "undefined") {
    return "/"
  }
  return `${window.location.pathname}${window.location.search}${window.location.hash}`
}

export function buildLoginRedirectURL(target?: string) {
  const redirect = target ?? getCurrentRelativeURL()
  const params = new URLSearchParams()
  if (redirect && redirect !== "/login") {
    params.set("redirect", redirect)
  }
  return params.size > 0 ? `/login?${params.toString()}` : "/login"
}

export function redirectToLogin(target?: string) {
  if (typeof window === "undefined") {
    return
  }
  window.location.assign(buildLoginRedirectURL(target))
}

export function buildEventSourceURL(path: string) {
  const token = getStoredAuthToken()
  if (!token) {
    return path
  }
  const url = new URL(path, window.location.origin)
  url.searchParams.set("access_token", token)
  return `${url.pathname}${url.search}`
}
