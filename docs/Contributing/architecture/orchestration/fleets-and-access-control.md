# Fleets and access control architecture

This document provides an overview of Fleet's Fleets and Access Control architecture.

## Introduction

Fleets and Access Control in Fleet enable organizations to manage user access to devices, queries, and other resources based on fleet membership and roles. This document provides insights into the design decisions, system components, and interactions specific to the Fleets and Access Control functionality.

## Architecture overview

The Fleets and Access Control architecture enables the organization of devices and users into fleets, and the management of user access to resources based on fleets membership and roles.

## Key components

- **Fleets**: Logical groupings of devices and users.
- **Roles**: Sets of permissions that define what actions users can perform.
- **Access Control**: The mechanism for controlling user access to resources.
- **Resource Ownership**: The association of resources with fleets.

## Role-Based Access Control (RBAC)

Fleet uses role-based access control (RBAC) to manage user access to resources. Roles define what actions users can perform on resources. See our [RBAC guide](https://fleetdm.com/guides/role-based-access) for details.

## Related resources

- [Role-based access control guide](https://fleetdm.com/guides/role-based-access)
- [Orchestration Product Group Documentation](../../product-groups/orchestration/) - Documentation for the Orchestration product group
- [Orchestration Development Guides](../../guides/orchestration/) - Guides for Orchestration development