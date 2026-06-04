export function formatTaskType(type?: string) {
  switch (type) {
    case "shell":
      return "Shell";
    case "http":
      return "HTTP";
    case "condition":
      return "Condition";
    case "for":
      return "For Loop";
    default:
      return type || "Unknown";
  }
}