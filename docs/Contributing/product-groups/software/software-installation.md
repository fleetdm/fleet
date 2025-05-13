# Software Installation

This document provides an overview of Fleet's Software Installation functionality.

## Introduction

Software Installation in Fleet enables the deployment and installation of software packages across the device fleet. This includes configuring, distributing, and executing software installations on devices.

## Key Features

### Cross-Platform Support

Fleet supports software installation on multiple platforms:

- **Windows**: MSI packages, EXE installers, Microsoft Store apps
- **macOS**: PKG packages, DMG images, App Store apps
- **Linux**: DEB/RPM packages, Flatpak/Snap apps

### Installation Configuration

Fleet provides flexible configuration options for software installations:

- Installation parameters and options
- Silent/unattended installation
- Pre and post-installation scripts
- Installation scheduling

### Targeted Deployment

Fleet enables targeted deployment of software:

- Installation on specific devices or device groups
- Team-based targeting
- Label-based targeting
- Platform-specific targeting

### Installation Status Tracking

Fleet tracks the status of software installations:

- Real-time installation progress
- Success/failure reporting
- Detailed error information
- Installation history

### Retry and Error Handling

Fleet provides robust retry and error handling:

- Automatic retry of failed installations
- Configurable retry policies
- Detailed error logging
- Notification of persistent failures

## Implementation

### Package Management

Fleet manages software packages for installation:

- Package storage and versioning
- Package metadata management
- Package validation
- Platform-specific package handling

### Installation Distribution

Fleet distributes installation instructions to devices:

- Secure transmission of installation parameters
- Bandwidth-efficient distribution
- Prioritization of critical installations
- Throttling for large-scale deployments

### Installation Execution

Fleet executes installations on devices:

- Platform-specific installation mechanisms
- Privilege elevation when required
- Environment preparation
- Installation verification

### Installation Reporting

Fleet collects and reports installation results:

- Success/failure status
- Installation logs
- Error details
- Installation metrics

## Platform-Specific Implementation

### Windows

On Windows, software installation is performed using:

- **MDM Commands**: For MDM-managed devices
- **PowerShell**: For script-based installations
- **Windows Installer**: For MSI-based installations

### macOS

On macOS, software installation is performed using:

- **MDM Commands**: For MDM-managed devices
- **Installer**: For PKG-based installations
- **DMG Mounting**: For DMG-based installations

### Linux

On Linux, software installation is performed using:

- **Package Managers**: For package-based installations
- **Shell Scripts**: For script-based installations
- **Flatpak/Snap**: For Flatpak/Snap-based installations

## Development

For information about developing Software Installation functionality in Fleet, see [Software Installation Architecture](../../architecture/software/software-installation.md) and [Software Development Guides](../../guides/software/).

## Related Resources

- [Software Installation Architecture](../../architecture/software/software-installation.md) - Detailed architecture documentation
- [Software Development Guides](../../guides/software/) - Guides for Software development