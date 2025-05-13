# Software Policies Architecture

This document provides an overview of Fleet's Software Policies architecture.

## Introduction

Software Policies in Fleet enable organizations to define and enforce rules about what software can be installed and run on devices. This document provides insights into the design decisions, system components, and interactions specific to the Software Policies functionality.

## Architecture Overview

The Software Policies architecture enables the definition, distribution, and enforcement of software policies across a fleet of devices. It leverages Fleet's software inventory and management capabilities to ensure compliance with organizational policies.

## Key Components

- **Policy Definition**: The definition of software policies.
- **Policy Distribution**: The distribution of policies to devices.
- **Policy Enforcement**: The enforcement of policies on devices.
- **Compliance Monitoring**: The monitoring of device compliance with policies.
- **Remediation**: The remediation of policy violations.

## Architecture Diagram

```
[Placeholder for Software Policies Architecture Diagram]
```

## Policy Types

Fleet supports the following types of software policies:

- **Allowed Software**: Software that is allowed to be installed and run.
- **Prohibited Software**: Software that is not allowed to be installed or run.
- **Required Software**: Software that must be installed.
- **Version Policies**: Policies about software versions (minimum, maximum, specific).

## Policy Definition Flow

### 1 - User Defines Policy

```
User -> UI -> Server -> DB
```

1. User defines a software policy through the UI or API.
2. Server validates the policy.
3. Server stores the policy in the database.

### 2 - Server Distributes Policy

```
Server -> Device
```

1. Server sends the policy to the device.
2. Device receives and validates the policy.

## Policy Enforcement Flow

### 1 - Device Monitors Software Changes

```
Device -> Software Changes -> Device
```

1. Device monitors for software installation, removal, and updates.
2. Device checks changes against policies.

### 2 - Device Enforces Policies

```
Device -> Software -> Device
```

1. Device blocks installation of prohibited software.
2. Device prompts for installation of required software.
3. Device enforces version policies.

### 3 - Device Reports Compliance

```
Device -> Server -> DB
```

1. Device reports compliance status to the server.
2. Server stores the status in the database.
3. Server updates the UI with the status.

## Remediation Flow

### 1 - Server Identifies Non-Compliance

```
Server -> DB -> Server
```

1. Server identifies devices that are not compliant with policies.

### 2 - Server Initiates Remediation

```
Server -> Device
```

1. Server sends remediation instructions to the device.
2. Device executes remediation actions.

### 3 - Device Reports Remediation Status

```
Device -> Server -> DB
```

1. Device reports remediation status to the server.
2. Server stores the status in the database.
3. Server updates the UI with the status.

## Policy Parameters

Software policies can be configured with various parameters:

- **Target Devices**: The devices to which the policy applies.
- **Software Criteria**: Criteria for identifying software (name, publisher, version, etc.).
- **Enforcement Level**: How strictly to enforce the policy (block, warn, log).
- **Exceptions**: Exceptions to the policy.
- **Remediation Actions**: Actions to take when a policy is violated.

## Related Resources

- [Software Product Group Documentation](../../product-groups/software/) - Documentation for the Software product group
- [Software Development Guides](../../guides/software/) - Guides for Software development