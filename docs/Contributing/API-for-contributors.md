# API for contributors

- [Packs](#packs)
- [Get or apply configuration files](#get-or-apply-configuration-files)
- [Live query](#live-query)
- [Device-authenticated routes](#device-authenticated-routes)
- [Downloadable installers](#downloadable-installers)
- [Setup](#setup)

This document includes the Fleet API routes that are helpful when developing or contributing to Fleet.

Unlike the [Fleet REST API documentation](https://fleetdm.com/docs/using-fleet/rest-api), only the Fleet UI, Fleet Desktop, and `fleetctl` clients use the API routes in this document:

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
      "team_ids": [],
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
    },
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

## Get or apply configuration files

These API routes are used by the `fleetctl` CLI tool. Users can manage Fleet with `fleetctl` and [configuration files in YAML syntax](https://fleetdm.com/docs/using-fleet/configuration-files/).

- [Get queries](#get-queries)
- [Get query](#get-query)
- [Apply queries](#apply-queries)
- [Apply policies](#aaply-policies)
- [Get packs](#get-packs)
- [Apply packs](#apply-packs)
- [Get pack by name](#get-pack-by-name)
- [Apply team](#apply-team)
- [Apply labels](#apply-labels)
- [Get labels](#get-labels)
- [Get label](#get-label-spec)
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

`POST /api/v1/fleet/spec/queries`

#### Parameters

| Name  | Type | In   | Description                                                      |
| ----- | ---- | ---- | ---------------------------------------------------------------- |
| specs | list | body | **Required.** The list of the queries to be created or modified. |

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
      "resolution": "some resolution steps here"
    },
    {
      "name": "Is FileVault enabled on macOS devices?",
      "query": "SELECT 1 FROM disk_encryption WHERE user_uuid IS NOT “” AND filevault_status = ‘on’ LIMIT 1;",
      "description": "Checks to make sure that the FileVault feature is enabled on macOS devices.",
      "resolution": "Choose Apple menu > System Preferences, then click Security & Privacy. Click the FileVault tab. Click the Lock icon, then enter an administrator name and password. Click Turn On FileVault.",
      "platform": "darwin"
    }
  ]
}
```

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

| Name          | Type   | In    | Description                                                                                                                                                                                                                         |
| ------------- | ------ | ----  | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| name          | string | body  | **Required.** The team's name.                                                                                                                                                                                                      |
| agent_options | object | body  | The agent options spec that is applied to the hosts assigned to the specified to team. These agent options completely override the global agent options specified in the [`GET /api/v1/fleet/config API route`](#get-configuration) |
| features      | object | body  | The features that are applied to the hosts assigned to the specified to team. These features completely override the global features specified in the [`GET /api/v1/fleet/config API route`](#get-configuration)                    |
| secrets       | list   | body  | A list of plain text strings is used as the enroll secrets. Existing secrets are replaced with this list, or left unmodified if this list is empty. Note that there is a limit of 50 secrets allowed.                               |
| force         | bool   | query | Force apply the spec even if there are (ignorable) validation errors. Those are unknown keys and agent options-related validations.                                                                                                 |
| dry_run       | bool   | query | Validate the provided JSON for unknown keys and invalid value types and return any validation errors, but do not apply the changes.                                                                                                 |

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
      ]
t    }
  ]
}
```

#### Default response

`Status: 200`

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

| Name    | Type   | In   | Description                                                    |
| ------  | ------ | ---- | -------------------------------------------------------------- |
| secrets | list   | body | **Required.** The plain text string used as the enroll secret. Note that there is a limit of 50 secrets allowed. |

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
- [Retrieve live query results (standard WebSocket API)](#retrieve-live-query-results-standard-web-socket-api)
- [Retrieve live query results (SockJS)](#retrieve-live-query-results-sock-js)

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

| Name              | Type    | In   | Description                                                                                                                                      |
|-------------------|---------|------|--------------------------------------------------------------------------------------------------------------------------------------------------|
| query             | string  | body | The query used to identify hosts to target. Searchable items include a host's hostname or IPv4 address.                                          |
| query_id          | integer | body | The saved query (if any) that will be run. The `observer_can_run` property on the query and the user's roles affect which targets are included.  |
| excluded_host_ids | array   | body | The list of host ids to omit from the search results.                                                           |

#### Example

`POST /api/v1/fleet/targets/search`

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

| Name     | Type    | In   | Description                                                                                                                                         |
|----------|---------|------|-----------------------------------------------------------------------------------------------------------------------------------------------------|
| query_id | integer | body | The saved query (if any) that will be run. The `observer_can_run` property on the query and the user's roles determine which targets are included.  |
| selected | object  | body | The object includes lists of selected host IDs, label IDs, and team IDs.                                                                            |

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

| Name     | Type    | In   | Description                                                                                                                                                           |
| -------- | ------- | ---- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| query    | string  | body | The SQL if using a custom query.                                                                                                                                      |
| query_id | integer | body | The saved query (if any) that will be run. Required if running query as an observer. The `observer_can_run` property on the query effects which targets are included. |
| selected | object  | body | **Required.** The desired targets for the query specified by ID. This object can contain `hosts`, `labels`, and/or `teams` properties. See examples below.            |

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

| Name     | Type    | In   | Description                                                                                                                                                  |
| -------- | ------- | ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| query    | string  | body | The SQL of the query.                                                                                                                                        |
| query_id | integer | body | The saved query (if any) that will be run. The `observer_can_run` property on the query effects which targets are included.                                  |
| selected | object  | body | **Required.** The desired targets for the query specified by name. This object can contain `hosts`, `labels`, and/or `teams` properties. See examples below. |

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

```
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
        // host data
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

```
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
        // host data
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

| Name            | Type   | In    | Description                                        |
| --------------- | ------ | ----- | ---------------------------------------------------|
| name            | string | query | The name of the cron schedule to trigger.          |

#### Example

`POST /api/latest/fleet/trigger?name=automations`

##### Default response

`Status: 200`

---

### Device-authenticated routes

Device-authenticated routes are routes used by the Fleet Desktop application. Unlike most other routes, Fleet user's API token does not authenticate them. They use a device-specific token.

- [Get device's host](#get-devices-host)
- [Refetch device's host](#refetch-devices-host)
- [Get device's Google Chrome profiles](#get-devices-google-chrome-profiles)
- [Get device's mobile device management (MDM) and Munki information](#get-devices-mobile-device-management-mdm-and-munki-information)
- [Get device's policies](#get-devices-policies)
- [Get device's API features](#get-devices-api-features)
- [Get device's transparency URL](#get-devices-transparency-url)

#### Get device's host

Returns the host information about the device that makes the request.

`GET /api/v1/fleet/device/{token}`

##### Parameters

| Name            | Type   | In    | Description                                        |
| --------------- | ------ | ----- | ---------------------------------------------------|
| token           | string | path  | The device's authentication token.                 |

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
    ]
  },
  "org_logo_url": "https://example.com/logo.jpg",
  "license": {
    "tier": "free",
    "expiration": "2031-01-01T00:00:00Z"
  }
}
```

#### Refetch device's host

Same as [Refetch host route](https://fleetdm.com/docs/using-fleet/rest-api#refetch-host) for the current device.

`POST /api/v1/fleet/device/{token}/refetch`

##### Parameters

| Name            | Type   | In    | Description                            |
| --------------- | ------ | ----- | ---------------------------------------|
| token           | string | path  | The device's authentication token.     |

#### Get device's Google Chrome profiles

Same as [Get host's Google Chrome profiles](https://fleetdm.com/docs/using-fleet/rest-api#get-hosts-google-chrome-profiles) for the current device.

`GET /api/v1/fleet/device/{token}/device_mapping`

##### Parameters

| Name            | Type   | In    | Description                            |
| --------------- | ------ | ----- | ---------------------------------------|
| token           | string | path  | The device's authentication token.     |

#### Get device's mobile device management (MDM) and Munki information

Same as [Get host's mobile device management and Munki information](https://fleetdm.com/docs/using-fleet/rest-api#get-hosts-mobile-device-management-mdm-and-munki-information) for the current device.

`GET /api/v1/fleet/device/{token}/macadmins`

##### Parameters

| Name            | Type   | In    | Description                            |
| --------------- | ------ | ----- | ---------------------------------------|
| token           | string | path  | The device's authentication token.     |


#### Get Fleet Desktop information
_Available in Fleet Premium_

Gets all information required by Fleet Desktop to notify the user if there are any failing policies.

`GET /api/v1/fleet/device/{token}/desktop`

##### Parameters

| Name            | Type   | In    | Description                            |
| --------------- | ------ | ----- | ---------------------------------------|
| token           | string | path  | The device's authentication token.     |

##### Example

`GET /api/v1/fleet/device/abcdef012456789/desktop`

##### Default response

`Status: 200`

```json
{
  "failing_policies_count": 3
}
```



#### Get device's policies

_Available in Fleet Premium_

Lists the policies applied to the current device.

`GET /api/v1/fleet/device/{token}/policies`

##### Parameters

| Name            | Type   | In    | Description                            |
| --------------- | ------ | ----- | ---------------------------------------|
| token           | string | path  | The device's authentication token.     |

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

| Name            | Type   | In    | Description                            |
| --------------- | ------ | ----- | ---------------------------------------|
| token           | string | path  | The device's authentication token.     |

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

Returns the URL to open when clicking the "Transparency" menu item in Fleet Desktop. Note that _Fleet Premium_ is required to configure a custom transparency URL.

`GET /api/v1/fleet/device/{token}/transparency`

##### Parameters

| Name            | Type   | In    | Description                            |
| --------------- | ------ | ----- | ---------------------------------------|
| token           | string | path  | The device's authentication token.     |

##### Example

`GET /api/v1/fleet/device/abcdef012456789/transparency`

##### Default response

`Status: 307`

Redirects to the transparency URL.

---

## Downloadable installers

These API routes are used by the UI in Fleet Sandbox.

- [Download an installer](#download-an-installer)
- [Check if an installer exists](#check-if-an-installer-exists)

### Download an installer

Downloads a pre-built fleet-osquery installer with the given parameters.

`POST /api/v1/fleet/download_installer/{kind}`

#### Parameters

| Name          | Type    | In                     | Description                                                        |
| ------------- | ------- | ---------------------- | ------------------------------------------------------------------ |
| kind          | string  | path                   | The installer kind: pkg, msi, deb or rpm.                          |
| enroll_secret | string  | x-www-form-urlencoded  | The global enroll secret.                                          |
| token         | string  | x-www-form-urlencoded  | The authentication token.                                          |
| desktop       | boolean | x-www-form-urlencoded  | Set to `true` to ask for an installer that includes Fleet Desktop. |

##### Default response

```
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

### Setup

Sets up a new Fleet instance with the given parameters.

`POST /api/v1/setup`

#### Parameters

| Name          | Type    | In                     | Description                                                        |
| ------------- | ------- | ---------------------- | ------------------------------------------------------------------ |
| admin         | object  | body                   | **Required.** Contains the following admin user details: `admin`, `email`, `name`, `password`, and `password_confirmation`.                        |
| org_info      | object  | body                   | **Required.** Contains the following organizational details: `org_name`.                         |
| server_url    | string  | body                   | **Required.** The URL of the Fleet instance.                                      |


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

<meta name="pageOrderInSection" value="800">
