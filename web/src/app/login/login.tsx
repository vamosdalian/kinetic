import * as React from "react"
import { Workflow } from "lucide-react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { useAuth } from "@/components/auth-provider"
import { LoginForm } from "@/components/login-form"

export function Login() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { isAuthenticated, login } = useAuth()
  const [email, setEmail] = React.useState("")
  const [password, setPassword] = React.useState("")
  const [submitting, setSubmitting] = React.useState(false)

  React.useEffect(() => {
    if (!isAuthenticated) {
      return
    }
    navigate(searchParams.get("redirect") || "/", { replace: true })
  }, [isAuthenticated, navigate, searchParams])

  const handleSubmit = React.useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault()
      setSubmitting(true)
      try {
        await login({ username: email.trim(), password })
        toast.success("Signed in")
        navigate(searchParams.get("redirect") || "/", { replace: true })
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to sign in")
      } finally {
        setSubmitting(false)
      }
    },
    [email, login, navigate, password, searchParams]
  )

  return (
    <div className="grid min-h-svh lg:grid-cols-2">
      <div className="flex flex-col gap-4 p-6 md:p-10">
        <div className="flex justify-center gap-2 md:justify-start">
          <a href="#" className="flex items-center gap-2 font-medium">
            <div className="flex size-6 items-center justify-center rounded-md border bg-background text-foreground">
              <Workflow className="size-4" />
            </div>
            Kinetic
          </a>
        </div>

        <div className="flex flex-1 items-center justify-center">
          <div className="w-full max-w-xs">
            <LoginForm
              email={email}
              password={password}
              submitting={submitting}
              onEmailChange={setEmail}
              onPasswordChange={setPassword}
              onSubmit={handleSubmit}
            />
          </div>
        </div>
      </div>

      <div className="relative hidden bg-muted lg:block">
        <img
          alt="Kinetic workflow preview"
          className="absolute inset-0 h-full w-full object-cover"
          src="/placeholder.png"
        />
      </div>
    </div>
  )
}
