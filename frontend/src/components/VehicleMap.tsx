import { useEffect } from "react";
import { CircleMarker, MapContainer, Popup, TileLayer, useMap } from "react-leaflet";
import type { Vehicle } from "../types/transit";
import { formatAge, formatSpeed, statusLabel } from "../utils/format";
import { statusColor } from "../utils/status";

interface VehicleMapProps {
  vehicles: Vehicle[];
  selectedVehicleId: string | null;
}

const montrealCenter: [number, number] = [45.5089, -73.5617];

interface FollowVehicleProps {
  vehicle?: Vehicle;
}

function FollowVehicle({ vehicle }: FollowVehicleProps) {
  const map = useMap();

  useEffect(() => {
    if (!vehicle) {
      return;
    }

    map.flyTo([vehicle.latitude, vehicle.longitude], map.getZoom(), {
      animate: true,
      duration: 0.7
    });
  }, [map, vehicle?.latitude, vehicle?.longitude, vehicle?.vehicleId]);

  return null;
}

export function VehicleMap({ vehicles, selectedVehicleId }: VehicleMapProps) {
  const selectedVehicle = vehicles.find((vehicle) => vehicle.vehicleId === selectedVehicleId);

  return (
    <section className="map-panel" aria-label="Live vehicle map">
      <MapContainer center={montrealCenter} zoom={12} scrollWheelZoom className="map">
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />
        <FollowVehicle vehicle={selectedVehicle} />
        {vehicles.map((vehicle) => {
          const isSelected = vehicle.vehicleId === selectedVehicleId;

          return (
            <CircleMarker
              center={[vehicle.latitude, vehicle.longitude]}
              color="#ffffff"
              fillColor={statusColor(vehicle.status)}
              fillOpacity={0.92}
              key={vehicle.vehicleId}
              pathOptions={{ weight: isSelected ? 4 : 2 }}
              radius={isSelected ? 12 : 9}
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
          );
        })}
      </MapContainer>
    </section>
  );
}
