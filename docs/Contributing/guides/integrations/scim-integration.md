# SCIM (System for Cross-domain Identity Management) integration

## Reference docs

- [scim.cloud](https://scim.cloud/)
- [SCIM: Core Schema (RFC7643)](https://datatracker.ietf.org/doc/html/rfc7643)
- [SCIM: Protocol (RFC7644)](https://datatracker.ietf.org/doc/html/rfc7644)
- [scim Go library](https://github.com/elimity-com/scim)

## Okta integration

- https://developer.okta.com/docs/guides/scim-provisioning-integration-prepare/main/

Sample provisioning settings that work. Capabilities can be disabled and attributes can be removed as needed.

![Okta to Fleet provisioning](../../assets/SCIM-Okta-provisioning.png)

From our testing with Okta, we see the following behavior that is worth noting:
- Okta does not use PATCH endpoint
- Okta does not DELETE users; if a new user needs to be created with the same username as a "deleted" user, then it overwrites the old user

### Automated test for Okta integration

First, create at least one SCIM user:

```
POST https://localhost:8080/api/latest/fleet/scim/Users
Authorization: Bearer <API key>
{
    "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
    "userName": "test.user@okta.local",
    "name": {
        "givenName": "Test",
        "familyName": "User"
    },
    "emails": [{
        "primary": true,
        "value": "test.user@okta.local",
        "type": "work"
    }],
    "active": true
}
```

Run test using [Runscope](https://www.runscope.com/). See [instructions](https://developer.okta.com/docs/guides/scim-provisioning-integration-prepare/main/#test-your-scim-api).

## Entra ID integration
- [SCIM guide](https://learn.microsoft.com/en-us/entra/identity/app-provisioning/use-scim-to-provision-users-and-groups)
- [SCIM validator](https://scimvalidator.microsoft.com/)
  - Note: only test attributes implemented by Fleet

By default, Entra ID SCIM client is not fully SCIM 2.0 compliant. [See details](https://learn.microsoft.com/en-us/entra/identity/app-provisioning/application-provisioning-config-problem-scim-compatibility). Fleet server does not support Entra ID's non-SCIM compliant client. To use the SCIM compliant Entra ID client, you must append the following URL parameter to the Fleet server's path: `aadOptscim062020`. This parameter is processed by Entra ID, not by Fleet. So, the Fleet URL should look like this:

```
https://<server_url>/api/v1/fleet/scim?aadOptscim062020
```

### Testing Entra ID integration

Use [scimvalidator.microsoft.com](https://scimvalidator.microsoft.com/). Only test the attributes that we have implemented.

We support the `emails` attribute, even though it is not called out in our customer-facing guide.

![SCIM-Entra-ID-Validator-User-attributes.png](../../assets/SCIM-Entra-ID-Validator-User-attributes.png)
![SCIM-Entra-ID-Validator-Group-attributes.png](../../assets/SCIM-Entra-ID-Validator-Group-attributes.png)

To see our supported attributes, check the schema:
```
GET https://localhost:8080/api/latest/fleet/scim/Schemas
```

Results (2025/05/06)

![SCIM-Entra-ID-Validator-results.png](../../assets/SCIM-Entra-ID-Validator-results.png)

## Authentication

We use same authentication as API. HTTP header: `Authorization: Bearer xyz`

## Diagrams

```mermaid
---
title: Initial DB schema (not kept up to date)
---
erDiagram
    HOST_SCIM_USER {
        host_id uint PK
        scim_user_id uint PK "FK"
    }
    SCIM_USERS {
        id uint PK
        external_id *string "Index"
        user_name string "Unique"
        given_name *string
        family_name *string
        active *bool
    }
    SCIM_USER_EMAILS {
        id uint PK
        scim_user_id uint FK
        type *string "Index"
        email string "Index"
        primary *bool
    }
    SCIM_USER_GROUP {
        scim_user_id string PK "FK"
        group_id uint PK "FK"
    }
    SCIM_GROUPS {
        id uint PK
        external_id *string "Index"
        display_name string "Index"
    }
    HOST_SCIM_USER }o--|| SCIM_USERS : "multiple hosts can have the same SCIM user"
    SCIM_USERS ||--o{ SCIM_USER_GROUP: "zero-to-many"
    SCIM_USER_GROUP }o--|| SCIM_GROUPS: "zero-to-many"
    SCIM_USERS ||--o{ SCIM_USER_EMAILS: "zero-to-many"
    COMMENT {
        _ _ "created_at and updated_at columns not shown"
    }
```

## Notes

- Okta and Entra ID do not support nested groups
