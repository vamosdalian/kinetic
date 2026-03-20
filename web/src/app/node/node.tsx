import * as React from "react";
import { type ColumnDef } from "@tanstack/react-table";
import { LoaderCircle, Plus, RefreshCw, X } from "lucide-react";

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
import { Input } from "@/components/ui/input";
import { apiClient } from "@/lib/api";
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
      const data = await apiClient<NodeItem[]>("/api/nodes");
      setNodes(data);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to load nodes");
    } finally {
      if (showLoader) {
        setLoading(false);
      }
    }
  }, []);

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

  const filteredNodes = React.useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) {
      return nodes;
    }

    return nodes.filter((node) => {
      const haystack = [
        node.node_id,
        node.name,
        node.ip,
        node.kind,
        node.status,
        ...node.tags.map((tag) => tag.tag),
      ]
        .join(" ")
        .toLowerCase();

      return haystack.includes(needle);
    });
  }, [nodes, query]);

  const columns = React.useMemo<ColumnDef<NodeItem>[]>(
    () => [
      {
        accessorKey: "name",
        header: "Node Name",
        cell: ({ row }) => (
          <div className="min-w-[160px]">
            <div className="font-medium">{row.original.name}</div>
            <div className="font-mono text-xs text-muted-foreground">
              {row.original.node_id}
            </div>
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
        cell: ({ row }) => (
          <Button
            variant="outline"
            size="sm"
            onClick={() => openAddTagDialog(row.original)}
          >
            <Plus className="h-4 w-4" />
            Add Tag
          </Button>
        ),
      },
    ],
    [deletingKey, handleDeleteTag, openAddTagDialog]
  );

  if (loading) {
    return (
      <div className="flex flex-1 flex-col min-h-0">
        <SiteHeader breadcrumbs={[{ label: "Node", href: null }]} />
        <div className="flex flex-1 items-center justify-center">
          <LoaderCircle className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-1 flex-col min-h-0">
      <SiteHeader breadcrumbs={[{ label: "Node", href: null }]} />
      <CommonTable
        columns={columns}
        data={filteredNodes}
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
