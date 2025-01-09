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

> For SSO and MFA users, email/password login is disabled. The API token can instead be retrieved from the "My account" page in the UI (/profile). On this page, choose "Get API token".

### Log in

Authenticates the user with the specified credentials. Use the token returned from this endpoint to authenticate further API requests.

`POST /api/v1/fleet/login`

> Logging in via the API is not supported for SSO and MFA users. The API token can instead be retrieved from the "My account" page in the UI (/profile). On this page, choose "Get API token".

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
    "mfa_enabled": false,
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

##### MFA Required

`Status: 202 Accepted`

```json
{
  "message": "We sent an email to you. Please click the magic link in the email to sign in.",
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

Returns a list of the activities that have been performed in Fleet. For a comprehensive list of activity types and detailed information, please see the [audit logs](https://fleetdm.com/docs/using-fleet/audit-activities) page.

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
      "created_at": "2023-07-27T14:35:08Z",
      "id": 25,
      "actor_full_name": "Anna Chao",
      "actor_id": 3,
      "actor_gravatar": "",
      "actor_email": "",
      "type": "uninstalled_software",
      "details": {
        "host_id": 1,
        "host_display_name": "Marko's MacBook Pro",
        "software_title": "Adobe Acrobat.app",
        "script_execution_id": "eeeddb94-52d3-4071-8b18-7322cd382abb",
        "status": "failed_install"
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

Fleet supports osquery's file carving functionality as of Fleet 3.3.0. This allows the Fleet server to request files (and sets of files) from Fleet's agent (fleetd), returning the full contents to Fleet.

To initiate a file carve using the Fleet API, you can use the [live query](#run-live-query) endpoint to run a query against the `carves` table.

For more information on executing a file carve in Fleet, go to the [File carving with Fleet docs](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/File-carving.md).

### List carves

Retrieves a list of the non expired carves. Carve contents remain available for 24 hours after the first data is provided from the osquery client.

`GET /api/v1/fleet/carves`

#### Parameters

| Name            | Type    | In    | Description                                                                                                                    |
|-----------------|---------|-------|--------------------------------------------------------------------------------------------------------------------------------|
| page            | integer | query | Page number of the results to fetch.                                                                                           |
| per_page        | integer | query | Results per page.                                                                                                              |
| order_key       | string  | query | What to order results by. Can be any field listed in the `results` array example below.                                        |
| order_direction | string  | query | **Requires `order_key`**. The direction of the order given the order key. Valid options are 'asc' or 'desc'. Default is 'asc'. |
| after           | string  | query | The value to get results after. This needs `order_key` defined, as that's the column that would be used.                       |
| expired         | boolean | query | Include expired carves (default: false)                                                                                        |

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

`GET /api/v1/fleet/carves/:id`

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

`GET /api/v1/fleet/carves/:id/block/:block_id`

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
- [Get team enroll secrets](#get-team-enroll-secrets)
- [Modify team enroll secrets](#modify-team-enroll-secrets)
- [Version](#version)

The Fleet server exposes API endpoints that handle the configuration of Fleet as well as endpoints that manage enroll secret operations. These endpoints require prior authentication, you so you'll need to log in before calling any of the endpoints documented below.

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

The `agent_options`, `sso_settings` and `smtp_settings` fields are only returned for admin users with global access. Learn more about roles and permissions [here](https://fleetdm.com/guides/role-based-access).

`mdm.macos_settings.custom_settings`, `mdm.windows_settings.custom_settings`, and `scripts` only include the configuration profiles and scripts applied using [Fleet's YAML](https://fleetdm.com/docs/configuration/yaml-files). To list profiles or scripts added in the UI or API, use the [List configuration profiles](https://fleetdm.com/docs/rest-api/rest-api#list-custom-os-settings-configuration-profiles) or [List scripts](https://fleetdm.com/docs/rest-api/rest-api#list-scripts) endpoints instead.

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
    "server_url": "https://instance.fleet.com",
    "enable_analytics": true,
    "live_query_disabled": false,
    "query_reports_disabled": false,
    "ai_features_disabled": false
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
  "activity_expiry_settings": {
    "activity_expiry_enabled": false,
    "activity_expiry_window": 0
  },
  "features": {
    "enable_host_users": true,
    "enable_software_inventory": true,
    "additional_queries": null
  },
  "mdm": {
    "windows_enabled_and_configured": true,
    "enable_disk_encryption": true,
    "macos_updates": {
      "minimum_version": "12.3.1",
      "deadline": "2022-01-01"
    },
    "ios_updates": {
      "minimum_version": "17.0.1",
      "deadline": "2024-08-01"
    },
    "ipados_updates": {
      "minimum_version": "17.0.1",
      "deadline": "2024-08-01"
    },
    "windows_updates": {
      "deadline_days": 5,
      "grace_period_days": 1
    },
    "macos_settings": {
      "custom_settings": [
        {
          "path": "path/to/profile1.mobileconfig",
          "labels": ["Label 1", "Label 2"]
        }
      ]
    },
    "windows_settings": {
      "custom_settings": [
        {
         "path": "path/to/profile2.xml",
         "labels": ["Label 3", "Label 4"]
        }
      ],
    },
    "scripts": ["path/to/script.sh"],
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
      "macos_setup_assistant": "path/to/config.json",
      "enable_release_device_manually": true
    },
    "client_url": "https://instance.fleet.com"
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
    },
    "activities_webhook":{
      "enable_activities_webhook":true,
      "destination_url": "https://server.com"
    }
  },
  "integrations": {
    "google_calendar": [
      {
        "domain": "example.com",
        "api_key_json": {
           "type": "service_account",
           "project_id": "fleet-in-your-calendar",
           "private_key_id": "<private key id>",
           "private_key": "-----BEGIN PRIVATE KEY-----\n<private key>\n-----END PRIVATE KEY-----\n",
           "client_email": "fleet-calendar-events@fleet-in-your-calendar.iam.gserviceaccount.com",
           "client_id": "<client id>",
           "auth_uri": "https://accounts.google.com/o/oauth2/auth",
           "token_uri": "https://oauth2.googleapis.com/token",
           "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
           "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/fleet-calendar-events%40fleet-in-your-calendar.iam.gserviceaccount.com",
           "universe_domain": "googleapis.com"
         }
      }
    ],
    "jira": [],
    "ndes_scep_proxy": {
      "admin_url": "https://example.com/certsrv/mscep_admin/",
      "password": "********",
      "url": "https://example.com/certsrv/mscep/mscep.dll",
      "username": "Administrator@example.com"
    },
    "zendesk": []
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
    "disable_schedule": false,
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

| Name                     | Type    | In    | Description   |
| -----------------------  | ------- | ----  | ------------------------------------------------------------------------------------------------------------------------------------ |
| org_info                 | object  | body  | See [org_info](#org-info).                                                                                                           |
| server_settings          | object  | body  | See [server_settings](#server-settings).                                                                                             |
| smtp_settings            | object  | body  | See [smtp_settings](#smtp-settings).                                                                                                 |
| sso_settings             | object  | body  | See [sso_settings](#sso-settings).                                                                                                   |
| host_expiry_settings     | object  | body  | See [host_expiry_settings](#host-expiry-settings).                                                                                   |
| activity_expiry_settings | object  | body  | See [activity_expiry_settings](#activity-expiry-settings).                                                                           |
| agent_options            | objects | body  | The agent_options spec that is applied to all hosts. In Fleet 4.0.0 the `api/v1/fleet/spec/osquery_options` endpoints were removed.  |
| fleet_desktop            | object  | body  | See [fleet_desktop](#fleet-desktop).                                                                                                 |
| webhook_settings         | object  | body  | See [webhook_settings](#webhook-settings).                                                                                           |
| integrations             | object  | body  | Includes `ndes_scep_proxy` object and `jira`, `zendesk`, and `google_calendar` arrays. See [integrations](#integrations) for details.                             |
| mdm                      | object  | body  | See [mdm](#mdm).                                                                                                                     |
| features                 | object  | body  | See [features](#features).                                                                                                           |
| scripts                  | array   | body  | A list of script files to add so they can be executed at a later time.                                                               |
| force                    | boolean | query | Whether to force-apply the agent options even if there are validation errors.                                                        |
| dry_run                  | boolean | query | Whether to validate the configuration and return any validation errors **without** applying changes.                                 |


#### Example

`PATCH /api/v1/fleet/config`

##### Request body

```json
{
  "scripts": []
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
    "server_url": "https://instance.fleet.com",
    "enable_analytics": true,
    "live_query_disabled": false,
    "query_reports_disabled": false,
    "ai_features_disabled": false
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
    "enable_sso": false,
    "enable_sso_idp_login": false,
    "enable_jit_provisioning": false
  },
  "host_expiry_settings": {
    "host_expiry_enabled": false,
    "host_expiry_window": 0
  },
  "activity_expiry_settings": {
    "activity_expiry_enabled": false,
    "activity_expiry_window": 0
  },
  "features": {
    "enable_host_users": true,
    "enable_software_inventory": true,
    "additional_queries": null
  },
  "license": {
    "tier": "free",
    "expiration": "0001-01-01T00:00:00Z"
  },
  "mdm": {
    "enabled_and_configured": false,
    "windows_enabled_and_configured": false,
    "enable_disk_encryption": true,
    "macos_updates": {
      "minimum_version": "12.3.1",
      "deadline": "2022-01-01"
    },
    "ios_updates": {
      "minimum_version": "17.0.1",
      "deadline": "2024-08-01"
    },
    "ipados_updates": {
      "minimum_version": "17.0.1",
      "deadline": "2024-08-01"
    },
    "windows_updates": {
      "deadline_days": 5,
      "grace_period_days": 1
    },
    "macos_settings": {
      "custom_settings": [
        {
          "path": "path/to/profile1.mobileconfig",
          "labels_exclude_any": ["Label 1", "Label 2"]
        },
        {
          "path": "path/to/profile2.json",
          "labels_include_all": ["Label 3", "Label 4"]
        },
	{
          "path": "path/to/profile3.json",
          "labels_include_any": ["Label 5", "Label 6"]
        },
      ]
    },
    "windows_settings": {
      "custom_settings": [
        {
          "path": "path/to/profile3.xml",
          "labels_exclude_any": ["Label 1", "Label 2"]
        }
      ]
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
    },
    "apple_server_url": "https://instance.fleet.com"
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
  "fleet_desktop": {
    "transparency_url": "https://fleetdm.com/better"
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
    },
    "activities_webhook":{
      "enable_activities_webhook":true,
      "destination_url": "https://server.com"
    }
  },
  "integrations": {
    "google_calendar": [
      {
        "domain": "",
        "api_key_json": null
      }
    ],
    "jira": [
      {
        "url": "https://jiraserver.com",
        "username": "some_user",
        "password": "sec4et!",
        "project_key": "jira_project",
        "enable_software_vulnerabilities": false
      }
    ],
    "ndes_scep_proxy": null,
    "zendesk": []
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
  "scripts": []
}
```


#### org_info

| Name                              | Type    | Description   |
| ---------------------             | ------- | ----------------------------------------------------------------------------------- |
| org_name                          | string  | The organization name.                                                              |
| org_logo_url                      | string  | The URL for the organization logo.                                                  |
| org_logo_url_light_background     | string  | The URL for the organization logo displayed in Fleet on top of light backgrounds.   |
| contact_url                       | string  | A URL that can be used by end users to contact the organization.                    |

<br/>

##### Example request body

```json
{
  "org_info": {
    "org_name": "Fleet Device Management",
    "org_logo_url": "https://fleetdm.com/logo.png",
    "org_logo_url_light_background": "https://fleetdm.com/logo-light.png",
    "contact_url": "https://fleetdm.com/company/contact"
  }
}
```

#### server_settings

| Name                              | Type    | Description   |
| ---------------------             | ------- | ------------------------------------------------------------------------------------------- |
| server_url                        | string  | The Fleet server URL.                                                                       |
| enable_analytics                  | boolean | Whether to send anonymous usage statistics. Always enabled for Fleet Premium customers.     |
| live_query_disabled               | boolean | Whether the live query capabilities are disabled.                                           |
| query_reports_disabled            | boolean | Whether query report capabilities are disabled.                                             |
| ai_features_disabled              | boolean | Whether AI features are disabled.                                                           |
| query_report_cap                  | integer | The maximum number of results to store per query report before the report is clipped. If increasing this cap, we recommend enabling reports for one query at time and monitoring your infrastructure. (Default: `1000`) |

<br/>

##### Example request body

```json
{
  "server_settings": {
    "server_url": "https://localhost:8080",
    "enable_analytics": true, 
    "live_query_disabled": false,
    "query_reports_disabled": false,
    "ai_features_disabled": false
  }
}
```

#### smtp_settings

| Name                              | Type    | Description   |
| ---------------------             | ------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| enable_smtp                       | boolean | Whether SMTP is enabled for the Fleet app.                                                                                                                            |
| sender_address                    | string  | The sender email address for the Fleet app. An invitation email is an example of the emails that may use this sender address                                          |
| server                            | string  | The SMTP server for the Fleet app.                                                                                                                                    |
| port                              | integer | The SMTP port for the Fleet app.                                                                                                                                      |
| authentication_type               | string  | The authentication type used by the SMTP server. Options include `"authtype_username_and_password"` or `"none"`                                                       |
| user_name                         | string  | The username used to authenticate requests made to the SMTP server.                                                                                                   |
| password                          | string  | The password used to authenticate requests made to the SMTP server.                                                                                                   |
| enable_ssl_tls                    | boolean | Whether or not SSL and TLS are enabled for the SMTP server.                                                                                                           |
| authentication_method             | string  | The authentication method used to make authenticate requests to SMTP server. Options include `"authmethod_plain"`, `"authmethod_cram_md5"`, and `"authmethod_login"`. |
| domain                            | string  | The domain for the SMTP server.                                                                                                                                       |
| verify_ssl_certs                  | boolean | Whether or not SSL certificates are verified by the SMTP server. Turn this off (not recommended) if you use a self-signed certificate.                                |
| enabled_start_tls                 | boolean | Detects if STARTTLS is enabled in your SMTP server and starts to use it.                                                                                              |

<br/>

##### Example request body

```json
{
  "smtp_settings": {
    "enable_smtp": true,
    "sender_address": "",
    "server": "localhost",
    "port": 1025,
    "authentication_type": "authtype_username_none",
    "user_name": "",
    "password": "",
    "enable_ssl_tls": true,
    "authentication_method": "authmethod_plain",
    "domain": "",
    "verify_ssl_certs": true,
    "enable_start_tls": true
  }
}
```

#### sso_settings

| Name                              | Type    | Description   |
| ---------------------             | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| enable_sso                        | boolean | Whether or not SSO is enabled for the Fleet application. If this value is true, you must also include most of the SSO settings parameters below.                       |
| entity_id                         | string  | The required entity ID is a URI that you use to identify Fleet when configuring the identity provider. Must be 5 or more characters.                                   |
| issuer_uri                        | string  | The URI you provide here must exactly match the Entity ID field used in the identity provider configuration.                                                           |
| idp_image_url                     | string  | An optional link to an image such as a logo for the identity provider.                                                                                                 |
| metadata_url                      | string  | A URL that references the identity provider metadata. If available from the identity provider, this is the preferred means of providing metadata. Must be either https or http |
| metadata                          | string  |  Metadata provided by the identity provider. Either `metadata` or a `metadata_url` must be provided.                                                                   |
| enable_sso_idp_login              | boolean | Determines whether Identity Provider (IdP) initiated login for Single sign-on (SSO) is enabled for the Fleet application.                                              |
| enable_jit_provisioning           | boolean | _Available in Fleet Premium._ When enabled, allows [just-in-time user provisioning](https://fleetdm.com/docs/deploy/single-sign-on-sso#just-in-time-jit-user-provisioning). |

<br/>

##### Example request body

```json
{
  "sso_settings": {
    "enable_sso": false,
    "entity_id": "",
    "issuer_uri": "",
    "idp_image_url": "",
    "metadata_url": "",
    "metadata": "",
    "idp_name": "",
    "enable_sso_idp_login": false,
    "enable_jit_provisioning": false
  }
}
```

#### host_expiry_settings

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| host_expiry_enabled               | boolean | When enabled, allows automatic cleanup of hosts that have not communicated with Fleet in some number of days.                                                  |
| host_expiry_window                | integer | If a host has not communicated with Fleet in the specified number of days, it will be removed. Must be greater than 0 if host_expiry_enabled is set to true.   |

<br/>

##### Example request body

```json
{
  "host_expiry_settings": {
    "host_expiry_enabled": true,
    "host_expiry_window": 7
  }
}
```

#### activity_expiry_settings

| Name                              | Type    | Description   |
| ---------------------             | ------- | --------------------------------------------------------------------------------------------------------------------------------- |
| activity_expiry_enabled           | boolean | When enabled, allows automatic cleanup of activities (and associated live query data) older than the specified number of days.    |
| activity_expiry_window            | integer | The number of days to retain activity records, if activity expiry is enabled.                                                     |

<br/>

##### Example request body

```json
{
  "activity_expiry_settings": {
    "activity_expiry_enabled": true,
    "activity_expiry_window": 90
  }
}
```

#### fleet_desktop

_Available in Fleet Premium._

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------- |
| transparency_url                  | string  | The URL used to display transparency information to users of Fleet Desktop.      |

<br/>

##### Example request body

```json
{
  "fleet_desktop": {
    "transparency_url": "https://fleetdm.com/better"
  }
}
```

#### webhook_settings

<!--
+ [`webhook_settings.host_status_webhook`](#webhook-settings-host-status-webhook)
+ [`webhook_settings.failing_policies_webhook`](#webhook-settings-failing-policies-webhook)
+ [`webhook_settings.vulnerabilities_webhook`](#webhook-settings-vulnerabilities-webhook)
+ [`webhook_settings.activities_webhook`](#webhook-settings-activities-webhook)
-->

| Name                              | Type  | Description   |
| ---------------------             | ----- | ---------------------------------------------------------------------------------------------- |
| host_status_webhook               | array | See [`webhook_settings.host_status_webhook`](#webhook-settings-host-status-webhook).           |
| failing_policies_webhook          | array | See [`webhook_settings.failing_policies_webhook`](#webhook-settings-failing-policies-webhook). |
| vulnerabilities_webhook           | array | See [`webhook_settings.vulnerabilities_webhook`](#webhook-settings-vulnerabilities-webhook).   |
| activities_webhook                | array | See [`webhook_settings.activities_webhook`](#webhook-settings-activities-webhook).             |

<br/>

##### webhook_settings.host_status_webhook

`webhook_settings.host_status_webhook` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| enable_host_status_webhook        | boolean | Whether or not the host status webhook is enabled.                                                                                          |
| destination_url                   | string  | The URL to deliver the webhook request to.                                                                                                  |
| host_percentage                   | integer | The minimum percentage of hosts that must fail to check in to Fleet in order to trigger the webhook request.                                |
| days_count                        | integer | The minimum number of days that the configured `host_percentage` must fail to check in to Fleet in order to trigger the webhook request.    |

<br/>

##### webhook_settings.failing_policies_webhook

`webhook_settings.failing_policies_webhook` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | ------------------------------------------------------------------------------------------------------------------- |
| enable_failing_policies_webhook   | boolean | Whether or not the failing policies webhook is enabled.                                                             |
| destination_url                   | string  | The URL to deliver the webhook requests to.                                                                         |
| policy_ids                        | array   | List of policy IDs to enable failing policies webhook.                                                              |
| host_batch_size                   | integer | Maximum number of hosts to batch on failing policy webhook requests. The default, 0, means no batching (all hosts failing a policy are sent on one request). |

<br/>

##### webhook_settings.vulnerabilities_webhook

`webhook_settings.vulnerabilities_webhook` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| enable_vulnerabilities_webhook    | boolean | Whether or not the vulnerabilities webhook is enabled.                                                                                                  |
| destination_url                   | string  | The URL to deliver the webhook requests to.                                                                                                             |
| host_batch_size                   | integer | Maximum number of hosts to batch on vulnerabilities webhook requests. The default, 0, means no batching (all vulnerable hosts are sent on one request). |

<br/>

##### webhook_settings.activities_webhook

`webhook_settings.activities_webhook` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | --------------------------------------------------------- |
| enable_activities_webhook         | boolean | Whether or not the activity feed webhook is enabled.      |
| destination_url                   | string  | The URL to deliver the webhook requests to.               |

<br/>

##### Example request body

```json
{
  "webhook_settings": {
    "host_status_webhook": {
      "enable_host_status_webhook": true,
      "destination_url": "https://server.com",
      "host_percentage": 5,
      "days_count": 7
    },
    "failing_policies_webhook":{
      "enable_failing_policies_webhook": true,
      "destination_url": "https://server.com",
      "policy_ids": [1, 2, 3],
      "host_batch_size": 1000
    },
    "vulnerabilities_webhook":{
      "enable_vulnerabilities_webhook":true,
      "destination_url": "https://server.com",
      "host_batch_size": 1000
    },
    "activities_webhook":{
      "enable_activities_webhook":true,
      "destination_url": "https://server.com"
    }
  }
}
```

#### integrations

<!--
+ [`integrations.jira`](#integrations-jira)
+ [`integrations.zendesk`](#integrations-zendesk)
+ [`integrations.google_calendar`](#integrations-google-calendar)
+ [`integrations.ndes_scep_proxy`](#integrations-ndes_scep_proxy)
-->

| Name            | Type   | Description                                                          |
|-----------------|--------|----------------------------------------------------------------------|
| jira            | array  | See [`integrations.jira`](#integrations-jira).                       |
| zendesk         | array  | See [`integrations.zendesk`](#integrations-zendesk).                 |
| google_calendar | array  | See [`integrations.google_calendar`](#integrations-google-calendar). |
| ndes_scep_proxy | object | See [`integrations.ndes_scep_proxy`](#integrations-ndes-scep-proxy). |

<br/>

##### integrations.jira

`integrations.jira` is an array of objects with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| enable_software_vulnerabilities   | boolean | Whether or not Jira integration is enabled for software vulnerabilities. Only one vulnerability automation can be enabled at a given time (enable_vulnerabilities_webhook and enable_software_vulnerabilities). |
| enable_failing_policies           | boolean | Whether or not Jira integration is enabled for failing policies. Only one failing policy automation can be enabled at a given time (enable_failing_policies_webhook and enable_failing_policies). |
| url                               | string  | The URL of the Jira server to integrate with. |
| username                          | string  | The Jira username to use for this Jira integration. |
| api_token                         | string  | The API token of the Jira username to use for this Jira integration. |
| project_key                       | string  | The Jira project key to use for this integration. Jira tickets will be created in this project. |

<br/>

> Note that when making changes to the `integrations.jira` array, all integrations must be provided (not just the one being modified). This is because the endpoint will consider missing integrations as deleted.

##### integrations.zendesk

`integrations.zendesk` is an array of objects with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| enable_software_vulnerabilities   | boolean | Whether or not Zendesk integration is enabled for software vulnerabilities. Only one vulnerability automation can be enabled at a given time (enable_vulnerabilities_webhook and enable_software_vulnerabilities). |
| enable_failing_policies           | boolean | Whether or not Zendesk integration is enabled for failing policies. Only one failing policy automation can be enabled at a given time (enable_failing_policies_webhook and enable_failing_policies). |
| url                               | string  | The URL of the Zendesk server to integrate with. |
| email                             | string  | The Zendesk user email to use for this Zendesk integration. |
| api_token                         | string  | The Zendesk API token to use for this Zendesk integration. |
| group_id                          | integer | The Zendesk group id to use for this integration. Zendesk tickets will be created in this group. |

<br/>

> Note that when making changes to the `integrations.zendesk` array, all integrations must be provided (not just the one being modified). This is because the endpoint will consider missing integrations as deleted.

##### integrations.google_calendar

`integrations.google_calendar` is an array of objects with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | --------------------------------------------------------------------------------------------------------------------- |
| domain                            | string  | The domain for the Google Workspace service account to be used for this calendar integration.                         |
| api_key_json                      | object  | The private key JSON downloaded when generating the service account API key to be used for this calendar integration. |

<br/>

##### integrations.ndes_scep_proxy

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.
`integrations.ndes_scep_proxy` is an object with the following structure:

| Name      | Type   | Description                                             |
|-----------|--------|---------------------------------------------------------|
| url       | string | **Required**. The URL of the NDES SCEP endpoint.        |
| admin_url | string | **Required**. The URL of the NDES admin endpoint.       |
| password  | string | **Required**. The password for the NDES admin endpoint. |
| username  | string | **Required**. The username for the NDES admin endpoint. |

Setting `integrations.ndes_scep_proxy` to `null` will clear existing settings. Not specifying `integrations.ndes_scep_proxy` in the payload will not change the existing settings.



##### Example request body

```json
{
  "integrations": {
    "jira": [
      {
        "enable_software_vulnerabilities": false,
        "enable_failing_poilicies": true,
        "url": "https://jiraserver.com",
        "username": "some_user",
        "api_token": "<TOKEN>",
        "project_key": "jira_project",
      }
    ],
    "zendesk": [],
    "google_calendar": [
      {
        "domain": "https://domain.com",
        "api_key_json": "<API KEY JSON>"
      }
    ],
    "ndes_scep_proxy": {
      "admin_url": "https://example.com/certsrv/mscep_admin/",
      "password": "abc123",
      "url": "https://example.com/certsrv/mscep/mscep.dll",
      "username": "Administrator@example.com"
    }
  }
}
```

#### mdm

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| windows_enabled_and_configured    | boolean | Enables Windows MDM support. |
| enable_disk_encryption            | boolean | _Available in Fleet Premium._ Hosts that belong to no team will have disk encryption enabled if set to true. |
| macos_updates         | object  | See [`mdm.macos_updates`](#mdm-macos-updates). |
| ios_updates         | object  | See [`mdm.ios_updates`](#mdm-ios-updates). |
| ipados_updates         | object  | See [`mdm.ipados_updates`](#mdm-ipados-updates). |
| windows_updates         | object  | See [`mdm.window_updates`](#mdm-windows-updates). |
| macos_migration         | object  | See [`mdm.macos_migration`](#mdm-macos-migration). |
| macos_setup         | object  | See [`mdm.macos_setup`](#mdm-macos-setup). |
| macos_settings         | object  | See [`mdm.macos_settings`](#mdm-macos-settings). |
| windows_settings         | object  | See [`mdm.windows_settings`](#mdm-windows-settings). |
| apple_server_url         | string  | Update this URL if you're self-hosting Fleet and you want your hosts to talk to this URL for MDM features. (If not configured, hosts will use the base URL of the Fleet instance.)  |

> Note: If `apple_server_url` changes and Apple (macOS, iOS, iPadOS) hosts already have MDM turned on, the end users will have to turn MDM off and back on to use MDM features.

<br/>

##### mdm.macos_updates

_Available in Fleet Premium._

`mdm.macos_updates` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| minimum_version                   | string  | Hosts that belong to no team and are enrolled into Fleet's MDM will be prompted to update when their OS is below this version. |
| deadline                          | string  | Hosts that belong to no team and are enrolled into Fleet's MDM will be forced to update their OS after this deadline (noon local time for hosts already on macOS 14 or above, 20:00 UTC for hosts on earlier macOS versions). |

<br/>

##### mdm.ios_updates

_Available in Fleet Premium._

`mdm.ios_updates` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| minimum_version                   | string  | Hosts that belong to no team will be prompted to update when their OS is below this version. |
| deadline                          | string  | Hosts that belong to no team will be forced to update their OS after this deadline (noon local time). |

<br/>

##### mdm.ipados_updates

_Available in Fleet Premium._

`mdm.ipados_updates` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| minimum_version                   | string  | Hosts that belong to no team will be prompted to update when their OS is below this version. |
| deadline                          | string  | Hosts that belong to no team will be forced to update their OS after this deadline (noon local time). |

<br/>

##### mdm.windows_updates

_Available in Fleet Premium._

`mdm.windows_updates` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| deadline_days                     | integer | Hosts that belong to no team and are enrolled into Fleet's MDM will have this number of days before updates are installed on Windows. |
| grace_period_days                 | integer | Hosts that belong to no team and are enrolled into Fleet's MDM will have this number of days before Windows restarts to install updates. |

<br/>

##### mdm.macos_migration

_Available in Fleet Premium._

`mdm.macos_migration` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| enable                            | boolean | Whether to enable the end user migration workflow for devices migrating from your old MDM solution. |
| mode                              | string  | The end user migration workflow mode for devices migrating from your old MDM solution. Options are `"voluntary"` or `"forced"`. |
| webhook_url                       | string  | The webhook url configured to receive requests to unenroll devices migrating from your old MDM solution. |

<br/>

##### mdm.macos_setup

_Available in Fleet Premium._

`mdm.macos_setup` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| enable_end_user_authentication    | boolean | If set to true, end user authentication will be required during automatic MDM enrollment of new macOS devices. Settings for your IdP provider must also be [configured](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#end-user-authentication-and-eula). |

<br/>

##### mdm.macos_settings

`mdm.macos_settings` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| custom_settings                   | array   | Only intended to be used by [Fleet's YAML](https://fleetdm.com/docs/configuration/yaml-files). To add macOS configuration profiles using Fleet's API, use the [Add configuration profile endpoint](https://fleetdm.com/docs/rest-api/rest-api#add-custom-os-setting-configuration-profile) instead. |

<br/>

##### mdm.windows_settings

`mdm.windows_settings` is an object with the following structure:

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| custom_settings                   | array   | Only intended to be used by [Fleet's YAML](https://fleetdm.com/docs/configuration/yaml-files). To add Windows configuration profiles using Fleet's API, use the [Add configuration profile endpoint](https://fleetdm.com/docs/rest-api/rest-api#add-custom-os-setting-configuration-profile) instead. |

<br/>

##### Example request body

```json
{
  "mdm": {
    "windows_enabled_and_configured": false,
    "enable_disk_encryption": true,
    "macos_updates": {
      "minimum_version": "12.3.1",
      "deadline": "2022-01-01"
    },
    "windows_updates": {
      "deadline_days": 5,
      "grace_period_days": 1
    },
    "macos_settings": {
      "custom_settings": [
        {
          "path": "path/to/profile1.mobileconfig",
          "labels": ["Label 1", "Label 2"]
        },
        {
          "path": "path/to/profile2.json",
          "labels": ["Label 3", "Label 4"]
        },
      ]
    },
    "windows_settings": {
      "custom_settings": [
        {
          "path": "path/to/profile3.xml",
          "labels": ["Label 1", "Label 2"]
        }
      ]     
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
  }
}
```

#### Features

| Name                              | Type    | Description   |
| ---------------------             | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| enable_host_users                 | boolean | Whether to enable the users feature in Fleet. (Default: `true`)                                                                          |
| enable_software_inventory         | boolean | Whether to enable the software inventory feature in Fleet. (Default: `true`)                                                             |
| additional_queries                | boolean | Whether to enable additional queries on hosts. (Default: `null`)                                                                         |

<br/>

##### Example request body

```json
{
  "features": {
    "enable_host_users": true,
    "enable_software_inventory": true,
    "additional_queries": null
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

### Get team enroll secrets

Returns the valid team enroll secrets.

`GET /api/v1/fleet/teams/:id/secrets`

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


### Modify team enroll secrets

Replaces all existing team enroll secrets.

`PATCH /api/v1/fleet/teams/:id/secrets`

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
  "build_date": "2021-03-27",
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
- [Turn off MDM for a host](#turn-off-mdm-for-a-host)
- [Bulk delete hosts by filter or ids](#bulk-delete-hosts-by-filter-or-ids)
- [Get human-device mapping](#get-human-device-mapping)
- [Update custom human-device mapping](#update-custom-human-device-mapping)
- [Get host's device health report](#get-hosts-device-health-report)
- [Get host's mobile device management (MDM) information](#get-hosts-mobile-device-management-mdm-information)
- [Get mobile device management (MDM) summary](#get-mobile-device-management-mdm-summary)
- [Get host's mobile device management (MDM) and Munki information](#get-hosts-mobile-device-management-mdm-and-munki-information)
- [Get aggregated host's mobile device management (MDM) and Munki information](#get-aggregated-hosts-macadmin-mobile-device-management-mdm-and-munki-information)
- [Get host's scripts](#get-hosts-scripts)
- [Get host's software](#get-hosts-software)
- [Get hosts report in CSV](#get-hosts-report-in-csv)
- [Get host's disk encryption key](#get-hosts-disk-encryption-key)
- [Lock host](#lock-host)
- [Unlock host](#unlock-host)
- [Wipe host](#wipe-host)
- [Get host's past activity](#get-hosts-past-activity)
- [Get host's upcoming activity](#get-hosts-upcoming-activity)
- [Add labels to host](#add-labels-to-host)
- [Remove labels from host](#remove-labels-from-host)
- [Live query one host (ad-hoc)](#live-query-one-host-ad-hoc)
- [Live query host by identifier (ad-hoc)](#live-query-host-by-identifier-ad-hoc)

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
- `last_restarted_at`: the last time that the host was restarted.

### List hosts

`GET /api/v1/fleet/hosts`

#### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                                                                                                                                                                                 |
| ----------------------- | ------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                                                                                                                                                                                        |
| per_page                | integer | query | Results per page.                                                                                                                                                                                                                                                                                                                           |
| order_key               | string  | query | What to order results by. Can be any column in the hosts table.                                                                                                                                                                                                                                                                             |
| after                   | string  | query | The value to get results after. This needs `order_key` defined, as that's the column that would be used. **Note:** Use `page` instead of `after`                                                                                                                                                                                                                                    |
| order_direction         | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include 'asc' and 'desc'. Default is 'asc'.                                                                                                                                                                                                               |
| status                  | string  | query | Indicates the status of the hosts to return. Can either be 'new', 'online', 'offline', 'mia' or 'missing'.                                                                                                                                                                                                                                  |
| query                   | string  | query | Search query keywords. Searchable fields include `hostname`, `hardware_serial`, `uuid`, `ipv4` and the hosts' email addresses (only searched if the query looks like an email address, i.e. contains an '@', no space, etc.).                                                                                                                |
| additional_info_filters | string  | query | A comma-delimited list of fields to include in each host's `additional` object.                                              |
| team_id                 | integer | query | _Available in Fleet Premium_. Filters to only include hosts in the specified team. Use `0` to filter by hosts assigned to "No team".                                                                                                                                                                                                                                                |
| policy_id               | integer | query | The ID of the policy to filter hosts by.                                                                                                                                                                                                                                                                                                    |
| policy_response         | string  | query | **Requires `policy_id`**. Valid options are 'passing' or 'failing'.                                                                                                                                                                                                                                       |
| software_version_id     | integer | query | The ID of the software version to filter hosts by.                                                                                                                                                                                                                                                                                                  |
| software_title_id       | integer | query | The ID of the software title to filter hosts by.                                                                                                                                                                                                                                                                                                  |
| software_status       | string | query | The status of the software install to filter hosts by.                                                                                                                                                                                                                                                                                                  |
| os_version_id | integer | query | The ID of the operating system version to filter hosts by. |
| os_name                 | string  | query | The name of the operating system to filter hosts by. `os_version` must also be specified with `os_name`                                                                                                                                                                                                                                     |
| os_version              | string  | query | The version of the operating system to filter hosts by. `os_name` must also be specified with `os_version`                                                                                                                                                                                                                                  |
| vulnerability           | string  | query | The cve to filter hosts by (including "cve-" prefix, case-insensitive).                                                                                                                                                                                                                                                                     |
| device_mapping          | boolean | query | Indicates whether `device_mapping` should be included for each host. See ["Get host's Google Chrome profiles](#get-hosts-google-chrome-profiles) for more information about this feature.                                                                                                                                                  |
| mdm_id                  | integer | query | The ID of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider and URL).                                                                                                                                                                                                |
| mdm_name                | string  | query | The name of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider).                                                                                                                                                                                                |
| mdm_enrollment_status   | string  | query | The _mobile device management_ (MDM) enrollment status to filter hosts by. Valid options are 'manual', 'automatic', 'enrolled', 'pending', or 'unenrolled'.                                                                                                                                                                                                             |
| connected_to_fleet   | boolean  | query | Filter hosts that are talking to this Fleet server for MDM features. In rare cases, hosts can be enrolled to one Fleet server but talk to a different Fleet server for MDM features. In this case, the value would be `false`. Always `false` for Linux hosts.                                                                                                                           |
| macos_settings          | string  | query | Filters the hosts by the status of the _mobile device management_ (MDM) profiles applied to hosts. Valid options are 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.**                                                                                                                                                                                                             |
| munki_issue_id          | integer | query | The ID of the _munki issue_ (a Munki-reported error or warning message) to filter hosts by (that is, filter hosts that are affected by that corresponding error or warning message).                                                                                                                                                        |
| low_disk_space          | integer | query | _Available in Fleet Premium_. Filters the hosts to only include hosts with less GB of disk space available than this value. Must be a number between 1-100.                                                                                                                                                                                  |
| disable_failing_policies| boolean | query | If `true`, hosts will return failing policies as 0 regardless of whether there are any that failed for the host. This is meant to be used when increased performance is needed in exchange for the extra information.                                                                                                                       |
| macos_settings_disk_encryption | string | query | Filters the hosts by disk encryption status. Valid options are 'verified', 'verifying', 'action_required', 'enforcing', 'failed', or 'removing_enforcement'. |
| bootstrap_package       | string | query | _Available in Fleet Premium_. Filters the hosts by the status of the MDM bootstrap package on the host. Valid options are 'installed', 'pending', or 'failed'. |
| os_settings          | string  | query | Filters the hosts by the status of the operating system settings applied to the hosts. Valid options are 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |
| os_settings_disk_encryption | string | query | Filters the hosts by disk encryption status. Valid options are 'verified', 'verifying', 'action_required', 'enforcing', 'failed', or 'removing_enforcement'.  **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |
| populate_software     | string | query | If `false` (or omitted), omits installed software details for each host. If `"without_vulnerability_details"`, include a list of installed software for each host, including which CVEs apply to the installed software versions. `true` adds vulnerability description, CVSS score, and other details when using Fleet Premium. See notes below on performance. |
| populate_policies     | boolean | query | If `true`, the response will include policy data for each host. |

> `software_id` is deprecated as of Fleet 4.42. It is maintained for backwards compatibility. Please use the `software_version_id` instead.

> `populate_software` returns a lot of data per host when set, and drastically more data when set to `true` on Fleet Premium. If you need vulnerability details for a large number of hosts, consider setting `populate_software` to `without_vulnerability_details` and pulling vulnerability details from the [Get vulnerability](#get-vulnerability) endpoint, as this returns details once per vulnerability rather than once per vulnerability per host.

If `software_title_id` is specified, an additional top-level key `"software_title"` is returned with the software title object corresponding to the `software_title_id`. See [List software](#list-software) response payload for details about this object.

If `software_version_id` is specified, an additional top-level key `"software"` is returned with the software object corresponding to the `software_version_id`. See [List software versions](#list-software-versions) response payload for details about this object.

If `additional_info_filters` is not specified, no `additional` information will be returned.

If `mdm_id` is specified, an additional top-level key `"mobile_device_management_solution"` is returned with the information corresponding to the `mdm_id`.

If `mdm_id`, `mdm_name`, `mdm_enrollment_status`, `os_settings`, or `os_settings_disk_encryption` is specified, then Windows Servers are excluded from the results.

If `munki_issue_id` is specified, an additional top-level key `munki_issue` is returned with the information corresponding to the `munki_issue_id`.

If `after` is being used with `created_at` or `updated_at`, the table must be specified in `order_key`. Those columns become `h.created_at` and `h.updated_at`.

#### Example

`GET /api/v1/fleet/hosts?page=0&per_page=100&order_key=hostname&query=2ce&populate_software=true&populate_policies=true`

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
      "last_restarted_at": "2020-11-01T03:01:45Z",
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
      "gigs_disk_space_available": 174.98,
      "percent_disk_space_available": 71,
      "gigs_total_disk_space": 246,
      "pack_stats": [
        {
          "pack_id": 0,
          "pack_name": "Global",
          "type": "global",
          "query_stats": [
            {
            "scheduled_query_name": "Get recently added or removed USB drives",
            "scheduled_query_id": 5535,
            "query_name": "Get recently added or removed USB drives",
            "discard_data": false,
            "last_fetched": null,
            "automations_enabled": false,
            "description": "Returns a record every time a USB device is plugged in or removed",
            "pack_name": "Global",
            "average_memory": 434176,
            "denylisted": false,
            "executions": 2,
            "interval": 86400,
            "last_executed": "2023-11-28T00:02:07Z",
            "output_size": 891,
            "system_time": 10,
            "user_time": 6,
            "wall_time": 0
            }
          ]
        }
      ],
      "issues": {
        "failing_policies_count": 1,
        "critical_vulnerabilities_count": 2, // Fleet Premium only
        "total_issues_count": 3
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
        "enrollment_status": "Pending",
        "dep_profile_error": true,
        "name": "Fleet",
        "server_url": "https://example.fleetdm.com/mdm/apple/mdm"
      },
      "software": [
        {
          "id": 1,
          "name": "glibc",
          "version": "2.12",
          "source": "rpm_packages",
          "generated_cpe": "cpe:2.3:a:gnu:glibc:2.12:*:*:*:*:*:*:*",
          "vulnerabilities": [
            {
              "cve": "CVE-2009-5155",
              "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2009-5155",
              "cvss_score": 7.5, // Fleet Premium only
              "epss_probability": 0.01537, // Fleet Premium only
              "cisa_known_exploit": false, // Fleet Premium only
              "cve_published": "2022-01-01 12:32:00", // Fleet Premium only
              "cve_description": "In the GNU C Library (aka glibc or libc6) before 2.28, parse_reg_exp in posix/regcomp.c misparses alternatives, which allows attackers to cause a denial of service (assertion failure and application exit) or trigger an incorrect result by attempting a regular-expression match.", // Fleet Premium only
              "resolved_in_version": "2.28" // Fleet Premium only
            }
          ],
          "installed_paths": ["/usr/lib/some-path-1"]
        }
      ],
      "policies": [
        {
          "id": 1,
          "name": "Gatekeeper enabled",
          "query": "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
          "description": "Checks if gatekeeper is enabled on macOS devices",
          "resolution": "Fix with these steps...",
          "platform": "darwin",
          "response": "fail",
          "critical": false
        }
      ]
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
| order_direction         | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include 'asc' and 'desc'. Default is 'asc'.                                                                                                                                                                                                               |
| after                   | string  | query | The value to get results after. This needs `order_key` defined, as that's the column that would be used.                                                                                                                                                                                                                                    |
| status                  | string  | query | Indicates the status of the hosts to return. Can either be 'new', 'online', 'offline', 'mia' or 'missing'.                                                                                                                                                                                                                                  |
| query                   | string  | query | Search query keywords. Searchable fields include `hostname`, `hardware_serial`, `uuid`, `ipv4` and the hosts' email addresses (only searched if the query looks like an email address, i.e. contains an '@', no space, etc.).                                                                                                                |
| team_id                 | integer | query | _Available in Fleet Premium_. Filters the hosts to only include hosts in the specified team.                                                                                                                                                                                                                                                 |
| policy_id               | integer | query | The ID of the policy to filter hosts by.                                                                                                                                                                                                                                                                                                    |
| policy_response         | string  | query | **Requires `policy_id`**. Valid options are 'passing' or 'failing'.                                                                                                                                                                                                                                       |
| software_version_id     | integer | query | The ID of the software version to filter hosts by.                                                                                                            |
| software_title_id       | integer | query | The ID of the software title to filter hosts by.                                                                                                              |
| os_version_id | integer | query | The ID of the operating system version to filter hosts by. |
| os_name                 | string  | query | The name of the operating system to filter hosts by. `os_version` must also be specified with `os_name`                                                                                                                                                                                                                                     |
| os_version              | string  | query | The version of the operating system to filter hosts by. `os_name` must also be specified with `os_version`                                                                                                                                                                                                                                  |
| vulnerability           | string  | query | The cve to filter hosts by (including "cve-" prefix, case-insensitive).                                                                                                                                                                                                                                                                     |
| label_id                | integer | query | A valid label ID. Can only be used in combination with `order_key`, `order_direction`, `after`, `status`, `query` and `team_id`.                                                                                                                                                                                                            |
| mdm_id                  | integer | query | The ID of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider and URL).                                                                                                                                                                                                |
| mdm_name                | string  | query | The name of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider).                                                                                                                                                                                                |
| mdm_enrollment_status   | string  | query | The _mobile device management_ (MDM) enrollment status to filter hosts by. Valid options are 'manual', 'automatic', 'enrolled', 'pending', or 'unenrolled'.                                                                                                                                                                                                             |
| macos_settings          | string  | query | Filters the hosts by the status of the _mobile device management_ (MDM) profiles applied to hosts. Valid options are 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.**                                                                                                                                                                                                             |
| munki_issue_id          | integer | query | The ID of the _munki issue_ (a Munki-reported error or warning message) to filter hosts by (that is, filter hosts that are affected by that corresponding error or warning message).                                                                                                                                                        |
| low_disk_space          | integer | query | _Available in Fleet Premium_. Filters the hosts to only include hosts with less GB of disk space available than this value. Must be a number between 1-100.                                                                                                                                                                                  |
| macos_settings_disk_encryption | string | query | Filters the hosts by disk encryption status. Valid options are 'verified', 'verifying', 'action_required', 'enforcing', 'failed', or 'removing_enforcement'. |
| bootstrap_package       | string | query | _Available in Fleet Premium_. Filters the hosts by the status of the MDM bootstrap package on the host. Valid options are 'installed', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |
| os_settings          | string  | query | Filters the hosts by the status of the operating system settings applied to the hosts. Valid options are 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |
| os_settings_disk_encryption | string | query | Filters the hosts by disk encryption status. Valid options are 'verified', 'verifying', 'action_required', 'enforcing', 'failed', or 'removing_enforcement'.  **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |

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
| team_id         | integer | query | _Available in Fleet Premium_. The ID of the team whose host counts should be included. Defaults to all teams. |
| platform        | string  | query | Platform to filter by when counting. Defaults to all platforms.                 |
| low_disk_space  | integer | query | _Available in Fleet Premium_. Returns the count of hosts with less GB of disk space available than this value. Must be a number between 1-100. |

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
    },
    {
      "id": 13,
      "name": "iOS",
      "description": "All iOS hosts",
      "label_type": "builtin"
    },
    {
      "id": 14,
      "name": "iPadOS",
      "description": "All iPadOS hosts",
      "label_type": "builtin"
    },
    {
      "id": 15,
      "name": "Fedora Linux",
      "description": "All Fedora hosts",
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
      "platform": "ios",
      "hosts_count": 1234
    },
    {
      "platform": "ipados",
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

`GET /api/v1/fleet/hosts/:id`

#### Parameters

| Name             | Type    | In    | Description                                                                         |
|------------------|---------|-------|-------------------------------------------------------------------------------------|
| id               | integer | path  | **Required**. The host's id.                                                        |
| exclude_software | boolean | query | If `true`, the response will not include a list of installed software for the host. |

#### Example

`GET /api/v1/fleet/hosts/121`

##### Default response

`Status: 200`

```json
{
  "host": {
    "created_at": "2021-08-19T02:02:22Z",
    "updated_at": "2021-08-19T21:14:58Z",
    "id": 1,
    "detail_updated_at": "2021-08-19T21:07:53Z",
    "last_restarted_at": "2020-11-01T03:01:45Z",
    "software_updated_at": "2020-11-05T05:09:44Z",
    "label_updated_at": "2021-08-19T21:07:53Z",
    "policy_updated_at": "2023-06-26T18:33:15Z",
    "last_enrolled_at": "2021-08-19T02:02:22Z",
    "seen_time": "2021-08-19T21:14:58Z",
    "refetch_requested": false,
    "hostname": "23cfc9caacf0",
    "uuid": "309a4b7d-0000-0000-8e7f-26ae0815ede8",
    "platform": "rhel",
    "osquery_version": "5.12.0",
    "orbit_version": "1.22.0",
    "fleet_desktop_version": "1.22.0",
    "scripts_enabled": true,
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
    "percent_disk_space_available": 74,
    "gigs_total_disk_space": 160,
    "disk_encryption_enabled": true,
    "status": "online",
    "display_text": "23cfc9caacf0",
    "issues": {
        "failing_policies_count": 1,
        "critical_vulnerabilities_count": 2, // Available in Fleet Premium
        "total_issues_count": 3
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
    "maintenance_window": {
      "starts_at": "2024-06-18T13:27:18−04:00",
      "timezone": "America/New_York"
    },
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
    "policies": [
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
      },
      {
        "id": 1,
        "name": "SomeQuery",
        "query": "SELECT * FROM foo;",
        "description": "this is a query",
        "resolution": "fix with these steps...",
        "platform": "windows,linux",
        "response": "pass",
        "critical": false
      }
    ],
    "software": [
      {
        "id": 408,
        "name": "osquery",
        "version": "4.5.1",
        "source": "rpm_packages",
        "browser": "",
        "generated_cpe": "",
        "vulnerabilities": null,
        "installed_paths": ["/usr/lib/some-path-1"]
      },
      {
        "id": 1146,
        "name": "tar",
        "version": "1.30",
        "source": "rpm_packages",
        "browser": "",
        "generated_cpe": "",
        "vulnerabilities": null
      },
      {
        "id": 321,
        "name": "SomeApp.app",
        "version": "1.0",
        "source": "apps",
        "browser": "",
        "bundle_identifier": "com.some.app",
        "last_opened_at": "2021-08-18T21:14:00Z",
        "generated_cpe": "",
        "vulnerabilities": null,
        "installed_paths": ["/usr/lib/some-path-2"]
      }
    ],
    "mdm": {
      "encryption_key_available": true,
      "enrollment_status": "On (manual)",
      "name": "Fleet",
      "connected_to_fleet": true,
      "server_url": "https://acme.com/mdm/apple/mdm",
      "device_status": "unlocked",
      "pending_action": "",
      "macos_settings": {
        "disk_encryption": null,
        "action_required": null
      },
      "macos_setup": {
        "bootstrap_package_status": "installed",
        "detail": "",
        "bootstrap_package_name": "test.pkg"
      },
      "os_settings": {
        "disk_encryption": {
          "status": null,
          "detail": ""
        }
      },
      "profiles": [
        {
          "profile_uuid": "954ec5ea-a334-4825-87b3-937e7e381f24",
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

> Note: `signature_information` is only set for macOS (.app) applications.

> Note:
> - `orbit_version: null` means this agent is not a fleetd agent
> - `fleet_desktop_version: null` means this agent is not a fleetd agent, or this agent is version <=1.23.0 which is not collecting the desktop version
> - `fleet_desktop_version: ""` means this agent is a fleetd agent but does not have fleet desktop
> - `scripts_enabled: null` means this agent is not a fleetd agent, or this agent is version <=1.23.0 which is not collecting the scripts enabled info

### Get host by identifier

Returns the information of the host specified using the `hostname`, `uuid`, or `hardware_serial` as an identifier.

If `hostname` is specified when there is more than one host with the same hostname, the endpoint returns the first matching host. In Fleet, hostnames are fully qualified domain names (FQDNs). `hostname` (e.g. johns-macbook-air.local) is not the same as `display_name` (e.g. John's MacBook Air).

`GET /api/v1/fleet/hosts/identifier/:identifier`

#### Parameters

| Name       | Type              | In   | Description                                                        |
| ---------- | ----------------- | ---- | ------------------------------------------------------------------ |
| identifier | string | path | **Required**. The host's `hostname`, `uuid`, or `hardware_serial`. |
| exclude_software | boolean | query | If `true`, the response will not include a list of installed software for the host. |

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
        "browser": "",
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
    "gigs_total_disk_space": 192,
    "issues": {
        "failing_policies_count": 1,
        "critical_vulnerabilities_count": 2, // Fleet Premium only
        "total_issues_count": 3
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
      "device_status": "unlocked",
      "pending_action": "lock",
      "macos_settings": {
        "disk_encryption": null,
        "action_required": null
      },
      "macos_setup": {
        "bootstrap_package_status": "installed",
        "detail": ""
      },
      "os_settings": {
        "disk_encryption": {
          "status": null,
          "detail": ""
        }
      },
      "profiles": [
        {
          "profile_uuid": "954ec5ea-a334-4825-87b3-937e7e381f24",
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

`GET /api/v1/fleet/device/:token`

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
        "browser": "",
        "generated_cpe": "",
        "vulnerabilities": null
      },
      {
        "id": 1146,
        "name": "tar",
        "version": "1.30",
        "source": "rpm_packages",
        "browser": "",
        "generated_cpe": "",
        "vulnerabilities": null
      },
      {
        "id": 321,
        "name": "SomeApp.app",
        "version": "1.0",
        "source": "apps",
        "browser": "",
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
    "percent_disk_space_available": 74,
    "gigs_total_disk_space": 160,
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
      "encryption_key_available": true,
      "enrollment_status": "On (manual)",
      "name": "Fleet",
      "connected_to_fleet": true,
      "server_url": "https://acme.com/mdm/apple/mdm",
      "macos_settings": {
        "disk_encryption": null,
        "action_required": null
      },
      "macos_setup": {
        "bootstrap_package_status": "installed",
        "detail": "",
        "bootstrap_package_name": "test.pkg"
      },
      "os_settings": {
        "disk_encryption": {
          "status": null,
          "detail": ""
        }
      },
      "profiles": [
        {
          "profile_uuid": "954ec5ea-a334-4825-87b3-937e7e381f24",
          "name": "profile1",
          "status": "verifying",
          "operation_type": "install",
          "detail": ""
        }
      ]
    }
  },
  "self_service": true,
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

`DELETE /api/v1/fleet/hosts/:id`

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

`POST /api/v1/fleet/hosts/:id/refetch`

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
| filters | object  | body | **Required** Contains any of the following four properties: `query` for search query keywords. Searchable fields include `hostname`, `hardware_serial`, `uuid`, and `ipv4`. `status` to indicate the status of the hosts to return. Can either be `new`, `online`, `offline`, `mia` or `missing`. `label_id` to indicate the selected label. `team_id` to indicate the selected team. Note: `label_id` and `status` cannot be used at the same time. |

#### Example

`POST /api/v1/fleet/hosts/transfer/filter`

##### Request body

```json
{
  "team_id": 1,
  "filters": {
    "status": "online",
    "team_id": 2,
  }
}
```

##### Default response

`Status: 200`


### Turn off MDM for a host

Turns off MDM for the specified macOS, iOS, or iPadOS host.

`DELETE /api/v1/fleet/hosts/:id/mdm`

#### Parameters

| Name | Type    | In   | Description                           |
| ---- | ------- | ---- | ------------------------------------- |
| id   | integer | path | **Required.** The host's ID in Fleet. |

#### Example

`DELETE /api/v1/fleet/hosts/42/mdm`

##### Default response

`Status: 204`


### Bulk delete hosts by filter or ids

`POST /api/v1/fleet/hosts/delete`

#### Parameters

| Name    | Type    | In   | Description                                                                                                                                                                                                                                                                                                                        |
| ------- | ------- | ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ids     | array   | body | A list of the host IDs you'd like to delete. If `ids` is specified, `filters` cannot be specified.                                                                                                                                                                                                                                                           |
| filters | object  | body | Contains any of the following four properties: `query` for search query keywords. Searchable fields include `hostname`, `hardware_serial`, `uuid`, and `ipv4`. `status` to indicate the status of the hosts to return. Can either be `new`, `online`, `offline`, `mia` or `missing`. `label_id` to indicate the selected label. `team_id` to indicate the selected team. If `filters` is specified, `id` cannot be specified. `label_id` and `status` cannot be used at the same time. |

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

Request (`filters` is specified and empty, to delete all hosts):
```json
{
  "filters": {}
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

### Get human-device mapping

Returns the end user's email(s) they use to log in to their Identity Provider (IdP) and Google Chrome profile.

Also returns the custom email that's set via the `PUT /api/v1/fleet/hosts/:id/device_mapping` endpoint (docs [here](#update-custom-human-device-mapping))

Note that IdP email is only supported on macOS hosts. It's collected once, during automatic enrollment (DEP), only if the end user authenticates with the IdP and the DEP profile has `await_device_configured` set to `true`.

`GET /api/v1/fleet/hosts/:id/device_mapping`

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
      "source": "identity_provider"
    },
    {
      "email": "user@example.com",
      "source": "google_chrome_profiles"
    },
    {
      "email": "user@example.com",
      "source": "custom"
    }
  ]
}
```

---

### Update custom human-device mapping

`PUT /api/v1/fleet/hosts/:id/device_mapping`

Updates the email for the `custom` data source in the human-device mapping. This source can only have one email.

#### Parameters

| Name       | Type              | In   | Description                                                                   |
| ---------- | ----------------- | ---- | ----------------------------------------------------------------------------- |
| id         | integer           | path | **Required**. The host's `id`.                                                |
| email      | string            | body | **Required**. The custom email.                                               |

#### Example

`PUT /api/v1/fleet/hosts/1/device_mapping`

##### Request body

```json
{
  "email": "user@example.com"
}
```

##### Default response

`Status: 200`

```json
{
  "host_id": 1,
  "device_mapping": [
    {
      "email": "user@example.com",
      "source": "identity_provider"
    },
    {
      "email": "user@example.com",
      "source": "google_chrome_profiles"
    },
    {
      "email": "user@example.com",
      "source": "custom"
    }
  ]
}
```

### Get host's device health report

Retrieves information about a single host's device health.

This report includes a subset of host vitals, and simplified policy and vulnerable software information. Data is cached to preserve performance. To get all up-to-date information about a host, use the "Get host" endpoint [here](#get-host).


`GET /api/v1/fleet/hosts/:id/health`

#### Parameters

| Name       | Type              | In   | Description                                                                   |
| ---------- | ----------------- | ---- | ----------------------------------------------------------------------------- |
| id         | integer           | path | **Required**. The host's `id`.                                                |

#### Example

`GET /api/v1/fleet/hosts/1/health`

##### Default response

`Status: 200`

```json
{
  "host_id": 1,
  "health": {
    "updated_at": "2023-09-16T18:52:19Z",
    "os_version": "CentOS Linux 8.3.2011",
    "disk_encryption_enabled": true,
    "failing_policies_count": 1,
    "failing_critical_policies_count": 1, // Fleet Premium only
    "failing_policies": [
      {
        "id": 123,
        "name": "Google Chrome is up to date",
        "critical": true, // Fleet Premium only
        "resolution": "Follow the Update Google Chrome instructions here: https://support.google.com/chrome/answer/95414?sjid=6534253818042437614-NA"
      }
    ],
    "vulnerable_software": [
      {
        "id": 321,
        "name": "Firefox.app",
        "version": "116.0.3",
      }
    ]
  }
}
```

---

### Get host's mobile device management (MDM) information

Currently supports Windows and MacOS. On MacOS this requires the [macadmins osquery
extension](https://github.com/macadmins/osquery-extension) which comes bundled
in [Fleet's agent (fleetd)](https://fleetdm.com/docs/get-started/anatomy#fleetd).

Retrieves a host's MDM enrollment status and MDM server URL.

If the host exists but is not enrolled to an MDM server, then this API returns `null`.

`GET /api/v1/fleet/hosts/:id/mdm`

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
in [Fleet's agent (fleetd)](https://fleetdm.com/docs/get-started/anatomy#fleetd).

Retrieves MDM enrollment summary. Windows servers are excluded from the aggregated data.

`GET /api/v1/fleet/hosts/summary/mdm`

#### Parameters

| Name     | Type    | In    | Description                                                                                                                                                                                                                                                                                                                        |
| -------- | ------- | ----- | -------------------------------------------------------------------------------- |
| team_id  | integer | query | _Available in Fleet Premium_. Filter by team                                      |
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

### Get host's mobile device management (MDM) and Munki information

Retrieves a host's MDM enrollment status, MDM server URL, and Munki version.

`GET /api/v1/fleet/hosts/:id/macadmins`

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
in [Fleet's agent (fleetd)](https://fleetdm.com/docs/get-started/anatomy#fleetd).
Currently supported only on macOS.


Retrieves aggregated host's MDM enrollment status and Munki versions.

`GET /api/v1/fleet/macadmins`

#### Parameters

| Name    | Type    | In    | Description                                                                                                                                                                                                                                                                                                                        |
| ------- | ------- | ----- | ---------------------------------------------------------------------------------------------------------------- |
| team_id | integer | query | _Available in Fleet Premium_. Filters the aggregate host information to only include hosts in the specified team. |                           |

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

### Resend host's configuration profile

Resends a configuration profile for the specified host.

`POST /api/v1/fleet/hosts/:id/configuration_profiles/:profile_uuid/resend`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| id   | integer | path | **Required.** The host's ID. |
| profile_uuid   | string | path | **Required.** The UUID of the configuration profile to resend to the host. |

#### Example

`POST /api/v1/fleet/hosts/233/configuration_profiles/fc14a20-84a2-42d8-9257-a425f62bb54d/resend`

##### Default response

`Status: 202`

### Get host's scripts

`GET /api/v1/fleet/hosts/:id/scripts`

#### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The host's id. |
| page | integer | query | Page number of the results to fetch.|
| per_page | integer | query | Results per page.|

#### Example

`GET /api/v1/fleet/hosts/123/scripts`

##### Default response

`Status: 200`

```json
"scripts": [
  {
    "script_id": 3,
    "name": "remove-zoom-artifacts.sh",
    "last_execution": {
      "execution_id": "e797d6c6-3aae-11ee-be56-0242ac120002",
      "executed_at": "2021-12-15T15:23:57Z",
      "status": "error"
    }
  },
  {
    "script_id": 5,
    "name": "set-timezone.sh",
    "last_execution": {
      "id": "e797d6c6-3aae-11ee-be56-0242ac120002",
      "executed_at": "2021-12-15T15:23:57Z",
      "status": "pending"
    }
  },
  {
    "script_id": 8,
    "name": "uninstall-zoom.sh",
    "last_execution": {
      "id": "e797d6c6-3aae-11ee-be56-0242ac120002",
      "executed_at": "2021-12-15T15:23:57Z",
      "status": "ran"
    }
  }
],
"meta": {
  "has_next_results": false,
  "has_previous_results": false
}

```

### Get host's software

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

`GET /api/v1/fleet/hosts/:id/software`

#### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The host's ID. |
| query   | string | query | Search query keywords. Searchable fields include `name`. |
| available_for_install | boolean | query | If `true` or `1`, only list software that is available for install (added by the user). Default is `false`. |
| page | integer | query | Page number of the results to fetch.|
| per_page | integer | query | Results per page.|

#### Example

`GET /api/v1/fleet/hosts/123/software`

##### Default response

`Status: 200`

```json
{
  "count": 3,
  "software": [
    {
      "id": 121,
      "name": "Google Chrome.app",
      "software_package": {
        "name": "GoogleChrome.pkg",
        "version": "125.12.0.3",
        "self_service": true,
        "last_install": {
          "install_uuid": "8bbb8ac2-b254-4387-8cba-4d8a0407368b",
          "installed_at": "2024-05-15T15:23:57Z"
        }
      },
      "app_store_app": null,
      "source": "apps",
      "status": "failed_install",
      "installed_versions": [
        {
          "version": "121.0",
          "bundle_identifier": "com.google.Chrome",
          "last_opened_at": "2024-04-01T23:03:07Z",
          "vulnerabilities": ["CVE-2023-1234","CVE-2023-4321","CVE-2023-7654"],
          "installed_paths": ["/Applications/Google Chrome.app"],
          "signature_information": [
            {
              "installed_path": "/Applications/Google Chrome.app",
              "team_identifier": "EQHXZ8M8AV"
            }
          ]
        }
      ]
    },
    {
      "id": 134,
      "name": "Falcon.app",
      "software_package": {
        "name": "FalconSensor-6.44.pkg",
        "self_service": false,
        "last_install": null,
        "last_uninstall": {
          "script_execution_id": "ed579e73-0f41-46c8-aaf4-3c1e5880ed27",
          "uninstalled_at": "2024-05-15T15:23:57Z"
        }
      },
      "app_store_app": null,    
      "source": "",
      "status": "pending_uninstall",
      "installed_versions": [],
    },
    {
      "id": 147,
      "name": "Logic Pro",
      "software_package": null,
      "app_store_app": {
        "app_store_id": "1091189122",
        "icon_url": "https://is1-ssl.mzstatic.com/image/thumb/Purple221/v4/f4/25/1f/f4251f60-e27a-6f05-daa7-9f3a63aac929/AppIcon-0-0-85-220-0-0-4-0-0-2x-0-0-0-0-0.png/512x512bb.png",
        "version": "2.04",
        "self_service": false,
        "last_install": {
          "command_uuid": "0aa14ae5-58fe-491a-ac9a-e4ee2b3aac40",
          "installed_at": "2024-05-15T15:23:57Z"
        },
      },
      "source": "apps",
      "status": "installed",
      "installed_versions": [
        {
          "version": "118.0",
          "bundle_identifier": "com.apple.logic10",
          "last_opened_at": "2024-04-01T23:03:07Z",
          "vulnerabilities": ["CVE-2023-1234"],
          "installed_paths": ["/Applications/Logic Pro.app"],
          "signature_information": [
            {
              "installed_path": "/Applications/Logic Pro.app",
              "team_identifier": ""
            }
          ]
        }
      ]
    },
  ],
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
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
| order_direction         | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include 'asc' and 'desc'. Default is 'asc'.                                                                                                                                                                                                               |
| status                  | string  | query | Indicates the status of the hosts to return. Can either be 'new', 'online', 'offline', 'mia' or 'missing'.                                                                                                                                                                                                                                  |
| query                   | string  | query | Search query keywords. Searchable fields include `hostname`, `hardware_serial`, `uuid`, `ipv4` and the hosts' email addresses (only searched if the query looks like an email address, i.e. contains an `@`, no space, etc.).                                                                                                               |
| team_id                 | integer | query | _Available in Fleet Premium_. Filters the hosts to only include hosts in the specified team.                                                                                                                                                                                                                                                |
| policy_id               | integer | query | The ID of the policy to filter hosts by.                                                                                                                                                                                                                                                                                                    |
| policy_response         | string  | query | **Requires `policy_id`**. Valid options are 'passing' or 'failing'. **Note: If `policy_id` is specified _without_ including `policy_response`, this will also return hosts where the policy is not configured to run or failed to run.** |
| software_version_id     | integer | query | The ID of the software version to filter hosts by.                                                                                                            |
| software_title_id       | integer | query | The ID of the software title to filter hosts by.                                                                                                              |
| os_version_id | integer | query | The ID of the operating system version to filter hosts by. |
| os_name                 | string  | query | The name of the operating system to filter hosts by. `os_version` must also be specified with `os_name`                                                                                                                                                                                                                                     |
| os_version              | string  | query | The version of the operating system to filter hosts by. `os_name` must also be specified with `os_version`                                                                                                                                                                                                                                  |
| vulnerability           | string  | query | The cve to filter hosts by (including "cve-" prefix, case-insensitive).                                                                                                                                                                                                                                                                     |
| mdm_id                  | integer | query | The ID of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider and URL).                                                                                                                                                                                                |
| mdm_name                | string  | query | The name of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider).                                                                                                                                                                                                      |
| mdm_enrollment_status   | string  | query | The _mobile device management_ (MDM) enrollment status to filter hosts by. Valid options are 'manual', 'automatic', 'enrolled', 'pending', or 'unenrolled'.                                                                                                                                                                                 |
| macos_settings          | string  | query | Filters the hosts by the status of the _mobile device management_ (MDM) profiles applied to hosts. Valid options are 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.**                                                                                                                                                                                                             |
| munki_issue_id          | integer | query | The ID of the _munki issue_ (a Munki-reported error or warning message) to filter hosts by (that is, filter hosts that are affected by that corresponding error or warning message).                                                                                                                                                        |
| low_disk_space          | integer | query | _Available in Fleet Premium_. Filters the hosts to only include hosts with less GB of disk space available than this value. Must be a number between 1-100.                                                                                                                                                                                 |
| label_id                | integer | query | A valid label ID. Can only be used in combination with `order_key`, `order_direction`, `status`, `query` and `team_id`.                                                                                                                                                                                                                     |
| bootstrap_package       | string | query | _Available in Fleet Premium_. Filters the hosts by the status of the MDM bootstrap package on the host. Valid options are 'installed', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |
| disable_failing_policies | boolean | query | If `true`, hosts will return failing policies as 0 (returned as the `issues` column) regardless of whether there are any that failed for the host. This is meant to be used when increased performance is needed in exchange for the extra information.      |

If `mdm_id`, `mdm_name` or `mdm_enrollment_status` is specified, then Windows Servers are excluded from the results.

#### Example

`GET /api/v1/fleet/hosts/report?software_id=123&format=csv&columns=hostname,primary_ip,platform`

##### Default response

`Status: 200`

```csv
created_at,updated_at,id,detail_updated_at,label_updated_at,policy_updated_at,last_enrolled_at,seen_time,refetch_requested,hostname,uuid,platform,osquery_version,os_version,build,platform_like,code_name,uptime,memory,cpu_type,cpu_subtype,cpu_brand,cpu_physical_cores,cpu_logical_cores,hardware_vendor,hardware_model,hardware_version,hardware_serial,computer_name,primary_ip_id,primary_ip,primary_mac,distributed_interval,config_tls_refresh,logger_tls_period,team_id,team_name,gigs_disk_space_available,percent_disk_space_available,gigs_total_disk_space,issues,device_mapping,status,display_text
2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,1,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,false,foo.local0,a4fc55a1-b5de-409c-a2f4-441f564680d3,debian,,,,,,0s,0,,,,0,0,,,,,,,,,0,0,0,,,0,0,0,0,,,,
2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:22:56Z,false,foo.local1,689539e5-72f0-4bf7-9cc5-1530d3814660,rhel,,,,,,0s,0,,,,0,0,,,,,,,,,0,0,0,,,0,0,0,0,,,,
2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,3,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:23:56Z,2022-03-15T17:21:56Z,false,foo.local2,48ebe4b0-39c3-4a74-a67f-308f7b5dd171,linux,,,,,,0s,0,,,,0,0,,,,,,,,,0,0,0,,,0,0,0,0,,,,
```

### Get host's disk encryption key

Retrieves the disk encryption key for a host.

The host will only return a key if its disk encryption status is "Verified." Get hosts' disk encryption statuses using the [List hosts endpoint](#list-hosts) and `os_settings_disk_encryption` parameter.

`GET /api/v1/fleet/hosts/:id/encryption_key`

#### Parameters

| Name | Type    | In   | Description                                                        |
| ---- | ------- | ---- | ------------------------------------------------------------------ |
| id   | integer | path | **Required** The id of the host to get the disk encryption key for. |


#### Example

`GET /api/v1/fleet/hosts/8/encryption_key`

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

`GET /api/v1/fleet/hosts/:id/configuration_profiles`

#### Parameters

| Name | Type    | In   | Description                      |
| ---- | ------- | ---- | -------------------------------- |
| id   | integer | path | **Required**. The ID of the host  |


#### Example

`GET /api/v1/fleet/hosts/8/configuration_profiles`

##### Default response

`Status: 200`

```json
{
  "host_id": 8,
  "profiles": [
    {
      "profile_uuid": "bc84dae7-396c-4e10-9d45-5768bce8b8bd",
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

### Lock host

_Available in Fleet Premium_

Sends a command to lock the specified macOS, Linux, or Windows host. The host is locked once it comes online.

To lock a macOS host, the host must have MDM turned on. To lock a Windows or Linux host, the host must have [scripts enabled](https://fleetdm.com/docs/using-fleet/scripts).


`POST /api/v1/fleet/hosts/:id/lock`

#### Parameters

| Name       | Type              | In   | Description                                                                   |
| ---------- | ----------------- | ---- | ----------------------------------------------------------------------------- |
| id | integer | path | **Required**. ID of the host to be locked. |
| view_pin | boolean | query | For macOS hosts, whether to return the unlock PIN. |

#### Example

`POST /api/v1/fleet/hosts/123/lock`

##### Default response

`Status: 204`


#### Example

`POST /api/v1/fleet/hosts/123/lock?view_pin=true`

##### Default response (macOS hosts)

`Status: 200`

```json
{
  "unlock_pin": "123456"
}
```

### Unlock host

_Available in Fleet Premium_

Sends a command to unlock the specified Windows or Linux host, or retrieves the unlock PIN for a macOS host.

To unlock a Windows or Linux host, the host must have [scripts enabled](https://fleetdm.com/docs/using-fleet/scripts).

`POST /api/v1/fleet/hosts/:id/unlock`

#### Parameters

| Name       | Type              | In   | Description                                                                   |
| ---------- | ----------------- | ---- | ----------------------------------------------------------------------------- |
| id | integer | path | **Required**. ID of the host to be unlocked. |

#### Example

`POST /api/v1/fleet/hosts/:id/unlock`

##### Default response (Windows or Linux hosts)

`Status: 204`


##### Default response (macOS hosts)

`Status: 200`

```json
{
  "host_id": 8,
  "unlock_pin": "123456"
}
```

### Wipe host

Sends a command to wipe the specified macOS, iOS, iPadOS, Windows, or Linux host. The host is wiped once it comes online.

To wipe a macOS, iOS, iPadOS, or Windows host, the host must have MDM turned on. To lock a Linux host, the host must have [scripts enabled](https://fleetdm.com/docs/using-fleet/scripts).

`POST /api/v1/fleet/hosts/:id/wipe`

#### Parameters

| Name       | Type              | In   | Description                                                                   |
| ---------- | ----------------- | ---- | ----------------------------------------------------------------------------- |
| id | integer | path | **Required**. ID of the host to be wiped. |

#### Example

`POST /api/v1/fleet/hosts/123/wipe`

##### Default response

`Status: 204`


### Get host's past activity

`GET /api/v1/fleet/hosts/:id/activities`

#### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The host's ID. |
| page | integer | query | Page number of the results to fetch.|
| per_page | integer | query | Results per page.|

#### Example

`GET /api/v1/fleet/hosts/12/activities`

##### Default response

`Status: 200`

```json
{
  "activities": [
    {
      "created_at": "2023-07-27T14:35:08Z",
      "actor_id": 1,
      "actor_full_name": "Anna Chao",
      "id": 4,
      "actor_gravatar": "",
      "actor_email": "",
      "type": "uninstalled_software",
      "details": {
        "host_id": 1,
        "host_display_name": "Marko’s MacBook Pro",
        "software_title": "Adobe Acrobat.app",
        "script_execution_id": "ecf22dba-07dc-40a9-b122-5480e948b756",
        "status": "failed_uninstall"
      }
    }, 
    {
      "created_at": "2023-07-27T14:35:08Z",
      "actor_id": 1,
      "actor_full_name": "Anna Chao",
      "id": 3,
      "actor_gravatar": "",
      "actor_email": "",
      "type": "uninstalled_software",
      "details": {
        "host_id": 1,
        "host_display_name": "Marko’s MacBook Pro",
        "software_title": "Adobe Acrobat.app",
        "script_execution_id": "ecf22dba-07dc-40a9-b122-5480e948b756",
        "status": "uninstalled"
      }
    },
    {
      "created_at": "2023-07-27T14:35:08Z",
      "id": 2,
      "actor_full_name": "Anna",
      "actor_id": 1,
      "actor_gravatar": "",
      "actor_email": "anna@example.com",
      "type": "ran_script",
      "details": {
        "host_id": 1,
        "host_display_name": "Steve's MacBook Pro",
        "script_name": "set-timezones.sh",
        "script_execution_id": "d6cffa75-b5b5-41ef-9230-15073c8a88cf",
        "async": true
      },
    },
    {
      "created_at": "2021-07-27T13:25:21Z",
      "id": 1,
      "actor_full_name": "Bob",
      "actor_id": 2,
      "actor_gravatar": "",
      "actor_email": "bob@example.com",
      "type": "ran_script",
      "details": {
        "host_id": 1,
        "host_display_name": "Steve's MacBook Pro",
        "script_name": "",
        "script_execution_id": "y3cffa75-b5b5-41ef-9230-15073c8a88cf",
        "async": false
      },
    },
  ],
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
}
```

### Get host's upcoming activity

`GET /api/v1/fleet/hosts/:id/activities/upcoming`

#### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The host's id. |
| page | integer | query | Page number of the results to fetch.|
| per_page | integer | query | Results per page.|

#### Example

`GET /api/v1/fleet/hosts/12/activities/upcoming`

##### Default response

`Status: 200`

```json
{
  "count": 3,
  "activities": [
    {
      "created_at": "2023-07-27T14:35:08Z",
      "actor_id": 1,
      "actor_full_name": "Anna Chao",
      "uuid": "cc081637-fdf9-4d44-929f-96dfaec00f67",
      "actor_gravatar": "",
      "actor_email": "",
      "type": "uninstalled_software",
      "fleet_initiated_activity": false,
      "details": {
        "host_id": 1,
        "host_display_name": "Marko's MacBook Pro",
        "software_title": "Adobe Acrobat.app",
        "script_execution_id": "ecf22dba-07dc-40a9-b122-5480e948b756",
        "status": "pending_uninstall",
      }
    },
    {
      "created_at": "2023-07-27T14:35:08Z",
      "uuid": "d6cffa75-b5b5-41ef-9230-15073c8a88cf",
      "actor_full_name": "Marko",
      "actor_id": 1,
      "actor_gravatar": "",
      "actor_email": "marko@example.com",
      "type": "ran_script",
      "details": {
        "host_id": 1,
        "host_display_name": "Steve's MacBook Pro",
        "script_name": "set-timezones.sh",
        "script_execution_id": "d6cffa75-b5b5-41ef-9230-15073c8a88cf",
        "async": true
      },
    },
    {
      "created_at": "2021-07-27T13:25:21Z",
      "uuid": "y3cffa75-b5b5-41ef-9230-15073c8a88cf",
      "actor_full_name": "Rachael",
      "actor_id": 1,
      "actor_gravatar": "",
      "actor_email": "rachael@example.com",
      "type": "ran_script",
      "details": {
        "host_id": 1,
        "host_display_name": "Steve's MacBook Pro",
        "script_name": "",
        "script_execution_id": "y3cffa75-b5b5-41ef-9230-15073c8a88cf",
        "async": false
      },
    },
  ],
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
}
```

### Add labels to host

Adds manual labels to a host.

`POST /api/v1/fleet/hosts/:id/labels`

#### Parameters

| Name   | Type    | In   | Description                  |
| ------ | ------- | ---- | ---------------------------- |
| labels | array   | body | The list of label names to add to the host. |


#### Example

`POST /api/v1/fleet/hosts/12/labels`

##### Request body

```json
{
  "labels": ["label1", "label2"]
}
```

##### Default response

`Status: 200`

### Remove labels from host

Removes manual labels from a host.

`DELETE /api/v1/fleet/hosts/:id/labels`

#### Parameters

| Name   | Type    | In   | Description                  |
| ------ | ------- | ---- | ---------------------------- |
| labels | array   | body | The list of label names to delete from the host. |


#### Example

`DELETE /api/v1/fleet/hosts/12/labels`

##### Request body

```json
{
  "labels": ["label3", "label4"]
}
```

##### Default response

`Status: 200`

### Live query one host (ad-hoc)

Runs an ad-hoc live query against the specified host and responds with the results.

The live query will stop if the targeted host is offline, or if the query times out. Timeouts happen if the host hasn't responded after the configured `FLEET_LIVE_QUERY_REST_PERIOD` (default 25 seconds) or if the `distributed_interval` agent option (default 10 seconds) is higher than the `FLEET_LIVE_QUERY_REST_PERIOD`.


`POST /api/v1/fleet/hosts/:id/query`

#### Parameters

| Name      | Type  | In   | Description                                                                                                                                                        |
|-----------|-------|------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| id        | integer  | path | **Required**. The target host ID. |
| query     | string   | body | **Required**. The query SQL. |


#### Example

`POST /api/v1/fleet/hosts/123/query`

##### Request body

```json
{
  "query": "SELECT model, vendor FROM usb_devices;"
}
```

##### Default response

`Status: 200`

```json
{
  "host_id": 123,
  "query": "SELECT model, vendor FROM usb_devices;",
  "status": "online", // "online" or "offline"
  "error": null,
  "rows": [
    {
      "model": "USB2.0 Hub",
      "vendor": "VIA Labs, Inc."
    }
  ]
}
```

Note that if the host is online and the query times out, this endpoint will return an error and `rows` will be `null`. If the host is offline, no error will be returned, and `rows` will be`null`.

### Live query host by identifier (ad-hoc)

Runs an ad-hoc live query against a host identified using `uuid` and responds with the results.

The live query will stop if the targeted host is offline, or if the query times out. Timeouts happen if the host hasn't responded after the configured `FLEET_LIVE_QUERY_REST_PERIOD` (default 25 seconds) or if the `distributed_interval` agent option (default 10 seconds) is higher than the `FLEET_LIVE_QUERY_REST_PERIOD`.


`POST /api/v1/fleet/hosts/identifier/:identifier/query`

#### Parameters

| Name      | Type  | In   | Description                                                                                                                                                        |
|-----------|-------|------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| identifier       | integer or string   | path | **Required**. The host's `hardware_serial`, `uuid`, `osquery_host_id`, `hostname`, or `node_key`. |
| query            | string   | body | **Required**. The query SQL. |


#### Example

`POST /api/v1/fleet/hosts/identifier/392547dc-0000-0000-a87a-d701ff75bc65/query`

##### Request body

```json
{
  "query": "SELECT model, vendor FROM usb_devices;"
}
```

##### Default response

`Status: 200`

```json
{
  "host_id": 123,
  "query": "SELECT model, vendor FROM usb_devices;",
  "status": "online", // "online" or "offline"
  "error": null,
  "rows": [
    {
      "model": "USB2.0 Hub",
      "vendor": "VIA Labs, Inc."
    }
  ]
}
```

Note that if the host is online and the query times out, this endpoint will return an error and `rows` will be `null`. If the host is offline, no error will be returned, and `rows` will be `null`.

---


## Labels

- [Add label](#add-label)
- [Update label](#update-label)
- [Get label](#get-label)
- [Get labels summary](#get-labels-summary)
- [List labels](#list-labels)
- [List hosts in a label](#list-hosts-in-a-label)
- [Delete label](#delete-label)
- [Delete label by ID](#delete-label-by-id)

### Add label

Add a dynamic or manual label.

`POST /api/v1/fleet/labels`

#### Parameters

| Name        | Type   | In   | Description                                                                                                                                                                                                                                  |
| ----------- | ------ | ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| name        | string | body | **Required**. The label's name.                                                                                                                                                                                                              |
| description | string | body | The label's description.                                                                                                                                                                                                                     |
| query       | string | body | The query in SQL syntax used to filter the hosts. Only one of either `query` (to create a dynamic label) or `hosts` (to create a manual label) can be included in the request.  |
| hosts       | array | body | The list of host identifiers (`hardware_serial`, `uuid`, `osquery_host_id`, `hostname`, or `name`) the label will apply to. Only one of either `query` (to create a dynamic label) or `hosts` (to create a manual label)  can be included in the request. |
| platform    | string | body | The specific platform for the label to target. Provides an additional filter. Choices for platform are `darwin`, `windows`, `ubuntu`, and `centos`. All platforms are included by default and this option is represented by an empty string. |

If both `query` and `hosts` aren't specified, a manual label with no hosts will be created.

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

### Update label

Updates the specified label. Note: Label queries and platforms are immutable. To change these, you must delete the label and create a new label.

`PATCH /api/v1/fleet/labels/:id`

#### Parameters

| Name        | Type    | In   | Description                   |
| ----------- | ------- | ---- | ----------------------------- |
| id          | integer | path | **Required**. The label's id. |
| name        | string  | body | The label's name.             |
| description | string  | body | The label's description.      |
| hosts       | array   | body | If updating a manual label: the list of host identifiers (`hardware_serial`, `uuid`, `osquery_host_id`, `hostname`, or `name`) the label will apply to. |


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

`GET /api/v1/fleet/labels/:id`

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

`GET /api/v1/fleet/labels/:id/hosts`

#### Parameters

| Name                     | Type    | In    | Description                                                                                                                                                                                                                |
| ---------------          | ------- | ----- | -----------------------------------------------------------------------------------------------------------------------------                                                                                              |
| id                       | integer | path  | **Required**. The label's id.                                                                                                                                                                                              |
| page                     | integer | query | Page number of the results to fetch.                                                                                                                                                                                       |
| per_page                 | integer | query | Results per page.                                                                                                                                                                                                          |
| order_key                | string  | query | What to order results by. Can be any column in the hosts table.                                                                                                                                                            |
| order_direction          | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include 'asc' and 'desc'. Default is 'asc'.                                                                                              |
| after                    | string  | query | The value to get results after. This needs `order_key` defined, as that's the column that would be used.                                                                                                                   |
| status                   | string  | query | Indicates the status of the hosts to return. Can either be 'new', 'online', 'offline', 'mia' or 'missing'.                                                                                                                 |
| query                    | string  | query | Search query keywords. Searchable fields include `hostname`, `hardware_serial`, `uuid`, and `ipv4`.                                                                                                                         |
| team_id                  | integer | query | _Available in Fleet Premium_. Filters the hosts to only include hosts in the specified team.                                                                                                                                |
| disable_failing_policies | boolean | query | If "true", hosts will return failing policies as 0 regardless of whether there are any that failed for the host. This is meant to be used when increased performance is needed in exchange for the extra information.      |
| mdm_id                   | integer | query | The ID of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider and URL).      |
| mdm_name                 | string  | query | The name of the _mobile device management_ (MDM) solution to filter hosts by (that is, filter hosts that use a specific MDM provider).      |
| mdm_enrollment_status    | string  | query | The _mobile device management_ (MDM) enrollment status to filter hosts by. Valid options are 'manual', 'automatic', 'enrolled', 'pending', or 'unenrolled'.                                                                                                                                                                                                             |
| macos_settings           | string  | query | Filters the hosts by the status of the _mobile device management_ (MDM) profiles applied to hosts. Valid options are 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.**                                                                                                                                                                                                             |
| low_disk_space           | integer | query | _Available in Fleet Premium_. Filters the hosts to only include hosts with less GB of disk space available than this value. Must be a number between 1-100.                                                                 |
| macos_settings_disk_encryption | string | query | Filters the hosts by disk encryption status. Valid options are 'verified', 'verifying', 'action_required', 'enforcing', 'failed', or 'removing_enforcement'. |
| bootstrap_package       | string | query | _Available in Fleet Premium_. Filters the hosts by the status of the MDM bootstrap package on the host. Valid options are 'installed', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |
| os_settings          | string  | query | Filters the hosts by the status of the operating system settings applied to the hosts. Valid options are 'verified', 'verifying', 'pending', or 'failed'. **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |
| os_settings_disk_encryption | string | query | Filters the hosts by disk encryption status. Valid options are 'verified', 'verifying', 'action_required', 'enforcing', 'failed', or 'removing_enforcement'.  **Note: If this filter is used in Fleet Premium without a team ID filter, the results include only hosts that are not assigned to any team.** |

If `mdm_id`, `mdm_name`, `mdm_enrollment_status`, `os_settings`, or `os_settings_disk_encryption` is specified, then Windows Servers are excluded from the results.

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

`DELETE /api/v1/fleet/labels/:name`

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

`DELETE /api/v1/fleet/labels/id/:id`

#### Parameters

| Name | Type    | In   | Description                   |
| ---- | ------- | ---- | ----------------------------- |
| id   | integer | path | **Required**. The label's id. |

#### Example

`DELETE /api/v1/fleet/labels/id/13`

##### Default response

`Status: 200`


---

## OS settings

- [Add custom OS setting (configuration profile)](#add-custom-os-setting-configuration-profile)
- [List custom OS settings (configuration profiles)](#list-custom-os-settings-configuration-profiles)
- [Get or download custom OS setting (configuration profile)](#get-or-download-custom-os-setting-configuration-profile)
- [Delete custom OS setting (configuration profile)](#delete-custom-os-setting-configuration-profile)
- [Update disk encryption enforcement](#update-disk-encryption-enforcement)
- [Get disk encryption statistics](#get-disk-encryption-statistics)
- [Get OS settings status](#get-os-settings-status)


### Add custom OS setting (configuration profile)

> [Add custom macOS setting](https://github.com/fleetdm/fleet/blob/fleet-v4.40.0/docs/REST%20API/rest-api.md#add-custom-macos-setting-configuration-profile) (`POST /api/v1/fleet/mdm/apple/profiles`) API endpoint is deprecated as of Fleet 4.41. It is maintained for backwards compatibility. Please use the below API endpoint instead.

Add a configuration profile to enforce custom settings on macOS and Windows hosts.

`POST /api/v1/fleet/configuration_profiles`

#### Parameters

| Name                      | Type     | In   | Description                                                                                                   |
| ------------------------- | -------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| profile                   | file     | form | **Required.** The .mobileconfig and JSON for macOS or XML for Windows file containing the profile. |
| team_id                   | string   | form | _Available in Fleet Premium_. The team ID for the profile. If specified, the profile is applied to only hosts that are assigned to the specified team. If not specified, the profile is applied to only to hosts that are not assigned to any team. |
| labels_include_all        | array     | form | _Available in Fleet Premium_. Profile will only be applied to hosts that have all of these labels. Only one of either `labels_include_all`, `labels_include_any` or `labels_exclude_any` can be included in the request. |
| labels_include_any        | array     | form | _Available in Fleet Premium_. Profile will only be applied to hosts that have any of these labels. Only one of either `labels_include_all`, `labels_include_any` or `labels_exclude_any` can be included in the request. |
| labels_exclude_any | array | form | _Available in Fleet Premium_. Profile will be applied to hosts that don’t have any of these labels. Only one of either `labels_include_all`, `labels_include_any` or `labels_exclude_any` can be included in the request. |

#### Example

Add a new configuration profile to be applied to macOS hosts
assigned to a team. Note that in this example the form data specifies`team_id` in addition to
`profile`.

`POST /api/v1/fleet/configuration_profiles`

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
Content-Disposition: form-data; name="labels_include_all"

Label name 1
--------------------------f02md47480und42y
Content-Disposition: form-data; name="labels_include_all"

Label name 2
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
  "profile_uuid": "954ec5ea-a334-4825-87b3-937e7e381f24"
}
```

###### Additional notes
If the response is `Status: 409 Conflict`, the body may include additional error details in the case
of duplicate payload display name or duplicate payload identifier (macOS profiles).


### List custom OS settings (configuration profiles)

> [List custom macOS settings](https://github.com/fleetdm/fleet/blob/fleet-v4.40.0/docs/REST%20API/rest-api.md#list-custom-macos-settings-configuration-profiles) (`GET /api/v1/fleet/mdm/apple/profiles`) API endpoint is deprecated as of Fleet 4.41. It is maintained for backwards compatibility. Please use the below API endpoint instead.

Get a list of the configuration profiles in Fleet.

For Fleet Premium, the list can
optionally be filtered by team ID. If no team ID is specified, team profiles are excluded from the
results (i.e., only profiles that are associated with "No team" are listed).

`GET /api/v1/fleet/configuration_profiles`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| team_id                   | string | query | _Available in Fleet Premium_. The team id to filter profiles.              |
| page                      | integer | query | Page number of the results to fetch.                                     |
| per_page                  | integer | query | Results per page.                                                        |

#### Example

List all configuration profiles for macOS and Windows hosts enrolled to Fleet's MDM that are not assigned to any team.

`GET /api/v1/fleet/configuration_profiles`

##### Default response

`Status: 200`

```json
{
  "profiles": [
    {
      "profile_uuid": "39f6cbbc-fe7b-4adc-b7a9-542d1af89c63",
      "team_id": 0,
      "name": "Example macOS profile",
      "platform": "darwin",
      "identifier": "com.example.profile",
      "created_at": "2023-03-31T00:00:00Z",
      "updated_at": "2023-03-31T00:00:00Z",
      "checksum": "dGVzdAo=",
      "labels_exclude_any": [
       {
        "name": "Label name 1",
        "id": 1
       }
      ]
    },
    {
      "profile_uuid": "f5ad01cc-f416-4b5f-88f3-a26da3b56a19",
      "team_id": 0,
      "name": "Example Windows profile",
      "platform": "windows",
      "created_at": "2023-04-31T00:00:00Z",
      "updated_at": "2023-04-31T00:00:00Z",
      "checksum": "aCLemVr)",
      "labels_include_all": [
        {
          "name": "Label name 2",
          "broken": true,
        },
        {
          "name": "Label name 3",
          "id": 3
        }
      ]
    }
  ],
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
}
```

If one or more assigned labels are deleted the profile is considered broken (`broken: true`). It won’t be applied to new hosts.

### Get or download custom OS setting (configuration profile)

> [Download custom macOS setting](https://github.com/fleetdm/fleet/blob/fleet-v4.40.0/docs/REST%20API/rest-api.md#download-custom-macos-setting-configuration-profile) (`GET /api/v1/fleet/mdm/apple/profiles/:profile_id`) API endpoint is deprecated as of Fleet 4.41. It is maintained for backwards compatibility. Please use the API endpoint below instead.

`GET /api/v1/fleet/configuration_profiles/:profile_uuid`

#### Parameters

| Name                      | Type    | In    | Description                                             |
| ------------------------- | ------- | ----- | ------------------------------------------------------- |
| profile_uuid              | string | url   | **Required** The UUID of the profile to download.  |
| alt                       | string  | query | If specified and set to "media", downloads the profile. |

#### Example (get a profile metadata)

`GET /api/v1/fleet/configuration_profiles/f663713f-04ee-40f0-a95a-7af428c351a9`

##### Default response

`Status: 200`

```json
{
  "profile_uuid": "f663713f-04ee-40f0-a95a-7af428c351a9",
  "team_id": 0,
  "name": "Example profile",
  "platform": "darwin",
  "identifier": "com.example.profile",
  "created_at": "2023-03-31T00:00:00Z",
  "updated_at": "2023-03-31T00:00:00Z",
  "checksum": "dGVzdAo=",
  "labels_include_all": [
    {
      "name": "Label name 1",
      "id": 1,
      "broken": true
    },
    {
      "name": "Label name 2",
      "id": 2
    }
  ]
}
```

#### Example (download a profile)

`GET /api/v1/fleet/configuration_profiles/f663713f-04ee-40f0-a95a-7af428c351a9?alt=media`

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

### Delete custom OS setting (configuration profile)

> [Delete custom macOS setting](https://github.com/fleetdm/fleet/blob/fleet-v4.40.0/docs/REST%20API/rest-api.md#delete-custom-macos-setting-configuration-profile) (`DELETE /api/v1/fleet/mdm/apple/profiles/:profile_id`) API endpoint is deprecated as of Fleet 4.41. It is maintained for backwards compatibility. Please use the below API endpoint instead.

`DELETE /api/v1/fleet/configuration_profiles/:profile_uuid`

#### Parameters

| Name                      | Type    | In    | Description                                                               |
| ------------------------- | ------- | ----- | ------------------------------------------------------------------------- |
| profile_uuid              | string  | url   | **Required** The UUID of the profile to delete. |

#### Example

`DELETE /api/v1/fleet/configuration_profiles/f663713f-04ee-40f0-a95a-7af428c351a9`

##### Default response

`Status: 200`


### Update disk encryption enforcement

> `PATCH /api/v1/fleet/mdm/apple/settings` API endpoint is deprecated as of Fleet 4.45. It is maintained for backward compatibility. Please use the new API endpoint below. See old API endpoint docs [here](https://github.com/fleetdm/fleet/blob/main/docs/REST%20API/rest-api.md?plain=1#L4296C29-L4296C29).

_Available in Fleet Premium_

`POST /api/v1/fleet/disk_encryption`

#### Parameters

| Name                   | Type    | In    | Description                                                                                 |
| -------------          | ------  | ----  | --------------------------------------------------------------------------------------      |
| team_id                | integer | body  | The team ID to apply the settings to. Settings applied to hosts in no team if absent.       |
| enable_disk_encryption | boolean | body  | Whether disk encryption should be enforced on devices that belong to the team (or no team). |

#### Example

`POST /api/v1/fleet/disk_encryption`

##### Default response

`204`


### Get disk encryption statistics

_Available in Fleet Premium_

Get aggregate status counts of disk encryption enforced on macOS and Windows hosts.

The summary can optionally be filtered by team ID.

`GET /api/v1/fleet/disk_encryption`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| team_id                   | string | query | _Available in Fleet Premium_. The team ID to filter the summary.           |

#### Example

`GET /api/v1/fleet/disk_encryption`

##### Default response

`Status: 200`

```json
{
  "verified": {"macos": 123, "windows": 123, "linux": 13},
  "verifying": {"macos": 123, "windows": 0, "linux": 0},
  "action_required": {"macos": 123, "windows": 0, "linux": 37},
  "enforcing": {"macos": 123, "windows": 123, "linux": 0},
  "failed": {"macos": 123, "windows": 123, "linux": 0},
  "removing_enforcement": {"macos": 123, "windows": 0, "linux": 0}
}
```


### Get OS settings status

> [Get macOS settings statistics](https://github.com/fleetdm/fleet/blob/fleet-v4.40.0/docs/REST%20API/rest-api.md#get-macos-settings-statistics) (`GET /api/v1/fleet/mdm/apple/profiles/summary`) API endpoint is deprecated as of Fleet 4.41. It is maintained for backwards compatibility. Please use the below API endpoint instead.

Get aggregate status counts of all OS settings (configuration profiles and disk encryption) enforced on hosts.

For Fleet Premium users, the counts can
optionally be filtered by `team_id`. If no `team_id` is specified, team profiles are excluded from the results (i.e., only profiles that are associated with "No team" are listed).

`GET /api/v1/fleet/configuration_profiles/summary`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| team_id                   | string | query | _Available in Fleet Premium_. The team ID to filter profiles.              |

#### Example

Get aggregate status counts of profiles for to macOS and Windows hosts that are assigned to "No team".

`GET /api/v1/fleet/configuration_profiles/summary`

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

---

## Setup experience

- [Set custom MDM setup enrollment profile](#set-custom-mdm-setup-enrollment-profile)
- [Get custom MDM setup enrollment profile](#get-custom-mdm-setup-enrollment-profile)
- [Delete custom MDM setup enrollment profile](#delete-custom-mdm-setup-enrollment-profile)
- [Get Over-the-Air (OTA) enrollment profile](#get-over-the-air-ota-enrollment-profile) 
- [Get manual enrollment profile](#get-manual-enrollment-profile)
- [Upload a bootstrap package](#upload-a-bootstrap-package)
- [Get metadata about a bootstrap package](#get-metadata-about-a-bootstrap-package)
- [Delete a bootstrap package](#delete-a-bootstrap-package)
- [Download a bootstrap package](#download-a-bootstrap-package)
- [Get a summary of bootstrap package status](#get-a-summary-of-bootstrap-package-status)
- [Configure setup experience](#configure-setup-experience)
- [Upload an EULA file](#upload-an-eula-file)
- [Get metadata about an EULA file](#get-metadata-about-an-eula-file)
- [Delete an EULA file](#delete-an-eula-file)
- [Download an EULA file](#download-an-eula-file)
- [List software (setup experience)](#list-software-setup-experience)
- [Update software (setup experience)](#update-software-setup-experience)
- [Add script (setup experience)](#add-script-setup-experience)
- [Get or download script (setup experience)](#get-or-download-script-setup-experience)
- [Delete script (setup experience)](#delete-script-setup-experience)



### Set custom MDM setup enrollment profile

_Available in Fleet Premium_

Sets the custom MDM setup enrollment profile for a team or no team.

`POST /api/v1/fleet/enrollment_profiles/automatic`

#### Parameters

| Name                      | Type    | In    | Description                                                                   |
| ------------------------- | ------  | ----- | -------------------------------------------------------------------------     |
| team_id                   | integer | json  | The team ID this custom enrollment profile applies to, or no team if omitted. |
| name                      | string  | json  | The filename of the uploaded custom enrollment profile.                       |
| enrollment_profile        | object  | json  | The custom enrollment profile's json, as documented in https://developer.apple.com/documentation/devicemanagement/profile. |

#### Example

`POST /api/v1/fleet/enrollment_profiles/automatic`

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

> NOTE: The `ConfigurationWebURL` and `URL` values in the custom MDM setup enrollment profile are automatically populated. Attempting to populate them with custom values may generate server response errors.

### Get custom MDM setup enrollment profile

_Available in Fleet Premium_

Gets the custom MDM setup enrollment profile for a team or no team.

`GET /api/v1/fleet/enrollment_profiles/automatic`

#### Parameters

| Name                      | Type    | In    | Description                                                                           |
| ------------------------- | ------  | ----- | -------------------------------------------------------------------------             |
| team_id                   | integer | query | The team ID for which to return the custom enrollment profile, or no team if omitted. |

#### Example

`GET /api/v1/fleet/enrollment_profiles/automatic?team_id=123`

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

`DELETE /api/v1/fleet/enrollment_profiles/automatic`

#### Parameters

| Name                      | Type    | In    | Description                                                                           |
| ------------------------- | ------  | ----- | -------------------------------------------------------------------------             |
| team_id                   | integer | query | The team ID for which to delete the custom enrollment profile, or no team if omitted. |

#### Example

`DELETE /api/v1/fleet/enrollment_profiles/automatic?team_id=123`

##### Default response

`Status: 204`


### Get Over-the-Air (OTA) enrollment profile

`GET /api/v1/fleet/enrollment_profiles/ota`

The returned value is a signed `.mobileconfig` OTA enrollment profile. Install this profile on macOS, iOS, or iPadOS hosts to enroll them to a specific team in Fleet and turn on MDM features.

To enroll macOS hosts, turn on MDM features, and add [human-device mapping](#get-human-device-mapping), install the [manual enrollment profile](#get-manual-enrollment-profile) instead.

Learn more about OTA profiles [here](https://developer.apple.com/library/archive/documentation/NetworkingInternet/Conceptual/iPhoneOTAConfiguration/OTASecurity/OTASecurity.html).

#### Parameters

| Name              | Type    | In    | Description                                                                      |
|-------------------|---------|-------|----------------------------------------------------------------------------------|
| enroll_secret     | string  | query | **Required**. The enroll secret of the team this host will be assigned to.       |

#### Example

`GET /api/v1/fleet/enrollment_profiles/ota?enroll_secret=foobar`

##### Default response

`Status: 200`

> **Note:** To confirm success, it is important for clients to match content length with the response header (this is done automatically by most clients, including the browser) rather than relying solely on the response status code returned by this endpoint.

##### Example response headers

```http
  Content-Length: 542
  Content-Type: application/x-apple-aspen-config; charset=utf-8
  Content-Disposition: attachment;filename="fleet-mdm-enrollment-profile.mobileconfig"
  X-Content-Type-Options: nosniff
```

###### Example response body

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple Inc//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>PayloadContent</key>
    <dict>
      <key>URL</key>
      <string>https://foo.example.com/api/fleet/ota_enrollment?enroll_secret=foobar</string>
      <key>DeviceAttributes</key>
      <array>
        <string>UDID</string>
        <string>VERSION</string>
        <string>PRODUCT</string>
	      <string>SERIAL</string>
      </array>
    </dict>
    <key>PayloadOrganization</key>
    <string>Acme Inc.</string>
    <key>PayloadDisplayName</key>
    <string>Acme Inc. enrollment</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
    <key>PayloadUUID</key>
    <string>fdb376e5-b5bb-4d8c-829e-e90865f990c9</string>
    <key>PayloadIdentifier</key>
    <string>com.fleetdm.fleet.mdm.apple.ota</string>
    <key>PayloadType</key>
    <string>Profile Service</string>
  </dict>
</plist>
```


### Get manual enrollment profile

Retrieves an unsigned manual enrollment profile for macOS hosts. Install this profile on macOS hosts to turn on MDM features manually.

To add [human-device mapping](#get-human-device-mapping), add the end user's email to the enrollment profle. Learn how [here](https://fleetdm.com/guides/config-less-fleetd-agent-deployment#basic-article).

`GET /api/v1/fleet/enrollment_profiles/manual`

##### Example

`GET /api/v1/fleet/enrollment_profiles/manual`

##### Default response

`Status: 200`

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<!-- ... -->
</plist>
```

### Upload a bootstrap package

_Available in Fleet Premium_

Upload a bootstrap package that will be automatically installed during DEP setup.

`POST /api/v1/fleet/bootstrap`

#### Parameters

| Name    | Type   | In   | Description                                                                                                                                                                                                            |
| ------- | ------ | ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| package | file   | form | **Required**. The bootstrap package installer. It must be a signed `pkg` file.                                                                                                                                         |
| team_id | string | form | The team ID for the package. If specified, the package will be installed to hosts that are assigned to the specified team. If not specified, the package will be installed to hosts that are not assigned to any team. |

#### Example

Upload a bootstrap package that will be installed to macOS hosts enrolled to MDM that are
assigned to a team. Note that in this example the form data specifies `team_id` in addition to
`package`.

`POST /api/v1/fleet/bootstrap`

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

`GET /api/v1/fleet/bootstrap/:team_id/metadata`

#### Parameters

| Name       | Type    | In    | Description                                                                                                                                                                                                        |
| -------    | ------  | ---   | ---------------------------------------------------------------------------------------------------------------------------------------------------------                                                          |
| team_id    | string  | url   | **Required** The team ID for the package. Zero (0) can be specified to get information about the bootstrap package for hosts that don't belong to a team.                                                          |
| for_update | boolean | query | If set to `true`, the authorization will be for a `write` action instead of a `read`. Useful for the write-only `gitops` role when requesting the bootstrap metadata to check if the package needs to be replaced. |

#### Example

`GET /api/v1/fleet/bootstrap/0/metadata`

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

`DELETE /api/v1/fleet/bootstrap/:team_id`

#### Parameters

| Name    | Type   | In  | Description                                                                                                                                               |
| ------- | ------ | --- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| team_id | string | url | **Required** The team ID for the package. Zero (0) can be specified to get information about the bootstrap package for hosts that don't belong to a team. |


#### Example

`DELETE /api/v1/fleet/bootstrap/1`

##### Default response

`Status: 200`


### Download a bootstrap package

_Available in Fleet Premium_

Download a bootstrap package.

`GET /api/v1/fleet/bootstrap`

#### Parameters

| Name  | Type   | In    | Description                                      |
| ----- | ------ | ----- | ------------------------------------------------ |
| token | string | query | **Required** The token of the bootstrap package. |

#### Example

`GET /api/v1/fleet/bootstrap?token=AA598E2A-7952-46E3-B89D-526D45F7E233`

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

The summary can optionally be filtered by team ID.

`GET /api/v1/fleet/bootstrap/summary`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| team_id                   | string | query | The team ID to filter the summary.                                        |

#### Example

`GET /api/v1/fleet/bootstrap/summary`

##### Default response

`Status: 200`

```json
{
  "installed": 10,
  "failed": 1,
  "pending": 4
}
```

### Configure setup experience

_Available in Fleet Premium_

`PATCH /api/v1/fleet/setup_experience`

#### Parameters

| Name                           | Type    | In    | Description                                                                                 |
| -------------          | ------  | ----  | --------------------------------------------------------------------------------------      |
| team_id                        | integer | body  | The team ID to apply the settings to. Settings applied to hosts in no team if absent.       |
| enable_end_user_authentication | boolean | body  | When enabled, require end users to authenticate with your identity provider (IdP) when they set up their new macOS hosts. |
| enable_release_device_manually | boolean | body  | When enabled, you're responsible for sending the DeviceConfigured command.|

#### Example

`PATCH /api/v1/fleet/setup_experience`

##### Request body

```json
{
  "team_id": 1,
  "enable_end_user_authentication": true,
  "enable_release_device_manually": true
}
```

##### Default response

`Status: 204`


### Upload an EULA file

_Available in Fleet Premium_

Upload an EULA that will be shown during the DEP flow.

`POST /api/v1/fleet/setup_experience/eula`

#### Parameters

| Name | Type | In   | Description                                       |
| ---- | ---- | ---- | ------------------------------------------------- |
| eula | file | form | **Required**. A PDF document containing the EULA. |

#### Example

`POST /api/v1/fleet/setup_experience/eula`

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

`GET /api/v1/fleet/setup_experience/eula/metadata`

#### Example

`GET /api/v1/fleet/setup_experience/eula/metadata`

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

`DELETE /api/v1/fleet/setup_experience/eula/:token`

#### Parameters

| Name  | Type   | In    | Description                              |
| ----- | ------ | ----- | ---------------------------------------- |
| token | string | path  | **Required** The token of the EULA file. |

#### Example

`DELETE /api/v1/fleet/setup_experience/eula/AA598E2A-7952-46E3-B89D-526D45F7E233`

##### Default response

`Status: 200`


### Download an EULA file

_Available in Fleet Premium_

Download an EULA file

`GET /api/v1/fleet/setup_experience/eula/:token`

#### Parameters

| Name  | Type   | In    | Description                              |
| ----- | ------ | ----- | ---------------------------------------- |
| token | string | path  | **Required** The token of the EULA file. |

#### Example

`GET /api/v1/fleet/setup_experience/eula/AA598E2A-7952-46E3-B89D-526D45F7E233`

##### Default response

`Status: 200`

```http
Status: 200
Content-Type: application/pdf
Content-Disposition: attachment
Content-Length: <length>
Body: <blob>
```

### List software (setup experience)

_Available in Fleet Premium_

List software that can or will be automatically installed during macOS setup. If `install_during_setup` is `true` it will be installed during setup.

`GET /api/v1/fleet/setup_experience/software`

| Name  | Type   | In    | Description                              |
| ----- | ------ | ----- | ---------------------------------------- |
| team_id | integer | query | _Available in Fleet Premium_. The ID of the team to filter software by. If not specified, it will filter only software that's available to hosts with no team. |
| page | integer | query | Page number of the results to fetch. |
| per_page | integer | query | Results per page. |


#### Example

`GET /api/v1/fleet/setup_experience/software?team_id=3`

##### Default response

`Status: 200`

```json
{
  "software_titles": [
    {
      "id": 12,
      "name": "Firefox.app",
      "software_package": {
        "name": "FirefoxInsall.pkg",
        "version": "125.6",
        "self_service": true,
        "install_during_setup": true
      },
      "app_store_app": null,
      "versions_count": 3,
      "source": "apps",
      "browser": "",
      "hosts_count": 48,
      "versions": [
        {
          "id": 123,
          "version": "1.12",
          "vulnerabilities": ["CVE-2023-1234","CVE-2023-4321","CVE-2023-7654"]
        },
        {
          "id": 124,
          "version": "3.4",
          "vulnerabilities": ["CVE-2023-1234","CVE-2023-4321","CVE-2023-7654"]
        },
        {
          "id": 12
          "version": "1.13",
          "vulnerabilities": ["CVE-2023-1234","CVE-2023-4321","CVE-2023-7654"]
        }
      ]
    }
  ],
  {
    "count": 2,
    "counts_updated_at": "2024-10-04T10:00:00Z",
    "meta": {
      "has_next_results": false,
      "has_previous_results": false
    }
  },
}
```

### Update software (setup experience)

_Available in Fleet Premium_

Set software that will be automatically installed during macOS setup. Software that isn't included in the request will be unset.

`PUT /api/v1/fleet/setup_experience/software`

| Name  | Type   | In    | Description                              |
| ----- | ------ | ----- | ---------------------------------------- |
| team_id | integer | query | _Available in Fleet Premium_. The ID of the team to set the software for. If not specified, it will set the software for hosts with no team. |
| software_title_ids | array | body | The ID of software titles to install during macOS setup. |

#### Example

`PUT /api/v1/fleet/setup_experience/software?team_id=3`

##### Default response

`Status: 200`

```json
{
  "software_title_ids": [23,3411,5032]
}
```

### Add script (setup experience)

_Available in Fleet Premium_

Add a script that will automatically run during macOS setup.

`POST /api/v1/fleet/setup_experience/script`

| Name  | Type   | In    | Description                              |
| ----- | ------ | ----- | ---------------------------------------- |
| team_id | integer | form | _Available in Fleet Premium_. The ID of the team to add the script to. If not specified, a script will be added for hosts with no team. |
| script | file | form | The ID of software titles to install during macOS setup. |

#### Example

`POST /api/v1/fleet/setup_experience/script`

##### Default response

`Status: 200`

##### Request headers

```http
Content-Length: 306
Content-Type: multipart/form-data; boundary=------------------------f02md47480und42y
```

##### Request body

```http
--------------------------f02md47480und42y
Content-Disposition: form-data; name="team_id"

1
--------------------------f02md47480und42y
Content-Disposition: form-data; name="script"; filename="myscript.sh"
Content-Type: application/octet-stream

echo "hello"
--------------------------f02md47480und42y--

```

### Get or download script (setup experience)

_Available in Fleet Premium_

Get a script that will automatically run during macOS setup.

`GET /api/v1/fleet/setup_experience/script`

| Name  | Type   | In    | Description                              |
| ----- | ------ | ----- | ---------------------------------------- |
| team_id | integer | query | _Available in Fleet Premium_. The ID of the team to get the script for. If not specified, script will be returned for hosts with no team. |
| alt  | string | query | If specified and set to "media", downloads the script's contents. |


#### Example (get script)

`GET /api/v1/fleet/setup_experience/script?team_id=3`

##### Default response

`Status: 200`

```json
{
  "id": 1,
  "team_id": 3,
  "name": "setup-experience-script.sh",
  "created_at": "2023-07-30T13:41:07Z",
  "updated_at": "2023-07-30T13:41:07Z"
}
```

#### Example (download script)

`GET /api/v1/fleet/setup_experience/script?team_id=3?alt=media`

##### Example response headers

```http
Content-Length: 13
Content-Type: application/octet-stream
Content-Disposition: attachment;filename="2023-09-27 script_1.sh"
```

###### Example response body

`Status: 200`

```
echo "hello"
```

### Delete script (setup experience)

_Available in Fleet Premium_

Delete a script that will automatically run during macOS setup.

`DELETE /api/v1/fleet/setup_experience/script`

| Name  | Type   | In    | Description                              |
| ----- | ------ | ----- | ---------------------------------------- |
| team_id | integer | query | _Available in Fleet Premium_. The ID of the team to get the script for. If not specified, script will be returned for hosts with no team. |

#### Example

`DELETE /api/v1/fleet/setup_experience/script?team_id=3`

##### Default response

`Status: 200`

---

## Commands

- [Run MDM command](#run-mdm-command)
- [Get MDM command results](#get-mdm-command-results)
- [List MDM commands](#list-mdm-commands)


### Run MDM command

> `POST /api/v1/fleet/mdm/apple/enqueue` API endpoint is deprecated as of Fleet 4.40. It is maintained for backward compatibility. Please use the new API endpoint below. See old API endpoint docs [here](https://github.com/fleetdm/fleet/blob/fleet-v4.39.0/docs/REST%20API/rest-api.md#run-custom-mdm-command).

This endpoint tells Fleet to run a custom MDM command, on the targeted macOS or Windows hosts, the next time they come online.

`POST /api/v1/fleet/commands/run`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| command                   | string | json  | A Base64 encoded MDM command as described in [Apple's documentation](https://developer.apple.com/documentation/devicemanagement/commands_and_queries) or [Windows's documentation](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mdm/0353f3d6-dbe2-42b6-b8d5-50db9333bba4). Supported formats are standard and raw (unpadded). You can paste your Base64 code to the [online decoder](https://devpal.co/base64-decode/) to check if you're using the valid format. |
| host_uuids                | array  | json  | An array of host UUIDs enrolled in Fleet on which the command should run. |

Note that the `EraseDevice` and `DeviceLock` commands are _available in Fleet Premium_ only.

#### Example

`POST /api/v1/fleet/commands/run`

##### Default response

`Status: 200`

```json
{
  "command_uuid": "a2064cef-0000-1234-afb9-283e3c1d487e",
  "request_type": "ProfileList"
}
```


### Get MDM command results

> `GET /api/v1/fleet/mdm/apple/commandresults` API endpoint is deprecated as of Fleet 4.40. It is maintained for backward compatibility. Please use the new API endpoint below. See old API endpoint docs [here](https://github.com/fleetdm/fleet/blob/fleet-v4.39.0/docs/REST%20API/rest-api.md#get-custom-mdm-command-results).

This endpoint returns the results for a specific custom MDM command.

In the reponse, the possible `status` values for macOS, iOS, and iPadOS hosts are the following:

* Pending: the command has yet to run on the host. The host will run the command the next time it comes online.
* NotNow: the host responded with "NotNow" status via the MDM protocol: the host received the command, but couldn’t execute it. The host will try to run the command the next time it comes online.
* Acknowledged: the host responded with "Acknowledged" status via the MDM protocol: the host processed the command successfully.
* Error: the host responded with "Error" status via the MDM protocol: an error occurred. Run the `fleetctl get mdm-command-results --id=<insert-command-id` to view the error.
* CommandFormatError: the host responded with "CommandFormatError" status via the MDM protocol: a protocol error occurred, which can result from a malformed command. Run the `fleetctl get mdm-command-results --id=<insert-command-id` to view the error.

The possible `status` values for Windows hosts are documented in Microsoft's documentation [here](https://learn.microsoft.com/en-us/windows/client-management/oma-dm-protocol-support#syncml-response-status-codes).

`GET /api/v1/fleet/commands/results`

#### Parameters

| Name                      | Type   | In    | Description                                                               |
| ------------------------- | ------ | ----- | ------------------------------------------------------------------------- |
| command_uuid              | string | query | The unique identifier of the command.                                     |

#### Example

`GET /api/v1/fleet/commands/results?command_uuid=a2064cef-0000-1234-afb9-283e3c1d487e`

##### Default response

`Status: 200`

```json
{
  "results": [
    {
      "host_uuid": "145cafeb-87c7-4869-84d5-e4118a927746",
      "command_uuid": "a2064cef-0000-1234-afb9-283e3c1d487e",
      "status": "Acknowledged",
      "updated_at": "2023-04-04:00:00Z",
      "request_type": "ProfileList",
      "hostname": "mycomputer",
      "payload": "PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0iVVRGLTgiPz4NCjwhRE9DVFlQRSBwbGlzdCBQVUJMSUMgIi0vL0FwcGxlLy9EVEQgUExJU1QgMS4wLy9FTiIgImh0dHA6Ly93d3cuYXBwbGUuY29tL0RURHMvUHJvcGVydHlMaXN0LTEuMC5kdGQiPg0KPHBsaXN0IHZlcnNpb249IjEuMCI+DQo8ZGljdD4NCg0KCTxrZXk+UGF5bG9hZERlc2NyaXB0aW9uPC9rZXk+DQoJPHN0cmluZz5UaGlzIHByb2ZpbGUgY29uZmlndXJhdGlvbiBpcyBkZXNpZ25lZCB0byBhcHBseSB0aGUgQ0lTIEJlbmNobWFyayBmb3IgbWFjT1MgMTAuMTQgKHYyLjAuMCksIDEwLjE1ICh2Mi4wLjApLCAxMS4wICh2Mi4wLjApLCBhbmQgMTIuMCAodjEuMC4wKTwvc3RyaW5nPg0KCTxrZXk+UGF5bG9hZERpc3BsYXlOYW1lPC9rZXk+DQoJPHN0cmluZz5EaXNhYmxlIEJsdWV0b290aCBzaGFyaW5nPC9zdHJpbmc+DQoJPGtleT5QYXlsb2FkRW5hYmxlZDwva2V5Pg0KCTx0cnVlLz4NCgk8a2V5PlBheWxvYWRJZGVudGlmaWVyPC9rZXk+DQoJPHN0cmluZz5jaXMubWFjT1NCZW5jaG1hcmsuc2VjdGlvbjIuQmx1ZXRvb3RoU2hhcmluZzwvc3RyaW5nPg0KCTxrZXk+UGF5bG9hZFNjb3BlPC9rZXk+DQoJPHN0cmluZz5TeXN0ZW08L3N0cmluZz4NCgk8a2V5PlBheWxvYWRUeXBlPC9rZXk+DQoJPHN0cmluZz5Db25maWd1cmF0aW9uPC9zdHJpbmc+DQoJPGtleT5QYXlsb2FkVVVJRDwva2V5Pg0KCTxzdHJpbmc+NUNFQkQ3MTItMjhFQi00MzJCLTg0QzctQUEyOEE1QTM4M0Q4PC9zdHJpbmc+DQoJPGtleT5QYXlsb2FkVmVyc2lvbjwva2V5Pg0KCTxpbnRlZ2VyPjE8L2ludGVnZXI+DQogICAgPGtleT5QYXlsb2FkUmVtb3ZhbERpc2FsbG93ZWQ8L2tleT4NCiAgICA8dHJ1ZS8+DQoJPGtleT5QYXlsb2FkQ29udGVudDwva2V5Pg0KCTxhcnJheT4NCgkJPGRpY3Q+DQoJCQk8a2V5PlBheWxvYWRDb250ZW50PC9rZXk+DQoJCQk8ZGljdD4NCgkJCQk8a2V5PmNvbS5hcHBsZS5CbHVldG9vdGg8L2tleT4NCgkJCQk8ZGljdD4NCgkJCQkJPGtleT5Gb3JjZWQ8L2tleT4NCgkJCQkJPGFycmF5Pg0KCQkJCQkJPGRpY3Q+DQoJCQkJCQkJPGtleT5tY3hfcHJlZmVyZW5jZV9zZXR0aW5nczwva2V5Pg0KCQkJCQkJCTxkaWN0Pg0KCQkJCQkJCQk8a2V5PlByZWZLZXlTZXJ2aWNlc0VuYWJsZWQ8L2tleT4NCgkJCQkJCQkJPGZhbHNlLz4NCgkJCQkJCQk8L2RpY3Q+DQoJCQkJCQk8L2RpY3Q+DQoJCQkJCTwvYXJyYXk+DQoJCQkJPC9kaWN0Pg0KCQkJPC9kaWN0Pg0KCQkJPGtleT5QYXlsb2FkRGVzY3JpcHRpb248L2tleT4NCgkJCTxzdHJpbmc+RGlzYWJsZXMgQmx1ZXRvb3RoIFNoYXJpbmc8L3N0cmluZz4NCgkJCTxrZXk+UGF5bG9hZERpc3BsYXlOYW1lPC9rZXk+DQoJCQk8c3RyaW5nPkN1c3RvbTwvc3RyaW5nPg0KCQkJPGtleT5QYXlsb2FkRW5hYmxlZDwva2V5Pg0KCQkJPHRydWUvPg0KCQkJPGtleT5QYXlsb2FkSWRlbnRpZmllcjwva2V5Pg0KCQkJPHN0cmluZz4wMjQwREQxQy03MERDLTQ3NjYtOTAxOC0wNDMyMkJGRUVBRDE8L3N0cmluZz4NCgkJCTxrZXk+UGF5bG9hZFR5cGU8L2tleT4NCgkJCTxzdHJpbmc+Y29tLmFwcGxlLk1hbmFnZWRDbGllbnQucHJlZmVyZW5jZXM8L3N0cmluZz4NCgkJCTxrZXk+UGF5bG9hZFVVSUQ8L2tleT4NCgkJCTxzdHJpbmc+MDI0MEREMUMtNzBEQy00NzY2LTkwMTgtMDQzMjJCRkVFQUQxPC9zdHJpbmc+DQoJCQk8a2V5PlBheWxvYWRWZXJzaW9uPC9rZXk+DQoJCQk8aW50ZWdlcj4xPC9pbnRlZ2VyPg0KCQk8L2RpY3Q+DQoJPC9hcnJheT4NCjwvZGljdD4NCjwvcGxpc3Q+",
      "result": "PD94bWwgdmVyc2lvbj0iMS4wIiBlbmNvZGluZz0iVVRGLTgiPz4NCjwhRE9DVFlQRSBwbGlzdCBQVUJMSUMgIi0vL0FwcGxlLy9EVEQgUExJU1QgMS4wLy9FTiIgImh0dHA6Ly93d3cuYXBwbGUuY29tL0RURHMvUHJvcGVydHlMaXN0LTEuMC5kdGQiPg0KPHBsaXN0IHZlcnNpb249IjEuMCI+DQo8ZGljdD4NCiAgICA8a2V5PkNvbW1hbmRVVUlEPC9rZXk+DQogICAgPHN0cmluZz4wMDAxX0luc3RhbGxQcm9maWxlPC9zdHJpbmc+DQogICAgPGtleT5TdGF0dXM8L2tleT4NCiAgICA8c3RyaW5nPkFja25vd2xlZGdlZDwvc3RyaW5nPg0KICAgIDxrZXk+VURJRDwva2V5Pg0KICAgIDxzdHJpbmc+MDAwMDgwMjAtMDAwOTE1MDgzQzgwMDEyRTwvc3RyaW5nPg0KPC9kaWN0Pg0KPC9wbGlzdD4="
    }
  ]
}
```

> Note: If the server has not yet received a result for a command, it will return an empty object (`{}`).

### List MDM commands

> `GET /api/v1/fleet/mdm/apple/commands` API endpoint is deprecated as of Fleet 4.40. It is maintained for backward compatibility. Please use the new API endpoint below. See old API endpoint docs [here](https://github.com/fleetdm/fleet/blob/fleet-v4.39.0/docs/REST%20API/rest-api.md#list-custom-mdm-commands).

This endpoint returns the list of custom MDM commands that have been executed.

`GET /api/v1/fleet/commands`

#### Parameters

| Name                      | Type    | In    | Description                                                               |
| ------------------------- | ------  | ----- | ------------------------------------------------------------------------- |
| page                      | integer | query | Page number of the results to fetch.                                      |
| per_page                  | integer | query | Results per page.                                                         |
| order_key                 | string  | query | What to order results by. Can be any field listed in the `results` array example below. |
| order_direction           | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |
| host_identifier           | string  | query | The host's `hostname`, `uuid`, or `hardware_serial`. |
| request_type              | string  | query | The request type to filter commands by. |

#### Example

`GET /api/v1/fleet/commands?per_page=5`

##### Default response

`Status: 200`

```json
{
  "results": [
    {
      "host_uuid": "145cafeb-87c7-4869-84d5-e4118a927746",
      "command_uuid": "a2064cef-0000-1234-afb9-283e3c1d487e",
      "status": "Acknowledged",
      "updated_at": "2023-04-04:00:00Z",
      "request_type": "ProfileList",
      "hostname": "mycomputer"
    },
    {
      "host_uuid": "322vghee-12c7-8976-83a1-e2118a927342",
      "command_uuid": "d76d69b7-d806-45a9-8e49-9d6dc533485c",
      "status": "200",
      "updated_at": "2023-05-04:00:00Z",
      "request_type": "./Device/Vendor/MSFT/Reboot/RebootNow",
      "hostname": "myhost"
    }
  ]
}
```

---

## Integrations

- [Get Apple Push Notification service (APNs)](#get-apple-push-notification-service-apns)
- [List Apple Business Manager (ABM) tokens](#list-apple-business-manager-abm-tokens)
- [List Volume Purchasing Program (VPP) tokens](#list-volume-purchasing-program-vpp-tokens)

### Get Apple Push Notification service (APNs)

`GET /api/v1/fleet/apns`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/apns`

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

### List Apple Business Manager (ABM) tokens

_Available in Fleet Premium_

`GET /api/v1/fleet/abm_tokens`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/abm_tokens`

##### Default response

`Status: 200`

```json
"abm_tokens": [
  {
    "id": 1,
    "apple_id": "apple@example.com",
    "org_name": "Fleet Device Management Inc.",
    "mdm_server_url": "https://example.com/mdm/apple/mdm",
    "renew_date": "2023-11-29T00:00:00Z",
    "terms_expired": false,
    "macos_team": {
      "name": "💻 Workstations",
      "id": 1
    },
    "ios_team": {
      "name": "📱🏢 Company-owned iPhones",
      "id": 2
    },
    "ipados_team": {
      "name": "🔳🏢 Company-owned iPads",
      "id": 3
    }
  }
]
```

### List Volume Purchasing Program (VPP) tokens

_Available in Fleet Premium_

`GET /api/v1/fleet/vpp_tokens`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/vpp_tokens`

##### Default response

`Status: 200`

```json
"vpp_tokens": [
  {
    "id": 1,
    "org_name": "Fleet Device Management Inc.",
    "location": "https://example.com/mdm/apple/mdm",
    "renew_date": "2023-11-29T00:00:00Z",
    "teams": [
      {
        "name": "💻 Workstations",
        "id": 1
      },
      {
        "name": "💻🐣 Workstations (canary)",
        "id": 2
      },
      {
        "name": "📱🏢 Company-owned iPhones",
        "id": 3
      },
      {
        "name": "🔳🏢 Company-owned iPads",
        "id": 4
      }
    ],
  }
]
```

### Get Volume Purchasing Program (VPP)


> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

_Available in Fleet Premium_

`GET /api/v1/fleet/vpp`

#### Example

`GET /api/v1/fleet/vpp`

##### Default response

`Status: 200`

```json
{
  "org_name": "Acme Inc.",
  "renew_date": "2023-11-29T00:00:00Z",
  "location": "Acme Inc. Main Address"
}
```

---

## Policies

- [List policies](#list-policies)
- [Count policies](#count-policies)
- [Get policy by ID](#get-policy-by-id)
- [Add policy](#add-policy)
- [Remove policies](#remove-policies)
- [Edit policy](#edit-policy)
- [Reset automations for all hosts failing policies](#reset-automations-for-all-hosts-failing-policies)

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
      "failing_host_count": 300,
      "host_count_updated_at": "2023-12-20T15:23:57Z"
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
      "failing_host_count": 0,
      "host_count_updated_at": "2023-12-20T15:23:57Z"
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

`GET /api/v1/fleet/global/policies/:id`

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
      "failing_host_count": 300,
      "host_count_updated_at": "2023-12-20T15:23:57Z"
    }
}
```

### Add policy

`POST /api/v1/fleet/global/policies`

#### Parameters

| Name        | Type    | In   | Description                          |
| ----------  | ------- | ---- | ------------------------------------ |
| name        | string  | body | The policy's name.                    |
| query       | string  | body | The policy's query in SQL.                    |
| description | string  | body | The policy's description.             |
| resolution  | string  | body | The resolution steps for the policy. |
| platform    | string  | body | Comma-separated target platforms, currently supported values are "windows", "linux", "darwin". The default, an empty string means target all platforms. |
| critical    | boolean | body | _Available in Fleet Premium_. Mark policy as critical/high impact. |

#### Example (preferred)

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
    "failing_host_count": 0,
    "host_count_updated_at": null
  }
}
```

### Remove policies

`POST /api/v1/fleet/global/policies/delete`

#### Parameters

| Name     | Type    | In   | Description                                       |
| -------- | ------- | ---- | ------------------------------------------------- |
| ids      | array   | body | **Required.** The IDs of the policies to delete.  |

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

`PATCH /api/v1/fleet/global/policies/:id`

#### Parameters

| Name        | Type    | In   | Description                          |
| ----------  | ------- | ---- | ------------------------------------ |
| id          | integer | path | The policy's ID.                     |
| name        | string  | body | The query's name.                    |
| query       | string  | body | The query in SQL.                    |
| description | string  | body | The query's description.             |
| resolution  | string  | body | The resolution steps for the policy. |
| platform    | string  | body | Comma-separated target platforms, currently supported values are "windows", "linux", "darwin". The default, an empty string means target all platforms. |
| critical    | boolean | body | _Available in Fleet Premium_. Mark policy as critical/high impact. |

#### Example

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
    "failing_host_count": 0,
    "host_count_updated_at": null
  }
}
```

### Reset automations for all hosts failing policies

Resets [automation](https://fleetdm.com/docs/using-fleet/automations#policy-automations) status for *all* hosts failing the specified policies. On the next automation run, any failing host will be considered newly failing.

`POST /api/v1/fleet/automations/reset`

#### Parameters

| Name        | Type     | In   | Description                                              |
| ----------  | -------- | ---- | -------------------------------------------------------- |
| policy_ids  | array    | body | Filters to only run policy automations for the specified policies. |
| team_ids    | array    | body | _Available in Fleet Premium_. Filters to only run policy automations for hosts in the specified teams. |


#### Example

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

## Team policies

- [List team policies](#list-team-policies)
- [Count team policies](#count-team-policies)
- [Get team policy by ID](#get-team-policy-by-id)
- [Add team policy](#add-team-policy)
- [Remove team policies](#remove-team-policies)
- [Edit team policy](#edit-team-policy)

_Available in Fleet Premium_

Team policies work the same as policies, but at the team level.

### List team policies

`GET /api/v1/fleet/teams/:id/policies`

#### Parameters

| Name               | Type    | In   | Description                                                                                                   |
| ------------------ | ------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| id                 | integer | path  | **Required.** Defines what team ID to operate on                                                                            |
| merge_inherited  | boolean | query | If `true`, will return both team policies **and** inherited ("All teams") policies the `policies` list, and will not return a separate `inherited_policies` list. |
| query                 | string | query | Search query keywords. Searchable fields include `name`. |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                                                                                                                                                                                        |
| per_page                | integer | query | Results per page. |


#### Example (default usage)

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
      "failing_host_count": 300,
      "host_count_updated_at": "2023-12-20T15:23:57Z",
      "calendar_events_enabled": true
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
      "failing_host_count": 0,
      "host_count_updated_at": "2023-12-20T15:23:57Z",
      "calendar_events_enabled": false,
      "run_script": {
        "name": "Encrypt Windows disk with BitLocker",
        "id": 234
      }
    },
    {
      "id": 3,
      "name": "macOS - install/update Adobe Acrobat",
      "query": "SELECT 1 FROM apps WHERE name = \"Adobe Acrobat.app\" AND bundle_short_version != \"24.002.21005\";",
      "description": "Checks if the hard disk is encrypted on Windows devices",
      "critical": false,
      "author_id": 43,
      "author_name": "Alice",
      "author_email": "alice@example.com",
      "team_id": 1,
      "resolution": "Resolution steps",
      "platform": "darwin",
      "created_at": "2021-12-16T14:37:37Z",
      "updated_at": "2021-12-16T16:39:00Z",
      "passing_host_count": 2300,
      "failing_host_count": 3,
      "host_count_updated_at": "2023-12-20T15:23:57Z",
      "calendar_events_enabled": false,
      "install_software": {
        "name": "Adobe Acrobat.app",
        "software_title_id": 1234
      }
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
      "failing_host_count": 9,
      "host_count_updated_at": "2023-12-20T15:23:57Z"
    }
  ]
}
```

#### Example (returns single list)

`GET /api/v1/fleet/teams/1/policies?merge_inherited=true`

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
      "failing_host_count": 300,
      "host_count_updated_at": "2023-12-20T15:23:57Z"
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
      "failing_host_count": 0,
      "host_count_updated_at": "2023-12-20T15:23:57Z"
    },
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
      "failing_host_count": 9,
      "host_count_updated_at": "2023-12-20T15:23:57Z"
    }
  ]
}
```

### Count team policies

`GET /api/v1/fleet/team/:team_id/policies/count`

#### Parameters
| Name               | Type    | In   | Description                                                                                                   |
| ------------------ | ------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| team_id                 | integer | path  | **Required.** Defines what team ID to operate on
| query                 | string | query | Search query keywords. Searchable fields include `name`. |
| merge_inherited     | boolean | query | If `true`, will include inherited ("All teams") policies in the count. |

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

`GET /api/v1/fleet/teams/:team_id/policies/:policy_id`

#### Parameters

| Name               | Type    | In   | Description                                                                                                   |
| ------------------ | ------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| team_id            | integer | path  | **Required.** Defines what team ID to operate on                                                                            |
| policy_id                 | integer | path | **Required.** The policy's ID.                                                                                |

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
    "failing_host_count": 0,
    "host_count_updated_at": null,
    "calendar_events_enabled": true,
    "install_software": {
      "name": "Adobe Acrobat.app",
      "software_title_id": 1234
    },
    "run_script": {
      "name": "Enable gatekeeper",
      "id": 1337
    }
  }
}
```

### Add team policy

> **Experimental feature**. Software related features (like install software policy automation) are undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

The semantics for creating a team policy are the same as for global policies, see [Add policy](#add-policy).

`POST /api/v1/fleet/teams/:id/policies`

#### Parameters

| Name              | Type    | In   | Description                                                                                                                                            |
|-------------------| ------- | ---- |--------------------------------------------------------------------------------------------------------------------------------------------------------|
| id                | integer | path | Defines what team ID to operate on.                                                                                                                    |
| name              | string  | body | The policy's name.                                                                                                                                     |
| query             | string  | body | The policy's query in SQL.                                                                                                                             |
| description       | string  | body | The policy's description.                                                                                                                              |
| resolution        | string  | body | The resolution steps for the policy.                                                                                                                   |
| platform          | string  | body | Comma-separated target platforms, currently supported values are "windows", "linux", "darwin". The default, an empty string means target all platforms. |
| critical          | boolean | body | _Available in Fleet Premium_. Mark policy as critical/high impact.                                                                                     |
| software_title_id | integer | body | _Available in Fleet Premium_. ID of software title to install if the policy fails.                                                                     |
| script_id         | integer | body | _Available in Fleet Premium_. ID of script to run if the policy fails.                                                                 |

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
    "failing_host_count": 0,
    "host_count_updated_at": null,
    "calendar_events_enabled": false,
    "install_software": {
      "name": "Adobe Acrobat.app",
      "software_title_id": 1234
    },
    "run_script": {
      "name": "Enable gatekeeper",
      "id": 1337
    }
  }
}
```

### Remove team policies

`POST /api/v1/fleet/teams/:team_id/policies/delete`

#### Parameters

| Name     | Type    | In   | Description                                       |
| -------- | ------- | ---- | ------------------------------------------------- |
| team_id  | integer | path  | **Required.** Defines what team ID to operate on                |
| ids      | array   | body | **Required.** The IDs of the policies to delete.  |

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

> **Experimental feature**. Software related features (like install software policy automation) are undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

`PATCH /api/v1/fleet/teams/:team_id/policies/:policy_id`

#### Parameters

| Name                    | Type    | In   | Description                                                                                                                                             |
|-------------------------| ------- | ---- |---------------------------------------------------------------------------------------------------------------------------------------------------------|
| team_id                 | integer | path | The team's ID.                                                                                                                                          |
| policy_id               | integer | path | The policy's ID.                                                                                                                                        |
| name                    | string  | body | The query's name.                                                                                                                                       |
| query                   | string  | body | The query in SQL.                                                                                                                                       |
| description             | string  | body | The query's description.                                                                                                                                |
| resolution              | string  | body | The resolution steps for the policy.                                                                                                                    |
| platform                | string  | body | Comma-separated target platforms, currently supported values are "windows", "linux", "darwin". The default, an empty string means target all platforms. |
| critical                | boolean | body | _Available in Fleet Premium_. Mark policy as critical/high impact.                                                                                      |
| calendar_events_enabled | boolean | body | _Available in Fleet Premium_. Whether to trigger calendar events when policy is failing.                                                                |
| software_title_id       | integer | body | _Available in Fleet Premium_. ID of software title to install if the policy fails. Set to `0` to remove the automation.                                   |
| script_id               | integer | body | _Available in Fleet Premium_. ID of script to run if the policy fails. Set to `0` to remove the automation.                                               |

#### Example

`PATCH /api/v1/fleet/teams/2/policies/42`

##### Request body

```json
{
  "name": "Gatekeeper enabled",
  "query": "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
  "description": "Checks if gatekeeper is enabled on macOS devices",
  "critical": true,
  "resolution": "Resolution steps",
  "platform": "darwin",
  "script_id": 1337
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
    "failing_host_count": 0,
    "host_count_updated_at": null,
    "calendar_events_enabled": true,
    "install_software": {
      "name": "Adobe Acrobat.app",
      "software_title_id": 1234
    },
    "run_script": {
      "name": "Enable gatekeeper",
      "id": 1337
    }
  }
}
```

---

## Queries

- [List queries](#list-queries)
- [Get query](#get-query)
- [Get query report](#get-query-report)
- [Get query report for one host](#get-query-report-for-one-host)
- [Create query](#create-query)
- [Modify query](#modify-query)
- [Delete query by name](#delete-query-by-name)
- [Delete query by ID](#delete-query-by-id)
- [Delete queries](#delete-queries)
- [Run live query](#run-live-query)



### List queries

Returns a list of global queries or team queries.

`GET /api/v1/fleet/queries`

#### Parameters

| Name            | Type    | In    | Description                                                                                                                   |
| --------------- | ------- | ----- | ----------------------------------------------------------------------------------------------------------------------------- |
| order_key       | string  | query | What to order results by. Can be any column in the queries table.                                                             |
| order_direction | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |
| team_id         | integer | query | _Available in Fleet Premium_. The ID of the parent team for the queries to be listed. When omitted, returns global queries.                  |
| query           | string  | query | Search query keywords. Searchable fields include `name`.                                                                      |
| merge_inherited | boolean | query | _Available in Fleet Premium_. If `true`, will include global queries in addition to team queries when filtering by `team_id`. (If no `team_id` is provided, this parameter is ignored.) |
| platform        | string  | query | Return queries that are scheduled to run on this platform. One of: `"macos"`, `"windows"`, `"linux"` (case-insensitive). (Since queries cannot be scheduled to run on `"chrome"` hosts, it's not a valid value here) |
| page                    | integer | query | Page number of the results to fetch. |
| per_page                | integer | query | Results per page. |

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
      "discard_data": false,
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
      "discard_data": true,
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
  ],
  "meta": {
    "has_next_results": true,
    "has_previous_results": false
  },
  "count": 200
}
```

### Get query

Returns the query specified by ID.

`GET /api/v1/fleet/queries/:id`

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
    "discard_data": false,
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

### Get query report

Returns the query report specified by ID.

`GET /api/v1/fleet/queries/:id/report`

#### Parameters

| Name      | Type    | In    | Description                                |
| --------- | ------- | ----- | ------------------------------------------ |
| id        | integer | path  | **Required**. The ID of the desired query. |

#### Example

`GET /api/v1/fleet/queries/31/report`

##### Default response

`Status: 200`

```json
{
  "query_id": 31,
  "report_clipped": false,
  "results": [
    {
      "host_id": 1,
      "host_name": "foo",
      "last_fetched": "2021-01-19T17:08:31Z",
      "columns": {
        "model": "USB 2.0 Hub",
        "vendor": "VIA Labs, Inc."
      }
    },
    {
      "host_id": 1,
      "host_name": "foo",
      "last_fetched": "2021-01-19T17:08:31Z",
      "columns": {
        "model": "USB Keyboard",
        "vendor": "VIA Labs, Inc."
      }
    },
    {
      "host_id": 2,
      "host_name": "bar",
      "last_fetched": "2021-01-19T17:20:00Z",
      "columns": {
        "model": "USB Reciever",
        "vendor": "Logitech"
      }
    },
    {
      "host_id": 2,
      "host_name": "bar",
      "last_fetched": "2021-01-19T17:20:00Z",
      "columns": {
        "model": "USB Reciever",
        "vendor": "Logitech"
      }
    },
    {
      "host_id": 2,
      "host_name": "bar",
      "last_fetched": "2021-01-19T17:20:00Z",
      "columns": {
        "model": "Display Audio",
        "vendor": "Apple Inc."
      }
    }
  ]
}
```

If a query has no results stored, then `results` will be an empty array:

```json
{
  "query_id": 32,
  "results": []
}
```

> Note: osquery scheduled queries do not return errors, so only non-error results are included in the report. If you suspect a query may be running into errors, you can use the [live query](#run-live-query) endpoint to get diagnostics.

### Get query report for one host

Returns a query report for a single host.

`GET /api/v1/fleet/hosts/:id/queries/:query_id`

#### Parameters

| Name      | Type    | In    | Description                                |
| --------- | ------- | ----- | ------------------------------------------ |
| id        | integer | path  | **Required**. The ID of the desired host.          |
| query_id  | integer | path  | **Required**. The ID of the desired query.         |

#### Example

`GET /api/v1/fleet/hosts/123/queries/31`

##### Default response

`Status: 200`

```json
{
  "query_id": 31,
  "host_id": 1,
  "host_name": "foo",
  "last_fetched": "2021-01-19T17:08:31Z",
  "report_clipped": false,
  "results": [
    {
      "columns": {
        "model": "USB 2.0 Hub",
        "vendor": "VIA Labs, Inc."
      }
    },
    {
      "columns": {
        "model": "USB Keyboard",
        "vendor": "VIA Labs, Inc."
      }
    },
    {
      "columns": {
        "model": "USB Reciever",
        "vendor": "Logitech"
      }
    }
  ]
}
```

If a query has no results stored for the specified host, then `results` will be an empty array:

```json
{
  "query_id": 31,
  "host_id": 1,
  "host_name": "foo",
  "last_fetched": "2021-01-19T17:08:31Z",
  "report_clipped": false,
  "results": []
}
```

> Note: osquery scheduled queries do not return errors, so only non-error results are included in the report. If you suspect a query may be running into errors, you can use the [live query](#run-live-query) endpoint to get diagnostics.

### Create query

Creates a global query or team query.

`POST /api/v1/fleet/queries`

#### Parameters

| Name                            | Type    | In   | Description                                                                                                                                            |
| ------------------------------- | ------- | ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| name                            | string  | body | **Required**. The name of the query.                                                                                                                   |
| query                           | string  | body | **Required**. The query in SQL syntax.                                                                                                                 |
| description                     | string  | body | The query's description.                                                                                                                               |
| observer_can_run                | boolean | body | Whether or not users with the `observer` role can run the query. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). This field is only relevant for the `observer` role. The `observer_plus` role can run any query and is not limited by this flag (`observer_plus` role was added in Fleet 4.30.0). |
| team_id                         | integer | body | _Available in Fleet Premium_. The parent team to which the new query should be added. If omitted, the query will be global.                                           |
| interval                        | integer | body | The amount of time, in seconds, the query waits before running. Can be set to `0` to never run. Default: 0.       |
| platform                        | string  | body | The OS platforms where this query will run (other platforms ignored). Comma-separated string. If omitted, runs on all compatible platforms.                        |
| min_osquery_version             | string  | body | The minimum required osqueryd version installed on a host. If omitted, all osqueryd versions are acceptable.                                                                          |
| automations_enabled             | boolean | body | Whether to send data to the configured log destination according to the query's `interval`. |
| logging                         | string  | body | The type of log output for this query. Valid values: `"snapshot"`(default), `"differential"`, or `"differential_ignore_removals"`.                        |
| discard_data                    | boolean | body | Whether to skip saving the latest query results for each host. Default: `false`. |


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
  "logging": "snapshot",
  "discard_data": false
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
    "discard_data": false,
    "packs": []
  }
}
```

### Modify query

Modifies the query specified by ID.

`PATCH /api/v1/fleet/queries/:id`

#### Parameters

| Name                        | Type    | In   | Description                                                                                                                                            |
| --------------------------- | ------- | ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| id                          | integer | path | **Required.** The ID of the query.                                                                                                                     |
| name                        | string  | body | The name of the query.                                                                                                                                 |
| query                       | string  | body | The query in SQL syntax.                                                                                                                               |
| description                 | string  | body | The query's description.                                                                                                                               |
| observer_can_run            | boolean | body | Whether or not users with the `observer` role can run the query. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). This field is only relevant for the `observer` role. The `observer_plus` role can run any query and is not limited by this flag (`observer_plus` role was added in Fleet 4.30.0). |
| interval                   | integer | body | The amount of time, in seconds, the query waits before running. Can be set to `0` to never run. Default: 0.       |
| platform                    | string  | body | The OS platforms where this query will run (other platforms ignored). Comma-separated string. If set to "", runs on all compatible platforms.                    |
| min_osquery_version             | string  | body | The minimum required osqueryd version installed on a host. If omitted, all osqueryd versions are acceptable.                                                                          |
| automations_enabled             | boolean | body | Whether to send data to the configured log destination according to the query's `interval`. |
| logging             | string  | body | The type of log output for this query. Valid values: `"snapshot"`(default), `"differential"`, or `"differential_ignore_removals"`.                        |
| discard_data        | boolean  | body | Whether to skip saving the latest query results for each host. |

> Note that any of the following conditions will cause the existing query report to be deleted:
> - Updating the `query` (SQL) field
> - Changing `discard_data` from `false` to `true`
> - Changing `logging` from `"snapshot"` to `"differential"` or `"differential_ignore_removals"`

#### Example

`PATCH /api/v1/fleet/queries/2`

##### Request body

```json
{
  "name": "new_title_for_my_query",
  "interval": 3600, // Once per hour,
  "platform": "",
  "min_osquery_version": "",
  "automations_enabled": false,
  "discard_data": true
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
    "discard_data": true,
    "packs": []
  }
}
```

### Delete query by name

Deletes the query specified by name.

`DELETE /api/v1/fleet/queries/:name`

#### Parameters

| Name | Type       | In   | Description                          |
| ---- | ---------- | ---- | ------------------------------------ |
| name | string     | path | **Required.** The name of the query. |
| team_id | integer | body | _Available in Fleet Premium_. The ID of the parent team of the query to be deleted. If omitted, Fleet will search among queries in the global context. |

#### Example

`DELETE /api/v1/fleet/queries/foo`

##### Default response

`Status: 200`


### Delete query by ID

Deletes the query specified by ID.

`DELETE /api/v1/fleet/queries/id/:id`

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

| Name | Type  | In   | Description                           |
| ---- | ----- | ---- | ------------------------------------- |
| ids  | array | body | **Required.** The IDs of the queries. |

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

> This updated API endpoint replaced `GET /api/v1/fleet/queries/run` in Fleet 4.43.0, for improved compatibility with many HTTP clients. The [deprecated endpoint](https://github.com/fleetdm/fleet/blob/fleet-v4.42.0/docs/REST%20API/rest-api.md#run-live-query) is maintained for backwards compatibility.

Runs a live query against the specified hosts and responds with the results.

The live query will stop if the request times out. Timeouts happen if targeted hosts haven't responded after the configured `FLEET_LIVE_QUERY_REST_PERIOD` (default 25 seconds) or if the `distributed_interval` agent option (default 10 seconds) is higher than the `FLEET_LIVE_QUERY_REST_PERIOD`.


`POST /api/v1/fleet/queries/:id/run`

#### Parameters

| Name      | Type  | In   | Description                                                                                                                                                        |
|-----------|-------|------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| query_id | integer | path | **Required**. The ID of the saved query to run. |
| host_ids  | array | body | **Required**. The IDs of the hosts to target. User must be authorized to target all of these hosts.                                                                |

#### Example

`POST /api/v1/fleet/queries/123/run`

##### Request body

```json
{
  "host_ids": [ 1, 4, 34, 27 ]
}
```

##### Default response

```json
{
  "query_id": 123,
  "targeted_host_count": 4,
  "responded_host_count": 2,
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

`PATCH /api/v1/fleet/global/schedule/:id`

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

`DELETE /api/v1/fleet/global/schedule/:id`

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

`GET /api/v1/fleet/teams/:id/schedule`

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

`POST /api/v1/fleet/teams/:id/schedule`

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

`PATCH /api/v1/fleet/teams/:team_id/schedule/:scheduled_query_id`

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

`DELETE /api/v1/fleet/teams/:team_id/schedule/:scheduled_query_id`

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

- [Run script](#run-script)
- [Get script result](#get-script-result)
- [Add script](#add-script)
- [Delete script](#delete-script)
- [List scripts](#list-scripts)
- [Get or download script](#get-or-download-script)
- [Get script details by host](#get-hosts-scripts)

### Run script

Run a script on a host.

The script will be added to the host's list of upcoming activities.

The new script will run after other activities finish. Failure of one activity won't cancel other activities.

By default, script runs time out after 5 minutes. You can modify this default in your [agent configuration](https://fleetdm.com/docs/configuration/agent-configuration#script-execution-timeout).

`POST /api/v1/fleet/scripts/run`

#### Parameters

| Name            | Type    | In   | Description                                                                                    |
| ----            | ------- | ---- | --------------------------------------------                                                   |
| host_id         | integer | body | **Required**. The ID of the host to run the script on.                                                |
| script_id       | integer | body | The ID of the existing saved script to run. Only one of either `script_id`, `script_contents`, or `script_name` can be included. |
| script_contents | string  | body | The contents of the script to run. Only one of either `script_id`, `script_contents`, or `script_name` can be included. |
| script_name       | integer | body | The name of the existing saved script to run. If specified, requires `team_id`. Only one of either `script_id`, `script_contents`, or `script_name` can be included in the request.   |
| team_id       | integer | body | The ID of the existing saved script to run. If specified, requires `script_name`. Only one of either `script_id`, `script_contents`, or `script_name` can be included in the request.  |

> Note that if any combination of `script_id`, `script_contents`, and `script_name` are included in the request, this endpoint will respond with an error.

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

### Get script result

Gets the result of a script that was executed.

#### Parameters

| Name         | Type   | In   | Description                                   |
| ----         | ------ | ---- | --------------------------------------------  |
| execution_id | string | path | **Required**. The execution id of the script. |

#### Example

`GET /api/v1/fleet/scripts/results/:execution_id`

##### Default response

`Status: 200`

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
  "runtime": 20,
  "created_at": "2024-09-11T20:30:24Z"
}
```

> Note: `exit_code` can be `null` if Fleet hasn't heard back from the host yet.

> Note: `created_at` is the creation timestamp of the script execution request.

### Add script

Uploads a script, making it available to run on hosts assigned to the specified team (or no team).

`POST /api/v1/fleet/scripts`

#### Parameters

| Name            | Type    | In   | Description                                      |
| ----            | ------- | ---- | --------------------------------------------     |
| script          | file    | form | **Required**. The file containing the script.    |
| team_id         | integer | form | _Available in Fleet Premium_. The team ID. If specified, the script will only be available to hosts assigned to this team. If not specified, the script will only be available to hosts on **no team**.  |

#### Example

`POST /api/v1/fleet/scripts`

##### Request headers

```http
Content-Length: 306
Content-Type: multipart/form-data; boundary=------------------------f02md47480und42y
```

##### Request body

```http
--------------------------f02md47480und42y
Content-Disposition: form-data; name="team_id"

1
--------------------------f02md47480und42y
Content-Disposition: form-data; name="script"; filename="myscript.sh"
Content-Type: application/octet-stream

echo "hello"
--------------------------f02md47480und42y--

```

##### Default response

`Status: 200`

```json
{
  "script_id": 1227
}
```

### Delete script

Deletes an existing script.

`DELETE /api/v1/fleet/scripts/:id`

#### Parameters

| Name            | Type    | In   | Description                                           |
| ----            | ------- | ---- | --------------------------------------------          |
| id              | integer | path | **Required**. The ID of the script to delete. |

#### Example

`DELETE /api/v1/fleet/scripts/1`

##### Default response

`Status: 204`

### List scripts

`GET /api/v1/fleet/scripts`

#### Parameters

| Name            | Type    | In    | Description                                                                                                                   |
| --------------- | ------- | ----- | ----------------------------------------------------------------------------------------------------------------------------- |
| team_id         | integer | query | _Available in Fleet Premium_. The ID of the team to filter scripts by. If not specified, it will filter only scripts that are available to hosts with no team. |
| page            | integer | query | Page number of the results to fetch.                                                                                          |
| per_page        | integer | query | Results per page.                                                                                                             |

#### Example

`GET /api/v1/fleet/scripts`

##### Default response

`Status: 200`

```json
{
  "scripts": [
    {
      "id": 1,
      "team_id": null,
      "name": "script_1.sh",
      "created_at": "2023-07-30T13:41:07Z",
      "updated_at": "2023-07-30T13:41:07Z"
    },
    {
      "id": 2,
      "team_id": null,
      "name": "script_2.sh",
      "created_at": "2023-08-30T13:41:07Z",
      "updated_at": "2023-08-30T13:41:07Z"
    }
  ],
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
}

```

### Get or download script

`GET /api/v1/fleet/scripts/:id`

#### Parameters

| Name | Type    | In    | Description                                                       |
| ---- | ------- | ----  | -------------------------------------                             |
| id   | integer | path  | **Required.** The desired script's ID.                            |
| alt  | string  | query | If specified and set to "media", downloads the script's contents. |

#### Example (get script)

`GET /api/v1/fleet/scripts/123`

##### Default response

`Status: 200`

```json
{
  "id": 123,
  "team_id": null,
  "name": "script_1.sh",
  "created_at": "2023-07-30T13:41:07Z",
  "updated_at": "2023-07-30T13:41:07Z"
}

```

#### Example (download script)

`GET /api/v1/fleet/scripts/123?alt=media`

##### Example response headers

```http
Content-Length: 13
Content-Type: application/octet-stream
Content-Disposition: attachment;filename="2023-09-27 script_1.sh"
```

###### Example response body

`Status: 200`

```
echo "hello"
```

## Sessions

- [Get session info](#get-session-info)
- [Delete session](#delete-session)

### Get session info

Returns the session information for the session specified by ID.

`GET /api/v1/fleet/sessions/:id`

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

`DELETE /api/v1/fleet/sessions/:id`

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

- [List software](#list-software)
- [List software versions](#list-software-versions)
- [List operating systems](#list-operating-systems)
- [Get software](#get-software)
- [Get software version](#get-software-version)
- [Get operating system version](#get-operating-system-version)
- [Add package](#add-package)
- [Modify package](#modify-package)
- [List App Store apps](#list-app-store-apps)
- [Add App Store app](#add-app-store-app)
- [List Fleet-maintained apps](#list-fleet-maintained-apps)
- [Get Fleet-maintained app](#get-fleet-maintained-app)
- [Add Fleet-maintained app](#add-fleet-maintained-app)
- [Install package or App Store app](#install-package-or-app-store-app)
- [Get package install result](#get-package-install-result)
- [Download package](#download-package)
- [Delete package or App Store app](#delete-package-or-app-store-app)

### List software

Get a list of all software.

`GET /api/v1/fleet/software/titles`

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

#### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                |
| ----------------------- | ------- | ----- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                       |
| per_page                | integer | query | Results per page.                                                                                                                                                          |
| order_key               | string  | query | What to order results by. Allowed fields are `name` and `hosts_count`. Default is `hosts_count` (descending).                                                              |
| order_direction         | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`.                                              |
| query                   | string  | query | Search query keywords. Searchable fields include `title` and `cve`.                                                                                                        |
| team_id                 | integer | query | _Available in Fleet Premium_. Filters the software to only include the software installed on the hosts that are assigned to the specified team. Use `0` to filter by hosts assigned to "No team".                            |
| vulnerable              | boolean | query | If true or 1, only list software that has detected vulnerabilities. Default is `false`.                                                                                    |
| available_for_install   | boolean | query | If `true` or `1`, only list software that is available for install (added by the user). Default is `false`.                                                                |
| self_service            | boolean | query | If `true` or `1`, only lists self-service software. Default is `false`.  |
| packages_only           | boolean | query | If `true` or `1`, only lists packages available for install (without App Store apps).  |
| min_cvss_score | integer | query | _Available in Fleet Premium_. Filters to include only software with vulnerabilities that have a CVSS version 3.x base score higher than the specified value.   |
| max_cvss_score | integer | query | _Available in Fleet Premium_. Filters to only include software with vulnerabilities that have a CVSS version 3.x base score lower than what's specified.   |
| exploit | boolean | query | _Available in Fleet Premium_. If `true`, filters to only include software with vulnerabilities that have been actively exploited in the wild (`cisa_known_exploit: true`). Default is `false`.  |
| platform | string | query | Filter software by platform. Supported values are `darwin`, `windows`, and `linux`.  |

#### Example

`GET /api/v1/fleet/software/titles?team_id=3`

##### Default response

`Status: 200`

```json
{
  "counts_updated_at": "2022-01-01 12:32:00",
  "count": 2,
  "software_titles": [
    {
      "id": 12,
      "name": "Firefox.app",
      "software_package": {
        "name": "FirefoxInsall.pkg",
        "version": "125.6",
        "self_service": true,
        "automatic_install_policies": [
          {
            "id": 343,
            "name": "[Install software] Firefox.app",
          }
        ],
      },
      "app_store_app": null,
      "versions_count": 3,
      "source": "apps",
      "browser": "",
      "hosts_count": 48,
      "versions": [
        {
          "id": 123,
          "version": "1.12",
          "vulnerabilities": ["CVE-2023-1234","CVE-2023-4321","CVE-2023-7654"]
        },
        {
          "id": 124,
          "version": "3.4",
          "vulnerabilities": ["CVE-2023-1234","CVE-2023-4321","CVE-2023-7654"]
        },
        {
          "id": 12,
          "version": "1.13",
          "vulnerabilities": ["CVE-2023-1234","CVE-2023-4321","CVE-2023-7654"]
        }
      ]
    },
    {
      "id": 22,
      "name": "Google Chrome.app",
      "software_package": null,
      "app_store_app": null,
      "versions_count": 5,
      "source": "apps",
      "browser": "",
      "hosts_count": 345,
      "versions": [
        {
          "id": 331,
          "version": "118.1",
          "vulnerabilities": ["CVE-2023-1234"]
        },
        {
          "id": 332,
          "version": "119.0",
          "vulnerabilities": ["CVE-2023-9876", "CVE-2023-2367"]
        },
        {
          "id": 334,
          "version": "119.4",
          "vulnerabilities": ["CVE-2023-1133", "CVE-2023-2224"]
        },
        {
          "id": 348,
          "version": "121.5",
          "vulnerabilities": ["CVE-2023-0987", "CVE-2023-5673", "CVE-2023-1334"]
        },
      ]
    },
    {
      "id": 32,
      "name": "1Password – Password Manager",
      "software_package": null,
      "app_store_app": null,
      "versions_count": 1,
      "source": "chrome_extensions",
      "browser": "chrome",
      "hosts_count": 345,
      "versions": [
        {
          "id": 4242,
          "version": "2.3.7",
          "vulnerabilities": []
        }
      ]
    }
  ],
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
}
```

### List software versions

Get a list of all software versions.

`GET /api/v1/fleet/software/versions`

#### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                |
| ----------------------- | ------- | ----- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                       |
| per_page                | integer | query | Results per page.                                                                                                                                                          |
| order_key               | string  | query | What to order results by. Allowed fields are `name`, `hosts_count`, `cve_published`, `cvss_score`, `epss_probability` and `cisa_known_exploit`. Default is `hosts_count` (descending).      |
| order_direction         | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`.                                              |
| query                   | string  | query | Search query keywords. Searchable fields include `name`, `version`, and `cve`.                                                                                             |
| team_id                 | integer | query | _Available in Fleet Premium_. Filters the software to only include the software installed on the hosts that are assigned to the specified team. Use `0` to filter by hosts assigned to "No team".                             |
| vulnerable              | boolean    | query | If true or 1, only list software that has detected vulnerabilities. Default is `false`.                                                                                    |
| min_cvss_score | integer | query | _Available in Fleet Premium_. Filters to include only software with vulnerabilities that have a CVSS version 3.x base score higher than the specified value.   |
| max_cvss_score | integer | query | _Available in Fleet Premium_. Filters to only include software with vulnerabilities that have a CVSS version 3.x base score lower than what's specified.   |
| exploit | boolean | query | _Available in Fleet Premium_. If `true`, filters to only include software with vulnerabilities that have been actively exploited in the wild (`cisa_known_exploit: true`). Default is `false`.  |
| without_vulnerability_details | boolean | query | _Available in Fleet Premium_. If `true` only vulnerability name is included in response. If `false` (or omitted), adds vulnerability description, CVSS score, and other details available in Fleet Premium. See notes below on performance. |

> For optimal performance, we recommend Fleet Premium users set `without_vulnerability_details` to `true` whenever possible. If set to `false` a large amount of data will be included in the response. If you need vulnerability details, consider using the [Get vulnerability](#get-vulnerability) endpoint.

#### Example

`GET /api/v1/fleet/software/versions`

##### Default response

`Status: 200`

```json
{
    "counts_updated_at": "2022-01-01 12:32:00",
    "count": 1,
    "software": [
      {
        "id": 1,
        "name": "glibc",
        "version": "2.12",
        "source": "rpm_packages",
        "browser": "",
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
            "cve_description": "In the GNU C Library (aka glibc or libc6) before 2.28, parse_reg_exp in posix/regcomp.c misparses alternatives, which allows attackers to cause a denial of service (assertion failure and application exit) or trigger an incorrect result by attempting a regular-expression match.",
            "resolved_in_version": "2.28"
          }
        ],
        "hosts_count": 1
      },
      {
        "id": 2,
        "name": "1Password – Password Manager",
        "version": "2.10.0",
        "source": "chrome_extensions",
        "browser": "chrome",
        "extension_id": "aeblfdkhhhdcdjpifhhbdiojplfjncoa",
        "generated_cpe": "cpe:2.3:a:1password:1password:2.19.0:*:*:*:*:chrome:*:*",
        "hosts_count": 345,
        "vulnerabilities": null
      }
    ],
    "meta": {
      "has_next_results": false,
      "has_previous_results": false
    }
}
```

### List operating systems

Returns a list of all operating systems.

`GET /api/v1/fleet/os_versions`

#### Parameters

| Name                | Type     | In    | Description                                                                                                                          |
| ---      | ---      | ---   | ---                                                                                                                                  |
| team_id             | integer | query | _Available in Fleet Premium_. Filters response data to the specified team. Use `0` to filter by hosts assigned to "No team".  |
| platform            | string   | query | Filters the hosts to the specified platform |
| os_name     | string | query | The name of the operating system to filter hosts by. `os_version` must also be specified with `os_name`                                                 |
| os_version    | string | query | The version of the operating system to filter hosts by. `os_name` must also be specified with `os_version`                                                 |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                       |
| per_page                | integer | query | Results per page.                                                                                                                                                          |
| order_key               | string  | query | What to order results by. Allowed fields are: `hosts_count`. Default is `hosts_count` (descending).      |
| order_direction | string | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |


##### Default response

`Status: 200`

```json
{
  "count": 1,
  "counts_updated_at": "2023-12-06T22:17:30Z",
  "os_versions": [
    {
      "os_version_id": 123,
      "hosts_count": 21,
      "name": "Microsoft Windows 11 Pro 23H2 10.0.22621.1234",
      "name_only": "Microsoft Windows 11 Pro 23H2",
      "version": "10.0.22621.1234",
      "platform": "windows",
      "generated_cpes": [],
      "vulnerabilities": [
        {
          "cve": "CVE-2022-30190",
          "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2022-30190",
          "cvss_score": 7.8,// Available in Fleet Premium
          "epss_probability": 0.9729,// Available in Fleet Premium
          "cisa_known_exploit": false,// Available in Fleet Premium
          "cve_published": "2022-06-01T00:15:00Z",// Available in Fleet Premium
          "cve_description": "Microsoft Windows Support Diagnostic Tool (MSDT) Remote Code Execution Vulnerability.",// Available in Fleet Premium
          "resolved_in_version": ""// Available in Fleet Premium
        }
      ]
    }
  ],
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
}
```

OS vulnerability data is currently available for Windows and macOS. For other platforms, `vulnerabilities` will be an empty array:

```json
{
  "hosts_count": 1,
  "name": "CentOS Linux 7.9.2009",
  "name_only": "CentOS",
  "version": "7.9.2009",
  "platform": "rhel",
  "generated_cpes": [],
  "vulnerabilities": []
}
```

### Get software

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

Returns information about the specified software. By default, `versions` are sorted in descending order by the `hosts_count` field.

`GET /api/v1/fleet/software/titles/:id`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| id   | integer | path | **Required.** The software title's ID. |
| team_id             | integer | query | _Available in Fleet Premium_. Filters response data to the specified team. Use `0` to filter by hosts assigned to "No team".  |

#### Example

`GET /api/v1/fleet/software/titles/12?team_id=3`

##### Default response

`Status: 200`

```json
{
  "software_title": {
    "id": 12,
    "name": "Falcon.app",
    "bundle_identifier": "crowdstrike.falcon.Agent",
    "software_package": {
      "name": "FalconSensor-6.44.pkg",
      "version": "6.44",
      "installer_id": 23,
      "team_id": 3,
      "uploaded_at": "2024-04-01T14:22:58Z",
      "install_script": "sudo installer -pkg '$INSTALLER_PATH' -target /",
      "pre_install_query": "SELECT 1 FROM macos_profiles WHERE uuid='c9f4f0d5-8426-4eb8-b61b-27c543c9d3db';",
      "post_install_script": "sudo /Applications/Falcon.app/Contents/Resources/falconctl license 0123456789ABCDEFGHIJKLMNOPQRSTUV-WX",
      "uninstall_script": "/Library/CS/falconctl uninstall",
      "self_service": true,
      "automatic_install_policies": [
        {
          "id": 343,
          "name": "[Install software] Crowdstrike Agent",
        }
      ],
      "status": {
        "installed": 3,
        "pending_install": 1,
        "failed_install": 0,
        "pending_uninstall": 2,
        "failed_uninstall": 1
      }
    },
    "app_store_app": null,
    "counts_updated_at": "2024-11-03T22:39:36Z",
    "source": "apps",
    "browser": "",
    "hosts_count": 48,
    "versions": [
      {
        "id": 123,
        "version": "117.0",
        "vulnerabilities": ["CVE-2023-1234"],
        "hosts_count": 37
      },
      {
        "id": 124,
        "version": "116.0",
        "vulnerabilities": ["CVE-2023-4321"],
        "hosts_count": 7
      },
      {
        "id": 127,
        "version": "115.5",
        "vulnerabilities": ["CVE-2023-7654"],
        "hosts_count": 4
      }
    ]
  }
}
```

#### Example (App Store app)

`GET /api/v1/fleet/software/titles/15`

##### Default response

`Status: 200`

```json
{
  "software_title": {
    "id": 15,
    "name": "Logic Pro",
    "bundle_identifier": "com.apple.logic10",
    "software_package": null,
    "app_store_app": {
      "name": "Logic Pro",
      "app_store_id": 1091189122,
      "latest_version": "2.04",
      "icon_url": "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/f1/65/1e/a4844ccd-486d-455f-bb31-67336fe46b14/AppIcon-1x_U007emarketing-0-7-0-85-220-0.png/512x512bb.jpg",
      "self_service": true,
      "status": {
        "installed": 3,
        "pending": 1,
        "failed": 2,
      }
    },
    "source": "apps",
    "browser": "",
    "hosts_count": 48,
    "versions": [
      {
        "id": 123,
        "version": "2.04",
        "vulnerabilities": [],
        "hosts_count": 24
      }
    ]
  }
}
```

### Get software version

Returns information about the specified software version.

`GET /api/v1/fleet/software/versions/:id`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| id   | integer | path | **Required.** The software version's ID. |
| team_id             | integer | query | _Available in Fleet Premium_. Filters response data to the specified team. Use `0` to filter by hosts assigned to "No team".  |

#### Example

`GET /api/v1/fleet/software/versions/12`

##### Default response

`Status: 200`

```json
{
  "software": {
    "id": 425224,
    "name": "Firefox.app",
    "version": "117.0",
    "bundle_identifier": "org.mozilla.firefox",
    "source": "apps",
    "browser": "",
    "generated_cpe": "cpe:2.3:a:mozilla:firefox:117.0:*:*:*:*:macos:*:*",
    "vulnerabilities": [
      {
        "cve": "CVE-2023-4863",
        "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2023-4863",
        "created_at": "2024-07-01T00:15:00Z",
        "cvss_score": 8.8, // Available in Fleet Premium
        "epss_probability": 0.4101, // Available in Fleet Premium
        "cisa_known_exploit": true, // Available in Fleet Premium
        "cve_published": "2023-09-12T15:15:00Z", // Available in Fleet Premium
        "resolved_in_version": "" // Available in Fleet Premium
      },
      {
        "cve": "CVE-2023-5169",
        "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2023-5169",
        "created_at": "2024-07-01T00:15:00Z",
        "cvss_score": 6.5, // Available in Fleet Premium
        "epss_probability": 0.00073, // Available in Fleet Premium
        "cisa_known_exploit": false, // Available in Fleet Premium
        "cve_published": "2023-09-27T15:19:00Z", // Available in Fleet Premium
        "resolved_in_version": "118" // Available in Fleet Premium
      }
    ]
  }
}
```


### Get operating system version

Retrieves information about the specified operating system (OS) version.

`GET /api/v1/fleet/os_versions/:id`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| id   | integer | path | **Required.** The OS version's ID. |
| team_id             | integer | query | _Available in Fleet Premium_. Filters response data to the specified team. Use `0` to filter by hosts assigned to "No team".  |

##### Default response

`Status: 200`

```json
{
  "counts_updated_at": "2023-12-06T22:17:30Z",
  "os_version": {
    "id": 123,
    "hosts_count": 21,
    "name": "Microsoft Windows 11 Pro 23H2 10.0.22621.1234",
    "name_only": "Microsoft Windows 11 Pro 23H2",
    "version": "10.0.22621.1234",
    "platform": "windows",
    "generated_cpes": [],
    "vulnerabilities": [
      {
        "cve": "CVE-2022-30190",
        "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2022-30190",
        "created_at": "2024-07-01T00:15:00Z",
        "cvss_score": 7.8,// Available in Fleet Premium
        "epss_probability": 0.9729,// Available in Fleet Premium
        "cisa_known_exploit": false,// Available in Fleet Premium
        "cve_published": "2022-06-01T00:15:00Z",// Available in Fleet Premium
        "cve_description": "Microsoft Windows Support Diagnostic Tool (MSDT) Remote Code Execution Vulnerability.",// Available in Fleet Premium
        "resolved_in_version": ""// Available in Fleet Premium
      }
    ]
  }
}
```

OS vulnerability data is currently available for Windows and macOS. For other platforms, `vulnerabilities` will be an empty array:

```json
{
  "id": 321,
  "hosts_count": 1,
  "name": "CentOS Linux 7.9.2009",
  "name_only": "CentOS",
  "version": "7.9.2009",
  "platform": "rhel",
  "generated_cpes": [],
  "vulnerabilities": []
}
```

### Add package

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

_Available in Fleet Premium._

Add a package (.pkg, .msi, .exe, .deb, .rpm) to install on macOS, Windows, or Linux hosts.


`POST /api/v1/fleet/software/package`

#### Parameters

| Name            | Type    | In   | Description                                      |
| ----            | ------- | ---- | --------------------------------------------     |
| software        | file    | form | **Required**. Installer package file. Supported packages are .pkg, .msi, .exe, .deb, and .rpm.   |
| team_id         | integer | form | **Required**. The team ID. Adds a software package to the specified team. |
| install_script  | string | form | Script that Fleet runs to install software. If not specified Fleet runs [default install script](https://github.com/fleetdm/fleet/tree/f71a1f183cc6736205510580c8366153ea083a8d/pkg/file/scripts) for each package type. |
| pre_install_query  | string | form | Query that is pre-install condition. If the query doesn't return any result, Fleet won't proceed to install. |
| post_install_script | string | form | The contents of the script to run after install. If the specified script fails (exit code non-zero) software install will be marked as failed and rolled back. |
| self_service | boolean | form | Self-service software is optional and can be installed by the end user. |


#### Example

`POST /api/v1/fleet/software/package`

##### Request header

```http
Content-Length: 8500
Content-Type: multipart/form-data; boundary=------------------------d8c247122f594ba0
```

##### Request body

```http
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="team_id"
1
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="self_service"
true
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="install_script"
sudo installer -pkg /temp/FalconSensor-6.44.pkg -target /
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="pre_install_query"
SELECT 1 FROM macos_profiles WHERE uuid='c9f4f0d5-8426-4eb8-b61b-27c543c9d3db';
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="post_install_script"
sudo /Applications/Falcon.app/Contents/Resources/falconctl license 0123456789ABCDEFGHIJKLMNOPQRSTUV-WX
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="software"; filename="FalconSensor-6.44.pkg"
Content-Type: application/octet-stream
<BINARY_DATA>
--------------------------d8c247122f594ba0
```

##### Default response

`Status: 200`

### Modify package

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

_Available in Fleet Premium._

Update a package to install on macOS, Windows, or Linux (Ubuntu) hosts.

`PATCH /api/v1/fleet/software/titles/:id/package`

#### Parameters

| Name            | Type    | In   | Description                                      |
| ----            | ------- | ---- | --------------------------------------------     |
| id | integer | path | ID of the software title being updated. |
| software        | file    | form | Installer package file. Supported packages are .pkg, .msi, .exe, .deb, and .rpm.   |
| team_id         | integer | form | **Required**. The team ID. Updates a software package in the specified team. |
| install_script  | string | form | Command that Fleet runs to install software. If not specified Fleet runs the [default install command](https://github.com/fleetdm/fleet/tree/f71a1f183cc6736205510580c8366153ea083a8d/pkg/file/scripts) for each package type. |
| pre_install_query  | string | form | Query that is pre-install condition. If the query doesn't return any result, the package will not be installed. |
| post_install_script | string | form | The contents of the script to run after install. If the specified script fails (exit code non-zero) software install will be marked as failed and rolled back. |
| self_service | boolean | form | Whether this is optional self-service software that can be installed by the end user. |

> Changes to the installer package will reset installation counts. Changes to any field other than `self_service` will cancel pending installs for the old package.
#### Example

`PATCH /api/v1/fleet/software/titles/1/package`

##### Request header

```http
Content-Length: 8500
Content-Type: multipart/form-data; boundary=------------------------d8c247122f594ba0
```

##### Request body

```http
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="team_id"
1
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="self_service"
true
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="install_script"
sudo installer -pkg /temp/FalconSensor-6.44.pkg -target /
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="pre_install_query"
SELECT 1 FROM macos_profiles WHERE uuid='c9f4f0d5-8426-4eb8-b61b-27c543c9d3db';
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="post_install_script"
sudo /Applications/Falcon.app/Contents/Resources/falconctl license 0123456789ABCDEFGHIJKLMNOPQRSTUV-WX
--------------------------d8c247122f594ba0
Content-Disposition: form-data; name="software"; filename="FalconSensor-6.44.pkg"
Content-Type: application/octet-stream
<BINARY_DATA>
--------------------------d8c247122f594ba0
```

##### Default response

`Status: 200`

```json
{
  "software_package": {
    "name": "FalconSensor-6.44.pkg",
    "version": "6.44",
    "installer_id": 23,
    "team_id": 3,
    "uploaded_at": "2024-04-01T14:22:58Z",
    "install_script": "sudo installer -pkg /temp/FalconSensor-6.44.pkg -target /",
    "pre_install_query": "SELECT 1 FROM macos_profiles WHERE uuid='c9f4f0d5-8426-4eb8-b61b-27c543c9d3db';",
    "post_install_script": "sudo /Applications/Falcon.app/Contents/Resources/falconctl license 0123456789ABCDEFGHIJKLMNOPQRSTUV-WX",
    "self_service": true,
    "status": {
      "installed": 0,
      "pending": 0,
      "failed": 0
    }
  }
}
```

### List App Store apps

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

Returns the list of Apple App Store (VPP) that can be added to the specified team. If an app is already added to the team, it's excluded from the list.

`GET /api/v1/fleet/software/app_store_apps`

#### Parameters

| Name    | Type | In | Description |
| ------- | ---- | -- | ----------- |
| team_id | integer | query | **Required**. The team ID. |

#### Example

`GET /api/v1/fleet/software/app_store_apps/?team_id=3`

##### Default response

`Status: 200`

```json
{
  "app_store_apps": [
    {
      "name": "Xcode",
      "icon_url": "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/f1/65/1e/a4844ccd-486d-455f-bb31-67336fe46b14/AppIcon-1x_U007emarketing-0-7-0-85-220-0.png/512x512bb.jpg",
      "latest_version": "15.4",
      "app_store_id": "497799835",
      "platform": "darwin"
    },
    {
      "name": "Logic Pro",
      "icon_url": "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/f1/65/1e/a4844ccd-486d-455f-bb31-67336fe46b14/AppIcon-1x_U007emarketing-0-7-0-85-220-0.png/512x512bb.jpg",
      "latest_version": "2.04",
      "app_store_id": "634148309",
      "platform": "ios"
    },
    {
      "name": "Logic Pro",
      "icon_url": "https://is1-ssl.mzstatic.com/image/thumb/Purple211/v4/f1/65/1e/a4844ccd-486d-455f-bb31-67336fe46b14/AppIcon-1x_U007emarketing-0-7-0-85-220-0.png/512x512bb.jpg",
      "latest_version": "2.04",
      "app_store_id": "634148309",
      "platform": "ipados"
    },
  ]
}
```

### Add App Store app

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

_Available in Fleet Premium._

Add App Store (VPP) app purchased in Apple Business Manager.

`POST /api/v1/fleet/software/app_store_apps`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| app_store_id   | string | body | **Required.** The ID of App Store app. |
| team_id       | integer | body | **Required**. The team ID. Adds VPP software to the specified team.  |
| platform | string | body | The platform of the app (`darwin`, `ios`, or `ipados`). Default is `darwin`. |
| self_service | boolean | body | Self-service software is optional and can be installed by the end user. |

#### Example

`POST /api/v1/fleet/software/app_store_apps`

##### Request body

```json
{
  "app_store_id": "497799835",
  "team_id": 2,
  "platform": "ipados",
  "self_service": true
}
```

##### Default response

`Status: 200`

### List Fleet-maintained apps

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

List available Fleet-maintained apps.

`GET /api/v1/fleet/software/fleet_maintained_apps`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| team_id  | integer | query | If supplied, only list apps for which an installer doesn't already exist for the specified team.  |
| page     | integer | query | Page number of the results to fetch.  |
| per_page | integer | query | Results per page.  |

#### Example

`GET /api/v1/fleet/software/fleet_maintained_apps?team_id=3`

##### Default response

`Status: 200`

```json
{
  "fleet_maintained_apps": [
    {
      "id": 1,
      "name": "1Password",
      "version": "8.10.40",
      "platform": "darwin"
    },
    {
      "id": 2,
      "name": "Adobe Acrobat Reader",
      "version": "24.002.21005",
      "platform": "darwin"
    },
    {
      "id": 3,
      "name": "Box Drive",
      "version": "2.39.179",
      "platform": "darwin"
    },
  ],
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
}
```

### Get Fleet-maintained app

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.
Returns information about the specified Fleet-maintained app.

`GET /api/v1/fleet/software/fleet_maintained_apps/:id`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| id   | integer | path | **Required.** The Fleet-maintained app's ID. |


#### Example

`GET /api/v1/fleet/software/fleet_maintained_apps/1`

##### Default response

`Status: 200`

```json
{
  "fleet_maintained_app": {
    "id": 1,
    "name": "1Password",
    "filename": "1Password-8.10.44-aarch64.zip",
    "version": "8.10.40",
    "platform": "darwin",
    "install_script": "#!/bin/sh\ninstaller -pkg \"$INSTALLER_PATH\" -target /",
    "uninstall_script": "#!/bin/sh\npkg_ids=$PACKAGE_ID\nfor pkg_id in '${pkg_ids[@]}'...",
  }
}
```

### Add Fleet-maintained app

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.
_Available in Fleet Premium._

Add Fleet-maintained app so it's available for install.

`POST /api/v1/fleet/software/fleet_maintained_apps`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| fleet_maintained_app_id   | integer | body | **Required.** The ID of Fleet-maintained app. |
| team_id       | integer | body | **Required**. The team ID. Adds Fleet-maintained app to the specified team.  |
| install_script  | string | body | Command that Fleet runs to install software. If not specified Fleet runs default install command for each Fleet-maintained app. |
| pre_install_query  | string | body | Query that is pre-install condition. If the query doesn't return any result, Fleet won't proceed to install. |
| post_install_script | string | body | The contents of the script to run after install. If the specified script fails (exit code non-zero) software install will be marked as failed and rolled back. |
| self_service | boolean | body | Self-service software is optional and can be installed by the end user. |

#### Example

`POST /api/v1/fleet/software/fleet_maintained_apps`

##### Request body

```json
{
  "fleet_maintained_app_id": 3,
  "team_id": 2
}
```

##### Default response

`Status: 200`

```json
{
  "software_title_id": 234
}
```

### Download package

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

_Available in Fleet Premium._

`GET /api/v1/fleet/software/titles/:id/package?alt=media`

#### Parameters

| Name            | Type    | In   | Description                                      |
| ----            | ------- | ---- | --------------------------------------------     |
| id   | integer | path | **Required**. The ID of the software title to download software package.|
| team_id | integer | query | **Required**. The team ID. Downloads a software package added to the specified team. |
| alt             | integer | query | **Required**. If specified and set to "media", downloads the specified software package. |

#### Example

`GET /api/v1/fleet/software/titles/123/package?alt=media?team_id=2`

##### Default response

`Status: 200`

```http
Status: 200
Content-Type: application/octet-stream
Content-Disposition: attachment
Content-Length: <length>
Body: <blob>
```

### Install package or App Store app

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

_Available in Fleet Premium._

Install software (package or App Store app) on a macOS, iOS, iPadOS, Windows, or Linux (Ubuntu) host. Software title must have a `software_package` or `app_store_app` to be installed.

Package installs time out after 1 hour.

`POST /api/v1/fleet/hosts/:id/software/:software_title_id/install`

#### Parameters

| Name              | Type       | In   | Description                                      |
| ---------         | ---------- | ---- | --------------------------------------------     |
| id                | integer    | path | **Required**. The host's ID.                     |
| software_title_id | integer    | path | **Required**. The software title's ID.           |

#### Example

`POST /api/v1/fleet/hosts/123/software/3435/install`

##### Default response

`Status: 202`

### Uninstall package

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.
_Available in Fleet Premium._

Uninstall software (package) on a macOS, Windows, or Linux (Ubuntu) host. Software title must have a `software_package` added to be uninstalled.

`POST /api/v1/fleet/hosts/:id/software/:software_title_id/uninstall`

#### Parameters

| Name              | Type       | In   | Description                                      |
| ---------         | ---------- | ---- | --------------------------------------------     |
| id                | integer    | path | **Required**. The host's ID.                     |
| software_title_id | integer    | path | **Required**. The software title's ID.           |

#### Example

`POST /api/v1/fleet/hosts/123/software/3435/uninstall`

##### Default response

`Status: 202`

### Get package install result

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

_Available in Fleet Premium._

`GET /api/v1/fleet/software/install/:install_uuid/results`

Get the results of a software package install.

To get the results of an App Store app install, use the [List MDM commands](#list-mdm-commands) and [Get MDM command results](#get-mdm-command-results) API enpoints. Fleet uses an MDM command to install App Store apps.

| Name            | Type    | In   | Description                                      |
| ----            | ------- | ---- | --------------------------------------------     |
| install_uuid | string | path | **Required**. The software installation UUID.|

#### Example

`GET /api/v1/fleet/software/install/b15ce221-e22e-4c6a-afe7-5b3400a017da/results`

##### Default response

`Status: 200`

```json
 {
   "install_uuid": "b15ce221-e22e-4c6a-afe7-5b3400a017da",
   "software_title": "Falcon.app",
   "software_title_id": 8353,
   "software_package": "FalconSensor-6.44.pkg",
   "host_id": 123,
   "host_display_name": "Marko's MacBook Pro",
   "status": "failed_install",
   "output": "Installing software...\nError: The operation can’t be completed because the item “Falcon” is in use.",
   "pre_install_query_output": "Query returned result\nSuccess",
   "post_install_script_output": "Running script...\nExit code: 1 (Failed)\nRolling back software install...\nSuccess"
 }
```

### Download package

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

_Available in Fleet Premium._

`GET /api/v1/fleet/software/titles/:software_title_id/package?alt=media`

#### Parameters

| Name            | Type    | In   | Description                                      |
| ----            | ------- | ---- | --------------------------------------------     |
| software_title_id   | integer | path | **Required**. The ID of the software title to download software package.|
| team_id | integer | query | **Required**. The team ID. Downloads a software package added to the specified team. |
| alt             | integer | query | **Required**. If specified and set to "media", downloads the specified software package. |

#### Example

`GET /api/v1/fleet/software/titles/123/package?alt=media?team_id=2`

##### Default response

`Status: 200`

```http
Status: 200
Content-Type: application/octet-stream
Content-Disposition: attachment
Content-Length: <length>
Body: <blob>
```

### Delete package or App Store app

> **Experimental feature**. This feature is undergoing rapid improvement, which may result in breaking changes to the API or configuration surface. It is not recommended for use in automated workflows.

_Available in Fleet Premium._

Deletes software that's available for install (package or App Store app).

`DELETE /api/v1/fleet/software/titles/:software_title_id/available_for_install`

#### Parameters

| Name            | Type    | In   | Description                                      |
| ----            | ------- | ---- | --------------------------------------------     |
| software_title_id              | integer | path | **Required**. The ID of the software title to delete software available for install. |
| team_id | integer | query | **Required**. The team ID. Deletes a software package added to the specified team. |

#### Example

`DELETE /api/v1/fleet/software/titles/24/available_for_install?team_id=2`

##### Default response

`Status: 204`

## Vulnerabilities

- [List vulnerabilities](#list-vulnerabilities)
- [Get vulnerability](#get-vulnerability)

### List vulnerabilities

Retrieves a list of all CVEs affecting software and/or OS versions.

`GET /api/v1/fleet/vulnerabilities`

#### Parameters

| Name                | Type     | In    | Description                                                                                                                          |
| ---      | ---      | ---   | ---                                                                                                                                  |
| team_id             | integer | query | _Available in Fleet Premium_. Filters only include vulnerabilities affecting the specified team. Use `0` to filter by hosts assigned to "No team".  |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                       |
| per_page                | integer | query | Results per page.                                                                                                                                                          |
| order_key               | string  | query | What to order results by. Allowed fields are: `cve`, `cvss_score`, `epss_probability`, `cve_published`, `created_at`, and `host_count`. Default is `created_at` (descending).      |
| order_direction | string | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |
| query | string | query | Search query keywords. Searchable fields include `cve`. |
| exploit | boolean | query | _Available in Fleet Premium_. If `true`, filters to only include vulnerabilities that have been actively exploited in the wild (`cisa_known_exploit: true`). Otherwise, includes vulnerabilities with any `cisa_known_exploit` value.  |


##### Default response

`Status: 200`

```json
{
  "vulnerabilities": [
    {
      "cve": "CVE-2022-30190",
      "created_at": "2022-06-01T00:15:00Z",
      "hosts_count": 1234,
      "hosts_count_updated_at": "2023-12-20T15:23:57Z",
      "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2022-30190",
      "cvss_score": 7.8,// Available in Fleet Premium
      "epss_probability": 0.9729,// Available in Fleet Premium
      "cisa_known_exploit": false,// Available in Fleet Premium
      "cve_published": "2022-06-01T00:15:00Z",// Available in Fleet Premium
      "cve_description": "Microsoft Windows Support Diagnostic Tool (MSDT) Remote Code Execution Vulnerability.",// Available in Fleet Premium
    }
  ],
  "count": 123,
  "counts_updated_at": "2024-02-02T16:40:37Z",
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
}
```


### Get vulnerability

Retrieve details about a vulnerability and its affected software and OS versions.

If no vulnerable OS versions or software were found, but Fleet is aware of the vulnerability, a 204 status code is returned.

#### Parameters

| Name    | Type    | In    | Description                                                                                                                  |
|---------|---------|-------|------------------------------------------------------------------------------------------------------------------------------|
| cve     | string  | path  | The cve to get information about (format must be CVE-YYYY-<4 or more digits>, case-insensitive).                             |
| team_id | integer | query | _Available in Fleet Premium_. Filters response data to the specified team. Use `0` to filter by hosts assigned to "No team". |

`GET /api/v1/fleet/vulnerabilities/:cve`

#### Example

`GET /api/v1/fleet/vulnerabilities/cve-2022-30190`

##### Default response

`Status: 200`

```json
"vulnerability": {
  "cve": "CVE-2022-30190",
  "created_at": "2022-06-01T00:15:00Z",
  "hosts_count": 1234,
  "hosts_count_updated_at": "2023-12-20T15:23:57Z",
  "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2022-30190",
  "cvss_score": 7.8,// Available in Fleet Premium
  "epss_probability": 0.9729,// Available in Fleet Premium
  "cisa_known_exploit": false,// Available in Fleet Premium
  "cve_published": "2022-06-01T00:15:00Z",// Available in Fleet Premium
  "cve_description": "Microsoft Windows Support Diagnostic Tool (MSDT) Remote Code Execution Vulnerability.",// Available in Fleet Premium
  "os_versions" : [
    {
      "os_version_id": 6,
      "hosts_count": 200,
      "name": "macOS 14.1.2",
      "name_only": "macOS",
      "version": "14.1.2",

      "resolved_in_version": "14.2",
      "generated_cpes": [
        "cpe:2.3:o:apple:macos:*:*:*:*:*:14.2:*:*",
        "cpe:2.3:o:apple:mac_os_x:*:*:*:*:*:14.2:*:*"
      ]
    }
  ],
  "software": [
    {
      "id": 2363,
      "name": "Docker Desktop",
      "version": "4.9.1",
      "source": "programs",
      "browser": "",
      "generated_cpe": "cpe:2.3:a:docker:docker_desktop:4.9.1:*:*:*:*:windows:*:*",
      "hosts_count": 50,
      "resolved_in_version": "5.0.0"
    }
  ]
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

`GET /api/v1/fleet/teams/:id`

`mdm.macos_settings.custom_settings`, `mdm.windows_settings.custom_settings`, and `scripts` only include the configuration profiles and scripts applied using [Fleet's YAML](https://fleetdm.com/docs/configuration/yaml-files). To list profiles or scripts added in the UI or API, use the [List configuration profiles](https://fleetdm.com/docs/rest-api/rest-api#list-custom-os-settings-configuration-profiles) or [List scripts](https://fleetdm.com/docs/rest-api/rest-api#list-scripts) endpoints instead.

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
    "integrations": {
      "google_calendar": {
        "enable_calendar_events": true,
        "webhook_url": "https://server.com/example"
      }
    },
    "mdm": {
      "enable_disk_encryption": true,
      "macos_updates": {
        "minimum_version": "12.3.1",
        "deadline": "2022-01-01"
      },
      "windows_updates": {
        "deadline_days": 5,
        "grace_period_days": 1
      },
      "macos_settings": {
        "custom_settings": [
          {
            "path": "path/to/profile1.mobileconfig",
            "labels": ["Label 1", "Label 2"]
          }
        ]
      },
      "windows_settings": {
        "custom_settings": [
          {
            "path": "path/to/profile2.xml",
            "labels": ["Label 3", "Label 4"]
          }
        ],
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

`PATCH /api/v1/fleet/teams/:id`

#### Parameters

| Name                                                    | Type    | In   | Description                                                                                                                                                                                               |
| ------------------------------------------------------- | ------- | ---- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| id                                                      | integer | path | **Required.** The desired team's ID.                                                                                                                                                                      |
| name                                                    | string  | body | The team's name.                                                                                                                                                                                          |
| host_ids                                                | array    | body | A list of hosts that belong to the team.                                                                                                                                                                  |
| user_ids                                                | array    | body | A list of users on the team.                                                                                                                                                             |
| webhook_settings                                        | object  | body | Webhook settings contains for the team.                                                                                                                                                                   |
| &nbsp;&nbsp;failing_policies_webhook                    | object  | body | Failing policies webhook settings.                                                                                                                                                                        |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_failing_policies_webhook | boolean | body | Whether or not the failing policies webhook is enabled.                                                                                                                                                   |
| &nbsp;&nbsp;&nbsp;&nbsp;destination_url                 | string  | body | The URL to deliver the webhook requests to.                                                                                                                                                               |
| &nbsp;&nbsp;&nbsp;&nbsp;policy_ids                      | array   | body | List of policy IDs to enable failing policies webhook.                                                                                                                                                    |
| &nbsp;&nbsp;host_status_webhook                    | object  | body | Host status webhook settings. |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_host_status_webhook | boolean | body | Whether or not the host status webhook is enabled. |
| &nbsp;&nbsp;&nbsp;&nbsp;destination_url            | string | body | The URL to deliver the webhook request to. |
| &nbsp;&nbsp;&nbsp;&nbsp;host_percentage            | integer | body | The minimum percentage of hosts that must fail to check in to Fleet in order to trigger the webhook request. |
| &nbsp;&nbsp;&nbsp;&nbsp;days_count | integer | body | The minimum number of days that the configured `host_percentage` must fail to check in to Fleet in order to trigger the webhook request. |
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
| &nbsp;&nbsp;macos_updates                               | object  | body | macOS updates settings.                                                                                                                                                                                   |
| &nbsp;&nbsp;&nbsp;&nbsp;minimum_version                 | string  | body | Hosts that belong to this team and are enrolled into Fleet's MDM will be prompted to update when their OS is below this version.                                                                           |
| &nbsp;&nbsp;&nbsp;&nbsp;deadline                        | string  | body | Hosts that belong to this team and are enrolled into Fleet's MDM will be forced to update their OS after this deadline (noon local time for hosts already on macOS 14 or above, 20:00 UTC for hosts on earlier macOS versions).                                                                    |
| &nbsp;&nbsp;ios_updates                               | object  | body | iOS updates settings.                                                                                                                                                                                   |
| &nbsp;&nbsp;&nbsp;&nbsp;minimum_version                 | string  | body | Hosts that belong to this team will be prompted to update when their OS is below this version.                                                                            |
| &nbsp;&nbsp;&nbsp;&nbsp;deadline                        | string  | body | Hosts that belong to this team will be forced to update their OS after this deadline (noon local time).                                                                    |
| &nbsp;&nbsp;ipados_updates                               | object  | body | iPadOS updates settings.                                                                                                                                                                                   |
| &nbsp;&nbsp;&nbsp;&nbsp;minimum_version                 | string  | body | Hosts that belong to this team will be prompted to update when their OS is below this version.                                                                            |
| &nbsp;&nbsp;&nbsp;&nbsp;deadline                        | string  | body | Hosts that belong to this team will be forced to update their OS after this deadline (noon local time).                                                                    |
| &nbsp;&nbsp;windows_updates                             | object  | body | Windows updates settings.                                                                                                                                                                                   |
| &nbsp;&nbsp;&nbsp;&nbsp;deadline_days                   | integer | body | Hosts that belong to this team and are enrolled into Fleet's MDM will have this number of days before updates are installed on Windows.                                                                   |
| &nbsp;&nbsp;&nbsp;&nbsp;grace_period_days               | integer | body | Hosts that belong to this team and are enrolled into Fleet's MDM will have this number of days before Windows restarts to install updates.                                                                    |
| &nbsp;&nbsp;macos_settings                              | object  | body | macOS-specific settings.                                                                                                                                                                                  |
| &nbsp;&nbsp;&nbsp;&nbsp;custom_settings                 | array    | body | Only intended to be used by [Fleet's YAML](https://fleetdm.com/docs/configuration/yaml-files). To add macOS configuration profiles using Fleet's API, use the [Add configuration profile endpoint](https://fleetdm.com/docs/rest-api/rest-api#add-custom-os-setting-configuration-profile) instead.                                                                                                                                      |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_disk_encryption          | boolean | body | Hosts that belong to this team will have disk encryption enabled if set to true.                                                                                        |
| &nbsp;&nbsp;windows_settings                            | object  | body | Windows-specific settings.                                                                                                                                                                                |
| &nbsp;&nbsp;&nbsp;&nbsp;custom_settings                 | array    | body | Only intended to be used by [Fleet's YAML](https://fleetdm.com/docs/configuration/yaml-files). To add Windows configuration profiles using Fleet's API, use the [Add configuration profile endpoint](https://fleetdm.com/docs/rest-api/rest-api#add-custom-os-setting-configuration-profile) instead.                                                                                                                             |
| &nbsp;&nbsp;macos_setup                                 | object  | body | Setup for automatic MDM enrollment of macOS hosts.                                                                                                                                                      |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_end_user_authentication  | boolean | body | If set to true, end user authentication will be required during automatic MDM enrollment of new macOS hosts. Settings for your IdP provider must also be [configured](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#end-user-authentication-and-eula).                                                                                      |
| integrations                                            | object  | body | Integration settings for this team.                                                                                                                                                                   |
| &nbsp;&nbsp;google_calendar                             | object  | body | Google Calendar integration settings.                                                                                                                                                                        |
| &nbsp;&nbsp;&nbsp;&nbsp;enable_calendar_events          | boolean | body | Whether or not calendar events are enabled for this team.                                                                                                                                                  |
| &nbsp;&nbsp;&nbsp;&nbsp;webhook_url                     | string | body | The URL to send a request to during calendar events, to trigger auto-remediation.                |
| host_expiry_settings                                    | object  | body | Host expiry settings for the team.                                                                                                                                                                         |
| &nbsp;&nbsp;host_expiry_enabled                         | boolean | body | When enabled, allows automatic cleanup of hosts that have not communicated with Fleet in some number of days. When disabled, defaults to the global setting.                                               |
| &nbsp;&nbsp;host_expiry_window                          | integer | body | If a host has not communicated with Fleet in the specified number of days, it will be removed.                                                                                                             |

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

### Add users to a team

_Available in Fleet Premium_

`PATCH /api/v1/fleet/teams/:id/users`

#### Parameters

| Name             | Type    | In   | Description                                  |
|------------------|---------|------|----------------------------------------------|
| id               | integer | path | **Required.** The desired team's ID.         |
| users            | string  | body | Array of users to add.                       |
| &nbsp;&nbsp;id   | integer | body | The id of the user.                          |
| &nbsp;&nbsp;role | string  | body | The team role that the user will be granted. Options are: "admin", "maintainer", "observer", "observer_plus", and "gitops". |

#### Example

`PATCH /api/v1/fleet/teams/1/users`

##### Request body

```json
{
  "users": [
    {
      "id": 1,
      "role": "admin"
    },
    {
      "id": 17,
      "role": "observer"
    }
  ]
}
```

##### Default response

`Status: 200`

```json
{
  "team": {
    "name": "Workstations",
    "id": 1,
    "user_count": 2,
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
      "enable_disk_encryption": true,
      "macos_updates": {
        "minimum_version": "12.3.1",
        "deadline": "2022-01-01"
      },
      "windows_updates": {
        "deadline_days": 5,
        "grace_period_days": 1
      },
      "macos_settings": {
        "custom_settings": [
          {
           "path": "path/to/profile1.mobileconfig",
           "labels": ["Label 1", "Label 2"]
          }
        ]
      },
      "windows_settings": {
        "custom_settings": [
          {
           "path": "path/to/profile2.xml",
           "labels": ["Label 3", "Label 4"]
          }
        ],
      },
      "macos_setup": {
        "bootstrap_package": "",
        "enable_end_user_authentication": false,
        "macos_setup_assistant": "path/to/config.json"
      }
    },
    "users": [
      {
        "created_at": "0001-01-01T00:00:00Z",
        "updated_at": "0001-01-01T00:00:00Z",
        "id": 1,
        "name": "Example User1",
        "email": "user1@example.com",
        "force_password_reset": false,
        "gravatar_url": "",
        "sso_enabled": false,
        "global_role": null,
        "api_only": false,
        "teams": null,
        "role": "admin"
      },
      {
        "created_at": "0001-01-01T00:00:00Z",
        "updated_at": "0001-01-01T00:00:00Z",
        "id": 17,
        "name": "Example User2",
        "email": "user2@example.com",
        "force_password_reset": false,
        "gravatar_url": "",
        "sso_enabled": false,
        "global_role": null,
        "api_only": false,
        "teams": null,
        "role": "observer"
      }
    ]
  }
}
```

### Modify team's agent options

_Available in Fleet Premium_

`POST /api/v1/fleet/teams/:id/agent_options`

#### Parameters

| Name                             | Type    | In    | Description                                                                                                                                                  |
| ---                              | ---     | ---   | ---                                                                                                                                                          |
| id                               | integer | path  | **Required.** The desired team's ID.                                                                                                                         |
| force                            | boolean | query | Force apply the options even if there are validation errors.                                                                                                 |
| dry_run                          | boolean | query | Validate the options and return any validation errors, but do not apply the changes.                                                                         |
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

`DELETE /api/v1/fleet/teams/:id`

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

| Name  | Type  | In   | Description                              |
| ----- | ----- | ---- | ---------------------------------------- |
| array | array | body | **Required** list of items to translate. |

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
- [Create invite](#create-invite)
- [List invites](#list-invites)
- [Delete invite](#delete-invite)
- [Verify invite](#verify-invite)
- [Modify invite](#modify-invite)

The Fleet server exposes API endpoints that handles common user management operations, including managing emailed invites to new users. All of these endpoints require prior authentication, so you'll need to log in before calling any of the endpoints documented below.

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
| team_id         | integer | query | _Available in Fleet Premium_. Filters the users to only include users in the specified team.                                   |

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
      "mfa_enabled": false,
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

#### Example

`POST /api/v1/fleet/users`

##### Request query parameters

```json
{
  "email": "janedoe@example.com",
  "invite_token": "SjdReDNuZW5jd3dCbTJtQTQ5WjJTc2txWWlEcGpiM3c=",
  "name": "janedoe",
  "password": "test-123",
  "password_confirmation": "test-123"
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
    "mfa_enabled": false,
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
| mfa_enabled | boolean | body | _Available in Fleet Premium._ Whether or not the user must click a magic link emailed to them to log in, after they successfully enter their username and password. Incompatible with SSO and API-only users. |
| api_only    | boolean | body | User is an "API-only" user (cannot use web UI) if true.                                                                                                                                                                                                                                                                                                  |
| global_role | string | body | The role assigned to the user. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). In Fleet 4.30.0 and 4.31.0, the `observer_plus` and `gitops` roles were introduced respectively. If `global_role` is specified, `teams` cannot be specified. For more information, see [manage access](https://fleetdm.com/docs/using-fleet/manage-access).                                                                                                                                                                        |
| admin_forced_password_reset    | boolean | body | Sets whether the user will be forced to reset its password upon first login (default=true) |
| teams                          | array   | body | _Available in Fleet Premium_. The teams and respective roles assigned to the user. Should contain an array of objects in which each object includes the team's `id` and the user's `role` on each team. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). In Fleet 4.30.0 and 4.31.0, the `observer_plus` and `gitops` roles were introduced respectively. If `teams` is specified, `global_role` cannot be specified. For more information, see [manage access](https://fleetdm.com/docs/using-fleet/manage-access). |

#### Example

`POST /api/v1/fleet/users/admin`

##### Request body

```json
{
  "name": "Jane Doe",
  "email": "janedoe@example.com",
  "password": "test-123",
  "api_only": true,
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
    "mfa_enabled": false,
    "api_only": true,
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
  },
  "token": "{API key}"
}
```

> Note: The new user's `token` (API key) is only included in the response after creating an api-only user (`api_only: true`).

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

`GET /api/v1/fleet/users/:id`

#### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The user's id. |

#### Example

`GET /api/v1/fleet/users/2`

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
    "mfa_enabled": false,
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

`PATCH /api/v1/fleet/users/:id`

#### Parameters

| Name        | Type    | In   | Description                                                                                                                                                                                                                                                                                                                                              |
| ----------- | ------- | ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| id          | integer | path | **Required**. The user's id.                                                                                                                                                                                                                                                                                                                             |
| name        | string  | body | The user's name.                                                                                                                                                                                                                                                                                                                                         |
| position    | string  | body | The user's position.                                                                                                                                                                                                                                                                                                                                     |
| email       | string  | body | The user's email.                                                                                                                                                                                                                                                                                                                                        |
| sso_enabled | boolean | body | Whether or not SSO is enabled for the user.                                                                                                                                                                                                                                                                                                              |
| mfa_enabled | boolean | body | _Available in Fleet Premium._ Whether or not the user must click a magic link emailed to them to log in, after they successfully enter their username and password. Incompatible with SSO and API-only users. |
| api_only    | boolean | body | User is an "API-only" user (cannot use web UI) if true.                                                                                                                                                                                                                                                                                                  |
| password    | string  | body | The user's current password, required to change the user's own email or password (not required for an admin to modify another user).                                                                                                                                                                                                                     |
| new_password| string  | body | The user's new password. |
| global_role | string  | body | The role assigned to the user. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). If `global_role` is specified, `teams` cannot be specified.                                                                                                                                                                         |
| teams       | array   | body | _Available in Fleet Premium_. The teams and respective roles assigned to the user. Should contain an array of objects in which each object includes the team's `id` and the user's `role` on each team. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). If `teams` is specified, `global_role` cannot be specified. |

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
    "mfa_enabled": false,
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
    "mfa_enabled": false,
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

`DELETE /api/v1/fleet/users/:id`

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

`POST /api/v1/fleet/users/:id/require_password_reset`

#### Parameters

| Name  | Type    | In   | Description                                                                                    |
| ----- | ------- | ---- | ---------------------------------------------------------------------------------------------- |
| id    | integer | path | **Required**. The user's id.                                                                   |
| require | boolean | body | Whether or not the user is required to reset their password during the next attempt to log in. |

#### Example

`POST /api/v1/fleet/users/123/require_password_reset`

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
    "mfa_enabled": false,
    "sso_enabled": false,
    "global_role": "observer",
    "teams": []
  }
}
```

### List a user's sessions

Returns a list of the user's sessions in Fleet.

`GET /api/v1/fleet/users/:id/sessions`

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

`DELETE /api/v1/fleet/users/:id/sessions`

#### Parameters

| Name | Type    | In   | Description                               |
| ---- | ------- | ---- | ----------------------------------------- |
| id   | integer | path | **Required**. The ID of the desired user. |

#### Example

`DELETE /api/v1/fleet/users/1/sessions`

##### Default response

`Status: 200`

### Create invite

`POST /api/v1/fleet/invites`

#### Parameters

| Name        | Type    | In   | Description                                                                                                                                           |
| ----------- | ------- | ---- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| global_role | string  | body | Role the user will be granted. Either a global role is needed, or a team role.                                                                        |
| email       | string  | body | **Required.** The email of the invited user. This email will receive the invitation link.                                                             |
| name        | string  | body | **Required.** The name of the invited user.                                                                                                           |
| sso_enabled | boolean | body | **Required.** Whether or not SSO will be enabled for the invited user.                                                                                |
| mfa_enabled | boolean | body | _Available in Fleet Premium._ Whether or not the invited user must click a magic link emailed to them to log in, after they successfully enter their username and password. Users can have SSO or MFA enabled, but not both. |
| teams       | array   | body | _Available in Fleet Premium_. A list of the teams the user is a member of. Each item includes the team's ID and the user's role in the specified team. |

#### Example

##### Request body

```json
{
  "email": "john_appleseed@example.com",
  "name": "John",
  "sso_enabled": false,
  "mfa_enabled": false,
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
    "mfa_enabled": false,
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
      "mfa_enabled": false,
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
      "mfa_enabled": false,
      "global_role": "admin",
      "teams": []
    }
  ]
}
```

### Delete invite

Delete the specified invite from Fleet.

`DELETE /api/v1/fleet/invites/:id`

#### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required.** The user's id. |

#### Example

`DELETE /api/v1/fleet/invites/123`

##### Default response

`Status: 200`


### Verify invite

Verify the specified invite.

`GET /api/v1/fleet/invites/:token`

#### Parameters

| Name  | Type    | In   | Description                            |
| ----- | ------- | ---- | -------------------------------------- |
| token | integer | path | **Required.** The user's invite token. |

#### Example

`GET /api/v1/fleet/invites/abcdef012456789`

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
    "mfa_enabled": false,
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

### Modify invite

`PATCH /api/v1/fleet/invites/:id`

#### Parameters

| Name        | Type    | In   | Description                                                                                                                                           |
| ----------- | ------- | ---- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| global_role | string  | body | Role the user will be granted. Either a global role is needed, or a team role.                                                                        |
| email       | string  | body | The email of the invited user. Updates on the email won't resend the invitation.                                                             |
| name        | string  | body | The name of the invited user.                                                                                                           |
| sso_enabled | boolean | body | Whether or not SSO will be enabled for the invited user.                                                                                |
| mfa_enabled | boolean | body | _Available in Fleet Premium._ Whether or not the invited user must click a magic link emailed to them to log in, after they successfully enter their username and password. Users can have SSO or MFA enabled, but not both. |
| teams       | array   | body | _Available in Fleet Premium_. A list of the teams the user is a member of. Each item includes the team's ID and the user's role in the specified team. |

#### Example

`PATCH /api/v1/fleet/invites/123`

##### Request body

```json
{
  "email": "john_appleseed@example.com",
  "name": "John",
  "sso_enabled": false,
  "mfa_enabled": false,
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
    "mfa_enabled": false,
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

`GET /debug/db/:key`

#### Parameters

None.

### Get profiling information

Returns runtime profiling data of the server in the format expected by `go tools pprof`. The responses are equivalent to those returned by the Go `http/pprof` package.

Valid keys are: `cmdline`, `profile`, `symbol` and `trace`.

`GET /debug/pprof/:key`

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
