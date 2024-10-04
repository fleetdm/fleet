# Users

The Fleet server exposes a handful of API endpoints that handles common user management operations. All the following endpoints require prior authentication meaning you must first log in successfully before calling any of the endpoints documented below.

## List all users

Returns a list of all enabled users

`GET /api/v1/fleet/users`

### Parameters

| Name            | Type    | In    | Description                                                                                                                   |
| --------------- | ------- | ----- | ----------------------------------------------------------------------------------------------------------------------------- |
| query           | string  | query | Search query keywords. Searchable fields include `name` and `email`.                                                          |
| order_key       | string  | query | What to order results by. Can be any column in the users table.                                                               |
| order_direction | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`. |
| page            | integer | query | Page number of the results to fetch.                                                                                          |
| query           | string  | query | Search query keywords. Searchable fields include `name` and `email`.                                                          |
| per_page        | integer | query | Results per page.                                                                                                             |
| team_id         | integer | query | _Available in Fleet Premium_. Filters the users to only include users in the specified team.                                   |

### Example

`GET /api/v1/fleet/users`

#### Request query parameters

None.

#### Default response

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

#### Failed authentication

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

## Create a user account with an invitation

Creates a user account after an invited user provides registration information and submits the form.

`POST /api/v1/fleet/users`

### Parameters

| Name                  | Type   | In   | Description                                                                                                                                                                                                                                                                                                                                              |
| --------------------- | ------ | ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| email                 | string | body | **Required**. The email address of the user.                                                                                                                                                                                                                                                                                                             |
| invite_token          | string | body | **Required**. Token provided to the user in the invitation email.                                                                                                                                                                                                                                                                                        |
| name                  | string | body | **Required**. The name of the user.                                                                                                                                                                                                                                                                                                                      |
| password              | string | body | The password chosen by the user (if not SSO user).                                                                                                                                                                                                                                                                                                       |
| password_confirmation | string | body | Confirmation of the password chosen by the user.                                                                                                                                                                                                                                                                                                         |
| global_role           | string | body | The role assigned to the user. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). In Fleet 4.30.0 and 4.31.0, the `observer_plus` and `gitops` roles were introduced respectively. If `global_role` is specified, `teams` cannot be specified. For more information, see [manage access](https://fleetdm.com/docs/using-fleet/manage-access).                                                                                                                                                                        |
| teams                 | array  | body | _Available in Fleet Premium_. The teams and respective roles assigned to the user. Should contain an array of objects in which each object includes the team's `id` and the user's `role` on each team. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). In Fleet 4.30.0 and 4.31.0, the `observer_plus` and `gitops` roles were introduced respectively. If `teams` is specified, `global_role` cannot be specified. For more information, see [manage access](https://fleetdm.com/docs/using-fleet/manage-access). |

### Example

`POST /api/v1/fleet/users`

#### Request query parameters

```json
{
  "email": "janedoe@example.com",
  "invite_token": "SjdReDNuZW5jd3dCbTJtQTQ5WjJTc2txWWlEcGpiM3c=",
  "name": "janedoe",
  "password": "test-123",
  "password_confirmation": "test-123",
  "teams": [
    {
      "id": 2,
      "role": "observer"
    },
    {
      "id": 4,
      "role": "observer"
    }
  ]
}
```

#### Default response

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
    "global_role": "admin",
    "teams": []
  }
}
```

#### Failed authentication

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

#### Expired or used invite code

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

#### Validation failed

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

## Create a user account without an invitation

Creates a user account without requiring an invitation, the user is enabled immediately.
By default, the user will be forced to reset its password upon first login.

`POST /api/v1/fleet/users/admin`

### Parameters

| Name        | Type    | In   | Description                                                                                                                                                                                                                                                                                                                                              |
| ----------- | ------- | ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| email       | string  | body | **Required**. The user's email address.                                                                                                                                                                                                                                                                                                                  |
| name        | string  | body | **Required**. The user's full name or nickname.                                                                                                                                                                                                                                                                                                          |
| password    | string  | body | The user's password (required for non-SSO users).                                                                                                                                                                                                                                                                                                        |
| sso_enabled | boolean | body | Whether or not SSO is enabled for the user.                                                                                                                                                                                                                                                                                                              |
| api_only    | boolean | body | User is an "API-only" user (cannot use web UI) if true.                                                                                                                                                                                                                                                                                                  |
| global_role | string | body | The role assigned to the user. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). In Fleet 4.30.0 and 4.31.0, the `observer_plus` and `gitops` roles were introduced respectively. If `global_role` is specified, `teams` cannot be specified. For more information, see [manage access](https://fleetdm.com/docs/using-fleet/manage-access).                                                                                                                                                                        |
| admin_forced_password_reset    | boolean | body | Sets whether the user will be forced to reset its password upon first login (default=true) |
| teams                          | array   | body | _Available in Fleet Premium_. The teams and respective roles assigned to the user. Should contain an array of objects in which each object includes the team's `id` and the user's `role` on each team. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). In Fleet 4.30.0 and 4.31.0, the `observer_plus` and `gitops` roles were introduced respectively. If `teams` is specified, `global_role` cannot be specified. For more information, see [manage access](https://fleetdm.com/docs/using-fleet/manage-access). |

### Example

`POST /api/v1/fleet/users/admin`

#### Request body

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

#### Default response

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

#### User doesn't exist

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

## Get user information

Returns all information about a specific user.

`GET /api/v1/fleet/users/:id`

### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The user's id. |

### Example

`GET /api/v1/fleet/users/2`

#### Request query parameters

```json
{
  "id": 1
}
```

#### Default response

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
    "global_role": "admin",
    "api_only": false,
    "teams": []
  }
}
```

#### User doesn't exist

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

## Modify user

`PATCH /api/v1/fleet/users/:id`

### Parameters

| Name        | Type    | In   | Description                                                                                                                                                                                                                                                                                                                                              |
| ----------- | ------- | ---- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| id          | integer | path | **Required**. The user's id.                                                                                                                                                                                                                                                                                                                             |
| name        | string  | body | The user's name.                                                                                                                                                                                                                                                                                                                                         |
| position    | string  | body | The user's position.                                                                                                                                                                                                                                                                                                                                     |
| email       | string  | body | The user's email.                                                                                                                                                                                                                                                                                                                                        |
| sso_enabled | boolean | body | Whether or not SSO is enabled for the user.                                                                                                                                                                                                                                                                                                              |
| api_only    | boolean | body | User is an "API-only" user (cannot use web UI) if true.                                                                                                                                                                                                                                                                                                  |
| password    | string  | body | The user's current password, required to change the user's own email or password (not required for an admin to modify another user).                                                                                                                                                                                                                     |
| new_password| string  | body | The user's new password. |
| global_role | string  | body | The role assigned to the user. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). If `global_role` is specified, `teams` cannot be specified.                                                                                                                                                                         |
| teams       | array   | body | _Available in Fleet Premium_. The teams and respective roles assigned to the user. Should contain an array of objects in which each object includes the team's `id` and the user's `role` on each team. In Fleet 4.0.0, 3 user roles were introduced (`admin`, `maintainer`, and `observer`). If `teams` is specified, `global_role` cannot be specified. |

### Example

`PATCH /api/v1/fleet/users/2`

#### Request body

```json
{
  "name": "Jane Doe",
  "global_role": "admin"
}
```

#### Default response

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
    "api_only": false,
    "teams": []
  }
}
```

### Example (modify a user's teams)

`PATCH /api/v1/fleet/users/2`

#### Request body

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

#### Default response

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

## Delete user

Delete the specified user from Fleet.

`DELETE /api/v1/fleet/users/:id`

### Parameters

| Name | Type    | In   | Description                  |
| ---- | ------- | ---- | ---------------------------- |
| id   | integer | path | **Required**. The user's id. |

### Example

`DELETE /api/v1/fleet/users/3`

#### Default response

`Status: 200`


## Require password reset

The selected user is logged out of Fleet and required to reset their password during the next attempt to log in. This also revokes all active Fleet API tokens for this user. Returns the user object.

`POST /api/v1/fleet/users/:id/require_password_reset`

### Parameters

| Name  | Type    | In   | Description                                                                                    |
| ----- | ------- | ---- | ---------------------------------------------------------------------------------------------- |
| id    | integer | path | **Required**. The user's id.                                                                   |
| require | boolean | body | Whether or not the user is required to reset their password during the next attempt to log in. |

### Example

`POST /api/v1/fleet/users/123/require_password_reset`

#### Request body

```json
{
  "require": true
}
```

#### Default response

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
    "sso_enabled": false,
    "global_role": "observer",
    "teams": []
  }
}
```

## List a user's sessions

Returns a list of the user's sessions in Fleet.

`GET /api/v1/fleet/users/:id/sessions`

### Parameters

None.

### Example

`GET /api/v1/fleet/users/1/sessions`

#### Default response

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

## Delete a user's sessions

Deletes the selected user's sessions in Fleet. Also deletes the user's API token.

`DELETE /api/v1/fleet/users/:id/sessions`

### Parameters

| Name | Type    | In   | Description                               |
| ---- | ------- | ---- | ----------------------------------------- |
| id   | integer | path | **Required**. The ID of the desired user. |

### Example

`DELETE /api/v1/fleet/users/1/sessions`

#### Default response

`Status: 200`

---

<meta name="description" value="Documentation for Fleet's users REST API endpoints.">
<meta name="pageOrderInSection" value="180">