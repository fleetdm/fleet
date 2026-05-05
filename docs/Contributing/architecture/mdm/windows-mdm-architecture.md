# Windows MDM architecture

This document provides an overview of Fleet's Windows Mobile Device Management (MDM) architecture.

## Introduction

Fleet's Windows MDM architecture is designed to manage Windows devices. This document provides insights into the design decisions, system components, and interactions specific to the Windows MDM functionality.

## Windows MDM protocol

Windows MDM is based on the OMA Device Management (OMA-DM) protocol and uses SyncML for data exchange. Microsoft has extended the protocol with Windows-specific features and capabilities.

### Key components

- **MDM Enrollment**: The process by which devices are registered with the MDM server.
- **MDM Sync**: The process by which devices communicate with the MDM server.
- **Configuration Service Providers (CSPs)**: Components that provide an interface to device settings.
- **MDM Policies**: Settings and configurations applied to managed devices.

## Architecture diagram

```
[Placeholder for Windows MDM Architecture Diagram]
```

## Enrollment flows

### User-initiated enrollment

1. User downloads and installs the enrollment package.
2. Device enrolls with the MDM server.
3. MDM server sends initial configuration profiles.

### Azure AD join

1. Device is joined to Azure AD.
2. Device automatically enrolls with the MDM server.

## Configuration Service Providers (CSPs)

Windows MDM uses Configuration Service Providers (CSPs) to configure and manage devices. CSPs are the interface through which the MDM server can access and modify device settings.

Common CSPs used by Fleet include:

- **Policy CSP**: For configuring device policies.
- **EnterpriseModernAppManagement CSP**: For managing modern applications.
- **DeviceStatus CSP**: For retrieving device status information.
- **WindowsUpdatePolicy CSP**: For configuring Windows Update settings.

## SyncML structure

Windows MDM uses SyncML for communication between the device and the MDM server. A typical SyncML message includes:

- **Header**: Contains session information.
- **Body**: Contains commands and data.
- **Status**: Contains the result of previous commands.

## Related resources

- [MDM Product Group Documentation](../../product-groups/mdm/) - Documentation for the MDM product group
- [MDM Development Guides](../../guides/mdm/) - Guides for MDM development
- [Windows MDM Glossary and Protocol](../../product-groups/mdm/windows-mdm-glossary-and-protocol.md) - Glossary of Windows MDM terms and protocol details