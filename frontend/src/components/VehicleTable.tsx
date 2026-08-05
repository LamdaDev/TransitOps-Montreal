import type { Vehicle } from "../types/transit";
import { formatAge, formatCoordinate, formatSpeed, statusLabel } from "../utils/format";
import { statusClass } from "../utils/status";

interface VehicleTableProps {
  vehicles: Vehicle[];
}

export function VehicleTable({ vehicles }: VehicleTableProps) {
  return (
    <section className="table-panel" aria-label="Vehicle table">
      <div className="panel-heading">
        <h2>Vehicles</h2>
        <span>{vehicles.length} tracked</span>
      </div>
      <div className="table-scroll">
        <table>
          <thead>
            <tr>
              <th>Vehicle ID</th>
              <th>Route</th>
              <th>Latitude</th>
              <th>Longitude</th>
              <th>Speed</th>
              <th>Last seen</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {vehicles.map((vehicle) => (
              <tr key={vehicle.vehicleId}>
                <td>{vehicle.vehicleId}</td>
                <td>{vehicle.routeId}</td>
                <td>{formatCoordinate(vehicle.latitude)}</td>
                <td>{formatCoordinate(vehicle.longitude)}</td>
                <td>{formatSpeed(vehicle.speed)}</td>
                <td>{formatAge(vehicle.timestamp)}</td>
                <td>
                  <span className={`status-pill ${statusClass(vehicle.status)}`}>
                    {statusLabel(vehicle.status)}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
