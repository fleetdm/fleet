# Windows MDM architecture

This document provides an overview of Fleet's Windows Mobile Device Management (MDM) architecture.

## Introduction

Fleet's Windows MDM architecture is designed to manage Windows devices. This document provides insights into the design decisions, system components, and interactions specific to the Windows MDM functionality.

## Windows MDM protocol

Windows MDM is based on the OMA Device Management (OMA-DM) protocol and uses SyncML for data exchange. Microsoft has extended the protocol with Windows-specific features and capabilities.

### Key components

- **MDM Enrollment**: The process by which devices are registered with the MDM server.
- **MDM Sync**: The process by which devices communicate with the MDM server.
- **Configuration Service Providers (CSPs)**: Components that provide an interface to device settings.
- **MDM Policies**: Settings and configurations applied to managed devices.

## Architecture diagram

```
[Placeholder for Windows MDM Architecture Diagram]
```

## Enrollment flows

### User-initiated enrollment

1. User downloads and installs the enrollment package.
2. Device enrolls with the MDM server.
3. MDM server sends initial configuration profiles.

### Azure AD join

1. Device is joined to Azure AD.
2. Device automatically enrolls with the MDM server.

## Configuration Service Providers (CSPs)

Windows MDM uses Configuration Service Providers (CSPs) to configure and manage devices. CSPs are the interface through which the MDM server can access and modify device settings.

Common CSPs used by Fleet include:

- **Policy CSP**: For configuring device policies.
- **EnterpriseModernAppManagement CSP**: For managing modern applications.
- **DeviceStatus CSP**: For retrieving device status information.
- **WindowsUpdatePolicy CSP**: For configuring Windows Update settings.

## SyncML structure

Windows MDM uses SyncML for communication between the device and the MDM server. A typical SyncML message includes:

- **Header**: Contains session information.
- **Body**: Contains commands and data.
- **Status**: Contains the result of previous commands.

## Command delivery and poll schedule

Fleet delivers commands and configuration profiles to a Windows device during an OMA-DM session. A device starts a session either on its scheduled poll (the DMClient `Poll` CSP interval) or when something triggers an unscheduled, client-initiated session.

Fleet Windows MDM traditionally relied on frequent polling to pick up queued commands. A short poll interval keeps command latency low, but at scale it produces a near-constant stream of sessions even when there is nothing to deliver, which adds load to the Fleet server and database. To reduce that load without increasing latency, Fleet relaxes the poll for hosts whose fleetd can be woken and uses fleetd to start a session only when there is work to do:

- **Capability**: fleetd reports a Windows MDM sync capability on its config check-in. Fleet persists this per enrollment so both the poll-schedule logic and the wake logic read a single signal.
- **Relaxed poll**: once a host is known to be wakeable, Fleet enqueues a SyncML `Replace` command that sets the DMClient `Poll` interval to a long value (8 hours). Like any other Windows MDM command, it is delivered in the response to the device's next OMA-DM session, and the device applies the new interval to its local poll schedule. Fleet records the intended schedule per enrollment, so the command is enqueued once per change (typically once per host lifetime) rather than on every session.
- **On-demand wake**: when a non-poll command is queued for the host, Fleet marks the enrollment as having pending commands. The host's next fleetd config check-in returns a sync request, and fleetd starts an immediate OMA-DM session to deliver it. Hosts running a version of fleetd that does not support on-demand sync keep the short poll interval and continue to rely on polling.

Immediately after a device boots or resumes, fleetd-initiated sessions can take a few minutes to begin connecting. This is harmless: Fleet keeps returning the sync request on every config check-in until the command is acknowledged, so the wake retries until a session succeeds.

Fleet wakes devices through fleetd rather than through WNS (Windows Push Notification Service), the push channel Windows MDM traditionally uses for server-initiated sessions. WNS requires additional setup and credential management, is sometimes unreliable, and is a larger engineering lift, so the existing fleetd channel is used instead. WNS-based wake remains an option to add in the future.

## Related resources

- [MDM Product Group Documentation](../../product-groups/mdm/) - Documentation for the MDM product group
- [MDM Development Guides](../../guides/mdm/) - Guides for MDM development
- [Windows MDM Glossary and Protocol](../../product-groups/mdm/windows-mdm-glossary-and-protocol.md) - Glossary of Windows MDM terms and protocol details