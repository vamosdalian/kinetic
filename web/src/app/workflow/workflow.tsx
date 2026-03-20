import * as React from "react";
import { useNavigate } from "react-router-dom";
import { v7 as uuidv7 } from "uuid";
import {
  type ColumnDef,
} from "@tanstack/react-table";
import {
  ArrowUpDown,
  MoreHorizontal,
  Play,
  SquarePen,
} from "lucide-react";
import { toast } from "sonner"
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { apiClient, apiClientFull } from "@/lib/api";
import { SiteHeader } from "@/components/site-header";
import { CommonTable } from "@/components/common-table";
import type {WorkflowListItem} from "./types"
// export type Workflow = {
//   id: string;
//   name: string;
//   enable: boolean;
//   version: string;
//   last_running_status: "pending" | "running" | "success" | "failed";
//   next_running_time: string;
//   create_at: string;
//   update_at: string;
// };

export function Workflow() {
  const navigate = useNavigate();
  const [data, setData] = React.useState<WorkflowListItem[]>([]);
  const [pagination, setPagination] = React.useState({
    pageIndex: 0,
    pageSize: 20,
  });
  const [pageCount, setPageCount] = React.useState(-1);
  const [query, setQuery] = React.useState("");

  const fetchWorkflows = React.useCallback(() => {
    const params = new URLSearchParams({
      page: String(pagination.pageIndex + 1),
      pageSize: String(pagination.pageSize),
    });
    if (query.trim()) {
      params.set("query", query.trim());
    }

    apiClientFull<WorkflowListItem[]>(
      `/api/workflows?${params.toString()}`
    ).then((res) => {
      setData(res.data);
      if (res.meta) {
        setPageCount(res.meta.totalPages);
      }
    });
  }, [pagination.pageIndex, pagination.pageSize, query]);

  React.useEffect(() => {
    fetchWorkflows();
  }, [fetchWorkflows]);

  React.useEffect(() => {
    setPagination((current) =>
      current.pageIndex === 0 ? current : { ...current, pageIndex: 0 }
    );
  }, [query]);

  const columns: ColumnDef<WorkflowListItem>[] = [
    {
      accessorKey: "name",
      header: ({ column }) => {
        return (
          <Button
            variant="ghost"
            onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
          >
            Name
            <ArrowUpDown className="ml-2 h-4 w-4" />
          </Button>
        );
      },
      cell: ({ row }) => (
        <div className="capitalize pl-3">{row.getValue("name")}</div>
      ),
    },
    {
      accessorKey: "enable",
      header: "Enable",
      cell: ({ row }) => (
        <div className="capitalize">{String(row.getValue("enable"))}</div>
      ),
    },
    {
      accessorKey: "version",
      header: "Version",
      cell: ({ row }) => (
        <div className="capitalize">{row.getValue("version")}</div>
      ),
    },
    // {
    //   accessorKey: "last_running_status",
    //   header: "Last Running Status",
    //   cell: ({ row }) => (
    //     <div className="lowercase">{row.getValue("last_running_status")}</div>
    //   ),
    // },
    // {
    //   accessorKey: "next_running_time",
    //   header: ({ column }) => {
    //     return (
    //       <Button
    //         variant="ghost"
    //         onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
    //       >
    //         Next Running Time
    //         <ArrowUpDown className="ml-2 h-4 w-4" />
    //       </Button>
    //     );
    //   },
    //   cell: ({ row }) => (
    //     <div className="capitalize pl-3">
    //       {row.getValue("next_running_time")}
    //     </div>
    //   ),
    // },
    {
      accessorKey: "create_at",
      header: ({ column }) => {
        return (
          <Button
            variant="ghost"
            onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
          >
            Create At
            <ArrowUpDown className="ml-2 h-4 w-4" />
          </Button>
        );
      },
      cell: ({ row }) => (
        <div className="capitalize pl-3">{row.getValue("create_at")}</div>
      ),
    },
    {
      accessorKey: "update_at",
      header: ({ column }) => {
        return (
          <Button
            variant="ghost"
            onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
          >
            Update At
            <ArrowUpDown className="ml-2 h-4 w-4" />
          </Button>
        );
      },
      cell: ({ row }) => (
        <div className="capitalize pl-3">{row.getValue("update_at")}</div>
      ),
    },
    {
      id: "actions",
      header: "Operation",
      enableHiding: false,
      cell: ({ row }) => {
        const workflow = row.original;

        const handleRun = async () => {
          try {
            await apiClient(`/api/workflows/${workflow.id}/run`, {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({}),
            });
            console.log("Workflow run started:", data);
            toast.success("Workflow run started successfully")
          } catch (error) {
            console.error("Error starting workflow run:", error);
            toast.error("Failed to start workflow run")
          }
        };

        const handleDelete = async () => {
          try {
            await apiClient(`/api/workflows/${workflow.id}`, {
              method: "DELETE",
            });
            fetchWorkflows();
            toast.success("Workflow deleted successfully");
          } catch (error) {
            console.error("Error deleting workflow:", error);
            toast.error("Failed to delete workflow");
          }
        };

        return (
          <div className="flex items-center space-x-1">
            <Button variant="ghost" className="h-8 w-8 p-0" onClick={handleRun}>
              <Play className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              className="h-8 w-8 p-0"
              onClick={() => {
                navigate(`/workflow/${workflow.id}`);
              }}
            >
              <SquarePen className="h-4 w-4" />
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
                <DropdownMenuItem
                  onClick={() => navigator.clipboard.writeText(workflow.id)}
                >
                  Copy workflow ID
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={handleDelete}
                  className="text-red-600"
                >
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        );
      },
    },
  ];

  return (
    <div className="flex flex-1 flex-col">
      <SiteHeader breadcrumbs={[{ label: "Workflow", href: null }]} />
      <CommonTable
        columns={columns}
        data={data}
        manualPagination={true}
        pageCount={pageCount}
        pagination={pagination}
        onPaginationChange={setPagination}
        renderToolbarActions={() => (
          <>
            <Input
              className="w-[280px]"
              placeholder="Search workflow name or ID..."
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
            <Button variant="outline" onClick={() => fetchWorkflows()}>
              Refresh
            </Button>
            <Button
              variant="outline"
              onClick={() => {
                const newId = uuidv7();
                navigate(`/workflow/${newId}?action=create`);
              }}
            >
              Create
            </Button>
          </>
        )}
      />
    </div>
  );
}
