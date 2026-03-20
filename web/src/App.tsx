import * as React from "react"
import "./App.css"
import { createBrowserRouter, RouterProvider } from "react-router-dom"
import { ThemeProvider } from "@/components/theme-provider"

const Page = React.lazy(() => import("./app/page"))

function AppLoader() {
  return (
    <div className="flex min-h-screen items-center justify-center text-sm text-muted-foreground">
      Loading application...
    </div>
  )
}

const router = createBrowserRouter([
  {
    path: "*",
    element: (
      <React.Suspense fallback={<AppLoader />}>
        <Page />
      </React.Suspense>
    ),
  }
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
