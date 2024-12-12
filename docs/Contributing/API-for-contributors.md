# API for contributors

- [Packs](#packs)
- [Mobile device management (MDM)](#mobile-device-management-mdm)
- [Get or apply configuration files](#get-or-apply-configuration-files)
- [Live query](#live-query)
- [Trigger cron schedule](#trigger-cron-schedule)
- [Device-authenticated routes](#device-authenticated-routes)
- [Orbit-authenticated routes](#orbit-authenticated-routes)
- [Downloadable installers](#downloadable-installers)
- [Setup](#setup)
- [Scripts](#scripts)
- [Software](#software)

> These endpoints are used by the Fleet UI, Fleet Desktop, and `fleetctl` clients and frequently change to reflect current functionality.

This document includes the internal Fleet API routes that are helpful when developing or contributing to Fleet.

If you are interested in gathering information from Fleet in a production environment, please see the [public Fleet REST API documentation](https://fleetdm.com/docs/using-fleet/rest-api).

## Packs

Scheduling queries in Fleet is the best practice for collecting data from hosts. To learn how to schedule queries, [check out the docs here](https://fleetdm.com/docs/using-fleet/fleet-ui#schedule-a-query).

The API routes to control packs are supported for backwards compatibility.

- [Create pack](#create-pack)
- [Modify pack](#modify-pack)
- [Get pack](#get-pack)
- [List packs](#list-packs)
- [Delete pack](#delete-pack)
- [Delete pack by ID](#delete-pack-by-id)
- [Get scheduled queries in a pack](#get-scheduled-queries-in-a-pack)
- [Add scheduled query to a pack](#add-scheduled-query-to-a-pack)
- [Get scheduled query](#get-scheduled-query)
- [Modify scheduled query](#modify-scheduled-query)
- [Delete scheduled query](#delete-scheduled-query)

### Create pack

`POST /api/v1/fleet/packs`

#### Parameters

| Name        | Type   | In   | Description                                                             |
| ----------- | ------ | ---- | ----------------------------------------------------------------------- |
| name        | string | body | **Required**. The pack's name.                                          |
| description | string | body | The pack's description.                                                 |
| host_ids    | list   | body | A list containing the targeted host IDs.                                |
| label_ids   | list   | body | A list containing the targeted label's IDs.                             |
| team_ids    | list   | body | _Available in Fleet Premium_ A list containing the targeted teams' IDs. |

#### Example

`POST /api/v1/fleet/packs`

##### Request query parameters

```json
{
  "description": "Collects osquery data.",
  "host_ids": [],
  "label_ids": [6],
  "name": "query_pack_1"
}
```

##### Default response

`Status: 200`

```json
{
  "pack": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 17,
    "name": "query_pack_1",
    "description": "Collects osquery data.",
    "query_count": 0,
    "total_hosts_count": 223,
    "host_ids": [],
    "label_ids": [
      6
    ],
    "team_ids": []
  }
}
```

### Modify pack

`PATCH /api/v1/fleet/packs/{id}`

#### Parameters

| Name        | Type    | In   | Description                                                             |
| ----------- | ------- | ---- | ----------------------------------------------------------------------- |
| id          | integer | path | **Required.** The pack's id.                                            |
| name        | string  | body | The pack's name.                                                        |
| description | string  | body | The pack's description.                                                 |
| host_ids    | list    | body | A list containing the targeted host IDs.                                |
| label_ids   | list    | body | A list containing the targeted label's IDs.                             |
| team_ids    | list    | body | _Available in Fleet Premium_ A list containing the targeted teams' IDs. |

#### Example

`PATCH /api/v1/fleet/packs/{id}`

##### Request query parameters

```json
{
  "description": "MacOS hosts are targeted",
  "host_ids": [],
  "label_ids": [7]
}
```

##### Default response

`Status: 200`

```json
{
  "pack": {
    "created_at": "2021-01-25T22:32:45Z",
    "updated_at": "2021-01-25T22:32:45Z",
    "id": 17,
    "name": "Title2",
    "description": "MacOS hosts are targeted",
    "query_count": 0,
    "total_hosts_count": 110,
    "host_ids": [],
    "label_ids": [
      7
    ],
    "team_ids": []
  }
}
```

### Get pack

`GET /api/v1/fleet/packs/{id}`

#### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required.** The pack's id. |

#### Example

`GET /api/v1/fleet/packs/17`

##### Default response

`Status: 200`

```json
{
  "pack": {
    "created_at": "2021-01-25T22:32:45Z",
    "updated_at": "2021-01-25T22:32:45Z",
    "id": 17,
    "name": "Title2",
    "description": "MacOS hosts are targeted",
    "disabled": false,
    "type": null,
    "query_count": 0,
    "total_hosts_count": 110,
    "host_ids": [],
    "label_ids": [
      7
    ],
    "team_ids": []
  }
}
```

### List packs

`GET /api/v1/fleet/packs`

#### Parameters

| Name            | Type   | In    | Description                                                                                                                   |
| --------------- | ------ | ----- | ----------------------------------------------------------------------------------------------------------------------------- |
| order_key       | string | query | What to order results by. Can be any column in the packs table.                                                               |
| order_direction | string | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |

#### Example

`GET /api/v1/fleet/packs`

##### Default response

`Status: 200`

```json
{
  "packs": [
    {
      "created_at": "2021-01-05T21:13:04Z",
      "updated_at": "2021-01-07T19:12:54Z",
      "id": 1,
      "name": "pack_number_one",
      "description": "This pack has a description",
      "disabled": true,
      "query_count": 1,
      "total_hosts_count": 53,
      "host_ids": [],
      "label_ids": [
        8
      ],
      "team_ids": []
    },
    {
      "created_at": "2021-01-19T17:08:31Z",
      "updated_at": "2021-01-19T17:08:31Z",
      "id": 2,
      "name": "query_pack_2",
      "query_count": 5,
      "total_hosts_count": 223,
      "host_ids": [],
      "label_ids": [
        6
      ],
      "team_ids": []
    }
  ]
}
```

### Delete pack

Delete pack by name.

`DELETE /api/v1/fleet/packs/{name}`

#### Parameters

| Name | Type   | In   | Description                    |
| ---- | ------ | ---- | ------------------------------ |
| name | string | path | **Required.** The pack's name. |

#### Example

`DELETE /api/v1/fleet/packs/pack_number_one`

##### Default response

`Status: 200`


### Delete pack by ID

`DELETE /api/v1/fleet/packs/id/{id}`

#### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required.** The pack's ID. |

#### Example

`DELETE /api/v1/fleet/packs/id/1`

##### Default response

`Status: 200`


### Get scheduled queries in a pack

`GET /api/v1/fleet/packs/{id}/scheduled`

#### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required.** The pack's ID. |

#### Example

`GET /api/v1/fleet/packs/1/scheduled`

##### Default response

`Status: 200`

```json
{
  "scheduled": [
    {
      "created_at": "0001-01-01T00:00:00Z",
      "updated_at": "0001-01-01T00:00:00Z",
      "id": 49,
      "pack_id": 15,
      "name": "new_query",
      "query_id": 289,
      "query_name": "new_query",
      "query": "SELECT * FROM osquery_info",
      "interval": 456,
      "snapshot": false,
      "removed": true,
      "platform": "windows",
      "version": "4.6.0",
      "shard": null,
      "denylist": null
    },
    {
      "created_at": "0001-01-01T00:00:00Z",
      "updated_at": "0001-01-01T00:00:00Z",
      "id": 50,
      "pack_id": 15,
      "name": "new_title_for_my_query",
      "query_id": 288,
      "query_name": "new_title_for_my_query",
      "query": "SELECT * FROM osquery_info",
      "interval": 677,
      "snapshot": true,
      "removed": false,
      "platform": "windows",
      "version": "4.6.0",
      "shard": null,
      "denylist": null
    },
    {
      "created_at": "0001-01-01T00:00:00Z",
      "updated_at": "0001-01-01T00:00:00Z",
      "id": 51,
      "pack_id": 15,
      "name": "osquery_info",
      "query_id": 22,
      "query_name": "osquery_info",
      "query": "SELECT i.*, p.resident_size, p.user_time, p.system_time, time.minutes AS counter FROM osquery_info i, processes p, time WHERE p.pid = i.pid;",
      "interval": 6667,
      "snapshot": true,
      "removed": false,
      "platform": "windows",
      "version": "4.6.0",
      "shard": null,
      "denylist": null
    }
  ]
}
```

### Add scheduled query to a pack

`POST /api/v1/fleet/schedule`

#### Parameters

| Name     | Type    | In   | Description                                                                                                   |
| -------- | ------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| pack_id  | integer | body | **Required.** The pack's ID.                                                                                  |
| query_id | integer | body | **Required.** The query's ID.                                                                                 |
| interval | integer | body | **Required.** The amount of time, in seconds, the query waits before running.                                 |
| snapshot | boolean | body | **Required.** Whether the queries logs show everything in its current state.                                  |
| removed  | boolean | body | **Required.** Whether "removed" actions should be logged.                                                     |
| platform | string  | body | The computer platform where this query will run (other platforms ignored). Empty value runs on all platforms. |
| shard    | integer | body | Restrict this query to a percentage (1-100) of target hosts.                                                  |
| version  | string  | body | The minimum required osqueryd version installed on a host.                                                    |

#### Example

`POST /api/v1/fleet/schedule`

#### Request body

```json
{
  "interval": 120,
  "pack_id": 15,
  "query_id": 23,
  "removed": true,
  "shard": null,
  "snapshot": false,
  "version": "4.5.0",
  "platform": "windows"
}
```

##### Default response

`Status: 200`

```json
{
  "scheduled": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 56,
    "pack_id": 17,
    "name": "osquery_events",
    "query_id": 23,
    "query_name": "osquery_events",
    "query": "SELECT name, publisher, type, subscriptions, events, active FROM osquery_events;",
    "interval": 120,
    "snapshot": false,
    "removed": true,
    "platform": "windows",
    "version": "4.5.0",
    "shard": 10
  }
}
```

### Get scheduled query

`GET /api/v1/fleet/schedule/{id}`

#### Parameters

| Name | Type    | In   | Description                             |
| ---- | ------- | ---- | --------------------------------------- |
| id   | integer | path | **Required.** The scheduled query's ID. |

#### Example

`GET /api/v1/fleet/schedule/56`

##### Default response

`Status: 200`

```json
{
  "scheduled": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 56,
    "pack_id": 17,
    "name": "osquery_events",
    "query_id": 23,
    "query_name": "osquery_events",
    "query": "SELECT name, publisher, type, subscriptions, events, active FROM osquery_events;",
    "interval": 120,
    "snapshot": false,
    "removed": true,
    "platform": "windows",
    "version": "4.5.0",
    "shard": 10,
    "denylist": null
  }
}
```

### Modify scheduled query

`PATCH /api/v1/fleet/schedule/{id}`

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

`PATCH /api/v1/fleet/schedule/56`

#### Request body

```json
{
  "platform": ""
}
```

##### Default response

`Status: 200`

```json
{
  "scheduled": {
    "created_at": "2021-01-28T19:40:04Z",
    "updated_at": "2021-01-28T19:40:04Z",
    "id": 56,
    "pack_id": 17,
    "name": "osquery_events",
    "query_id": 23,
    "query_name": "osquery_events",
    "query": "SELECT name, publisher, type, subscriptions, events, active FROM osquery_events;",
    "interval": 120,
    "snapshot": false,
    "removed": true,
    "platform": "",
    "version": "4.5.0",
    "shard": 10
  }
}
```

### Delete scheduled query

`DELETE /api/v1/fleet/schedule/{id}`

#### Parameters

| Name | Type    | In   | Description                             |
| ---- | ------- | ---- | --------------------------------------- |
| id   | integer | path | **Required.** The scheduled query's ID. |

#### Example

`DELETE /api/v1/fleet/schedule/56`

##### Default response

`Status: 200`

---

## Mobile device management (MDM)

> Only Fleet MDM specific endpoints are located within the root /mdm/ path.

The MDM endpoints exist to support the related command-line interface sub-commands of `fleetctl`, such as `fleetctl generate mdm-apple` and `fleetctl get mdm-apple`, as well as the Fleet UI.

- [Generate Apple Business Manager public key (ADE)](#generate-apple-business-manager-public-key-ade)
- [Request Certificate Signing Request (CSR)](#request-certificate-signing-request-csr)
- [Upload APNS certificate](#upload-apns-certificate)
- [Add ABM token](#add-abm-token)
- [Count ABM tokens](#count-abm-tokens)
- [Turn off Apple MDM](#turn-off-apple-mdm)
- [Update ABM token's teams](#update-abm-tokens-teams)
- [Renew ABM token](#renew-abm-token)
- [Delete ABM token](#delete-abm-token)
- [Add VPP token](#add-VPP-token)
- [Update VPP token's teams](#update-vpp-tokens-teams)
- [Renew VPP token](#renew-vpp-token)
- [Delete VPP token](#delete-vpp-token)
- [Batch-apply MDM custom settings](#batch-apply-mdm-custom-settings)
- [Batch-apply packages](#batch-apply-packages)
- [Batch-apply App Store apps](#batch-apply-app-store-apps)
- [Get token to download package](#get-token-to-download-package)
- [Download package using a token](#download-package-using-a-token)
- [Initiate SSO during DEP enrollment](#initiate-sso-during-dep-enrollment)
- [Complete SSO during DEP enrollment](#complete-sso-during-dep-enrollment)
- [Over the air enrollment](#over-the-air-enrollment)
- [Preassign profiles to devices](#preassign-profiles-to-devices)
- [Match preassigned profiles](#match-preassigned-profiles)
- [Get FileVault statistics](#get-filevault-statistics)
- [Upload VPP content token](#upload-vpp-content-token)
- [Disable VPP](#disable-vpp)
- [SCEP proxy](#scep-proxy)


### Generate Apple Business Manager public key (ADE)

`GET /api/v1/fleet/mdm/apple/abm_public_key`

#### Example

`GET /api/v1/fleet/mdm/apple/abm_public_key`

##### Default response

`Status: 200`

```json
{
    "public_key": "23K9LCBGG26gc2AjcmV9Kz="
}
```

### Request Certificate Signing Request (CSR)

`GET /api/v1/fleet/mdm/apple/request_csr`

#### Example

`GET /api/v1/fleet/mdm/apple/request_csr`

##### Default response

```
Status: 200
```

```json
{
    "csr": "lKT5LCBJJ20gc2VjcmV0Cg="
}
```


### Upload APNS certificate

`POST /api/v1/fleet/mdm/apple/apns_certificate`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| certificate | file | form | *Required* The file conataining the APNS certificate (.pem) |

#### Example

`POST /api/v1/fleet/mdm/apple/apns_certificate`

##### Request header

```http
Content-Length: 850
Content-Type: multipart/form-data; boundary=------------------------f02md47480und42y
```

##### Request body

```http
--------------------------f02md47480und42y
Content-Disposition: form-data; name="certificate"; filename="apns_cert.pem"
Content-Type: application/octet-stream

<CERTIFICATE_DATA>

--------------------------f02md47480und42y
```

##### Default response

`Status: 200`

### Add ABM token

`POST /api/v1/fleet/abm_tokens`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| token | file | form | *Required* The file containing the token (.p7m) from Apple Business Manager |

#### Example

`POST /api/v1/fleet/abm_tokens`

##### Request header

```http
Content-Length: 850
Content-Type: multipart/form-data; boundary=------------------------f02md47480und42y
```

##### Request body

```http
--------------------------f02md47480und42y
Content-Disposition: form-data; name="token"; filename="server_token_abm.p7m"
Content-Type: application/octet-stream

<TOKEN_DATA>

--------------------------f02md47480und42y
```

##### Default response

`Status: 200`

```json
"abm_token": {
  "id": 1,
  "apple_id": "apple@example.com",
  "org_name": "Fleet Device Management Inc.",
  "mdm_server_url": "https://example.com/mdm/apple/mdm",
  "renew_date": "2024-10-20T00:00:00Z",
  "terms_expired": false,
  "macos_team": null,
  "ios_team": null,
  "ipados_team": null
}
```

### Count ABM tokens

`GET /api/v1/fleet/abm_tokens/count`

Get the number of ABM tokens on the Fleet server.

#### Parameters

None.

#### Example

`GET /api/v1/fleet/abm_tokens/count`

##### Default response

`Status: 200`

```json
{
  "count": 1
}
```

### Turn off Apple MDM

`DELETE /api/v1/fleet/mdm/apple/apns_certificate`

#### Example

`DELETE /api/v1/fleet/mdm/apple/apns_certificate`

##### Default response

`Status: 204`

### Update ABM token's teams

`PATCH /api/v1/fleet/abm_tokens/:id/teams`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| id | integer | path | *Required* The ABM token's ID |
| macos_team_id | integer | body | macOS hosts are automatically added to this team in Fleet when they appear in Apple Business Manager. If not specified, defaults to "No team" |
| ios_team_id | integer | body | iOS hosts are automatically added to this team in Fleet when they appear in Apple Business Manager. If not specified, defaults to "No team" |
| ipados_team_id | integer | body | iPadOS hosts are automatically added to this team in Fleet when they appear in Apple Business Manager. If not specified, defaults to "No team" |

#### Example

`PATCH /api/v1/fleet/abm_tokens/1/teams`

##### Request body

```json
{
  "macos_team_id": 1,
  "ios_team_id": 2,
  "ipados_team_id": 3
}
```

##### Default response

`Status: 200`

```json
"abm_token": {
  "id": 1,
  "apple_id": "apple@example.com",
  "org_name": "Fleet Device Management Inc.",
  "mdm_server_url": "https://example.com/mdm/apple/mdm",
  "renew_date": "2024-11-29T00:00:00Z",
  "terms_expired": false,
  "macos_team": 1,
  "ios_team": 2,
  "ipados_team": 3
}
```

### Renew ABM token

`PATCH /api/v1/fleet/abm_tokens/:id/renew`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| id | integer | path | *Required* The ABM token's ID |

#### Example

`PATCH /api/v1/fleet/abm_tokens/1/renew`

##### Request header

```http
Content-Length: 850
Content-Type: multipart/form-data; boundary=------------------------f02md47480und42y
```

##### Request body

```http
--------------------------f02md47480und42y
Content-Disposition: form-data; name="token"; filename="server_token_abm.p7m"
Content-Type: application/octet-stream

<TOKEN_DATA>

--------------------------f02md47480und42y
```

##### Default response

`Status: 200`

```json
"abm_token": {
  "id": 1,
  "apple_id": "apple@example.com",
  "org_name": "Fleet Device Management Inc.",
  "mdm_server_url": "https://example.com/mdm/apple/mdm",
  "renew_date": "2025-10-20T00:00:00Z",
  "terms_expired": false,
  "macos_team": null,
  "ios_team": null,
  "ipados_team": null
}
```

### Delete ABM token

`DELETE /api/v1/fleet/abm_tokens/:id`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| id | integer | path | *Required* The ABM token's ID |

#### Example

`DELETE /api/v1/fleet/abm_tokens/1`

##### Default response

`Status: 204`

### Add VPP token

`POST /api/v1/fleet/vpp_tokens`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| token | file | form | *Required* The file containing the content token (.vpptoken) from Apple Business Manager |

#### Example

`POST /api/v1/fleet/vpp_tokens`

##### Request header

```http
Content-Length: 850
Content-Type: multipart/form-data; boundary=------------------------f02md47480und42y
```

##### Request body

```http
--------------------------f02md47480und42y
Content-Disposition: form-data; name="token"; filename="sToken_for_Acme.vpptoken"
Content-Type: application/octet-stream
<TOKEN_DATA>
--------------------------f02md47480und42y
```

##### Default response

`Status: 200`

```json
"vpp_token": {
  "id": 1,
  "org_name": "Fleet Device Management Inc.",
  "location": "https://example.com/mdm/apple/mdm",
  "renew_date": "2024-10-20T00:00:00Z",
  "terms_expired": false,
  "teams": null
}
```

### Update VPP token's teams

`PATCH /api/v1/fleet/vpp_tokens/:id/teams`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| id | integer | path | *Required* The ABM token's ID |
| team_ids | list | body | If you choose specific teams, App Store apps in this VPP account will only be available to install on hosts in these teams. If not specified, defaults to all teams. |

#### Example

`PATCH /api/v1/fleet/vpp_tokens/1/teams`

##### Request body

```json
{
  "team_ids": [1, 2, 3]
}
```

##### Default response

`Status: 200`

```json
"vpp_token": {
  "id": 1,
  "org_name": "Fleet Device Management Inc.",
  "location": "https://example.com/mdm/apple/mdm",
  "renew_date": "2024-10-20T00:00:00Z",
  "terms_expired": false,
  "teams": [
    {
      "team_id": 1,
      "name": "Team 1"
    },
    {
      "team_id": 2,
      "name": "Team 2"
    },
    {
      "team_id": 2,
      "name": "Team 3"
    },
  ]
}
```

### Renew VPP token

`PATCH /api/v1/fleet/vpp_tokens/:id/renew`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| id | integer | path | *Required* The VPP token's ID |

##### Request header

```http
Content-Length: 850
Content-Type: multipart/form-data; boundary=------------------------f02md47480und42y
```

##### Request body

```http
--------------------------f02md47480und42y
Content-Disposition: form-data; name="token"; filename="sToken_for_Acme.vpptoken"
Content-Type: application/octet-stream

<TOKEN_DATA>

--------------------------f02md47480und42y
```

##### Default response

`Status: 200`

```json
"vpp_token": {
  "id": 1,
  "org_name": "Fleet Device Management Inc.",
  "location": "https://example.com/mdm/apple/mdm",
  "renew_date": "2025-10-20T00:00:00Z",
  "terms_expired": false,
  "teams": [1, 2, 3]
}
```

### Delete VPP token

`DELETE /api/v1/fleet/vpp_token/:id`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| id | integer | path | *Required* The VPP token's ID |

#### Example

`DELETE /api/v1/fleet/vpp_tokens/1`

##### Default response

`Status: 204`

### Batch-apply MDM custom settings

`POST /api/v1/fleet/mdm/profiles/batch`

#### Parameters

| Name      | Type   | In    | Description                                                                                                                       |
| --------- | ------ | ----- | --------------------------------------------------------------------------------------------------------------------------------- |
| team_id   | number | query | _Available in Fleet Premium_ The team ID to apply the custom settings to. Only one of `team_name`/`team_id` can be provided.          |
| team_name | string | query | _Available in Fleet Premium_ The name of the team to apply the custom settings to. Only one of `team_name`/`team_id` can be provided. |
| dry_run   | bool   | query | Validate the provided profiles and return any validation errors, but do not apply the changes.                                    |
| profiles  | json   | body  | An array of objects, consisting of a `profile` base64-encoded .mobileconfig or JSON for macOS and XML (Windows) file, `labels_include_all`, `labels_include_any`, or `labels_exclude_any` array of strings (label names), and `name` display name (for Windows configuration profiles and macOS declaration profiles). |


If no team (id or name) is provided, the profiles are applied for all hosts (for _Fleet Free_) or for hosts that are not assigned to any team (for _Fleet Premium_). After the call, the provided list of `profiles` will be the active profiles for that team (or no team) - that is, any existing profile that is not part of that list will be removed, and an existing profile with the same payload identifier (macOS) as a new profile will be edited. If the list of provided `profiles` is empty, all profiles are removed for that team (or no team).

#### Example

`POST /api/v1/fleet/mdm/profiles/batch`

##### Default response

`204`

### Initiate SSO during DEP enrollment

This endpoint initiates the SSO flow, the response contains an URL that the client can use to redirect the user to initiate the SSO flow in the configured IdP.

`POST /api/v1/fleet/mdm/sso`

#### Parameters

None.

#### Example

`POST /api/v1/fleet/mdm/sso`

##### Default response

```json
{
  "url": "https://idp-provider.com/saml?SAMLRequest=...",
}
```

### Complete SSO during DEP enrollment

This is the callback endpoint that the identity provider will use to send security assertions to Fleet. This is where Fleet receives and processes the response from the identify provider.

`POST /api/v1/fleet/mdm/sso/callback`

#### Parameters

| Name         | Type   | In   | Description                                                 |
| ------------ | ------ | ---- | ----------------------------------------------------------- |
| SAMLResponse | string | body | **Required**. The SAML response from the identity provider. |

#### Example

`POST /api/v1/fleet/mdm/sso/callback`

##### Request body

```json
{
  "SAMLResponse": "<SAML response from IdP>"
}
```

##### Default response

`Status: 302`

If the credentials are valid, the server redirects the client to the Fleet UI. The URL contains the following query parameters that can be used to complete the DEP enrollment flow:

- `enrollment_reference` a reference that must be passed along with `profile_token` to the endpoint to download an enrollment profile.
- `profile_token` is a token that can be used to download an enrollment profile (.mobileconfig).
- `eula_token` (optional) if an EULA was uploaded, this contains a token that can be used to view the EULA document.

### Over the air enrollment

This endpoint handles over the air (OTA) MDM enrollments

`POST /api/v1/fleet/ota_enrollment`

#### Parameters

| Name                | Type   | In   | Description                                                                                                                                                                                                                                                                                        |
| ------------------- | ------ | ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| enroll_secret       | string | url  | **Required** Assigns the host to a team with a matching enroll secret                                                                                                                                                                                                                 |
| XML device response | XML    | body | **Required**. The XML response from the device. Fields are documented [here](https://developer.apple.com/library/archive/documentation/NetworkingInternet/Conceptual/iPhoneOTAConfiguration/ConfigurationProfileExamples/ConfigurationProfileExamples.html#//apple_ref/doc/uid/TP40009505-CH4-SW7) |

> Note: enroll secrets can contain special characters. Ensure any special characters are [properly escaped](https://developer.mozilla.org/en-US/docs/Glossary/Percent-encoding).

#### Example

`POST /api/v1/fleet/ota_enrollment?enroll_secret=0Z6IuKpKU4y7xl%2BZcrp2gPcMi1kKNs3p`

##### Default response

`Status: 200`

Per [the spec](https://developer.apple.com/library/archive/documentation/NetworkingInternet/Conceptual/iPhoneOTAConfiguration/Introduction/Introduction.html#//apple_ref/doc/uid/TP40009505-CH1-SW1), the response is different depending on the signature of the XML device response:

- If the body is signed with a certificate that can be validated by our root SCEP certificate, it returns an enrollment profile.
- Otherwise, it returns a SCEP payload.

### Preassign profiles to devices

_Available in Fleet Premium_

This endpoint stores a profile to be assigned to a host at some point in the future. The actual assignment happens when the [Match preassigned profiles](#match-preassigned-profiles) endpoint is called. The reason for this "pre-assign" step is to collect all profiles that are meant to be assigned to a host, and match the list of profiles to an existing team (or create one with that set of profiles if none exist) so that the host can be assigned to that team and inherit its list of profiles.

`POST /api/v1/fleet/mdm/apple/profiles/preassign`

#### Parameters

| Name                     | Type    | In   | Description                                                                                  |
| ------------             | ------- | ---- | -----------------------------------------------------------                                  |
| external_host_identifier | string  | body | **Required**. The identifier of the host as generated by the external service (e.g. Puppet). |
| host_uuid                | string  | body | **Required**. The UUID of the host.                                                          |
| profile                  | string  | body | **Required**. The base64-encoded .mobileconfig content of the MDM profile.                   |
| group                    | string  | body | The group label associated with that profile. This information is used to generate team names if they need to be created. |
| exclude                  | boolean | body | Whether to skip delivering the profile to this host. |

#### Example

`POST /api/v1/fleet/mdm/apple/profiles/preassign`

##### Request body

```json
{
  "external_host_identifier": "id-01234",
  "host_uuid": "c0532a64-bec2-4cf9-aa37-96fe47ead814",
  "profile": "<base64-encoded profile>",
  "group": "Workstations",
  "exclude": false
}
```

##### Default response

`Status: 204`

### Get FileVault statistics

_Available in Fleet Premium_

Get aggregate status counts of disk encryption enforced on macOS hosts.

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


### Match preassigned profiles

_Available in Fleet Premium_

This endpoint uses the profiles stored by the [Preassign profiles to devices](#preassign-profiles-to-devices) endpoint to match the set of profiles to an existing team if possible, creating one if none exists. It then assigns the host to that team so that it receives the associated profiles. It is meant to be called only once all desired profiles have been pre-assigned to the host.

`POST /api/v1/fleet/mdm/apple/profiles/match`

#### Parameters

| Name                     | Type   | In   | Description                                                                                  |
| ------------             | ------ | ---- | -----------------------------------------------------------                                  |
| external_host_identifier | string | body | **Required**. The identifier of the host as generated by the external service (e.g. Puppet). |

#### Example

`POST /api/v1/fleet/mdm/apple/profiles/match`

##### Request body

```json
{
  "external_host_identifier": "id-01234"
}
```

##### Default response

`Status: 204`

### Upload VPP content token

`POST /api/v1/fleet/mdm/apple/vpp_token`

#### Parameters

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| token | file | form | *Required* The file containing the content token (.vpptoken) from Apple Business Manager |

#### Example

`POST /api/v1/fleet/mdm/apple/vpp_token`

##### Request header

```http
Content-Length: 850
Content-Type: multipart/form-data; boundary=------------------------f02md47480und42y
```

##### Request body

```http
--------------------------f02md47480und42y
Content-Disposition: form-data; name="token"; filename="sToken_for_Acme.vpptoken"
Content-Type: application/octet-stream
<TOKEN_DATA>
--------------------------f02md47480und42y
```

##### Default response

`Status: 200`


### Disable VPP

`DELETE /api/v1/fleet/mdm/apple/vpp_token`

#### Example

`DELETE /api/v1/fleet/mdm/apple/vpp_token`

##### Default response

`Status: 204`

### SCEP proxy

`/mdm/scep/proxy/{identifier}`

This endpoint is used to proxy SCEP requests to the configured SCEP server. It uses the [SCEP protocol](https://datatracker.ietf.org/doc/html/rfc8894). The `identifier` is in the format `hostUUID,profileUUID`.

## Get or apply configuration files

These API routes are used by the `fleetctl` CLI tool. Users can manage Fleet with `fleetctl` and [configuration files in YAML syntax](https://fleetdm.com/docs/using-fleet/configuration-files/).

- [Get queries](#get-queries)
- [Get query](#get-query)
- [Apply queries](#apply-queries)
- [Apply policies](#apply-policies)
- [Get packs](#get-packs)
- [Apply packs](#apply-packs)
- [Get pack by name](#get-pack-by-name)
- [Apply team](#apply-team)
- [Apply labels](#apply-labels)
- [Get labels](#get-labels)
- [Get label](#get-label)
- [Get enroll secrets](#get-enroll-secrets)
- [Modify enroll secrets](#modify-enroll-secrets)

### Get queries

Returns a list of all queries in the Fleet instance. Each item returned includes the name, description, and SQL of the query.

`GET /api/v1/fleet/spec/queries`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/spec/queries`

##### Default response

`Status: 200`

```json
{
  "specs": [
    {
      "name": "query1",
      "description": "query",
      "query": "SELECT * FROM osquery_info"
    },
    {
      "name": "osquery_schedule",
      "description": "Report performance stats for each file in the query schedule.",
      "query": "SELECT name, interval, executions, output_size, wall_time, (user_time/executions) AS avg_user_time, (system_time/executions) AS avg_system_time, average_memory, last_executed FROM osquery_schedule;"
    }
]
}
```

### Get query

Returns the name, description, and SQL of the query specified by name.

`GET /api/v1/fleet/spec/queries/{name}`

#### Parameters

| Name | Type   | In   | Description                          |
| ---- | ------ | ---- | ------------------------------------ |
| name | string | path | **Required.** The name of the query. |

#### Example

`GET /api/v1/fleet/spec/queries/query1`

##### Default response

`Status: 200`

```json
{
  "specs": {
    "name": "query1",
    "description": "query",
    "query": "SELECT * FROM osquery_info"
  }
}
```

### Apply queries

Creates and/or modifies the queries included in the list. To modify an existing query, the name of the query must already be used by an existing query. If a query with the specified name doesn't exist in Fleet, a new query will be created.

If a query field is not specified in the "spec" then its default value depending on its type will be assumed, e.g. if `interval` is not set then `0` will be assumed, if `discard_data` is omitted then `false` will be assumed, etc.

`POST /api/v1/fleet/spec/queries`

#### Parameters

| Name  | Type | In   | Description                                                      |
| ----- | ---- | ---- | ---------------------------------------------------------------- |
| specs | list | body | **Required.** The list of the queries to be created or modified. |

For more information about the query fields, please refer to the [Create query endpoint](https://fleetdm.com/docs/using-fleet/rest-api#create-query).

#### Example

`POST /api/v1/fleet/spec/queries`

##### Request body

```json
{
  "specs": [
    {
      "name": "new_query",
      "description": "This will be a new query because a query with the name 'new_query' doesn't exist in Fleet.",
      "query": "SELECT * FROM osquery_info"
    },
    {
      "name": "osquery_schedule",
      "description": "This queries description and SQL will be modified because a query with the name 'osquery_schedule' exists in Fleet.",
      "query": "SELECT * FROM osquery_info"
    }
  ]
}
```

##### Default response

`Status: 200`

### Get packs

Returns all packs in the Fleet instance.

`GET /api/v1/fleet/spec/packs`

#### Example

`GET /api/v1/fleet/spec/packs`

##### Default response

`Status: 200`

```json
{
  "specs": [
    {
      "id": 1,
      "name": "pack_1",
      "description": "Description",
      "disabled": false,
      "targets": {
        "labels": ["All Hosts"],
        "teams": null
      },
      "queries": [
        {
          "query": "new_query",
          "name": "new_query",
          "description": "",
          "interval": 456,
          "snapshot": false,
          "removed": true,
          "platform": "windows",
          "version": "4.5.0"
        },
        {
          "query": "new_title_for_my_query",
          "name": "new_title_for_my_query",
          "description": "",
          "interval": 677,
          "snapshot": true,
          "removed": false,
          "platform": "",
          "version": ""
        },
        {
          "query": "osquery_info",
          "name": "osquery_info",
          "description": "",
          "interval": 6667,
          "snapshot": true,
          "removed": false,
          "platform": "",
          "version": ""
        },
        {
          "query": "query1",
          "name": "query1",
          "description": "",
          "interval": 7767,
          "snapshot": false,
          "removed": true,
          "platform": "",
          "version": ""
        },
        {
          "query": "osquery_events",
          "name": "osquery_events",
          "description": "",
          "interval": 454,
          "snapshot": false,
          "removed": true,
          "platform": "",
          "version": ""
        },
        {
          "query": "osquery_events",
          "name": "osquery_events-1",
          "description": "",
          "interval": 120,
          "snapshot": false,
          "removed": true,
          "platform": "",
          "version": ""
        }
      ]
    },
    {
      "id": 2,
      "name": "pack_2",
      "disabled": false,
      "targets": {
        "labels": null,
        "teams": null
      },
      "queries": [
        {
          "query": "new_query",
          "name": "new_query",
          "description": "",
          "interval": 333,
          "snapshot": false,
          "removed": true,
          "platform": "windows",
          "version": "4.5.0",
          "shard": 10,
          "denylist": null
        }
      ]
    }
  ]
}
```

### Apply policies

Creates and/or modifies the policies included in the list. To modify an existing policy, the name of the policy included in the list must already be used by an existing policy. If a policy with the specified name doesn't exist in Fleet, a new policy will be created.

NOTE: when updating a policy, team and platform will be ignored.

`POST /api/v1/fleet/spec/policies`

#### Parameters

| Name  | Type | In   | Description                                                       |
| ----- | ---- | ---- | ----------------------------------------------------------------- |
| specs | list | body | **Required.** The list of the policies to be created or modified. |

#### Example

`POST /api/v1/fleet/spec/policies`

##### Request body

```json
{
  "specs": [
    {
      "name": "new policy",
      "description": "This will be a new policy because a policy with the name 'new policy' doesn't exist in Fleet.",
      "query": "SELECT * FROM osquery_info",
      "team": "No team",
      "resolution": "some resolution steps here",
      "critical": false
    },
    {
      "name": "Is FileVault enabled on macOS devices?",
      "query": "SELECT 1 FROM disk_encryption WHERE user_uuid IS NOT “” AND filevault_status = ‘on’ LIMIT 1;",
      "team": "Workstations",
      "description": "Checks to make sure that the FileVault feature is enabled on macOS devices.",
      "resolution": "Choose Apple menu > System Preferences, then click Security & Privacy. Click the FileVault tab. Click the Lock icon, then enter an administrator name and password. Click Turn On FileVault.",
      "platform": "darwin",
      "critical": true,
      "script_id": 123
    },
    {
      "name": "Is Adobe Acrobat installed and up to date?",
      "query": "SELECT 1 FROM apps WHERE name = 'Adobe Acrobat Reader.app' AND version_compare(bundle_short_version, '23.001.20687') >= 0;",
      "team": "Workstations",
      "description": "Checks to make sure that Adobe Acrobat is installed and up to date.",
      "platform": "darwin",
      "critical": false,
      "software_title_id": 12
    },
  ]
}
```

The fields `critical`, `script_id`, and `software_title_id` are available in Fleet Premium.

##### Default response

`Status: 200`

### Apply packs

Creates and/or modifies the packs included in the list.

`POST /api/v1/fleet/spec/packs`

#### Parameters

| Name  | Type | In   | Description                                                                                   |
| ----- | ---- | ---- | --------------------------------------------------------------------------------------------- |
| specs | list | body | **Required.** A list that includes the specs for each pack to be added to the Fleet instance. |

#### Example

`POST /api/v1/fleet/spec/packs`

##### Request body

```json
{
  "specs": [
    {
      "id": 1,
      "name": "pack_1",
      "description": "Description",
      "disabled": false,
      "targets": {
        "labels": ["All Hosts"],
        "teams": null
      },
      "queries": [
        {
          "query": "new_query",
          "name": "new_query",
          "description": "",
          "interval": 456,
          "snapshot": false,
          "removed": true
        },
        {
          "query": "new_title_for_my_query",
          "name": "new_title_for_my_query",
          "description": "",
          "interval": 677,
          "snapshot": true,
          "removed": false
        },
        {
          "query": "osquery_info",
          "name": "osquery_info",
          "description": "",
          "interval": 6667,
          "snapshot": true,
          "removed": false
        },
        {
          "query": "query1",
          "name": "query1",
          "description": "",
          "interval": 7767,
          "snapshot": false,
          "removed": true
        },
        {
          "query": "osquery_events",
          "name": "osquery_events",
          "description": "",
          "interval": 454,
          "snapshot": false,
          "removed": true
        },
        {
          "query": "osquery_events",
          "name": "osquery_events-1",
          "description": "",
          "interval": 120,
          "snapshot": false,
          "removed": true
        }
      ]
    },
    {
      "id": 2,
      "name": "pack_2",
      "disabled": false,
      "targets": {
        "labels": null,
        "teams": null
      },
      "queries": [
        {
          "query": "new_query",
          "name": "new_query",
          "description": "",
          "interval": 333,
          "snapshot": false,
          "removed": true,
          "platform": "windows"
        }
      ]
    }
  ]
}
```

##### Default response

`Status: 200`

### Get pack by name

Returns a pack.

`GET /api/v1/fleet/spec/packs/{name}`

#### Parameters

| Name | Type   | In   | Description                    |
| ---- | ------ | ---- | ------------------------------ |
| name | string | path | **Required.** The pack's name. |

#### Example

`GET /api/v1/fleet/spec/packs/pack_1`

##### Default response

`Status: 200`

```json
{
  "specs": {
    "id": 15,
    "name": "pack_1",
    "description": "Description",
    "disabled": false,
    "targets": {
      "labels": ["All Hosts"],
      "teams": null
    },
    "queries": [
      {
        "query": "new_title_for_my_query",
        "name": "new_title_for_my_query",
        "description": "",
        "interval": 677,
        "snapshot": true,
        "removed": false,
        "platform": "",
        "version": ""
      },
      {
        "query": "osquery_info",
        "name": "osquery_info",
        "description": "",
        "interval": 6667,
        "snapshot": true,
        "removed": false,
        "platform": "",
        "version": ""
      },
      {
        "query": "query1",
        "name": "query1",
        "description": "",
        "interval": 7767,
        "snapshot": false,
        "removed": true,
        "platform": "",
        "version": ""
      },
      {
        "query": "osquery_events",
        "name": "osquery_events",
        "description": "",
        "interval": 454,
        "snapshot": false,
        "removed": true,
        "platform": "",
        "version": ""
      },
      {
        "query": "osquery_events",
        "name": "osquery_events-1",
        "description": "",
        "interval": 120,
        "snapshot": false,
        "removed": true,
        "platform": "",
        "version": ""
      }
    ]
  }
}
```

### Apply team

_Available in Fleet Premium_

If the `name` specified is associated with an existing team, this API route, completely replaces this team's existing `agent_options` and `secrets` with those that are specified.

If the `name` is not already associated with an existing team, this API route creates a new team with the specified `name`, `agent_options`, and `secrets`.

`POST /api/v1/fleet/spec/teams`

#### Parameters

| Name                                      | Type   | In    | Description                                                                                                                                                                                                                         |
| ----------------------------------------- | ------ | ----- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| name                                      | string | body  | **Required.** The team's name.                                                                                                                                                                                                      |
| agent_options                             | object | body  | The agent options spec that is applied to the hosts assigned to the specified to team. These agent options completely override the global agent options specified in the [`GET /api/v1/fleet/config API route`](#get-configuration) |
| features                                  | object | body  | The features that are applied to the hosts assigned to the specified to team. These features completely override the global features specified in the [`GET /api/v1/fleet/config API route`](#get-configuration)                    |
| secrets                                   | array   | body  | A list of plain text strings is used as the enroll secrets. Existing secrets are replaced with this list, or left unmodified if this list is empty. Note that there is a limit of 50 secrets allowed.                               |
| mdm                                       | object | body  | The team's MDM configuration options.                                                                                                                                                                                               |
| mdm.macos_updates                         | object | body  | The OS updates macOS configuration options for Nudge.                                                                                                                                                                               |
| mdm.macos_updates.minimum_version         | string | body  | The required minimum operating system version.                                                                                                                                                                                      |
| mdm.macos_updates.deadline                | string | body  | The required installation date for Nudge to enforce the operating system version.                                                                                                                                                   |
| mdm.macos_settings                        | object | body  | The macOS-specific MDM settings.                                                                                                                                                                                                    |
| mdm.macos_settings.custom_settings        | array   | body  | The list of objects consists of a `path` to .mobileconfig or JSON file and `labels_include_all`, `labels_include_any`, or `labels_exclude_any` list of label names.                                                                                                                                                         |
| mdm.windows_settings                        | object | body  | The Windows-specific MDM settings.                                                                                                                                                                                                    |
| mdm.windows_settings.custom_settings        | array   | body  | The list of objects consists of a `path` to XML files and `labels_include_all`, `labels_include_any`, or `labels_exclude_any` list of label names.                                                                                                                                                         |
| scripts                                   | array   | body  | A list of script files to add to this team so they can be executed at a later time.                                                                                                                                                 |
| software                                   | object   | body  | The team's software that will be available for install.  |
| software.packages                          | array   | body  | An array of objects with values below. |
| software.packages.url                      | string   | body  | URL to the software package (PKG, MSI, EXE or DEB). |
| software.packages.install_script           | string   | body  | Command that Fleet runs to install software. |
| software.packages.pre_install_query        | string   | body  | Condition query that determines if the install will proceed. |
| software.packages.post_install_script      | string   | body  | Script that runs after software install. |
| software.packages.uninstall_script       | string   | body  | Command that Fleet runs to uninstall software. |
| software.packages.self_service           | boolean   | body  | Condition query that determines if the install will proceed. |
| software.packages.labels_include_any     | array   | body  | Target hosts that have any label in the array. Only one of `labels_include_any` or `labels_exclude_any` can be included. If neither are included, all hosts are targeted. |
| software.packages.labels_exclude_any     | array   | body  | Target hosts that don't have any label in the array. Only one of `labels_include_any` or `labels_exclude_any` can be included. If neither are included, all hosts are targeted. |
| software.app_store_apps                   | array   | body  | An array of objects with values below. |
| software.app_store_apps.app_store_id      | string   | body  | ID of the App Store app. |
| software.app_store_apps.self_service      | boolean   | body  | Specifies whether or not end users can install self-service. |
| mdm.macos_settings.enable_disk_encryption | bool   | body  | Whether disk encryption should be enabled for hosts that belong to this team.                                                                                                                                                       |
| force                                     | bool   | query | Force apply the spec even if there are (ignorable) validation errors. Those are unknown keys and agent options-related validations.                                                                                                 |
| dry_run                                   | bool   | query | Validate the provided JSON for unknown keys and invalid value types and return any validation errors, but do not apply the changes.                                                                                                 |

#### Example

`POST /api/v1/fleet/spec/teams`

##### Request body

```json
{
  "specs": [
    {
      "name": "Client Platform Engineering",
      "features": {
        "enable_host_users": false,
        "enable_software_inventory": true,
        "additional_queries": {
          "foo": "SELECT * FROM bar;"
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
          "overrides": {}
        }
      },
      "secrets": [
        {
          "secret": "fTp52/twaxBU6gIi0J6PHp8o5Sm1k1kn"
        },
        {
          "secret": "bhD5kiX2J+KBgZSk118qO61ZIdX/v8On"
        }
      ],
      "mdm": {
        "macos_updates": {
          "minimum_version": "12.3.1",
          "deadline": "2023-12-01"
        },
        "macos_settings": {
          "custom_settings": [
            {
              "path": "path/to/profile1.mobileconfig"
              "labels_include_all": ["Label 1", "Label 2"]
            },
            {
              "path": "path/to/profile2.json"
              "labels_exclude_any": ["Label 3", "Label 4"]
            },
          ],
          "enable_disk_encryption": true
        },
        "windows_settings": {
          "custom_settings": [
            {
              "path": "path/to/profile3.xml"
              "labels_include_all": ["Label 1", "Label 2"]
            }
          ]
        }
      },
      "scripts": ["path/to/script.sh"],
      "software": { 
        "packages": [
          {
            "url": "https://cdn.zoom.us/prod/5.16.10.26186/x64/ZoomInstallerFull.msi",
            "pre_install_query": "SELECT 1 FROM macos_profiles WHERE uuid='c9f4f0d5-8426-4eb8-b61b-27c543c9d3db';",
            "post_install_script": "sudo /Applications/Falcon.app/Contents/Resources/falconctl license 0123456789ABCDEFGHIJKLMNOPQRSTUV-WX",
            "self_service": true,
          }
        ],
        "app_store_apps": [
          {
            "app_store_id": "12464567",
            "self_service": true
          }
        ]
      }  
    }
  ]
}
```

#### Default response

`Status: 200`

```json
{
  "team_ids_by_name": {
    "Client Platform Engineering": 123
  }
}
```

### Apply labels

Adds the supplied labels to Fleet. Each label requires the `name`, and `label_membership_type` properties.

If the `label_membership_type` is set to `dynamic`, the `query` property must also be specified with the value set to a query in SQL syntax.

If the `label_membership_type` is set to `manual`, the `hosts` property must also be specified with the value set to a list of hostnames.

`POST /api/v1/fleet/spec/labels`

#### Parameters

| Name  | Type | In   | Description                                                                                                   |
| ----- | ---- | ---- | ------------------------------------------------------------------------------------------------------------- |
| specs | list | path | A list of the label to apply. Each label requires the `name`, `query`, and `label_membership_type` properties |

#### Example

`POST /api/v1/fleet/spec/labels`

##### Request body

```json
{
  "specs": [
    {
      "name": "Ubuntu",
      "description": "Filters Ubuntu hosts",
      "query": "SELECT 1 FROM os_version WHERE platform = 'ubuntu';",
      "label_membership_type": "dynamic"
    },
    {
      "name": "local_machine",
      "description": "Includes only my local machine",
      "label_membership_type": "manual",
      "hosts": ["snacbook-pro.local"]
    }
  ]
}
```

##### Default response

`Status: 200`

### Get labels

`GET /api/v1/fleet/spec/labels`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/spec/labels`

##### Default response

`Status: 200`

```json
{
  "specs": [
    {
      "id": 6,
      "name": "All Hosts",
      "description": "All hosts which have enrolled in Fleet",
      "query": "SELECT 1;",
      "label_type": "builtin",
      "label_membership_type": "dynamic"
    },
    {
      "id": 7,
      "name": "macOS",
      "description": "All macOS hosts",
      "query": "SELECT 1 FROM os_version WHERE platform = 'darwin';",
      "platform": "darwin",
      "label_type": "builtin",
      "label_membership_type": "dynamic"
    },
    {
      "id": 8,
      "name": "Ubuntu Linux",
      "description": "All Ubuntu hosts",
      "query": "SELECT 1 FROM os_version WHERE platform = 'ubuntu';",
      "platform": "ubuntu",
      "label_type": "builtin",
      "label_membership_type": "dynamic"
    },
    {
      "id": 9,
      "name": "CentOS Linux",
      "description": "All CentOS hosts",
      "query": "SELECT 1 FROM os_version WHERE platform = 'centos' OR name LIKE '%centos%'",
      "label_type": "builtin",
      "label_membership_type": "dynamic"
    },
    {
      "id": 10,
      "name": "MS Windows",
      "description": "All Windows hosts",
      "query": "SELECT 1 FROM os_version WHERE platform = 'windows';",
      "platform": "windows",
      "label_type": "builtin",
      "label_membership_type": "dynamic"
    },
    {
      "id": 11,
      "name": "Ubuntu",
      "description": "Filters Ubuntu hosts",
      "query": "SELECT 1 FROM os_version WHERE platform = 'ubuntu';",
      "label_membership_type": "dynamic"
    }
  ]
}
```

### Get label

Returns the label specified by name.

`GET /api/v1/fleet/spec/labels/{name}`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/spec/labels/local_machine`

##### Default response

`Status: 200`

```json
{
  "specs": {
    "id": 12,
    "name": "local_machine",
    "description": "Includes only my local machine",
    "query": "",
    "label_membership_type": "manual"
  }
}
```

### Get enroll secrets

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
        "secret": "fTp52/twaxBU6gIi0J6PHp8o5Sm1k1kn",
        "created_at": "2021-01-07T19:40:04Z"
      },
      {
        "secret": "bhD5kiX2J+KBgZSk118qO61ZIdX/v8On",
        "created_at": "2021-01-04T21:18:07Z"
      }
    ]
  }
}
```

### Modify enroll secrets

This replaces the active global enroll secrets with the secrets specified.

`POST /api/v1/fleet/spec/enroll_secret`

#### Parameters

| Name    | Type | In   | Description                                                                                                      |
| ------- | ---- | ---- | ---------------------------------------------------------------------------------------------------------------- |
| secrets | list | body | **Required.** The plain text string used as the enroll secret. Note that there is a limit of 50 secrets allowed. |

#### Example

##### Request body

```json
{
  "spec": {
    "secrets": [
      {
        "secret": "fTp52/twaxBU6gIi0J6PHp8o5Sm1k1kn"
      }
    ]
  }
}
```

`POST /api/v1/fleet/spec/enroll_secret`

##### Default response

`Status: 200`

---

## Live query

These API routes are used by the Fleet UI.

- [Check live query status](#check-live-query-status)
- [Check result store status](#check-result-store-status)
- [Search targets](#search-targets)
- [Count targets](#count-targets)
- [Run live query](#run-live-query)
- [Run live query by name](#run-live-query-by-name)
- [Retrieve live query results (standard WebSocket API)](#retrieve-live-query-results-standard-websocket-api)
- [Retrieve live query results (SockJS)](#retrieve-live-query-results-sockjs)

### Check live query status

This checks the status of Fleet's ability to run a live query. If an error is present in the response, Fleet won't be able to run a live query successfully. The Fleet UI uses this endpoint to make sure that the Fleet instance is correctly configured to run live queries.

`GET /api/v1/fleet/status/live_query`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/status/live_query`

##### Default response

`Status: 200`

### Check result store status

This checks Fleet's result store status. If an error is present in the response, Fleet won't be able to run a live query successfully. The Fleet UI uses this endpoint to make sure that the Fleet instance is correctly configured to run live queries.

`GET /api/v1/fleet/status/result_store`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/status/result_store`

##### Default response

`Status: 200`

### Search targets

Accepts a search query and a list of host IDs to omit and returns a set of up to ten matching hosts. If
a query ID is provided and the referenced query allows observers to run, targets will include hosts
for which the user has an observer role.

`POST /api/latest/fleet/hosts/search`

#### Parameters

| Name              | Type    | In   | Description                                                                                                                                     |
| ----------------- | ------- | ---- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| query             | string  | body | The query used to identify hosts to target. Searchable items include a `display_name`, `hostname`, `hardware_serial`, `uuid` or `primary_ip`. |
| query_id          | integer | body | The saved query (if any) that will be run. The `observer_can_run` property on the query and the user's roles affect which targets are included. |
| excluded_host_ids | array   | body | The list of host ids to omit from the search results.                                                                                           |

#### Example

`POST /api/v1/fleet/hosts/search`

##### Request body

```json
{
  "query": "foo",
  "query_id": 42,
  "selected": {
    "hosts": [],
    "labels": [],
    "teams": [1]
  }
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
        "id": 1337,
        "detail_updated_at": "2021-02-03T21:58:10Z",
        "label_updated_at": "2021-02-03T21:58:10Z",
        "last_enrolled_at": "2021-02-03T16:11:43Z",
        "seen_time": "2021-02-03T21:58:20Z",
        "hostname": "foof41482833",
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
        "computer_name": "foof41482833",
        "display_name": "foof41482833",
        "primary_ip": "172.20.0.3",
        "primary_mac": "02:42:ac:14:00:03",
        "distributed_interval": 10,
        "config_tls_refresh": 10,
        "logger_tls_period": 10,
        "additional": {},
        "status": "offline",
        "display_text": "foof41482833"
      }
    ]
  }
}
```

### Count targets

Counts the number of online and offline hosts included in a given set of selected targets.

`POST /api/latest/fleet/targets/count`

#### Parameters

| Name     | Type    | In   | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| -------- | ------- | ---- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| query_id | integer | body | The saved query (if any) that will be run. The `observer_can_run` property on the query and the user's roles determine which targets are included.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| selected | object  | body | The object includes lists of selected host IDs (`selected.hosts`), label IDs (`selected.labels`), and team IDs (`selected.teams`). When provided, builtin label IDs, custom label IDs and team IDs become `AND` filters. Within each selector, selecting two or more teams, two or more builtin labels, or two or more custom labels, behave as `OR` filters. There's one special case for the builtin label "All hosts", if such label is selected, then all other label and team selectors are ignored (and all hosts will be selected). If a host ID is explicitly included in `selected.hosts`, then it is assured that the query will be selected to run on it (no matter the contents of `selected.labels` and `selected.teams`). Use `0` team ID to filter by hosts assigned to "No team". See examples below. |

#### Example

`POST /api/latest/fleet/targets/count`

##### Request body

```json
{
  "query_id": 1337,
  "selected": {
    "hosts": [],
    "labels": [42],
    "teams": []
  }
}
```

##### Default response

```json
{
  "targets_count": 813,
  "targets_offline": 813,
  "targets_online": 0
}
```

### Run live query

Runs the specified query as a live query on the specified hosts or group of hosts and returns a new live query campaign. Individual hosts must be specified with the host's ID. Label IDs also specify groups of hosts.

After you initiate the query, [get results via WebSocket](#retrieve-live-query-results-standard-websocket-api).

`POST /api/v1/fleet/queries/run`

#### Parameters

| Name     | Type    | In   | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| -------- | ------- | ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| query    | string  | body | The SQL if using a custom query.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| query_id | integer | body | The saved query (if any) that will be run. Required if running query as an observer. The `observer_can_run` property on the query effects which targets are included.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| selected | object  | body | **Required.** The object includes lists of selected host IDs (`selected.hosts`), label IDs (`selected.labels`), and team IDs (`selected.teams`). When provided, builtin label IDs, custom label IDs and team IDs become `AND` filters. Within each selector, selecting two or more teams, two or more builtin labels, or two or more custom labels, behave as `OR` filters. There's one special case for the builtin label "All hosts", if such label is selected, then all other label and team selectors are ignored (and all hosts will be selected). If a host ID is explicitly included in `selected.hosts`, then it is assured that the query will be selected to run on it (no matter the contents of `selected.labels` and `selected.teams`). Use `0` team ID to filter by hosts assigned to "No team". See examples below. |

One of `query` and `query_id` must be specified.

#### Example with one host targeted by ID

`POST /api/v1/fleet/queries/run`

##### Request body

```json
{
  "query": "SELECT instance_id FROM system_info",
  "selected": {
    "hosts": [171]
  }
}
```

##### Default response

`Status: 200`

```json
{
  "campaign": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "Metrics": {
      "TotalHosts": 1,
      "OnlineHosts": 0,
      "OfflineHosts": 1,
      "MissingInActionHosts": 0,
      "NewHosts": 1
    },
    "id": 1,
    "query_id": 3,
    "status": 0,
    "user_id": 1
  }
}
```

#### Example with multiple hosts targeted by label ID

`POST /api/v1/fleet/queries/run`

##### Request body

```json
{
  "query": "SELECT instance_id FROM system_info;",
  "selected": {
    "labels": [7]
  }
}
```

##### Default response

`Status: 200`

```json
{
  "campaign": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "Metrics": {
      "TotalHosts": 102,
      "OnlineHosts": 0,
      "OfflineHosts": 24,
      "MissingInActionHosts": 0,
      "NewHosts": 0
    },
    "id": 2,
    "query_id": 3,
    "status": 0,
    "user_id": 1
  }
}
```

### Run live query by name

Runs the specified saved query as a live query on the specified targets. Returns a new live query campaign. Individual hosts must be specified with the host's hostname. Groups of hosts are specified by label name.

After the query has been initiated, [get results via WebSocket](#retrieve-live-query-results-standard-websocket-api).

`POST /api/v1/fleet/queries/run_by_names`

#### Parameters

| Name     | Type    | In   | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| -------- | ------- | ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| query    | string  | body | The SQL of the query.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| query_id | integer | body | The saved query (if any) that will be run. The `observer_can_run` property on the query effects which targets are included.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| selected | object  | body | **Required.** The object includes lists of selected hostnames (`selected.hosts`), label names (`labels`). When provided, builtin label names and custom label names become `AND` filters. Within each selector, selecting two or more builtin labels, or two or more custom labels, behave as `OR` filters. There's one special case for the builtin label `"All hosts"`, if such label is selected, then all other label and team selectors are ignored (and all hosts will be selected). If a host's hostname is explicitly included in `selected.hosts`, then it is assured that the query will be selected to run on it (no matter the contents of `selected.labels`). See examples below. |

One of `query` and `query_id` must be specified.

#### Example with one host targeted by hostname

`POST /api/v1/fleet/queries/run_by_names`

##### Request body

```json
{
  "query_id": 1,
  "selected": {
    "hosts": ["macbook-pro.local"]
  }
}
```

##### Default response

`Status: 200`

```json
{
  "campaign": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "Metrics": {
      "TotalHosts": 1,
      "OnlineHosts": 0,
      "OfflineHosts": 1,
      "MissingInActionHosts": 0,
      "NewHosts": 1
    },
    "id": 1,
    "query_id": 3,
    "status": 0,
    "user_id": 1
  }
}
```

#### Example with multiple hosts targeted by label name

`POST /api/v1/fleet/queries/run_by_names`

##### Request body

```json
{
  "query": "SELECT instance_id FROM system_info",
  "selected": {
    "labels": ["All Hosts"]
  }
}
```

##### Default response

`Status: 200`

```json
{
  "campaign": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "Metrics": {
      "TotalHosts": 102,
      "OnlineHosts": 0,
      "OfflineHosts": 24,
      "MissingInActionHosts": 0,
      "NewHosts": 1
    },
    "id": 2,
    "query_id": 3,
    "status": 0,
    "user_id": 1
  }
}
```

### Retrieve live query results (standard WebSocket API)

You can retrieve the results of a live query using the [standard WebSocket API](#https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API/Writing_WebSocket_client_applications).

Before you retrieve the live query results, you must create a live query campaign by running the live query. Use the [Run live query](#run-live-query) or [Run live query by name](#run-live-query-by-name) endpoints to create a live query campaign.

Note that live queries are automatically cancelled if this method is not called to start retrieving the results within 60 seconds of initiating the query.

`/api/v1/fleet/results/websocket`

### Parameters

| Name       | Type    | In  | Description                                                      |
| ---------- | ------- | --- | ---------------------------------------------------------------- |
| token      | string  |     | **Required.** The token used to authenticate with the Fleet API. |
| campaignID | integer |     | **Required.** The ID of the live query campaign.                 |

### Example

#### Example script to handle request and response

```js
const socket = new WebSocket('wss://<your-base-url>/api/v1/fleet/results/websocket');

socket.onopen = () => {
  socket.send(JSON.stringify({ type: 'auth', data: { token: <auth-token> } }));
  socket.send(JSON.stringify({ type: 'select_campaign', data: { campaign_id: <campaign-id> } }));
};

socket.onmessage = ({ data }) => {
  console.log(data);
  const message = JSON.parse(data);
  if (message.type === 'status' && message.data.status === 'finished') {
    socket.close();
  }
}
```

### Detailed request and response walkthrough with example data

#### webSocket.onopen()

##### Response data

```json
o
```

#### webSocket.send()

##### Request data

```json
[
  {
    "type": "auth",
    "data": { "token": <insert_token_here> }
  }
]
```

```json
[
  {
    "type": "select_campaign",
    "data": { "campaign_id": 12 }
  }
]
```

#### webSocket.onmessage()

##### Response data

```json
// Sends the total number of hosts targeted and segments them by status

[
  {
    "type": "totals",
    "data": {
      "count": 24,
      "online": 6,
      "offline": 18,
      "missing_in_action": 0
    }
  }
]
```

```json
// Sends the expected results, actual results so far, and the status of the live query

[
  {
    "type": "status",
    "data": {
      "expected_results": 6,
      "actual_results": 0,
      "status": "pending"
    }
  }
]
```

```json
// Sends the result for a given host

[
  {
    "type": "result",
    "data": {
      "distributed_query_execution_id": 39,
      "host": {
        "id": 42,
        "hostname": "foobar",
        "display_name": "foobar"
      },
      "rows": [
        // query results data for the given host
      ],
      "error": null
    }
  }
]
```

```json
// Sends the status of "finished" when messages with the results for all expected hosts have been sent

[
  {
    "type": "status",
    "data": {
      "expected_results": 6,
      "actual_results": 6,
      "status": "finished"
    }
  }
]
```

### Retrieve live query results (SockJS)

You can also retrieve live query results with a [SockJS client](https://github.com/sockjs/sockjs-client). The script to handle the request and response messages will look similar to the standard WebSocket API script with slight variations. For example, the constructor used for SockJS is `SockJS` while the constructor used for the standard WebSocket API is `WebSocket`.

Note that SockJS has been found to be substantially less reliable than the [standard WebSockets approach](#retrieve-live-query-results-standard-websocket-api).

`/api/v1/fleet/results/`

### Parameters

| Name       | Type    | In  | Description                                                      |
| ---------- | ------- | --- | ---------------------------------------------------------------- |
| token      | string  |     | **Required.** The token used to authenticate with the Fleet API. |
| campaignID | integer |     | **Required.** The ID of the live query campaign.                 |

### Example

#### Example script to handle request and response

```js
const socket = new SockJS(`<your-base-url>/api/v1/fleet/results`, undefined, {});

socket.onopen = () => {
  socket.send(JSON.stringify({ type: 'auth', data: { token: <token> } }));
  socket.send(JSON.stringify({ type: 'select_campaign', data: { campaign_id: <campaignID> } }));
};

socket.onmessage = ({ data }) => {
  console.log(data);
  const message = JSON.parse(data);

  if (message.type === 'status' && message.data.status === 'finished') {
    socket.close();
  }
}
```

##### Detailed request and response walkthrough

#### socket.onopen()

##### Response data

```json
o
```

#### socket.send()

##### Request data

```json
[
  {
    "type": "auth",
    "data": { "token": <insert_token_here> }
  }
]
```

```json
[
  {
    "type": "select_campaign",
    "data": { "campaign_id": 12 }
  }
]
```

#### socket.onmessage()

##### Response data

```json
// Sends the total number of hosts targeted and segments them by status

[
  {
    "type": "totals",
    "data": {
      "count": 24,
      "online": 6,
      "offline": 18,
      "missing_in_action": 0
    }
  }
]
```

```json
// Sends the expected results, actual results so far, and the status of the live query

[
  {
    "type": "status",
    "data": {
      "expected_results": 6,
      "actual_results": 0,
      "status": "pending"
    }
  }
]
```

```json
// Sends the result for a given host

[
  {
    "type": "result",
    "data": {
      "distributed_query_execution_id": 39,
      "host": {
        "id": 42,
        "hostname": "foobar",
        "display_name": "foobar"
      },
      "rows": [
        // query results data for the given host
      ],
      "error": null
    }
  }
]
```

```json
// Sends the status of "finished" when messages with the results for all expected hosts have been sent

[
  {
    "type": "status",
    "data": {
      "expected_results": 6,
      "actual_results": 6,
      "status": "finished"
    }
  }
]
```

---

## Trigger cron schedule

This API is used by the `fleetctl` CLI tool to make requests to trigger an ad hoc run of all jobs in
a specified cron schedule.

### Trigger

This makes a request to trigger the specified cron schedule. Upon receiving the request, the Fleet
server first checks the current status of the schedule, and it returns an error if a run is
currently pending.

`POST /api/latest/fleet/trigger`

#### Parameters

| Name | Type   | In    | Description                               |
| ---- | ------ | ----- | ----------------------------------------- |
| name | string | query | The name of the cron schedule to trigger. Supported trigger names are `apple_mdm_dep_profile_assigner`, `automations`, `cleanups_then_aggregation`, `integrations`, `mdm_apple_profile_manager`, `usage_statistics`, and `vulnerabilities`|


#### Example

`POST /api/latest/fleet/trigger?name=automations`

##### Default response

`Status: 200`

---

## Device-authenticated routes

Device-authenticated routes are routes used by the Fleet Desktop application. Unlike most other routes, Fleet user's API token does not authenticate them. They use a device-specific token.

- [Refetch device's host](#refetch-devices-host)
- [Get device's Google Chrome profiles](#get-devices-google-chrome-profiles)
- [Get device's mobile device management (MDM) and Munki information](#get-devices-mobile-device-management-mdm-and-munki-information)
- [Get Fleet Desktop information](#get-fleet-desktop-information)
- [Get device's software](#get-devices-software)
- [Get device's policies](#get-devices-policies)
- [Get device's API features](#get-devices-api-features)
- [Get device's transparency URL](#get-devices-transparency-url)
- [Download device's MDM manual enrollment profile](#download-devices-mdm-manual-enrollment-profile)
- [Migrate device to Fleet from another MDM solution](#migrate-device-to-fleet-from-another-mdm-solution)
- [Trigger Linux disk encryption escrow](#trigger-linux-disk-encryption-escrow)
- [Report an agent error](#report-an-agent-error)

#### Refetch device's host

Same as [Refetch host route](https://fleetdm.com/docs/using-fleet/rest-api#refetch-host) for the current device.

`POST /api/v1/fleet/device/{token}/refetch`

##### Parameters

| Name  | Type   | In   | Description                        |
| ----- | ------ | ---- | ---------------------------------- |
| token | string | path | The device's authentication token. |

#### Get device's Google Chrome profiles

Same as [Get host's Google Chrome profiles](https://fleetdm.com/docs/using-fleet/rest-api#get-hosts-google-chrome-profiles) for the current device.

`GET /api/v1/fleet/device/{token}/device_mapping`

##### Parameters

| Name  | Type   | In   | Description                        |
| ----- | ------ | ---- | ---------------------------------- |
| token | string | path | The device's authentication token. |

#### Get device's mobile device management (MDM) and Munki information

Same as [Get host's mobile device management and Munki information](https://fleetdm.com/docs/using-fleet/rest-api#get-hosts-mobile-device-management-mdm-and-munki-information) for the current device.

`GET /api/v1/fleet/device/{token}/macadmins`

##### Parameters

| Name  | Type   | In   | Description                        |
| ----- | ------ | ---- | ---------------------------------- |
| token | string | path | The device's authentication token. |

#### Ping Server with Device Token
Ping the server. OK response expected if the device token is still valid.

`HEAD /api/v1/fleet/device/{token}/ping`

##### Parameters

| Name  | Type   | In   | Description                        |
| ----- | ------ | ---- | ---------------------------------- |
| token | string | path | The device's authentication token. |

##### Example

`HEAD /api/v1/fleet/device/abcdef012456789/ping`

##### Default response

`Status: 200`

#### Get Fleet Desktop information
_Available in Fleet Premium_

Gets all information required by Fleet Desktop, this includes things like the number of failed policies or notifications to show/hide menu items.

`GET /api/v1/fleet/device/{token}/desktop`

##### Parameters

| Name  | Type   | In   | Description                        |
| ----- | ------ | ---- | ---------------------------------- |
| token | string | path | The device's authentication token. |

##### Example

`GET /api/v1/fleet/device/abcdef012456789/desktop`

##### Default response

`Status: 200`

```json
{
  "failing_policies_count": 3,
  "self_service": true,
  "notifications": {
    "needs_mdm_migration": true,
    "renew_enrollment_profile": false,
  },
  "config": {
    "org_info": {
      "org_name": "Fleet",
      "org_logo_url": "https://example.com/logo.jpg",
      "org_logo_url_light_background": "https://example.com/logo-light.jpg",
      "contact_url": "https://fleetdm.com/company/contact"
    },
    "mdm": {
      "macos_migration": {
        "mode": "forced"
      }
    }
  }
}
```

In regards to the `notifications` key:

- `needs_mdm_migration` means that the device fits all the requirements to allow the user to initiate an MDM migration to Fleet.
- `renew_enrollment_profile` means that the device is currently unmanaged from MDM but should be DEP enrolled into Fleet.

#### Get device's software

Lists the software installed on the current device.

`GET /api/v1/fleet/device/{token}/software`

##### Parameters

| Name  | Type   | In   | Description                        |
| ----- | ------ | ---- | ---------------------------------- |
| token | string | path | The device's authentication token. |
| self_service | bool | query | Filter `self_service` software. |
| query   | string | query | Search query keywords. Searchable fields include `name`. |
| page | integer | query | Page number of the results to fetch.|
| per_page | integer | query | Results per page.|

##### Example

`GET /api/v1/fleet/device/bbb7cdcc-f1d9-4b39-af9e-daa0f35728e8/software`

##### Default response

`Status: 200`

```json
{
  "count": 2,
  "software": [
    {
      "id": 121,
      "name": "Google Chrome.app",
      "software_package": {
        "name": "GoogleChrome.pkg"
        "version": "125.12.2"
        "self_service": true,
     	"last_install": {
          "install_uuid": "8bbb8ac2-b254-4387-8cba-4d8a0407368b",
      	  "installed_at": "2024-05-15T15:23:57Z"
        },
      },
      "app_store_app": null,
      "source": "apps",
      "status": "failed",
      "installed_versions": [
        {
          "version": "121.0",
          "last_opened_at": "2024-04-01T23:03:07Z",
          "vulnerabilities": ["CVE-2023-1234","CVE-2023-4321","CVE-2023-7654"],
          "installed_paths": ["/Applications/Google Chrome.app"]
        }
      ],
       "software_package": {
        "name": "google-chrome-124-0-6367-207.pkg",
        "version": "121.0",
        "self_service": true,
        "icon_url": null,
        "last_install": null
      },
      "app_store_app": null
    },
    {
      "id": 143,
      "name": "Firefox.app",
      "software_package": null,
      "app_store_app": null,
      "source": "apps",
      "status": null,
      "installed_versions": [
        {
          "version": "125.6",
          "last_opened_at": "2024-04-01T23:03:07Z",
          "vulnerabilities": ["CVE-2023-1234","CVE-2023-4321","CVE-2023-7654"],
          "installed_paths": ["/Applications/Firefox.app"]
        }
      ],
      "software_package": null,
      "app_store_app": {
        "app_store_id": "12345",
        "version": "125.6",
        "self_service": false,
        "icon_url": "https://example.com/logo-light.jpg",
        "last_install": null
      },
    }
  ],
  "meta": {
    "has_next_results": false,
    "has_previous_results": false
  }
}
```

#### Install self-service software

Install self-service software on macOS, Windows, or Linux (Ubuntu) host. The software must have a `self_service` flag `true` to be installed.

`POST /api/v1/fleet/device/{token}/software/install/:software_title_id`

##### Parameters

| Name  | Type   | In   | Description                        |
| ----- | ------ | ---- | ---------------------------------- |
| token | string | path | **Required**. The device's authentication token. |
| software_title_id | string | path | **Required**. The software title's ID. |

##### Example

`POST /api/v1/fleet/device/22aada07-dc73-41f2-8452-c0987543fd29/software/install/123`

##### Default response

`Status: 202`

#### Get device's policies

_Available in Fleet Premium_

Lists the policies applied to the current device.

`GET /api/v1/fleet/device/{token}/policies`

##### Parameters

| Name  | Type   | In   | Description                        |
| ----- | ------ | ---- | ---------------------------------- |
| token | string | path | The device's authentication token. |

##### Example

`GET /api/v1/fleet/device/abcdef012456789/policies`

##### Default response

`Status: 200`

```json
{
  "policies": [
    {
      "id": 1,
      "name": "SomeQuery",
      "query": "SELECT * FROM foo;",
      "description": "this is a query",
      "resolution": "fix with these steps...",
      "platform": "windows,linux",
      "response": "pass"
    },
    {
      "id": 2,
      "name": "SomeQuery2",
      "query": "SELECT * FROM bar;",
      "description": "this is another query",
      "resolution": "fix with these other steps...",
      "platform": "darwin",
      "response": "fail"
    },
    {
      "id": 3,
      "name": "SomeQuery3",
      "query": "SELECT * FROM baz;",
      "description": "",
      "resolution": "",
      "platform": "",
      "response": ""
    }
  ]
}
```

#### Get device's API features

This supports the dynamic discovery of API features supported by the server for device-authenticated routes. This allows supporting different versions of Fleet Desktop and Fleet server instances (older or newer) while supporting the evolution of the API features. With this mechanism, an older Fleet Desktop can ignore features it doesn't know about, and a newer one can avoid requesting features about which the server doesn't know.

`GET /api/v1/fleet/device/{token}/api_features`

##### Parameters

| Name  | Type   | In   | Description                        |
| ----- | ------ | ---- | ---------------------------------- |
| token | string | path | The device's authentication token. |

##### Example

`GET /api/v1/fleet/device/abcdef012456789/api_features`

##### Default response

`Status: 200`

```json
{
  "features": {}
}
```

#### Get device's transparency URL

Returns the URL to open when clicking the "About Fleet" menu item in Fleet Desktop. Note that _Fleet Premium_ is required to configure a custom transparency URL.

`GET /api/v1/fleet/device/{token}/transparency`

##### Parameters

| Name  | Type   | In   | Description                        |
| ----- | ------ | ---- | ---------------------------------- |
| token | string | path | The device's authentication token. |

##### Example

`GET /api/v1/fleet/device/abcdef012456789/transparency`

##### Default response

`Status: 307`

Redirects to the transparency URL.

#### Download device's MDM manual enrollment profile

Downloads the Mobile Device Management (MDM) enrollment profile to install on the device for a manual enrollment into Fleet MDM.

`GET /api/v1/fleet/device/{token}/mdm/apple/manual_enrollment_profile`

##### Parameters

| Name  | Type   | In   | Description                        |
| ----- | ------ | ---- | ---------------------------------- |
| token | string | path | The device's authentication token. |

##### Example

`GET /api/v1/fleet/device/abcdef012456789/mdm/apple/manual_enrollment_profile`

##### Default response

`Status: 200`

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<!-- ... -->
</plist>
```

---

#### Migrate device to Fleet from another MDM solution

Signals the Fleet server to send a webbook request with the device UUID and serial number to the webhook URL configured for MDM migration. **Requires Fleet Premium license**

`POST /api/v1/fleet/device/{token}/migrate_mdm`

##### Parameters

| Name  | Type   | In   | Description                        |
| ----- | ------ | ---- | ---------------------------------- |
| token | string | path | The device's authentication token. |

##### Example

`POST /api/v1/fleet/device/abcdef012456789/migrate_mdm`

##### Default response

`Status: 204`

---

### Trigger Linux disk encryption escrow

_Available in Fleet Premium_

Signals the fleet server to queue up the LUKS disk encryption escrow process (LUKS passphrase and slot key). If validation succeeds (disk encryption must be enforced for the team, the host's platform must be supported, the host's disk must already be encrypted, and the host's Orbit version must be new enough), this adds a notification flag for Orbit that, triggers escrow from the Orbit side.

`POST /api/v1/fleet/device/{token}/mdm/linux/trigger_escrow`

##### Parameters

| Name  | Type   | In   | Description                        |
| ----- | ------ | ---- | ---------------------------------- |
| token | string | path | The device's authentication token. |

##### Example

`POST /api/v1/fleet/device/abcdef012456789/mdm/linux/trigger_escrow`

##### Default response

`Status: 204`

---

### Report an agent error

Notifies the server about an agent error, resulting in two outcomes:

- The error gets saved in Redis and can later be accessed using `fleetctl debug archive`.

> Note: to allow `fleetd` agents to use this endpoint, you need to set a [custom environment variable](./Configuration-for-contributors.md#fleet_enable_post_client_debug_errors). `fleetd` agents will always report vital errors to Fleet.

`POST /api/v1/fleet/device/{token}/debug/errors`

#### Parameters

| Name                  | Type     | Description                                                                                                                               |
|-----------------------|----------|-------------------------------------------------------------------------------------------------------------------------------------------|
| error_source          | string   | Process name that error originated from ex. orbit, fleet-desktop                                                                          |
| error_source_version  | string   | version of error_source                                                                                                                   |
| error_timestamp       | datetime | Time in UTC that error occured                                                                                                            |
| error_message         | string   | error message                                                                                                                             |
| error_additional_info | obj      | Any additional identifiers to assist debugging                                                                                            |
| vital                 | boolean  | Whether the error is vital and should also be reported to Fleet via usage statistics. Do not put sensitive information into vital errors. |

##### Default response

`Status: 200`

---

## Orbit-authenticated routes

- [Escrow LUKS data](#escrow-luks-data)
- [Get the status of a device in the setup experience](#get-the-status-of-a-device-in-the-setup-experience)
- [Set or update device token](#set-or-update-device-token)
- [Get orbit script](#get-orbit-script)
- [Post orbit script result](#post-orbit-script-result)
- [Put orbit device mapping](#put-orbit-device-mapping)
- [Post orbit software install result](#post-orbit-software-install-result)
- [Download software installer](#download-software-installer)
- [Get orbit software install details](#get-orbit-software-install-details)
- [Post disk encryption key](#post-disk-encryption-key)

---

### Escrow LUKS data

`POST /api/fleet/orbit/luks_data`

##### Parameters

| Name  | Type   | In   | Description                        |
| ----- | ------ | ---- | ---------------------------------- |
| orbit_node_key | string | body | The Orbit node key for authentication. |
| client_error | string | body | An error description if the LUKS key escrow process fails client-side. If provided, passphrase/salt/key slot request parameters are ignored and may be omitted. |
| passphrase | string | body | The LUKS passphrase generated for Fleet (the end user's existing passphrase is not transmitted) |
| key_slot | int | body | The LUKS key slot ID corresponding to the provided passphrase |
| salt | string | body | The salt corresponding to the specified LUKS key slot. Provided to track cases where an end user rotates LUKS credentials (at which point we'll no longer be able to decrypt data with the escrowed passphrase). |

##### Example

`POST /api/v1/fleet/orbit/luks_data`

##### Request body

```json
{
  "orbit_node_key":"FbvSsWfTRwXEecUlCBTLmBcjGFAdzqd/",
  "passphrase": "6e657665-7220676f-6e6e6120-67697665-20796f75-207570",
  "salt": "d34db33f",
  "key_slot": 1,
  "client_error": ""
}
```

##### Default response

`Status: 204`

---

### Get the status of a device in the setup experience

`POST /api/fleet/orbit/setup_experience/status`

##### Parameters

| Name  | Type   | In   | Description                        |
| ----- | ------ | ---- | ---------------------------------- |
| orbit_node_key | string | body | The Orbit node key for authentication. |
| force_release | boolean | body | Force a host release from ADE flow, in case the setup is taking too long. |


##### Example

`POST /api/v1/fleet/orbit/setup_experience/status`

##### Request body

```json
{
  "orbit_node_key":"FbvSsWfTRwXEecUlCBTLmBcjGFAdzqd/",
  "force_release":false
}
```

##### Default response

`Status: 200`

```json
{
    "setup_experience_results": {
        "script": {
            "name": "setup_script.sh",
            "status": "success",
            "execution_id": "b16fdd31-71cc-4258-ab27-744490809ebd"
        },
        "software": [
            {
                "name": "Zoom Workplace",
                "status": "success",
                "software_title_id": 957
            },
            {
                "name": "Bear: Markdown Notes",
                "status": "success",
                "software_title_id": 287
            },
            {
                "name": "Evernote",
                "status": "success",
                "software_title_id": 1313
            }
        ],
        "configuration_profiles": [
            {
                "profile_uuid": "ae6a9efd5-9166-11ef-83af-0242ac12000b",
                "name": "Fleetd configuration",
                "status": "verified"
            },
            {
                "profile_uuid": "ae6aa8108-9166-11ef-83af-0242ac12000b",
                "name": "Fleet root certificate authority (CA)",
                "status": "verified"
            }
        ],
        "org_logo_url": ""
    }
}

```

### Set or update device token

`POST /api/fleet/orbit/device_token`

##### Parameters

| Name              | Type   | In   | Description                                 |
| ----------------- | ------ | ---- | ------------------------------------------- |
| orbit_node_key    | string | body | The Orbit node key for authentication.      |
| device_auth_token | string | body | The device auth token to set for this host. |

##### Example

`POST /api/v1/fleet/orbit/device_token`

##### Request body

```json
{
  "orbit_node_key":"FbvSsWfTRwXEecUlCBTLmBcjGFAdzqd/",
  "device_auth_token": "2267a440-4cfb-48af-804b-d52224a05e1b"
}
```

##### Default response

`Status: 200`

### Get Orbit config

`POST /api/fleet/orbit/config`

##### Parameters

| Name           | Type   | In   | Description                            |
| -------------- | ------ | ---- | -------------------------------------- |
| orbit_node_key | string | body | The Orbit node key for authentication. |

##### Example

`POST /api/fleet/orbit/config`

##### Request body

```json
{
  "orbit_node_key":"FbvSsWfTRwXEecUlCBTLmBcjGFAdzqd/"
}
```

##### Default response

`Status: 200`

```json
{
  "script_execution_timeout": 3600,
  "command_line_startup_flags": {
    "--verbose": true
  },
  "extensions": {
    "hello_world_linux": {
      "channel": "stable",
      "platform": "linux"
    }
  },
  "nudge_config": {
    "osVersionRequirements": [
      {
        "requiredInstallationDate": "2024-12-04T20:00:00Z",
        "requiredMinimumOSVersion": "15.1.1",
        "aboutUpdateURLs": [
          {
            "_language": "en",
            "aboutUpdateURL": "https://fleetdm.com/learn-more-about/os-updates"
          }
        ]
      }
    ],
    "userInterface": {
      "simpleMode": true,
      "showDeferralCount": false,
      "updateElements": [
        {
          "_language": "en",
          "actionButtonText": "Update",
          "mainHeader": "Your device requires an update"
        }
      ]
    },
    "userExperience": {
      "initialRefreshCycle": 86400,
      "approachingRefreshCycle": 86400,
      "imminentRefreshCycle": 7200,
      "elapsedRefreshCycle": 3600
    }
  },
  "notifications": {
    "renew_enrollment_profile": true,
    "rotate_disk_encryption_key": true,
    "needs_mdm_migration": true,
    "needs_programmatic_windows_mdm_enrollment": true,
    "windows_mdm_discovery_endpoint": "/some/path/here",
    "needs_programmatic_windows_mdm_unenrollment": true,
    "pending_script_execution_ids": [
      "a129a440-4cfb-48af-804b-d52224a05e1b"
    ],
    "enforce_bitlocker_encryption": true,
    "pending_software_installer_ids": [
      "2267a440-4cfb-48af-804b-d52224a05e1b"
    ],
    "run_setup_experience": true,
    "run_disk_encryption_escrow": true
  },
  "update_channels": {
    "orbit": "stable",
    "osqueryd": "stable",
    "desktop": "stable"
  }
}
```

### Get script execution result by execution ID

`POST /api/fleet/orbit/scripts/request`

##### Parameters

| Name           | Type   | In   | Description                            |
| -------------- | ------ | ---- | -------------------------------------- |
| orbit_node_key | string | body | The Orbit node key for authentication. |
| execution_id   | string | body | The UUID of the script execution.      |

##### Example

`POST /api/fleet/orbit/scripts/request`

##### Request body

```json
{
  "orbit_node_key":"FbvSsWfTRwXEecUlCBTLmBcjGFAdzqd/",
  "execution_id": "006112E7-7383-4F21-999C-8FA74BB3F573"
}
```

##### Default response

`Status: 200`

```json
{
  "host_id": 12,
  "execution_id": "006112E7-7383-4F21-999C-8FA74BB3F573",
  "script_contents": "echo hello",
  "output": "hello",
  "runtime": 1,
  "exit_code": 0,
  "timeout": 30,
  "script_id": 42,
  "policy_id": 10,
  "team_id": 1,
  "message": ""
}
```

### Upload Orbit script result

`POST /api/fleet/orbit/scripts/result`

##### Parameters

| Name           | Type   | In     | Description                                                             |
| -------------- | ------ | ------ | ----------------------------------------------------------------------- |
| orbit_node_key | string | body   | The Orbit node key for authentication.                                  |
| host_id        | number | body   | The ID of the host on which the script ran.                             |
| execution_id   | string | body   | The UUID of the script execution.                                       |
| output         | string | body   | The output of the script.                                               |
| runtime        | string | number | The amount of time the script ran for (in seconds).                     |
| exit_code      | string | number | The exit code of the script.                                            |
| timeout        | string | number | The maximum amount of time this script was allowed to run (in seconds). |

##### Example

`POST /api/fleet/orbit/scripts/result`

##### Request body

```json
{
  "orbit_node_key":"FbvSsWfTRwXEecUlCBTLmBcjGFAdzqd/",
  "host_id": 12,
  "execution_id": "006112E7-7383-4F21-999C-8FA74BB3F573",
  "output": "hello",
  "runtime": 1,
  "exit_code": 0,
  "timeout": 30
}
```

##### Default response

`Status: 200`

### Set Orbit device mapping

`POST /api/fleet/orbit/device_mapping`

##### Parameters

| Name           | Type   | In   | Description                              |
| -------------- | ------ | ---- | ---------------------------------------- |
| orbit_node_key | string | body | The Orbit node key for authentication.   |
| email          | string | body | The email to use for the device mapping. |

##### Example

`POST /api/fleet/orbit/device_mapping`

##### Request body

```json
{
  "orbit_node_key":"FbvSsWfTRwXEecUlCBTLmBcjGFAdzqd/",
  "email": "test@example.com"
}
```

##### Default response

`Status: 200`

### Upload Orbit software install result

`POST /api/fleet/orbit/software_install/result`

##### Parameters

| Name                          | Type   | In   | Description                                             |
| ----------------------------- | ------ | ---- | ------------------------------------------------------- |
| orbit_node_key                | string | body | The Orbit node key for authentication.                  |
| host_id                       | number | body | The ID of the host on which the software was installed. |
| install_uuid                  | string | body | The UUID of the installation attempt.                   |
| pre_install_condition_output  | string | body | The output from the pre-install condition query.        |
| install_script_exit_code      | number | body | The exit code from the install script.                  |
| install_script_output         | string | body | The output from the install script.                     |
| post_install_script_exit_code | number | body | The exit code from the post-install script.             |
| post_install_script_output    | string | body | The output from the post-install script.                |

##### Example

`POST /api/fleet/orbit/software_install/result`

##### Request body

```json
{
  "orbit_node_key":"FbvSsWfTRwXEecUlCBTLmBcjGFAdzqd/",
  "host_id ": 12,
  "install_uuid ": "4D91F9C3-919B-4D5B-ABFC-528D648F27D1",
  "pre_install_condition_output ": "example",
  "install_script_exit_code ": 0,
  "install_script_output ": "software installed",
  "post_install_script_exit_code ": 1,
  "post_install_script_output ": "error: post-install script failed"
}
```

##### Default response

`Status: 204`

### Download software installer

`POST /api/fleet/orbit/software_install/package`

##### Parameters

| Name           | Type   | In    | Description                                                          |
| -------------- | ------ | ----- | -------------------------------------------------------------------- |
| orbit_node_key | string | body  | The Orbit node key for authentication.                               |
| installer_id   | number | body  | The ID of the software installer to download.                        |
| alt            | string | query | Indicates whether to download the package. Must be set to `"media"`. |

##### Example

`POST /api/fleet/orbit/software_install/package`

##### Request body

```json
{
  "orbit_node_key":"FbvSsWfTRwXEecUlCBTLmBcjGFAdzqd/",
  "installer_id": 15
}
```

##### Default response

`Status: 200`

```http
Status: 200
Content-Type: application/octet-stream
Content-Disposition: attachment
Content-Length: <length>
Body: <blob>
```

### Get orbit software install details

`POST /api/fleet/orbit/software_install/details`

##### Parameters

| Name           | Type   | In   | Description                                    |
| -------------- | ------ | ---- | ---------------------------------------------- |
| orbit_node_key | string | body | The Orbit node key for authentication.         |
| install_uuid   | string | body | The UUID of the software installation attempt. |

##### Example

`POST /api/fleet/orbit/software_install/details`

##### Request body

```json
{
  "orbit_node_key":"FbvSsWfTRwXEecUlCBTLmBcjGFAdzqd/",
  "install_uuid": "1652210E-619E-43BA-B3CC-17F4247823F3"
}
```

##### Default response

`Status: 200`

```json
{
  "install_id": "1652210E-619E-43BA-B3CC-17F4247823F3",
  "installer_id": 12,
  "pre_install_condition": "SELECT * FROM osquery_info;",
  "install_script": "sudo run-installer",
  "uninstall_script": "sudo run-uninstaller",
  "post_install_script": "echo done",
  "self_service": true,
}
```

### Upload disk encryption key

`POST /api/fleet/orbit/disk_encryption_key`

##### Parameters

| Name           | Type   | In   | Description                               |
| -------------- | ------ | ---- | ----------------------------------------- |
| orbit_node_key | string | body | The Orbit node key for authentication.    |
| encryption_key | string | body | The encryption key bytes.                 |
| client_error   | string | body | The error reported by the client, if any. |

##### Example

`POST /api/fleet/orbit/disk_encryption_key`

##### Request body

```json
{
  "orbit_node_key":"FbvSsWfTRwXEecUlCBTLmBcjGFAdzqd/",
  "encryption_key": "Zm9vYmFyem9vYmFyZG9vYmFybG9vYmFy",
  "client_error": "example error",
}
```

##### Default response

`Status: 204`

---

## Downloadable installers

These API routes are used by the UI in Fleet Sandbox.

- [Download an installer](#download-an-installer)
- [Check if an installer exists](#check-if-an-installer-exists)

### Download an installer

Downloads a pre-built fleet-osquery installer with the given parameters.

`POST /api/v1/fleet/download_installer/{kind}`

#### Parameters

| Name          | Type    | In                    | Description                                                        |
| ------------- | ------- | --------------------- | ------------------------------------------------------------------ |
| kind          | string  | path                  | The installer kind: pkg, msi, deb or rpm.                          |
| enroll_secret | string  | x-www-form-urlencoded | The global enroll secret.                                          |
| token         | string  | x-www-form-urlencoded | The authentication token.                                          |
| desktop       | boolean | x-www-form-urlencoded | Set to `true` to ask for an installer that includes Fleet Desktop. |

##### Default response

```http
Status: 200
Content-Type: application/octet-stream
Content-Disposition: attachment
Content-Length: <length>
Body: <blob>
```

If an installer with the provided parameters is found, the installer is returned as a binary blob in the body of the response.

##### Installer doesn't exist

`Status: 400`

This error occurs if an installer with the provided parameters doesn't exist.


### Check if an installer exists

Checks if a pre-built fleet-osquery installer with the given parameters exists.

`HEAD /api/v1/fleet/download_installer/{kind}`

#### Parameters

| Name          | Type    | In    | Description                                                        |
| ------------- | ------- | ----- | ------------------------------------------------------------------ |
| kind          | string  | path  | The installer kind: pkg, msi, deb or rpm.                          |
| enroll_secret | string  | query | The global enroll secret.                                          |
| desktop       | boolean | query | Set to `true` to ask for an installer that includes Fleet Desktop. |

##### Default response

`Status: 200`

If an installer with the provided parameters is found.

##### Installer doesn't exist

`Status: 400`

If an installer with the provided parameters doesn't exist.

## Setup

Sets up a new Fleet instance with the given parameters.

`POST /api/v1/setup`

#### Parameters

| Name       | Type   | In   | Description                                                                                                                 |
| ---------- | ------ | ---- | --------------------------------------------------------------------------------------------------------------------------- |
| admin      | object | body | **Required.** Contains the following admin user details: `admin`, `email`, `name`, `password`, and `password_confirmation`. |
| org_info   | object | body | **Required.** Contains the following organizational details: `org_name`.                                                    |
| server_url | string | body | **Required.** The URL of the Fleet instance.                                                                                |


##### Request body

```json
{
	"admin": {
		"admin": true,
		"email": "janedoe@example.com",
		"name": "Jane Doe",
		"password": "password!234",
		"password_confirmation": "password!234"
	},
	"org_info": {
		"org_name": "Fleet Device Management"
	},
	"server_url": "https://localhost:8080"
}
```

##### Default response

`Status: 200`

If the Fleet instance is provided required parameters to complete setup.

```json
{
  "admin": {
    "created_at": "2021-01-07T19:40:04Z",
    "updated_at": "2021-01-07T19:40:04Z",
    "id": 1,
    "name": "Jane Doe",
    "email": "janedoe@example.com",
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false,
    "global_role": "admin",
    "api_only": false,
    "teams": []
  },
  "org_info": {
    "org_name": "Fleet Device Management",
    "org_logo_url": "https://fleetdm.com/logo.png"
  },
  "server_url": "https://localhost:8080",
  "osquery_enroll_secret": null,
  "token": "ur4RWGBeiNmNzer/dnGzgUQ+jxrJe19xuHg/LhLkbhuZMQu35scyBHUHs68+RJxZynxQnuTz4WTHXayAJJaGgg=="
}

```

## Scripts

### Batch-apply scripts

_Available in Fleet Premium_

`POST /api/v1/fleet/scripts/batch`

#### Parameters

| Name      | Type   | In    | Description                                                                                                                                                           |
| --------- | ------ | ----- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| team_id | number | query | The ID of the team to add the scripts to. Only one team identifier (`team_id` or `team_name`) can be included in the request, omit this parameter if using `team_name`.
| team_name | string | query | The name of the team to add the scripts to. Only one team identifier (`team_id` or `team_name`) can be included in the request, omit this parameter if using `team_id`.
| dry_run   | bool   | query | Validate the provided scripts and return any validation errors, but do not apply the changes.                                                                         |
| scripts   | array  | body  | An array of objects with the scripts payloads. Each item must contain `name` with the script name and `script_contents` with the script contents encoded in base64    |

If both `team_id` and `team_name` parameters are included, this endpoint will respond with an error. If no `team_name` or `team_id` is provided, the scripts will be applied for **all hosts**.

> Note that this endpoint replaces all the active scripts for the specified team (or no team). Any existing script that is not included in the list will be removed, and existing scripts with the same name as a new script will be edited. Providing an empty list of scripts will remove existing scripts.

#### Example

`POST /api/v1/fleet/scripts/batch`

##### Default response

`Status: 200`

```json
{
  "scripts": [
    {
      "team_id": 3,
      "id": 6690,
      "name": "Ensure shields are up"
    },
    {
      "team_id": 3,
      "id": 10412,
      "name": "Ensure flux capacitor is charged"
    }
  ]
}
```

### Run live script

Run a live script and get results back (5 minute timeout). Live scripts only runs on the host if it has no other scripts running.

`POST /api/v1/fleet/scripts/run/sync`

#### Parameters

| Name            | Type    | In   | Description                                                                                    |
| ----            | ------- | ---- | --------------------------------------------                                                   |
| host_id         | integer | body | **Required**. The ID of the host to run the script on.                                                |
| script_id       | integer | body | The ID of the existing saved script to run. Only one of either `script_id`, `script_contents`, or `script_name` can be included. |
| script_contents | string  | body | The contents of the script to run. Only one of either `script_id`, `script_contents`, or `script_name` can be included. |
| script_name       | integer | body | The name of the existing saved script to run. If specified, requires `team_id`. Only one of either `script_id`, `script_contents`, or `script_name` can be included.   |
| team_id       | integer | body | The ID of the existing saved script to run. If specified, requires `script_name`. Only one of either `script_id`, `script_contents`, or `script_name` can be included in the request.  |

> Note that if any combination of `script_id`, `script_contents`, and `script_name` are included in the request, this endpoint will respond with an error.

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
## Software

### Batch-apply software

_Available in Fleet Premium._

`POST /api/v1/fleet/software/batch`

This endpoint is asynchronous, meaning it will start a background process to download and apply the software and return a `request_uuid` in the JSON response that can be used to query the status of the batch-apply (using the `GET /api/v1/fleet/software/batch/:request_uuid` endpoint defined below).

#### Parameters

| Name      | Type   | In    | Description                                                                                                                                                           |
| --------- | ------ | ----- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| team_name | string | query | The name of the team to add the software package to. Ommitting these parameters will add software to 'No Team'. |
| dry_run   | bool   | query | If `true`, will validate the provided software packages and return any validation errors, but will not apply the changes.                                                                         |
| software  | object   | body  | The team's software that will be available for install.  |
| software.packages   | array   | body  | An array of objects with values below. |
| software.packages.url                      | string   | body  | URL to the software package (PKG, MSI, EXE or DEB). |
| software.packages.install_script           | string   | body  | Command that Fleet runs to install software. |
| software.packages.pre_install_query        | string   | body  | Condition query that determines if the install will proceed. |
| software.packages.post_install_script      | string   | body  | Script that runs after software install. |
| software.packages.uninstall_script      | string   | body  | Command that Fleet runs to uninstall software. |
| software.packages.self_service           | boolean   | body  | Condition query that determines if the install will proceed. |
| software.packages.labels_include_any     | array   | body  | Target hosts that have any label in the array. Only one of `labels_include_any` or `labels_exclude_any` can be included. If neither are included, all hosts are targeted. |
| software.packages.labels_exclude_any     | array   | body  | Target hosts that don't have any labels in the array. Only one of `labels_include_any` or `labels_exclude_any` can be included. If neither are included, all hosts are targeted. |

#### Example

`POST /api/v1/fleet/software/batch`

##### Default response

`Status: 202`
```json
{
  "request_uuid": "ec23c7b6-c336-4109-b89d-6afd859659b4",
}
```

### Get status of software batch-apply request

_Available in Fleet Premium._

`GET /api/v1/fleet/software/batch/:request_uuid`

This endpoint allows querying the status of a batch-apply software request (`POST /api/v1/fleet/software/batch`).
Returns `"status"` field that can be one of `"processing"`, `"complete"` or `"failed"`.
If `"status"` is `"completed"` then the `"packages"` field contains the applied packages.
If `"status"` is `"processing"` then the operation is ongoing and the request should be retried.
If `"status"` is `"failed"` then the `"message"` field contains the error message.

#### Parameters

| Name         | Type   | In    | Description                                                                                                                                                           |
| ------------ | ------ | ----- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| request_uuid | string | query | The request_uuid returned by the `POST /api/v1/fleet/software/batch` endpoint. |
| team_name    | string | query | The name of the team to add the software package to. Ommitting these parameters will add software to 'No Team'. |
| dry_run      | bool   | query | If `true`, will validate the provided software packages and return any validation errors, but will not apply the changes.                                                                         |

##### Default responses

`Status: 200`
```json
{
  "status": "processing",
  "message": "",
  "packages": null
}
```

`Status: 200`
```json
{
  "status": "completed",
  "message": "",
  "packages": [
    {
      "team_id": 1,
      "title_id": 2751,
      "url": "https://ftp.mozilla.org/pub/firefox/releases/129.0.2/win64/en-US/Firefox%20Setup%20129.0.2.msi"
    }
  ]
}
```

`Status: 200`
```json
{
  "status": "failed",
  "message": "validation failed: software.url Couldn't edit software. URL (\"https://foobar.does.not.exist.com\") returned \"Not Found\". Please make sure that URLs are reachable from your Fleet server.",
  "packages": null
}
```

### Batch-apply App Store apps

_Available in Fleet Premium._

`POST /api/latest/fleet/software/app_store_apps/batch`

#### Parameters

| Name      | Type   | In    | Description                                                                                                                                                           |
| --------- | ------ | ----- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| team_name | string | query | The name of the team to add the software package to. Ommitting this parameter will add software to "No team". |
| dry_run   | bool   | query | If `true`, will validate the provided VPP apps and return any validation errors, but will not apply the changes.                                                                         |
| app_store_apps | list   | body  | An array of objects. Each object contains `app_store_id` and `self_service`. |
| app_store_apps.app_store_id | string   | body  | ID of the App Store app. |
| app_store_apps.self_service | boolean   | body  | Whether the VPP app is "Self-service" or not. |

#### Example

`POST /api/latest/fleet/software/app_store_apps/batch`
```json
{
  "team_name": "Foobar",
  "app_store_apps": {
    {
      "app_store_id": "597799333",
      "self_service": false
    },
    {
      "app_store_id": "497799835",
      "self_service": true,
    }
  }
}
```

##### Default response

`Status: 204`

### Get token to download package

_Available in Fleet Premium._

`POST /api/v1/fleet/software/titles/:software_title_id/package/token?alt=media`

The returned token is a one-time use token that expires after 10 minutes.

#### Parameters

| Name              | Type    | In    | Description                                                      |
|-------------------|---------|-------|------------------------------------------------------------------|
| software_title_id | integer | path  | **Required**. The ID of the software title for software package. |
| team_id           | integer | query | **Required**. The team ID containing the software package.       |
| alt               | integer | query | **Required**. Must be specified and set to "media".              |

#### Example

`POST /api/v1/fleet/software/titles/123/package/token?alt=media&team_id=2`

##### Default response

`Status: 200`

```json
{
  "token": "e905e33e-07fe-4f82-889c-4848ed7eecb7"
}
```

### Download package using a token

_Available in Fleet Premium._

`GET /api/v1/fleet/software/titles/:software_title_id/package/token/:token?alt=media`

#### Parameters

| Name              | Type    | In   | Description                                                              |
|-------------------|---------|------|--------------------------------------------------------------------------|
| software_title_id | integer | path | **Required**. The ID of the software title to download software package. |
| token             | string  | path | **Required**. The token to download the software package.                |

#### Example

`GET /api/v1/fleet/software/titles/123/package/token/e905e33e-07fe-4f82-889c-4848ed7eecb7`

##### Default response

`Status: 200`

```http
Status: 200
Content-Type: application/octet-stream
Content-Disposition: attachment
Content-Length: <length>
Body: <blob>
```
