# REST API

- [Authentication](#authentication)
- [Activities](#activities)
- [Fleet configuration](#fleet-configuration)
- [File carving](#file-carving)
- [Hosts](#hosts)
- [Labels](#labels)
- [Mobile device management (MDM)](#mobile-device-management-mdm)
- [Policies](#policies)
- [Queries](#queries)
- [Schedule (deprecated)](#schedule)
- [Scripts](#scripts)
- [Sessions](#sessions)
- [Software](#software)
- [Targets](#targets)
- [Teams](#teams)
- [Translator](#translator)
- [Users](#users)
- [API errors](#api-responses)

Use the Fleet APIs to automate Fleet.

This page includes a list of available resources and their API routes.

## Authentication

- [Retrieve your API token](#retrieve-your-api-token)
- [Log in](#log-in)
- [Log out](#log-out)
- [Forgot password](#forgot-password)
- [Change password](#change-password)
- [Reset password](#reset-password)
- [Me](#me)
- [SSO config](#sso-config)
- [Initiate SSO](#initiate-sso)
- [SSO callback](#sso-callback)

### Retrieve your API token

All API requests to the Fleet server require API token authentication unless noted in the documentation. API tokens are tied to your Fleet user account.

To get an API token, retrieve it from "My account" > "Get API token" in the Fleet UI (`/profile`). Or, you can send a request to the [login API endpoint](#log-in) to get your token.

Then, use that API token to authenticate all subsequent API requests by sending it in the "Authorization" request header, prefixed with "Bearer ":

```http
Authorization: Bearer <your token>
```

> For SSO users, email/password login is disabled. The API token can instead be retrieved from the "My account" page in the UI (/profile). On this page, choose "Get API token".

### Log in

Authenticates the user with the specified credentials. Use the token returned from this endpoint to authenticate further API requests.

`POST /api/v1/fleet/login`

> This API endpoint is not available to SSO users, since email/password login is disabled for SSO users. To get an API token for an SSO user, you can use the Fleet UI.

#### Parameters

| Name     | Type   | In   | Description                                   |
| -------- | ------ | ---- | --------------------------------------------- |
| email    | string | body | **Required**. The user's email.               |
| password | string | body | **Required**. The user's plain text password. |

#### Example

`POST /api/v1/fleet/login`

##### Request body

```json
{
  "email": "janedoe@example.com",
  "password": "VArCjNW7CfsxGp67"
}
```

##### Default response

`Status: 200`

```json
{
  "user": {
    "created_at": "2020-11-13T22:57:12Z",
    "updated_at": "2020-11-13T22:57:12Z",
    "id": 1,
    "name": "Jane Doe",
    "email": "janedoe@example.com",
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false,
    "global_role": "admin",
    "teams": []
  },
  "token": "{your token}"
}
```

##### Authentication failed

`Status: 401 Unauthorized`

```json
{
  "message": "Authentication failed",
  "errors": [
    {
      "name": "base",
      "reason": "Authentication failed"
    }
  ],
  "uuid": "1272014b-902b-4b36-bcdb-75fde5eac1fc"
}
```

##### Too many requests / Rate limiting

`Status: 429 Too Many Requests`
`Header: retry-after: N`

> This response includes a header `retry-after` that indicates how many more seconds you are blocked before you can try again.

```json
{
  "message": "limit exceeded, retry after: Ns",
  "errors": [
    {
      "name": "base",
      "reason": "limit exceeded, retry after: Ns"
    }
  ]
}
```

---

### Log out

Logs out the authenticated user.

`POST /api/v1/fleet/logout`

#### Example

`POST /api/v1/fleet/logout`

##### Default response

`Status: 200`

---

### Forgot password

Sends a password reset email to the specified email. Requires that SMTP or SES is configured for your Fleet server.

`POST /api/v1/fleet/forgot_password`

#### Parameters

| Name  | Type   | In   | Description                                                             |
| ----- | ------ | ---- | ----------------------------------------------------------------------- |
| email | string | body | **Required**. The email of the user requesting the reset password link. |

#### Example

`POST /api/v1/fleet/forgot_password`

##### Request body

```json
{
  "email": "janedoe@example.com"
}
```

##### Default response

`Status: 200`

##### Unknown error

`Status: 500`

```json
{
  "message": "Unknown Error",
  "errors": [
    {
      "name": "base",
      "reason": "email not configured"
    }
  ]
}
```

---

### Change password

`POST /api/v1/fleet/change_password`

Changes the password for the authenticated user.

#### Parameters

| Name         | Type   | In   | Description                            |
| ------------ | ------ | ---- | -------------------------------------- |
| old_password | string | body | **Required**. The user's old password. |
| new_password | string | body | **Required**. The user's new password. |

#### Example

`POST /api/v1/fleet/change_password`

##### Request body

```json
{
  "old_password": "VArCjNW7CfsxGp67",
  "new_password": "zGq7mCLA6z4PzArC"
}
```

##### Default response

`Status: 200`

##### Validation failed

`Status: 422 Unprocessable entity`

```json
{
  "message": "Validation Failed",
  "errors": [
    {
      "name": "old_password",
      "reason": "old password does not match"
    }
  ]
}
```

### Reset password

Resets a user's password. Which user is determined by the password reset token used. The password reset token can be found in the password reset email sent to the desired user.

`POST /api/v1/fleet/reset_password`

#### Parameters

| Name                      | Type   | In   | Description                                                               |
| ------------------------- | ------ | ---- | ------------------------------------------------------------------------- |
| new_password              | string | body | **Required**. The new password.                                           |
| new_password_confirmation | string | body | **Required**. Confirmation for the new password.                          |
| password_reset_token      | string | body | **Required**. The token provided to the user in the password reset email. |

#### Example

`POST /api/v1/fleet/reset_password`

##### Request body

```json
{
  "new_password": "abc123",
  "new_password_confirmation": "abc123",
  "password_reset_token": "UU5EK0JhcVpsRkY3NTdsaVliMEZDbHJ6TWdhK3oxQ1Q="
}
```

##### Default response

`Status: 200`


---

### Me

Retrieves the user data for the authenticated user.

`GET /api/v1/fleet/me`

#### Example

`GET /api/v1/fleet/me`

##### Default response

`Status: 200`

```json
{
  "user": {
    "created_at": "2020-11-13T22:57:12Z",
    "updated_at": "2020-11-16T23:49:41Z",
    "id": 1,
    "name": "Jane Doe",
    "email": "janedoe@example.com",
    "global_role": "admin",
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false,
    "teams": []
  }
}
```

---

### Perform required password reset

Resets the password of the authenticated user. Requires that `force_password_reset` is set to `true` prior to the request.

`POST /api/v1/fleet/perform_required_password_reset`

#### Example

`POST /api/v1/fleet/perform_required_password_reset`

##### Request body

```json
{
  "new_password": "sdPz8CV5YhzH47nK"
}
```

##### Default response

`Status: 200`

```json
{
  "user": {
    "created_at": "2020-11-13T22:57:12Z",
    "updated_at": "2020-11-17T00:09:23Z",
    "id": 1,
    "name": "Jane Doe",
    "email": "janedoe@example.com",
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false,
    "global_role": "admin",
    "teams": []
  }
}
```

---

### SSO config

Gets the current SSO configuration.

`GET /api/v1/fleet/sso`

#### Example

`GET /api/v1/fleet/sso`

##### Default response

`Status: 200`

```json
{
  "settings": {
    "idp_name": "IDP Vendor 1",
    "idp_image_url": "",
    "sso_enabled": false
  }
}
```

---

### Initiate SSO

`POST /api/v1/fleet/sso`

#### Parameters

| Name      | Type   | In   | Description                                                                 |
| --------- | ------ | ---- | --------------------------------------------------------------------------- |
| relay_url | string | body | **Required**. The relative url to be navigated to after successful sign in. |

#### Example

`POST /api/v1/fleet/sso`

##### Request body

```json
{
  "relay_url": "/hosts/manage"
}
```

##### Default response

`Status: 200`

##### Unknown error

`Status: 500`

```json
{
  "message": "Unknown Error",
  "errors": [
    {
      "name": "base",
      "reason": "InitiateSSO getting metadata: Get \"https://idp.example.org/idp-meta.xml\": dial tcp: lookup idp.example.org on [2001:558:feed::1]:53: no such host"
    }
  ]
}
```

### SSO callback

This is the callback endpoint that the identity provider will use to send security assertions to Fleet. This is where Fleet receives and processes the response from the identify provider.

`POST /api/v1/fleet/sso/callback`

#### Parameters

| Name         | Type   | In   | Description                                                 |
| ------------ | ------ | ---- | ----------------------------------------------------------- |
| SAMLResponse | string | body | **Required**. The SAML response from the identity provider. |

#### Example

`POST /api/v1/fleet/sso/callback`

##### Request body

```json
{
  "SAMLResponse": "<SAML response from IdP>"
}
```

##### Default response

`Status: 200`


---

## Activities

### List activities

Returns a list of the activities that have been performed in Fleet as well as additional metadata.
for pagination. For a comprehensive list of activity types and detailed information, please see the [audit logs](https://fleetdm.com/docs/using-fleet/audit-activities) page.

`GET /api/v1/fleet/activities`

#### Parameters

| Name            | Type    | In    | Description                                                 |
|:--------------- |:------- |:----- |:------------------------------------------------------------|
| page            | integer | query | Page number of the results to fetch.                                                                                          |
| per_page        | integer | query | Results per page.                                                                                                             |
| order_key       | string  | query | What to order results by. Can be any column in the `activites` table.                                                         |
| order_direction | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |

#### Example

`GET /api/v1/fleet/activities?page=0&per_page=10&order_key=created_at&order_direction=desc`

##### Default response

```json
{
  "activities": [
    {
      "created_at": "2021-07-30T13:41:07Z",
      "id": 24,
      "actor_full_name": "name",
      "actor_id": 1,
      "actor_gravatar": "",
      "actor_email": "name@example.com",
      "type": "live_query",
      "details": {
        "targets_count": 231
      }
    },
    {
      "created_at": "2021-07-29T15:35:33Z",
      "id": 23,
      "actor_full_name": "name",
      "actor_id": 1,
      "actor_gravatar": "",
      "actor_email": "name@example.com",
      "type": "deleted_multiple_saved_query",
      "details": {
        "query_ids": [
          2,
          24,
          25
        ]
      }
    },
    {
      "created_at": "2021-07-29T14:40:30Z",
      "id": 22,
      "actor_full_name": "name",
      "actor_id": 1,
      "actor_gravatar": "",
      "actor_email": "name@example.com",
      "type": "created_team",
      "details": {
        "team_id": 3,
        "team_name": "Oranges"
      }
    },
    {
      "created_at": "2021-07-29T14:40:27Z",
      "id": 21,
      "actor_full_name": "name",
      "actor_id": 1,
      "actor_gravatar": "",
      "actor_email": "name@example.com",
      "type": "created_team",
      "details": {
        "team_id": 2,
        "team_name": "Apples"
      }
    },
    {
      "created_at": "2021-07-27T14:35:08Z",
      "id": 20,
      "actor_full_name": "name",
      "actor_id": 1,
      "actor_gravatar": "",
      "actor_email": "name@example.com",
      "type": "created_pack",
      "details": {
        "pack_id": 2,
        "pack_name": "New pack"
      }
    },
    {
      "created_at": "2021-07-27T13:25:21Z",
      "id": 19,
      "actor_full_name": "name",
      "actor_id": 1,
      "actor_gravatar": "",
      "actor_email": "name@example.com",
      "type": "live_query",
      "details": {
        "targets_count": 14
      }
    },
    {
      "created_at": "2021-07-27T13:25:14Z",
      "id": 18,
      "actor_full_name": "name",
      "actor_id": 1,
      "actor_gravatar": "",
      "actor_email": "name@example.com",
      "type": "live_query",
      "details": {
        "targets_count": 14
      }
    },
    {
      "created_at": "2021-07-26T19:28:24Z",
      "id": 17,
      "actor_full_name": "name",
      "actor_id": 1,
      "actor_gravatar": "",
      "actor_email": "name@example.com",
      "type": "live_query",
      "details": {
        "target_counts": 1
      }
    },
    {
      "created_at": "2021-07-26T17:27:37Z",
      "id": 16,
      "actor_full_name": "name",
      "actor_id": 1,
      "actor_gravatar": "",
      "actor_email": "name@example.com",
      "type": "live_query",
      "details": {
        "target_counts": 14
      }
    },
    {
      "created_at": "2021-07-26T17:27:08Z",
      "id": 15,
      "actor_full_name": "name",
      "actor_id": 1,
      "actor_gravatar": "",
      "actor_email": "name@example.com",
      "type": "live_query",
      "details": {
        "target_counts": 14
      }
    }
  ],
  "meta": {
    "has_next_results": true,
    "has_previous_results": false
  }
}

```

---

## File carving

- [List carves](#list-carves)
- [Get carve](#get-carve)
- [Get carve block](#get-carve-block)

Fleet supports osquery's file carving functionality as of Fleet 3.3.0. This allows the Fleet server to request files (and sets of files) from osquery agents, returning the full contents to Fleet.

To initiate a file carve using the Fleet API, you can use the [live query](#run-live-query) endpoint to run a query against the `carves` table.

For more information on executing a file carve in Fleet, go to the [File carving with Fleet docs](https://fleetdm.com/docs/using-fleet/fleetctl-cli#file-carving-with-fleet).

### List carves

Retrieves a list of the non expired carves. Carve contents remain available for 24 hours after the first data is provided from the osquery client.

`GET /api/v1/fleet/carves`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/carves`

##### Default response

`Status: 200`

```json
{
  "carves": [
    {
      "id": 1,
      "created_at": "2021-02-23T22:52:01Z",
      "host_id": 7,
      "name": "macbook-pro.local-2021-02-23T22:52:01Z-fleet_distributed_query_30",
      "block_count": 1,
      "block_size": 2000000,
      "carve_size": 2048,
      "carve_id": "c6958b5f-4c10-4dc8-bc10-60aad5b20dc8",
      "request_id": "fleet_distributed_query_30",
      "session_id": "065a1dc3-40ad-441c-afff-80c2ad7dac28",
      "expired": false,
      "max_block": 0
    },
    {
      "id": 2,
      "created_at": "2021-02-23T22:53:03Z",
      "host_id": 7,
      "name": "macbook-pro.local-2021-02-23T22:53:03Z-fleet_distributed_query_31",
      "block_count": 2,
      "block_size": 2000000,
      "carve_size": 3400704,
      "carve_id": "2b9170b9-4e11-4569-a97c-2f18d18bec7a",
      "request_id": "fleet_distributed_query_31",
      "session_id": "f73922ed-40a4-4e98-a50a-ccda9d3eb755",
      "expired": false,
      "max_block": 1,
      "error": "S3 multipart carve upload: EntityTooSmall: Your proposed upload is smaller than the minimum allowed object size"
    }
  ]
}
```

### Get carve

Retrieves the specified carve.

`GET /api/v1/fleet/carves/{id}`

#### Parameters

| Name | Type    | In   | Description                           |
| ---- | ------- | ---- | ------------------------------------- |
| id   | integer | path | **Required.** The desired carve's ID. |

#### Example

`GET /api/v1/fleet/carves/1`

##### Default response

`Status: 200`

```json
{
  "carve": {
    "id": 1,
    "created_at": "2021-02-23T22:52:01Z",
    "host_id": 7,
    "name": "macbook-pro.local-2021-02-23T22:52:01Z-fleet_distributed_query_30",
    "block_count": 1,
    "block_size": 2000000,
    "carve_size": 2048,
    "carve_id": "c6958b5f-4c10-4dc8-bc10-60aad5b20dc8",
    "request_id": "fleet_distributed_query_30",
    "session_id": "065a1dc3-40ad-441c-afff-80c2ad7dac28",
    "expired": false,
    "max_block": 0
  }
}
```

### Get carve block

Retrieves the specified carve block. This endpoint retrieves the data that was carved.

`GET /api/v1/fleet/carves/{id}/block/{block_id}`

#### Parameters

| Name     | Type    | In   | Description                                 |
| -------- | ------- | ---- | ------------------------------------------- |
| id       | integer | path | **Required.** The desired carve's ID.       |
| block_id | integer | path | **Required.** The desired carve block's ID. |

#### Example

`GET /api/v1/fleet/carves/1/block/0`

##### Default response

`Status: 200`

```json
{
    "data": "aG9zdHMAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA..."
}
```
---

## Fleet configuration

- [Get certificate](#get-certificate)
- [Get configuration](#get-configuration)
- [Modify configuration](#modify-configuration)
- [Get global enroll secrets](#get-global-enroll-secrets)
- [Modify global enroll secrets](#modify-global-enroll-secrets)
- [Get enroll secrets for a team](#get-enroll-secrets-for-a-team)
- [Modify enroll secrets for a team](#modify-enroll-secrets-for-a-team)
- [Create invite](#create-invite)
- [List invites](#list-invites)
- [Delete invite](#delete-invite)
- [Verify invite](#verify-invite)
- [Update invite](#update-invite)
- [Version](#version)

The Fleet server exposes a handful of API endpoints that handle the configuration of Fleet as well as endpoints that manage invitation and enroll secret operations. All the following endpoints require prior authentication meaning you must first log in successfully before calling any of the endpoints documented below.

### Get certificate

Returns the Fleet certificate.

`GET /api/v1/fleet/config/certificate`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/config/certificate`

##### Default response

`Status: 200`

```json
{
  "certificate_chain": <certificate_chain>
}
```

### Get configuration

Returns all information about the Fleet's configuration.

> NOTE: The `agent_options`, `sso_settings` and `smtp_settings` fields are only returned to Global Admin users.

`GET /api/v1/fleet/config`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/config`

##### Default response

`Status: 200`

```json
{
  "org_info": {
    "org_name": "fleet",
    "org_logo_url": "",
    "contact_url": "https://fleetdm.com/company/contact"
  },
  "server_settings": {
    "server_url": "https://localhost:8080",
    "live_query_disabled": false,
    "enable_analytics": true
  },
  "smtp_settings": {
    "enable_smtp": false,
    "configured": false,
    "sender_address": "",
    "server": "",
    "port": 587,
    "authentication_type": "authtype_username_password",
    "user_name": "",
    "password": "********",
    "enable_ssl_tls": true,
    "authentication_method": "authmethod_plain",
    "domain": "",
    "verify_ssl_certs": true,
    "enable_start_tls": true
  },
  "sso_settings": {
    "entity_id": "",
    "issuer_uri": "",
    "idp_image_url": "",
    "metadata": "",
    "metadata_url": "",
    "idp_name": "",
    "enable_sso": false,
    "enable_sso_idp_login": false,
    "enable_jit_provisioning": false
  },
  "host_expiry_settings": {
    "host_expiry_enabled": false,
    "host_expiry_window": 0
  },
  "features": {
    "additional_queries": null
  },
  "mdm": {
    "apple_bm_default_team": "",
    "apple_bm_terms_expired": false,
    "enabled_and_configured": true,
    "windows_enabled_and_configured": true,
    "macos_updates": {
      "minimum_version": "12.3.1",
      "deadline": "2022-01-01"
    },
    "macos_settings": {
      "custom_settings": ["path/to/profile1.mobileconfig"],
      "enable_disk_encryption": true
    },
    "end_user_authentication": {
      "entity_id": "",
      "issuer_uri": "",
      "metadata": "",
      "metadata_url": "",
      "idp_name": ""
    },
    "macos_migration": {
      "enable": false,
      "mode": "voluntary",
      "webhook_url": "https://webhook.example.com"
    },
    "macos_setup": {
      "bootstrap_package": "",
      "enable_end_user_authentication": false,
      "macos_setup_assistant": "path/to/config.json"
    }
  },
  "agent_options": {
    "spec": {
      "config": {
        "options": {
          "pack_delimiter": "/",
          "logger_tls_period": 10,
          "distributed_plugin": "tls",
          "disable_distributed": false,
          "logger_tls_endpoint": "/api/v1/osquery/log",
          "distributed_interval": 10,
          "distributed_tls_max_attempts": 3
        },
        "decorators": {
          "load": [
            "SELECT uuid AS host_uuid FROM system_info;",
            "SELECT hostname AS hostname FROM system_info;"
          ]
        }
      },
      "overrides": {},
      "command_line_flags": {}
    }
  },
  "license": {
     "tier": "free",
     "expiration": "0001-01-01T00:00:00Z"
   },
  "logging": {
      "debug": false,
      "json": false,
      "result": {
          "plugin": "firehose",
          "config": {
              "region": "us-east-1",
              "status_stream": "",
              "result_stream": "result-topic"
          }
      },
      "status": {
          "plugin": "filesystem",
          "config": {
              "status_log_file": "foo_status",
              "result_log_file": "",
              "enable_log_rotation": false,
              "enable_log_compression": false
          }
      }
  },
  "vulnerability_settings": {
    "databases_path": ""
  },
  "webhook_settings": {
    "host_status_webhook": {
      "enable_host_status_webhook": true,
      "destination_url": "https://server.com",
      "host_percentage": 5,
      "days_count": 7
    },
    "failing_policies_webhook":{
      "enable_failing_policies_webhook":true,
      "destination_url": "https://server.com",
      "policy_ids": [1, 2, 3],
      "host_batch_size": 1000
    },
    "vulnerabilities_webhook":{
      "enable_vulnerabilities_webhook":true,
      "destination_url": "https://server.com",
      "host_batch_size": 1000
    }
  },
  "integrations": {
    "jira": null
  },
  "logging": {
    "debug": false,
    "json": false,
    "result": {
        "plugin": "filesystem",
        "config": {
          "status_log_file": "/var/folders/xh/bxm1d2615tv3vrg4zrxq540h0000gn/T/osquery_status",
          "result_log_file": "/var/folders/xh/bxm1d2615tv3vrg4zrxq540h0000gn/T/osquery_result",
          "enable_log_rotation": false,
          "enable_log_compression": false
        }
      },
    "status": {
      "plugin": "filesystem",
      "config": {
        "status_log_file": "/var/folders/xh/bxm1d2615tv3vrg4zrxq540h0000gn/T/osquery_status",
        "result_log_file": "/var/folders/xh/bxm1d2615tv3vrg4zrxq540h0000gn/T/osquery_result",
        "enable_log_rotation": false,
        "enable_log_compression": false
      }
    }
  },
  "update_interval": {
    "osquery_detail": 3600000000000,
    "osquery_policy": 3600000000000
  },
  "vulnerabilities": {
    "cpe_database_url": "",
    "current_instance_checks": "auto",
    "cve_feed_prefix_url": "",
    "databases_path": "",
    "disable_data_sync": false,
    "periodicity": 3600000000000,
    "recent_vulnerability_max_age": 2592000000000000
  }
}
```

### Modify configuration

Modifies the Fleet's configuration with the supplied information.

`PATCH /api/v1/fleet/config`

#### Parameters

| Name                              | Type    | In    | Description                                                                                                                                                                            |
| ---------------------             | ------- | ----  | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| org_name                          | string  | body  | _Organization information_. The organization name.                                                                                                                                     |
| org_logo_url                      | string  | body  | _Organization information_. The URL for the organization logo.                                                                                                                         |
| org_logo_url_light_background     | string  | body  | _Organization information_. The URL for the organization logo displayed in Fleet on top of light backgrounds.                                                                          |
| contact_url                       | string  | body  | _Organization information_. A URL that can be used by end users to contact the organization.                                                                                          |
| server_url                        | string  | body  | _Server settings_. The Fleet server URL.                                                                                                                                               |
| live_query_disabled               | boolean | body  | _Server settings_. Whether the live query capabilities are disabled.                                                                                                                   |
| enable_smtp                       | boolean | body  | _SMTP settings_. Whether SMTP is enabled for the Fleet app.                                                                                                                            |
| sender_address                    | string  | body  | _SMTP settings_. The sender email address for the Fleet app. An invitation email is an example of the emails that may use this sender address                                          |
| server                            | string  | body  | _SMTP settings_. The SMTP server for the Fleet app.                                                                                                                                    |
| port                              | integer | body  | _SMTP settings_. The SMTP port for the Fleet app.                                                                                                                                      |
| authentication_type               | string  | body  | _SMTP settings_. The authentication type used by the SMTP server. Options include `"authtype_username_and_password"` or `"none"`                                                       |
| username_name                     | string  | body  | _SMTP settings_. The username used to authenticate requests made to the SMTP server.                                                                                                   |
| password                          | string  | body  | _SMTP settings_. The password used to authenticate requests made to the SMTP server.                                                                                                   |
| enable_ssl_tls                    | boolean | body  | _SMTP settings_. Whether or not SSL and TLS are enabled for the SMTP server.                                                                                                           |
| authentication_method             | string  | body  | _SMTP settings_. The authentication method used to make authenticate requests to SMTP server. Options include `"authmethod_plain"`, `"authmethod_cram_md5"`, and `"authmethod_login"`. |
| domain                            | string  | body  | _SMTP settings_. The domain for the SMTP server.                                                                                                                                       |
| verify_ssl_certs                  | boolean | body  | _SMTP settings_. Whether or not SSL certificates are verified by the SMTP server. Turn this off (not recommended) if you use a self-signed certificate.                                |
| enabled_start_tls                 | boolean | body  | _SMTP settings_. Detects if STARTTLS is enabled in your SMTP server and starts to use it.                                                                                              |
| enabled_sso                       | boolean | body  | _SSO settings_. Whether or not SSO is enabled for the Fleet application. If this value is true, you must also include most of the SSO settings parameters below.                       |
| entity_id                         | string  | body  | _SSO settings_. The required entity ID is a URI that you use to identify Fleet when configuring the identity provider.                                                                 |
| issuer_uri                        | string  | body  | _SSO settings_. The URI you provide here must exactly match the Entity ID field used in the identity provider configuration.                                                           |
| idp_image_url                     | string  | body  | _SSO settings_. An optional link to an image such as a logo for the identity provider.                                                                                                 |
| metadata                          | string  | body  | _SSO settings_. Metadata provided by the identity provider. Either metadata or a metadata URL must be provided.                                                                        |
| metadata_url                      | string  | body  | _SSO settings_. A URL that references the identity provider metadata. If available from the identity provider, this is the preferred means of providing metadata.                      |
| host_expiry_enabled               | boolean | body  | _Host expiry settings_. When enabled, allows automatic cleanup of hosts that have not communicated with Fleet in some number of days.                                                  |
| host_expiry_window                | integer | body  | _Host expiry settings_. If a host has not communicated with Fleet in the specified number of days, it will be removed.                                                                 |
| agent_options                     | objects | body  | The agent_options spec that is applied to all hosts. In Fleet 4.0.0 the `api/v1/fleet/spec/osquery_options` endpoints were removed.                                                    |
| transparency_url                  | string  | body  | _Fleet Desktop_. The URL used to display transparency information to users of Fleet Desktop. **Requires Fleet Premium license**                                                           |
| enable_host_status_webhook        | boolean | body  | _webhook_settings.host_status_webhook settings_. Whether or not the host status webhook is enabled.                                                                 |
| destination_url                   | string  | body  | _webhook_settings.host_status_webhook settings_. The URL to deliver the webhook request to.                                                     |
| host_percentage                   | integer | body  | _webhook_settings.host_status_webhook settings_. The minimum percentage of hosts that must fail to check in to Fleet in order to trigger the webhook request.                                                              |
| days_count                        | integer | body  | _webhook_settings.host_status_webhook settings_. The minimum number of days that the configured `host_percentage` must fail to check in to Fleet in order to trigger the webhook request.                                |
| enable_failing_policies_webhook   | boolean | body  | _webhook_settings.failing_policies_webhook settings_. Whether or not the failing policies webhook is enabled. |
| destination_url                   | string  | body  | _webhook_settings.failing_policies_webhook settings_. The URL to deliver the webhook requests to.                                                     |
| policy_ids                        | array   | body  | _webhook_settings.failing_policies_webhook settings_. List of policy IDs to enable failing policies webhook.                                                              |
| host_batch_size                   | integer | body  | _webhook_settings.failing_policies_webhook settings_. Maximum number of hosts to batch on failing policy webhook requests. The default, 0, means no batching (all hosts failing a policy are sent on one request). |
| enable_vulnerabilities_webhook    | boolean | body  | _webhook_settings.vulnerabilities_webhook settings_. Whether or not the vulnerabilities webhook is enabled. |
| destination_url                   | string  | body  | _webhook_settings.vulnerabilities_webhook settings_. The URL to deliver the webhook requests to.                                                     |
| host_batch_size                   | integer | body  | _webhook_settings.vulnerabilities_webhook settings_. Maximum number of hosts to batch on vulnerabilities webhook requests. The default, 0, means no batching (all vulnerable hosts are sent on one request). |
| enable_software_vulnerabilities   | boolean | body  | _integrations.jira[] settings_. Whether or not Jira integration is enabled for software vulnerabilities. Only one vulnerability automation can be enabled at a given time (enable_vulnerabilities_webhook and enable_software_vulnerabilities). |
| enable_failing_policies           | boolean | body  | _integrations.jira[] settings_. Whether or not Jira integration is enabled for failing policies. Only one failing policy automation can be enabled at a given time (enable_failing_policies_webhook and enable_failing_policies). |
| url                               | string  | body  | _integrations.jira[] settings_. The URL of the Jira server to integrate with. |
| username                          | string  | body  | _integrations.jira[] settings_. The Jira username to use for this Jira integration. |
| api_token                         | string  | body  | _integrations.jira[] settings_. The API token of the Jira username to use for this Jira integration. |
| project_key                       | string  | body  | _integrations.jira[] settings_. The Jira project key to use for this integration. Jira tickets will be created in this project. |
| enable_software_vulnerabilities   | boolean | body  | _integrations.zendesk[] settings_. Whether or not Zendesk integration is enabled for software vulnerabilities. Only one vulnerability automation can be enabled at a given time (enable_vulnerabilities_webhook and enable_software_vulnerabilities). |
| enable_failing_policies           | boolean | body  | _integrations.zendesk[] settings_. Whether or not Zendesk integration is enabled for failing policies. Only one failing policy automation can be enabled at a given time (enable_failing_policies_webhook and enable_failing_policies). |
| url                               | string  | body  | _integrations.zendesk[] settings_. The URL of the Zendesk server to integrate with. |
| email                             | string  | body  | _integrations.zendesk[] settings_. The Zendesk user email to use for this Zendesk integration. |
| api_token                         | string  | body  | _integrations.zendesk[] settings_. The Zendesk API token to use for this Zendesk integration. |
| group_id                          | integer | body  | _integrations.zendesk[] settings_. The Zendesk group id to use for this integration. Zendesk tickets will be created in this group. |
| apple_bm_default_team             | string  | body  | _mdm settings_. The default team to use with Apple Business Manager. **Requires Fleet Premium license** |
| windows_enabled_and_configured    | boolean | body  | _mdm settings_. Enables Windows MDM support. |
| minimum_version                   | string  | body  | _mdm.macos_updates settings_. Hosts that belong to no team and are enrolled into Fleet's MDM will be nudged until their macOS is at or above this version. **Requires Fleet Premium license** |
| deadline                          | string  | body  | _mdm.macos_updates settings_. Hosts that belong to no team and are enrolled into Fleet's MDM won't be able to dismiss the Nudge window once this deadline is past. **Requires Fleet Premium license** |
| enable                          | boolean  | body  | _mdm.macos_migration settings_. Whether to enable the end user migration workflow for devices migrating from your old MDM solution. **Requires Fleet Premium license** |
| mode                          | string  | body  | _mdm.macos_migration settings_. The end user migration workflow mode for devices migrating from your old MDM solution. Options are `"voluntary"` or `"forced"`. **Requires Fleet Premium license** |
| webhook_url                          | string  | body  | _mdm.macos_migration settings_. The webhook url configured to receive requests to unenroll devices migrating from your old MDM solution. **Requires Fleet Premium license** |
| custom_settings                   | list    | body  | _mdm.macos_settings settings_. Hosts that belong to no team and are enrolled into Fleet's MDM will have those custom profiles applied. |
| enable_disk_encryption            | boolean | body  | _mdm.macos_settings settings_. Hosts that belong to no team and are enrolled into Fleet's MDM will have disk encryption enabled if set to true. **Requires Fleet Premium license** |
| enable_end_user_authentication            | boolean | body  | _mdm.macos_setup settings_. If set to true, end user authentication will be required during automatic MDM enrollment of new macOS devices. Settings for your IdP provider must also be [configured](https://fleetdm.com/docs/using-fleet/mdm-macos-setup#end-user-authentication). **Requires Fleet Premium license** |
| additional_queries                | boolean | body  | Whether or not additional queries are enabled on hosts.                                                                                                                                |
| force                             | bool    | query | Force apply the agent options even if there are validation errors.                                                                                                 |
| dry_run                           | bool    | query | Validate the configuration and return any validation errors, but do not apply the changes.                                                                         |

Note that when making changes to the `integrations` object, all integrations must be provided (not just the one being modified). This is because the endpoint will consider missing integrations as deleted.

#### Example

`PATCH /api/v1/fleet/config`

##### Request body

```json
{
  "org_info": {
    "org_name": "Fleet Device Management",
    "org_logo_url": "https://fleetdm.com/logo.png"
  },
  "smtp_settings": {
    "enable_smtp": true,
    "server": "localhost",
    "port": "1025"
  }
}
```

##### Default response

`Status: 200`

```json
{
  "org_info": {
    "org_name": "Fleet Device Management",
    "org_logo_url": "https://fleetdm.com/logo.png",
    "org_logo_url_light_background": "https://fleetdm.com/logo-light.png",
    "contact_url": "https://fleetdm.com/company/contact"
  },
  "server_settings": {
    "server_url": "https://localhost:8080",
    "live_query_disabled": false
  },
  "smtp_settings": {
    "enable_smtp": true,
    "configured": true,
    "sender_address": "",
    "server": "localhost",
    "port": 1025,
    "authentication_type": "authtype_username_none",
    "user_name": "",
    "password": "********",
    "enable_ssl_tls": true,
    "authentication_method": "authmethod_plain",
    "domain": "",
    "verify_ssl_certs": true,
    "enable_start_tls": true
  },
  "sso_settings": {
    "entity_id": "",
    "issuer_uri": "",
    "idp_image_url": "",
    "metadata": "",
    "metadata_url": "",
    "idp_name": "",
    "enable_sso": false
  },
  "host_expiry_settings": {
    "host_expiry_enabled": false,
    "host_expiry_window": 0
  },
  "features": {
    "additional_queries": null
  },
  "license": {
    "tier": "free",
    "expiration": "0001-01-01T00:00:00Z"
  },
  "mdm": {
    "apple_bm_default_team": "",
    "apple_bm_terms_expired": false,
    "apple_bm_enabled_and_configured": false,
    "enabled_and_configured": false,
    "windows_enabled_and_configured": false,
    "macos_updates": {
      "minimum_version": "12.3.1",
      "deadline": "2022-01-01"
    },
    "macos_settings": {
      "custom_settings": ["path/to/profile1.mobileconfig"],
      "enable_disk_encryption": true
    },
    "end_user_authentication": {
      "entity_id": "",
      "issuer_uri": "",
      "metadata": "",
      "metadata_url": "",
      "idp_name": ""
    },
    "macos_migration": {
      "enable": false,
      "mode": "voluntary",
      "webhook_url": "https://webhook.example.com"
    },
    "macos_setup": {
      "bootstrap_package": "",
      "enable_end_user_authentication": false,
      "macos_setup_assistant": "path/to/config.json"
    }
  },
  "agent_options": {
    "config": {
      "options": {
        "pack_delimiter": "/",
        "logger_tls_period": 10,
        "distributed_plugin": "tls",
        "disable_distributed": false,
        "logger_tls_endpoint": "/api/v1/osquery/log",
        "distributed_interval": 10,
        "distributed_tls_max_attempts": 3
      },
      "decorators": {
        "load": [
          "SELECT uuid AS host_uuid FROM system_info;",
          "SELECT hostname AS hostname FROM system_info;"
        ]
      }
    },
    "overrides": {},
    "command_line_flags": {}
  },
  "vulnerability_settings": {
    "databases_path": ""
  },
  "webhook_settings": {
    "host_status_webhook": {
      "enable_host_status_webhook": true,
      "destination_url": "https://server.com",
      "host_percentage": 5,
      "days_count": 7
    },
    "failing_policies_webhook":{
      "enable_failing_policies_webhook":true,
      "destination_url": "https://server.com",
      "policy_ids": [1, 2, 3],
      "host_batch_size": 1000
    },
    "vulnerabilities_webhook":{
      "enable_vulnerabilities_webhook":true,
      "destination_url": "https://server.com",
      "host_batch_size": 1000
    }
  },
  "integrations": {
    "jira": [
      {
        "url": "https://jiraserver.com",
        "username": "some_user",
        "password": "sec4et!",
        "project_key": "jira_project",
        "enable_software_vulnerabilities": false
      }
    ]
  },
  "logging": {
      "debug": false,
      "json": false,
      "result": {
          "plugin": "firehose",
          "config": {
              "region": "us-east-1",
              "status_stream": "",
              "result_stream": "result-topic"
          }
      },
      "status": {
          "plugin": "filesystem",
          "config": {
              "status_log_file": "foo_status",
              "result_log_file": "",
              "enable_log_rotation": false,
              "enable_log_compression": false
          }
      }
  }
}
```

### Get global enroll secrets

Returns the valid global enroll secrets.

`GET /api/v1/fleet/spec/enroll_secret`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/spec/enroll_secret`

##### Default response

`Status: 200`

```json
{
    "spec": {
        "secrets": [
            {
                "secret": "vhPzPOnCMOMoqSrLxKxzSADyqncayacB",
                "created_at": "2021-11-12T20:24:57Z"
            },
            {
                "secret": "jZpexWGiXmXaFAKdrdttFHdJBqEnqlVF",
                "created_at": "2021-11-12T20:24:57Z"
            }
        ]
    }
}
```

### Modify global enroll secrets

Replaces all existing global enroll secrets.

`POST /api/v1/fleet/spec/enroll_secret`

#### Parameters

| Name      | Type    | In   | Description                                                        |
| --------- | ------- | ---- | ------------------------------------------------------------------ |
| spec      | object  | body | **Required**. Attribute "secrets" must be a list of enroll secrets |

#### Example

Replace all global enroll secrets with a new enroll secret.

`POST /api/v1/fleet/spec/enroll_secret`

##### Request body

```json
{
    "spec": {
        "secrets": [
            {
                "secret": "KuSkYFsHBQVlaFtqOLwoUIWniHhpvEhP"
            }
        ]
    }
}
```

##### Default response

`Status: 200`

```json
{}
```

#### Example

Delete all global enroll secrets.

`POST /api/v1/fleet/spec/enroll_secret`

##### Request body

```json
{
    "spec": {
        "secrets": []
    }
}
```

##### Default response

`Status: 200`

```json
{}
```

### Get enroll secrets for a team

Returns the valid team enroll secrets.

`GET /api/v1/fleet/teams/{id}/secrets`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/teams/1/secrets`

##### Default response

`Status: 200`

```json
{
  "secrets": [
    {
      "created_at": "2021-06-16T22:05:49Z",
      "secret": "aFtH2Nq09hrvi73ErlWNQfa7M53D3rPR",
      "team_id": 1
    }
  ]
}
```


### Modify enroll secrets for a team

Replaces all existing team enroll secrets.

`PATCH /api/v1/fleet/teams/{id}/secrets`

#### Parameters

| Name      | Type    | In   | Description                            |
| --------- | ------- | ---- | -------------------------------------- |
| id        | integer | path | **Required**. The team's id.           |
| secrets   | array   | body | **Required**. A list of enroll secrets |

#### Example

Replace all of a team's existing enroll secrets with a new enroll secret

`PATCH /api/v1/fleet/teams/2/secrets`

##### Request body

```json
{
  "secrets": [
    {
      "secret": "n07v32y53c237734m3n201153c237"
    }
  ]
}
```

##### Default response

`Status: 200`

```json
{
  "secrets": [
    {
      "secret": "n07v32y53c237734m3n201153c237",
      "created_at": "0001-01-01T00:00:00Z"
    }
  ]
}
```

#### Example

Delete all of a team's existing enroll secrets

`PATCH /api/v1/fleet/teams/2/secrets`

##### Request body

```json
{
  "secrets": []
}
```

##### Default response

`Status: 200`

```json
{
  "secrets": null
}
```

### Create invite

`POST /api/v1/fleet/invites`

#### Parameters

| Name        | Type    | In   | Description                                                                                                                                           |
| ----------- | ------- | ---- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| global_role | string  | body | Role the user will be granted. Either a global role is needed, or a team role.                                                                        |
| email       | string  | body | **Required.** The email of the invited user. This email will receive the invitation link.                                                             |
| name        | string  | body | **Required.** The name of the invited user.                                                                                                           |
| sso_enabled | boolean | body | **Required.** Whether or not SSO will be enabled for the invited user.                                                                                |
| teams       | list    | body | _Available in Fleet Premium_ A list of the teams the user is a member of. Each item includes the team's ID and the user's role in the specified team. |

#### Example

##### Request body

```json
{
  "email": "john_appleseed@example.com",
  "name": "John",
  "sso_enabled": false,
  "global_role": null,
  "teams": [
    {
      "id": 2,
      "role": "observer"
    },
    {
      "id": 3,
      "role": "maintainer"
    }
  ]
}
```

`POST /api/v1/fleet/invites`

##### Default response

`Status: 200`

```json
{
  "invite": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 3,
    "invited_by": 1,
    "email": "john_appleseed@example.com",
    "name": "John",
    "sso_enabled": false,
    "teams": [
      {
        "id": 10,
        "created_at": "0001-01-01T00:00:00Z",
        "name": "Apples",
        "description": "",
        "agent_options": null,
        "user_count": 0,
        "host_count": 0,
        "role": "observer"
      },
      {
        "id": 14,
        "created_at": "0001-01-01T00:00:00Z",
        "name": "Best of the Best Engineering",
        "description": "",
        "agent_options": null,
        "user_count": 0,
        "host_count": 0,
        "role": "maintainer"
      }
    ]
  }
}
```

### List invites

Returns a list of the active invitations in Fleet.

`GET /api/v1/fleet/invites`

#### Parameters

| Name            | Type   | In    | Description                                                                                                                   |
| --------------- | ------ | ----- | ----------------------------------------------------------------------------------------------------------------------------- |
| order_key       | string | query | What to order results by. Can be any column in the invites table.                                                             |
| order_direction | string | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |
| query           | string | query | Search query keywords. Searchable fields include `name` and `email`.                                                          |

#### Example

`GET /api/v1/fleet/invites`

##### Default response

`Status: 200`

```json
{
  "invites": [
    {
      "created_at": "0001-01-01T00:00:00Z",
      "updated_at": "0001-01-01T00:00:00Z",
      "id": 3,
      "email": "john_appleseed@example.com",
      "name": "John",
      "sso_enabled": false,
      "global_role": "admin",
      "teams": []
    },
    {
      "created_at": "0001-01-01T00:00:00Z",
      "updated_at": "0001-01-01T00:00:00Z",
      "id": 4,
      "email": "bob_marks@example.com",
      "name": "Bob",
      "sso_enabled": false,
      "global_role": "admin",
      "teams": []
    }
  ]
}
```

### Delete invite

Delete the specified invite from Fleet.

`DELETE /api/v1/fleet/invites/{id}`

#### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required.** The user's id. |

#### Example

`DELETE /api/v1/fleet/invites/{id}`

##### Default response

`Status: 200`


### Verify invite

Verify the specified invite.

`GET /api/v1/fleet/invites/{token}`

#### Parameters

| Name  | Type    | In   | Description                            |
| ----- | ------- | ---- | -------------------------------------- |
| token | integer | path | **Required.** The user's invite token. |

#### Example

`GET /api/v1/fleet/invites/{token}`

##### Default response

`Status: 200`

```json
{
    "invite": {
        "created_at": "2021-01-15T00:58:33Z",
        "updated_at": "2021-01-15T00:58:33Z",
        "id": 4,
        "email": "steve@example.com",
        "name": "Steve",
        "sso_enabled": false,
        "global_role": "admin",
        "teams": []
    }
}
```

##### Not found

`Status: 404`

```json
{
    "message": "Resource Not Found",
    "errors": [
        {
            "name": "base",
            "reason": "Invite with token <token> was not found in the datastore"
        }
    ]
}
```

### Update invite

`PATCH /api/v1/fleet/invites/{id}`

#### Parameters

| Name        | Type    | In   | Description                                                                                                                                           |
| ----------- | ------- | ---- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| global_role | string  | body | Role the user will be granted. Either a global role is needed, or a team role.                                                                        |
| email       | string  | body | The email of the invited user. Updates on the email won't resend the invitation.                                                             |
| name        | string  | body | The name of the invited user.                                                                                                           |
| sso_enabled | boolean | body | Whether or not SSO will be enabled for the invited user.                                                                                |
| teams       | list    | body | _Available in Fleet Premium_ A list of the teams the user is a member of. Each item includes the team's ID and the user's role in the specified team. |

#### Example

`PATCH /api/v1/fleet/invites/123`

##### Request body

```json
{
  "email": "john_appleseed@example.com",
  "name": "John",
  "sso_enabled": false,
  "global_role": null,
  "teams": [
    {
      "id": 2,
      "role": "observer"
    },
    {
      "id": 3,
      "role": "maintainer"
    }
  ]
}
```

##### Default response

`Status: 200`

```json
{
  "invite": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 3,
    "invited_by": 1,
    "email": "john_appleseed@example.com",
    "name": "John",
    "sso_enabled": false,
    "teams": [
      {
        "id": 10,
        "created_at": "0001-01-01T00:00:00Z",
        "name": "Apples",
        "description": "",
        "agent_options": null,
        "user_count": 0,
        "host_count": 0,
        "role": "observer"
      },
      {
        "id": 14,
        "created_at": "0001-01-01T00:00:00Z",
        "name": "Best of the Best Engineering",
        "description": "",
        "agent_options": null,
        "user_count": 0,
        "host_count": 0,
        "role": "maintainer"
      }
    ]
  }
}
```

### Version

Get version and build information from the Fleet server.

`GET /api/v1/fleet/version`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/version`

##### Default response

`Status: 200`

```json
{
  "version": "3.9.0-93-g1b67826f-dirty",
  "branch": "version",
  "revision": "1b67826fe4bf40b2f45ec53e01db9bf467752e74",
  "go_version": "go1.15.7",
  "build_date": "2021-03-27T00:28:48Z",
  "build_user": "zwass"
}
```

---

## Hosts

- [On the different timestamps in the host data structure](#on-the-different-timestamps-in-the-host-data-structure)
- [List hosts](#list-hosts)
- [Count hosts](#count-hosts)
- [Get hosts summary](#get-hosts-summary)
- [Get host](#get-host)
- [Get host by identifier](#get-host-by-identifier)
- [Get host by device token](#get-host-by-device-token)
- [Delete host](#delete-host)
- [Refetch host](#refetch-host)
- [Transfer hosts to a team](#transfer-hosts-to-a-team)
- [Transfer hosts to a team by filter](#transfer-hosts-to-a-team-by-filter)
- [Bulk delete hosts by filter or ids](#bulk-delete-hosts-by-filter-or-ids)
- [Get host's Google Chrome profiles](#get-hosts-google-chrome-profiles)
- [Get host's mobile device management (MDM) information](#get-hosts-mobile-device-management-mdm-information)
- [Get mobile device management (MDM) summary](#get-mobile-device-management-mdm-summary)
- [Get host's macadmin mobile device management (MDM) and Munki information](#get-hosts-macadmin-mobile-device-management-mdm-and-munki-information)
- [Get aggregated host's mobile device management (MDM) and Munki information](#get-aggregated-hosts-macadmin-mobile-device-management-mdm-and-munki-information)
- [Get host OS versions](#get-host-os-versions)
- [Get hosts report in CSV](#get-hosts-report-in-csv)
- [Get host's disk encryption key](#get-hosts-disk-encryption-key)

### On the different timestamps in the host data structure

Hosts have a set of timestamps usually named with an "_at" suffix, such as created_at, enrolled_at, etc. Before we go
through each of them and what they mean, we need to understand a bit more about how the host data structure is
represented in the database.

The table `hosts` is the main one. It holds the core data for a host. A host doesn't exist if there is no row for it in
this table. This table also holds most of the timestamps, but it doesn't hold all of the host data. This is an important
detail as we'll see below.

There's adjacent tables to this one that usually follow the name convention `host_<extra data descriptor>`. Examples of
this are: `host_additional` that holds additional query results, `host_software` that links a host with many rows from
the `software` table.

- `created_at`: the time the row in the database was created, which usually corresponds to the first enrollment of the host.
- `updated_at`: the last time the row in the database for the `hosts` table was updated.
- `detail_updated_at`: the last time Fleet updated host data, based on the results from the detail queries (this includes updates to host associated tables, e.g. `host_users`).
- `label_updated_at`: the last time Fleet updated the label membership for the host based on the results from the queries ran.
- `last_enrolled_at`: the last time the host enrolled to Fleet.
- `policy_updated_at`: the last time we updated the policy results for the host based on the queries ran.
- `seen_time`: the last time the host contacted the fleet server, regardless of what operation it was for.
- `software_updated_at`: the last time software changed for the host in any way.

### List hosts

`GET /api/v1/fleet/hosts`

#### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                                                                                                                                                                                 |
| ----------------------- | ------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                                                                                                                                                                                        |
| per_page                | integer | query | Results per page.                                                                                                                                                                                                                                                                                                                           |
| order_key               | string  | query | What to order results by. Can be any column in the hosts table.                                                                                                                                                                                                                                                                             |
| after                   | string  | query | The value to get results after. This needs `order_key` defined, as that's the column that would be used.  **Note:** Use `page` instead of `after`.                                                                                                                                                                                                                                   |
| order_direction         | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`.                                                                                                                                                                                                               |
| status                  | string  | query | Indicates the status of the hosts to return. Can either be `new`, `online`, `offline`, `mia` or `missing`.                                                                                                                                                                                                                                  |
| query                   | string  | query | Search query keywords. Searchable fields include `hostname`, `machine_serial`, `uuid`, `ipv4` and the hosts' email addresses (only searched if the query looks like an email address, i.e. contains an `@`, no space, etc.).                                                                                                                |
| additional_info_filters | string  | query | A comma-delimited list of fields to include in each host's additional information object. See [Fleet Configuration Options](https://fleetdm.com/docs/using-fleet/fleetctl-cli#fleet-configuration-options) for an example configuration with hosts' additional information. Use `*` to get all stored fields.                                                  |
| team_id                 | integer | query | _Available in Fleet Premium_ Filters the hosts to only include hosts in the specified team.                                                                                                                                                                                                                                                 |
| policy_id               | integer | query | The ID of the policy to filter hosts by.                                                                                                                                                                                                                                                                                                    |
| policy_response         | string  | query | Valid options are `passing` or `failing`.  `policy_id` must also be specified with `policy_response`.                                                                                                                                                                                                                                       |
| software_id             | integer | query | The ID of the software to filter hosts by.                                                                                                                                                                                                                                                                                                  |
| os_id                   | integer | query | The ID of the operating system to filter hosts by.                                                                                                                                                                                                                                                                                          |
| os_name                 | string  | query | The name of the operating system to filter hosts by. `os_version` must also be specified with `os_name`                                                                                                                                                                                                                                     |
| os_version              | string  | query | The version of the operating system to filter hosts by. `os_name` must also be specified with `os_version`                                                                                                                                                                                                                                  |
| device_mapping          | boolean | query | Indicates whether `device_mapping` should be included for each host. See ["Get host's Google Chrome profiles](#get-hosts-google-chrome-profiles) for more information about this feature.                                                                                                                                                  |
| mdm_id                  | integer | query | The ID of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider and URL).                                                                                                                                                                                                |
| mdm_name                | string  | query | The name of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider).                                                                                                                                                                                                |
| mdm_enrollment_status   | string  | query | The _mobile device management_ (MDM) enrollment status to filter hosts by. Can be one of 'manual', 'automatic', 'enrolled', 'pending', or 'unenrolled'.                                                                                                                                                                                                             |
| macos_settings          | string  | query | Filters the hosts by the status of the _mobile device management_ (MDM) profiles applied to hosts. Can be one of 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team id filter, the results include only hosts that are not assigned to any team.**                                                                                                                                                                                                             |
| munki_issue_id          | integer | query | The ID of the _munki issue_ (a Munki-reported error or warning message) to filter hosts by (that is, filter hosts that are affected by that corresponding error or warning message).                                                                                                                                                        |
| low_disk_space          | integer | query | _Available in Fleet Premium_ Filters the hosts to only include hosts with less GB of disk space available than this value. Must be a number between 1-100.                                                                                                                                                                                  |
| disable_failing_policies| boolean | query | If "true", hosts will return failing policies as 0 regardless of whether there are any that failed for the host. This is meant to be used when increased performance is needed in exchange for the extra information.                                                                                                                       |
| macos_settings_disk_encryption | string | query | Filters the hosts by the status of the macOS disk encryption MDM profile on the host. Can be one of `verified`, `verifying`, `action_required`, `enforcing`, `failed`, or `removing_enforcement`. |
| bootstrap_package       | string | query | _Available in Fleet Premium_ Filters the hosts by the status of the MDM bootstrap package on the host. Can be one of `installed`, `pending`, or `failed`. |

If `additional_info_filters` is not specified, no `additional` information will be returned.

If `software_id` is specified, an additional top-level key `"software"` is returned with the software object corresponding to the `software_id`. See [List all software](#list-all-software) response payload for details about this object.

If `mdm_id` is specified, an additional top-level key `"mobile_device_management_solution"` is returned with the information corresponding to the `mdm_id`.

If `mdm_id`, `mdm_name` or `mdm_enrollment_status` is specified, then Windows Servers are excluded from the results.

If `munki_issue_id` is specified, an additional top-level key `"munki_issue"` is returned with the information corresponding to the `munki_issue_id`.

If `after` is being used with `created_at` or `updated_at`, the table must be specified in `order_key`. Those columns become `h.created_at` and `h.updated_at`.

#### Example

`GET /api/v1/fleet/hosts?page=0&per_page=100&order_key=hostname&query=2ce`

##### Request query parameters

```json
{
  "page": 0,
  "per_page": 100,
  "order_key": "hostname"
}
```

##### Default response

`Status: 200`

```json
{
  "hosts": [
    {
      "created_at": "2020-11-05T05:09:44Z",
      "updated_at": "2020-11-05T06:03:39Z",
      "id": 1,
      "detail_updated_at": "2020-11-05T05:09:45Z",
      "software_updated_at": "2020-11-05T05:09:44Z",
      "label_updated_at": "2020-11-05T05:14:51Z",
      "policy_updated_at": "2023-06-26T18:33:15Z",
      "last_enrolled_at": "2023-02-26T22:33:12Z",
      "seen_time": "2020-11-05T06:03:39Z",
      "hostname": "2ceca32fe484",
      "uuid": "392547dc-0000-0000-a87a-d701ff75bc65",
      "platform": "centos",
      "osquery_version": "2.7.0",
      "os_version": "CentOS Linux 7",
      "build": "",
      "platform_like": "rhel fedora",
      "code_name": "",
      "uptime": 8305000000000,
      "memory": 2084032512,
      "cpu_type": "6",
      "cpu_subtype": "142",
      "cpu_brand": "Intel(R) Core(TM) i5-8279U CPU @ 2.40GHz",
      "cpu_physical_cores": 4,
      "cpu_logical_cores": 4,
      "hardware_vendor": "",
      "hardware_model": "",
      "hardware_version": "",
      "hardware_serial": "",
      "computer_name": "2ceca32fe484",
      "display_name": "2ceca32fe484",
      "public_ip": "",
      "primary_ip": "",
      "primary_mac": "",
      "distributed_interval": 10,
      "config_tls_refresh": 10,
      "logger_tls_period": 8,
      "additional": {},
      "status": "offline",
      "display_text": "2ceca32fe484",
      "team_id": null,
      "team_name": null,
      "pack_stats": null,
      "issues": {
        "failing_policies_count": 2,
        "total_issues_count": 2
      },
      "geolocation": {
        "country_iso": "US",
        "city_name": "New York",
        "geometry": {
          "type": "point",
          "coordinates": [40.6799, -74.0028]
        }
      },
      "mdm": {
        "encryption_key_available": false,
        "enrollment_status": null,
        "name": "",
        "server_url": null
      }
    }
  ]
}
```

> Note: the response above assumes a [GeoIP database is configured](https://fleetdm.com/docs/deploying/configuration#geoip), otherwise the `geolocation` object won't be included.

Response payload with the `mdm_id` filter provided:

```json
{
  "hosts": [...],
  "mobile_device_management_solution": {
    "server_url": "http://some.url/mdm",
    "name": "MDM Vendor Name",
    "id": 999
  }
}
```

Response payload with the `munki_issue_id` filter provided:

```json
{
  "hosts": [...],
  "munki_issue": {
    "id": 1,
    "name": "Could not retrieve managed install primary manifest",
    "type": "error"
  }
}
```

### Count hosts

`GET /api/v1/fleet/hosts/count`

#### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                                                                                                                                                                                 |
| ----------------------- | ------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| order_key               | string  | query | What to order results by. Can be any column in the hosts table.                                                                                                                                                                                                                                                                             |
| order_direction         | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`.                                                                                                                                                                                                               |
| after                   | string  | query | The value to get results after. This needs `order_key` defined, as that's the column that would be used.                                                                                                                                                                                                                                    |
| status                  | string  | query | Indicates the status of the hosts to return. Can either be `new`, `online`, `offline`, `mia` or `missing`.                                                                                                                                                                                                                                  |
| query                   | string  | query | Search query keywords. Searchable fields include `hostname`, `machine_serial`, `uuid`, `ipv4` and the hosts' email addresses (only searched if the query looks like an email address, i.e. contains an `@`, no space, etc.).                                                                                                                |
| team_id                 | integer | query | _Available in Fleet Premium_ Filters the hosts to only include hosts in the specified team.                                                                                                                                                                                                                                                 |
| policy_id               | integer | query | The ID of the policy to filter hosts by.                                                                                                                                                                                                                                                                                                    |
| policy_response         | string  | query | Valid options are `passing` or `failing`.  `policy_id` must also be specified with `policy_response`.                                                                                                                                                                                                                                       |
| software_id             | integer | query | The ID of the software to filter hosts by.                                                                                                                                                                                                                                                                                                  |
| os_id                   | integer | query | The ID of the operating system to filter hosts by.                                                                                                                                                                                                                                                                                          |
| os_name                 | string  | query | The name of the operating system to filter hosts by. `os_version` must also be specified with `os_name`                                                                                                                                                                                                                                     |
| os_version              | string  | query | The version of the operating system to filter hosts by. `os_name` must also be specified with `os_version`                                                                                                                                                                                                                                  |
| label_id                | integer | query | A valid label ID. Can only be used in combination with `order_key`, `order_direction`, `after`, `status`, `query` and `team_id`.                                                                                                                                                                                                            |
| mdm_id                  | integer | query | The ID of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider and URL).                                                                                                                                                                                                |
| mdm_name                | string  | query | The name of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider).                                                                                                                                                                                                |
| mdm_enrollment_status   | string  | query | The _mobile device management_ (MDM) enrollment status to filter hosts by. Can be one of 'manual', 'automatic', 'enrolled', 'pending', or 'unenrolled'.                                                                                                                                                                                                             |
| macos_settings          | string  | query | Filters the hosts by the status of the _mobile device management_ (MDM) profiles applied to hosts. Can be one of 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team id filter, the results include only hosts that are not assigned to any team.**                                                                                                                                                                                                             |
| munki_issue_id          | integer | query | The ID of the _munki issue_ (a Munki-reported error or warning message) to filter hosts by (that is, filter hosts that are affected by that corresponding error or warning message).                                                                                                                                                        |
| low_disk_space          | integer | query | _Available in Fleet Premium_ Filters the hosts to only include hosts with less GB of disk space available than this value. Must be a number between 1-100.                                                                                                                                                                                  |
| macos_settings_disk_encryption | string | query | Filters the hosts by the status of the macOS disk encryption MDM profile on the host. Can be one of `verified`, `verifying`, `action_required`, `enforcing`, `failed`, or `removing_enforcement`. |
| bootstrap_package       | string | query | _Available in Fleet Premium_ Filters the hosts by the status of the MDM bootstrap package on the host. Can be one of `installed`, `pending`, or `failed`. **Note: If this filter is used in Fleet Premium without a team id filter, the results include only hosts that are not assigned to any team.** |

If `additional_info_filters` is not specified, no `additional` information will be returned.

If `mdm_id`, `mdm_name` or `mdm_enrollment_status` is specified, then Windows Servers are excluded from the results.

#### Example

`GET /api/v1/fleet/hosts/count?page=0&per_page=100&order_key=hostname&query=2ce`

##### Request query parameters

```json
{
  "page": 0,
  "per_page": 100,
  "order_key": "hostname"
}
```

##### Default response

`Status: 200`

```json
{
  "count": 123
}
```

### Get hosts summary

Returns the count of all hosts organized by status. `online_count` includes all hosts currently enrolled in Fleet. `offline_count` includes all hosts that haven't checked into Fleet recently. `mia_count` includes all hosts that haven't been seen by Fleet in more than 30 days. `new_count` includes the hosts that have been enrolled to Fleet in the last 24 hours.

`GET /api/v1/fleet/host_summary`

#### Parameters

| Name            | Type    | In    | Description                                                                     |
| --------------- | ------- | ----  | ------------------------------------------------------------------------------- |
| team_id         | integer | query | The ID of the team whose host counts should be included. Defaults to all teams. |
| platform        | string  | query | Platform to filter by when counting. Defaults to all platforms.                 |
| low_disk_space  | integer | query | _Available in Fleet Premium_ Returns the count of hosts with less GB of disk space available than this value. Must be a number between 1-100. |

#### Example

`GET /api/v1/fleet/host_summary?team_id=1&low_disk_space=32`

##### Default response

`Status: 200`

```json
{
  "team_id": 1,
  "totals_hosts_count": 2408,
  "online_count": 2267,
  "offline_count": 141,
  "mia_count": 0,
  "missing_30_days_count": 0,
  "new_count": 0,
  "all_linux_count": 1204,
  "low_disk_space_count": 12,
  "builtin_labels": [
    {
      "id": 6,
      "name": "All Hosts",
      "description": "All hosts which have enrolled in Fleet",
      "label_type": "builtin"
    },
    {
      "id": 7,
      "name": "macOS",
      "description": "All macOS hosts",
      "label_type": "builtin"
    },
    {
      "id": 8,
      "name": "Ubuntu Linux",
      "description": "All Ubuntu hosts",
      "label_type": "builtin"
    },
    {
      "id": 9,
      "name": "CentOS Linux",
      "description": "All CentOS hosts",
      "label_type": "builtin"
    },
    {
      "id": 10,
      "name": "MS Windows",
      "description": "All Windows hosts",
      "label_type": "builtin"
    },
    {
      "id": 11,
      "name": "Red Hat Linux",
      "description": "All Red Hat Enterprise Linux hosts",
      "label_type": "builtin"
    },
    {
      "id": 12,
      "name": "All Linux",
      "description": "All Linux distributions",
      "label_type": "builtin"
    }
  ],
  "platforms": [
    {
      "platform": "chrome",
      "hosts_count": 1234
    },
    {
      "platform": "darwin",
      "hosts_count": 1234
    },
    {
      "platform": "rhel",
      "hosts_count": 1234
    },
    {
      "platform": "ubuntu",
      "hosts_count": 12044
    },
    {
      "platform": "windows",
      "hosts_count": 12044
    }

  ]
}
```

### Get host

Returns the information of the specified host.

`GET /api/v1/fleet/hosts/{id}`

#### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The host's id. |

#### Example

`GET /api/v1/fleet/hosts/121`

##### Default response

`Status: 200`

```json
{
  "host": {
    "created_at": "2021-08-19T02:02:22Z",
    "updated_at": "2021-08-19T21:14:58Z",
    "software": [
      {
        "id": 408,
        "name": "osquery",
        "version": "4.5.1",
        "source": "rpm_packages",
        "generated_cpe": "",
        "vulnerabilities": null,
        "installed_paths": ["/usr/lib/some-path-1"]
      },
      {
        "id": 1146,
        "name": "tar",
        "version": "1.30",
        "source": "rpm_packages",
        "generated_cpe": "",
        "vulnerabilities": null
      },
      {
        "id": 321,
        "name": "SomeApp.app",
        "version": "1.0",
        "source": "apps",
        "bundle_identifier": "com.some.app",
        "last_opened_at": "2021-08-18T21:14:00Z",
        "generated_cpe": "",
        "vulnerabilities": null,
        "installed_paths": ["/usr/lib/some-path-2"]
      }
    ],
    "id": 1,
    "detail_updated_at": "2021-08-19T21:07:53Z",
    "software_updated_at": "2020-11-05T05:09:44Z",
    "label_updated_at": "2021-08-19T21:07:53Z",
    "policy_updated_at": "2023-06-26T18:33:15Z",
    "last_enrolled_at": "2021-08-19T02:02:22Z",
    "seen_time": "2021-08-19T21:14:58Z",
    "refetch_requested": false,
    "hostname": "23cfc9caacf0",
    "uuid": "309a4b7d-0000-0000-8e7f-26ae0815ede8",
    "platform": "rhel",
    "osquery_version": "4.5.1",
    "os_version": "CentOS Linux 8.3.2011",
    "build": "",
    "platform_like": "rhel",
    "code_name": "",
    "uptime": 210671000000000,
    "memory": 16788398080,
    "cpu_type": "x86_64",
    "cpu_subtype": "158",
    "cpu_brand": "Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz",
    "cpu_physical_cores": 12,
    "cpu_logical_cores": 12,
    "hardware_vendor": "",
    "hardware_model": "",
    "hardware_version": "",
    "hardware_serial": "",
    "computer_name": "23cfc9caacf0",
    "display_name": "23cfc9caacf0",
    "public_ip": "",
    "primary_ip": "172.27.0.6",
    "primary_mac": "02:42:ac:1b:00:06",
    "distributed_interval": 10,
    "config_tls_refresh": 10,
    "logger_tls_period": 10,
    "team_id": null,
    "pack_stats": null,
    "team_name": null,
    "additional": {},
    "gigs_disk_space_available": 46.1,
    "percent_disk_space_available": 73,
    "disk_encryption_enabled": true,
    "users": [
      {
        "uid": 0,
        "username": "root",
        "type": "",
        "groupname": "root",
        "shell": "/bin/bash"
      },
      {
        "uid": 1,
        "username": "bin",
        "type": "",
        "groupname": "bin",
        "shell": "/sbin/nologin"
      }
    ],
    "labels": [
      {
        "created_at": "2021-08-19T02:02:17Z",
        "updated_at": "2021-08-19T02:02:17Z",
        "id": 6,
        "name": "All Hosts",
        "description": "All hosts which have enrolled in Fleet",
        "query": "SELECT 1;",
        "platform": "",
        "label_type": "builtin",
        "label_membership_type": "dynamic"
      },
      {
        "created_at": "2021-08-19T02:02:17Z",
        "updated_at": "2021-08-19T02:02:17Z",
        "id": 9,
        "name": "CentOS Linux",
        "description": "All CentOS hosts",
        "query": "SELECT 1 FROM os_version WHERE platform = 'centos' OR name LIKE '%centos%'",
        "platform": "",
        "label_type": "builtin",
        "label_membership_type": "dynamic"
      },
      {
        "created_at": "2021-08-19T02:02:17Z",
        "updated_at": "2021-08-19T02:02:17Z",
        "id": 12,
        "name": "All Linux",
        "description": "All Linux distributions",
        "query": "SELECT 1 FROM osquery_info WHERE build_platform LIKE '%ubuntu%' OR build_distro LIKE '%centos%';",
        "platform": "",
        "label_type": "builtin",
        "label_membership_type": "dynamic"
      }
    ],
    "packs": [],
    "status": "online",
    "display_text": "23cfc9caacf0",
    "policies": [
      {
        "id": 1,
        "name": "SomeQuery",
        "query": "SELECT * FROM foo;",
        "description": "this is a query",
        "resolution": "fix with these steps...",
        "platform": "windows,linux",
        "response": "pass",
        "critical": false
      },
      {
        "id": 2,
        "name": "SomeQuery2",
        "query": "SELECT * FROM bar;",
        "description": "this is another query",
        "resolution": "fix with these other steps...",
        "platform": "darwin",
        "response": "fail",
        "critical": false
      },
      {
        "id": 3,
        "name": "SomeQuery3",
        "query": "SELECT * FROM baz;",
        "description": "",
        "resolution": "",
        "platform": "",
        "response": "",
        "critical": false
      }
    ],
    "issues": {
      "failing_policies_count": 2,
      "total_issues_count": 2
    },
    "batteries": [
      {
        "cycle_count": 999,
        "health": "Normal"
      }
    ],
    "geolocation": {
      "country_iso": "US",
      "city_name": "New York",
      "geometry": {
        "type": "point",
        "coordinates": [40.6799, -74.0028]
      }
    },
    "mdm": {
      "encryption_key_available": false,
      "enrollment_status": null,
      "name": "",
      "server_url": null,
      "macos_settings": {
        "disk_encryption": null,
        "action_required": null
      },
      "macos_setup": {
        "bootstrap_package_status": "installed",
        "detail": "",
        "bootstrap_package_name": "test.pkg"
      },
      "profiles": [
        {
          "profile_id": 999,
          "name": "profile1",
          "status": "verifying",
          "operation_type": "install",
          "detail": ""
        }
      ]
    }
  }
}
```

> Note: the response above assumes a [GeoIP database is configured](https://fleetdm.com/docs/deploying/configuration#geoip), otherwise the `geolocation` object won't be included.

> Note: `installed_paths` may be blank depending on installer package. For example, on Linux, RPM-installed packages do not provide installed path information.

### Get host by identifier

Returns the information of the host specified using the `uuid`, `osquery_host_id`, `hostname`, or
`node_key` as an identifier

`GET /api/v1/fleet/hosts/identifier/{identifier}`

#### Parameters

| Name       | Type              | In   | Description                                                                   |
| ---------- | ----------------- | ---- | ----------------------------------------------------------------------------- |
| identifier | integer or string | path | **Required**. The host's `uuid`, `osquery_host_id`, `hostname`, or `node_key` |

#### Example

`GET /api/v1/fleet/hosts/identifier/392547dc-0000-0000-a87a-d701ff75bc65`

##### Default response

`Status: 200`

```json
{
  "host": {
    "created_at": "2022-02-10T02:29:13Z",
    "updated_at": "2022-10-14T17:07:11Z",
    "software": [
      {
          "id": 16923,
          "name": "Automat",
          "version": "0.8.0",
          "source": "python_packages",
          "generated_cpe": "",
          "vulnerabilities": null,
          "installed_paths": ["/usr/lib/some_path/"]
      }
    ],
    "id": 33,
    "detail_updated_at": "2022-10-14T17:07:12Z",
    "label_updated_at": "2022-10-14T17:07:12Z",
    "policy_updated_at": "2022-10-14T17:07:12Z",
    "last_enrolled_at": "2022-02-10T02:29:13Z",
    "software_updated_at": "2020-11-05T05:09:44Z",
    "seen_time": "2022-10-14T17:45:41Z",
    "refetch_requested": false,
    "hostname": "23cfc9caacf0",
    "uuid": "392547dc-0000-0000-a87a-d701ff75bc65",
    "platform": "ubuntu",
    "osquery_version": "5.5.1",
    "os_version": "Ubuntu 20.04.3 LTS",
    "build": "",
    "platform_like": "debian",
    "code_name": "focal",
    "uptime": 20807520000000000,
    "memory": 1024360448,
    "cpu_type": "x86_64",
    "cpu_subtype": "63",
    "cpu_brand": "DO-Regular",
    "cpu_physical_cores": 1,
    "cpu_logical_cores": 1,
    "hardware_vendor": "",
    "hardware_model": "",
    "hardware_version": "",
    "hardware_serial": "",
    "computer_name": "23cfc9caacf0",
    "public_ip": "",
    "primary_ip": "172.27.0.6",
    "primary_mac": "02:42:ac:1b:00:06",
    "distributed_interval": 10,
    "config_tls_refresh": 60,
    "logger_tls_period": 10,
    "team_id": 2,
    "pack_stats": [
      {
        "pack_id": 1,
        "pack_name": "Global",
        "type": "global",
        "query_stats": [
          {
            "scheduled_query_name": "Get running processes (with user_name)",
            "scheduled_query_id": 49,
            "query_name": "Get running processes (with user_name)",
            "pack_name": "Global",
            "pack_id": 1,
            "average_memory": 260000,
            "denylisted": false,
            "executions": 1,
            "interval": 86400,
            "last_executed": "2022-10-14T10:00:01Z",
            "output_size": 198,
            "system_time": 20,
            "user_time": 80,
            "wall_time": 0
          }
        ]
      }
    ],
    "team_name": null,
    "gigs_disk_space_available": 19.29,
    "percent_disk_space_available": 74,
    "issues": {
        "total_issues_count": 0,
        "failing_policies_count": 0
    },
    "labels": [
            {
            "created_at": "2021-09-14T05:11:02Z",
            "updated_at": "2021-09-14T05:11:02Z",
            "id": 12,
            "name": "All Linux",
            "description": "All Linux distributions",
            "query": "SELECT 1 FROM osquery_info WHERE build_platform LIKE '%ubuntu%' OR build_distro LIKE '%centos%';",
            "platform": "",
            "label_type": "builtin",
            "label_membership_type": "dynamic"
        }
    ],
    "packs": [
          {
            "created_at": "2021-09-17T05:28:54Z",
            "updated_at": "2021-09-17T05:28:54Z",
            "id": 1,
            "name": "Global",
            "description": "Global pack",
            "disabled": false,
            "type": "global",
            "labels": null,
            "label_ids": null,
            "hosts": null,
            "host_ids": null,
            "teams": null,
            "team_ids": null
        }
    ],
    "policies": [
      {
            "id": 142,
            "name": "Full disk encryption enabled (macOS)",
            "query": "SELECT 1 FROM disk_encryption WHERE user_uuid IS NOT '' AND filevault_status = 'on' LIMIT 1;",
            "description": "Checks to make sure that full disk encryption (FileVault) is enabled on macOS devices.",
            "author_id": 31,
            "author_name": "",
            "author_email": "",
            "team_id": null,
            "resolution": "To enable full disk encryption, on the failing device, select System Preferences > Security & Privacy > FileVault > Turn On FileVault.",
            "platform": "darwin,linux",
            "created_at": "2022-09-02T18:52:19Z",
            "updated_at": "2022-09-02T18:52:19Z",
            "response": "fail",
            "critical": false
        }
    ],
    "batteries": [
      {
        "cycle_count": 999,
        "health": "Normal"
      }
    ],
    "geolocation": {
      "country_iso": "US",
      "city_name": "New York",
      "geometry": {
        "type": "point",
        "coordinates": [40.6799, -74.0028]
      }
    },
    "status": "online",
    "display_text": "dogfood-ubuntu-box",
    "display_name": "dogfood-ubuntu-box",
    "mdm": {
      "encryption_key_available": false,
      "enrollment_status": null,
      "name": "",
      "server_url": null,
      "macos_settings": {
        "disk_encryption": null,
        "action_required": null
      },
      "macos_setup": {
        "bootstrap_package_status": "installed",
        "detail": ""
      },
      "profiles": [
        {
          "profile_id": 999,
          "name": "profile1",
          "status": "verifying",
          "operation_type": "install",
          "detail": ""
        }
      ]
    }
  }
}
```

> Note: the response above assumes a [GeoIP database is configured](https://fleetdm.com/docs/deploying/configuration#geoip), otherwise the `geolocation` object won't be included.

> Note: `installed_paths` may be blank depending on installer package. For example, on Linux, RPM-installed packages do not provide installed path information.

#### Get host by device token

Returns a subset of information about the host specified by `token`. To get all information about a host, use the "Get host" endpoint [here](#get-host).

This is the API route used by the **My device** page in Fleet desktop to display information about the host to the end user.

`GET /api/v1/fleet/device/{token}`

##### Parameters

| Name  | Type   | In   | Description                        |
| ----- | ------ | ---- | ---------------------------------- |
| token | string | path | The device's authentication token. |

##### Example

`GET /api/v1/fleet/device/abcdef012456789`

##### Default response

`Status: 200`

```json
{
  "host": {
    "created_at": "2021-08-19T02:02:22Z",
    "updated_at": "2021-08-19T21:14:58Z",
    "software": [
      {
        "id": 408,
        "name": "osquery",
        "version": "4.5.1",
        "source": "rpm_packages",
        "generated_cpe": "",
        "vulnerabilities": null
      },
      {
        "id": 1146,
        "name": "tar",
        "version": "1.30",
        "source": "rpm_packages",
        "generated_cpe": "",
        "vulnerabilities": null
      },
      {
        "id": 321,
        "name": "SomeApp.app",
        "version": "1.0",
        "source": "apps",
        "bundle_identifier": "com.some.app",
        "last_opened_at": "2021-08-18T21:14:00Z",
        "generated_cpe": "",
        "vulnerabilities": null
      }
    ],
    "id": 1,
    "detail_updated_at": "2021-08-19T21:07:53Z",
    "label_updated_at": "2021-08-19T21:07:53Z",
    "last_enrolled_at": "2021-08-19T02:02:22Z",
    "seen_time": "2021-08-19T21:14:58Z",
    "refetch_requested": false,
    "hostname": "23cfc9caacf0",
    "uuid": "309a4b7d-0000-0000-8e7f-26ae0815ede8",
    "platform": "rhel",
    "osquery_version": "4.5.1",
    "os_version": "CentOS Linux 8.3.2011",
    "build": "",
    "platform_like": "rhel",
    "code_name": "",
    "uptime": 210671000000000,
    "memory": 16788398080,
    "cpu_type": "x86_64",
    "cpu_subtype": "158",
    "cpu_brand": "Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz",
    "cpu_physical_cores": 12,
    "cpu_logical_cores": 12,
    "hardware_vendor": "",
    "hardware_model": "",
    "hardware_version": "",
    "hardware_serial": "",
    "computer_name": "23cfc9caacf0",
    "display_name": "23cfc9caacf0",
    "public_ip": "",
    "primary_ip": "172.27.0.6",
    "primary_mac": "02:42:ac:1b:00:06",
    "distributed_interval": 10,
    "config_tls_refresh": 10,
    "logger_tls_period": 10,
    "team_id": null,
    "pack_stats": null,
    "team_name": null,
    "additional": {},
    "gigs_disk_space_available": 46.1,
    "percent_disk_space_available": 73,
    "disk_encryption_enabled": true,
    "dep_assigned_to_fleet": false,
    "users": [
      {
        "uid": 0,
        "username": "root",
        "type": "",
        "groupname": "root",
        "shell": "/bin/bash"
      },
      {
        "uid": 1,
        "username": "bin",
        "type": "",
        "groupname": "bin",
        "shell": "/sbin/nologin"
      }
    ],
    "labels": [
      {
        "created_at": "2021-08-19T02:02:17Z",
        "updated_at": "2021-08-19T02:02:17Z",
        "id": 6,
        "name": "All Hosts",
        "description": "All hosts which have enrolled in Fleet",
        "query": "SELECT 1;",
        "platform": "",
        "label_type": "builtin",
        "label_membership_type": "dynamic"
      },
      {
        "created_at": "2021-08-19T02:02:17Z",
        "updated_at": "2021-08-19T02:02:17Z",
        "id": 9,
        "name": "CentOS Linux",
        "description": "All CentOS hosts",
        "query": "SELECT 1 FROM os_version WHERE platform = 'centos' OR name LIKE '%centos%'",
        "platform": "",
        "label_type": "builtin",
        "label_membership_type": "dynamic"
      },
      {
        "created_at": "2021-08-19T02:02:17Z",
        "updated_at": "2021-08-19T02:02:17Z",
        "id": 12,
        "name": "All Linux",
        "description": "All Linux distributions",
        "query": "SELECT 1 FROM osquery_info WHERE build_platform LIKE '%ubuntu%' OR build_distro LIKE '%centos%';",
        "platform": "",
        "label_type": "builtin",
        "label_membership_type": "dynamic"
      }
    ],
    "packs": [],
    "status": "online",
    "display_text": "23cfc9caacf0",
    "batteries": [
      {
        "cycle_count": 999,
        "health": "Good"
      }
    ],
    "mdm": {
      "encryption_key_available": false,
      "enrollment_status": null,
      "name": "",
      "server_url": null,
      "macos_settings": {
        "disk_encryption": null,
        "action_required": null
      },
      "macos_setup": {
        "bootstrap_package_status": "installed",
        "detail": "",
        "bootstrap_package_name": "test.pkg"
      },
      "profiles": [
        {
          "profile_id": 999,
          "name": "profile1",
          "status": "verifying",
          "operation_type": "install",
          "detail": ""
        }
      ]
    }
  },
  "org_logo_url": "https://example.com/logo.jpg",
  "license": {
    "tier": "free",
    "expiration": "2031-01-01T00:00:00Z"
  },
  "global_config": {
    "mdm": {
      "enabled_and_configured": false
    }
  }
}
```

### Delete host

Deletes the specified host from Fleet. Note that a deleted host will fail authentication with the previous node key, and in most osquery configurations will attempt to re-enroll automatically. If the host still has a valid enroll secret, it will re-enroll successfully.

`DELETE /api/v1/fleet/hosts/{id}`

#### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The host's id. |

#### Example

`DELETE /api/v1/fleet/hosts/121`

##### Default response

`Status: 200`


### Refetch host

Flags the host details, labels and policies to be refetched the next time the host checks in for distributed queries. Note that we cannot be certain when the host will actually check in and update the query results. Further requests to the host APIs will indicate that the refetch has been requested through the `refetch_requested` field on the host object.

`POST /api/v1/fleet/hosts/{id}/refetch`

#### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The host's id. |

#### Example

`POST /api/v1/fleet/hosts/121/refetch`

##### Default response

`Status: 200`


### Transfer hosts to a team

_Available in Fleet Premium_

`POST /api/v1/fleet/hosts/transfer`

#### Parameters

| Name    | Type    | In   | Description                                                             |
| ------- | ------- | ---- | ----------------------------------------------------------------------- |
| team_id | integer | body | **Required**. The ID of the team you'd like to transfer the host(s) to. |
| hosts   | array   | body | **Required**. A list of host IDs.                                       |

#### Example

`POST /api/v1/fleet/hosts/transfer`

##### Request body

```json
{
  "team_id": 1,
  "hosts": [3, 2, 4, 6, 1, 5, 7]
}
```

##### Default response

`Status: 200`


### Transfer hosts to a team by filter

_Available in Fleet Premium_

`POST /api/v1/fleet/hosts/transfer/filter`

#### Parameters

| Name    | Type    | In   | Description                                                                                                                                                                                                                                                                                                                        |
| ------- | ------- | ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| team_id | integer | body | **Required**. The ID of the team you'd like to transfer the host(s) to.                                                                                                                                                                                                                                                            |
| filters | object  | body | **Required** Contains any of the following three properties: `query` for search query keywords. Searchable fields include `hostname`, `machine_serial`, `uuid`, and `ipv4`. `status` to indicate the status of the hosts to return. Can either be `new`, `online`, `offline`, `mia` or `missing`. `label_id` to indicate the selected label. `label_id` and `status` cannot be used at the same time. |

#### Example

`POST /api/v1/fleet/hosts/transfer/filter`

##### Request body

```json
{
  "team_id": 1,
  "filters": {
    "status": "online"
  }
}
```

##### Default response

`Status: 200`

### Bulk delete hosts by filter or ids

`POST /api/v1/fleet/hosts/delete`

#### Parameters

| Name    | Type    | In   | Description                                                                                                                                                                                                                                                                                                                        |
| ------- | ------- | ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ids     | list    | body | A list of the host IDs you'd like to delete. If `ids` is specified, `filters` cannot be specified.                                                                                                                                                                                                                                                           |
| filters | object  | body | Contains any of the following four properties: `query` for search query keywords. Searchable fields include `hostname`, `machine_serial`, `uuid`, and `ipv4`. `status` to indicate the status of the hosts to return. Can either be `new`, `online`, `offline`, `mia` or `missing`. `label_id` to indicate the selected label. `team_id` to indicate the selected team. If `filters` is specified, `id` cannot be specified. `label_id` and `status` cannot be used at the same time. |

Either ids or filters are required.

Request (`ids` is specified):

```json
{
  "ids": [1]
}
```

Request (`filters` is specified):
```json
{
  "filters": {
    "status": "online",
    "label_id": 1,
    "team_id": 1,
    "query": "abc"
  }
}
```

#### Example

`POST /api/v1/fleet/hosts/delete`

##### Request body

```json
{
  "filters": {
    "status": "online",
    "team_id": 1
  }
}
```

##### Default response

`Status: 200`

### Get host's Google Chrome profiles

Retrieves a host's Google Chrome profile information which can be used to link a host to a specific user by email.

Requires [Fleetd](https://fleetdm.com/docs/using-fleet/fleetd), the osquery manager from Fleet. Fleetd can be built with [fleetctl](https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer).

`GET /api/v1/fleet/hosts/{id}/device_mapping`

#### Parameters

| Name       | Type              | In   | Description                                                                   |
| ---------- | ----------------- | ---- | ----------------------------------------------------------------------------- |
| id         | integer           | path | **Required**. The host's `id`.                                                |

#### Example

`GET /api/v1/fleet/hosts/1/device_mapping`

##### Default response

`Status: 200`

```json
{
  "host_id": 1,
  "device_mapping": [
    {
      "email": "user@example.com",
      "source": "google_chrome_profiles"
    }
  ]
}
```

---

### Get host's mobile device management (MDM) information

Currently supports Windows and MacOS. On MacOS this requires the [macadmins osquery
extension](https://github.com/macadmins/osquery-extension) which comes bundled
in [Fleet's osquery installers](https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer).

Retrieves a host's MDM enrollment status and MDM server URL.

If the host exists but is not enrolled to an MDM server, then this API returns `null`.

`GET /api/v1/fleet/hosts/{id}/mdm`

#### Parameters

| Name    | Type    | In   | Description                                                                                                                                                                                                                                                                                                                        |
| ------- | ------- | ---- | -------------------------------------------------------------------------------- |
| id      | integer | path | **Required** The id of the host to get the details for                           |

#### Example

`GET /api/v1/fleet/hosts/32/mdm`

##### Default response

`Status: 200`

```json
{
  "enrollment_status": "On (automatic)",
  "server_url": "some.mdm.com",
  "name": "Some MDM",
  "id": 3
}
```

---

### Get mobile device management (MDM) summary

Currently supports Windows and MacOS. On MacOS this requires the [macadmins osquery
extension](https://github.com/macadmins/osquery-extension) which comes bundled
in [Fleet's osquery installers](https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer).

Retrieves MDM enrollment summary. Windows servers are excluded from the aggregated data.

`GET /api/v1/fleet/hosts/summary/mdm`

#### Parameters

| Name     | Type    | In    | Description                                                                                                                                                                                                                                                                                                                        |
| -------- | ------- | ----- | -------------------------------------------------------------------------------- |
| team_id  | integer | query | _Available in Fleet Premium_ Filter by team                                      |
| platform | string  | query | Filter by platform ("windows" or "darwin")                                       |

A `team_id` of `0` returns the statistics for hosts that are not part of any team. A `null` or missing `team_id` returns statistics for all hosts regardless of the team.

#### Example

`GET /api/v1/fleet/hosts/summary/mdm?team_id=1&platform=windows`

##### Default response

`Status: 200`

```json
{
  "counts_updated_at": "2021-03-21T12:32:44Z",
  "mobile_device_management_enrollment_status": {
    "enrolled_manual_hosts_count": 0,
    "enrolled_automated_hosts_count": 2,
    "unenrolled_hosts_count": 0,
    "hosts_count": 2
  },
  "mobile_device_management_solution": [
    {
      "id": 2,
      "name": "Solution1",
      "server_url": "solution1.com",
      "hosts_count": 1
    },
    {
      "id": 3,
      "name": "Solution2",
      "server_url": "solution2.com",
      "hosts_count": 1
    }
  ]
}
```

---

### Get host's macadmin mobile device management (MDM) and Munki information

Requires the [macadmins osquery
extension](https://github.com/macadmins/osquery-extension) which comes bundled
in [Fleet's osquery
installers](https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer).
Currently supported only on macOS.

Retrieves a host's MDM enrollment status, MDM server URL, and Munki version.

`GET /api/v1/fleet/hosts/{id}/macadmins`

#### Parameters

| Name    | Type    | In   | Description                                                                                                                                                                                                                                                                                                                        |
| ------- | ------- | ---- | -------------------------------------------------------------------------------- |
| id      | integer | path | **Required** The id of the host to get the details for                           |

#### Example

`GET /api/v1/fleet/hosts/32/macadmins`

##### Default response

`Status: 200`

```json
{
  "macadmins": {
    "munki": {
      "version": "1.2.3"
    },
    "munki_issues": [
      {
        "id": 1,
        "name": "Could not retrieve managed install primary manifest",
        "type": "error",
        "created_at": "2022-08-01T05:09:44Z"
      },
      {
        "id": 2,
        "name": "Could not process item Figma for optional install. No pkginfo found in catalogs: release",
        "type": "warning",
        "created_at": "2022-08-01T05:09:44Z"
      }
    ],
    "mobile_device_management": {
      "enrollment_status": "On (automatic)",
      "server_url": "http://some.url/mdm",
      "name": "MDM Vendor Name",
      "id": 999
    }
  }
}
```

---

### Get aggregated host's macadmin mobile device management (MDM) and Munki information

Requires the [macadmins osquery
extension](https://github.com/macadmins/osquery-extension) which comes bundled
in [Fleet's osquery
installers](https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer).
Currently supported only on macOS.


Retrieves aggregated host's MDM enrollment status and Munki versions.

`GET /api/v1/fleet/macadmins`

#### Parameters

| Name    | Type    | In    | Description                                                                                                                                                                                                                                                                                                                        |
| ------- | ------- | ----- | ---------------------------------------------------------------------------------------------------------------- |
| team_id | integer | query | _Available in Fleet Premium_ Filters the aggregate host information to only include hosts in the specified team. |                           |

A `team_id` of `0` returns the statistics for hosts that are not part of any team. A `null` or missing `team_id` returns statistics for all hosts regardless of the team.

#### Example

`GET /api/v1/fleet/macadmins`

##### Default response

`Status: 200`

```json
{
  "macadmins": {
    "counts_updated_at": "2021-03-21T12:32:44Z",
    "munki_versions": [
      {
        "version": "5.5",
        "hosts_count": 8360
      },
      {
        "version": "5.4",
        "hosts_count": 1700
      },
      {
        "version": "5.3",
        "hosts_count": 400
      },
      {
        "version": "5.2.3",
        "hosts_count": 112
      },
      {
        "version": "5.2.2",
        "hosts_count": 50
      }
    ],
    "munki_issues": [
      {
        "id": 1,
        "name": "Could not retrieve managed install primary manifest",
        "type": "error",
        "hosts_count": 2851
      },
      {
        "id": 2,
        "name": "Could not process item Figma for optional install. No pkginfo found in catalogs: release",
        "type": "warning",
        "hosts_count": 1983
      }
    ],
    "mobile_device_management_enrollment_status": {
      "enrolled_manual_hosts_count": 124,
      "enrolled_automated_hosts_count": 124,
      "unenrolled_hosts_count": 112
    },
    "mobile_device_management_solution": [
      {
        "id": 1,
        "name": "SimpleMDM",
        "hosts_count": 8360,
        "server_url": "https://a.simplemdm.com/mdm"
      },
      {
        "id": 2,
        "name": "Intune",
        "hosts_count": 1700,
        "server_url": "https://enrollment.manage.microsoft.com"
      }
    ]
  }
}
```

### Get host OS versions

Retrieves the aggregated host OS versions information.

`GET /api/v1/fleet/os_versions`

#### Parameters

| Name                | Type     | In    | Description                                                                                                                          |
| ---      | ---      | ---   | ---                                                                                                                                  |
| team_id             | integer | query | _Available in Fleet Premium_ Filters the hosts to only include hosts in the specified team. If not provided, all hosts are included. |
| platform            | string   | query | Filters the hosts to the specified platform |
| os_name     | string | query | The name of the operating system to filter hosts by. `os_version` must also be specified with `os_name`                                                 |
| os_version    | string | query | The version of the operating system to filter hosts by. `os_name` must also be specified with `os_version`                                                 |

##### Default response

`Status: 200`

```json
{
  "counts_updated_at": "2022-03-22T21:38:31Z",
  "os_versions": [
    {
      "hosts_count": 1,
      "name": "CentOS 6.10.0",
      "name_only": "CentOS",
      "version": "6.10.0",
      "platform": "rhel",
      "os_id": 1
    },
    {
      "hosts_count": 1,
      "name": "CentOS Linux 7.9.2009",
      "name_only": "CentOS",
      "version": "7.9.2009",
      "platform": "rhel",
      "os_id": 2
    },
    {
      "hosts_count": 1,
      "name": "CentOS Linux 8.3.2011",
      "name_only": "CentOS",
      "version": "8.2.2011",
      "platform": "rhel",
      "os_id": 3
    },
    {
      "hosts_count": 1,
      "name": "Debian GNU/Linux 10.0.0",
      "name_only": "Debian GNU/Linux",
      "version": "10.0.0",
      "platform": "debian",
      "os_id": 4
    },
    {
      "hosts_count": 1,
      "name": "Debian GNU/Linux 9.0.0",
      "name_only": "Debian GNU/Linux",
      "version": "9.0.0",
      "platform": "debian",
      "os_id": 5
    },
    {
      "hosts_count": 1,
      "name": "Ubuntu 16.4.0 LTS",
      "name_only": "Ubuntu",
      "version": "16.4.0 LTS",
      "platform": "ubuntu",
      "os_id": 6
    }
  ]
}
```

### Get hosts report in CSV

Returns the list of hosts corresponding to the search criteria in CSV format, ready for download when
requested by a web browser.

`GET /api/v1/fleet/hosts/report`

#### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                                                                                                                                                                                 |
| ----------------------- | ------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| format                  | string  | query | **Required**, must be "csv" (only supported format for now).                                                                                                                                                                                                                                                                                |
| columns                 | string  | query | Comma-delimited list of columns to include in the report (returns all columns if none is specified).                                                                                                                                                                                                                                        |
| order_key               | string  | query | What to order results by. Can be any column in the hosts table.                                                                                                                                                                                                                                                                             |
| order_direction         | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`.                                                                                                                                                                                                               |
| status                  | string  | query | Indicates the status of the hosts to return. Can either be `new`, `online`, `offline`, `mia` or `missing`.                                                                                                                                                                                                                                  |
| query                   | string  | query | Search query keywords. Searchable fields include `hostname`, `machine_serial`, `uuid`, `ipv4` and the hosts' email addresses (only searched if the query looks like an email address, i.e. contains an `@`, no space, etc.).                                                                                                                |
| team_id                 | integer | query | _Available in Fleet Premium_ Filters the hosts to only include hosts in the specified team.                                                                                                                                                                                                                                                 |
| policy_id               | integer | query | The ID of the policy to filter hosts by.                                                                                                                                                                                                                                                                                                    |
| policy_response         | string  | query | Valid options are `passing` or `failing`. `policy_id` must also be specified with `policy_response`. **Note: If `policy_id` is specified _without_ including `policy_response`, this will also return hosts where the policy is not configured to run or failed to run.** |
| software_id             | integer | query | The ID of the software to filter hosts by.                                                                                                                                                                                                                                                                                                  |
| os_id                   | integer | query | The ID of the operating system to filter hosts by.                                                                                                                                                                                                                                                                                          |
| os_name                 | string  | query | The name of the operating system to filter hosts by. `os_version` must also be specified with `os_name`                                                                                                                                                                                                                                     |
| os_version              | string  | query | The version of the operating system to filter hosts by. `os_name` must also be specified with `os_version`                                                                                                                                                                                                                                  |
| mdm_id                  | integer | query | The ID of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider and URL).                                                                                                                                                                                                |
| mdm_name                | string  | query | The name of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider).                                                                                                                                                                                                |
| mdm_enrollment_status   | string  | query | The _mobile device management_ (MDM) enrollment status to filter hosts by. Can be one of 'manual', 'automatic', 'enrolled', 'pending', or 'unenrolled'.                                                                                                                                                                                                             |
| macos_settings          | string  | query | Filters the hosts by the status of the _mobile device management_ (MDM) profiles applied to hosts. Can be one of 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team id filter, the results include only hosts that are not assigned to any team.**                                                                                                                                                                                                             |
| munki_issue_id          | integer | query | The ID of the _munki issue_ (a Munki-reported error or warning message) to filter hosts by (that is, filter hosts that are affected by that corresponding error or warning message).                                                                                                                                                        |
| low_disk_space          | integer | query | _Available in Fleet Premium_ Filters the hosts to only include hosts with less GB of disk space available than this value. Must be a number between 1-100.                                                                                                                                                                                  |
| label_id                | integer | query | A valid label ID. Can only be used in combination with `order_key`, `order_direction`, `status`, `query` and `team_id`.                                                                                                                                                                                                                     |
| bootstrap_package       | string | query | _Available in Fleet Premium_ Filters the hosts by the status of the MDM bootstrap package on the host. Can be one of `installed`, `pending`, or `failed`. **Note: If this filter is used in Fleet Premium without a team id filter, the results include only hosts that are not assigned to any team.** |
| disable_failing_policies | boolean | query | If `true`, hosts will return failing policies as 0 (returned as the `issues` column) regardless of whether there are any that failed for the host. This is meant to be used when increased performance is needed in exchange for the extra information.      |

If `mdm_id`, `mdm_name` or `mdm_enrollment_status` is specified, then Windows Servers are excluded from the results.

#### Example

`GET /api/v1/fleet/hosts/report?software_id=123&format=csv&columns=hostname,primary_ip,platform`

##### Default response

`Status: 200`

```csv
created_at,updated_at,id,detail_updated_at,label_updated_at,policy_updated_at,last_enrolled_at,seen_time,refetch_requested,hostname,uuid,platform,osquery_version,os_version,build,platform_like,code_name,uptime,memory,cpu_type,cpu_subtype,cpu_brand,cpu_physical_cores,cpu_logical_cores,hardware_vendor,hardware_model,hardware_version,hardware_serial,computer_name,primary_ip_id,primary_ip,primary_mac,distributed_interval,config_tls_refresh,logger_tls_period,team_id,team_name,gigs_disk_space_available,percent_disk_space_available,issues,device_mapping,status,display_text
2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,1,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,false,foo.local0,a4fc55a1-b5de-409c-a2f4-441f564680d3,debian,,,,,,0s,0,,,,0,0,,,,,,,,,0,0,0,,,0,0,0,,,,
2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:22:56Z,false,foo.local1,689539e5-72f0-4bf7-9cc5-1530d3814660,rhel,,,,,,0s,0,,,,0,0,,,,,,,,,0,0,0,,,0,0,0,,,,
2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,3,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:21:56Z,false,foo.local2,48ebe4b0-39c3-4a74-a67f-308f7b5dd171,linux,,,,,,0s,0,,,,0,0,,,,,,,,,0,0,0,,,0,0,0,,,,
```

### Get host's disk encryption key

Requires the [macadmins osquery extension](https://github.com/macadmins/osquery-extension) which comes bundled
in [Fleet's osquery installers](https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer).

Requires Fleet's MDM properly [enabled and configured](https://fleetdm.com/docs/using-fleet/mdm-setup).

Retrieves the disk encryption key for a host.

`GET /api/v1/fleet/mdm/hosts/:id/encryption_key`

#### Parameters

| Name | Type    | In   | Description                                                        |
| ---- | ------- | ---- | ------------------------------------------------------------------ |
| id   | integer | path | **Required** The id of the host to get the disk encryption key for |


#### Example

`GET /api/v1/fleet/mdm/hosts/8/encryption_key`

##### Default response

`Status: 200`

```json
{
  "host_id": 8,
  "encryption_key": {
    "key": "5ADZ-HTZ8-LJJ4-B2F8-JWH3-YPBT",
    "updated_at": "2022-12-01T05:31:43Z"
  }
}
```

### Get configuration profiles assigned to a host

Requires Fleet's MDM properly [enabled and configured](https://fleetdm.com/docs/using-fleet/mdm-setup).

Retrieves a list of the configuration profiles assigned to a host.

`GET /api/v1/fleet/mdm/hosts/:id/profiles`

#### Parameters

| Name | Type    | In   | Description                      |
| ---- | ------- | ---- | -------------------------------- |
| id   | integer | path | **Required**. The ID of the host  |


#### Example

`GET /api/v1/fleet/mdm/hosts/8/profiles`

##### Default response

`Status: 200`

```json
{
  "host_id": 8,
  "profiles": [
    {
      "profile_id": 1337,
      "team_id": 0,
      "name": "Example profile",
      "identifier": "com.example.profile",
      "created_at": "2023-03-31T00:00:00Z",
      "updated_at": "2023-03-31T00:00:00Z",
      "checksum": "dGVzdAo="
    }
  ]
}
```

---


## Labels

- [Create label](#create-label)
- [Modify label](#modify-label)
- [Get label](#get-label)
- [Get labels summary](#get-labels-summary)
- [List labels](#list-labels)
- [List hosts in a label](#list-hosts-in-a-label)
- [Delete label](#delete-label)
- [Delete label by ID](#delete-label-by-id)

### Create label

Creates a dynamic label.

`POST /api/v1/fleet/labels`

#### Parameters

| Name        | Type   | In   | Description                                                                                                                                                                                                                                  |
| ----------- | ------ | ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| name        | string | body | **Required**. The label's name.                                                                                                                                                                                                              |
| description | string | body | The label's description.                                                                                                                                                                                                                     |
| query       | string | body | **Required**. The query in SQL syntax used to filter the hosts.                                                                                                                                                                              |
| platform    | string | body | The specific platform for the label to target. Provides an additional filter. Choices for platform are `darwin`, `windows`, `ubuntu`, and `centos`. All platforms are included by default and this option is represented by an empty string. |

#### Example

`POST /api/v1/fleet/labels`

##### Request body

```json
{
  "name": "Ubuntu hosts",
  "description": "Filters ubuntu hosts",
  "query": "SELECT 1 FROM os_version WHERE platform = 'ubuntu';",
  "platform": ""
}
```

##### Default response

`Status: 200`

```json
{
  "label": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 1,
    "name": "Ubuntu hosts",
    "description": "Filters ubuntu hosts",
    "query": "SELECT 1 FROM os_version WHERE platform = 'ubuntu';",
    "label_type": "regular",
    "label_membership_type": "dynamic",
    "display_text": "Ubuntu hosts",
    "count": 0,
    "host_ids": null
  }
}
```

### Modify label

Modifies the specified label. Note: Label queries and platforms are immutable. To change these, you must delete the label and create a new label.

`PATCH /api/v1/fleet/labels/{id}`

#### Parameters

| Name        | Type    | In   | Description                   |
| ----------- | ------- | ---- | ----------------------------- |
| id          | integer | path | **Required**. The label's id. |
| name        | string  | body | The label's name.             |
| description | string  | body | The label's description.      |

#### Example

`PATCH /api/v1/fleet/labels/1`

##### Request body

```json
{
  "name": "macOS label",
  "description": "Now this label only includes macOS machines",
  "platform": "darwin"
}
```

##### Default response

`Status: 200`

```json
{
  "label": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 1,
    "name": "Ubuntu hosts",
    "description": "Filters ubuntu hosts",
    "query": "SELECT 1 FROM os_version WHERE platform = 'ubuntu';",
    "platform": "darwin",
    "label_type": "regular",
    "label_membership_type": "dynamic",
    "display_text": "Ubuntu hosts",
    "count": 0,
    "host_ids": null
  }
}
```

### Get label

Returns the specified label.

`GET /api/v1/fleet/labels/{id}`

#### Parameters

| Name | Type    | In   | Description                   |
| ---- | ------- | ---- | ----------------------------- |
| id   | integer | path | **Required**. The label's id. |

#### Example

`GET /api/v1/fleet/labels/1`

##### Default response

`Status: 200`

```json
{
  "label": {
    "created_at": "2021-02-09T22:09:43Z",
    "updated_at": "2021-02-09T22:15:58Z",
    "id": 12,
    "name": "Ubuntu",
    "description": "Filters ubuntu hosts",
    "query": "SELECT 1 FROM os_version WHERE platform = 'ubuntu';",
    "label_type": "regular",
    "label_membership_type": "dynamic",
    "display_text": "Ubuntu",
    "count": 0,
    "host_ids": null
  }
}
```

### Get labels summary

Returns a list of all the labels in Fleet.

`GET /api/v1/fleet/labels/summary`

#### Example

`GET /api/v1/fleet/labels/summary`

##### Default response

`Status: 200`

```json
{
  "labels": [
    {
      "id": 6,
      "name": "All Hosts",
      "description": "All hosts which have enrolled in Fleet",
      "label_type": "builtin"
    },
    {
      "id": 7,
      "name": "macOS",
      "description": "All macOS hosts",
      "label_type": "builtin"
    },
    {
      "id": 8,
      "name": "Ubuntu Linux",
      "description": "All Ubuntu hosts",
      "label_type": "builtin"
    },
    {
      "id": 9,
      "name": "CentOS Linux",
      "description": "All CentOS hosts",
      "label_type": "builtin"
    },
    {
      "id": 10,
      "name": "MS Windows",
      "description": "All Windows hosts",
      "label_type": "builtin"
    }
  ]
}
```

### List labels

Returns a list of all the labels in Fleet.

`GET /api/v1/fleet/labels`

#### Parameters

| Name            | Type    | In    | Description   |
| --------------- | ------- | ----- |------------------------------------- |
| order_key       | string  | query | What to order results by. Can be any column in the labels table.                                                  |
| order_direction | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |

#### Example

`GET /api/v1/fleet/labels`

##### Default response

`Status: 200`

```json
{
  "labels": [
    {
      "created_at": "2021-02-02T23:55:25Z",
      "updated_at": "2021-02-02T23:55:25Z",
      "id": 6,
      "name": "All Hosts",
      "description": "All hosts which have enrolled in Fleet",
      "query": "SELECT 1;",
      "label_type": "builtin",
      "label_membership_type": "dynamic",
      "host_count": 7,
      "display_text": "All Hosts",
      "count": 7,
      "host_ids": null
    },
    {
      "created_at": "2021-02-02T23:55:25Z",
      "updated_at": "2021-02-02T23:55:25Z",
      "id": 7,
      "name": "macOS",
      "description": "All macOS hosts",
      "query": "SELECT 1 FROM os_version WHERE platform = 'darwin';",
      "platform": "darwin",
      "label_type": "builtin",
      "label_membership_type": "dynamic",
      "host_count": 1,
      "display_text": "macOS",
      "count": 1,
      "host_ids": null
    },
    {
      "created_at": "2021-02-02T23:55:25Z",
      "updated_at": "2021-02-02T23:55:25Z",
      "id": 8,
      "name": "Ubuntu Linux",
      "description": "All Ubuntu hosts",
      "query": "SELECT 1 FROM os_version WHERE platform = 'ubuntu';",
      "platform": "ubuntu",
      "label_type": "builtin",
      "label_membership_type": "dynamic",
      "host_count": 3,
      "display_text": "Ubuntu Linux",
      "count": 3,
      "host_ids": null
    },
    {
      "created_at": "2021-02-02T23:55:25Z",
      "updated_at": "2021-02-02T23:55:25Z",
      "id": 9,
      "name": "CentOS Linux",
      "description": "All CentOS hosts",
      "query": "SELECT 1 FROM os_version WHERE platform = 'centos' OR name LIKE '%centos%'",
      "label_type": "builtin",
      "label_membership_type": "dynamic",
      "host_count": 3,
      "display_text": "CentOS Linux",
      "count": 3,
      "host_ids": null
    },
    {
      "created_at": "2021-02-02T23:55:25Z",
      "updated_at": "2021-02-02T23:55:25Z",
      "id": 10,
      "name": "MS Windows",
      "description": "All Windows hosts",
      "query": "SELECT 1 FROM os_version WHERE platform = 'windows';",
      "platform": "windows",
      "label_type": "builtin",
      "label_membership_type": "dynamic",
      "display_text": "MS Windows",
      "count": 0,
      "host_ids": null
    }
  ]
}
```

### List hosts in a label

Returns a list of the hosts that belong to the specified label.

`GET /api/v1/fleet/labels/{id}/hosts`

#### Parameters

| Name                     | Type    | In    | Description                                                                                                                                                                                                                |
| ---------------          | ------- | ----- | -----------------------------------------------------------------------------------------------------------------------------                                                                                              |
| id                       | integer | path  | **Required**. The label's id.                                                                                                                                                                                              |
| page                     | integer | query | Page number of the results to fetch.                                                                                                                                                                                       |
| per_page                 | integer | query | Results per page.                                                                                                                                                                                                          |
| order_key                | string  | query | What to order results by. Can be any column in the hosts table.                                                                                                                                                            |
| order_direction          | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`.                                                                                              |
| after                    | string  | query | The value to get results after. This needs `order_key` defined, as that's the column that would be used.                                                                                                                   |
| status                   | string  | query | Indicates the status of the hosts to return. Can either be `new`, `online`, `offline`, `mia` or `missing`.                                                                                                                 |
| query                    | string  | query | Search query keywords. Searchable fields include `hostname`, `machine_serial`, `uuid`, and `ipv4`.                                                                                                                         |
| team_id                  | integer | query | _Available in Fleet Premium_ Filters the hosts to only include hosts in the specified team.                                                                                                                                |
| disable_failing_policies | boolean | query | If "true", hosts will return failing policies as 0 regardless of whether there are any that failed for the host. This is meant to be used when increased performance is needed in exchange for the extra information.      |
| mdm_id                   | integer | query | The ID of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider and URL).      |
| mdm_name                 | string  | query | The name of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider).      |
| mdm_enrollment_status    | string  | query | The _mobile device management_ (MDM) enrollment status to filter hosts by. Can be one of 'manual', 'automatic', 'enrolled', 'pending', or 'unenrolled'.                                                                                                                                                                                                             |
| macos_settings           | string  | query | Filters the hosts by the status of the _mobile device management_ (MDM) profiles applied to hosts. Can be one of 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team id filter, the results include only hosts that are not assigned to any team.**                                                                                                                                                                                                             |
| low_disk_space           | integer | query | _Available in Fleet Premium_ Filters the hosts to only include hosts with less GB of disk space available than this value. Must be a number between 1-100.                                                                 |
| macos_settings_disk_encryption | string | query | Filters the hosts by the status of the macOS disk encryption MDM profile on the host. Can be one of `verified`, `verifying`, `action_required`, `enforcing`, `failed`, or `removing_enforcement`. |
| bootstrap_package       | string | query | _Available in Fleet Premium_ Filters the hosts by the status of the MDM bootstrap package on the host. Can be one of `installed`, `pending`, or `failed`. **Note: If this filter is used in Fleet Premium without a team id filter, the results include only hosts that are not assigned to any team.** |

If `mdm_id`, `mdm_name` or `mdm_enrollment_status` is specified, then Windows Servers are excluded from the results.

#### Example

`GET /api/v1/fleet/labels/6/hosts&query=floobar`

##### Default response

`Status: 200`

```json
{
  "hosts": [
    {
      "created_at": "2021-02-03T16:11:43Z",
      "updated_at": "2021-02-03T21:58:19Z",
      "id": 2,
      "detail_updated_at": "2021-02-03T21:58:10Z",
      "label_updated_at": "2021-02-03T21:58:10Z",
      "policy_updated_at": "2023-06-26T18:33:15Z",
      "last_enrolled_at": "2021-02-03T16:11:43Z",
      "software_updated_at": "2020-11-05T05:09:44Z",
      "seen_time": "2021-02-03T21:58:20Z",
      "refetch_requested": false,
      "hostname": "floobar42",
      "uuid": "a2064cef-0000-0000-afb9-283e3c1d487e",
      "platform": "ubuntu",
      "osquery_version": "4.5.1",
      "os_version": "Ubuntu 20.4.0",
      "build": "",
      "platform_like": "debian",
      "code_name": "",
      "uptime": 32688000000000,
      "memory": 2086899712,
      "cpu_type": "x86_64",
      "cpu_subtype": "142",
      "cpu_brand": "Intel(R) Core(TM) i5-8279U CPU @ 2.40GHz",
      "cpu_physical_cores": 4,
      "cpu_logical_cores": 4,
      "hardware_vendor": "",
      "hardware_model": "",
      "hardware_version": "",
      "hardware_serial": "",
      "computer_name": "e2e7f8d8983d",
      "display_name": "e2e7f8d8983d",
      "primary_ip": "172.20.0.2",
      "primary_mac": "02:42:ac:14:00:02",
      "distributed_interval": 10,
      "config_tls_refresh": 10,
      "logger_tls_period": 10,
      "team_id": null,
      "pack_stats": null,
      "team_name": null,
      "status": "offline",
      "display_text": "e2e7f8d8983d",
      "mdm": {
        "encryption_key_available": false,
        "enrollment_status": null,
        "name": "",
        "server_url": null
      }
    }
  ]
}
```

### Delete label

Deletes the label specified by name.

`DELETE /api/v1/fleet/labels/{name}`

#### Parameters

| Name | Type   | In   | Description                     |
| ---- | ------ | ---- | ------------------------------- |
| name | string | path | **Required**. The label's name. |

#### Example

`DELETE /api/v1/fleet/labels/ubuntu_label`

##### Default response

`Status: 200`


### Delete label by ID

Deletes the label specified by ID.

`DELETE /api/v1/fleet/labels/id/{id}`

#### Parameters

| Name | Type    | In   | Description                   |
| ---- | ------- | ---- | ----------------------------- |
| id   | integer | path | **Required**. The label's id. |

#### Example

`DELETE /api/v1/fleet/labels/id/13`

##### Default response

`Status: 200`

---

## Mobile device management (MDM)

These API endpoints are used to automate MDM features in Fleet. Read more about MDM features in Fleet [here](https://fleetdm.com/docs/using-fleet/mdm-setup).

- [Add custom macOS setting (configuration profile)](#add-custom-macos-setting-configuration-profile)
- [List custom macOS settings (configuration profiles)](#list-custom-macos-settings-configuration-profiles)
- [Download custom macOS setting (configuration profile)](#download-custom-macos-setting-configuration-profile)
- [Delete custom macOS setting (configuration profile)](#delete-custom-macos-setting-configuration-profile)
- [Update disk encryption enforcement](#update-disk-encryption-enforcement)
- [Get disk encryption statistics](#get-disk-encryption-statistics)
- [Get macOS settings statistics](#get-macos-settings-statistics)
- [Run custom MDM command](#run-custom-mdm-command)
- [Get custom MDM command results](#get-custom-mdm-command-results)
- [List custom MDM commands](#list-custom-mdm-commands)
- [Set custom MDM setup enrollment profile](#set-custom-mdm-setup-enrollment-profile)
- [Get custom MDM setup enrollment profile](#get-custom-mdm-setup-enrollment-profile)
- [Delete custom MDM setup enrollment profile](#delete-custom-mdm-setup-enrollment-profile)
- [Get Apple Push Notification service (APNs)](#get-apple-push-notification-service-apns)
- [Get Apple Business Manager (ABM)](#get-apple-business-manager-abm)
- [Turn off MDM for a host](#turn-off-mdm-for-a-host)
- [Upload a bootstrap package](#upload-a-bootstrap-package)
- [Get metadata about a bootstrap package](#get-metadata-about-a-bootstrap-package)
- [Delete a bootstrap package](#delete-a-bootstrap-package)
- [Download a bootstrap package](#download-a-bootstrap-package)
- [Get a summary of bootstrap package status](#get-a-summary-of-bootstrap-package-status)
- [Upload an EULA file](#upload-an-eula-file)
- [Get metadata about an EULA file](#get-metadata-about-an-eula-file)
- [Delete an EULA file](#delete-an-eula-file)
- [Download an EULA file](#download-an-eula-file)

### Add custom macOS setting (configuration profile)

Add a configuration profile to enforce custom settings on macOS hosts.

`POST /api/v1/fleet/mdm/apple/profiles`

#### Parameters

| Name                      | Type     | In   | Description                                                               |
| ------------------------- | -------- | ---- | ------------------------------------------------------------------------- |
| profile                   | file     | form | **Required**. The mobileconfig file containing the profile.               |
| team_id                   | string   | form | _Available in Fleet Premium_ The team id for the profile. If specified, the profile is applied to only hosts that are assigned to the specified team. If not specified, the profile is applied to only to hosts that are not assigned to any team. |

#### Example

Add a new configuration profile to be applied to macOS hosts enrolled to Fleet's MDM that are
assigned to a team. Note that in this example the form data specifies`team_id` in addition to
`profile`.

`POST /api/v1/fleet/mdm/apple/profiles`

##### Request headers

```http
Content-Length: 850
Content-Type: multipart/form-data; boundary=------------------------f02md47480und42y
```

##### Request body

```http
--------------------------f02md47480und42y
Content-Disposition: form-data; name="team_id"

1
--------------------------f02md47480und42y
Content-Disposition: form-data; name="profile"; filename="Foo.mobileconfig"
Content-Type: application/octet-stream

<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array/>
  <key>PayloadDisplayName</key>
  <string>Example profile</string>
  <key>PayloadIdentifier</key>
  <string>com.example.profile</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>0BBF3E23-7F56-48FC-A2B6-5ACC598A4A69</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
--------------------------f02md47480und42y--

```

##### Default response

`Status: 200`

```json
{
  "profile_id": 42
}
```

###### Additional notes
If the response is `Status: 409 Conflict`, the body may include additional error details in the case
of duplicate payload display name or duplicate payload identifier.


### List custom macOS settings (configuration profiles)

Get a list of the configuration profiles in Fleet.

For Fleet Premium, the list can
optionally be filtered by team ID. If no team ID is specified, team profiles are excluded from the
results (i.e., only profiles that are associated with "No team" are listed).

`GET /api/v1/fleet/mdm/apple/profiles`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| team_id                   | string | query | _Available in Fleet Premium_ The team id to filter profiles.              |

#### Example

List all configuration profiles for macOS hosts enrolled to Fleet's MDM that are not assigned to any team.

`GET /api/v1/fleet/mdm/apple/profiles`

##### Default response

`Status: 200`

```json
{
  "profiles": [
    {
      "profile_id": 1337,
      "team_id": 0,
      "name": "Example profile",
      "identifier": "com.example.profile",
      "created_at": "2023-03-31T00:00:00Z",
      "updated_at": "2023-03-31T00:00:00Z",
      "checksum": "dGVzdAo="
    }
  ]
}
```

### Download custom macOS setting (configuration profile)

`GET /api/v1/fleet/mdm/apple/profiles/{profile_id}`

#### Parameters

| Name                      | Type    | In    | Description                                                               |
| ------------------------- | ------- | ----- | ------------------------------------------------------------------------- |
| profile_id                | integer | url   | **Required** The id of the profile to download.                           |

#### Example

`GET /api/v1/fleet/mdm/apple/profiles/42`

##### Default response

`Status: 200`

**Note** To confirm success, it is important for clients to match content length with the response
header (this is done automatically by most clients, including the browser) rather than relying
solely on the response status code returned by this endpoint.

##### Example response headers

```http
  Content-Length: 542
  Content-Type: application/octet-stream
  Content-Disposition: attachment;filename="2023-03-31 Example profile.mobileconfig"
```

###### Example response body
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array/>
  <key>PayloadDisplayName</key>
  <string>Example profile</string>
  <key>PayloadIdentifier</key>
  <string>com.example.profile</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>0BBF3E23-7F56-48FC-A2B6-5ACC598A4A69</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
</dict>
</plist>
```

### Delete custom macOS setting (configuration profile)

`DELETE /api/v1/fleet/mdm/apple/profiles/{profile_id}`

#### Parameters

| Name                      | Type    | In    | Description                                                               |
| ------------------------- | ------- | ----- | ------------------------------------------------------------------------- |
| profile_id                | integer | url   | **Required** The id of the profile to delete.                             |

#### Example

`DELETE /api/v1/fleet/mdm/apple/profiles/42`

##### Default response

`Status: 200`

### Update disk encryption enforcement

_Available in Fleet Premium_

`PATCH /api/v1/fleet/mdm/apple/settings`

#### Parameters

| Name                   | Type    | In    | Description                                                                                 |
| -------------          | ------  | ----  | --------------------------------------------------------------------------------------      |
| team_id                | integer | body  | The team ID to apply the settings to. Settings applied to hosts in no team if absent.       |
| enable_disk_encryption | boolean | body  | Whether disk encryption should be enforced on devices that belong to the team (or no team). |

#### Example

`PATCH /api/v1/fleet/mdm/apple/settings`

##### Default response

`204`

### Get disk encryption statistics

_Available in Fleet Premium_

Get aggregate status counts of disk encryption enforced on hosts.

The summary can optionally be filtered by team id.

`GET /api/v1/fleet/mdm/apple/filevault/summary`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| team_id                   | string | query | _Available in Fleet Premium_ The team id to filter the summary.            |

#### Example

Get aggregate status counts of Apple disk encryption profiles applying to macOS hosts enrolled to Fleet's MDM that are not assigned to any team.

`GET /api/v1/fleet/mdm/apple/filevault/summary`

##### Default response

`Status: 200`

```json
{
  "verified": 123,
  "verifying": 123,
  "action_required": 123,
  "enforcing": 123,
  "failed": 123,
  "removing_enforcement": 123
}
```

### Get macOS settings statistics

Get aggregate status counts of all macOS settings (configuraiton profiles and disk encryption) enforced on hosts.

For Fleet Premium uses, the statistics can
optionally be filtered by team id. If no team id is specified, team profiles are excluded from the
results (i.e., only profiles that are associated with "No team" are listed).

`GET /api/v1/fleet/mdm/apple/profiles/summary`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| team_id                   | string | query | _Available in Fleet Premium_ The team id to filter profiles.              |

#### Example

Get aggregate status counts of MDM profiles applying to macOS hosts enrolled to Fleet's MDM that are not assigned to any team.

`GET /api/v1/fleet/mdm/apple/profiles/summary`

##### Default response

`Status: 200`

```json
{
  "verified": 123,
  "verifying": 123,
  "failed": 123,
  "pending": 123
}
```

### Run custom MDM command

This endpoint tells Fleet to run a custom MDM command, on the targeted macOS hosts, the next time they come online.

`POST /api/v1/fleet/mdm/apple/enqueue`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| command                   | string | json  | A base64-encoded MDM command as described in [Apple's documentation](https://developer.apple.com/documentation/devicemanagement/commands_and_queries). Supported formats are standard ([RFC 4648](https://www.rfc-editor.org/rfc/rfc4648.html)) and raw (unpadded) encoding ([RFC 4648 section 3.2](https://www.rfc-editor.org/rfc/rfc4648.html#section-3.2)) |
| device_ids                | array  | json  | An array of host UUIDs enrolled in Fleet's MDM on which the command should run.                   |

Note that the `EraseDevice` and `DeviceLock` commands are _available in Fleet Premium_ only.

#### Example

`POST /api/v1/fleet/mdm/apple/enqueue`

##### Default response

`Status: 200`

```json
{
  "command_uuid": "a2064cef-0000-1234-afb9-283e3c1d487e",
  "request_type": "ProfileList"
}
```

### Get custom MDM command results

This endpoint returns the results for a specific custom MDM command.

`GET /api/v1/fleet/mdm/apple/commandresults`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| command_uuid              | string | query | The unique identifier of the command.                                     |

#### Example

`GET /api/v1/fleet/mdm/apple/commandresults?command_uuid=a2064cef-0000-1234-afb9-283e3c1d487e`

##### Default response

`Status: 200`

```json
{
  "results": [
    {
      "device_id": "145cafeb-87c7-4869-84d5-e4118a927746",
      "command_uuid": "a2064cef-0000-1234-afb9-283e3c1d487e",
      "status": "Acknowledged",
      "updated_at": "2023-04-04:00:00Z",
      "request_type": "ProfileList",
      "hostname": "mycomputer",
      "result": "PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0iVVRGLTgiPz4KPCFET0NUWVBFIHBsaXN0IFBVQkxJQyAiLS8vQXBwbGUvL0RURCBQTElTVCAxLjAvL0VOIiAiaHR0cDovL3d3dy5hcHBsZS5jb20vRFREcy9Qcm9wZXJ0eUxpc3QtMS4wLmR0ZCI-CjxwbGlzdCB2ZXJzaW9uPSIxLjAiPgo8ZGljdD4KICAgIDxrZXk-Q29tbWFuZDwva2V5PgogICAgPGRpY3Q-CiAgICAgICAgPGtleT5NYW5hZ2VkT25seTwva2V5PgogICAgICAgIDxmYWxzZS8-CiAgICAgICAgPGtleT5SZXF1ZXN0VHlwZTwva2V5PgogICAgICAgIDxzdHJpbmc-UHJvZmlsZUxpc3Q8L3N0cmluZz4KICAgIDwvZGljdD4KICAgIDxrZXk-Q29tbWFuZFVVSUQ8L2tleT4KICAgIDxzdHJpbmc-MDAwMV9Qcm9maWxlTGlzdDwvc3RyaW5nPgo8L2RpY3Q-CjwvcGxpc3Q-"
    }
  ]
}
```

### List custom MDM commands

This endpoint returns the list of custom MDM commands that have been executed.

`GET /api/v1/fleet/mdm/apple/commands`

#### Parameters

| Name                      | Type    | In    | Description                                                               |
| ------------------------- | ------  | ----- | ------------------------------------------------------------------------- |
| page                      | integer | query | Page number of the results to fetch.                                      |
| per_page                  | integer | query | Results per page.                                                         |
| order_key                 | string  | query | What to order results by. Can be any field listed in the `results` array example below. |
| order_direction           | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |

#### Example

`GET /api/v1/fleet/mdm/apple/commands?per_page=5`

##### Default response

`Status: 200`

```json
{
  "results": [
    {
      "device_id": "145cafeb-87c7-4869-84d5-e4118a927746",
      "command_uuid": "a2064cef-0000-1234-afb9-283e3c1d487e",
      "status": "Acknowledged",
      "updated_at": "2023-04-04:00:00Z",
      "request_type": "ProfileList",
      "hostname": "mycomputer"
    }
  ]
}
```

### Set custom MDM setup enrollment profile

_Available in Fleet Premium_

Sets the custom MDM setup enrollment profile for a team or no team.

`POST /api/v1/fleet/mdm/apple/enrollment_profile`

#### Parameters

| Name                      | Type    | In    | Description                                                                   |
| ------------------------- | ------  | ----- | -------------------------------------------------------------------------     |
| team_id                   | integer | json  | The team id this custom enrollment profile applies to, or no team if omitted. |
| name                      | string  | json  | The filename of the uploaded custom enrollment profile.                       |
| enrollment_profile        | object  | json  | The custom enrollment profile's json, as documented in https://developer.apple.com/documentation/devicemanagement/profile. |

#### Example

`POST /api/v1/fleet/mdm/apple/enrollment_profile`

##### Default response

`Status: 200`

```json
{
  "team_id": 123,
  "name": "dep_profile.json",
  "uploaded_at": "2023-04-04:00:00Z",
  "enrollment_profile": {
    "is_mandatory": true,
    "is_mdm_removable": false
  }
}
```

### Get custom MDM setup enrollment profile

_Available in Fleet Premium_

Gets the custom MDM setup enrollment profile for a team or no team.

`GET /api/v1/fleet/mdm/apple/enrollment_profile`

#### Parameters

| Name                      | Type    | In    | Description                                                                           |
| ------------------------- | ------  | ----- | -------------------------------------------------------------------------             |
| team_id                   | integer | query | The team id for which to return the custom enrollment profile, or no team if omitted. |

#### Example

`GET /api/v1/fleet/mdm/apple/enrollment_profile?team_id=123`

##### Default response

`Status: 200`

```json
{
  "team_id": 123,
  "name": "dep_profile.json",
  "uploaded_at": "2023-04-04:00:00Z",
  "enrollment_profile": {
    "is_mandatory": true,
    "is_mdm_removable": false
  }
}
```

### Delete custom MDM setup enrollment profile

_Available in Fleet Premium_

Deletes the custom MDM setup enrollment profile assigned to a team or no team.

`DELETE /api/v1/fleet/mdm/apple/enrollment_profile`

#### Parameters

| Name                      | Type    | In    | Description                                                                           |
| ------------------------- | ------  | ----- | -------------------------------------------------------------------------             |
| team_id                   | integer | query | The team id for which to delete the custom enrollment profile, or no team if omitted. |

#### Example

`DELETE /api/v1/fleet/mdm/apple/enrollment_profile?team_id=123`

##### Default response

`Status: 204`

### Get Apple Push Notification service (APNs)

`GET /api/v1/fleet/mdm/apple`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/mdm/apple`

##### Default response

`Status: 200`

```json
{
  "common_name": "APSP:04u52i98aewuh-xxxx-xxxx-xxxx-xxxx",
  "serial_number": "1234567890987654321",
  "issuer": "Apple Application Integration 2 Certification Authority",
  "renew_date": "2023-09-30T00:00:00Z"
}
```

### Get Apple Business Manager (ABM)

_Available in Fleet Premium_

`GET /api/v1/fleet/mdm/apple_bm`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/mdm/apple_bm`

##### Default response

`Status: 200`

```json
{
  "apple_id": "apple@example.com",
  "org_name": "Fleet Device Management",
  "mdm_server_url": "https://example.com/mdm/apple/mdm",
  "renew_date": "2023-11-29T00:00:00Z",
  "default_team": ""
}
```

### Turn off MDM for a host

`PATCH /api/v1/fleet/mdm/hosts/{id}/unenroll`

#### Parameters

| Name | Type    | In   | Description                           |
| ---- | ------- | ---- | ------------------------------------- |
| id   | integer | path | **Required.** The host's ID in Fleet. |

#### Example

`PATCH /api/v1/fleet/mdm/hosts/42/unenroll`

##### Default response

`Status: 200`


### Upload a bootstrap package

_Available in Fleet Premium_

Upload a bootstrap package that will be automatically installed during DEP setup.

`POST /api/v1/fleet/mdm/apple/bootstrap`

#### Parameters

| Name    | Type   | In   | Description                                                                                                                                                                                                            |
| ------- | ------ | ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| package | file   | form | **Required**. The bootstrap package installer. It must be a signed `pkg` file.                                                                                                                                         |
| team_id | string | form | The team id for the package. If specified, the package will be installed to hosts that are assigned to the specified team. If not specified, the package will be installed to hosts that are not assigned to any team. |

#### Example

Upload a bootstrap package that will be installed to macOS hosts enrolled to MDM that are
assigned to a team. Note that in this example the form data specifies `team_id` in addition to
`package`.

`POST /api/v1/fleet/mdm/apple/profiles`

##### Request headers

```http
Content-Length: 850
Content-Type: multipart/form-data; boundary=------------------------f02md47480und42y
```

##### Request body

```http
--------------------------f02md47480und42y
Content-Disposition: form-data; name="team_id"
1
--------------------------f02md47480und42y
Content-Disposition: form-data; name="package"; filename="bootstrap-package.pkg"
Content-Type: application/octet-stream
<BINARY_DATA>
--------------------------f02md47480und42y--
```

##### Default response

`Status: 200`

### Get metadata about a bootstrap package

_Available in Fleet Premium_

Get information about a bootstrap package that was uploaded to Fleet.

`GET /api/v1/fleet/mdm/apple/bootstrap/{team_id}/metadata`

#### Parameters

| Name       | Type    | In    | Description                                                                                                                                                                                                        |
| -------    | ------  | ---   | ---------------------------------------------------------------------------------------------------------------------------------------------------------                                                          |
| team_id    | string  | url   | **Required** The team id for the package. Zero (0) can be specified to get information about the bootstrap package for hosts that don't belong to a team.                                                          |
| for_update | boolean | query | If set to `true`, the authorization will be for a `write` action instead of a `read`. Useful for the write-only `gitops` role when requesting the bootstrap metadata to check if the package needs to be replaced. |

#### Example

`GET /api/v1/fleet/mdm/apple/bootstrap/0/metadata`

##### Default response

`Status: 200`

```json
{
  "name": "bootstrap-package.pkg",
  "team_id": 0,
  "sha256": "6bebb4433322fd52837de9e4787de534b4089ac645b0692dfb74d000438da4a3",
  "token": "AA598E2A-7952-46E3-B89D-526D45F7E233",
  "created_at": "2023-04-20T13:02:05Z"
}
```

In the response above:

- `token` is the value you can use to [download a bootstrap package](#download-a-bootstrap-package)
- `sha256` is the SHA256 digest of the bytes of the bootstrap package file.

### Delete a bootstrap package

_Available in Fleet Premium_

Delete a team's bootstrap package.

`DELETE /api/v1/fleet/mdm/apple/bootstrap/{team_id}`

#### Parameters

| Name    | Type   | In  | Description                                                                                                                                               |
| ------- | ------ | --- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| team_id | string | url | **Required** The team id for the package. Zero (0) can be specified to get information about the bootstrap package for hosts that don't belong to a team. |


#### Example

`DELETE /api/v1/fleet/mdm/apple/bootstrap/1`

##### Default response

`Status: 200`

### Download a bootstrap package

_Available in Fleet Premium_

Download a bootstrap package.

`GET /api/v1/fleet/mdm/apple/bootstrap`

#### Parameters

| Name  | Type   | In    | Description                                      |
| ----- | ------ | ----- | ------------------------------------------------ |
| token | string | query | **Required** The token of the bootstrap package. |

#### Example

`GET /api/v1/fleet/mdm/apple/bootstrap?token=AA598E2A-7952-46E3-B89D-526D45F7E233`

##### Default response

`Status: 200`

```http
Status: 200
Content-Type: application/octet-stream
Content-Disposition: attachment
Content-Length: <length>
Body: <blob>
```

### Get a summary of bootstrap package status

_Available in Fleet Premium_

Get aggregate status counts of bootstrap packages delivered to DEP enrolled hosts.

The summary can optionally be filtered by team id.

`GET /api/v1/fleet/mdm/apple/bootstrap/summary`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| team_id                   | string | query | The team id to filter the summary.                                        |

#### Example

`GET /api/v1/fleet/mdm/apple/bootstrap/summary`

##### Default response

`Status: 200`

```json
{
  "installed": 10,
  "failed": 1,
  "pending": 4
}
```

### Turn on end user authentication for macOS setup

_Available in Fleet Premium_

`PATCH /api/v1/fleet/mdm/apple/setup`

#### Parameters

| Name                           | Type    | In    | Description                                                                                 |
| -------------          | ------  | ----  | --------------------------------------------------------------------------------------      |
| team_id                        | integer | body  | The team ID to apply the settings to. Settings applied to hosts in no team if absent.       |
| enable_end_user_authentication | boolean | body  | Whether end user authentication should be enabled for new macOS devices that automatically enroll to the team (or no team). |

#### Example

`PATCH /api/v1/fleet/mdm/apple/setup`

##### Request body

```json
{
  "team_id": 1,
  "enabled_end_user_authentication": true
}
```

##### Default response

`Status: 204`



### Upload an EULA file

_Available in Fleet Premium_

Upload an EULA that will be shown during the DEP flow.

`POST /api/v1/fleet/mdm/apple/setup/eula`

#### Parameters

| Name | Type | In   | Description                                       |
| ---- | ---- | ---- | ------------------------------------------------- |
| eula | file | form | **Required**. A PDF document containing the EULA. |

#### Example

`POST /api/v1/fleet/mdm/apple/setup/eula`

##### Request headers

```http
Content-Length: 850
Content-Type: multipart/form-data; boundary=------------------------f02md47480und42y
```

##### Request body

```http
--------------------------f02md47480und42y
Content-Disposition: form-data; name="eula"; filename="eula.pdf"
Content-Type: application/octet-stream
<BINARY_DATA>
--------------------------f02md47480und42y--
```

##### Default response

`Status: 200`

### Get metadata about an EULA file

_Available in Fleet Premium_

Get information about the EULA file that was uploaded to Fleet. If no EULA was previously uploaded, this endpoint returns a `404` status code.

`GET /api/v1/fleet/mdm/apple/setup/eula/metadata`

#### Example

`GET /api/v1/fleet/mdm/apple/setup/eula/metadata`

##### Default response

`Status: 200`

```json
{
  "name": "eula.pdf",
  "token": "AA598E2A-7952-46E3-B89D-526D45F7E233",
  "created_at": "2023-04-20T13:02:05Z"
}
```

In the response above:

- `token` is the value you can use to [download an EULA](#download-an-eula-file)

### Delete an EULA file

_Available in Fleet Premium_

Delete an EULA file.

`DELETE /api/v1/fleet/mdm/apple/setup/eula/{token}`

#### Parameters

| Name  | Type   | In    | Description                              |
| ----- | ------ | ----- | ---------------------------------------- |
| token | string | path  | **Required** The token of the EULA file. |

#### Example

`DELETE /api/v1/fleet/mdm/apple/setup/eula/AA598E2A-7952-46E3-B89D-526D45F7E233`

##### Default response

`Status: 200`

### Download an EULA file

_Available in Fleet Premium_

Download an EULA file

`GET /api/v1/fleet/mdm/apple/setup/eula/{token}`

#### Parameters

| Name  | Type   | In    | Description                              |
| ----- | ------ | ----- | ---------------------------------------- |
| token | string | path  | **Required** The token of the EULA file. |

#### Example

`GET /api/v1/fleet/mdm/apple/setup/eula/AA598E2A-7952-46E3-B89D-526D45F7E233`

##### Default response

`Status: 200`

```http
Status: 200
Content-Type: application/pdf
Content-Disposition: attachment
Content-Length: <length>
Body: <blob>
```

---

## Policies

- [List policies](#list-policies)
- [Count policies](#count-policies)
- [Get policy by ID](#get-policy-by-id)
- [Add policy](#add-policy)
- [Remove policies](#remove-policies)
- [Edit policy](#edit-policy)
- [Run automation for all failing hosts of a policy](#run-automation-for-all-failing-hosts-of-a-policy)

Policies are yes or no questions you can ask about your hosts.

Policies in Fleet are defined by osquery queries.

A passing host answers "yes" to a policy if the host returns results for a policy's query.

A failing host answers "no" to a policy if the host does not return results for a policy's query.

For example, a policy might ask “Is Gatekeeper enabled on macOS devices?“ This policy's osquery query might look like the following: `SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;`

### List policies

`GET /api/v1/fleet/global/policies`

#### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                                                                                                                                                                                 |
| ----------------------- | ------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                                                                                                                                                                                        |
| per_page                | integer | query | Results per page.

#### Example

`GET /api/v1/fleet/global/policies`

##### Default response

`Status: 200`

```json
{
  "policies": [
    {
      "id": 1,
      "name": "Gatekeeper enabled",
      "query": "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
      "description": "Checks if gatekeeper is enabled on macOS devices",
      "critical": false,
      "author_id": 42,
      "author_name": "John",
      "author_email": "john@example.com",
      "team_id": null,
      "resolution": "Resolution steps",
      "platform": "darwin",
      "created_at": "2021-12-15T15:23:57Z",
      "updated_at": "2021-12-15T15:23:57Z",
      "passing_host_count": 2000,
      "failing_host_count": 300
    },
    {
      "id": 2,
      "name": "Windows machines with encrypted hard disks",
      "query": "SELECT 1 FROM bitlocker_info WHERE protection_status = 1;",
      "description": "Checks if the hard disk is encrypted on Windows devices",
      "critical": true,
      "author_id": 43,
      "author_name": "Alice",
      "author_email": "alice@example.com",
      "team_id": null,
      "resolution": "Resolution steps",
      "platform": "windows",
      "created_at": "2021-12-31T14:52:27Z",
      "updated_at": "2022-02-10T20:59:35Z",
      "passing_host_count": 2300,
      "failing_host_count": 0
    }
  ]
}
```

---

### Count policies

`GET /api/v1/fleet/policies/count`


#### Parameters
| Name               | Type    | In   | Description                                                                                                   |
| ------------------ | ------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| query                 | string | query | Search query keywords. Searchable fields include `name`.  |

#### Example

`GET /api/v1/fleet/policies/count`

##### Default response

`Status: 200`

```json
{
  "count": 43
}
```

---

### Get policy by ID

`GET /api/v1/fleet/global/policies/{id}`

#### Parameters

| Name               | Type    | In   | Description                                                                                                   |
| ------------------ | ------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| id                 | integer | path | **Required.** The policy's ID.                                                                                |

#### Example

`GET /api/v1/fleet/global/policies/1`

##### Default response

`Status: 200`

```json
{
  "policy": {
      "id": 1,
      "name": "Gatekeeper enabled",
      "query": "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
      "description": "Checks if gatekeeper is enabled on macOS devices",
      "critical": false,
      "author_id": 42,
      "author_name": "John",
      "author_email": "john@example.com",
      "team_id": null,
      "resolution": "Resolution steps",
      "platform": "darwin",
      "created_at": "2021-12-15T15:23:57Z",
      "updated_at": "2021-12-15T15:23:57Z",
      "passing_host_count": 2000,
      "failing_host_count": 300
    }
}
```

### Add policy

There are two ways of adding a policy:
1. by setting "name", "query", "description". This is the preferred way.
2. (Legacy) re-using the data of an existing query, by setting "query_id". If "query_id" is set,
then "query" must not be set, and "name" and "description" are ignored.

An error is returned if both "query" and "query_id" are set on the request.

`POST /api/v1/fleet/global/policies`

#### Parameters

| Name        | Type    | In   | Description                          |
| ----------  | ------- | ---- | ------------------------------------ |
| name        | string  | body | The query's name.                    |
| query       | string  | body | The query in SQL.                    |
| description | string  | body | The query's description.             |
| resolution  | string  | body | The resolution steps for the policy. |
| query_id    | integer | body | An existing query's ID (legacy).     |
| platform    | string  | body | Comma-separated target platforms, currently supported values are "windows", "linux", "darwin". The default, an empty string means target all platforms. |
| critical    | boolean | body | _Available in Fleet Premium_ Mark policy as critical/high impact. |

Either `query` or `query_id` must be provided.

#### Example Add Policy

`POST /api/v1/fleet/global/policies`

#### Request body

```json
{
  "name": "Gatekeeper enabled",
  "query": "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
  "description": "Checks if gatekeeper is enabled on macOS devices",
  "resolution": "Resolution steps",
  "platform": "darwin",
  "critical": true
}
```

##### Default response

`Status: 200`

```json
{
  "policy": {
    "id": 43,
    "name": "Gatekeeper enabled",
    "query": "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
    "description": "Checks if gatekeeper is enabled on macOS devices",
    "critical": true,
    "author_id": 42,
    "author_name": "John",
    "author_email": "john@example.com",
    "team_id": null,
    "resolution": "Resolution steps",
    "platform": "darwin",
    "created_at": "2022-03-17T20:15:55Z",
    "updated_at": "2022-03-17T20:15:55Z",
    "passing_host_count": 0,
    "failing_host_count": 0
  }
}
```

#### Example Legacy Add Policy

`POST /api/v1/fleet/global/policies`

#### Request body

```json
{
  "query_id": 12
}
```

Where `query_id` references an existing `query`.

##### Default response

`Status: 200`

```json
{
  "policy": {
    "id": 43,
    "name": "Gatekeeper enabled",
    "query": "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
    "description": "Checks if gatekeeper is enabled on macOS devices",
    "critical": true,
    "author_id": 42,
    "author_name": "John",
    "author_email": "john@example.com",
    "team_id": null,
    "resolution": "Resolution steps",
    "platform": "darwin",
    "created_at": "2022-03-17T20:15:55Z",
    "updated_at": "2022-03-17T20:15:55Z",
    "passing_host_count": 0,
    "failing_host_count": 0
  }
}
```

### Remove policies

`POST /api/v1/fleet/global/policies/delete`

#### Parameters

| Name     | Type    | In   | Description                                       |
| -------- | ------- | ---- | ------------------------------------------------- |
| ids      | list    | body | **Required.** The IDs of the policies to delete.  |

#### Example

`POST /api/v1/fleet/global/policies/delete`

#### Request body

```json
{
  "ids": [ 1 ]
}
```

##### Default response

`Status: 200`

```json
{
  "deleted": 1
}
```

### Edit policy

`PATCH /api/v1/fleet/global/policies/{policy_id}`

#### Parameters

| Name        | Type    | In   | Description                          |
| ----------  | ------- | ---- | ------------------------------------ |
| id          | integer | path | The policy's ID.                     |
| name        | string  | body | The query's name.                    |
| query       | string  | body | The query in SQL.                    |
| description | string  | body | The query's description.             |
| resolution  | string  | body | The resolution steps for the policy. |
| platform    | string  | body | Comma-separated target platforms, currently supported values are "windows", "linux", "darwin". The default, an empty string means target all platforms. |
| critical    | boolean | body | _Available in Fleet Premium_ Mark policy as critical/high impact. |

#### Example Edit Policy

`PATCH /api/v1/fleet/global/policies/42`

##### Request body

```json
{
  "name": "Gatekeeper enabled",
  "query": "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
  "description": "Checks if gatekeeper is enabled on macOS devices",
  "critical": true,
  "resolution": "Resolution steps",
  "platform": "darwin"
}
```

##### Default response

`Status: 200`

```json
{
  "policy": {
    "id": 42,
    "name": "Gatekeeper enabled",
    "query": "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
    "description": "Checks if gatekeeper is enabled on macOS devices",
    "critical": true,
    "author_id": 43,
    "author_name": "John",
    "author_email": "john@example.com",
    "team_id": null,
    "resolution": "Resolution steps",
    "platform": "darwin",
    "created_at": "2022-03-17T20:15:55Z",
    "updated_at": "2022-03-17T20:15:55Z",
    "passing_host_count": 0,
    "failing_host_count": 0
  }
}
```

### Run Automation for all failing hosts of a policy.

Normally automations (Webhook/Integrations) runs on all hosts when a policy-check
fails but didn't fail before. This feature to mark policies to call automation for
all hosts that already fail the policy, too and possibly again.

`POST /api/v1/fleet/automations/reset`

#### Parameters

| Name        | Type     | In   | Description                                              |
| ----------  | -------- | ---- | -------------------------------------------------------- |
| team_ids    | list     | body | Run automation for all hosts in policies of these teams  |
| policy_ids  | list     | body | Run automations for all hosts these policies             |

_Teams are available in Fleet Premium_

#### Example Edit Policy

`POST /api/v1/fleet/automations/reset`

##### Request body

```json
{
    "team_ids": [1],
    "policy_ids": [1, 2, 3]
}
```

##### Default response

`Status: 200`

```json
{}
```

---

### Team policies

- [List team policies](#list-team-policies)
- [Count team policies](#count-team-policies)
- [Get team policy by ID](#get-team-policy-by-id)
- [Add team policy](#add-team-policy)
- [Remove team policies](#remove-team-policies)
- [Edit team policy](#edit-team-policy)

_Available in Fleet Premium_

Team policies work the same as policies, but at the team level.

### List team policies

`GET /api/v1/fleet/teams/{id}/policies`

#### Parameters

| Name               | Type    | In   | Description                                                                                                   |
| ------------------ | ------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| id                 | integer | url  | Required. Defines what team id to operate on                                                                            |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                                                                                                                                                                                        |
| per_page                | integer | query | Results per page. |
#### Example

`GET /api/v1/fleet/teams/1/policies`

##### Default response

`Status: 200`

```json
{
  "policies": [
    {
      "id": 1,
      "name": "Gatekeeper enabled",
      "query": "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
      "description": "Checks if gatekeeper is enabled on macOS devices",
      "critical": true,
      "author_id": 42,
      "author_name": "John",
      "author_email": "john@example.com",
      "team_id": 1,
      "resolution": "Resolution steps",
      "platform": "darwin",
      "created_at": "2021-12-16T14:37:37Z",
      "updated_at": "2021-12-16T16:39:00Z",
      "passing_host_count": 2000,
      "failing_host_count": 300
    },
    {
      "id": 2,
      "name": "Windows machines with encrypted hard disks",
      "query": "SELECT 1 FROM bitlocker_info WHERE protection_status = 1;",
      "description": "Checks if the hard disk is encrypted on Windows devices",
      "critical": false,
      "author_id": 43,
      "author_name": "Alice",
      "author_email": "alice@example.com",
      "team_id": 1,
      "resolution": "Resolution steps",
      "platform": "windows",
      "created_at": "2021-12-16T14:37:37Z",
      "updated_at": "2021-12-16T16:39:00Z",
      "passing_host_count": 2300,
      "failing_host_count": 0
    }
  ],
  "inherited_policies": [
    {
      "id": 136,
      "name": "Arbitrary Test Policy (all platforms) (all teams)",
      "query": "SELECT 1 FROM osquery_info WHERE 1=1;",
      "description": "If you're seeing this, mostly likely this is because someone is testing out failing policies in dogfood. You can ignore this.",
      "critical": true,
      "author_id": 77,
      "author_name": "Test Admin",
      "author_email": "test@admin.com",
      "team_id": null,
      "resolution": "To make it pass, change \"1=0\" to \"1=1\". To make it fail, change \"1=1\" to \"1=0\".",
      "platform": "darwin,windows,linux",
      "created_at": "2022-08-04T19:30:18Z",
      "updated_at": "2022-08-30T15:08:26Z",
      "passing_host_count": 10,
      "failing_host_count": 9
    }
  ]
}
```

### Count team policies

`GET /api/v1/fleet/team/{team_id}/policies/count`

#### Parameters
| Name               | Type    | In   | Description                                                                                                   |
| ------------------ | ------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| query                 | string | query | Search query keywords. Searchable fields include `name`. |

#### Example

`GET /api/v1/fleet/team/1/policies/count`

##### Default response

`Status: 200`

```json
{
  "count": 43
}
```

---

### Get team policy by ID

`GET /api/v1/fleet/teams/{team_id}/policies/{id}`

#### Parameters

| Name               | Type    | In   | Description                                                                                                   |
| ------------------ | ------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| team_id            | integer | url  | Defines what team id to operate on                                                                            |
| id                 | integer | path | **Required.** The policy's ID.                                                                                |

#### Example

`GET /api/v1/fleet/teams/1/policies/43`

##### Default response

`Status: 200`

```json
{
  "policy": {
    "id": 43,
    "name": "Gatekeeper enabled",
    "query": "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
    "description": "Checks if gatekeeper is enabled on macOS devices",
    "critical": true,
    "author_id": 42,
    "author_name": "John",
    "author_email": "john@example.com",
    "team_id": 1,
    "resolution": "Resolution steps",
    "platform": "darwin",
    "created_at": "2021-12-16T14:37:37Z",
    "updated_at": "2021-12-16T16:39:00Z",
    "passing_host_count": 0,
    "failing_host_count": 0
  }
}
```

### Add team policy

The semantics for creating a team policy are the same as for global policies, see [Add policy](#add-policy).

`POST /api/v1/fleet/teams/{team_id}/policies`

#### Parameters

| Name        | Type    | In   | Description                          |
| ----------  | ------- | ---- | ------------------------------------ |
| team_id     | integer | url  | Defines what team id to operate on.  |
| name        | string  | body | The query's name.                    |
| query       | string  | body | The query in SQL.                    |
| description | string  | body | The query's description.             |
| resolution  | string  | body | The resolution steps for the policy. |
| query_id    | integer | body | An existing query's ID (legacy).     |
| platform    | string  | body | Comma-separated target platforms, currently supported values are "windows", "linux", "darwin". The default, an empty string means target all platforms. |
| critical    | boolean | body | _Available in Fleet Premium_ Mark policy as critical/high impact. |

Either `query` or `query_id` must be provided.

#### Example

`POST /api/v1/fleet/teams/1/policies`

##### Request body

```json
{
  "name": "Gatekeeper enabled",
  "query": "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
  "description": "Checks if gatekeeper is enabled on macOS devices",
  "critical": true,
  "resolution": "Resolution steps",
  "platform": "darwin"
}
```

##### Default response

`Status: 200`

```json
{
  "policy": {
    "id": 43,
    "name": "Gatekeeper enabled",
    "query": "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
    "description": "Checks if gatekeeper is enabled on macOS devices",
    "critical": true,
    "author_id": 42,
    "author_name": "John",
    "author_email": "john@example.com",
    "team_id": 1,
    "resolution": "Resolution steps",
    "platform": "darwin",
    "created_at": "2021-12-16T14:37:37Z",
    "updated_at": "2021-12-16T16:39:00Z",
    "passing_host_count": 0,
    "failing_host_count": 0
  }
}
```

### Remove team policies

`POST /api/v1/fleet/teams/{team_id}/policies/delete`

#### Parameters

| Name     | Type    | In   | Description                                       |
| -------- | ------- | ---- | ------------------------------------------------- |
| team_id  | integer | url  | Defines what team id to operate on                |
| ids      | list    | body | **Required.** The IDs of the policies to delete.  |

#### Example

`POST /api/v1/fleet/teams/1/policies/delete`

##### Request body

```json
{
  "ids": [ 1 ]
}
```

##### Default response

`Status: 200`

```json
{
  "deleted": 1
}
```

### Edit team policy

`PATCH /api/v1/fleet/teams/{team_id}/policies/{policy_id}`

#### Parameters

| Name        | Type    | In   | Description                          |
| ----------  | ------- | ---- | ------------------------------------ |
| team_id     | integer | path | The team's ID.                       |
| policy_id   | integer | path | The policy's ID.                     |
| name        | string  | body | The query's name.                    |
| query       | string  | body | The query in SQL.                    |
| description | string  | body | The query's description.             |
| resolution  | string  | body | The resolution steps for the policy. |
| platform    | string  | body | Comma-separated target platforms, currently supported values are "windows", "linux", "darwin". The default, an empty string means target all platforms. |
| critical    | boolean | body | _Available in Fleet Premium_ Mark policy as critical/high impact. |

#### Example Edit Policy

`PATCH /api/v1/fleet/teams/2/policies/42`

##### Request body

```json
{
  "name": "Gatekeeper enabled",
  "query": "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
  "description": "Checks if gatekeeper is enabled on macOS devices",
  "critical": true,
  "resolution": "Resolution steps",
  "platform": "darwin"
}
```

##### Default response

`Status: 200`

```json
{
  "policy": {
    "id": 42,
    "name": "Gatekeeper enabled",
    "query": "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
    "description": "Checks if gatekeeper is enabled on macOS devices",
    "critical": true,
    "author_id": 43,
    "author_name": "John",
    "author_email": "john@example.com",
    "resolution": "Resolution steps",
    "platform": "darwin",
    "team_id": 2,
    "created_at": "2021-12-16T14:37:37Z",
    "updated_at": "2021-12-16T16:39:00Z",
    "passing_host_count": 0,
    "failing_host_count": 0
  }
}
```

---

## Queries

- [Get query](#get-query)
- [List queries](#list-queries)
- [Create query](#create-query)
- [Modify query](#modify-query)
- [Delete query by name](#delete-query-by-name)
- [Delete query by ID](#delete-query-by-id)
- [Delete queries](#delete-queries)
- [Run live query](#run-live-query)

### Get query

Returns the query specified by ID.

`GET /api/v1/fleet/queries/{id}`

#### Parameters

| Name | Type    | In   | Description                                |
| ---- | ------- | ---- | ------------------------------------------ |
| id   | integer | path | **Required**. The id of the desired query. |

#### Example

`GET /api/v1/fleet/queries/31`

##### Default response

`Status: 200`

```json
{
  "query": {
    "created_at": "2021-01-19T17:08:24Z",
    "updated_at": "2021-01-19T17:08:24Z",
    "id": 31,
    "name": "centos_hosts",
    "description": "",
    "query": "select 1 from os_version where platform = \"centos\";",
    "team_id": null,
    "interval": 3600,
    "platform": "",
    "min_osquery_version": "",
    "automations_enabled": true,
    "logging": "snapshot",
    "saved": true,
    "observer_can_run": true,
    "author_id": 1,
    "author_name": "John",
    "author_email": "john@example.com",
    "packs": [
      {
        "created_at": "2021-01-19T17:08:31Z",
        "updated_at": "2021-01-19T17:08:31Z",
        "id": 14,
        "name": "test_pack",
        "description": "",
        "platform": "",
        "disabled": false
      }
    ],
    "stats": {
      "system_time_p50": 1.32,
      "system_time_p95": 4.02,
      "user_time_p50": 3.55,
      "user_time_p95": 3.00,
      "total_executions": 3920
    }
  }
}
```

### List queries

Returns a list of global queries or team queries.

`GET /api/v1/fleet/queries`

#### Parameters

| Name            | Type    | In    | Description                                                                                                                   |
| --------------- | ------- | ----- | ----------------------------------------------------------------------------------------------------------------------------- |
| order_key       | string  | query | What to order results by. Can be any column in the queries table.                                                             |
| order_direction | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |
| team_id         | integer | query | The ID of the parent team for the queries to be listed. When omitted, returns global queries.                  |


#### Example

`GET /api/v1/fleet/queries`

##### Default response

`Status: 200`

```json
{
"queries": [
  {
    "created_at": "2021-01-04T21:19:57Z",
    "updated_at": "2021-01-04T21:19:57Z",
    "id": 1,
    "name": "query1",
    "description": "query",
    "query": "SELECT * FROM osquery_info",
    "team_id": null,
    "interval": 3600,
    "platform": "darwin,windows,linux",
    "min_osquery_version": "",
    "automations_enabled": true,
    "logging": "snapshot",
    "saved": true,
    "observer_can_run": true,
    "author_id": 1,
    "author_name": "noah",
    "author_email": "noah@example.com",
    "packs": [
      {
        "created_at": "2021-01-05T21:13:04Z",
        "updated_at": "2021-01-07T19:12:54Z",
        "id": 1,
        "name": "Pack",
        "description": "Pack",
        "platform": "",
        "disabled": true
      }
    ],
    "stats": {
      "system_time_p50": 1.32,
      "system_time_p95": 4.02,
      "user_time_p50": 3.55,
      "user_time_p95": 3.00,
      "total_executions": 3920
    }
  },
  {
    "created_at": "2021-01-19T17:08:24Z",
    "updated_at": "2021-01-19T17:08:24Z",
    "id": 3,
    "name": "osquery_schedule",
    "description": "Report performance stats for each file in the query schedule.",
    "query": "select name, interval, executions, output_size, wall_time, (user_time/executions) as avg_user_time, (system_time/executions) as avg_system_time, average_memory, last_executed from osquery_schedule;",
    "team_id": null,
    "interval": 3600,
    "platform": "",
    "version": "",
    "automations_enabled": true,
    "logging": "differential",
    "saved": true,
    "observer_can_run": true,
    "author_id": 1,
    "author_name": "noah",
    "author_email": "noah@example.com",
    "packs": [
      {
        "created_at": "2021-01-19T17:08:31Z",
        "updated_at": "2021-01-19T17:08:31Z",
        "id": 14,
        "name": "test_pack",
        "description": "",
        "platform": "",
        "disabled": false
      }
    ],
    "stats": {
      "system_time_p50": null,
      "system_time_p95": null,
      "user_time_p50": null,
      "user_time_p95": null,
      "total_executions": null
    }
  }
]}
```

### Create query
Creates a global query or team query.

`POST /api/v1/fleet/queries`

#### Parameters

| Name                            | Type    | In   | Description                                                                                                                                            |
| ------------------------------- | ------- | ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| name                            | string  | body | **Required**. The name of the query.                                                                                                                   |
| query                           | string  | body | **Required**. The query in SQL syntax.                                                                                                                 |
| description                     | string  | body | The query's description.                                                                                                                               |
| observer_can_run                | bool    | body | Whether or not users with the `observer` role can run the query. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). This field is only relevant for the `observer` role. The `observer_plus` role can run any query and is not limited by this flag (`observer_plus` role was added in Fleet 4.30.0). |
| team_id                         | integer | body | The parent team to which the new query should be added. If omitted, the query will be global.                                           |
| interval                       | integer | body | The amount of time, in seconds, the query waits before running. Can be set to `0` to never run. Default: 0.       |
| platform                        | string  | body | The OS platforms where this query will run (other platforms ignored). Comma-separated string. If omitted, runs on all compatible platforms.                        |
| min_osquery_version             | string  | body | The minimum required osqueryd version installed on a host. If omitted, all osqueryd versions are acceptable.                                                                          |
| automations_enabled             | boolean | body | Whether to send data to the configured log destination according to the query's `interval`. |
| logging             | string  | body | The type of log output for this query. Valid values: `"snapshot"`(default), `"differential", or "differential_ignore_removals"`.                        |

#### Example

`POST /api/v1/fleet/queries`

##### Request body

```json
{
  "name": "new_query",
  "description": "This is a new query.",
  "query": "SELECT * FROM osquery_info",
  "interval": 3600, // Once per hour
  "platform": "darwin,windows,linux",
  "min_osquery_version": "",
  "automations_enabled": true,
  "logging": "snapshot"
}
```

##### Default response

`Status: 200`

```json
{
  "query": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 288,
    "name": "new_query",
    "query": "SELECT * FROM osquery_info",
    "description": "This is a new query.",
    "team_id": null,
    "interval": 3600,
    "platform": "darwin,windows,linux",
    "min_osquery_version": "",
    "automations_enabled": true,
    "logging": "snapshot",
    "saved": true,
    "author_id": 1,
    "author_name": "",
    "author_email": "",
    "observer_can_run": true,
    "packs": []
  }
}
```

### Modify query

Modifies the query specified by ID.

`PATCH /api/v1/fleet/queries/{id}`

#### Parameters

| Name                        | Type    | In   | Description                                                                                                                                            |
| --------------------------- | ------- | ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| id                          | integer | path | **Required.** The ID of the query.                                                                                                                     |
| name                        | string  | body | The name of the query.                                                                                                                                 |
| query                       | string  | body | The query in SQL syntax.                                                                                                                               |
| description                 | string  | body | The query's description.                                                                                                                               |
| observer_can_run            | bool    | body | Whether or not users with the `observer` role can run the query. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). This field is only relevant for the `observer` role. The `observer_plus` role can run any query and is not limited by this flag (`observer_plus` role was added in Fleet 4.30.0). |
| interval                   | integer | body | The amount of time, in seconds, the query waits before running. Can be set to `0` to never run. Default: 0.       |
| platform                    | string  | body | The OS platforms where this query will run (other platforms ignored). Comma-separated string. If set to "", runs on all compatible platforms.                    |
| min_osquery_version             | string  | body | The minimum required osqueryd version installed on a host. If omitted, all osqueryd versions are acceptable.                                                                          |
| automations_enabled             | boolean | body | Whether to send data to the configured log destination according to the query's `interval`. |
| logging             | string  | body | The type of log output for this query. Valid values: `"snapshot"`(default), `"differential", or "differential_ignore_removals"`.                        |

#### Example

`PATCH /api/v1/fleet/queries/2`

##### Request body

```json
{
  "name": "new_title_for_my_query",
  "interval": 3600, // Once per hour,
  "platform": "",
  "min_osquery_version": "",
  "automations_enabled": false
}
```

##### Default response

`Status: 200`

```json
{
  "query": {
    "created_at": "2021-01-22T17:23:27Z",
    "updated_at": "2021-01-22T17:23:27Z",
    "id": 288,
    "name": "new_title_for_my_query",
    "description": "This is a new query.",
    "query": "SELECT * FROM osquery_info",
    "team_id": null,
    "interval": 3600,
    "platform": "",
    "min_osquery_version": "",
    "automations_enabled": false,
    "logging": "snapshot",
    "saved": true,
    "author_id": 1,
    "author_name": "noah",
    "observer_can_run": true,
    "packs": []
  }
}
```

### Delete query by name

Deletes the query specified by name.

`DELETE /api/v1/fleet/queries/{name}`

#### Parameters

| Name | Type       | In   | Description                          |
| ---- | ---------- | ---- | ------------------------------------ |
| name | string     | path | **Required.** The name of the query. |
| team_id | integer | body | The ID of the parent team of the query to be deleted. If omitted, Fleet will search among queries in the global context. |

#### Example

`DELETE /api/v1/fleet/queries/{name}`

##### Default response

`Status: 200`


### Delete query by ID

Deletes the query specified by ID.

`DELETE /api/v1/fleet/queries/id/{id}`

#### Parameters

| Name | Type    | In   | Description                        |
| ---- | ------- | ---- | ---------------------------------- |
| id   | integer | path | **Required.** The ID of the query. |

#### Example

`DELETE /api/v1/fleet/queries/id/28`

##### Default response

`Status: 200`


### Delete queries

Deletes the queries specified by ID. Returns the count of queries successfully deleted.

`POST /api/v1/fleet/queries/delete`

#### Parameters

| Name | Type | In   | Description                           |
| ---- | ---- | ---- | ------------------------------------- |
| ids  | list | body | **Required.** The IDs of the queries. |

#### Example

`POST /api/v1/fleet/queries/delete`

##### Request body

```json
{
  "ids": [
    2, 24, 25
  ]
}
```

##### Default response

`Status: 200`

```json
{
  "deleted": 3
}
```

### Run live query

Run one or more live queries against the specified hosts and responds with the results
collected after 25 seconds.

If multiple queries are provided, they run concurrently. Response time is capped at 25 seconds from
when the API request was received, regardless of how many queries you are running, and regardless
whether all results have been gathered or not. This API does not return any results until the fixed
time period elapses, at which point all of the collected results are returned.

The fixed time period is configurable via environment variable on the Fleet server (eg.
`FLEET_LIVE_QUERY_REST_PERIOD=90s`). If setting a higher value, be sure that you do not exceed your
load balancer timeout.

> WARNING: This API endpoint collects responses in-memory (RAM) on the Fleet compute instance handling this request, which can overflow if the result set is large enough.  This has the potential to crash the process and/or cause an autoscaling event in your cloud provider, depending on how Fleet is deployed.

`GET /api/v1/fleet/queries/run`

#### Parameters


| Name      | Type   | In   | Description                                   |
| --------- | ------ | ---- | --------------------------------------------- |
| query_ids | array  | body | **Required**. The IDs of the saved queries to run. |
| host_ids  | array  | body | **Required**. The IDs of the hosts to target. |

#### Example

`GET /api/v1/fleet/queries/run`

##### Request body

```json
{
  "query_ids": [ 1, 2 ],
  "host_ids": [ 1, 4, 34, 27 ]
}
```

##### Default response

```json
{
  "summary": {
    "targeted_host_count": 4,
    "responded_host_count": 2
  },
  "live_query_results": [
    {
      "query_id": 2,
      "results": [
        {
          "host_id": 1,
          "rows": [
            {
              "build_distro": "10.12",
              "build_platform": "darwin",
              "config_hash": "7bb99fa2c8a998c9459ec71da3a84d66c592d6d3",
              "config_valid": "1",
              "extensions": "active",
              "instance_id": "9a2ec7bf-4946-46ea-93bf-455e0bcbd068",
              "pid": "23413",
              "platform_mask": "21",
              "start_time": "1635194306",
              "uuid": "4C182AC7-75F7-5AF4-A74B-1E165ED35742",
              "version": "4.9.0",
              "watcher": "23412"
            }
          ],
          "error": null
        },
        {
          "host_id": 2,
          "rows": [],
          "error": "no such table: os_version"
        }
      ]
    }
  ]
}
```

---

## Schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility.
> Please use the [queries](#queries) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

- [Get schedule (deprecated)](#get-schedule)
- [Add query to schedule (deprecated)](#add-query-to-schedule)
- [Edit query in schedule (deprecated)](#edit-query-in-schedule)
- [Remove query from schedule (deprecated)](#remove-query-from-schedule)
- [Team schedule](#team-schedule)

Scheduling queries in Fleet is the best practice for collecting data from hosts.

These API routes let you control your scheduled queries.

### Get schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility.
> Please use the [queries](#queries) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

`GET /api/v1/fleet/global/schedule`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/global/schedule`

##### Default response

`Status: 200`

```json
{
  "global_schedule": [
    {
      "created_at": "0001-01-01T00:00:00Z",
      "updated_at": "0001-01-01T00:00:00Z",
      "id": 4,
      "pack_id": 1,
      "name": "arp_cache",
      "query_id": 2,
      "query_name": "arp_cache",
      "query": "select * from arp_cache;",
      "interval": 120,
      "snapshot": true,
      "removed": null,
      "platform": "",
      "version": "",
      "shard": null,
      "denylist": null,
      "stats": {
        "system_time_p50": 1.32,
        "system_time_p95": 4.02,
        "user_time_p50": 3.55,
        "user_time_p95": 3.00,
        "total_executions": 3920
      }
    },
    {
      "created_at": "0001-01-01T00:00:00Z",
      "updated_at": "0001-01-01T00:00:00Z",
      "id": 5,
      "pack_id": 1,
      "name": "disk_encryption",
      "query_id": 7,
      "query_name": "disk_encryption",
      "query": "select * from disk_encryption;",
      "interval": 86400,
      "snapshot": true,
      "removed": null,
      "platform": "",
      "version": "",
      "shard": null,
      "denylist": null,
      "stats": {
        "system_time_p50": 1.32,
        "system_time_p95": 4.02,
        "user_time_p50": 3.55,
        "user_time_p95": 3.00,
        "total_executions": 3920
      }
    }
  ]
}
```

### Add query to schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility.
> Please use the [queries](#queries) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

`POST /api/v1/fleet/global/schedule`

#### Parameters

| Name     | Type    | In   | Description                                                                                                                      |
| -------- | ------- | ---- | -------------------------------------------------------------------------------------------------------------------------------- |
| query_id | integer | body | **Required.** The query's ID.                                                                                                    |
| interval | integer | body | **Required.** The amount of time, in seconds, the query waits before running.                                                    |
| snapshot | boolean | body | **Required.** Whether the queries logs show everything in its current state.                                                     |
| removed  | boolean | body | Whether "removed" actions should be logged. Default is `null`.                                                                   |
| platform | string  | body | The computer platform where this query will run (other platforms ignored). Empty value runs on all platforms. Default is `null`. |
| shard    | integer | body | Restrict this query to a percentage (1-100) of target hosts. Default is `null`.                                                  |
| version  | string  | body | The minimum required osqueryd version installed on a host. Default is `null`.                                                    |

#### Example

`POST /api/v1/fleet/global/schedule`

##### Request body

```json
{
  "interval": 86400,
  "query_id": 2,
  "snapshot": true
}
```

##### Default response

`Status: 200`

```json
{
  "scheduled": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 1,
    "pack_id": 5,
    "name": "arp_cache",
    "query_id": 2,
    "query_name": "arp_cache",
    "query": "select * from arp_cache;",
    "interval": 86400,
    "snapshot": true,
    "removed": null,
    "platform": "",
    "version": "",
    "shard": null,
    "denylist": null
  }
}
```

> Note that the `pack_id` is included in the response object because Fleet's Schedule feature uses [osquery query packs](https://osquery.readthedocs.io/en/stable/deployment/configuration/#query-packs) under the hood.

### Edit query in schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility.
> Please use the [queries](#queries) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

`PATCH /api/v1/fleet/global/schedule/{id}`

#### Parameters

| Name     | Type    | In   | Description                                                                                                   |
| -------- | ------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| id       | integer | path | **Required.** The scheduled query's ID.                                                                       |
| interval | integer | body | The amount of time, in seconds, the query waits before running.                                               |
| snapshot | boolean | body | Whether the queries logs show everything in its current state.                                                |
| removed  | boolean | body | Whether "removed" actions should be logged.                                                                   |
| platform | string  | body | The computer platform where this query will run (other platforms ignored). Empty value runs on all platforms. |
| shard    | integer | body | Restrict this query to a percentage (1-100) of target hosts.                                                  |
| version  | string  | body | The minimum required osqueryd version installed on a host.                                                    |

#### Example

`PATCH /api/v1/fleet/global/schedule/5`

##### Request body

```json
{
  "interval": 604800
}
```

##### Default response

`Status: 200`

```json
{
  "scheduled": {
    "created_at": "2021-07-16T14:40:15Z",
    "updated_at": "2021-07-16T14:40:15Z",
    "id": 5,
    "pack_id": 1,
    "name": "arp_cache",
    "query_id": 2,
    "query_name": "arp_cache",
    "query": "select * from arp_cache;",
    "interval": 604800,
    "snapshot": true,
    "removed": null,
    "platform": "",
    "shard": null,
    "denylist": null
  }
}
```

### Remove query from schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility.
> Please use the [queries](#queries) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

`DELETE /api/v1/fleet/global/schedule/{id}`

#### Parameters

None.

#### Example

`DELETE /api/v1/fleet/global/schedule/5`

##### Default response

`Status: 200`


---

### Team schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility.
> Please use the [queries](#queries) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

- [Get team schedule (deprecated)](#get-team-schedule)
- [Add query to team schedule (deprecated)](#add-query-to-team-schedule)
- [Edit query in team schedule (deprecated)](#edit-query-in-team-schedule)
- [Remove query from team schedule (deprecated)](#remove-query-from-team-schedule)

This allows you to easily configure scheduled queries that will impact a whole team of devices.

#### Get team schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility.
> Please use the [queries](#queries) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

`GET /api/v1/fleet/teams/{id}/schedule`

#### Parameters

| Name            | Type    | In    | Description                                                                                                                   |
| --------------- | ------- | ----- | ----------------------------------------------------------------------------------------------------------------------------- |
| id              | integer | path  | **Required**. The team's ID.                                                                                                  |
| page            | integer | query | Page number of the results to fetch.                                                                                          |
| per_page        | integer | query | Results per page.                                                                                                             |
| order_key       | string  | query | What to order results by. Can be any column in the `activites` table.                                                         |
| order_direction | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |

#### Example

`GET /api/v1/fleet/teams/2/schedule`

##### Default response

`Status: 200`

```json
{
  "scheduled": [
    {
      "created_at": "0001-01-01T00:00:00Z",
      "updated_at": "0001-01-01T00:00:00Z",
      "id": 4,
      "pack_id": 2,
      "name": "arp_cache",
      "query_id": 2,
      "query_name": "arp_cache",
      "query": "select * from arp_cache;",
      "interval": 120,
      "snapshot": true,
      "platform": "",
      "version": "",
      "removed": null,
      "shard": null,
      "denylist": null,
      "stats": {
        "system_time_p50": 1.32,
        "system_time_p95": 4.02,
        "user_time_p50": 3.55,
        "user_time_p95": 3.00,
        "total_executions": 3920
      }
    },
    {
      "created_at": "0001-01-01T00:00:00Z",
      "updated_at": "0001-01-01T00:00:00Z",
      "id": 5,
      "pack_id": 3,
      "name": "disk_encryption",
      "query_id": 7,
      "query_name": "disk_encryption",
      "query": "select * from disk_encryption;",
      "interval": 86400,
      "snapshot": true,
      "removed": null,
      "platform": "",
      "version": "",
      "shard": null,
      "denylist": null,
      "stats": {
        "system_time_p50": 1.32,
        "system_time_p95": 4.02,
        "user_time_p50": 3.55,
        "user_time_p95": 3.00,
        "total_executions": 3920
      }
    }
  ]
}
```

#### Add query to team schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility.
> Please use the [queries](#queries) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

`POST /api/v1/fleet/teams/{id}/schedule`

#### Parameters

| Name     | Type    | In   | Description                                                                                                                      |
| -------- | ------- | ---- | -------------------------------------------------------------------------------------------------------------------------------- |
| id       | integer | path | **Required.** The teams's ID.                                                                                                    |
| query_id | integer | body | **Required.** The query's ID.                                                                                                    |
| interval | integer | body | **Required.** The amount of time, in seconds, the query waits before running.                                                    |
| snapshot | boolean | body | **Required.** Whether the queries logs show everything in its current state.                                                     |
| removed  | boolean | body | Whether "removed" actions should be logged. Default is `null`.                                                                   |
| platform | string  | body | The computer platform where this query will run (other platforms ignored). Empty value runs on all platforms. Default is `null`. |
| shard    | integer | body | Restrict this query to a percentage (1-100) of target hosts. Default is `null`.                                                  |
| version  | string  | body | The minimum required osqueryd version installed on a host. Default is `null`.                                                    |

#### Example

`POST /api/v1/fleet/teams/2/schedule`

##### Request body

```json
{
  "interval": 86400,
  "query_id": 2,
  "snapshot": true
}
```

##### Default response

`Status: 200`

```json
{
  "scheduled": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 1,
    "pack_id": 5,
    "name": "arp_cache",
    "query_id": 2,
    "query_name": "arp_cache",
    "query": "select * from arp_cache;",
    "interval": 86400,
    "snapshot": true,
    "removed": null,
    "shard": null,
    "denylist": null
  }
}
```

#### Edit query in team schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility.
> Please use the [queries](#queries) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

`PATCH /api/v1/fleet/teams/{team_id}/schedule/{scheduled_query_id}`

#### Parameters

| Name               | Type    | In   | Description                                                                                                   |
| ------------------ | ------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| team_id            | integer | path | **Required.** The team's ID.                                                                                  |
| scheduled_query_id | integer | path | **Required.** The scheduled query's ID.                                                                       |
| interval           | integer | body | The amount of time, in seconds, the query waits before running.                                               |
| snapshot           | boolean | body | Whether the queries logs show everything in its current state.                                                |
| removed            | boolean | body | Whether "removed" actions should be logged.                                                                   |
| platform           | string  | body | The computer platform where this query will run (other platforms ignored). Empty value runs on all platforms. |
| shard              | integer | body | Restrict this query to a percentage (1-100) of target hosts.                                                  |
| version            | string  | body | The minimum required osqueryd version installed on a host.                                                    |

#### Example

`PATCH /api/v1/fleet/teams/2/schedule/5`

##### Request body

```json
{
  "interval": 604800
}
```

##### Default response

`Status: 200`

```json
{
  "scheduled": {
    "created_at": "2021-07-16T14:40:15Z",
    "updated_at": "2021-07-16T14:40:15Z",
    "id": 5,
    "pack_id": 1,
    "name": "arp_cache",
    "query_id": 2,
    "query_name": "arp_cache",
    "query": "select * from arp_cache;",
    "interval": 604800,
    "snapshot": true,
    "removed": null,
    "platform": "",
    "shard": null,
    "denylist": null
  }
}
```

#### Remove query from team schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility.
> Please use the [queries](#queries) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

`DELETE /api/v1/fleet/teams/{team_id}/schedule/{scheduled_query_id}`

#### Parameters

| Name               | Type    | In   | Description                             |
| ------------------ | ------- | ---- | --------------------------------------- |
| team_id            | integer | path | **Required.** The team's ID.            |
| scheduled_query_id | integer | path | **Required.** The scheduled query's ID. |

#### Example

`DELETE /api/v1/fleet/teams/2/schedule/5`

##### Default response

`Status: 200`

---

## Scripts

- [Run script asynchronously](#run-script-asynchronously)
- [Run script synchronously](#run-script-synchronously)


### Run script asynchronously

_Available in Fleet Premium_

Creates a script execution request and returns the execution identifier to retrieve results at a later time.

`POST /api/v1/fleet/scripts/run`

#### Parameters

| Name            | Type    | In   | Description                                      |
| ----            | ------- | ---- | --------------------------------------------     |
| host_id         | integer | body | **Required**. The host id to run the script on.  |
| script_contents | string  | body | **Required**. The contents of the script to run. |

#### Example

`POST /api/v1/fleet/scripts/run`

##### Default response

`Status: 202`

```json
{
  "host_id": 1227,
  "execution_id": "e797d6c6-3aae-11ee-be56-0242ac120002"
}
```

### Run script synchronously

_Available in Fleet Premium_

Creates a script execution request and waits for a result to return (up to a 1 minute timeout).

`POST /api/v1/fleet/scripts/run/sync`

#### Parameters

| Name            | Type    | In   | Description                                      |
| ----            | ------- | ---- | --------------------------------------------     |
| host_id         | integer | body | **Required**. The host id to run the script on.  |
| script_contents | string  | body | **Required**. The contents of the script to run. |

#### Example

`POST /api/v1/fleet/scripts/run/sync`

##### Default response

`Status: 200`

```json
{
  "host_id": 1227,
  "execution_id": "e797d6c6-3aae-11ee-be56-0242ac120002",
  "script_contents": "echo 'hello'",
  "output": "hello",
  "message": "",
  "runtime": 1,
  "host_timeout": false,
  "exit_code": 0
}
```

### Get script result

_Available in Fleet Premium_

Gets the result of a script that was executed.

#### Parameters

| Name         | Type   | In   | Description                                   |
| ----         | ------ | ---- | --------------------------------------------  |
| execution_id | string | path | **Required**. The execution id of the script. |

#### Example

`GET /api/v1/fleet/scripts/results/{execution_id}`

##### Default Response

`Status: 2000`

```json
{
  "script_contents": "echo 'hello'",
  "exit_code": 0,
  "output": "hello",
  "message": "",
  "hostname": "Test Host",
  "host_timeout": false,
  "host_id": 1,
  "execution_id": "e797d6c6-3aae-11ee-be56-0242ac120002",
  "runtime": 20
}
```

> Note: `exit_code` can be `null` if Fleet hasn't heard back from the host yet.

---

## Sessions

- [Get session info](#get-session-info)
- [Delete session](#delete-session)

### Get session info

Returns the session information for the session specified by ID.

`GET /api/v1/fleet/sessions/{id}`

#### Parameters

| Name | Type    | In   | Description                                  |
| ---- | ------- | ---- | -------------------------------------------- |
| id   | integer | path | **Required**. The ID of the desired session. |

#### Example

`GET /api/v1/fleet/sessions/1`

##### Default response

`Status: 200`

```json
{
  "session_id": 1,
  "user_id": 1,
  "created_at": "2021-03-02T18:41:34Z"
}
```

### Delete session

Deletes the session specified by ID. When the user associated with the session next attempts to access Fleet, they will be asked to log in.

`DELETE /api/v1/fleet/sessions/{id}`

#### Parameters

| Name | Type    | In   | Description                                  |
| ---- | ------- | ---- | -------------------------------------------- |
| id   | integer | path | **Required**. The id of the desired session. |

#### Example

`DELETE /api/v1/fleet/sessions/1`

##### Default response

`Status: 200`


---

## Software

- [List all software](#list-all-software)
- [Count software](#count-software)

### List all software

`GET /api/v1/fleet/software`

#### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                |
| ----------------------- | ------- | ----- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                       |
| per_page                | integer | query | Results per page.                                                                                                                                                          |
| order_key               | string  | query | What to order results by. Allowed fields are `name`, `hosts_count`, `cve_published`, `cvss_score`, `epss_probability` and `cisa_known_exploit`. Default is `hosts_count` (descending).      |
| order_direction         | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`.                                              |
| query                   | string  | query | Search query keywords. Searchable fields include `name`, `version`, and `cve`.                                                                                             |
| team_id                 | integer | query | _Available in Fleet Premium_ Filters the software to only include the software installed on the hosts that are assigned to the specified team.                             |
| vulnerable              | bool    | query | If true or 1, only list software that has detected vulnerabilities. Default is `false`.                                                                                    |

#### Example

`GET /api/v1/fleet/software`

##### Default response

`Status: 200`

```json
{
    "counts_updated_at": "2022-01-01 12:32:00",
    "software": [
      {
        "id": 1,
        "name": "glibc",
        "version": "2.12",
        "source": "rpm_packages",
        "release": "1.212.el6",
        "vendor": "CentOS",
        "arch": "x86_64",
        "generated_cpe": "cpe:2.3:a:gnu:glibc:2.12:*:*:*:*:*:*:*",
        "vulnerabilities": [
          {
            "cve": "CVE-2009-5155",
            "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2009-5155",
            "cvss_score": 7.5,
            "epss_probability": 0.01537,
            "cisa_known_exploit": false,
            "cve_published": "2022-01-01 12:32:00",
            "cve_description": "In the GNU C Library (aka glibc or libc6) before 2.28, parse_reg_exp in posix/regcomp.c misparses alternatives, which allows attackers to cause a denial of service (assertion failure and application exit) or trigger an incorrect result by attempting a regular-expression match."
          }
        ],
        "hosts_count": 1
      }
    ]
}
```

### Count software

`GET /api/v1/fleet/software/count`

#### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                                                                                                                                                                                 |
| ----------------------- | ------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| query                   | string  | query | Search query keywords. Searchable fields include `name`, `version`, and `cve`.                                                                                                                                                                                                                                                               |
| team_id                 | integer | query | _Available in Fleet Premium_ Filters the software to only include the software installed on the hosts that are assigned to the specified team.                                                                                                                                                                                              |
| vulnerable              | bool    | query | If true or 1, only list software that has detected vulnerabilities.                                                                                                                                                                                                                                                                         |

#### Example

`GET /api/v1/fleet/software/count`

##### Default response

`Status: 200`

```json
{
  "count": 43
}
```
---

## Targets

In Fleet, targets are used to run queries against specific hosts or groups of hosts. Labels are used to create groups in Fleet.

### Search targets

The search targets endpoint returns two lists. The first list includes the possible target hosts in Fleet given the search query provided and the hosts already selected as targets. The second list includes the possible target labels in Fleet given the search query provided and the labels already selected as targets.

The returned lists are filtered based on the hosts the requesting user has access to.

`POST /api/v1/fleet/targets`

#### Parameters

| Name     | Type    | In   | Description                                                                                                                                                                |
| -------- | ------- | ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| query    | string  | body | The search query. Searchable items include a host's hostname or IPv4 address and labels.                                                                                   |
| query_id | integer | body | The saved query (if any) that will be run. The `observer_can_run` property on the query and the user's roles effect which targets are included.                            |
| selected | object  | body | The targets already selected. The object includes a `hosts` property which contains a list of host IDs, a `labels` with label IDs and/or a `teams` property with team IDs. |

#### Example

`POST /api/v1/fleet/targets`

##### Request body

```json
{
  "query": "172",
  "selected": {
    "hosts": [],
    "labels": [7]
  },
  "include_observer": true
}
```

##### Default response

```json
{
  "targets": {
    "hosts": [
      {
        "created_at": "2021-02-03T16:11:43Z",
        "updated_at": "2021-02-03T21:58:19Z",
        "id": 3,
        "detail_updated_at": "2021-02-03T21:58:10Z",
        "label_updated_at": "2021-02-03T21:58:10Z",
        "policy_updated_at": "2023-06-26T18:33:15Z",
        "last_enrolled_at": "2021-02-03T16:11:43Z",
        "software_updated_at": "2020-11-05T05:09:44Z",
        "seen_time": "2021-02-03T21:58:20Z",
        "hostname": "7a2f41482833",
        "uuid": "a2064cef-0000-0000-afb9-283e3c1d487e",
        "platform": "rhel",
        "osquery_version": "4.5.1",
        "os_version": "CentOS 6.10.0",
        "build": "",
        "platform_like": "rhel",
        "code_name": "",
        "uptime": 32688000000000,
        "memory": 2086899712,
        "cpu_type": "x86_64",
        "cpu_subtype": "142",
        "cpu_brand": "Intel(R) Core(TM) i5-8279U CPU @ 2.40GHz",
        "cpu_physical_cores": 4,
        "cpu_logical_cores": 4,
        "hardware_vendor": "",
        "hardware_model": "",
        "hardware_version": "",
        "hardware_serial": "",
        "computer_name": "7a2f41482833",
        "display_name": "7a2f41482833",
        "primary_ip": "172.20.0.3",
        "primary_mac": "02:42:ac:14:00:03",
        "distributed_interval": 10,
        "config_tls_refresh": 10,
        "logger_tls_period": 10,
        "additional": {},
        "status": "offline",
        "display_text": "7a2f41482833"
      },
      {
        "created_at": "2021-02-03T16:11:43Z",
        "updated_at": "2021-02-03T21:58:19Z",
        "id": 4,
        "detail_updated_at": "2021-02-03T21:58:10Z",
        "label_updated_at": "2021-02-03T21:58:10Z",
        "policy_updated_at": "2023-06-26T18:33:15Z",
        "last_enrolled_at": "2021-02-03T16:11:43Z",
        "software_updated_at": "2020-11-05T05:09:44Z",
        "seen_time": "2021-02-03T21:58:20Z",
        "hostname": "78c96e72746c",
        "uuid": "a2064cef-0000-0000-afb9-283e3c1d487e",
        "platform": "ubuntu",
        "osquery_version": "4.5.1",
        "os_version": "Ubuntu 16.4.0",
        "build": "",
        "platform_like": "debian",
        "code_name": "",
        "uptime": 32688000000000,
        "memory": 2086899712,
        "cpu_type": "x86_64",
        "cpu_subtype": "142",
        "cpu_brand": "Intel(R) Core(TM) i5-8279U CPU @ 2.40GHz",
        "cpu_physical_cores": 4,
        "cpu_logical_cores": 4,
        "hardware_vendor": "",
        "hardware_model": "",
        "hardware_version": "",
        "hardware_serial": "",
        "computer_name": "78c96e72746c",
        "display_name": "78c96e72746c",
        "primary_ip": "172.20.0.7",
        "primary_mac": "02:42:ac:14:00:07",
        "distributed_interval": 10,
        "config_tls_refresh": 10,
        "logger_tls_period": 10,
        "additional": {},
        "status": "offline",
        "display_text": "78c96e72746c"
      }
    ],
    "labels": [
      {
        "created_at": "2021-02-02T23:55:25Z",
        "updated_at": "2021-02-02T23:55:25Z",
        "id": 6,
        "name": "All Hosts",
        "description": "All hosts which have enrolled in Fleet",
        "query": "SELECT 1;",
        "label_type": "builtin",
        "label_membership_type": "dynamic",
        "host_count": 5,
        "display_text": "All Hosts",
        "count": 5
      }
    ],
    "teams": [
      {
        "id": 1,
        "created_at": "2021-05-27T20:02:20Z",
        "name": "Client Platform Engineering",
        "description": "",
        "agent_options": null,
        "user_count": 4,
        "host_count": 2,
        "display_text": "Client Platform Engineering",
        "count": 2
      }
    ]
  },
  "targets_count": 1,
  "targets_online": 1,
  "targets_offline": 0,
  "targets_missing_in_action": 0
}
```

---

## Teams

- [List teams](#list-teams)
- [Get team](#get-team)
- [Create team](#create-team)
- [Modify team](#modify-team)
- [Modify team's agent options](#modify-teams-agent-options)
- [Delete team](#delete-team)

### List teams

_Available in Fleet Premium_

`GET /api/v1/fleet/teams`

#### Parameters

| Name            | Type    | In    | Description                                                                                                                   |
| --------------- | ------- | ----- | ----------------------------------------------------------------------------------------------------------------------------- |
| page            | integer | query | Page number of the results to fetch.                                                                                          |
| per_page        | integer | query | Results per page.                                                                                                             |
| order_key       | string  | query | What to order results by. Can be any column in the `teams` table.                                                             |
| order_direction | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |
| query           | string  | query | Search query keywords. Searchable fields include `name`.                                                                      |

#### Example

`GET /api/v1/fleet/teams`

##### Default response

`Status: 200`

```json
{
  "teams": [
    {
      "id": 1,
      "created_at": "2021-07-28T15:58:21Z",
      "name": "workstations",
      "description": "",
      "agent_options": {
        "config": {
          "options": {
            "pack_delimiter": "/",
            "logger_tls_period": 10,
            "distributed_plugin": "tls",
            "disable_distributed": false,
            "logger_tls_endpoint": "/api/v1/osquery/log",
            "distributed_interval": 10,
            "distributed_tls_max_attempts": 3
          },
          "decorators": {
            "load": [
              "SELECT uuid AS host_uuid FROM system_info;",
              "SELECT hostname AS hostname FROM system_info;"
            ]
          }
        },
        "overrides": {},
        "command_line_flags": {}
      },
      "user_count": 0,
      "host_count": 0,
      "secrets": [
        {
          "secret": "",
          "created_at": "2021-07-28T15:58:21Z",
          "team_id": 10
        }
      ]
    },
    {
      "id": 2,
      "created_at": "2021-08-05T21:41:42Z",
      "name": "servers",
      "description": "",
      "agent_options": {
        "spec": {
          "config": {
            "options": {
              "pack_delimiter": "/",
              "logger_tls_period": 10,
              "distributed_plugin": "tls",
              "disable_distributed": false,
              "logger_tls_endpoint": "/api/v1/osquery/log",
              "distributed_interval": 10,
              "distributed_tls_max_attempts": 3
            },
            "decorators": {
              "load": [
                "SELECT uuid AS host_uuid FROM system_info;",
                "SELECT hostname AS hostname FROM system_info;"
              ]
            }
          },
          "overrides": {},
          "command_line_flags": {}
        },
        "user_count": 0,
        "host_count": 0,
        "secrets": [
          {
            "secret": "+ncixtnZB+IE0OrbrkCLeul3U8LMVITd",
            "created_at": "2021-08-05T21:41:42Z",
            "team_id": 15
          }
        ]
      }
    }
  ]
}
```

### Get team

_Available in Fleet Premium_

`GET /api/v1/fleet/teams/{id}`

#### Parameters

| Name | Type    | In   | Description                          |
| ---- | ------  | ---- | ------------------------------------ |
| id   | integer | path | **Required.** The desired team's ID. |

#### Example

`GET /api/v1/fleet/teams/1`

##### Default response

`Status: 200`

```json
{
  "team": {
    "name": "Workstations",
    "id": 1,
    "user_count": 4,
    "host_count": 0,
    "agent_options": {
      "config": {
        "options": {
          "pack_delimiter": "/",
          "logger_tls_period": 10,
          "distributed_plugin": "tls",
          "disable_distributed": false,
          "logger_tls_endpoint": "/api/v1/osquery/log",
          "distributed_interval": 10,
          "distributed_tls_max_attempts": 3
        },
        "decorators": {
          "load": [
            "SELECT uuid AS host_uuid FROM system_info;",
            "SELECT hostname AS hostname FROM system_info;"
          ]
        }
      },
      "overrides": {},
      "command_line_flags": {}
    },
    "webhook_settings": {
      "failing_policies_webhook": {
        "enable_failing_policies_webhook": false,
        "destination_url": "",
        "policy_ids": null,
        "host_batch_size": 0
      }
    },
    "mdm": {
      "macos_updates": {
        "minimum_version": "12.3.1",
        "deadline": "2022-01-01"
      },
      "macos_settings": {
        "custom_settings": ["path/to/profile.mobileconfig"],
        "enable_disk_encryption": false
      },
      "macos_setup": {
        "bootstrap_package": "",
        "enable_end_user_authentication": false,
        "macos_setup_assistant": "path/to/config.json"
      }
    }
  }
}
```

### Create team

_Available in Fleet Premium_

`POST /api/v1/fleet/teams`

#### Parameters

| Name | Type   | In   | Description                    |
| ---- | ------ | ---- | ------------------------------ |
| name | string | body | **Required.** The team's name. |

#### Example

`POST /api/v1/fleet/teams`

##### Request body

```json
{
  "name": "workstations"
}
```

##### Default response

`Status: 200`

```json
{
  "teams": [
    {
      "name": "workstations",
      "id": 1,
      "user_count": 0,
      "host_count": 0,
      "agent_options": {
        "config": {
          "options": {
            "pack_delimiter": "/",
            "logger_tls_period": 10,
            "distributed_plugin": "tls",
            "disable_distributed": false,
            "logger_tls_endpoint": "/api/v1/osquery/log",
            "distributed_interval": 10,
            "distributed_tls_max_attempts": 3
          },
          "decorators": {
            "load": [
              "SELECT uuid AS host_uuid FROM system_info;",
              "SELECT hostname AS hostname FROM system_info;"
            ]
          }
        },
        "overrides": {},
        "command_line_flags": {}
      },
      "webhook_settings": {
        "failing_policies_webhook": {
          "enable_failing_policies_webhook": false,
          "destination_url": "",
          "policy_ids": null,
          "host_batch_size": 0
        }
      }
    }
  ]
}
```

### Modify team

_Available in Fleet Premium_

`PATCH /api/v1/fleet/teams/{id}`

#### Parameters

| Name                                                    | Type    | In   | Description                                                                                                                                                                                               |
| ------------------------------------------------------- | ------- | ---- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| id                                                      | integer | path | **Required.** The desired team's ID.                                                                                                                                                                      |
| name                                                    | string  | body | The team's name.                                                                                                                                                                                          |
| host_ids                                                | list    | body | A list of hosts that belong to the team.                                                                                                                                                                  |
| user_ids                                                | list    | body | A list of users that are members of the team.                                                                                                                                                             |
| webhook_settings                                        | object  | body | Webhook settings contains for the team.                                                                                                                                                                   |
| &nbsp;&nbsp;failing_policies_webhook                    | object  | body | Failing policies webhook settings.                                                                                                                                                                        |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_failing_policies_webhook | boolean | body | Whether or not the failing policies webhook is enabled.                                                                                                                                                   |
| &nbsp;&nbsp;&nbsp;&nbsp;destination_url                 | string  | body | The URL to deliver the webhook requests to.                                                                                                                                                               |
| &nbsp;&nbsp;&nbsp;&nbsp;policy_ids                      | array   | body | List of policy IDs to enable failing policies webhook.                                                                                                                                                    |
| &nbsp;&nbsp;&nbsp;&nbsp;host_batch_size                 | integer | body | Maximum number of hosts to batch on failing policy webhook requests. The default, 0, means no batching (all hosts failing a policy are sent on one request).                                              |
| integrations                                            | object  | body | Integrations settings for the team. Note that integrations referenced here must already exist globally, created by a call to [Modify configuration](#modify-configuration).                               |
| &nbsp;&nbsp;jira                                        | array   | body | Jira integrations configuration.                                                                                                                                                                          |
| &nbsp;&nbsp;&nbsp;&nbsp;url                             | string  | body | The URL of the Jira server to use.                                                                                                                                                                        |
| &nbsp;&nbsp;&nbsp;&nbsp;project_key                     | string  | body | The project key of the Jira integration to use. Jira tickets will be created in this project.                                                                                                             |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_failing_policies         | boolean | body | Whether or not that Jira integration is enabled for failing policies. Only one failing policy automation can be enabled at a given time (enable_failing_policies_webhook and enable_failing_policies).    |
| &nbsp;&nbsp;zendesk                                     | array   | body | Zendesk integrations configuration.                                                                                                                                                                       |
| &nbsp;&nbsp;&nbsp;&nbsp;url                             | string  | body | The URL of the Zendesk server to use.                                                                                                                                                                     |
| &nbsp;&nbsp;&nbsp;&nbsp;group_id                        | integer | body | The Zendesk group id to use. Zendesk tickets will be created in this group.                                                                                                                               |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_failing_policies         | boolean | body | Whether or not that Zendesk integration is enabled for failing policies. Only one failing policy automation can be enabled at a given time (enable_failing_policies_webhook and enable_failing_policies). |
| mdm                                                     | object  | body | MDM settings for the team.                                                                                                                                                                                |
| &nbsp;&nbsp;macos_updates                               | object  | body | MacOS updates settings.                                                                                                                                                                                   |
| &nbsp;&nbsp;&nbsp;&nbsp;minimum_version                 | string  | body | Hosts that belong to this team and are enrolled into Fleet's MDM will be nudged until their macOS is at or above this version.                                                                            |
| &nbsp;&nbsp;&nbsp;&nbsp;deadline                        | string  | body | Hosts that belong to this team and are enrolled into Fleet's MDM won't be able to dismiss the Nudge window once this deadline is past.                                                                    |
| &nbsp;&nbsp;macos_settings                              | object  | body | MacOS-specific settings.                                                                                                                                                                                  |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_disk_encryption          | boolean | body | Hosts that belong to this team and are enrolled into Fleet's MDM will have disk encryption enabled if set to true.                                                                                        |
| &nbsp;&nbsp;macos_setup                                 | object  | body | Setup for automatic MDM enrollment of macOS devices.                                                                                                                                                                                  |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_end_user_authentication          | boolean | body | If set to true, end user authentication will be required during automatic MDM enrollment of new macOS devices. Settings for your IdP provider must also be [configured](https://fleetdm.com/docs/using-fleet/mdm-macos-setup#end-user-authentication).                                                                                      |


#### Example (add users to a team)

`PATCH /api/v1/fleet/teams/1/users`

##### Request body

```json
{
  "user_ids": [1, 17, 22, 32]
}
```

##### Default response

`Status: 200`

```json
{
  "team": {
    "name": "Workstations",
    "id": 1,
    "user_count": 4,
    "host_count": 0,
    "agent_options": {
      "config": {
        "options": {
          "pack_delimiter": "/",
          "logger_tls_period": 10,
          "distributed_plugin": "tls",
          "disable_distributed": false,
          "logger_tls_endpoint": "/api/v1/osquery/log",
          "distributed_interval": 10,
          "distributed_tls_max_attempts": 3
        },
        "decorators": {
          "load": [
            "SELECT uuid AS host_uuid FROM system_info;",
            "SELECT hostname AS hostname FROM system_info;"
          ]
        }
      },
      "overrides": {},
      "command_line_flags": {}
    },
    "webhook_settings": {
      "failing_policies_webhook": {
        "enable_failing_policies_webhook": false,
        "destination_url": "",
        "policy_ids": null,
        "host_batch_size": 0
      }
    },
    "mdm": {
      "macos_updates": {
        "minimum_version": "12.3.1",
        "deadline": "2022-01-01"
      },
      "macos_settings": {
        "custom_settings": ["path/to/profile.mobileconfig"],
        "enable_disk_encryption": false
      },
      "macos_setup": {
        "bootstrap_package": "",
        "enable_end_user_authentication": false,
        "macos_setup_assistant": "path/to/config.json"
      }
    }
  }
}
```

#### Example (transfer hosts to a team)

`PATCH /api/v1/fleet/teams/1`

##### Request body

```json
{
  "host_ids": [3, 6, 7, 8, 9, 20, 32, 44]
}
```

##### Default response

`Status: 200`

```json
{
  "team": {
    "name": "Workstations",
    "id": 1,
    "user_count": 4,
    "host_count": 8,
    "agent_options": {
      "config": {
        "options": {
          "pack_delimiter": "/",
          "logger_tls_period": 10,
          "distributed_plugin": "tls",
          "disable_distributed": false,
          "logger_tls_endpoint": "/api/v1/osquery/log",
          "distributed_interval": 10,
          "distributed_tls_max_attempts": 3
        },
        "decorators": {
          "load": [
            "SELECT uuid AS host_uuid FROM system_info;",
            "SELECT hostname AS hostname FROM system_info;"
          ]
        }
      },
      "overrides": {},
      "command_line_flags": {}
    },
    "webhook_settings": {
      "failing_policies_webhook": {
        "enable_failing_policies_webhook": false,
        "destination_url": "",
        "policy_ids": null,
        "host_batch_size": 0
      }
    }
  }
}
```

### Modify team's agent options

_Available in Fleet Premium_

`POST /api/v1/fleet/teams/{id}/agent_options`

#### Parameters

| Name                             | Type    | In    | Description                                                                                                                                                  |
| ---                              | ---     | ---   | ---                                                                                                                                                          |
| id                               | integer | path  | **Required.** The desired team's ID.                                                                                                                         |
| force                            | bool    | query | Force apply the options even if there are validation errors.                                                                                                 |
| dry_run                          | bool    | query | Validate the options and return any validation errors, but do not apply the changes.                                                                         |
| _JSON data_                      | object  | body  | The JSON to use as agent options for this team. See [Agent options](https://fleetdm.com/docs/using-fleet/configuration-files#agent-options) for details.                              |

#### Example

`POST /api/v1/fleet/teams/1/agent_options`

##### Request body

```json
{
  "config": {
    "options": {
      "pack_delimiter": "/",
      "logger_tls_period": 20,
      "distributed_plugin": "tls",
      "disable_distributed": false,
      "logger_tls_endpoint": "/api/v1/osquery/log",
      "distributed_interval": 60,
      "distributed_tls_max_attempts": 3
    },
    "decorators": {
      "load": [
        "SELECT uuid AS host_uuid FROM system_info;",
        "SELECT hostname AS hostname FROM system_info;"
      ]
    }
  },
  "overrides": {},
  "command_line_flags": {}
}
```

##### Default response

`Status: 200`

```json
{
  "team": {
    "name": "Workstations",
    "id": 1,
    "user_count": 4,
    "host_count": 8,
    "agent_options": {
      "config": {
        "options": {
          "pack_delimiter": "/",
          "logger_tls_period": 20,
          "distributed_plugin": "tls",
          "disable_distributed": false,
          "logger_tls_endpoint": "/api/v1/osquery/log",
          "distributed_interval": 60,
          "distributed_tls_max_attempts": 3
        },
        "decorators": {
          "load": [
            "SELECT uuid AS host_uuid FROM system_info;",
            "SELECT hostname AS hostname FROM system_info;"
          ]
        }
      },
      "overrides": {},
      "command_line_flags": {}
    },
    "webhook_settings": {
      "failing_policies_webhook": {
        "enable_failing_policies_webhook": false,
        "destination_url": "",
        "policy_ids": null,
        "host_batch_size": 0
      }
    }
  }
}
```

### Delete team

_Available in Fleet Premium_

`DELETE /api/v1/fleet/teams/{id}`

#### Parameters

| Name | Type    | In   | Description                          |
| ---- | ------  | ---- | ------------------------------------ |
| id   | integer | path | **Required.** The desired team's ID. |

#### Example

`DELETE /api/v1/fleet/teams/1`

#### Default response

`Status: 200`

---

## Translator

- [Translate IDs](#translate-ids)

### Translate IDs

Transforms a host name into a host id. For example, the Fleet UI use this endpoint when sending live queries to a set of hosts.

`POST /api/v1/fleet/translate`

#### Parameters

| Name | Type  | In   | Description                              |
| ---- | ----- | ---- | ---------------------------------------- |
| list | array | body | **Required** list of items to translate. |

#### Example

`POST /api/v1/fleet/translate`

##### Request body

```json
{
  "list": [
    {
      "type": "user",
      "payload": {
        "identifier": "some@email.com"
      }
    },
    {
      "type": "label",
      "payload": {
        "identifier": "labelA"
      }
    },
    {
      "type": "team",
      "payload": {
        "identifier": "team1"
      }
    },
    {
      "type": "host",
      "payload": {
        "identifier": "host-ABC"
      }
    }
  ]
}
```

##### Default response

`Status: 200`

```json
{
  "list": [
    {
      "type": "user",
      "payload": {
        "identifier": "some@email.com",
        "id": 32
      }
    },
    {
      "type": "label",
      "payload": {
        "identifier": "labelA",
        "id": 1
      }
    },
    {
      "type": "team",
      "payload": {
        "identifier": "team1",
        "id": 22
      }
    },
    {
      "type": "host",
      "payload": {
        "identifier": "host-ABC",
        "id": 45
      }
    }
  ]
}
```
---

## Users

- [List all users](#list-all-users)
- [Create a user account with an invitation](#create-a-user-account-with-an-invitation)
- [Create a user account without an invitation](#create-a-user-account-without-an-invitation)
- [Get user information](#get-user-information)
- [Modify user](#modify-user)
- [Delete user](#delete-user)
- [Require password reset](#require-password-reset)
- [List a user's sessions](#list-a-users-sessions)
- [Delete a user's sessions](#delete-a-users-sessions)

The Fleet server exposes a handful of API endpoints that handles common user management operations. All the following endpoints require prior authentication meaning you must first log in successfully before calling any of the endpoints documented below.

### List all users

Returns a list of all enabled users

`GET /api/v1/fleet/users`

#### Parameters

| Name            | Type    | In    | Description                                                                                                                   |
| --------------- | ------- | ----- | ----------------------------------------------------------------------------------------------------------------------------- |
| query           | string  | query | Search query keywords. Searchable fields include `name` and `email`.                                                          |
| order_key       | string  | query | What to order results by. Can be any column in the users table.                                                               |
| order_direction | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |
| page            | integer | query | Page number of the results to fetch.                                                                                          |
| query           | string  | query | Search query keywords. Searchable fields include `name` and `email`.                                                          |
| per_page        | integer | query | Results per page.                                                                                                             |
| team_id         | string  | query | _Available in Fleet Premium_ Filters the users to only include users in the specified team.                                   |

#### Example

`GET /api/v1/fleet/users`

##### Request query parameters

None.

##### Default response

`Status: 200`

```json
{
  "users": [
    {
      "created_at": "2020-12-10T03:52:53Z",
      "updated_at": "2020-12-10T03:52:53Z",
      "id": 1,
      "name": "Jane Doe",
      "email": "janedoe@example.com",
      "force_password_reset": false,
      "gravatar_url": "",
      "sso_enabled": false,
      "global_role": null,
      "api_only": false,
      "teams": [
        {
          "id": 1,
          "created_at": "0001-01-01T00:00:00Z",
          "name": "workstations",
          "description": "",
          "role": "admin"
        }
      ]
    }
  ]
}
```

##### Failed authentication

`Status: 401 Authentication Failed`

```json
{
  "message": "Authentication Failed",
  "errors": [
    {
      "name": "base",
      "reason": "Authentication failed"
    }
  ]
}
```

### Create a user account with an invitation

Creates a user account after an invited user provides registration information and submits the form.

`POST /api/v1/fleet/users`

#### Parameters

| Name                  | Type   | In   | Description                                                                                                                                                                                                                                                                                                                                              |
| --------------------- | ------ | ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| email                 | string | body | **Required**. The email address of the user.                                                                                                                                                                                                                                                                                                             |
| invite_token          | string | body | **Required**. Token provided to the user in the invitation email.                                                                                                                                                                                                                                                                                        |
| name                  | string | body | **Required**. The name of the user.                                                                                                                                                                                                                                                                                                                      |
| password              | string | body | The password chosen by the user (if not SSO user).                                                                                                                                                                                                                                                                                                       |
| password_confirmation | string | body | Confirmation of the password chosen by the user.                                                                                                                                                                                                                                                                                                         |
| global_role           | string | body | The role assigned to the user. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). In Fleet 4.30.0 and 4.31.0, the `observer_plus` and `gitops` roles were introduced respectively. If `global_role` is specified, `teams` cannot be specified. For more information, see [manage access](https://fleetdm.com/docs/using-fleet/manage-access).                                                                                                                                                                        |
| teams                 | array  | body | _Available in Fleet Premium_ The teams and respective roles assigned to the user. Should contain an array of objects in which each object includes the team's `id` and the user's `role` on each team. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). In Fleet 4.30.0 and 4.31.0, the `observer_plus` and `gitops` roles were introduced respectively. If `teams` is specified, `global_role` cannot be specified. For more information, see [manage access](https://fleetdm.com/docs/using-fleet/manage-access). |

#### Example

`POST /api/v1/fleet/users`

##### Request query parameters

```json
{
  "email": "janedoe@example.com",
  "invite_token": "SjdReDNuZW5jd3dCbTJtQTQ5WjJTc2txWWlEcGpiM3c=",
  "name": "janedoe",
  "password": "test-123",
  "password_confirmation": "test-123",
  "teams": [
    {
      "id": 2,
      "role": "observer"
    },
    {
      "id": 4,
      "role": "observer"
    }
  ]
}
```

##### Default response

`Status: 200`

```json
{
  "user": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 2,
    "name": "janedoe",
    "email": "janedoe@example.com",
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false,
    "global_role": "admin",
    "teams": []
  }
}
```

##### Failed authentication

`Status: 401 Authentication Failed`

```json
{
  "message": "Authentication Failed",
  "errors": [
    {
      "name": "base",
      "reason": "Authentication failed"
    }
  ]
}
```

##### Expired or used invite code

`Status: 404 Resource Not Found`

```json
{
  "message": "Resource Not Found",
  "errors": [
    {
      "name": "base",
      "reason": "Invite with token SjdReDNuZW5jd3dCbTJtQTQ5WjJTc2txWWlEcGpiM3c= was not found in the datastore"
    }
  ]
}
```

##### Validation failed

`Status: 422 Validation Failed`

The same error will be returned whenever one of the required parameters fails the validation.

```json
{
  "message": "Validation Failed",
  "errors": [
    {
      "name": "name",
      "reason": "cannot be empty"
    }
  ]
}
```

### Create a user account without an invitation

Creates a user account without requiring an invitation, the user is enabled immediately.
By default, the user will be forced to reset its password upon first login.

`POST /api/v1/fleet/users/admin`

#### Parameters

| Name        | Type    | In   | Description                                                                                                                                                                                                                                                                                                                                              |
| ----------- | ------- | ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| email       | string  | body | **Required**. The user's email address.                                                                                                                                                                                                                                                                                                                  |
| name        | string  | body | **Required**. The user's full name or nickname.                                                                                                                                                                                                                                                                                                          |
| password    | string  | body | The user's password (required for non-SSO users).                                                                                                                                                                                                                                                                                                        |
| sso_enabled | boolean | body | Whether or not SSO is enabled for the user.                                                                                                                                                                                                                                                                                                              |
| api_only    | boolean | body | User is an "API-only" user (cannot use web UI) if true.                                                                                                                                                                                                                                                                                                  |
| global_role | string | body | The role assigned to the user. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). In Fleet 4.30.0 and 4.31.0, the `observer_plus` and `gitops` roles were introduced respectively. If `global_role` is specified, `teams` cannot be specified. For more information, see [manage access](https://fleetdm.com/docs/using-fleet/manage-access).                                                                                                                                                                        |
| admin_forced_password_reset    | boolean | body | Sets whether the user will be forced to reset its password upon first login (default=true) |
| teams                          | array   | body | _Available in Fleet Premium_ The teams and respective roles assigned to the user. Should contain an array of objects in which each object includes the team's `id` and the user's `role` on each team. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). In Fleet 4.30.0 and 4.31.0, the `observer_plus` and `gitops` roles were introduced respectively. If `teams` is specified, `global_role` cannot be specified. For more information, see [manage access](https://fleetdm.com/docs/using-fleet/manage-access). |

#### Example

`POST /api/v1/fleet/users/admin`

##### Request body

```json
{
  "name": "Jane Doe",
  "email": "janedoe@example.com",
  "password": "test-123",
  "teams": [
    {
      "id": 2,
      "role": "observer"
    },
    {
      "id": 3,
      "role": "maintainer"
    }
  ]
}
```

##### Default response

`Status: 200`

```json
{
  "user": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 5,
    "name": "Jane Doe",
    "email": "janedoe@example.com",
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false,
    "api_only": false,
    "global_role": null,
    "teams": [
      {
        "id": 2,
        "role": "observer"
      },
      {
        "id": 3,
        "role": "maintainer"
      }
    ]
  }
}
```

##### User doesn't exist

`Status: 404 Resource Not Found`

```json
{
  "message": "Resource Not Found",
  "errors": [
    {
      "name": "base",
      "reason": "User with id=1 was not found in the datastore"
    }
  ]
}
```

### Get user information

Returns all information about a specific user.

`GET /api/v1/fleet/users/{id}`

#### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The user's id. |

#### Example

`GET /api/v1/fleet/users/2`

##### Request query parameters

```json
{
  "id": 1
}
```

##### Default response

`Status: 200`

```json
{
  "user": {
    "created_at": "2020-12-10T05:20:25Z",
    "updated_at": "2020-12-10T05:24:27Z",
    "id": 2,
    "name": "Jane Doe",
    "email": "janedoe@example.com",
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false,
    "global_role": "admin",
    "api_only": false,
    "teams": []
  }
}
```

##### User doesn't exist

`Status: 404 Resource Not Found`

```json
{
  "message": "Resource Not Found",
  "errors": [
    {
      "name": "base",
      "reason": "User with id=5 was not found in the datastore"
    }
  ]
}
```

### Modify user

`PATCH /api/v1/fleet/users/{id}`

#### Parameters

| Name        | Type    | In   | Description                                                                                                                                                                                                                                                                                                                                              |
| ----------- | ------- | ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| id          | integer | path | **Required**. The user's id.                                                                                                                                                                                                                                                                                                                             |
| name        | string  | body | The user's name.                                                                                                                                                                                                                                                                                                                                         |
| position    | string  | body | The user's position.                                                                                                                                                                                                                                                                                                                                     |
| email       | string  | body | The user's email.                                                                                                                                                                                                                                                                                                                                        |
| sso_enabled | boolean | body | Whether or not SSO is enabled for the user.                                                                                                                                                                                                                                                                                                              |
| api_only    | boolean | body | User is an "API-only" user (cannot use web UI) if true.                                                                                                                                                                                                                                                                                                  |
| password    | string  | body | The user's current password, required to change the user's own email or password (not required for an admin to modify another user).                                                                                                                                                                                                                     |
| new_password| string  | body | The user's new password. |
| global_role | string  | body | The role assigned to the user. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). If `global_role` is specified, `teams` cannot be specified.                                                                                                                                                                         |
| teams       | array   | body | _Available in Fleet Premium_ The teams and respective roles assigned to the user. Should contain an array of objects in which each object includes the team's `id` and the user's `role` on each team. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). If `teams` is specified, `global_role` cannot be specified. |

#### Example

`PATCH /api/v1/fleet/users/2`

##### Request body

```json
{
  "name": "Jane Doe",
  "global_role": "admin"
}
```

##### Default response

`Status: 200`

```json
{
  "user": {
    "created_at": "2021-02-03T16:11:06Z",
    "updated_at": "2021-02-03T16:11:06Z",
    "id": 2,
    "name": "Jane Doe",
    "email": "janedoe@example.com",
    "global_role": "admin",
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false,
    "api_only": false,
    "teams": []
  }
}
```

#### Example (modify a user's teams)

`PATCH /api/v1/fleet/users/2`

##### Request body

```json
{
  "teams": [
    {
      "id": 1,
      "role": "observer"
    },
    {
      "id": 2,
      "role": "maintainer"
    }
  ]
}
```

##### Default response

`Status: 200`

```json
{
  "user": {
    "created_at": "2021-02-03T16:11:06Z",
    "updated_at": "2021-02-03T16:11:06Z",
    "id": 2,
    "name": "Jane Doe",
    "email": "janedoe@example.com",
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false,
    "global_role": "admin",
    "teams": [
      {
        "id": 2,
        "role": "observer"
      },
      {
        "id": 3,
        "role": "maintainer"
      }
    ]
  }
}
```

### Delete user

Delete the specified user from Fleet.

`DELETE /api/v1/fleet/users/{id}`

#### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The user's id. |

#### Example

`DELETE /api/v1/fleet/users/3`

##### Default response

`Status: 200`


### Require password reset

The selected user is logged out of Fleet and required to reset their password during the next attempt to log in. This also revokes all active Fleet API tokens for this user. Returns the user object.

`POST /api/v1/fleet/users/{id}/require_password_reset`

#### Parameters

| Name  | Type    | In   | Description                                                                                    |
| ----- | ------- | ---- | ---------------------------------------------------------------------------------------------- |
| id    | integer | path | **Required**. The user's id.                                                                   |
| require | boolean | body | Whether or not the user is required to reset their password during the next attempt to log in. |

#### Example

`POST /api/v1/fleet/users/{id}/require_password_reset`

##### Request body

```json
{
  "require": true
}
```

##### Default response

`Status: 200`

```json
{
  "user": {
    "created_at": "2021-02-23T22:23:34Z",
    "updated_at": "2021-02-23T22:28:52Z",
    "id": 2,
    "name": "Jane Doe",
    "email": "janedoe@example.com",
    "force_password_reset": true,
    "gravatar_url": "",
    "sso_enabled": false,
    "global_role": "observer",
    "teams": []
  }
}
```

### List a user's sessions

Returns a list of the user's sessions in Fleet.

`GET /api/v1/fleet/users/{id}/sessions`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/users/1/sessions`

##### Default response

`Status: 200`

```json
{
  "sessions": [
    {
      "session_id": 2,
      "user_id": 1,
      "created_at": "2021-02-03T16:12:50Z"
    },
    {
      "session_id": 3,
      "user_id": 1,
      "created_at": "2021-02-09T23:40:23Z"
    },
    {
      "session_id": 6,
      "user_id": 1,
      "created_at": "2021-02-23T22:23:58Z"
    }
  ]
}
```

### Delete a user's sessions

Deletes the selected user's sessions in Fleet. Also deletes the user's API token.

`DELETE /api/v1/fleet/users/{id}/sessions`

#### Parameters

| Name | Type    | In   | Description                               |
| ---- | ------- | ---- | ----------------------------------------- |
| id   | integer | path | **Required**. The ID of the desired user. |

#### Example

`DELETE /api/v1/fleet/users/1/sessions`

##### Default response

`Status: 200`

## Debug

- [Get a summary of errors](#get-a-summary-of-errors)
- [Get database information](#get-database-information)
- [Get profiling information](#get-profiling-information)

The Fleet server exposes a handful of API endpoints to retrieve debug information about the server itself in order to help troubleshooting. All the following endpoints require prior authentication meaning you must first log in successfully before calling any of the endpoints documented below.

### Get a summary of errors

Returns a set of all the errors that happened in the server during the interval of time defined by the [logging_error_retention_period](https://fleetdm.com/docs/deploying/configuration#logging-error-retention-period) configuration.

The server only stores and returns a single instance of each error.

`GET /debug/errors`

#### Parameters

| Name  | Type    | In    | Description                                                                       |
| ----- | ------- | ----- | --------------------------------------------------------------------------------- |
| flush | boolean | query | Whether or not clear the errors from Redis after reading them. Default is `false` |

#### Example

`GET /debug/errors?flush=true`

##### Default response

`Status: 200`

```json
[
  {
    "count": "3",
    "chain": [
      {
        "message": "Authorization header required"
      },
      {
        "message": "missing FleetError in chain",
        "data": {
          "timestamp": "2022-06-03T14:16:01-03:00"
        },
        "stack": [
          "github.com/fleetdm/fleet/v4/server/contexts/ctxerr.Handle (ctxerr.go:262)",
          "github.com/fleetdm/fleet/v4/server/service.encodeError (transport_error.go:80)",
          "github.com/go-kit/kit/transport/http.Server.ServeHTTP (server.go:124)"
        ]
      }
    ]
  }
]
```

### Get database information

Returns information about the current state of the database; valid keys are:

- `locks`: returns transaction locking information.
- `innodb-status`: returns InnoDB status information.
- `process-list`: returns running processes (queries, etc).

`GET /debug/db/{key}`

#### Parameters

None.

### Get profiling information

Returns runtime profiling data of the server in the format expected by `go tools pprof`. The responses are equivalent to those returned by the Go `http/pprof` package.

Valid keys are: `cmdline`, `profile`, `symbol` and `trace`.

`GET /debug/pprof/{key}`

#### Parameters
None.

## API errors

Fleet returns API errors as a JSON document with the following fields:
- `message`: This field contains the kind of error (bad request error, authorization error, etc.).
- `errors`: List of errors with `name` and `reason` keys.
- `uuid`: Unique identifier for the error. This identifier can be matched to Fleet logs which might contain more information about the cause of the error.

Sample of an error when trying to send an empty body on a request that expects a JSON body:
```sh
$ curl -k -H "Authorization: Bearer $TOKEN" -H 'Content-Type:application/json' "https://localhost:8080/api/v1/fleet/sso" -d ''
```
Response:
```json
{
  "message": "Bad request",
  "errors": [
    {
      "name": "base",
      "reason": "Expected JSON Body"
    }
  ],
  "uuid": "c0532a64-bec2-4cf9-aa37-96fe47ead814"
}
```

---

<meta name="description" value="Documentation for Fleet's REST API. See example requests and responses for each API endpoint.">
<meta name="pageOrderInSection" value="30">
