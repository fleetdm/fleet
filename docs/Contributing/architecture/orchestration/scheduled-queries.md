# Scheduled queries architecture

This document provides an overview of Fleet's Scheduled Queries architecture.

## Introduction

Scheduled queries in Fleet allow users to configure queries that run on a regular schedule, providing ongoing visibility into device status, configuration, and security posture. This document provides insights into the design decisions, system components, and interactions specific to the Scheduled Queries functionality.

## Architecture overview

The Scheduled Queries architecture enables the configuration, distribution, and execution of queries on a schedule across a fleet of devices. It leverages osquery's scheduled query capabilities to execute SQL queries on devices and return results to the Fleet server.

## Key components

- **Query Configuration**: The definition of a query, including the SQL statement, schedule, and target devices.
- **Configuration Distribution**: The mechanism for distributing query configurations to devices.
- **Result Collection**: The process of collecting and processing query results.
- **Result Storage**: The storage of query results for analysis and alerting.

## Architecture diagram

```
[Placeholder for Scheduled Queries Architecture Diagram]
```

## Query execution flow

### 1 - Fleet user creates a scheduled query

```
Fleet User -> API Client (Frontend or Fleetctl) -> Server -> DB
```

1. Fleet user creates a scheduled query for a team or globally through the UI or API.
2. Server stores the query configuration in the database.

### 2 - Agent gets config file (with the scheduled query)

```
osquery agent -> Server -> DB
```

1. osquery agent requests the configuration file from the server.
2. Server merges team and global configurations.
3. Server returns the merged configuration to the agent.

### 3 - Agent returns results to be (optionally) logged

```
osquery agent -> Server -> Optional External Log
```

1. osquery agent runs the query according to the schedule.
2. osquery agent sends the results to the server.
3. Server optionally forwards the results to an external logging system.

## Configuration options

osquery agents have several configuration options that affect scheduled queries:

1. **Config TLS Refresh**: The frequency at which the agent pulls down the configuration file (typically 10 seconds).
2. **Logger TLS**: The frequency of sending query results (typically 10 seconds).

## Performance considerations

Scheduled queries can impact device performance, especially for complex queries or queries that run frequently. The following considerations should be taken into account:

- **Query Complexity**: Complex queries can consume significant CPU resources on devices.
- **Query Frequency**: Queries that run frequently can impact device performance.
- **Result Size**: Large result sets can consume significant memory and network bandwidth.

## Related resources

- [Orchestration Product Group Documentation](../../product-groups/orchestration/) - Documentation for the Orchestration product group
- [Orchestration Development Guides](../../guides/orchestration/) - Guides for Orchestration development