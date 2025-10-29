# Query packs architecture

This document provides an overview of Fleet's Query Packs architecture.

## Introduction

Query packs in Fleet allow users to group related queries together for easier management and distribution. This document provides insights into the design decisions, system components, and interactions specific to the Query Packs functionality.

## Architecture overview

The Query Packs architecture enables the organization, configuration, and distribution of groups of queries across a fleet of devices. It leverages osquery's pack capabilities to execute multiple queries on devices and return results to the Fleet server.

## Key components

- **Pack Definition**: The definition of a pack, including the queries it contains and their schedules.
- **Pack Distribution**: The mechanism for distributing packs to devices.
- **Query Execution**: The process of executing queries within a pack.
- **Result Collection**: The process of collecting and processing query results.

## Architecture diagram

```
[Placeholder for Query Packs Architecture Diagram]
```

## Pack execution flow

### 1 - Fleet user creates a query pack

```
Fleet User -> API Client (Frontend or Fleetctl) -> Server -> DB
```

1. Fleet user creates a query pack for a team or globally through the UI or API.
2. Server stores the pack configuration in the database.

### 2 - Agent gets config file (with the query pack)

```
osquery agent -> Server -> DB
```

1. osquery agent requests the configuration file from the server.
2. Server merges team and global configurations, including packs.
3. Server returns the merged configuration to the agent.

### 3 - Agent executes queries and returns results

```
osquery agent -> Server -> Optional External Log
```

1. osquery agent runs the queries in the pack according to their schedules.
2. osquery agent sends the results to the server.
3. Server optionally forwards the results to an external logging system.

## Pack configuration

Query packs have several configuration options:

- **Name**: The name of the pack.
- **Description**: A description of the pack's purpose.
- **Queries**: The queries included in the pack.
- **Targets**: The devices or teams targeted by the pack.
- **Schedules**: The schedules for each query in the pack.

## Performance considerations

Query packs can impact device performance, especially for packs with complex queries or queries that run frequently. The following considerations should be taken into account:

- **Query Complexity**: Complex queries can consume significant CPU resources on devices.
- **Query Frequency**: Queries that run frequently can impact device performance.
- **Pack Size**: Packs with many queries can impact device performance.

## Related resources

- [Orchestration Product Group Documentation](../../product-groups/orchestration/) - Documentation for the Orchestration product group
- [Orchestration Development Guides](../../guides/orchestration/) - Guides for Orchestration development