# Orchestration architecture overview

This document provides an overview of Fleet's Orchestration architecture.

## Introduction

Fleet's Orchestration architecture is designed to manage and report on devices at scale using osquery, providing visibility into device status, configuration, and security posture. This document provides insights into the design decisions, system components, and interactions specific to the Orchestration functionality.

## System components

The Orchestration architecture consists of the following main components:

- **Fleet Server**: The central component that manages device communication, report execution, and result processing.
- **osquery**: The open-source agent that runs on devices and executes queries.
- **Report Engine**: The component that processes and executes reports on devices.
- **Result Processing**: The component that processes and stores report results.
- **Fleets and Access Control**: The component that manages user access to devices and reports.

## Architecture diagram

```
[Placeholder for Orchestration Architecture Diagram]
```

## Report types

Fleet supports two main types of queries:

- **Live reports**: Ad-hoc reports that are executed in real-time and return results immediately.
- **Scheduled reports**: Reports that are executed on a schedule and store results for later analysis.

### Live queries

Live queries are executed in real-time and return results immediately. The process for executing a live report is as follows:

1. User initiates a report through the UI or API.
2. Fleet server creates a campaign for the report.
3. Devices check in with the Fleet server and receive the report.
4. Devices execute the report and return results.
5. Fleet server processes and displays the results.

### Scheduled queries

Scheduled queries are executed on a schedule and store results for later analysis. The process for executing a scheduled report is as follows:

1. User creates a scheduled report through the UI or API.
2. Fleet server stores the report in the database.
3. Devices check in with the Fleet server and receive the report configuration.
4. Devices execute the report on the specified schedule.
5. Devices return results to the Fleet server.
6. Fleet server processes and stores the results.

## Integration points

The Orchestration architecture integrates with the following components:

- **Fleet Server**: For device management and report execution.
- **Database**: For storing report configurations and results.
- **Redis**: For storing live report results.
- **External Logging Systems**: For storing report results for long-term analysis.

## Related resources

- [Orchestration Product Group Documentation](../../product-groups/orchestration/) - Documentation for the Orchestration product group
- [Orchestration Development Guides](../../guides/orchestration/) - Guides for Orchestration development