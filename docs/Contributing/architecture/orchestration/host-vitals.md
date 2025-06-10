# Host vitals architecture

This document provides an overview of Fleet's Host Vitals architecture.

## Introduction

Host Vitals in Fleet provide real-time and historical information about the health and status of devices, including CPU usage, memory usage, disk usage, and uptime. This document provides insights into the design decisions, system components, and interactions specific to the Host Vitals functionality.

## Architecture overview

The Host Vitals architecture enables the collection, processing, and visualization of device health metrics across a fleet of devices. It leverages osquery's capabilities to collect system metrics and Fleet's infrastructure to process and display them.

## Key components

- **Metrics Collection**: The process of collecting system metrics from devices.
- **Metrics Processing**: The processing of collected metrics for storage and analysis.
- **Metrics Storage**: The storage of metrics for historical analysis.
- **Metrics Visualization**: The display of metrics in the Fleet UI.

## Architecture diagram

```
[Placeholder for Host Vitals Architecture Diagram]
```

## Metrics collection flow

### 1 - Agent collects metrics

```
osquery agent -> Server
```

1. osquery agent collects system metrics using osquery tables.
2. osquery agent sends the metrics to the Fleet server.

### 2 - Server processes and stores metrics

```
Server -> DB
```

1. Server processes the received metrics.
2. Server stores the metrics in the database.

### 3 - UI displays metrics

```
UI -> Server -> DB
```

1. UI requests metrics from the server.
2. Server retrieves metrics from the database.
3. Server returns metrics to the UI.
4. UI displays the metrics.

## Host vitals collected

See the [host details API](https://fleetdm.com/docs/rest-api/rest-api#get-host) documentation for details on collected vitals.

## Performance considerations

Host Vitals collection can impact device and server performance, especially for large fleets. The following considerations should be taken into account:

- **Collection Frequency**: More frequent collection can impact device performance.
- **Metric Count**: Collecting more metrics can impact device and server performance.
- **Fleet Size**: Collecting metrics from a large number of devices can impact server performance.

## Related resources

- [Host details API documentation](https://fleetdm.com/docs/rest-api/rest-api#get-host)
- [Orchestration Product Group Documentation](../../product-groups/orchestration/) - Documentation for the Orchestration product group
- [Orchestration Development Guides](../../guides/orchestration/) - Guides for Orchestration development
- [Understanding Host Vitals](../../product-groups/orchestration/understanding-host-vitals.md) - Detailed documentation on host vitals