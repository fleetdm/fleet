# API for contributors

- [Authentication](#authentication)
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
- [Users](#users)

> These endpoints are used by the Fleet UI, Fleet Desktop, and `fleetctl` clients and frequently change to reflect current functionality.

This document includes the internal Fleet API routes that are helpful when developing or contributing to Fleet.

If you are interested in gathering information from Fleet in a production environment, please see the [public Fleet REST API documentation](https://fleetdm.com/docs/using-fleet/rest-api).

## Authentication

### Create session

`POST /api/v1/fleet/sessions`

#### Parameters

| Name | Type | In | Description |
| token | string | body | **Required**. The token retrieved from the magic link email. |

#### Response

See [the Log in endpoint](https://fleetdm.com/docs/rest-api/rest-api#log-in) for the current
successful response format.

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

[...4139 lines omitted...]

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

### Update software title name

`PATCH /api/v1/fleet/software/titles/:software_title_id/name`

Only available for software titles that have a non-empty bundle ID, as titles without a bundle
ID will be added back as new rows on the next software ingest with the same name. Endpoint authorization limited
to global admins as this changes the software title's name across all teams.

> **Experimental endpoint**. This endpoint is not guaranteed to continue to exist on future minor releases of Fleet.

#### Parameters

| Name              | Type    | In   | Description                                        |
|-------------------|---------|------|----------------------------------------------------|
| software_title_id | integer | path | **Required**. The ID of the software title to modify. |
| name              | string  | body | **Required**. The new name of the title.           |

#### Example

`PATCH /api/v1/fleet/software/titles/1/name`

```json
{
  "name": "2 Chrome 2 Furious.app"
}
```

##### Default response

`Status: 205`

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
| software.packages.self_service           | boolean   | body  | Specifies whether or not end users can install self-service. |
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
| app_store_apps | list   | body  | An array of objects with . Each object contains `app_store_id` and `self_service`. |
| app_store_apps.app_store_id | string   | body  | ID of the App Store app. |
| app_store_apps.self_service | boolean   | body  | Whether the VPP app is "Self-service" or not. |
| app_store_apps.labels_include_any | array   | body  | App will only be available for install on hosts that **have any** of these labels. Only one of either `labels_include_any` or `labels_exclude_any` can be included in the request. |
| app_store_apps.labels_exclude_any | array   | body  | App will only be available for install on hosts that **don't have any** of these labels. Only one of either `labels_include_any` or `labels_exclude_any` can be included in the request. |

#### Example

`POST /api/latest/fleet/software/app_store_apps/batch`
```json
{
  "team_name": "Foobar",
  "app_store_apps": [
    {
      "app_store_id": "597799333",
      "self_service": false,
      "labels_include_any": [
        "Engineering",
        "Customer Support"
      ]
    },
    {
      "app_store_id": "497799835",
      "self_service": true
    }
  ]
}
```

##### Default response

`Status: 200`

```json
{
  "app_store_apps": [
    {
      "team_id": 1,
      "title_id": 123,
      "app_store_id": "597799333",
      "platform": "darwin"
    },
    {
      "team_id": 1,
      "title_id": 124,
      "app_store_id": "597799333",
      "platform": "ios"
    },
    {
      "team_id": 1,
      "title_id": 125,
      "app_store_id": "597799333",
      "platform": "ipados"
    }
  ]
}
```

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

---

## Users

### Update user-specific UI settings

`PATCH /api/v1/fleet/users/:id`

#### Parameters

| Name      | Type   | In    | Description                                                                                                                                                           |
| --------- | ------ | ----- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| settings  | object   | body  | The updated user settings. |

#### Example

`PATCH /api/v1/fleet/users/1`
```json
{
  "hidden_host_columns": ["hostname"]
}
```

##### Default response
* Note that user settings are *not* included in this response. See below `GET`s for how to get user settings.
`Status: 200`
```json
{
    "user": {
        "created_at": "2025-01-08T01:04:23Z",
        "updated_at": "2025-01-09T00:08:19Z",
        "id": 1,
        "name": "Sum Bahdee",
        "email": "sum@org.com",
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

### Include settings when getting a user

`GET /api/v1/fleet/users/:id?include_ui_settings=true`

Use of `include_ui_settings=true` is considered the contributor API functionality – without that
param, this endpoint is considered a documented REST API endpoint

#### Parameters

| Name                | Type   | In    | Description                                                                                       |
| ------------------- | ------ | ----- | --------------------------------------------------------------------------------------------------|
| include_ui_settings | bool   | query | If `true`, will include the user's settings in the response. For now, this is a single ui setting.

#### Example

`GET /api/v1/fleet/users/2/?include_ui_settings=true`

##### Default response

`Status: 200`
```json
{
  "user": {...},
  "available_teams": {...}
  "settings": {"hidden_host_columns": ["hostname"]},
}
```

### Include settings when getting current user

`GET /api/v1/fleet/me/?include_ui_settings=true`

Use of `include_ui_settings=true` is considered the contributor API functionality – without that
param, this endpoint is considered a documented REST API endpoint

#### Parameters

| Name                | Type   | In    | Description                                                                                       |
| ------------------- | ------ | ----- | --------------------------------------------------------------------------------------------------|
| include_ui_settings | bool   | query | If `true`, will include the user's settings in the response. For now, this is a single ui setting.

#### Example

`GET /api/v1/fleet/me?include_ui_settings=true`

##### Default response

`Status: 200`
```json
{
  "user": {...},
  "available_teams": {...}
  "settings": {"hidden_host_columns": ["hostname"]},
}
