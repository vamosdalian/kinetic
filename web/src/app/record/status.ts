export function getStatusBadgeClassName(status: string) {
  switch (status) {
    case "success":
      return "border-transparent bg-green-500 text-white hover:bg-green-500";
    case "assigned":
    case "running":
      return "border-transparent bg-blue-500 text-white hover:bg-blue-500";
    case "failed":
      return "border-transparent bg-red-500 text-white hover:bg-red-500";
    case "cancelled":
      return "border-transparent bg-slate-500 text-white hover:bg-slate-500";
    case "unknown":
      return "border-transparent bg-orange-400 text-orange-950 hover:bg-orange-400";
    case "created":
    case "pending":
    case "queued":
    case "skipped":
    default:
      return "border-transparent bg-amber-400 text-amber-950 hover:bg-amber-400";
  }
}

export function isTerminalRunStatus(status: string) {
  return ["success", "failed", "cancelled"].includes(status);
}

export function getRunRowClassName(status: string) {
  if (status === "failed") {
    return "bg-red-50/70 hover:bg-red-100/70 dark:bg-red-950/20 dark:hover:bg-red-950/30";
  }
  return undefined;
}
