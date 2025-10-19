import React, { memo } from "react";
import { BaseNode, BaseNodeContent } from "@/components/base-node";
import { Trash2, Play, SquareTerminal } from "lucide-react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { useReactFlow } from "@xyflow/react";
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

interface NodeData {
  name: string;
}

export const ActionBarNodeDemo = memo((props: NodeProps) => {
  const { deleteElements } = useReactFlow();
  const { id,selected } = props;
  const data = React.useMemo(
    () => props.data as unknown as NodeData,
    [props.data]
  );

  React.useEffect(() => {
    console.log("Node props:", props);
  }, [props]);

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (props.id) {
      deleteElements({ nodes: [{ id: props.id }] });
    }
  };

  const showProps = () => {
    console.log("Node props:", props);
  }

  return (
    <BaseNode className={`relative rounded-xs w-48 h-8 transition-all ${
        selected ? "ring ring-blue-500" : ""
      }`} onClick={showProps}>
      <BaseNodeContent className="text-xs p-2 flex items-center justify-between w-full">
        <div className="flex items-center gap-2 min-w-0 w-full">
          <SquareTerminal className="w-3 h-3 flex-shrink-0" />

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
            <div
              className="w-4 h-4 flex items-center justify-center rounded hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer transition-colors"
              onClick={(e) => {
                e.stopPropagation();
                console.log("Play clicked,id", id);
              }}
            >
              <Play className="w-3 h-3" />
            </div>
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
