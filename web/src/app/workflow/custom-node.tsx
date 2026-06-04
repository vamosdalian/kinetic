import React, { memo } from "react";
import { BaseNode, BaseNodeContent } from "@/components/base-node";
import {
  Trash2,
  SquareTerminal,
  Globe,
  GitBranch,
  Repeat2,
  HelpCircle,
} from "lucide-react";
import { Handle, Position, type NodeProps, useReactFlow } from "@xyflow/react";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "@/components/ui/popover";
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface NodeData {
  name: string;
  type: string;
}

const handleStyle = {
  width: "8px",
  height: "8px",
  background: "#F1F5F9",
  border: "1px solid #CCD6E3",
  borderRadius: "50%",
  zIndex: 10,
  pointerEvents: "auto" as const,
};

function TypeIcon({ type }: { type: string }) {
  switch (type) {
    case "shell":
      return <SquareTerminal className="w-3 h-3 flex-shrink-0" />;
    case "http":
      return <Globe className="w-3 h-3 flex-shrink-0" />;
    case "condition":
      return <GitBranch className="w-3 h-3 flex-shrink-0" />;
    case "for":
      return <Repeat2 className="w-3 h-3 flex-shrink-0" />;
    default:
      return <HelpCircle className="w-3 h-3 flex-shrink-0" />;
  }
}

function SourceHandles({ type }: { type: string }) {
  if (type === "condition" || type === "for") {
    const leftHandle = type === "for" ? "body" : "true";
    const rightHandle = type === "for" ? "done" : "false";
    const leftLabel = type === "for" ? "Loop" : "True";
    const rightLabel = type === "for" ? "Done" : "False";

    return (
      <>
        <div
          className="pointer-events-none absolute -bottom-5 -translate-x-1/2 whitespace-nowrap text-[9px] font-medium text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100"
          style={{ left: "38%" }}
        >
          {leftLabel}
        </div>
        <div
          className="pointer-events-none absolute -bottom-5 -translate-x-1/2 whitespace-nowrap text-[9px] font-medium text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100"
          style={{ left: "66%" }}
        >
          {rightLabel}
        </div>
        <Handle
          type="source"
          id={leftHandle}
          position={Position.Bottom}
          style={{ ...handleStyle, left: "38%" }}
        />
        <Handle
          type="source"
          id={rightHandle}
          position={Position.Bottom}
          style={{ ...handleStyle, left: "66%" }}
        />
      </>
    );
  }

  return <Handle type="source" position={Position.Bottom} style={handleStyle} />;
}

export const ActionBarNodeDemo = memo((props: NodeProps) => {
  const { deleteElements } = useReactFlow();
  const { selected } = props;
  const data = React.useMemo(
    () => props.data as unknown as NodeData,
    [props.data]
  );

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation();
    deleteElements({ nodes: [{ id: props.id }] });
  };

  return (
    <BaseNode
      className={cn(
        "group relative rounded-xs w-48 h-8 transition-all",
        selected ? "ring ring-blue-500" : ""
      )}
    >
      <BaseNodeContent className="text-xs p-2 flex items-center justify-between w-full h-full">
        <div className="flex items-center gap-2 min-w-0 w-full">
          <TypeIcon type={data.type} />

          <Tooltip delayDuration={1000}>
            <TooltipTrigger asChild>
              <span className="truncate flex-grow min-w-0 font-medium text-[10px]">
                {data.name}
              </span>
            </TooltipTrigger>
            <TooltipContent>
              <p>{data.name}</p>
            </TooltipContent>
          </Tooltip>

          <div className="flex items-center gap-2 flex-shrink-0">
            <div className="w-4 h-4 flex items-center justify-center rounded hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer transition-colors">
              <Popover>
                <PopoverTrigger>
                  <Trash2 className="w-3 h-3" />
                </PopoverTrigger>
                <PopoverContent className="w-48">
                  <div className="space-y-3">
                    <p className="text-sm">Delete this task?</p>
                    <div className="flex justify-between">
                      <Button variant="outline" size="sm">
                        Cancel
                      </Button>
                      <Button
                        variant="default"
                        size="sm"
                        onClick={handleDelete}
                      >
                        Confirm
                      </Button>
                    </div>
                  </div>
                </PopoverContent>
              </Popover>
            </div>
          </div>
        </div>
      </BaseNodeContent>

      <SourceHandles type={data.type} />
      <Handle type="target" position={Position.Top} style={handleStyle} />
    </BaseNode>
  );
});
