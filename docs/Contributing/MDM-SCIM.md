# SCIM (System for Cross-domain Identity Management) integration

## Reference docs

- [scim.cloud](https://scim.cloud/)
- [SCIM: Core Schema (RFC7643)](https://datatracker.ietf.org/doc/html/rfc7643)
- [SCIM: Protocol (RFC7644)](https://datatracker.ietf.org/doc/html/rfc7644)
- [scim Go library](https://github.com/elimity-com/scim)

### Okta integration

- https://developer.okta.com/docs/guides/scim-provisioning-integration-prepare/main/

### Entra ID integration
- [SCIM guide](https://learn.microsoft.com/en-us/entra/identity/app-provisioning/use-scim-to-provision-users-and-groups)
- [SCIM validator](https://scimvalidator.microsoft.com/)
  - Only test attributes that we implemented

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
        scim_user_id uint PK "FK"
        type *string PK
        email string PK "Index"
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
