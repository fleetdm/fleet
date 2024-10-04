# Policies

Policies are yes or no questions you can ask about your hosts.

Policies in Fleet are defined by osquery queries.

A passing host answers "yes" to a policy if the host returns results for a policy's query.

A failing host answers "no" to a policy if the host does not return results for a policy's query.

For example, a policy might ask “Is Gatekeeper enabled on macOS devices?“ This policy's osquery query might look like the following: `SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;`

## List policies

`GET /api/v1/fleet/global/policies`

### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                                                                                                                                                                                 |
| ----------------------- | ------- | ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                                                                                                                                                                                        |
| per_page                | integer | query | Results per page.

### Example

`GET /api/v1/fleet/global/policies`

#### Default response

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

## Count policies

`GET /api/v1/fleet/policies/count`


### Parameters
| Name               | Type    | In   | Description                                                                                                   |
| ------------------ | ------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| query                 | string | query | Search query keywords. Searchable fields include `name`.  |

### Example

`GET /api/v1/fleet/policies/count`

#### Default response

`Status: 200`

```json
{
  "count": 43
}
```

---

## Get policy by ID

`GET /api/v1/fleet/global/policies/:id`

### Parameters

| Name               | Type    | In   | Description                                                                                                   |
| ------------------ | ------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| id                 | integer | path | **Required.** The policy's ID.                                                                                |

### Example

`GET /api/v1/fleet/global/policies/1`

#### Default response

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

## Add policy

`POST /api/v1/fleet/global/policies`

### Parameters

| Name        | Type    | In   | Description                          |
| ----------  | ------- | ---- | ------------------------------------ |
| name        | string  | body | The policy's name.                    |
| query       | string  | body | The policy's query in SQL.                    |
| description | string  | body | The policy's description.             |
| resolution  | string  | body | The resolution steps for the policy. |
| platform    | string  | body | Comma-separated target platforms, currently supported values are "windows", "linux", "darwin". The default, an empty string means target all platforms. |
| critical    | boolean | body | _Available in Fleet Premium_. Mark policy as critical/high impact. |

### Example (preferred)

`POST /api/v1/fleet/global/policies`

### Request body

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

#### Default response

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

## Remove policies

`POST /api/v1/fleet/global/policies/delete`

### Parameters

| Name     | Type    | In   | Description                                       |
| -------- | ------- | ---- | ------------------------------------------------- |
| ids      | array   | body | **Required.** The IDs of the policies to delete.  |

### Example

`POST /api/v1/fleet/global/policies/delete`

### Request body

```json
{
  "ids": [ 1 ]
}
```

#### Default response

`Status: 200`

```json
{
  "deleted": 1
}
```

## Edit policy

`PATCH /api/v1/fleet/global/policies/:id`

### Parameters

| Name        | Type    | In   | Description                          |
| ----------  | ------- | ---- | ------------------------------------ |
| id          | integer | path | The policy's ID.                     |
| name        | string  | body | The query's name.                    |
| query       | string  | body | The query in SQL.                    |
| description | string  | body | The query's description.             |
| resolution  | string  | body | The resolution steps for the policy. |
| platform    | string  | body | Comma-separated target platforms, currently supported values are "windows", "linux", "darwin". The default, an empty string means target all platforms. |
| critical    | boolean | body | _Available in Fleet Premium_. Mark policy as critical/high impact. |

### Example

`PATCH /api/v1/fleet/global/policies/42`

#### Request body

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

#### Default response

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

## Run automation for all failing hosts of a policy

Triggers [automations](https://fleetdm.com/docs/using-fleet/automations#policy-automations) for *all* hosts failing the specified policies, regardless of whether the policies were previously failing on those hosts.

`POST /api/v1/fleet/automations/reset`

### Parameters

| Name        | Type     | In   | Description                                              |
| ----------  | -------- | ---- | -------------------------------------------------------- |
| policy_ids  | array    | body | Filters to only run policy automations for the specified policies. |
| team_ids    | array    | body | _Available in Fleet Premium_. Filters to only run policy automations for hosts in the specified teams. |


### Example

`POST /api/v1/fleet/automations/reset`

#### Request body

```json
{
    "team_ids": [1],
    "policy_ids": [1, 2, 3]
}
```

#### Default response

`Status: 200`

```json
{}
```

---

# Team policies

- [List team policies](#list-team-policies)
- [Count team policies](#count-team-policies)
- [Get team policy by ID](#get-team-policy-by-id)
- [Add team policy](#add-team-policy)
- [Remove team policies](#remove-team-policies)
- [Edit team policy](#edit-team-policy)

_Available in Fleet Premium_

Team policies work the same as policies, but at the team level.

## List team policies

`GET /api/v1/fleet/teams/:id/policies`

### Parameters

| Name               | Type    | In   | Description                                                                                                   |
| ------------------ | ------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| id                 | integer | path  | **Required.** Defines what team ID to operate on                                                                            |
| merge_inherited  | boolean | query | If `true`, will return both team policies **and** inherited ("All teams") policies the `policies` list, and will not return a separate `inherited_policies` list. |
| query                 | string | query | Search query keywords. Searchable fields include `name`. |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                                                                                                                                                                                        |
| per_page                | integer | query | Results per page. |


### Example (default usage)

`GET /api/v1/fleet/teams/1/policies`

#### Default response

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
      "calendar_events_enabled": false
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

### Example (returns single list)

`GET /api/v1/fleet/teams/1/policies?merge_inherited=true`

#### Default response

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

## Count team policies

`GET /api/v1/fleet/team/:team_id/policies/count`

### Parameters
| Name               | Type    | In   | Description                                                                                                   |
| ------------------ | ------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| team_id                 | integer | path  | **Required.** Defines what team ID to operate on
| query                 | string | query | Search query keywords. Searchable fields include `name`. |
| merge_inherited     | boolean | query | If `true`, will include inherited ("All teams") policies in the count. |

### Example

`GET /api/v1/fleet/team/1/policies/count`

#### Default response

`Status: 200`

```json
{
  "count": 43
}
```

---

## Get team policy by ID

`GET /api/v1/fleet/teams/:team_id/policies/:policy_id`

### Parameters

| Name               | Type    | In   | Description                                                                                                   |
| ------------------ | ------- | ---- | ------------------------------------------------------------------------------------------------------------- |
| team_id            | integer | path  | **Required.** Defines what team ID to operate on                                                                            |
| policy_id                 | integer | path | **Required.** The policy's ID.                                                                                |

### Example

`GET /api/v1/fleet/teams/1/policies/43`

#### Default response

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
    "calendar_events_enabled": true
  }
}
```

## Add team policy

The semantics for creating a team policy are the same as for global policies, see [Add policy](#add-policy).

`POST /api/v1/fleet/teams/:id/policies`

### Parameters

| Name        | Type    | In   | Description                          |
| ----------  | ------- | ---- | ------------------------------------ |
| id         | integer | path | Defines what team ID to operate on.  |
| name        | string  | body | The policy's name.                    |
| query       | string  | body | The policy's query in SQL.                    |
| description | string  | body | The policy's description.             |
| resolution  | string  | body | The resolution steps for the policy. |
| platform    | string  | body | Comma-separated target platforms, currently supported values are "windows", "linux", "darwin". The default, an empty string means target all platforms. |
| critical    | boolean | body | _Available in Fleet Premium_. Mark policy as critical/high impact. |
| software_title_id  | integer | body | _Available in Fleet Premium_. ID of software title to install if the policy fails. |

Either `query` or `query_id` must be provided.

### Example

`POST /api/v1/fleet/teams/1/policies`

#### Request body

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

#### Default response

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
    }
  }
}
```

## Remove team policies

`POST /api/v1/fleet/teams/:team_id/policies/delete`

### Parameters

| Name     | Type    | In   | Description                                       |
| -------- | ------- | ---- | ------------------------------------------------- |
| team_id  | integer | path  | **Required.** Defines what team ID to operate on                |
| ids      | array   | body | **Required.** The IDs of the policies to delete.  |

### Example

`POST /api/v1/fleet/teams/1/policies/delete`

#### Request body

```json
{
  "ids": [ 1 ]
}
```

#### Default response

`Status: 200`

```json
{
  "deleted": 1
}
```

## Edit team policy

`PATCH /api/v1/fleet/teams/:team_id/policies/:policy_id`

### Parameters

| Name        | Type    | In   | Description                          |
| ----------  | ------- | ---- | ------------------------------------ |
| team_id     | integer | path | The team's ID.                       |
| policy_id   | integer | path | The policy's ID.                     |
| name        | string  | body | The query's name.                    |
| query       | string  | body | The query in SQL.                    |
| description | string  | body | The query's description.             |
| resolution  | string  | body | The resolution steps for the policy. |
| platform    | string  | body | Comma-separated target platforms, currently supported values are "windows", "linux", "darwin". The default, an empty string means target all platforms. |
| critical    | boolean | body | _Available in Fleet Premium_. Mark policy as critical/high impact. |
| calendar_events_enabled    | boolean | body | _Available in Fleet Premium_. Whether to trigger calendar events when policy is failing. |
| software_title_id  | integer | body | _Available in Fleet Premium_. ID of software title to install if the policy fails. |

### Example

`PATCH /api/v1/fleet/teams/2/policies/42`

#### Request body

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

#### Default response

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
    }
  }
}
```

---

<meta name="description" value="Documentation for Fleet's policy REST API endpoints.">
<meta name="pageOrderInSection" value="110">