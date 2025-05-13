# Software Overview

This document provides an overview of Fleet's Software functionality.

## Introduction

Fleet's Software functionality allows organizations to manage software across their device fleet, including software inventory, vulnerability management, and software installation.

## Key Features

### Software Inventory

Fleet provides a comprehensive inventory of software installed on devices across the fleet. This includes:

- Detailed information about installed software (name, version, publisher, etc.)
- Cross-platform support for Windows, macOS, and Linux
- Integration with osquery for efficient data collection

### Vulnerability Management

Fleet identifies and manages software vulnerabilities in the device fleet:

- Comparison of installed software with known vulnerabilities
- Severity assessment using CVSS scores
- Remediation recommendations
- Integration with external vulnerability databases

### Software Installation

Fleet enables the deployment and installation of software packages:

- Cross-platform support for Windows, macOS, and Linux
- Support for various package formats (MSI, PKG, DMG, etc.)
- Installation status tracking
- Error handling and reporting

### Software Updates

Fleet manages software updates across the device fleet:

- Identification of available updates
- Controlled deployment of updates
- Update status tracking
- Support for system and application updates

### Software Policies

Fleet enforces software policies across the device fleet:

- Definition of allowed, prohibited, and required software
- Version policy enforcement
- Compliance monitoring
- Automated remediation

## Architecture

The Software functionality is built on Fleet's core architecture and integrates with other Fleet components:

- **Fleet Server**: Manages device communication and data processing
- **osquery**: Collects software information from devices
- **Database**: Stores software inventory and policy information
- **UI**: Provides a user interface for managing software

For more detailed information about the Software architecture, see [Software Architecture](../../architecture/software/).

## Development

For information about developing Software functionality in Fleet, see [Software Development Guides](../../guides/software/).

## Related Resources

- [Software Architecture](../../architecture/software/) - Detailed architecture documentation
- [Software Development Guides](../../guides/software/) - Guides for Software development