import * as React from "react";
import { type ColumnDef } from "@tanstack/react-table";
import { BookmarkPlus, LoaderCircle, MoreHorizontal, Plus, RefreshCw, X } from "lucide-react";

import { CommonTable } from "@/components/common-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { apiClient, apiClientFull } from "@/lib/api";
import { getNodeStatusBadgeClassName } from "@/lib/node-status";
import { SiteHeader } from "@/components/site-header";
import { cn } from "@/lib/utils";
import { toast } from "sonner";

interface NodeTag {
  tag: string;
  system_managed: boolean;
}

interface NodeItem {
  node_id: string;
  name: string;
  ip: string;
  kind: string;
  status: string;
  max_concurrency: number;
  running_count: number;
  last_heartbeat_at: string;
  last_stream_at: string;
  tags: NodeTag[];
}

export function Node() {
  const [nodes, setNodes] = React.useState<NodeItem[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [pagination, setPagination] = React.useState({
    pageIndex: 0,
    pageSize: 10,
  });
  const [pageCount, setPageCount] = React.useState(-1);
  const [query, setQuery] = React.useState("");
  const [selectedNode, setSelectedNode] = React.useState<NodeItem | null>(null);
  const [newTag, setNewTag] = React.useState("");
  const [submitting, setSubmitting] = React.useState(false);
  const [deletingKey, setDeletingKey] = React.useState<string | null>(null);

  const fetchNodes = React.useCallback(async (showLoader = true) => {
    if (showLoader) {
      setLoading(true);
    }

    try {
      const params = new URLSearchParams({
        page: String(pagination.pageIndex + 1),
        pageSize: String(pagination.pageSize),
      });
      if (query.trim()) {
        params.set("query", query.trim());
      }
      const response = await apiClientFull<NodeItem[]>(`/api/nodes?${params.toString()}`);
      setNodes(response.data);
      setPageCount(response.meta?.totalPages ?? -1);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to load nodes");
    } finally {
      if (showLoader) {
        setLoading(false);
      }
    }
  }, [pagination.pageIndex, pagination.pageSize, query]);

  React.useEffect(() => {
    void fetchNodes();
  }, [fetchNodes]);

  const openAddTagDialog = React.useCallback((node: NodeItem) => {
    setSelectedNode(node);
    setNewTag("");
  }, []);

  const closeAddTagDialog = React.useCallback(() => {
    setSelectedNode(null);
    setNewTag("");
  }, []);

  const handleAddTag = React.useCallback(async () => {
    if (!selectedNode) {
      return;
    }

    const tag = newTag.trim();
    if (!tag) {
      toast.error("Tag is required");
      return;
    }

    setSubmitting(true);
    try {
      await apiClient(`/api/nodes/${selectedNode.node_id}/tags`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ tag }),
      });
      toast.success("Tag added");
      closeAddTagDialog();
      await fetchNodes(false);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to add tag");
    } finally {
      setSubmitting(false);
    }
  }, [closeAddTagDialog, fetchNodes, newTag, selectedNode]);

  const handleDeleteTag = React.useCallback(
    async (nodeID: string, tag: string) => {
      const key = `${nodeID}:${tag}`;
      setDeletingKey(key);
      try {
        await apiClient(`/api/nodes/${nodeID}/tags/${encodeURIComponent(tag)}`, {
          method: "DELETE",
        });
        toast.success("Tag removed");
        await fetchNodes(false);
      } catch (error) {
        toast.error(error instanceof Error ? error.message : "Failed to remove tag");
      } finally {
        setDeletingKey(null);
      }
    },
    [fetchNodes]
  );

  React.useEffect(() => {
    setPagination((current) =>
      current.pageIndex === 0 ? current : { ...current, pageIndex: 0 }
    );
  }, [query]);

  const columns = React.useMemo<ColumnDef<NodeItem>[]>(
    () => [
      {
        accessorKey: "node_id",
        header: "Node ID",
        cell: ({ row }) => <div className="font-mono text-xs">{row.original.node_id}</div>,
      },
      {
        accessorKey: "name",
        header: "Node Name",
        cell: ({ row }) => (
          <div className="min-w-[160px]">
            <div className="font-medium">{row.original.name}</div>
          </div>
        ),
      },
      {
        accessorKey: "ip",
        header: "IP",
        cell: ({ row }) => (
          <div className="font-mono text-xs">{row.original.ip || "-"}</div>
        ),
      },
      {
        accessorKey: "status",
        header: "Status",
        cell: ({ row }) => (
          <Badge
            variant="outline"
            className={cn("capitalize", getNodeStatusBadgeClassName(row.original.status))}
          >
            {row.original.status}
          </Badge>
        ),
      },
      {
        accessorKey: "kind",
        header: "Kind",
        cell: ({ row }) => <div className="capitalize">{row.original.kind}</div>,
      },
      {
        accessorKey: "max_concurrency",
        header: "Max Concurrency",
      },
      {
        accessorKey: "running_count",
        header: "Running",
      },
      {
        accessorKey: "last_heartbeat_at",
        header: "Last Heartbeat",
        cell: ({ row }) => <div>{row.original.last_heartbeat_at || "-"}</div>,
      },
      {
        accessorKey: "tags",
        header: "Tags",
        cell: ({ row }) => (
          <div className="flex max-w-[420px] flex-wrap gap-2">
            {row.original.tags.length === 0 ? (
              <span className="text-sm text-muted-foreground">No tags</span>
            ) : (
              row.original.tags.map((tag) => {
                const deleting = deletingKey === `${row.original.node_id}:${tag.tag}`;
                return (
                  <Badge
                    key={tag.tag}
                    variant={tag.system_managed ? "secondary" : "outline"}
                    className="gap-1 pr-1"
                  >
                    <span>{tag.tag}</span>
                    {tag.system_managed ? (
                      <span className="text-[10px] text-muted-foreground">system</span>
                    ) : (
                      <button
                        type="button"
                        className="rounded-xs p-0.5 text-muted-foreground transition hover:bg-muted hover:text-foreground"
                        onClick={() => void handleDeleteTag(row.original.node_id, tag.tag)}
                        disabled={deleting}
                        aria-label={`Delete ${tag.tag}`}
                      >
                        {deleting ? (
                          <LoaderCircle className="h-3 w-3 animate-spin" />
                        ) : (
                          <X className="h-3 w-3" />
                        )}
                      </button>
                    )}
                  </Badge>
                );
              })
            )}
          </div>
        ),
      },
      {
        id: "actions",
        header: "Action",
        enableHiding: false,
        cell: ({ row }) => {
          const node = row.original

          return (
            <div className="flex items-center space-x-1">
              <Button
                variant="ghost"
                className="h-8 w-8 p-0"
                onClick={() => openAddTagDialog(node)}
              >
                <span className="sr-only">Add tag</span>
                <BookmarkPlus className="h-4 w-4" />
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
                  <DropdownMenuItem onClick={() => navigator.clipboard.writeText(node.node_id)}>
                    Copy Node ID
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          )
        },
      },
    ],
    [deletingKey, handleDeleteTag, openAddTagDialog]
  );

  return (
    <div className="flex flex-1 flex-col min-h-0">
      <SiteHeader breadcrumbs={[{ label: "Node", href: null }]} />
      <CommonTable
        columns={columns}
        data={nodes}
        loading={loading}
        initialColumnVisibility={{ node_id: false }}
        manualPagination={true}
        pageCount={pageCount}
        pagination={pagination}
        onPaginationChange={setPagination}
        renderToolbarActions={() => (
          <>
            <Input
              className="w-[280px]"
              placeholder="Search node, IP, tag..."
              value={query}
              onChange={(event) => setQuery(event.target.value)}
            />
            <Button variant="outline" onClick={() => void fetchNodes(false)}>
              <RefreshCw className="h-4 w-4" />
              Refresh
            </Button>
          </>
        )}
      />

      <Dialog open={!!selectedNode} onOpenChange={(open) => !open && closeAddTagDialog()}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add Tag</DialogTitle>
            <DialogDescription>
              {selectedNode
                ? `Add a custom routing tag to ${selectedNode.name}.`
                : "Add a custom routing tag."}
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-2">
            <Input
              placeholder="e.g. gpu, cn-shanghai, team-a"
              value={newTag}
              onChange={(event) => setNewTag(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter") {
                  event.preventDefault();
                  void handleAddTag();
                }
              }}
            />
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={closeAddTagDialog} disabled={submitting}>
              Cancel
            </Button>
            <Button onClick={() => void handleAddTag()} disabled={submitting}>
              {submitting ? (
                <LoaderCircle className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Plus className="mr-2 h-4 w-4" />
              )}
              Add Tag
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
