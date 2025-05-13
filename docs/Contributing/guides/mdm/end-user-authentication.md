# End User Authentication Guide

This guide provides instructions for developing End User Authentication functionality in Fleet's MDM.

## Introduction

End User Authentication in Fleet's MDM allows for associating managed devices with specific users, enabling user-based policies and personalized device management. This guide covers the development and implementation of end user authentication features.

## Prerequisites

Before you begin developing End User Authentication functionality, you should have:

- A development environment set up according to the [Building Fleet](../../getting-started/building-fleet.md) guide
- Basic understanding of authentication protocols (SAML, OAuth/OIDC, LDAP)
- Access to identity providers for testing (e.g., Okta, Azure AD, Google Workspace)
- Understanding of Fleet's MDM architecture

## Authentication Flows

Fleet supports multiple authentication flows for end users:

### User-Initiated Authentication

1. User accesses the Fleet Desktop application or web portal
2. User is redirected to the identity provider for authentication
3. After successful authentication, the user is redirected back to Fleet with an authentication token
4. Fleet associates the user with the device

### Automated Authentication

1. Device enrolls with the MDM server
2. MDM server retrieves user information from the identity provider
3. MDM server associates the user with the device

## Identity Provider Integration

### SAML Integration

SAML (Security Assertion Markup Language) is used for integration with identity providers like Okta and Azure AD:

1. Configure the SAML application in the identity provider
2. Configure Fleet to use the SAML identity provider
3. Test the SAML authentication flow

Example SAML configuration:

```yaml
saml:
  enabled: true
  idp_metadata_url: "https://idp.example.com/metadata"
  sp_entity_id: "https://fleet.example.com"
  acs_url: "https://fleet.example.com/api/v1/fleet/sso/callback"
  attribute_mapping:
    email: "email"
    name: "name"
    groups: "groups"
```

### OAuth/OIDC Integration

OAuth/OIDC (OpenID Connect) is used for integration with identity providers like Google Workspace:

1. Configure the OAuth application in the identity provider
2. Configure Fleet to use the OAuth identity provider
3. Test the OAuth authentication flow

Example OAuth configuration:

```yaml
oauth:
  enabled: true
  client_id: "your-client-id"
  client_secret: "your-client-secret"
  auth_url: "https://idp.example.com/auth"
  token_url: "https://idp.example.com/token"
  user_info_url: "https://idp.example.com/userinfo"
  scopes: ["openid", "email", "profile"]
  attribute_mapping:
    email: "email"
    name: "name"
    groups: "groups"
```

### LDAP Integration

LDAP (Lightweight Directory Access Protocol) is used for integration with directory services like Active Directory:

1. Configure Fleet to connect to the LDAP server
2. Configure the LDAP search and attribute mapping
3. Test the LDAP authentication flow

Example LDAP configuration:

```yaml
ldap:
  enabled: true
  server: "ldap://ldap.example.com:389"
  bind_dn: "cn=admin,dc=example,dc=com"
  bind_password: "password"
  user_search_base: "ou=users,dc=example,dc=com"
  user_search_filter: "(uid=%s)"
  group_search_base: "ou=groups,dc=example,dc=com"
  group_search_filter: "(memberUid=%s)"
  attribute_mapping:
    email: "mail"
    name: "cn"
    groups: "memberOf"
```

## User-Device Association

### Database Schema

The user-device association is stored in the Fleet database:

```sql
CREATE TABLE user_device_mappings (
  id INT AUTO_INCREMENT PRIMARY KEY,
  user_id INT NOT NULL,
  device_id INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (device_id) REFERENCES devices(id)
);
```

### API Endpoints

Fleet provides API endpoints for managing user-device associations:

- `GET /api/v1/fleet/devices/:id/user`: Get the user associated with a device
- `POST /api/v1/fleet/devices/:id/user`: Associate a user with a device
- `DELETE /api/v1/fleet/devices/:id/user`: Remove the user association from a device

### User Interface

Fleet provides UI components for managing user-device associations:

- Device details page with user information
- User details page with associated devices
- User search and selection for device association

## User-Based Policies

### Policy Definition

User-based policies are defined in Fleet's policy system:

```yaml
policy:
  name: "Example User Policy"
  description: "Example policy applied based on user attributes"
  scope: "user"
  conditions:
    user_groups: ["admin", "developer"]
  settings:
    - key: "allow_app_installation"
      value: true
    - key: "require_password"
      value: true
```

### Policy Enforcement

User-based policies are enforced through Fleet's MDM system:

1. User is authenticated and associated with a device
2. User attributes (groups, roles, etc.) are retrieved
3. Policies are evaluated based on user attributes
4. Matching policies are applied to the device

## Testing

### Manual Testing

1. Configure an identity provider for testing
2. Authenticate a user through the identity provider
3. Verify the user is associated with the device
4. Test user-based policies

### Automated Testing

Fleet includes automated tests for End User Authentication functionality:

```bash
# Run End User Authentication tests
go test -v ./server/service/user_auth_test.go
```

## Debugging

### Authentication Issues

- **Identity Provider Configuration**: Verify the identity provider is correctly configured
- **Attribute Mapping**: Check if user attributes are correctly mapped
- **Token Validation**: Ensure authentication tokens are properly validated

### User-Device Association Issues

- **Database Queries**: Check if user-device associations are correctly stored in the database
- **API Endpoints**: Verify the API endpoints for user-device associations are working correctly
- **UI Components**: Test the UI components for user-device associations

## Related Resources

- [End User Authentication Architecture](../../architecture/mdm/end-user-authentication.md)
- [MDM End User Authentication](../../product-groups/mdm/mdm-end-user-authentication.md)
- [SAML Documentation](https://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf)
- [OAuth/OIDC Documentation](https://openid.net/specs/openid-connect-core-1_0.html)
- [LDAP Documentation](https://tools.ietf.org/html/rfc4511)