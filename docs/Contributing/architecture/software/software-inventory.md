# Software Inventory Architecture

This document provides an overview of Fleet's Software Inventory architecture.

## Introduction

Software Inventory in Fleet provides visibility into the software installed on devices across the fleet. This document provides insights into the design decisions, system components, and interactions specific to the Software Inventory functionality.

## Architecture Overview

The Software Inventory architecture enables the collection, processing, and visualization of software information across a fleet of devices. It leverages osquery's capabilities to collect software information and Fleet's infrastructure to process and display it.

## Key Components

- **Inventory Collection**: The process of collecting software information from devices.
- **Inventory Processing**: The processing of collected information for storage and analysis.
- **Inventory Storage**: The storage of software information for querying and analysis.
- **Inventory Visualization**: The display of software information in the Fleet UI.

## Architecture Diagram

```
[Placeholder for Software Inventory Architecture Diagram]
```

## Inventory Collection Flow

### 1 - Agent Collects Software Information

```
osquery agent -> Server
```

1. osquery agent collects software information using osquery tables.
2. osquery agent sends the information to the Fleet server.

### 2 - Server Processes and Stores Information

```
Server -> DB
```

1. Server processes the received information.
2. Server stores the information in the database.

### 3 - UI Displays Information

```
UI -> Server -> DB
```

1. UI requests software information from the server.
2. Server retrieves information from the database.
3. Server returns information to the UI.
4. UI displays the information.

## Collected Information

Software Inventory collects the following information:

- **Name**: The name of the software.
- **Version**: The version of the software.
- **Publisher**: The publisher of the software.
- **Installation Date**: The date the software was installed.
- **Installation Location**: The location where the software is installed.
- **Size**: The size of the software installation.

## Platform-Specific Considerations

### macOS

On macOS, software information is collected from:

- **Applications**: Information about installed applications.
- **Packages**: Information about installed packages.
- **App Store**: Information about applications installed from the App Store.

### Windows

On Windows, software information is collected from:

- **Registry**: Information about software registered in the Windows Registry.
- **WMI**: Information about software available through Windows Management Instrumentation.
- **Microsoft Store**: Information about applications installed from the Microsoft Store.

### Linux

On Linux, software information is collected from:

- **Package Managers**: Information about packages installed through package managers (apt, yum, etc.).
- **Flatpak/Snap**: Information about applications installed through Flatpak or Snap.

## Related Resources

- [Software Product Group Documentation](../../product-groups/software/) - Documentation for the Software product group
- [Software Development Guides](../../guides/software/) - Guides for Software development