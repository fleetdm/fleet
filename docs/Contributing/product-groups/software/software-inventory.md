# Software Inventory

This document provides an overview of Fleet's Software Inventory functionality.

## Introduction

Software Inventory in Fleet provides visibility into the software installed on devices across the fleet. This includes detailed information about installed applications, packages, and system software.

## Key Features

### Comprehensive Inventory

Fleet collects comprehensive information about installed software:

- Application name, version, and publisher
- Installation date and location
- Size and other metadata
- Platform-specific details

### Cross-Platform Support

Fleet supports software inventory collection on multiple platforms:

- **Windows**: Registry-based applications, Microsoft Store apps, and more
- **macOS**: Applications, packages, and App Store apps
- **Linux**: Package manager-based applications, Flatpak/Snap apps, and more

### Real-Time Updates

Software inventory is updated in real-time:

- New software installations are detected promptly
- Software removals are reflected in the inventory
- Version changes are tracked

### Search and Filter

Fleet provides powerful search and filter capabilities for software inventory:

- Search by name, version, publisher, etc.
- Filter by platform, installation date, etc.
- Group by various criteria

### Reporting

Fleet generates reports based on software inventory:

- Software distribution across the fleet
- Version distribution for specific software
- Installation trends over time

## Implementation

### Data Collection

Software inventory data is collected using osquery:

- Platform-specific tables are used to collect data
- Data is collected at regular intervals
- Changes are detected and reported

### Data Processing

Collected data is processed by the Fleet server:

- Raw data is normalized and enriched
- Duplicate entries are merged
- Historical data is maintained

### Data Storage

Processed data is stored in the Fleet database:

- Efficient storage for large inventories
- Fast retrieval for UI and API access
- Historical data for trend analysis

### Data Presentation

Inventory data is presented in the Fleet UI:

- List and detail views
- Search and filter capabilities
- Visualization of trends and distributions

## Development

For information about developing Software Inventory functionality in Fleet, see [Software Inventory Architecture](../../architecture/software/software-inventory.md) and [Software Development Guides](../../guides/software/).

## Related Resources

- [Software Inventory Architecture](../../architecture/software/software-inventory.md) - Detailed architecture documentation
- [Software Development Guides](../../guides/software/) - Guides for Software development