# REST API

- [Queries](#queries)
- [Schedule](#schedule)

## Queries

- [Get query](#get-query)
- [List queries](#list-queries)
- [Create query](#create-query)
- [Modify query](#modify-query)
- [Delete query by name](#delete-query-by-name)
- [Delete query by ID](#delete-query-by-id)
- [Delete queries](#delete-queries)
- [Run live query](#run-live-query)
- [Team queries](#team-queries)

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
    "team_id": null,
    "query": "select 1 from os_version where platform = \"centos\";",
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
    ]
  }
}
```

### List queries

Returns a list of all queries in the Fleet instance (both global and team queries).

> `team_id` will be blank for global queries.

`GET /api/v1/fleet/queries`

#### Parameters

| Name            | Type   | In    | Description                                                                                                                   |
| --------------- | ------ | ----- | ----------------------------------------------------------------------------------------------------------------------------- |
| order_key       | string | query | What to order results by. Can be any column in the queries table.                                                             |
| order_direction | string | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |

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
    "team_id": 123,
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
    ]
  }
]}
```

### Create query
Creates a global query.

`POST /api/v1/fleet/queries`

#### Parameters

| Name             | Type   | In   | Description                                                                                                                                            |
| ---------------- | ------ | ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| name             | string | body | **Required**. The name of the query.                                                                                                                   |
| query            | string | body | **Required**. The query in SQL syntax.                                                                                                                 |
| description      | string | body | The query's description.                                                                                                                               |
| observer_can_run | bool   | body | Whether or not users with the `observer` role can run the query. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). This field is only relevant for the `observer` role. The `observer_plus` role can run any query and is not limited by this flag (`observer_plus` role was added in Fleet 4.30.0). |

#### Example

`POST /api/v1/fleet/queries`

##### Request body

```json
{
  "description": "This is a new query.",
  "name": "new_query",
  "query": "SELECT * FROM osquery_info"
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
    "team_id": null,
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

### Modify query

Modifies the query specified by ID with the data submitted in the request body.

`PATCH /api/v1/fleet/queries/{id}`

#### Parameters

| Name             | Type    | In   | Description                                                                                                                                            |
| ---------------- | ------- | ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| id               | integer | path | **Required.** The ID of the query.                                                                                                                     |
| name             | string  | body | The name of the query.                                                                                                                                 |
| query            | string  | body | The query in SQL syntax.                                                                                                                               |
| description      | string  | body | The query's description.                                                                                                                               |
| observer_can_run | bool    | body | Whether or not users with the `observer` role can run the query. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). This field is only relevant for the `observer` role. The `observer_plus` role can run any query and is not limited by this flag (`observer_plus` role was added in Fleet 4.30.0). |

#### Example

`PATCH /api/v1/fleet/queries/2`

##### Request body

```json
{
  "name": "new_title_for_my_query"
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
    "team_id": null,
    "query": "SELECT * FROM osquery_info",
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

| Name | Type   | In   | Description                          |
| ---- | ------ | ---- | ------------------------------------ |
| name | string | path | **Required.** The name of the query. |

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

### Team queries

- [List team queries](#list-team-queries)
- [Get team query](#get-team-query)
- [Create team query](#create-team-query)
- [Modify team query](#modify-team-query)
- [Delete team query by name](#delete-team-query-by-name)
- [Delete team query by ID](#delete-team-query-by-id)
- [Delete team queries](#delete-team-queries)

### List team queries
Returns a list of all queries in the specified team.

`GET /api/v1/fleet/team/{id}/queries`

#### Parameters

| Name            | Type    | In    | Description                                                                                                                   |
| --------------- | ------  | ----- | ----------------------------------------------------------------------------------------------------------------------------- |
| id              | integer | path  | **Required** The ID of the parent team.
| order_key       | string  | query | What to order results by. Can be any column in the queries table.                                                             |
| order_direction | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |

#### Example

`GET /api/v1/fleet/team/123/queries`

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
    "team_id": 123,
    "description": "query",
    "query": "SELECT * FROM osquery_info",
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
    "team_id": 123,
    "query": "select name, interval, executions, output_size, wall_time, (user_time/executions) as avg_user_time, (system_time/executions) as avg_system_time, average_memory, last_executed from osquery_schedule;",
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
    ]
  }
]}
```

### Get team query
Returns the query specified by ID from the among the team queries specified by team_id.

`GET /api/v1/fleet/team/{team_id}/queries/{id}`

#### Parameters

| Name      | Type    | In   | Description                                      |
| ----      | ------- | ---- | ------------------------------------------------ |
| id        | integer | path | **Required**. The id of the desired team query.  |
| team_id   | integer | path | **Required**. The id of the query's parent team. |

#### Example

`GET /api/v1/fleet/team/123/queries/31`

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
    "team_id": 123,
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
    ]
  }
}
```

### Create team query
Creates a query in the specified team.

`POST /api/v1/fleet/team/{team_id}/queries`

#### Parameters

| Name             | Type    | In   | Description                                                                                                                                            |
| ---------------- | ------  | ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| team_id          | integer | path | **Required**. The id of the parent team                                                                                                                |
| name             | string  | body | **Required**. The name of the query.                                                                                                                   |
| query            | string  | body | **Required**. The query in SQL syntax.                                                                                                                 |
| description      | string  | body | The query's description.                                                                                                                               |
| observer_can_run | bool    | body | Whether or not users with the `observer` role can run the query. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). This field is only relevant for the `observer` role. The `observer_plus` role can run any query and is not limited by this flag (`observer_plus` role was added in Fleet 4.30.0). |

#### Example

`POST /api/v1/fleet/team/123/queries`

##### Request body

```json
{
  "description": "This is a new query.",
  "name": "new_query",
  "query": "SELECT * FROM osquery_info"
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
    "team_id": 123,
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

### Modify team query
Modifies the team query specified by ID and team_id with the data submitted in the request body.

`PATCH /api/v1/fleet/team/{team_id}/queries/{id}`

#### Parameters

| Name             | Type    | In   | Description                                                                                                                                            |
| ---------------- | ------- | ---- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| id               | integer | path | **Required.** The ID of the query.                                                                                                                     |
| team_id          | integer | path | **Required.** The ID of the parent team.
| name             | string  | body | The name of the query.                                                                                                                                 |
| query            | string  | body | The query in SQL syntax.                                                                                                                               |
| description      | string  | body | The query's description.                                                                                                                               |
| observer_can_run | bool    | body | Whether or not users with the `observer` role can run the query. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). This field is only relevant for the `observer` role. The `observer_plus` role can run any query and is not limited by this flag (`observer_plus` role was added in Fleet 4.30.0). |

#### Example

`PATCH /api/v1/fleet/team/123/queries/2`

##### Request body

```json
{
  "name": "new_title_for_my_query"
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
    "team_id": 123,
    "query": "SELECT * FROM osquery_info",
    "saved": true,
    "author_id": 1,
    "author_name": "noah",
    "observer_can_run": true,
    "packs": []
  }
}
```
### Delete team query by name

Deletes the team query specified by name and team_id.

`DELETE /api/v1/fleet/team/{team_id}/queries/{name}`

#### Parameters

| Name    | Type    | In   | Description                              |
| ------- | ------- | ---- | ---------------------------------------- |
| name    | string  | path | **Required.** The name of the query.     |
| team_id | integer | path | **Required.** The id of the parent team. |

#### Example

`DELETE /api/v1/fleet/team/123/queries/my_query`

##### Default response

`Status: 200`

### Delete team query by ID

Deletes the team query specified by ID and team_id.

`DELETE /api/v1/fleet/team/{team_id}/queries/id/{id}`

#### Parameters

| Name    | Type    | In   | Description                              |
| ------- | ------- | ---- | ---------------------------------------- |
| id      | integer | path | **Required.** The ID of the query.       |
| team_id | integer | path | **Required.** The id of the parent team. |

#### Example

`DELETE /api/v1/fleet/team/123/queries/id/28`

##### Default response

`Status: 200`


### Delete team queries

Deletes the queries specified by ID and team_id. Returns the count of queries successfully deleted.

`POST /api/v1/fleet/team/{team_id}/queries/delete`

#### Parameters

| Name    | Type    | In   | Description                              |
| ------- | ------- | ---- | ---------------------------------------- |
| ids     | list    | body | **Required.** The IDs of the queries.    |
| team_id | integer | path | **Required.** The id of the parent team. |

#### Example

`POST /api/v1/fleet/team/123/queries/delete`

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

---

## Schedule

- [Get schedule](#get-schedule)
- [Add query to schedule](#add-query-to-schedule)
- [Edit query in schedule](#edit-query-in-schedule)
- [Remove query from schedule](#remove-query-from-schedule)
- [Team schedule](#team-schedule)

Scheduling queries in Fleet is the best practice for collecting data from hosts.

These API routes let you control your scheduled queries.

### Get schedule

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

`DELETE /api/v1/fleet/global/schedule/{id}`

#### Parameters

None.

#### Example

`DELETE /api/v1/fleet/global/schedule/5`

##### Default response

`Status: 200`


---

### Team schedule

- [Get team schedule](#get-team-schedule)
- [Add query to team schedule](#add-query-to-team-schedule)
- [Edit query in team schedule](#edit-query-in-team-schedule)
- [Remove query from team schedule](#remove-query-from-team-schedule)

`In Fleet 4.2.0, the Team Schedule feature was introduced.`

This allows you to easily configure scheduled queries that will impact a whole team of devices.

#### Get team schedule

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
<meta name="pageOrderInSection" value="400">
