# Apple MDM Architecture

This document provides an overview of Fleet's Apple Mobile Device Management (MDM) architecture.

## Introduction

Fleet's Apple MDM architecture is designed to manage Apple devices, including macOS and iOS devices. This document provides insights into the design decisions, system components, and interactions specific to the Apple MDM functionality.

## Apple MDM Protocol

The Apple MDM protocol is a set of web services that allow an MDM server to communicate with Apple devices. The protocol is based on HTTP and uses XML (Property Lists) for data exchange.

### Key Components

- **MDM Enrollment**: The process by which devices are registered with the MDM server.
- **MDM Check-in**: The process by which devices communicate with the MDM server.
- **MDM Commands**: Instructions sent to devices to perform specific actions.
- **Push Notifications**: Used to notify devices that they should check in with the MDM server.

## NanoMDM Integration

Fleet uses [NanoMDM](https://github.com/micromdm/nanomdm) to handle the core protocol operations for Apple MDM. Fleet extends the protocol and adds custom handling to align with our desired workflows.

After NanoMDM processes a protocol operation, it allows for custom logic by implementing the `CheckinAndCommandService` interface. Our implementation can be found in the service layer.

## Architecture Diagram

```
[Placeholder for Apple MDM Architecture Diagram]
```

## Enrollment Flows

### User-Initiated Enrollment

1. User downloads and installs the enrollment profile.
2. Device enrolls with the MDM server.
3. MDM server sends initial configuration profiles.

### Automated Device Enrollment (ADE)

1. Device is assigned to the organization in Apple Business Manager.
2. Device is assigned an enrollment profile in Fleet.
3. When the device is activated, it automatically enrolls with the MDM server.

## Certificate Management

Apple MDM relies heavily on certificates for device identity and secure communication. Fleet manages the following certificates:

- **MDM Vendor Certificate**: Used to sign enrollment profiles.
- **SCEP Certificate**: Used for device identity.
- **Push Certificate**: Used for sending push notifications to devices.

## Related Resources

- [MDM Product Group Documentation](../../product-groups/mdm/) - Documentation for the MDM product group
- [MDM Development Guides](../../guides/mdm/) - Guides for MDM development