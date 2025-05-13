# End User Authentication Architecture

This document provides an overview of Fleet's End User Authentication architecture for MDM.

## Introduction

End User Authentication in Fleet's MDM allows for associating managed devices with specific users, enabling user-based policies and personalized device management. This document provides insights into the design decisions, system components, and interactions specific to the End User Authentication functionality.

## Architecture Overview

The End User Authentication architecture integrates with identity providers (IdPs) to authenticate users and associate them with their devices. This enables user-specific policies and configurations to be applied to devices.

## Key Components

- **Identity Provider Integration**: Integration with external identity providers such as Okta, Azure AD, and Google Workspace.
- **User-Device Association**: Mapping between users and their devices.
- **User-Based Policies**: Policies that are applied based on the user's identity.
- **Authentication Flow**: The process by which users authenticate and are associated with their devices.

## Architecture Diagram

```
[Placeholder for End User Authentication Architecture Diagram]
```

## Authentication Flows

### User-Initiated Authentication

1. User accesses the Fleet Desktop application or web portal.
2. User is redirected to the identity provider for authentication.
3. After successful authentication, the user is redirected back to Fleet with an authentication token.
4. Fleet associates the user with the device.

### Automated Authentication

1. Device enrolls with the MDM server.
2. MDM server retrieves user information from the identity provider.
3. MDM server associates the user with the device.

## Identity Provider Integration

Fleet integrates with various identity providers to authenticate users:

- **SAML**: For integration with SAML-based identity providers such as Okta and Azure AD.
- **OAuth/OIDC**: For integration with OAuth/OIDC-based identity providers such as Google Workspace.
- **LDAP**: For integration with LDAP-based identity providers such as Active Directory.

## User-Device Association

User-device association is stored in the Fleet database and is used to apply user-specific policies and configurations to devices. The association can be established through:

- **User Authentication**: When a user authenticates on a device.
- **Directory Integration**: When user information is retrieved from a directory service.
- **Manual Assignment**: When an administrator manually assigns a user to a device.

## Related Resources

- [MDM Product Group Documentation](../../product-groups/mdm/) - Documentation for the MDM product group
- [MDM Development Guides](../../guides/mdm/) - Guides for MDM development
- [MDM End User Authentication](../../product-groups/mdm/mdm-end-user-authentication.md) - Detailed documentation on MDM end user authentication