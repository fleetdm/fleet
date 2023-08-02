# Queries

- [Get query](#get-query)
- [List queries](#list-queries)
- [Create query](#create-query)
- [Modify query](#modify-query)
- [Delete query by name](#delete-query-by-name)
- [Delete query by ID](#delete-query-by-id)
- [Delete queries](#delete-queries)
- [Run live query](#run-live-query)

## Get query

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

## List queries

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

## Create query

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
    "description": "This is a new query.",
    "query": "SELECT * FROM osquery_info",
    "saved": true,
    "author_id": 1,
    "author_name": "",
    "author_email": "",
    "observer_can_run": true,
    "packs": []
  }
}
```

## Modify query

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

## Delete query by name

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


## Delete query by ID

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


## Delete queries

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

## Run live query

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

<meta name="description" value="Learn about the endpoints used to manage queries in Fleet's REST API.">
<meta name="pageOrderInSection" value="1000">