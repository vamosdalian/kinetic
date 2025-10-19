import * as React from "react";
import { useNavigate } from "react-router-dom";
import {
  type ColumnDef,
  type ColumnFiltersState,
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type SortingState,
  useReactTable,
  type VisibilityState,
} from "@tanstack/react-table";
import {
  ArrowUpDown,
  ChevronDown,
  MoreHorizontal,
  Play,
  SquarePen,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useWorkflowStore } from "./workflow-store";

const data: Workflow[] = [
  {
    id: "m5gr84i9",
    name: "Workflow 1",
    enable: true,
    version: "1.0.0",
    last_running_status: "success",
    next_running_time: "2025-09-16 10:00:00",
    create_at: "2025-09-15 10:00:00",
  },
  {
    id: "3u1reuv4",
    name: "Workflow 2",
    enable: false,
    version: "1.2.0",
    last_running_status: "failed",
    next_running_time: "2025-09-17 12:00:00",
    create_at: "2025-09-14 11:00:00",
  },
  {
    id: "derv1ws0",
    name: "Workflow 3",
    enable: true,
    version: "2.0.0",
    last_running_status: "running",
    next_running_time: "2025-09-16 14:00:00",
    create_at: "2025-09-13 14:00:00",
  },
];

export type Workflow = {
  id: string;
  name: string;
  enable: boolean;
  version: string;
  last_running_status: "pending" | "running" | "success" | "failed";
  next_running_time: string;
  create_at: string;
};

export function Workflow() {
  const navigate = useNavigate();
  const { setWorkflowId } = useWorkflowStore();
  const columns = React.useMemo<ColumnDef<Workflow>[]>(
    () => [
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
      {
        accessorKey: "last_running_status",
        header: "Last Running Status",
        cell: ({ row }) => (
          <div className="lowercase">{row.getValue("last_running_status")}</div>
        ),
      },
      {
        accessorKey: "next_running_time",
        header: ({ column }) => {
          return (
            <Button
              variant="ghost"
              onClick={() =>
                column.toggleSorting(column.getIsSorted() === "asc")
              }
            >
              Next Running Time
              <ArrowUpDown className="ml-2 h-4 w-4" />
            </Button>
          );
        },
        cell: ({ row }) => (
          <div className="capitalize pl-3">
            {row.getValue("next_running_time")}
          </div>
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
        id: "actions",
        header: "Operation",
        enableHiding: false,
        cell: ({ row }) => {
          const workflow = row.original;

          return (
            <div className="flex items-center space-x-1">
              <Button variant="ghost" className="h-8 w-8 p-0">
                <Play className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                className="h-8 w-8 p-0"
                onClick={() => {
                  setWorkflowId(workflow.id);
                  navigate(`/workflow/${workflow.name}`);
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
                  <DropdownMenuItem>View workflow details</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          );
        },
      },
    ],
    [navigate, setWorkflowId]
  );

  const [sorting, setSorting] = React.useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>(
    []
  );
  const [columnVisibility, setColumnVisibility] =
    React.useState<VisibilityState>({});

  const table = useReactTable({
    data,
    columns,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    onColumnVisibilityChange: setColumnVisibility,
    state: {
      sorting,
      columnFilters,
      columnVisibility,
    },
  });

  return (
    <div className="w-full p-4">
      <div className="flex items-center justify-between py-4">
        <Input
          placeholder="Filter name..."
          value={(table.getColumn("name")?.getFilterValue() as string) ?? ""}
          onChange={(event) =>
            table.getColumn("name")?.setFilterValue(event.target.value)
          }
          className="max-w-sm"
        />
        <div className="space-x-2">
          <Button variant="outline" onClick={() => navigate(`/workflow/new`)}>
            Create
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" className="ml-auto">
                Columns <ChevronDown className="ml-2 h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {table
                .getAllColumns()
                .filter((column) => column.getCanHide())
                .map((column) => {
                  return (
                    <DropdownMenuCheckboxItem
                      key={column.id}
                      className="capitalize"
                      checked={column.getIsVisible()}
                      onCheckedChange={(value) =>
                        column.toggleVisibility(!!value)
                      }
                    >
                      {column.id}
                    </DropdownMenuCheckboxItem>
                  );
                })}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => {
                  return (
                    <TableHead key={header.id} className="bg-muted">
                      {header.isPlaceholder
                        ? null
                        : flexRender(
                            header.column.columnDef.header,
                            header.getContext()
                          )}
                    </TableHead>
                  );
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow
                  key={row.id}
                  data-state={row.getIsSelected() && "selected"}
                >
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(
                        cell.column.columnDef.cell,
                        cell.getContext()
                      )}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-24 text-center"
                >
                  No results.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      <div className="flex items-center justify-end space-x-2 py-4">
        <div className="flex-1 text-sm text-muted-foreground"></div>
        <div className="space-x-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => table.previousPage()}
            disabled={!table.getCanPreviousPage()}
          >
            Previous
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => table.nextPage()}
            disabled={!table.getCanNextPage()}
          >
            Next
          </Button>
        </div>
      </div>
    </div>
  );
}
