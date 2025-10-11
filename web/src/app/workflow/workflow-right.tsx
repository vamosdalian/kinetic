// import * as React from "react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Taskform } from "./task-form";

export function WorkflowRight() {
  return (
    <ScrollArea className="h-[calc(100vh-var(--header-height))]">
      {/* <div className="h-[1000px] bg-red-100">hello</div> */}
      <Taskform></Taskform>
    </ScrollArea>
  );
}
