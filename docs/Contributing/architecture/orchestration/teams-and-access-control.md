# Teams and Access Control Architecture

This document provides an overview of Fleet's Teams and Access Control architecture.

## Introduction

Teams and Access Control in Fleet enable organizations to manage user access to devices, queries, and other resources based on team membership and roles. This document provides insights into the design decisions, system components, and interactions specific to the Teams and Access Control functionality.

## Architecture Overview

The Teams and Access Control architecture enables the organization of devices and users into teams, and the management of user access to resources based on team membership and roles.

## Key Components

- **Teams**: Logical groupings of devices and users.
- **Roles**: Sets of permissions that define what actions users can perform.
- **Access Control**: The mechanism for controlling user access to resources.
- **Resource Ownership**: The association of resources with teams.

## Architecture Diagram

```
[Placeholder for Teams and Access Control Architecture Diagram]
```

## Team Structure

Teams in Fleet have the following characteristics:

- **Name**: A unique name for the team.
- **Description**: A description of the team's purpose.
- **Members**: Users who are members of the team.
- **Devices**: Devices that are assigned to the team.
- **Resources**: Resources (queries, packs, etc.) that are owned by the team.

## Role-Based Access Control

Fleet uses role-based access control (RBAC) to manage user access to resources. Roles define what actions users can perform on resources.

### Default Roles

Fleet includes the following default roles:

- **Admin**: Full access to all resources.
- **Maintainer**: Can manage resources but cannot manage users or teams.
- **Observer**: Can view resources but cannot modify them.

### Permission Model

Permissions in Fleet are defined at the resource level and are associated with roles. The permission model includes:

- **Resource Types**: The types of resources that can be accessed (devices, queries, packs, etc.).
- **Actions**: The actions that can be performed on resources (view, create, update, delete).
- **Scopes**: The scope of the permission (global, team).

## Resource Ownership

Resources in Fleet can be owned by a team or be global. Resource ownership determines who can access and manage the resource:

- **Team Resources**: Resources owned by a team can be accessed by team members based on their roles.
- **Global Resources**: Resources that are not owned by a team can be accessed by all users based on their roles.

## Related Resources

- [Orchestration Product Group Documentation](../../product-groups/orchestration/) - Documentation for the Orchestration product group
- [Orchestration Development Guides](../../guides/orchestration/) - Guides for Orchestration development
- [Teams](../../product-groups/orchestration/teams.md) - Detailed documentation on teams