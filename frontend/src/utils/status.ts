import type { VehicleStatus } from "../types/transit";

export function statusClass(status: VehicleStatus): string {
  switch (status) {
    case "ACTIVE":
      return "status-active";
    case "STALE":
      return "status-stale";
    case "BUNCHING_RISK":
      return "status-bunching";
    case "DELAYED":
      return "status-delayed";
    default:
      return "status-unknown";
  }
}

export function statusColor(status: VehicleStatus): string {
  switch (status) {
    case "ACTIVE":
      return "#15803d";
    case "STALE":
      return "#64748b";
    case "BUNCHING_RISK":
      return "#dc2626";
    case "DELAYED":
      return "#b45309";
    default:
      return "#334155";
  }
}
