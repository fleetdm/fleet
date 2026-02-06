# Software architecture overview

This document provides an overview of Fleet's Software architecture.

## Introduction

Fleet's Software architecture is designed to manage software across the device fleet, including software inventory, vulnerability management, and software installation. This document provides insights into the design decisions, system components, and interactions specific to the Software functionality.

## System components

The Software architecture consists of the following main components:

- **Software Inventory**: The component that collects and manages information about installed software.
- **Vulnerability Management**: The component that identifies and manages software vulnerabilities.
- **Software Installation**: The component that manages the installation of software on devices.
- **Software Updates**: The component that manages software updates on devices.
- **Software Policies**: The component that defines and enforces software policies.

## Architecture diagram

```
[Placeholder for Software Architecture Diagram]
```

## Software installation

The Software Installation component manages the installation of software on devices. It leverages platform-specific mechanisms to install software packages.

### Installation flow

1. User initiates software installation through the UI or API.
2. Server sends installation instructions to the device.
3. Device installs the software using platform-specific mechanisms.
4. Device reports installation status to the server.

## Integration points

The Software architecture integrates with the following components:

- **Fleet Server**: For device management and software inventory collection.
- **Database**: For storing software inventory and vulnerability information.
- **External Vulnerability Databases**: For retrieving vulnerability information.
- **Platform-Specific Installation Mechanisms**: For installing software on devices.

## Related resources

- [Software Product Group Documentation](../../product-groups/software/) - Documentation for the Software product group
