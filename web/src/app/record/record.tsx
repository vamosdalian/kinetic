import * as React from "react";
import { useNavigate } from "react-router-dom";
import { type ColumnDef } from "@tanstack/react-table";
import {
  ArrowUpDown,
  Eye,
  MoreHorizontal,
  RotateCcw,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { apiClient, apiClientFull } from "@/lib/api";
import { CommonTable } from "@/components/common-table";
import { SiteHeader } from "@/components/site-header";
import { type WorkflowRunListItem } from "./types";
import {
  getRunRowClassName,
  getStatusBadgeClassName,
} from "./status";
import { toast } from "sonner";

const statusOptions = [
  { value: "all", label: "All Status" },
  { value: "created", label: "Created" },
  { value: "running", label: "Running" },
  { value: "success", label: "Success" },
  { value: "failed", label: "Failed" },
  { value: "cancelled", label: "Cancelled" },
];

export function Record() {
  const navigate = useNavigate();
  const [data, setData] = React.useState<WorkflowRunListItem[]>([]);
  const [pagination, setPagination] = React.useState({
    pageIndex: 0,
    pageSize: 10,
  });
  const [pageCount, setPageCount] = React.useState(-1);
  const [workflowQuery, setWorkflowQuery] = React.useState("");
  const [runQuery, setRunQuery] = React.useState("");
  const [statusFilter, setStatusFilter] = React.useState("all");
  const [rerunningRunId, setRerunningRunId] = React.useState<string | null>(null);

  const fetchRuns = React.useCallback(async () => {
    try {
      const response = await apiClientFull<WorkflowRunListItem[]>(
        `/api/workflow_runs?page=${pagination.pageIndex + 1}&pageSize=${pagination.pageSize}`
      );
      setData(response.data);
      setPageCount(response.meta?.totalPages ?? -1);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to load workflow runs");
    }
  }, [pagination.pageIndex, pagination.pageSize]);

  React.useEffect(() => {
    void fetchRuns();
  }, [fetchRuns]);

  const handleRerun = React.useCallback(
    async (runID: string) => {
      setRerunningRunId(runID);
      try {
        const response = await apiClient<{ run_id: string }>(
          `/api/workflow_runs/${runID}/rerun`,
          {
            method: "POST",
          }
        );
        toast.success("Workflow run restarted");
        navigate(`/record/${response.run_id}`);
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to rerun workflow");
      } finally {
        setRerunningRunId(null);
      }
    },
    [navigate]
  );

  const filteredData = React.useMemo(() => {
    return data.filter((run) => {
      const workflowMatch = run.name
        .toLowerCase()
        .includes(workflowQuery.trim().toLowerCase());
      const runMatch = run.run_id
        .toLowerCase()
        .includes(runQuery.trim().toLowerCase());
      const statusMatch = statusFilter === "all" || run.status === statusFilter;
      return workflowMatch && runMatch && statusMatch;
    });
  }, [data, runQuery, statusFilter, workflowQuery]);

  React.useEffect(() => {
    setPagination((current) =>
      current.pageIndex === 0 ? current : { ...current, pageIndex: 0 }
    );
  }, [workflowQuery, runQuery, statusFilter]);

  const columns = React.useMemo<ColumnDef<WorkflowRunListItem>[]>(
    () => [
      {
        accessorKey: "run_id",
        header: "Run ID",
        cell: ({ row }) => (
          <div className="font-mono text-xs">{row.getValue("run_id")}</div>
        ),
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
        cell: ({ row }) => <div className="pl-3">{row.getValue("name")}</div>,
      },
      {
        accessorKey: "version",
        header: "Version",
        cell: ({ row }) => <div>{row.getValue("version")}</div>,
      },
      {
        accessorKey: "status",
        header: "Status",
        cell: ({ row }) => (
          <Badge
            variant="outline"
            className={`capitalize ${getStatusBadgeClassName(
              row.getValue("status")
            )}`}
          >
            {row.getValue("status")}
          </Badge>
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
        cell: ({ row }) => <div className="pl-3">{row.getValue("create_at")}</div>,
      },
      {
        accessorKey: "started_at",
        header: "Started At",
        cell: ({ row }) => <div>{row.getValue("started_at") || "-"}</div>,
      },
      {
        accessorKey: "finished_at",
        header: "Finished At",
        cell: ({ row }) => <div>{row.getValue("finished_at") || "-"}</div>,
      },
      {
        id: "actions",
        header: "Operation",
        enableHiding: false,
        cell: ({ row }) => {
          const run = row.original;
          const isRerunning = rerunningRunId === run.run_id;

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
              <Button
                variant="ghost"
                className="h-8 w-8 p-0"
                disabled={isRerunning}
                onClick={() => void handleRerun(run.run_id)}
              >
                <RotateCcw className="h-4 w-4" />
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
                  <DropdownMenuItem onClick={() => void handleRerun(run.run_id)}>
                    Rerun
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          );
        },
      },
    ],
    [handleRerun, navigate, rerunningRunId]
  );

  return (
    <div className="flex flex-1 flex-col min-h-0">
      <SiteHeader breadcrumbs={[{ label: "Record", href: null }]} />
      <CommonTable
        columns={columns}
        data={filteredData}
        manualPagination={true}
        pageCount={pageCount}
        pagination={pagination}
        onPaginationChange={setPagination}
        getRowClassName={(row) => getRunRowClassName(row.status)}
        renderToolbarActions={() => (
          <>
            <Input
              className="w-[220px]"
              placeholder="Search workflow..."
              value={workflowQuery}
              onChange={(e) => setWorkflowQuery(e.target.value)}
            />
            <Input
              className="w-[220px]"
              placeholder="Search run ID..."
              value={runQuery}
              onChange={(e) => setRunQuery(e.target.value)}
            />
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-[170px]">
                <SelectValue placeholder="Filter status" />
              </SelectTrigger>
              <SelectContent>
                {statusOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button variant="outline" onClick={() => void fetchRuns()}>
              Refresh
            </Button>
          </>
        )}
      />
    </div>
  );
}
