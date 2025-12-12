# Teams and access control architecture

This document provides an overview of Fleet's Teams and Access Control architecture.

## Introduction

Teams and Access Control in Fleet enable organizations to manage user access to devices, queries, and other resources based on team membership and roles. This document provides insights into the design decisions, system components, and interactions specific to the Teams and Access Control functionality.

## Architecture overview

The Teams and Access Control architecture enables the organization of devices and users into teams, and the management of user access to resources based on team membership and roles.

## Key components

- **Teams**: Logical groupings of devices and users.
- **Roles**: Sets of permissions that define what actions users can perform.
- **Access Control**: The mechanism for controlling user access to resources.
- **Resource Ownership**: The association of resources with teams.

## Role-Based Access Control (RBAC)

Fleet uses role-based access control (RBAC) to manage user access to resources. Roles define what actions users can perform on resources. See our [RBAC guide](https://fleetdm.com/guides/role-based-access) for details.

## Related resources

- [Role-based access control guide](https://fleetdm.com/guides/role-based-access)
- [Orchestration Product Group Documentation](../../product-groups/orchestration/) - Documentation for the Orchestration product group
- [Orchestration Development Guides](../../guides/orchestration/) - Guides for Orchestration development
- [Teams](../../product-groups/orchestration/teams.md) - Detailed documentation on teams