# MDM architecture overview

This document provides an overview of Fleet's Mobile Device Management (MDM) architecture.

## Introduction

Fleet's MDM architecture is designed to manage devices across different platforms, including Apple (macOS, iOS) and Windows. This document provides insights into the design decisions, system components, and interactions specific to the MDM functionality.

## System components

The MDM architecture consists of the following main components:

- **MDM Server**: The central component that manages device enrollment, configuration, and commands.
- **Device Enrollment**: The process by which devices are registered with the MDM server.
- **Configuration Profiles**: Settings and policies that are applied to managed devices.
- **Commands**: Instructions sent to devices to perform specific actions.
- **Device Communication**: The protocols and mechanisms used for communication between the MDM server and devices.

## Architecture diagram

```
[Placeholder for MDM Architecture Diagram]
```

## Integration points

The MDM architecture integrates with the following components:

- **Fleet Server**: For device management and policy enforcement.
- **Database**: For storing device information, configurations, and policies.
- **Authentication Systems**: For user and device authentication.
- **Certificate Authorities**: For issuing and managing device certificates.

## Platform-specific considerations

### Apple MDM

See [Apple MDM Architecture](../apple-mdm/apple-mdm-architecture.md) for details on Apple-specific MDM architecture.

### Windows MDM

See [Windows MDM Architecture](../windows-mdm/windows-mdm-architecture.md) for details on Windows-specific MDM architecture.

## Related resources

- [MDM index](README.md) - Cross-platform MDM contributor docs
- [Contributor guides](../guides/README.md) - How-to guides for common tasks