export function getNodeStatusBadgeClassName(status: string) {
  switch (status) {
    case "online":
      return "border-transparent bg-green-500 text-white hover:bg-green-500"
    case "offline":
    default:
      return "border-transparent bg-slate-500 text-white hover:bg-slate-500"
  }
}
