# End user authentication architecture

This document provides an overview of Fleet's End User Authentication architecture for MDM.

## Introduction

End User Authentication in Fleet's MDM allows for associating managed devices with specific users, enabling user-based policies and personalized device management. This document provides insights into the design decisions, system components, and interactions specific to the End User Authentication functionality.

## Architecture overview

The End User Authentication architecture integrates with identity providers (IdPs) to authenticate users and associate them with their devices. This enables user-specific policies and configurations to be applied to devices.

## Key components

- **Identity Provider Integration**: Integration with external identity providers such as Okta, Azure AD, and Google Workspace.
- **User-Device Association**: Mapping between users and their devices.
- **Authentication Flow**: The process by which users authenticate and are associated with their devices.

## Architecture diagram

```
[Placeholder for End User Authentication Architecture Diagram]
```

## Authentication flows

### Automated authentication

1. New device starts the enrollment process and is prompted to login to the IDP.
2. Fleet captures the username and stores it to register after enrollment.
3. Device enrolls with the MDM server.
4. Optionally MDM server maps user information to SCIM data already sent by IdP.
5. MDM server associates the user with the device.

## Identity provider integration

Fleet integrates with various identity providers to authenticate users:

- **SAML**: For integration with SAML-based identity providers such as Okta and Azure AD.

## User-device association

User-device association is stored in the Fleet database and is used to apply user-specific policies and configurations to devices. The association can be established through:

- **Directory Integration**: When user information is retrieved from a directory service.
- **Automated Authentication**: When a user enrolls in MDM after authenticating with an IDP.

## Related resources

- [MDM Product Group Documentation](../../product-groups/mdm/) - Documentation for the MDM product group
- [MDM Development Guides](../../guides/mdm/) - Guides for MDM development
- [MDM End User Authentication](../../product-groups/mdm/mdm-end-user-authentication.md) - Detailed documentation on MDM end user authentication