import * as React from "react"
import "./App.css"
import { Outlet, createBrowserRouter, RouterProvider } from "react-router-dom"
import { AuthProvider } from "@/components/auth-provider"
import { RequireAuth } from "@/components/require-auth"
import { ThemeProvider } from "@/components/theme-provider"

const Page = React.lazy(() => import("./app/page"))
const Login = React.lazy(async () => {
  const mod = await import("./app/login/login")
  return { default: mod.Login }
})

function AppLoader() {
  return (
    <div className="flex min-h-screen items-center justify-center text-sm text-muted-foreground">
      Loading application...
    </div>
  )
}

function AuthLayout() {
  return (
    <AuthProvider>
      <Outlet />
    </AuthProvider>
  )
}

const router = createBrowserRouter([
  {
    element: <AuthLayout />,
    children: [
      {
        path: "/login",
        element: (
          <React.Suspense fallback={<AppLoader />}>
            <Login />
          </React.Suspense>
        ),
      },
      {
        element: <RequireAuth />,
        children: [
          {
            path: "*",
            element: (
              <React.Suspense fallback={<AppLoader />}>
                <Page />
              </React.Suspense>
            ),
          },
        ],
      },
    ],
  },
])

function App() {
  return (
    <ThemeProvider
      attribute="class"
      defaultTheme="system"
      enableSystem
      disableTransitionOnChange
    >
      <RouterProvider router={router} />
    </ThemeProvider>
  )
}

export default App
