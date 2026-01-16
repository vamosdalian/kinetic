import * as React from "react";
import { useNavigate } from "react-router-dom";
import {
  type ColumnDef,
} from "@tanstack/react-table";
import {
  ArrowUpDown,
  MoreHorizontal,
  Eye,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { apiClient } from "@/lib/api";
import { CommonTable } from "@/components/common-table";
import { type WorkflowRunListItem } from "./types";

export function Record() {
  const navigate = useNavigate();
  const [data, setData] = React.useState<WorkflowRunListItem[]>([]);

  React.useEffect(() => {
    // Note: Pagination support is available in API but for now we fetch all/defaults
    apiClient<WorkflowRunListItem[]>("/api/workflow_runs").then((data) => {
      setData(data);
    });
  }, []);

  const columns = React.useMemo<ColumnDef<WorkflowRunListItem>[]>(
    () => [
      {
        accessorKey: "run_id",
        header: "Run ID",
        cell: ({ row }) => <div className="font-mono text-xs">{row.getValue("run_id")}</div>,
      },
      {
        accessorKey: "name",
        header: ({ column }) => {
          return (
            <Button
              variant="ghost"
              onClick={() =>
                column.toggleSorting(column.getIsSorted() === "asc")
              }
            >
              Workflow Name
              <ArrowUpDown className="ml-2 h-4 w-4" />
            </Button>
          );
        },
        cell: ({ row }) => (
          <div className="capitalize pl-3">{row.getValue("name")}</div>
        ),
      },
      {
        accessorKey: "version",
        header: "Version",
        cell: ({ row }) => (
          <div className="capitalize">{row.getValue("version")}</div>
        ),
      },
      {
        accessorKey: "status",
        header: "Status",
        cell: ({ row }) => (
          <div className="capitalize">{row.getValue("status")}</div>
        ),
      },
      {
        accessorKey: "create_at",
        header: ({ column }) => {
          return (
            <Button
              variant="ghost"
              onClick={() =>
                column.toggleSorting(column.getIsSorted() === "asc")
              }
            >
              Created At
              <ArrowUpDown className="ml-2 h-4 w-4" />
            </Button>
          );
        },
        cell: ({ row }) => (
          <div className="capitalize pl-3">{row.getValue("create_at")}</div>
        ),
      },
       {
        accessorKey: "started_at",
        header: "Started At",
        cell: ({ row }) => (
          <div className="capitalize">{row.getValue("started_at")}</div>
        ),
      },
       {
        accessorKey: "finished_at",
        header: "Finished At",
        cell: ({ row }) => (
          <div className="capitalize">{row.getValue("finished_at")}</div>
        ),
      },
      {
        id: "actions",
        header: "Operation",
        enableHiding: false,
        cell: ({ row }) => {
          const run = row.original;

          return (
            <div className="flex items-center space-x-1">
              <Button
                variant="ghost"
                className="h-8 w-8 p-0"
                onClick={() => {
                  navigate(`/record/${run.run_id}`);
                }}
              >
                <Eye className="h-4 w-4" />
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
                    onClick={() => navigator.clipboard.writeText(run.run_id)}
                  >
                    Copy Run ID
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          );
        },
      },
    ],
    [navigate]
  );

  return <CommonTable columns={columns} data={data} />;
}
