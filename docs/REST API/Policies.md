# Policies

- [List policies](#list-policies)
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

## List policies

`GET /api/v1/fleet/global/policies`

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

## Get policy by ID

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

## Add policy

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

## Remove policies

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

## Edit policy

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

## Run Automation for all failing hosts of a policy.

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

## Team policies

- [List team policies](#list-team-policies)
- [Get team policy by ID](#get-team-policy-by-id)
- [Add team policy](#add-team-policy)
- [Remove team policies](#remove-team-policies)
- [Edit team policy](#edit-team-policy)

_Available in Fleet Premium_

Team policies work the same as policies, but at the team level.

## List team policies

`GET /api/v1/fleet/teams/{id}/policies`

#### Parameters

| Name               | Type    | In   | Description                                                                                                   |
| ------------------ | ------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| id                 | integer | url  | Required. Defines what team id to operate on                                                                            |

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

## Get team policy by ID

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

## Add team policy

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

## Remove team policies

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

## Edit team policy

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


<meta name="pageOrderInSection" value="900">