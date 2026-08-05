import type { Vehicle } from "../types/transit";
import { formatAge, formatCoordinate, formatSpeed, statusLabel } from "../utils/format";
import { statusClass } from "../utils/status";

interface VehicleTableProps {
  vehicles: Vehicle[];
  selectedVehicleId: string | null;
  onSelectVehicle: (vehicleId: string) => void;
}

export function VehicleTable({
  vehicles,
  selectedVehicleId,
  onSelectVehicle
}: VehicleTableProps) {
  return (
    <section className="table-panel" aria-label="Vehicle table">
      <div className="panel-heading">
        <h2>Vehicles</h2>
        <span>
          {selectedVehicleId ? `Tracking ${selectedVehicleId}` : `${vehicles.length} tracked`}
        </span>
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
            {vehicles.map((vehicle) => {
              const isSelected = vehicle.vehicleId === selectedVehicleId;

              return (
                <tr
                  className={`vehicle-row${isSelected ? " is-tracked" : ""}`}
                  key={vehicle.vehicleId}
                  onClick={() => onSelectVehicle(vehicle.vehicleId)}
                >
                  <td>
                    <button
                      aria-pressed={isSelected}
                      className="vehicle-select-button"
                      onClick={(event) => {
                        event.stopPropagation();
                        onSelectVehicle(vehicle.vehicleId);
                      }}
                      type="button"
                    >
                      {vehicle.vehicleId}
                    </button>
                  </td>
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
              );
            })}
          </tbody>
        </table>
      </div>
    </section>
  );
}
