import React, { memo } from "react";
import { BaseNode, BaseNodeContent } from "@/components/base-node";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { CheckCircle2, XCircle, Clock, Loader2, CircleDashed } from "lucide-react";
// import { Badge } from "@/components/ui/badge";

import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface NodeData {
  name: string;
  type: string;
  status?: string;
  exit_code?: number;
}

const StatusIcon = ({ status }: { status?: string }) => {
  switch (status) {
    case "success":
      return <CheckCircle2 className="w-3 h-3 text-green-500 flex-shrink-0" />;
    case "failed":
      return <XCircle className="w-3 h-3 text-red-500 flex-shrink-0" />;
    case "running":
      return <Loader2 className="w-3 h-3 text-blue-500 animate-spin flex-shrink-0" />;
    case "pending":
    case "created":
      return <Clock className="w-3 h-3 text-yellow-500 flex-shrink-0" />;
    default:
      return <CircleDashed className="w-3 h-3 text-gray-400 flex-shrink-0" />;
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
      <BaseNodeContent className="text-xs p-2 flex items-center justify-between w-full">
        <div className="flex items-center gap-2 min-w-0 w-full">
          <StatusIcon status={data.status} />

           <Tooltip delayDuration={1000}>
            <TooltipTrigger asChild>
              <span className="truncate flex-grow min-w-0 font-medium text-[10px]">
                {data.name}
              </span>
            </TooltipTrigger>
            <TooltipContent>
               <p>{data.name}</p>
               <p className="text-xs text-muted-foreground">Type: {data.type}</p>
               <p className="text-xs text-muted-foreground">Status: {data.status} {data.status === 'failed' && data.exit_code !== undefined ? `(${data.exit_code})` : ''}</p>
            </TooltipContent>
          </Tooltip>
          
          {data.status === 'failed' && data.exit_code !== undefined && (
             <span className="text-[10px] text-red-500 font-mono flex-shrink-0">
               {data.exit_code}
             </span>
          )}
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
