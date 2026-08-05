# TransitOps Montréal

A full-stack operations dashboard that turns STM-style transit vehicle data into route reliability insights.

TransitOps Montréal focuses on the operator view of public transit: vehicle coverage, route spacing, bus bunching, stale telemetry, and route health. It is built as a weekend-sized MVP with a mock GTFS-Realtime-style provider so the project works without STM API access.

## Why I Built This

I use Montréal transit and noticed most apps focus on passenger arrival times. I wanted to explore the operations side: bus spacing, bunching, stale vehicles, route health, and how raw mobility data can become actionable insights.

## Tech Stack

- Go backend
- GraphQL API
- PostgreSQL snapshot storage
- React + TypeScript frontend
- Leaflet + OpenStreetMap map
- Docker Compose local environment

## Features

- Route selector for 24 Sherbrooke, 55 Saint-Laurent, 80 Avenue du Parc, 165 Côte-des-Neiges, and 470 Express Pierrefonds
- Live map of mock STM vehicle positions around Montréal
- Fleet summary cards for active vehicles, stale vehicles, bunching events, largest headway gap, last update time, and route health
- Vehicle table with location, speed, last seen time, and computed operational status
- Operational insight panel for bunching, long gaps, stale telemetry, and healthy route states
- Go ingestion service that stores vehicle snapshots over time in PostgreSQL
- Mock provider designed to be replaced later with an STM GTFS-Realtime provider

## Architecture

Mock/STM data provider -> Go ingestion service -> PostgreSQL -> GraphQL API -> React dashboard

The backend stores every vehicle snapshot in `vehicle_snapshots` instead of overwriting the latest state. The dashboard currently reads the latest snapshot per vehicle, while the database remains replay-ready for future historical analysis.

## GraphQL API

The backend exposes a single GraphQL transport endpoint:

```text
http://localhost:8080/graphql
```

Core operations:

```graphql
query Dashboard($routeId: String!) {
  routes {
    id
    shortName
    longName
    color
  }
  vehicles(routeId: $routeId) {
    vehicleId
    routeId
    latitude
    longitude
    speed
    timestamp
    status
  }
  routeMetrics(routeId: $routeId) {
    activeVehicleCount
    staleVehicleCount
    bunchingEventCount
    largestHeadwayGapMinutes
    averageSpacingMinutes
    healthStatus
    lastUpdated
  }
  routeInsights(routeId: $routeId)
}

mutation {
  ingestMock {
    insertedCount
    timestamp
    source
  }
}
```

## How To Run Locally

### Option 1: Docker Compose

```bash
docker compose up --build
```

Then open:

```text
http://localhost:5173
```

GraphiQL is available at:

```text
http://localhost:8080/graphql
```

### Option 2: Run Services Separately

Start PostgreSQL:

```bash
docker compose up postgres
```

Run the backend:

```bash
cd backend
go run ./cmd/server
```

Run the frontend:

```bash
cd frontend
npm install
npm run dev
```

## Environment Variables

```text
DATABASE_URL=postgres://transitops:transitops@localhost:5432/transitops?sslmode=disable
PORT=8080
DATA_PROVIDER=mock
INGEST_INTERVAL_SECONDS=10
STM_API_KEY=
STM_GTFS_RT_VEHICLE_POSITIONS_URL=
```

The app works with `DATA_PROVIDER=mock` and does not require an external API key.

## MVP Metrics Logic

- Stale vehicle: latest snapshot is older than 2 minutes
- Bunching risk: two vehicles on the same route are within 400 meters by Haversine distance
- Largest gap: latest vehicles sorted by mock route progress from `0.0` to `1.0`
- Route health: `HEALTHY`, `WATCH`, or `DEGRADED` based on bunching, stale telemetry, and long gaps

## Future Improvements

- Real STM GTFS-Realtime integration
- Replay mode for vehicle snapshots
- Historical route reliability reports
- Schedule adherence using static GTFS
- GraphQL subscriptions for live updates
- Alert acknowledgement workflow
- Deployment to Render, Fly.io, or Vercel

## Resume Bullets

- Built a full-stack transit operations dashboard using Go, React, TypeScript, PostgreSQL, GraphQL, and GTFS-Realtime-style data to analyze STM bus movement, route spacing, and route health.
- Implemented a Go ingestion service that stores vehicle snapshots over time and computes operational metrics such as active vehicles, stale vehicles, bunching risk, and largest route gap.
- Developed a React dashboard with map-based fleet visualization, route filters, summary metrics, and operational insight generation to turn raw vehicle data into actionable reliability signals.
