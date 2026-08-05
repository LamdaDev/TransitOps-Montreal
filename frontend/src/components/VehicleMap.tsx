import { CircleMarker, MapContainer, Popup, TileLayer } from "react-leaflet";
import type { Vehicle } from "../types/transit";
import { formatAge, formatSpeed, statusLabel } from "../utils/format";
import { statusColor } from "../utils/status";

interface VehicleMapProps {
  vehicles: Vehicle[];
}

const montrealCenter: [number, number] = [45.5089, -73.5617];

export function VehicleMap({ vehicles }: VehicleMapProps) {
  return (
    <section className="map-panel" aria-label="Live vehicle map">
      <MapContainer center={montrealCenter} zoom={12} scrollWheelZoom className="map">
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />
        {vehicles.map((vehicle) => (
          <CircleMarker
            center={[vehicle.latitude, vehicle.longitude]}
            color="#ffffff"
            fillColor={statusColor(vehicle.status)}
            fillOpacity={0.92}
            key={vehicle.vehicleId}
            pathOptions={{ weight: 2 }}
            radius={9}
          >
            <Popup>
              <div className="map-popup">
                <strong>{vehicle.vehicleId}</strong>
                <span>Route {vehicle.routeId}</span>
                <span>{formatSpeed(vehicle.speed)}</span>
                <span>{formatAge(vehicle.timestamp)}</span>
                <span>{statusLabel(vehicle.status)}</span>
              </div>
            </Popup>
          </CircleMarker>
        ))}
      </MapContainer>
    </section>
  );
}
