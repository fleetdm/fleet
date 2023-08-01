# Schedule (deprecated)

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility. 
> Please use the [queries](./Queries.md) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

- [Get schedule (deprecated)](#get-schedule)
- [Add query to schedule (deprecated)](#add-query-to-schedule)
- [Edit query in schedule (deprecated)](#edit-query-in-schedule)
- [Remove query from schedule (deprecated)](#remove-query-from-schedule)


Scheduling queries in Fleet is the best practice for collecting data from hosts.

These API routes let you control your scheduled queries.

## Get schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility. 
> Please use the [queries](./Queries.md) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

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

## Add query to schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility. 
> Please use the [queries](./Queries.md) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

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

## Edit query in schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility. 
> Please use the [queries](./Queries.md) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

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

## Remove query from schedule

`DELETE /api/v1/fleet/global/schedule/{id}`

#### Parameters

None.

#### Example

`DELETE /api/v1/fleet/global/schedule/5`

##### Default response

`Status: 200`


---

## Team schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility. 
> Please use the [queries](./Queries.md) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

- [Get team schedule (deprecated)](#get-team-schedule)
- [Add query to team schedule (deprecated)](#add-query-to-team-schedule)
- [Edit query in team schedule (deprecated)](#edit-query-in-team-schedule)
- [Remove query from team schedule (deprecated)](#remove-query-from-team-schedule)

This allows you to easily configure scheduled queries that will impact a whole team of devices.

### Get team schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility. 
> Please use the [queries](./Queries.md) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

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

### Add query to team schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility. 
> Please use the [queries](./Queries.md) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

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

### Edit query in team schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility. 
> Please use the [queries](./Queries.md) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

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

### Remove query from team schedule

> The schedule API endpoints are deprecated as of Fleet 4.35. They are maintained for backwards compatibility. 
> Please use the [queries](./Queries.md) endpoints, which as of 4.35 have attributes such as `interval` and `platform` that enable scheduling.

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


<meta name="pageOrderInSection" value="1100">
<meta name="title" value="Schedule (deprecated)">