import type { Route } from "../types/transit";

interface RouteSelectorProps {
  routes: Route[];
  selectedRoute: string;
  onSelectRoute: (routeId: string) => void;
}

export function RouteSelector({
  routes,
  selectedRoute,
  onSelectRoute
}: RouteSelectorProps) {
  return (
    <label className="route-selector">
      <span>Route</span>
      <select
        value={selectedRoute}
        onChange={(event) => onSelectRoute(event.target.value)}
      >
        {routes.map((route) => (
          <option key={route.id} value={route.id}>
            {route.shortName} {route.longName}
          </option>
        ))}
      </select>
    </label>
  );
}
