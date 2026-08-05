export function formatDateTime(value: string): string {
  if (!value) {
    return "No data";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(undefined, {
    hour: "numeric",
    minute: "2-digit",
    second: "2-digit"
  }).format(date);
}

export function formatShortTime(value: string): string {
  if (!value) {
    return "No data";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(undefined, {
    hour: "numeric",
    minute: "2-digit"
  }).format(date);
}

export function formatAge(value: string): string {
  if (!value) {
    return "No data";
  }

  const date = new Date(value);
  const diffSeconds = Math.max(0, Math.round((Date.now() - date.getTime()) / 1000));
  if (diffSeconds < 60) {
    return `${diffSeconds}s ago`;
  }

  const minutes = Math.round(diffSeconds / 60);
  return `${minutes}m ago`;
}

export function formatSpeed(speed?: number | null): string {
  if (speed === null || speed === undefined) {
    return "N/A";
  }

  return `${speed.toFixed(1)} km/h`;
}

export function formatCoordinate(value: number): string {
  return value.toFixed(5);
}

export function statusLabel(status: string): string {
  return status.replaceAll("_", " ").toLowerCase();
}
