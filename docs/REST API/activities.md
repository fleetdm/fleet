# Activities

## List activities

Returns a list of the activities that have been performed in Fleet as well as additional metadata.
for pagination. For a comprehensive list of activity types and detailed information, please see the [audit logs](https://fleetdm.com/docs/using-fleet/audit-activities) page.

`GET /api/v1/fleet/activities`

### Parameters

| Name            | Type    | In    | Description                                                 |
|:--------------- |:------- |:----- |:------------------------------------------------------------|
| page            | integer | query | Page number of the results to fetch.                                                                                          |
| per_page        | integer | query | Results per page.                                                                                                             |
| order_key       | string  | query | What to order results by. Can be any column in the `activites` table.                                                         |
| order_direction | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |

### Example

`GET /api/v1/fleet/activities?page=0&per_page=10&order_key=created_at&order_direction=desc`

#### Default response

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
        "status": "failed"
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

<meta name="description" value="Documentation for Fleet's activity REST API endpoints.">
<meta name="pageOrderInSection" value="10">