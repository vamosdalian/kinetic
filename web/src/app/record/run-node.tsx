import React, { memo } from "react";
import { BaseNode, BaseNodeContent } from "@/components/base-node";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { 
  SquareTerminal, 
  Globe, 
  GitBranch, 
  Repeat2,
  // Circle,
  HelpCircle
} from "lucide-react";
import { Badge } from "@/components/ui/badge";

import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { getStatusBadgeClassName } from "./status";
import { cn } from "@/lib/utils";

interface NodeData {
  name: string;
  type: string;
  status: string;
  exit_code?: number;
  assigned_node_id?: string;
  effective_tag?: string;
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

const TypeIcon = ({ type }: { type: string }) => {
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
};

const SourceHandles = ({ type }: { type: string }) => {
  if (type === "condition" || type === "for") {
    const leftHandle = type === "for" ? "body" : "true";
    const rightHandle = type === "for" ? "done" : "false";

    return (
      <>
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
};

export const RunNode = memo((props: NodeProps) => {
  const { selected } = props;
  const data = React.useMemo(
    () => props.data as unknown as NodeData,
    [props.data]
  );

  return (
    <BaseNode
      className={cn(
        "relative rounded-xs w-48 h-8 transition-all",
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
          
          <Badge 
            variant="outline"
            className={`h-4 px-1 text-[9px] pointer-events-none capitalize ${getStatusBadgeClassName(data.status)}`}
          >
            {data.status}
          </Badge>
        </div>
      </BaseNodeContent>

      <SourceHandles type={data.type} />
      <Handle type="target" position={Position.Top} style={handleStyle} />
    </BaseNode>
  );
});
