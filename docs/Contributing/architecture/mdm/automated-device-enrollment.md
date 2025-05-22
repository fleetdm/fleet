# Automated Device Enrollment architecture

This document provides an overview of Fleet's Automated Device Enrollment (ADE) architecture for MDM.

## Introduction

Automated Device Enrollment (ADE) in Fleet's MDM allows for zero-touch deployment of devices, enabling organizations to automatically enroll and configure devices without manual intervention. This document provides insights into the design decisions, system components, and interactions specific to the ADE functionality.

## Architecture overview

The ADE architecture integrates with platform-specific enrollment programs (Apple Business Manager/Apple School Manager for Apple devices) to automatically enroll devices when they are activated.

## Key components

- **Enrollment Program Integration**: Integration with platform-specific enrollment programs.
- **Device Assignment**: Mapping between devices and enrollment configurations.
- **Enrollment Profiles**: Configurations applied during the enrollment process.
- **Synchronization**: Mechanisms to synchronize device information with enrollment programs.

## Architecture diagram

```
[Placeholder for Automated Device Enrollment Architecture Diagram]
```

## Platform-specific implementation

### Apple automated device enrollment

For Apple devices, ADE (formerly known as DEP) involves the following components:

- **Apple Business Manager/Apple School Manager**: Web portals for managing device enrollment.
- **MDM Server Tokens**: Tokens that authenticate the MDM server with Apple's services.
- **Enrollment Profiles**: Configurations that define the enrollment experience.
- **Device Assignments**: Mapping between devices and enrollment profiles.

#### Synchronization process

Synchronization of devices from all ABM tokens uploaded to Fleet happens in the `dep_syncer` cron job, which runs every 1 minute.

We keep a record of all devices ingested via the ADE sync in the `host_dep_assignments` table. Entries in this table are soft-deleted.

On every run, we pull the list of added/modified/deleted devices and:

1. If the host was added/modified, we:
   - Create/match a row in the `hosts` table for the new host. This allows IT admin to view the host by serial in team lists before it turns on MDM or has `fleetd` installed.
   - Always assign a JSON profile for added devices. We assign JSON profile for modified devices if the profile has not been modified according to Apple DEP device response.
2. If the host was deleted, we soft delete the `host_dep_assignments` entry.

#### Special case: Host in ABM is deleted in Fleet

If an IT admin deletes a host in the UI/API, and we have a non-deleted entry in `host_dep_assignments` for the host, we immediately create a new host entry as if the device was just ingested from the ABM sync.

## Related resources

- [MDM Product Group Documentation](../../product-groups/mdm/) - Documentation for the MDM product group
- [MDM Development Guides](../../guides/mdm/) - Guides for MDM development