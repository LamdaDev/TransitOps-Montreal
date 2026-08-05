import { useEffect } from "react";
import { CircleMarker, MapContainer, Polyline, Popup, TileLayer, useMap } from "react-leaflet";
import type { RouteShapePoint, Vehicle, VehicleTrail, VehicleTrailPoint } from "../types/transit";
import { formatAge, formatSpeed, statusLabel } from "../utils/format";
import { statusColor } from "../utils/status";

interface VehicleMapProps {
  vehicles: Vehicle[];
  selectedVehicleId: string | null;
  routeShape: RouteShapePoint[];
  routeColor?: string | null;
  vehicleTrails: VehicleTrail[];
}

const montrealCenter: [number, number] = [45.5089, -73.5617];
const defaultRouteColor = "#0f766e";
const trailBreakDistanceDegrees = 0.012;

type MapPosition = [number, number];

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

function splitTrail(points: VehicleTrailPoint[]): MapPosition[][] {
  const segments: MapPosition[][] = [];
  let segment: MapPosition[] = [];

  for (const point of points) {
    const position: MapPosition = [point.latitude, point.longitude];
    const previousPosition = segment[segment.length - 1];
    const hasLargeJump =
      previousPosition &&
      Math.hypot(
        previousPosition[0] - position[0],
        previousPosition[1] - position[1]
      ) > trailBreakDistanceDegrees;

    if (hasLargeJump) {
      if (segment.length > 1) {
        segments.push(segment);
      }
      segment = [];
    }

    segment.push(position);
  }

  if (segment.length > 1) {
    segments.push(segment);
  }

  return segments;
}

export function VehicleMap({
  vehicles,
  selectedVehicleId,
  routeShape,
  routeColor,
  vehicleTrails
}: VehicleMapProps) {
  const selectedVehicle = vehicles.find((vehicle) => vehicle.vehicleId === selectedVehicleId);
  const shapePositions = routeShape.map(
    (point): MapPosition => [point.latitude, point.longitude]
  );
  const routeLineColor = routeColor ?? defaultRouteColor;

  return (
    <section className="map-panel" aria-label="Live vehicle map">
      <MapContainer center={montrealCenter} zoom={12} scrollWheelZoom className="map">
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />
        <FollowVehicle vehicle={selectedVehicle} />
        {shapePositions.length > 1 ? (
          <Polyline
            pathOptions={{
              color: routeLineColor,
              lineCap: "round",
              lineJoin: "round",
              opacity: 0.72,
              weight: 5
            }}
            positions={shapePositions}
          />
        ) : null}
        {vehicleTrails.flatMap((trail) => {
          const latestVehicle = vehicles.find((vehicle) => vehicle.vehicleId === trail.vehicleId);
          const isSelected = trail.vehicleId === selectedVehicleId;
          const trailColor = isSelected
            ? routeLineColor
            : statusColor(latestVehicle?.status ?? "UNKNOWN");

          return splitTrail(trail.points).map((segment, index) => (
            <Polyline
              key={`${trail.vehicleId}-${index}`}
              pathOptions={{
                color: trailColor,
                dashArray: isSelected ? undefined : "4 7",
                lineCap: "round",
                lineJoin: "round",
                opacity: isSelected ? 0.94 : 0.42,
                weight: isSelected ? 5 : 3
              }}
              positions={segment}
            />
          ));
        })}
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
