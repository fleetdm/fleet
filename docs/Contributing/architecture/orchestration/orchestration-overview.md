# Orchestration architecture overview

This document provides an overview of Fleet's Orchestration architecture.

## Introduction

Fleet's Orchestration architecture is designed to manage and query devices at scale using osquery, providing visibility into device status, configuration, and security posture. This document provides insights into the design decisions, system components, and interactions specific to the Orchestration functionality.

## System components

The Orchestration architecture consists of the following main components:

- **Fleet Server**: The central component that manages device communication, query execution, and result processing.
- **osquery**: The open-source agent that runs on devices and executes queries.
- **Query Engine**: The component that processes and executes queries on devices.
- **Result Processing**: The component that processes and stores query results.
- **Teams and Access Control**: The component that manages user access to devices and queries.

## Architecture diagram

```
[Placeholder for Orchestration Architecture Diagram]
```

## Query types

Fleet supports two main types of queries:

- **Live Queries**: Ad-hoc queries that are executed in real-time and return results immediately.
- **Scheduled Queries**: Queries that are executed on a schedule and store results for later analysis.

### Live queries

Live queries are executed in real-time and return results immediately. The process for executing a live query is as follows:

1. User initiates a query through the UI or API.
2. Fleet server creates a campaign for the query.
3. Devices check in with the Fleet server and receive the query.
4. Devices execute the query and return results.
5. Fleet server processes and displays the results.

### Scheduled queries

Scheduled queries are executed on a schedule and store results for later analysis. The process for executing a scheduled query is as follows:

1. User creates a scheduled query through the UI or API.
2. Fleet server stores the query in the database.
3. Devices check in with the Fleet server and receive the query configuration.
4. Devices execute the query on the specified schedule.
5. Devices return results to the Fleet server.
6. Fleet server processes and stores the results.

## Integration points

The Orchestration architecture integrates with the following components:

- **Fleet Server**: For device management and query execution.
- **Database**: For storing query configurations and results.
- **Redis**: For storing live query results.
- **External Logging Systems**: For storing query results for long-term analysis.

## Related resources

- [Orchestration Product Group Documentation](../../product-groups/orchestration/) - Documentation for the Orchestration product group
- [Orchestration Development Guides](../../guides/orchestration/) - Guides for Orchestration development