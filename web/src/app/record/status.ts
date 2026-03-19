export function getStatusBadgeClassName(status: string) {
  switch (status) {
    case "success":
      return "border-transparent bg-green-500 text-white hover:bg-green-600";
    case "running":
      return "border-blue-200 bg-blue-500 text-white hover:bg-blue-600";
    case "failed":
      return "border-red-200 bg-red-500 text-white hover:bg-red-600";
    case "created":
    case "pending":
    case "skipped":
    default:
      return "border-amber-200 bg-amber-400 text-amber-950 hover:bg-amber-500";
  }
}
