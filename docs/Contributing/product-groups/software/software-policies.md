# Software Policies

This document provides an overview of Fleet's Software Policies functionality.

## Introduction

Software Policies in Fleet enable organizations to define and enforce rules about what software can be installed and run on devices. This includes allowed and prohibited software, version requirements, and automated remediation.

## Key Features

### Policy Types

Fleet supports multiple types of software policies:

- **Allowed Software**: Software that is allowed to be installed and run
- **Prohibited Software**: Software that is not allowed to be installed or run
- **Required Software**: Software that must be installed
- **Version Policies**: Policies about software versions (minimum, maximum, specific)

### Policy Definition

Fleet provides flexible options for defining software policies:

- Software identification by name, publisher, version
- Version range specification
- Platform-specific policies
- Policy exceptions

### Policy Targeting

Fleet enables targeted application of policies:

- Policies for specific devices or device groups
- Team-based targeting
- Label-based targeting
- Platform-specific targeting

### Policy Enforcement

Fleet enforces software policies through multiple mechanisms:

- Installation blocking
- Software removal
- Version enforcement
- User notification

### Compliance Monitoring

Fleet monitors device compliance with software policies:

- Real-time compliance status
- Compliance history
- Detailed violation information
- Compliance metrics and reporting

### Automated Remediation

Fleet provides automated remediation for policy violations:

- Software installation for required software
- Software removal for prohibited software
- Software updates for version violations
- Configurable remediation actions

## Implementation

### Policy Definition

Software policies are defined through the Fleet UI or API:

- Policy parameters and conditions
- Policy scope and targeting
- Policy enforcement level
- Policy exceptions

### Policy Distribution

Policies are distributed to devices through Fleet's device management infrastructure:

- Secure transmission of policy definitions
- Policy versioning
- Incremental policy updates
- Policy acknowledgment

### Policy Evaluation

Devices evaluate software against policies:

- Software inventory comparison
- Installation monitoring
- Version checking
- Exception handling

### Policy Enforcement

Policies are enforced through platform-specific mechanisms:

- Installation blocking
- Software removal
- Update enforcement
- User notification

### Compliance Reporting

Compliance status is reported back to the Fleet server:

- Compliance state
- Violation details
- Remediation actions taken
- Enforcement failures

## Platform-Specific Implementation

### Windows

On Windows, software policies are enforced using:

- **AppLocker**: For application execution control
- **MDM Policies**: For MDM-managed devices
- **Windows Installer**: For installation control

### macOS

On macOS, software policies are enforced using:

- **Gatekeeper**: For application execution control
- **MDM Profiles**: For MDM-managed devices
- **System Integrity Protection**: For system software protection

### Linux

On Linux, software policies are enforced using:

- **Package Manager Policies**: For installation control
- **AppArmor/SELinux**: For application execution control
- **Custom Scripts**: For policy enforcement

## Development

For information about developing Software Policies functionality in Fleet, see [Software Policies Architecture](../../architecture/software/software-policies.md) and [Software Development Guides](../../guides/software/).

## Related Resources

- [Software Policies Architecture](../../architecture/software/software-policies.md) - Detailed architecture documentation
- [Software Development Guides](../../guides/software/) - Guides for Software development