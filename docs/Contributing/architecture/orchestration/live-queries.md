# Live Queries Architecture

This document provides an overview of Fleet's Live Queries architecture.

## Introduction

Live queries in Fleet allow users to execute ad-hoc queries against devices in real-time, providing immediate visibility into device status, configuration, and security posture. This document provides insights into the design decisions, system components, and interactions specific to the Live Queries functionality.

## Architecture Overview

The Live Queries architecture enables real-time query execution and result processing across a fleet of devices. It leverages osquery's distributed query capabilities to execute SQL queries on devices and return results to the Fleet server.

## Key Components

- **Query Campaign**: A collection of devices targeted for a specific query.
- **Query Distribution**: The mechanism for distributing queries to devices.
- **Result Collection**: The process of collecting and processing query results.
- **Result Storage**: The storage of query results for display and analysis.

## Architecture Diagram

```
[Placeholder for Live Queries Architecture Diagram]
```

## Query Execution Flow

### 1 - Fleet User Initiates the Query

```
Fleet User -> API Client (Frontend or Fleetctl) -> Server -> DB/Redis
```

1. Fleet user starts a live query through the UI or API.
2. API client initiates a campaign and gets an ID.
3. API client registers for notifications with the campaign ID.
4. Server stores the campaign information in the database/Redis.

### 2 - Agent Returns Results

```
osquery agent -> Server -> DB/Redis
```

1. osquery agent asks for queries from the server.
2. Server returns queries if found in the database/Redis.
3. osquery agent executes the queries and returns results to the server.
4. Server stores the results in Redis.

## Performance Considerations

Live queries can impact device performance, especially for complex queries or queries that return large result sets. The following considerations should be taken into account:

- **Query Complexity**: Complex queries can consume significant CPU resources on devices.
- **Result Size**: Large result sets can consume significant memory and network bandwidth.
- **Device Count**: Executing queries across a large number of devices can impact server performance.

## Troubleshooting

Common issues with live queries include:

- **Timeout**: Queries that take too long to execute may time out.
- **Connection Issues**: Devices may be unable to connect to the Fleet server.
- **Query Syntax Errors**: Incorrect SQL syntax can cause queries to fail.

For more information on troubleshooting live queries, see [Troubleshooting Live Queries](../../guides/troubleshooting-live-queries.md).

## Related Resources

- [Orchestration Product Group Documentation](../../product-groups/orchestration/) - Documentation for the Orchestration product group
- [Orchestration Development Guides](../../guides/orchestration/) - Guides for Orchestration development