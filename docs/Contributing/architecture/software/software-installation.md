# Software Installation Architecture

This document provides an overview of Fleet's Software Installation architecture.

## Introduction

Software Installation in Fleet enables the deployment and installation of software packages across the device fleet. This document provides insights into the design decisions, system components, and interactions specific to the Software Installation functionality.

## Architecture Overview

The Software Installation architecture enables the configuration, distribution, and installation of software packages across a fleet of devices. It leverages platform-specific mechanisms to install software packages on devices.

## Key Components

- **Package Management**: The management of software packages available for installation.
- **Installation Configuration**: The configuration of installation parameters.
- **Installation Distribution**: The distribution of installation instructions to devices.
- **Installation Execution**: The execution of installation instructions on devices.
- **Installation Reporting**: The reporting of installation status and results.

## Architecture Diagram

```
[Placeholder for Software Installation Architecture Diagram]
```

## Installation Flow

### 1 - User Initiates Software Installation

```
User -> UI -> Server -> DB
```

1. User initiates software installation through the UI or API.
2. Server stores the installation request in the database.

### 2 - Server Distributes Installation Instructions

```
Server -> Device
```

1. Server sends installation instructions to the device.
2. Device receives and validates the instructions.

### 3 - Device Executes Installation

```
Device -> Software Package -> Device
```

1. Device downloads the software package.
2. Device installs the software using platform-specific mechanisms.

### 4 - Device Reports Installation Status

```
Device -> Server -> DB
```

1. Device reports installation status to the server.
2. Server stores the status in the database.
3. Server updates the UI with the status.

## Platform-Specific Implementation

### macOS

On macOS, software installation is performed using:

- **MDM Commands**: For MDM-managed devices, installation is performed using MDM commands.
- **Installer**: For non-MDM-managed devices, installation is performed using the `installer` command.
- **Package Managers**: For package-based software, installation is performed using package managers like Homebrew.

### Windows

On Windows, software installation is performed using:

- **MDM Commands**: For MDM-managed devices, installation is performed using MDM commands.
- **PowerShell**: For non-MDM-managed devices, installation is performed using PowerShell scripts.
- **Windows Installer**: For MSI-based software, installation is performed using the Windows Installer service.

### Linux

On Linux, software installation is performed using:

- **Package Managers**: Installation is performed using package managers like apt, yum, or dnf.
- **Shell Scripts**: For custom installations, installation is performed using shell scripts.

## Installation Parameters

Software installation can be configured with various parameters:

- **Target Devices**: The devices on which to install the software.
- **Installation Options**: Options specific to the software being installed.
- **Scheduling**: When to perform the installation.
- **Retry Logic**: How to handle installation failures.

## Related Resources

- [Software Product Group Documentation](../../product-groups/software/) - Documentation for the Software product group
- [Software Development Guides](../../guides/software/) - Guides for Software development