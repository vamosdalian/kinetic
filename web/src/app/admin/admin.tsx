import * as React from "react"
import { type ColumnDef } from "@tanstack/react-table"
import { LoaderCircle, MoreHorizontal, Pencil, Plus, RefreshCw, Trash2 } from "lucide-react"
import { toast } from "sonner"

import { useAuth } from "@/components/auth-provider"
import { CommonTable } from "@/components/common-table"
import { SiteHeader } from "@/components/site-header"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { apiClient } from "@/lib/api"
import { type AdminUserListItem } from "@/lib/auth"

type UserDialogMode = "create" | "password" | "delete" | null

export function Admin() {
  const { user: currentUser } = useAuth()
  const [users, setUsers] = React.useState<AdminUserListItem[]>([])
  const [loading, setLoading] = React.useState(true)
  const [submitting, setSubmitting] = React.useState(false)
  const [dialogMode, setDialogMode] = React.useState<UserDialogMode>(null)
  const [selectedUser, setSelectedUser] = React.useState<AdminUserListItem | null>(null)
  const [username, setUsername] = React.useState("")
  const [password, setPassword] = React.useState("")

  const fetchUsers = React.useCallback(async (showLoader = true) => {
    if (showLoader) {
      setLoading(true)
    }
    try {
      const nextUsers = await apiClient<AdminUserListItem[]>("/api/admin/users")
      setUsers(nextUsers)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to load users")
    } finally {
      if (showLoader) {
        setLoading(false)
      }
    }
  }, [])

  React.useEffect(() => {
    void fetchUsers()
  }, [fetchUsers])

  const resetDialog = React.useCallback(() => {
    setDialogMode(null)
    setSelectedUser(null)
    setUsername("")
    setPassword("")
  }, [])

  const openCreateDialog = React.useCallback(() => {
    setDialogMode("create")
    setSelectedUser(null)
    setUsername("")
    setPassword("")
  }, [])

  const openPasswordDialog = React.useCallback((nextUser: AdminUserListItem) => {
    setDialogMode("password")
    setSelectedUser(nextUser)
    setPassword("")
  }, [])

  const openDeleteDialog = React.useCallback((nextUser: AdminUserListItem) => {
    setDialogMode("delete")
    setSelectedUser(nextUser)
  }, [])

  const handleSubmit = React.useCallback(async () => {
    if (dialogMode === "create") {
      setSubmitting(true)
      try {
        await apiClient<AdminUserListItem>("/api/admin/users", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            username: username.trim(),
            password,
          }),
        })
        toast.success("User created")
        resetDialog()
        await fetchUsers(false)
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to create user")
      } finally {
        setSubmitting(false)
      }
      return
    }

    if (dialogMode === "password" && selectedUser) {
      setSubmitting(true)
      try {
        await apiClient(`/api/admin/users/${selectedUser.id}/password`, {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            password,
          }),
        })
        toast.success("Password updated")
        resetDialog()
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to update password")
      } finally {
        setSubmitting(false)
      }
      return
    }

    if (dialogMode === "delete" && selectedUser) {
      setSubmitting(true)
      try {
        await apiClient(`/api/admin/users/${selectedUser.id}`, {
          method: "DELETE",
        })
        toast.success("User deleted")
        resetDialog()
        await fetchUsers(false)
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to delete user")
      } finally {
        setSubmitting(false)
      }
    }
  }, [dialogMode, fetchUsers, password, resetDialog, selectedUser, username])

  const columns = React.useMemo<ColumnDef<AdminUserListItem>[]>(
    () => [
      {
        accessorKey: "id",
        header: "User ID",
        cell: ({ row }) => <div className="font-mono text-xs">{row.original.id}</div>,
      },
      {
        accessorKey: "username",
        header: "Username",
        cell: ({ row }) => (
          <div className="flex min-w-[220px] flex-col gap-1">
            <div className="flex items-center gap-2">
              <span className="font-medium">{row.original.username}</span>
              {row.original.is_bootstrap_admin ? (
                <Badge variant="secondary">bootstrap</Badge>
              ) : null}
              {currentUser?.id === row.original.id ? (
                <Badge variant="outline">you</Badge>
              ) : null}
            </div>
          </div>
        ),
      },
      {
        accessorKey: "permission",
        header: "Permission",
        cell: ({ row }) => <Badge variant="outline">{row.original.permission}</Badge>,
      },
      {
        accessorKey: "created_at",
        header: "Created",
      },
      {
        accessorKey: "updated_at",
        header: "Updated",
      },
      {
        id: "actions",
        header: "Actions",
        enableHiding: false,
        cell: ({ row }) => {
          const account = row.original
          const deleteDisabled = account.is_bootstrap_admin || currentUser?.id === account.id

          return (
            <div className="flex items-center space-x-1">
              <Button
                variant="ghost"
                className="h-8 w-8 p-0"
                onClick={() => openPasswordDialog(account)}
              >
                <span className="sr-only">Reset password</span>
                <Pencil className="h-4 w-4" />
              </Button>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" className="h-8 w-8 p-0">
                    <span className="sr-only">Open menu</span>
                    <MoreHorizontal className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuLabel>Actions</DropdownMenuLabel>
                  <DropdownMenuItem onClick={() => navigator.clipboard.writeText(account.id)}>
                    Copy User ID
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    disabled={deleteDisabled}
                    onClick={() => openDeleteDialog(account)}
                  >
                    <Trash2 className="mr-2 h-4 w-4" />
                    Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          )
        },
      },
    ],
    [currentUser?.id, openDeleteDialog, openPasswordDialog]
  )

  return (
    <div className="flex flex-1 flex-col min-h-0">
      <SiteHeader breadcrumbs={[{ label: "Admin", href: null }]} />
      <CommonTable
        columns={columns}
        data={users}
        loading={loading}
        initialColumnVisibility={{ id: false }}
        renderToolbarActions={() => (
          <>
            <Button variant="outline" onClick={() => void fetchUsers()}>
              <RefreshCw className="h-4 w-4" />
              Refresh
            </Button>
            <Button variant="outline" onClick={openCreateDialog}>
              <Plus className="h-4 w-4" />
              New User
            </Button>
          </>
        )}
      />

      <Dialog open={dialogMode !== null} onOpenChange={(open) => (!open ? resetDialog() : undefined)}>
        <DialogContent>
          {dialogMode === "create" ? (
            <>
              <DialogHeader>
                <DialogTitle>Create User</DialogTitle>
                <DialogDescription>Create another admin account for the controller UI.</DialogDescription>
              </DialogHeader>
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="create-username">Username</Label>
                  <Input
                    id="create-username"
                    disabled={submitting}
                    value={username}
                    onChange={(event) => setUsername(event.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="create-password">Password</Label>
                  <Input
                    id="create-password"
                    disabled={submitting}
                    type="password"
                    value={password}
                    onChange={(event) => setPassword(event.target.value)}
                  />
                </div>
              </div>
            </>
          ) : null}

          {dialogMode === "password" && selectedUser ? (
            <>
              <DialogHeader>
                <DialogTitle>Reset Password</DialogTitle>
                <DialogDescription>Update the password for {selectedUser.username}.</DialogDescription>
              </DialogHeader>
              <div className="space-y-2">
                <Label htmlFor="reset-password">New Password</Label>
                <Input
                  id="reset-password"
                  disabled={submitting}
                  type="password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                />
              </div>
            </>
          ) : null}

          {dialogMode === "delete" && selectedUser ? (
            <>
              <DialogHeader>
                <DialogTitle>Delete User</DialogTitle>
                <DialogDescription>
                  Delete {selectedUser.username}. This action cannot be undone.
                </DialogDescription>
              </DialogHeader>
            </>
          ) : null}

          <DialogFooter>
            <Button disabled={submitting} variant="outline" onClick={resetDialog}>
              Cancel
            </Button>
            <Button disabled={submitting} variant={dialogMode === "delete" ? "destructive" : "default"} onClick={() => void handleSubmit()}>
              {submitting ? <LoaderCircle className="h-4 w-4 animate-spin" /> : null}
              {dialogMode === "create" ? "Create User" : null}
              {dialogMode === "password" ? "Update Password" : null}
              {dialogMode === "delete" ? "Delete User" : null}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
