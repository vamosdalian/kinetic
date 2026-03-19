import React, { memo } from "react";
import { BaseNode, BaseNodeContent } from "@/components/base-node";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { 
  SquareTerminal, 
  Globe, 
  Code, 
  GitBranch, 
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

interface NodeData {
  name: string;
  type: string;
  status: string;
  exit_code?: number;
}

const TypeIcon = ({ type }: { type: string }) => {
  switch (type) {
    case "shell":
      return <SquareTerminal className="w-3 h-3 flex-shrink-0" />;
    case "http":
      return <Globe className="w-3 h-3 flex-shrink-0" />;
    case "python":
      return <Code className="w-3 h-3 flex-shrink-0" />;
    case "condition":
      return <GitBranch className="w-3 h-3 flex-shrink-0" />;
    default:
      return <HelpCircle className="w-3 h-3 flex-shrink-0" />;
  }
};

export const RunNode = memo((props: NodeProps) => {
  const { selected } = props;
  const data = React.useMemo(
    () => props.data as unknown as NodeData,
    [props.data]
  );

  return (
    <BaseNode className={`relative rounded-xs w-48 h-8 transition-all ${
        selected ? "ring ring-blue-500" : ""
      }`}>
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
               <div className="text-xs text-muted-foreground space-y-1 mt-1">
                 <p>Type: {data.type}</p>
                 <p>Status: {data.status} {data.status === 'failed' && data.exit_code !== undefined ? `(${data.exit_code})` : ''}</p>
               </div>
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

      <Handle
        type="source"
        position={Position.Bottom}
        style={{
          width: "8px",
          height: "8px",
          background: "#F1F5F9",
          border: "1px solid #CCD6E3",
          borderRadius: "50%",
          zIndex: 10,
          pointerEvents: "auto",
        }}
      />
      <Handle
        type="target"
        position={Position.Top}
        style={{
          width: "8px",
          height: "8px",
          background: "#F1F5F9",
          border: "1px solid #CCD6E3",
          borderRadius: "50%",
          zIndex: 10,
          pointerEvents: "auto",
        }}
      />
    </BaseNode>
  );
});
