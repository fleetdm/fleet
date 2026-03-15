# Report packs architecture

This document provides an overview of Fleet's Report Packs architecture.

## Introduction

Report packs in Fleet allow users to group related reports together for easier management and distribution. This document provides insights into the design decisions, system components, and interactions specific to the Report Packs functionality.

## Architecture overview

The Report Packs architecture enables the organization, configuration, and distribution of groups of reports across a fleet of devices. It leverages osquery's pack capabilities to execute multiple reports on devices and return results to the Fleet server.

## Key components

- **Pack Definition**: The definition of a pack, including the reports it contains and their schedules.
- **Pack Distribution**: The mechanism for distributing packs to devices.
- **Report Execution**: The process of executing reports within a pack.
- **Result Collection**: The process of collecting and processing report results.

## Architecture diagram

```
[Placeholder for Report Packs Architecture Diagram]
```

## Pack execution flow

### 1 - Fleet user creates a report pack

```
Fleet User -> API Client (Frontend or Fleetctl) -> Server -> DB
```

1. Fleet user creates a report pack for a fleet or globally through the UI or API.
2. Server stores the pack configuration in the database.

### 2 - Agent gets config file (with the report pack)

```
osquery agent -> Server -> DB
```

1. osquery agent requests the configuration file from the server.
2. Server merges fleet and global configurations, including packs.
3. Server returns the merged configuration to the agent.

### 3 - Agent executes reports and returns results

```
osquery agent -> Server -> Optional External Log
```

1. osquery agent runs the reports in the pack according to their schedules.
2. osquery agent sends the results to the server.
3. Server optionally forwards the results to an external logging system.

## Pack configuration

Report packs have several configuration options:

- **Name**: The name of the pack.
- **Description**: A description of the pack's purpose.
- **Reports**: The reports included in the pack.
- **Targets**: The devices or fleets targeted by the pack.
- **Schedules**: The schedules for each report in the pack.

## Performance considerations

Report packs can impact device performance, especially for packs with complex reports or reports that run frequently. The following considerations should be taken into account:

- **Report Complexity**: Complex reports can consume significant CPU resources on devices.
- **Report Frequency**: Reports that run frequently can impact device performance.
- **Pack Size**: Packs with many reports can impact device performance.

## Related resources

- [Orchestration Product Group Documentation](../../product-groups/orchestration/) - Documentation for the Orchestration product group
- [Orchestration Development Guides](../../guides/orchestration/) - Guides for Orchestration development