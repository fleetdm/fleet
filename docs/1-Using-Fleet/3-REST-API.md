# REST API
- [Overview](#overview)
  - [fleetctl](#fleetctl)
  - [Current API](#current-api)
- [Authentication](#authentication)
- [Hosts](#hosts)
- [Users](#users)
- [Queries](#queries)
- [Packs](#packs)
- [Fleet configuration](#fleet-configuration)
- [Osquery options](#osquery-options)

## Overview

Fleet is powered by a Go API server which serves three types of endpoints:

- Endpoints starting with `/api/v1/osquery/` are osquery TLS server API endpoints. All of these endpoints are used for talking to osqueryd agents and that's it.
- Endpoints starting with `/api/v1/kolide/` are endpoints to interact with the Fleet data model (packs, queries, scheduled queries, labels, hosts, etc) as well as application endpoints (configuring settings, logging in, session management, etc).
- All other endpoints are served the React single page application bundle. The React app uses React Router to determine whether or not the URI is a valid route and what to do.

Only osquery agents should interact with the osquery API, but we'd like to support the eventual use of the Fleet API extensively. The API is not very well documented at all right now, but we have plans to:

- Generate and publish detailed documentation via a tool built using [test2doc](https://github.com/adams-sarah/test2doc) (or similar).
- Release a JavaScript Fleet API client library (which would be derived from the [current](https://github.com/fleetdm/fleet/blob/master/frontend/kolide/index.js) JavaScript API client).
- Commit to a stable, standardized API format.

### fleetctl

Many of the operations that a user may wish to perform with an API are currently best performed via the [fleetctl](./2-fleetctl-CLI.md) tooling. These CLI tools allow updating of the osquery configuration entities, as well as performing live queries.

### Current API

The general idea with the current API is that there are many entities throughout the Fleet application, such as:

- Queries
- Packs
- Labels
- Hosts

Each set of objects follows a similar REST access pattern.

- You can `GET /api/v1/kolide/packs` to get all packs
- You can `GET /api/v1/kolide/packs/1` to get a specific pack.
- You can `DELETE /api/v1/kolide/packs/1` to delete a specific pack.
- You can `POST /api/v1/kolide/packs` (with a valid body) to create a new pack.
- You can `PATCH /api/v1/kolide/packs/1` (with a valid body) to modify a specific pack.

Queries, packs, scheduled queries, labels, invites, users, sessions all behave this way. Some objects, like invites, have additional HTTP methods for additional functionality. Some objects, such as scheduled queries, are merely a relationship between two other objects (in this case, a query and a pack) with some details attached.

All of these objects are put together and distributed to the appropriate osquery agents at the appropriate time. At this time, the best source of truth for the API is the [HTTP handler file](https://github.com/fleetdm/fleet/blob/master/server/service/handler.go) in the Go application. The REST API is exposed via a transport layer on top of an RPC service which is implemented using a micro-service library called [Go Kit](https://github.com/go-kit/kit). If using the Fleet API is important to you right now, being familiar with Go Kit would definitely be helpful.



## Authentication

- [Log in](#log-in)
- [Log out](#log-out)
- [Forgot password](#forgot-password)
- [Change password](#change-password)
- [Me](#me)
- [SSO config](#sso-config)
- [Initiate SSO](#initiate-sso)

All API requests to the Fleet server require API token authentication unless noted in the documentation.

To get an API token, send a request to the [login endpoint](#log-in):

```
{
  "token": "<your token>",
  "user": {
    "created_at": "2020-11-13T22:57:12Z",
    "updated_at": "2020-11-13T22:57:12Z",
    "id": 1,
    "username": "jane",
    "name": "",
    "email": "janedoe@example.com",
    "admin": true,
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false
  }
}
```

Then, use that API token to authenticate all subsequent API requests by sending it in the "Authorization" request header, prefixed with "Bearer ":

```
Authorization: Bearer <your token>
```

> For SSO users, username/password login is disabled.  The API token can instead be retrieved from the "Settings" page in the UI.

### Log in

Authenticates the user with the specified credentials. Use the token returned from this endpoint to authenticate further API requests.

`POST /api/v1/kolide/login`

#### Parameters

| Name     | Type   | In   | Description                                   |
| -------- | ------ | ---- | --------------------------------------------- |
| username | string | body | **Required**. The user's email.               |
| password | string | body | **Required**. The user's plain text password. |

#### Example

`POST /api/v1/kolide/login`

##### Request body

```
{
  "username": "janedoe@example.com",
  "password": "VArCjNW7CfsxGp67"
}
```

##### Default response

`Status: 200`

```
{
  "user": {
    "created_at": "2020-11-13T22:57:12Z",
    "updated_at": "2020-11-13T22:57:12Z",
    "id": 1,
    "username": "jane",
    "name": "",
    "email": "janedoe@example.com",
    "admin": true,
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false
  },
  "token": "{your token}"
}
```

---

### Log out

Logs out the authenticated user.

`POST /api/v1/kolide/logout`

#### Example

`POST /api/v1/kolide/logout`

##### Default response

`Status: 200`

---

### Forgot password

Sends a password reset email to the specified email. Requires that SMTP is configured for your Fleet server.

`POST /api/v1/kolide/forgot_password`

#### Parameters

| Name  | Type   | In   | Description                                                             |
| ----- | ------ | ---- | ----------------------------------------------------------------------- |
| email | string | body | **Required**. The email of the user requesting the reset password link. |

#### Example

`POST /api/v1/kolide/forgot_password`

##### Request body

```
{
  "email": "janedoe@example.com"
}
```

##### Default response

`Status: 200`

##### Unknown error

`Status: 500`

```
{
  "message": "Unknown Error",
  "errors": [
    {
      "name": "base",
      "reason": "email not configured",
    }
  ]
}
```

---

### Change password

`POST /api/v1/kolide/change_password`

Changes the password for the authenticated user.

#### Parameters

| Name         | Type   | In   | Description                            |
| ------------ | ------ | ---- | -------------------------------------- |
| old_password | string | body | **Required**. The user's old password. |
| new_password | string | body | **Required**. The user's new password. |

#### Example

`POST /api/v1/kolide/change_password`

##### Request body

```
{
  "old_password": "VArCjNW7CfsxGp67",
  "new_password": "zGq7mCLA6z4PzArC",
}
```

##### Default response

`Status: 200`

##### Validation failed

`Status: 422 Unprocessable entity`

```
{
  "message": "Validation Failed",
  "errors": [
    {
      "name": "old_password",
      "reason": "old password does not match"
    }
  ]
}
```

---

### Me

Retrieves the user data for the authenticated user.

`POST /api/v1/kolide/me`

#### Example

`POST /api/v1/kolide/me`

##### Default response

`Status: 200`

```
{
  "user": {
    "created_at": "2020-11-13T22:57:12Z",
    "updated_at": "2020-11-16T23:49:41Z",
    "id": 1,
    "username": "jane",
    "name": "",
    "email": "janedoe@example.com",
    "admin": true,
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false
  }
}
```

---

### Perform required password reset

Resets the password of the authenticated user. Requires that `force_password_reset` is set to `true` prior to the request.

`POST /api/v1/kolide/perform_require_password_reset`

#### Example

`POST /api/v1/kolide/perform_required_password_reset`

##### Request body

```
{
  "new_password": "sdPz8CV5YhzH47nK"
}
```

##### Default response

`Status: 200`

```
{
  "user": {
    "created_at": "2020-11-13T22:57:12Z",
    "updated_at": "2020-11-17T00:09:23Z",
    "id": 1,
    "username": "jane",
    "name": "",
    "email": "janedoe@example.com",
    "admin": true,
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false
  }
}
```

---

### SSO config

Gets the current SSO configuration.

`GET /api/v1/kolide/sso`

#### Example

`GET /api/v1/kolide/sso`

##### Default response

`Status: 200`

```
{
  "settings": {
    "idp_name": "IDP Vendor 1",
    "idp_image_url": "",
    "sso_enabled": false
  }
}
```

---

### Initiate SSO

`POST /api/v1/kolide/sso`

#### Parameters

| Name      | Type   | In   | Description                                                                |
| --------- | ------ | ---- | -------------------------------------------------------------------------- |
| relay_url | string | body | **Required**. The relative url to be navigated to after succesful sign in. |

#### Example

`POST /api/v1/kolide/sso`

##### Request body

```
{
  "relay_url": "/hosts/manage"
}
```

##### Default response

`Status: 200`

##### Unknown error

`Status: 500`

```
{
  "message": "Unknown Error",
  "errors": [
    {
      "name": "base",
      "reason": "InitiateSSO getting metadata: Get \"https://idp.example.org/idp-meta.xml\": dial tcp: lookup idp.example.org on [2001:558:feed::1]:53: no such host"
    }
  ]
}
```

---

## Hosts

- [List hosts](#list-hosts)
- [Get hosts summary](#get-hosts-summary)
- [Get host](#get-host)
- [Get host by identifier](#get-host-by-identifier)
- [Delete host](#delete-host)

### List hosts

`GET /api/v1/kolide/hosts`

#### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                                                                                                                                    |
| ----------------------- | ------- | ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                                                                                                                                           |
| per_page                | integer | query | Results per page.                                                                                                                                                                                                                                                                              |
| order_key               | string  | query | What to order results by. Can be any column in the hosts table.                                                                                                                                                                                                                                |
| status                  | string  | query | Indicates the status of the hosts to return. Can either be `new`, `online`, `offline`, or `mia`.                                                                                                                                                                                               |
| additional_info_filters | string  | query | A comma-delimited list of fields to include in each host's additional information object. See [Fleet Configuration Options](https://github.com/fleetdm/fleet/blob/master/docs/1-Using-Fleet/2-fleetctl-CLI.md#fleet-configuration-options) for an example configuration with hosts' additional information. |

#### Example

`GET /api/v1/kolide/hosts?page=0&per_page=100&order_key=host_name`

##### Request query parameters

```
{
  "page": 0,
  "per_page": 100,
  "order_key": "host_name",
}
```

##### Default response

`Status: 200`

```
{
  "hosts": [
    {
      "created_at": "2020-11-05T05:09:44Z",
      "updated_at": "2020-11-05T06:03:39Z",
      "id": 1,
      "detail_updated_at": "2020-11-05T05:09:45Z",
      "label_updated_at": "2020-11-05T05:14:51Z",
      "seen_time": "2020-11-05T06:03:39Z",
      "hostname": "2ceca32fe484",
      "uuid": "392547dc-0000-0000-a87a-d701ff75bc65",
      "platform": "centos",
      "osquery_version": "2.7.0",
      "os_version": "CentOS Linux 7",
      "build": "",
      "platform_like": "rhel fedora",
      "code_name": "",
      "uptime": 8305000000000,
      "memory": 2084032512,
      "cpu_type": "6",
      "cpu_subtype": "142",
      "cpu_brand": "Intel(R) Core(TM) i5-8279U CPU @ 2.40GHz",
      "cpu_physical_cores": 4,
      "cpu_logical_cores": 4,
      "hardware_vendor": "",
      "hardware_model": "",
      "hardware_version": "",
      "hardware_serial": "",
      "computer_name": "2ceca32fe484",
      "primary_ip": "",
      "primary_mac": "",
      "distributed_interval": 10,
      "config_tls_refresh": 10,
      "logger_tls_period": 8,
      "additional": {},
      "enroll_secret_name": "default",
      "status": "offline",
      "display_text": "2ceca32fe484"
    },
    {
      "created_at": "2020-11-05T05:09:44Z",
      "updated_at": "2020-11-05T06:03:39Z",
      "id": 2,
      "detail_updated_at": "2020-11-05T05:09:45Z",
      "label_updated_at": "2020-11-05T05:14:52Z",
      "seen_time": "2020-11-05T06:03:40Z",
      "hostname": "4cc885c20110",
      "uuid": "392547dc-0000-0000-a87a-d701ff75bc65",
      "platform": "centos",
      "osquery_version": "2.7.0",
      "os_version": "CentOS 6.8.0",
      "build": "",
      "platform_like": "rhel",
      "code_name": "",
      "uptime": 8305000000000,
      "memory": 2084032512,
      "cpu_type": "6",
      "cpu_subtype": "142",
      "cpu_brand": "Intel(R) Core(TM) i5-8279U CPU @ 2.40GHz",
      "cpu_physical_cores": 4,
      "cpu_logical_cores": 4,
      "hardware_vendor": "",
      "hardware_model": "",
      "hardware_version": "",
      "hardware_serial": "",
      "computer_name": "4cc885c20110",
      "primary_ip": "",
      "primary_mac": "",
      "distributed_interval": 10,
      "config_tls_refresh": 10,
      "logger_tls_period": 8,
      "additional": {},
      "enroll_secret_name": "default",
      "status": "offline",
      "display_text": "4cc885c20110"
    },
  ]
}
```

### Get hosts summary

Returns the count of all hosts organized by status. `online_count` includes all hosts currently enrolled in Fleet. `offline_count` includes all hosts that haven't checked into Fleet recently. `mia_count` includes all hosts that haven't been seen by Fleet in more than 30 days. `new_count` includes the hosts that have been enrolled to Fleet in the last 24 hours.

`GET /api/v1/kolide/host_summary`

#### Parameters

None.

#### Example

`GET /api/v1/kolide/host_summary`


##### Default response

`Status: 200`

```
{
  "online_count": 2267,
  "offline_count": 141,
  "mia_count": 0,
  "new_count": 0
}
```

### Get host

Returns the information of the specified host.

`GET /api/v1/kolide/hosts/{id}`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| id                 | integer | path | **Required**. The host's id.                    |

#### Example

`GET /api/v1/kolide/hosts/121`


##### Default response

`Status: 200`

```
{
    "host": {
        "created_at": "2021-01-19T18:04:12Z",
        "updated_at": "2021-01-19T20:21:27Z",
        "id": 121,
        "detail_updated_at": "2021-01-19T20:04:22Z",
        "label_updated_at": "2021-01-19T20:04:22Z",
        "last_enrolled_at": "2021-01-19T18:04:12Z",
        "seen_time": "2021-01-19T20:21:27Z",
        "hostname": "259404d30eb6",
        "uuid": "f01c4390-0000-0000-a1e5-14346a5724dc",
        "platform": "ubuntu",
        "osquery_version": "2.10.2",
        "os_version": "Ubuntu 14.4.0",
        "build": "",
        "platform_like": "debian",
        "code_name": "",
        "uptime": 11202000000000,
        "memory": 2085326848,
        "cpu_type": "6",
        "cpu_subtype": "142",
        "cpu_brand": "Intel(R) Core(TM) i5-8279U CPU @ 2.40GHz",
        "cpu_physical_cores": 4,
        "cpu_logical_cores": 4,
        "hardware_vendor": "",
        "hardware_model": "",
        "hardware_version": "",
        "hardware_serial": "",
        "computer_name": "259404d30eb6",
        "primary_ip": "172.19.0.4",
        "primary_mac": "02:42:ac:13:00:04",
        "distributed_interval": 10,
        "config_tls_refresh": 10,
        "logger_tls_period": 10,
        "additional": {},
        "enroll_secret_name": "bar",
        "status": "offline",
        "display_text": "259404d30eb6"
    }
}
```

### Get host by identifier

Returns the information of the host specified using the `uuid`, `osquery_host_id`, `hostname`, or
`node_key` as an identifier

`GET /api/v1/kolide/hosts/identifier/{identifier}`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| identifier                 | integer or string | path | **Required**. The host's `uuid`, `osquery_host_id`, `hostname`, or `node_key`|

#### Example

`GET /api/v1/kolide/hosts/identifier/f01c4390-0000-0000-a1e5-14346a5724dc`


##### Default response

`Status: 200`

```
{
    "host": {
        "created_at": "2021-01-19T18:04:12Z",
        "updated_at": "2021-01-19T20:21:27Z",
        "id": 121,
        "detail_updated_at": "2021-01-19T20:04:22Z",
        "label_updated_at": "2021-01-19T20:04:22Z",
        "last_enrolled_at": "2021-01-19T18:04:12Z",
        "seen_time": "2021-01-19T20:21:27Z",
        "hostname": "259404d30eb6",
        "uuid": "f01c4390-0000-0000-a1e5-14346a5724dc",
        "platform": "ubuntu",
        "osquery_version": "2.10.2",
        "os_version": "Ubuntu 14.4.0",
        "build": "",
        "platform_like": "debian",
        "code_name": "",
        "uptime": 11202000000000,
        "memory": 2085326848,
        "cpu_type": "6",
        "cpu_subtype": "142",
        "cpu_brand": "Intel(R) Core(TM) i5-8279U CPU @ 2.40GHz",
        "cpu_physical_cores": 4,
        "cpu_logical_cores": 4,
        "hardware_vendor": "",
        "hardware_model": "",
        "hardware_version": "",
        "hardware_serial": "",
        "computer_name": "259404d30eb6",
        "primary_ip": "172.19.0.4",
        "primary_mac": "02:42:ac:13:00:04",
        "distributed_interval": 10,
        "config_tls_refresh": 10,
        "logger_tls_period": 10,
        "additional": {},
        "enroll_secret_name": "bar",
        "status": "offline",
        "display_text": "259404d30eb6"
    }
}
```

### Delete host

Deletes the specified host from Fleet. Note that a deleted host will fail authentication with the previous node key, and in most osquery configurations will attempt to re-enroll automatically. If the host still has a valid enroll secret, it will re-enroll successfully.

`DELETE /api/v1/kolide/hosts/{id}`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| id                 | integer | path | **Required**. The host's id.                    |

#### Example

`DELETE /api/v1/kolide/hosts/121`


##### Default response

`Status: 200`

```
{}
```

---

## Users

- [List all users](#list-all-users)
- [Create a user account with an invitation](#create-a-user-account-with-an-invitation)
- [Create a user account without an invitation](#create-a-user-account-without-an-invitation)
- [Get user information](#get-user-information)

The Fleet server exposes a handful of API endpoints that handles common user management operations. All the following endpoints require prior authentication meaning you must first log in successfully before calling any of the endpoints documented below.

### List all users

Returns a list of all enabled users

`GET /api/v1/kolide/users`

#### Parameters

None.

#### Example

`GET /api/v1/kolide/users`

##### Request query parameters

None.

##### Default response

`Status: 200`

```
{
  "users": [
    {
      "created_at": "2020-12-10T03:52:53Z",
      "updated_at": "2020-12-10T03:52:53Z",
      "id": 1,
      "username": "janedoe",
      "name": "",
      "email": "janedoe@example.com",
      "admin": true,
      "enabled": true,
      "force_password_reset": false,
      "gravatar_url": "",
      "sso_enabled": false
    }
  ]
}
```

##### Failed authentication

`Status: 401 Authentication Failed`

```
{
  "message": "Authentication Failed",
  "errors": [
    {
      "name": "base",
      "reason": "username or email and password do not match"
    }
  ]
}
```

### Create a user account with an invitation

Creates a user account after an invited user provides registration information and submits the form.

`POST /api/v1/kolide/users`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| email                 | string | body | **Required**. The email address of the user.                    |
| invite_token          | string | body | **Required**. Token provided to the user in the invitation email. |
| name                  | string | body | The name of the user.                                           |
| username              | string | body | **Required**. The username chosen by the user                   |
| password              | string | body | **Required**. The password chosen by the user.                  |
| password_confirmation | string | body | **Required**. Confirmation of the password chosen by the user.  |

#### Example

`POST /api/v1/kolide/users`

##### Request query parameters

```
{
  "email": "janedoe@example.com",
  "invite_token": "SjdReDNuZW5jd3dCbTJtQTQ5WjJTc2txWWlEcGpiM3c=",
  "name": "janedoe",
  "username": "janedoe",
  "password": "test-123",
  "password_confirmation": "test-123"
}
```

##### Default response

`Status: 200`

```
{
  "user": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 2,
    "username": "janedoe",
    "name": "janedoe",
    "email": "janedoe@example.com",
    "admin": false,
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false
  }
}
```

##### Failed authentication

`Status: 401 Authentication Failed`

```
{
  "message": "Authentication Failed",
  "errors": [
    {
      "name": "base",
      "reason": "username or email and password do not match"
    }
  ]
}
```

##### Expired or used invite code

`Status: 404 Resource Not Found`

```
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

##### Validation failed

`Status: 422 Validation Failed`

The same error will be returned whenever one of the required parameters fails the validation.

```
{
  "message": "Validation Failed",
  "errors": [
    {
      "name": "username",
      "reason": "cannot be empty"
    }
  ]
}
```

### Create a user account without an invitation

Creates a user account without requiring an invitation, the user is enabled immediately.

`POST /api/v1/kolide/users/admin`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| username   | string  | body | **Required**. The user's username.               |
| email      | string  | body | **Required**. The user's email address.          |
| password   | string  | body | **Required**. The user's password.               |
| invited_by | integer | body | **Required**. ID of the admin creating the user. |
| admin      | boolean | body | **Required**. Whether the user has admin privileges. |

#### Example

`POST /api/v1/kolide/users/admin`

##### Request query parameters

```
{
  "username": "janedoe",
  "email": "janedoe@example.com",
  "password": "test-123",
  "invited_by":1,
  "admin":true
}
```

##### Default response

`Status: 200`

```
{
  "user": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 5,
    "username": "janedoe",
    "name": "",
    "email": "janedoe@example.com",
    "admin": false,
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false
  }
}
```

##### Failed authentication

`Status: 401 Authentication Failed`

```
{
  "message": "Authentication Failed",
  "errors": [
    {
      "name": "base",
      "reason": "username or email and password do not match"
    }
  ]
}
```

##### User doesn't exist

`Status: 404 Resource Not Found`

```
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

### Get user information

Returns all information about a specific user.

`GET /api/v1/kolide/users/{id}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required**. The user's id. |

#### Example

`GET /api/v1/kolide/users/2`

##### Request query parameters

```
{
  "id": 1
}
```

##### Default response

`Status: 200`

```
{
  "user": {
    "created_at": "2020-12-10T05:20:25Z",
    "updated_at": "2020-12-10T05:24:27Z",
    "id": 2,
    "username": "janedoe",
    "name": "janedoe",
    "email": "janedoe@example.com",
    "admin": true,
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false
  }
}
```

##### Failed authentication

`Status: 401 Authentication Failed`

```
{
  "message": "Authentication Failed",
  "errors": [
    {
      "name": "base",
      "reason": "username or email and password do not match"
    }
  ]
}
```

##### User doesn't exist

`Status: 404 Resource Not Found`

```
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

---

## Queries

- [Get query](#get-query)
- [List queries](#list-queries)
- [Create query](#create-query)
- [Modify query](#modify-query)
- [Delete query](#delete-query)
- [Delete query by ID](#delete-query-by-id)
- [Get queries specs](#get-queries-specs)
- [Get query spec](#get-query-spec)
- [Apply queries specs](#apply-queries-specs)
- [Run live query](#run-live-query)
- [Run live query by query name](#run-live-query-by-query-name)

### Get query

Returns the query specified by ID.

`GET /api/v1/kolide/queries/{id}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| id   | integer  | path | **Required**. The id of the desired query.               |

#### Example

`GET /api/v1/kolide/queries/31`


##### Default response

`Status: 200`

```
{
  "query": {
    "created_at": "2021-01-19T17:08:24Z",
    "updated_at": "2021-01-19T17:08:24Z",
    "id": 31,
    "name": "centos_hosts",
    "description": "",
    "query": "select 1 from os_version where platform = \"centos\";",
    "saved": true,
    "author_id": 1,
    "author_name": "John",
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

Returns a list of all queries in the Fleet instance.

`GET /api/v1/kolide/queries`

#### Parameters

None.

#### Example

`GET /api/v1/kolide/queries`


##### Default response

`Status: 200`

```
{
"queries": [
  {
    "created_at": "2021-01-04T21:19:57Z",
    "updated_at": "2021-01-04T21:19:57Z",
    "id": 1,
    "name": "query1",
    "description": "query",
    "query": "SELECT * FROM osquery_info",
    "saved": true,
    "author_id": 1,
    "author_name": "noah",
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
    ]
  },
  {
    "created_at": "2021-01-19T17:08:24Z",
    "updated_at": "2021-01-19T17:08:24Z",
    "id": 2,
    "name": "osquery_version",
    "description": "The version of the Launcher and Osquery process",
    "query": "select launcher.version, osquery.version from kolide_launcher_info launcher, osquery_info osquery;",
    "saved": true,
    "author_id": 1,
    "author_name": "noah",
    "packs": [
      {
        "created_at": "2021-01-19T17:08:31Z",
        "updated_at": "2021-01-19T17:08:31Z",
        "id": 14,
        "name": "test_pack",
        "description": "",
        "platform": "",
        "disabled": false
      },
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
  },
  {
    "created_at": "2021-01-19T17:08:24Z",
    "updated_at": "2021-01-19T17:08:24Z",
    "id": 3,
    "name": "osquery_schedule",
    "description": "Report performance stats for each file in the query schedule.",
    "query": "select name, interval, executions, output_size, wall_time, (user_time/executions) as avg_user_time, (system_time/executions) as avg_system_time, average_memory, last_executed from osquery_schedule;",
    "saved": true,
    "author_id": 1,
    "author_name": "noah",
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
  },
]
```

### Create query

`POST /api/v1/kolide/queries`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| name   | string  | body | **Required**. The name of the query.               |
| query   | string  | body | **Required**. The query in SQL syntax.              |
| description   | string  | body | The query's description.               |

#### Example

`POST /api/v1/kolide/queries`

##### Request body

```
{
  "description": "This is a new query."
  "name": "new_query"
  "query": "SELECT * FROM osquery_info"
}
```

##### Default response

`Status: 200`

```
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
    "packs": []
  }
}
```

### Modify query

Returns the query specified by ID.

`PATCH /api/v1/kolide/queries/{id}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| id   | integer  | path | **Required.** The ID of the query.               |
| name   | string  | body | The name of the query.               |
| query   | string  | body | The query in SQL syntax.              |
| description   | string  | body | The query's description.               |

#### Example

`PATCH /api/v1/kolide/queries/2`

##### Request body

```
{
  "name": "new_title_for_my_query"
}
```

##### Default response

`Status: 200`

```
{
  "query": {
    "created_at": "2021-01-22T17:23:27Z",
    "updated_at": "2021-01-22T17:23:27Z",
    "id": 288,
    "name": "new_title_for_my_query",
    "description": "This is a new query.",
    "query": "SELECT * FROM osquery_info",
    "saved": true,
    "author_id": 1,
    "author_name": "noah",
    "packs": []
  }
}
```

### Delete query

Deletes the query specified by name.

`DELETE /api/v1/kolide/queries/{name}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| name   | string  | path | **Required.** The name of the query.               |

#### Example

`DELETE /api/v1/kolide/queries/{name}`

##### Default response

`Status: 200`

```
{}
```

### Delete query by ID

Deletes the query specified by ID.

`DELETE /api/v1/kolide/queries/id/{id}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| id   | integer  | path | **Required.** The ID of the query.               |

#### Example

`DELETE /api/v1/kolide/queries/id/28`

##### Default response

`Status: 200`

```
{}
```

### Get queries specs

Returns a list of all queries in the Fleet instance. Each item returned includes the name, description, and SQL of the query.

`GET /api/v1/kolide/spec/queries`

#### Parameters

None.

#### Example

`GET /api/v1/kolide/spec/queries`

##### Default response

`Status: 200`

```
{
  "specs": [
    {
        "name": "query1",
        "description": "query",
        "query": "SELECT * FROM osquery_info"
    },
    {
        "name": "osquery_version",
        "description": "The version of the Launcher and Osquery process",
        "query": "select launcher.version, osquery.version from kolide_launcher_info launcher, osquery_info osquery;"
    },
    {
        "name": "osquery_schedule",
        "description": "Report performance stats for each file in the query schedule.",
        "query": "select name, interval, executions, output_size, wall_time, (user_time/executions) as avg_user_time, (system_time/executions) as avg_system_time, average_memory, last_executed from osquery_schedule;"
    },
  ]
}
```

### Get query spec

Returns the name, description, and SQL of the query specified by name.

`GET /api/v1/kolide/spec/queries/{name}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| name   | string  | path | **Required.** The name of the query.               |

#### Example

`GET /api/v1/kolide/spec/queries/query1`

##### Default response

`Status: 200`

```
{
    "specs": {
        "name": "query1",
        "description": "query",
        "query": "SELECT * FROM osquery_info"
    }
}
```

### Apply queries specs

Creates and/or modifies the queries included in the specs list. To modify an existing query, the name of the query included in `specs` must already be used by an existing query. If a query with the specified name doesn't exist in Fleet, a new query will be created.

`POST /api/v1/kolide/spec/queries`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| specs   | list  | body | **Required.** The list of the queries to be created or modified.               |

#### Example

`POST /api/v1/kolide/spec/queries`

##### Request body

```
{
  "specs": [
    {
        "name": "new_query",
        "description": "This will be a new query because a query with the name 'new_query' doesn't exist in Fleet.",
        "query": "SELECT * FROM osquery_info"
    },
    {
        "name": "osquery_version",
        "description": "Only this queries description will be modified because a query with the name 'osquery_version' exists in Fleet.",
        "query": "select launcher.version, osquery.version from kolide_launcher_info launcher, osquery_info osquery;"
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

```
{}
```

### Run live query

Runs the specified query as a live query on the specified hosts or group of hosts. Returns a new live query campaign. Individual hosts must be specified with the host's ID. Groups of hosts are specified by label ID.

`POST /api/v1/kolide/spec/queries/run`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| query   | string  | body | **Required.** The SQL of the query               |
| selected   | object  | body | **Required.** The desired targets for the query. This object must contain `hosts` and `labels` properties. See example below     |

#### Example with one host targeted by ID

`POST /api/v1/kolide/spec/queries/run`

##### Request body

```
{
  "query": "select instance_id from system_info;"
  "selected": { "hosts": [171], "labels": []}
}
```

##### Default response

`Status: 200`

```
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
    "id": 273,
    "query_id": 293,
    "status": 0,
    "user_id": 1
  }
}
```

#### Example with multiple hosts targeted by label ID

`POST /api/v1/kolide/spec/queries/run`

##### Request body

```
{
  "query": "select instance_id from system_info;"
  "selected": { "hosts": [171], "labels": []}
}
```

##### Default response

`Status: 200`

```
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
    "id": 273,
    "query_id": 293,
    "status": 0,
    "user_id": 1
  }
}
```

### Run live query by query name

Runs the specified query by name as a live query on the specified hosts or group of hosts. Returns a new live query campaign. Individual hosts must be specified with the host's ID. Groups of hosts are specified by label.

`POST /api/v1/kolide/spec/queries/run`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| name   | string  | body | **Required.** The name of the query.             |
| selected   | object  | body | **Required.** The desired targets for the query. This object must contain `hosts` and `labels` properties. See example below.     |

#### Example with one host targeted

`POST /api/v1/kolide/spec/queries/run`

##### Request body

```
{
  "name": "instance_id"
  "selected": { "hosts": [171], "labels": [] }
}
```

##### Default response

`Status: 200`

```
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
      "id": 275,
      "query_id": 295,
      "status": 0,
      "user_id": 1
  }
}
```

---

## Packs

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
- [Get packs specs](#get-packs-specs)
- [Apply packs specs](#apply-packs-specs)
- [Get pack spec by name](#get-pack-spec-by-name)

### Create pack

`POST /api/v1/kolide/packs`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| name   | string | body | **Required**. The pack's name. |
| description   | string | body | The pack's description. |
| host_ids   | list | body | A list containing the targeted host IDs. |
| label_ids   | list | body | A list containing the targeted label's IDs. |

#### Example

`POST /api/v1/kolide/packs`

##### Request query parameters

```
{
  "description": "Collects osquery data."
  "host_ids": []
  "label_ids": [6]
  "name": "query_pack_1"
}
```

##### Default response

`Status: 200`

```
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
    ]
  }
}
```

### Modify pack

`PATCH /api/v1/kolide/packs/{id}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required.** The pack's id. |
| name   | string | body | The pack's name. |
| description   | string | body | The pack's description. |
| host_ids   | list | body | A list containing the targeted host IDs. |
| label_ids   | list | body | A list containing the targeted label's IDs. |

#### Example

`PATCH /api/v1/kolide/packs/{id}`

##### Request query parameters

```
{
  "description": "MacOS hosts are targeted"
  "host_ids": []
  "label_ids": [7]
}
```

##### Default response

`Status: 200`

```
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
    ]
  }
}
```

### Get pack

`GET /api/v1/kolide/packs/{id}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required.** The pack's id. |

#### Example

`GET /api/v1/kolide/packs/17`

##### Default response

`Status: 200`

```
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
    ]
  }
}
```

### List packs

`GET /api/v1/kolide/packs`

#### Parameters

None.

#### Example

`GET /api/v1/kolide/packs`

##### Default response

`Status: 200`

```
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
      ]
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
      ]
    },
  ]
}
```

### Delete pack

`DELETE /api/v1/kolide/packs/{name}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| name   | string | path | **Required.** The pack's name. |

#### Example

`DELETE /api/v1/kolide/packs/pack_number_one`

##### Default response

`Status: 200`

```
{}
```

### Delete pack by ID

`DELETE /api/v1/kolide/packs/id/{id}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required.** The pack's ID. |

#### Example

`DELETE /api/v1/kolide/packs/id/1`

##### Default response

`Status: 200`

```
{}
```

### Get scheduled queries in a pack

`GET /api/v1/kolide/packs/{id}/scheduled`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required.** The pack's ID. |

#### Example

`GET /api/v1/kolide/packs/1/scheduled`

##### Default response

`Status: 200`

```
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
      "shard": null
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
      "shard": null
    },
    {
      "created_at": "0001-01-01T00:00:00Z",
      "updated_at": "0001-01-01T00:00:00Z",
      "id": 51,
      "pack_id": 15,
      "name": "osquery_info",
      "query_id": 22,
      "query_name": "osquery_info",
      "query": "select i.*, p.resident_size, p.user_time, p.system_time, time.minutes as counter from osquery_info i, processes p, time where p.pid = i.pid;",
      "interval": 6667,
      "snapshot": true,
      "removed": false,
      "shard": null
    },
  ]
}
```

### Add scheduled query to a pack

`POST /api/v1/kolide/schedule`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| pack_id   | integer | body | **Required.** The pack's ID. |
| query_id   | integer | body | **Required.** The query's ID. |
| interval   | integer | body | **Required.** The amount of time, in seconds, the query waits before running. |
| snapshot   | boolean | body | **Required.** Whether the queries logs show everything in its current state. |
| removed   | boolean | body | **Required.** Whether "removed" actions should be logged. |
| platform   | string | body | The computer platform where this query will run (other platforms ignored). Empty value runs on all platforms. |
| shard   | integer | body | Restrict this query to a percentage (1-100) of target hosts. |
| version   | string | body | The minimum required osqueryd version installed on a host. |

#### Example

`POST /api/v1/kolide/schedule`

#### Request body 

```
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

```
{
  "scheduled": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 56,
    "pack_id": 17,
    "name": "osquery_events",
    "query_id": 23,
    "query_name": "osquery_events",
    "query": "select name, publisher, type, subscriptions, events, active from osquery_events;",
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

`GET /api/v1/kolide/schedule/{id}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required.** The scheduled query's ID. |

#### Example

`GET /api/v1/kolide/schedule/56`

##### Default response

`Status: 200`

```
{
  "scheduled": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 56,
    "pack_id": 17,
    "name": "osquery_events",
    "query_id": 23,
    "query_name": "osquery_events",
    "query": "select name, publisher, type, subscriptions, events, active from osquery_events;",
    "interval": 120,
    "snapshot": false,
    "removed": true,
    "platform": "windows",
    "version": "4.5.0",
    "shard": 10
  }
}
```

### Modify scheduled query

`PATCH /api/v1/kolide/schedule/{id}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required.** The scheduled query's ID. |
| interval   | integer | body | The amount of time, in seconds, the query waits before running. |
| snapshot   | boolean | body | Whether the queries logs show everything in its current state. |
| removed   | boolean | body | Whether "removed" actions should be logged. |
| platform   | string | body | The computer platform where this query will run (other platforms ignored). Empty value runs on all platforms. |
| shard   | integer | body | Restrict this query to a percentage (1-100) of target hosts. |
| version   | string | body | The minimum required osqueryd version installed on a host. |

#### Example

`PATCH /api/v1/kolide/schedule/56`

#### Request body 

```
{
  "platform": "",
}
```

##### Default response

`Status: 200`

```
{
  "scheduled": {
    "created_at": "2021-01-28T19:40:04Z",
    "updated_at": "2021-01-28T19:40:04Z",
    "id": 56,
    "pack_id": 17,
    "name": "osquery_events",
    "query_id": 23,
    "query_name": "osquery_events",
    "query": "select name, publisher, type, subscriptions, events, active from osquery_events;",
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

`DELETE /api/v1/kolide/schedule/{id}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required.** The scheduled query's ID. |

#### Example

`DELETE /api/v1/kolide/schedule/56`

##### Default response

`Status: 200`

```
{}
```

### Get packs specs

Returns the specs for all packs in the Fleet instance.

`GET /api/v1/kolide/spec/packs`

#### Example

`GET /api/v1/kolide/spec/packs`

##### Default response

`Status: 200`

```
{
  "specs": [
    {
      "id": 1,
      "name": "pack_1",
      "description": "Description",
      "disabled": false,
      "targets": {
        "labels": [
          "All Hosts"
        ]
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
        "labels": null
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
    },
  ]
}
```

### Apply packs specs

Returns the specs for all packs in the Fleet instance.

`POST /api/v1/kolide/spec/packs`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| specs   | list | body | **Required.** A list that includes the specs for each pack to be added to the Fleet instance. |

#### Example

`POST /api/v1/kolide/spec/packs`

##### Request body

```
{
  "specs": [
    {
      "id": 1,
      "name": "pack_1",
      "description": "Description",
      "disabled": false,
      "targets": {
        "labels": [
          "All Hosts"
        ]
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
        "labels": null
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
    },
  ]
}
```

##### Default response

`Status: 200`

```
{}
```

### Get pack spec by name

Returns the spec for the specified pack by pack name.

`GET /api/v1/kolide/spec/packs/{name}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| name   | string | path | **Required.** The pack's name. |

#### Example

`GET /api/v1/kolide/spec/packs/pack_1`

##### Default response

`Status: 200`

```
{
  "specs": {
    "id": 15,
    "name": "pack_1",
    "description": "Description",
    "disabled": false,
    "targets": {
      "labels": [
        "All Hosts"
      ]
    },
    "queries": [
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
  }
}
```

---

## Fleet configuration

- [Get certificate](#get-certificate)
- [Get configuration](#get-configuration)
- [Modify configuration](#modify-configuration)
- [Get enroll secrets](#get-enroll-secrets)
- [Modify enroll secrets](#modify-enroll-secrets)
- [Create invite](#create-invite)
- [List invites](#list-invites)
- [Delete invite](#delete-invite)
- [Verify invite](#verify-invite)

The Fleet server exposes a handful of API endpoints that handle the configuration of Fleet as well as endpoints that manage invitation and enroll secret operations. All the following endpoints require prior authentication meaning you must first log in successfully before calling any of the endpoints documented below.

### Get certificate

Returns the Fleet certificate.

`GET /api/v1/kolide/config/certificate`

#### Parameters

None.

#### Example

`GET /api/v1/kolide/config/certificate`


##### Default response

`Status: 200`

```
{
  "certificate_chain": <certificate_chain>
}
```

### Get configuration

Returns all information about the Fleet's configuration.

`GET /api/v1/kolide/config`

#### Parameters

None.

#### Example

`GET /api/v1/kolide/config`


##### Default response

`Status: 200`

```
{
  "org_info": {
    "org_name": "fleet",
    "org_logo_url": ""
  },
  "server_settings": {
    "kolide_server_url": "https://localhost:8080",
    "live_query_disabled": false
  },
  "smtp_settings": {
    "enable_smtp": false,
    "configured": false,
    "sender_address": "",
    "server": "",
    "port": 587,
    "authentication_type": "authtype_username_password",
    "user_name": "",
    "password": "********",
    "enable_ssl_tls": true,
    "authentication_method": "authmethod_plain",
    "domain": "",
    "verify_ssl_certs": true,
    "enable_start_tls": true
  },
  "sso_settings": {
    "entity_id": "",
    "issuer_uri": "",
    "idp_image_url": "",
    "metadata": "",
    "metadata_url": "",
    "idp_name": "",
    "enable_sso": false
  },
  "host_expiry_settings": {
    "host_expiry_enabled": false,
    "host_expiry_window": 0
  },
  "host_settings": {
    "additional_queries": null
  }
}
```

### Modify configuration

Modifies the Fleet's configuration with the supplied information.

`PATCH /api/v1/kolide/config`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| org_name   | string  | body | *Organization information*. The organization name.               |
| org_logo_url      | string  | body | *Organization information*. The URL for the organization logo.          |
| kolide_server_url   | string  | body | *Server settings*. The Fleet server URL.               |
| live_query_disabled | boolean | body | *Server settings*. Whether the live query capabilities are disabled. |
| enable_smtp      | boolean | body | *SMTP settings*. Whether SMTP is enabled for the Fleet app. |
| sender_address      | string | body | *SMTP settings*. The sender email address for the Fleet app. An invitation email is an example of the emails that may use this sender address  |
| server      | string | body | *SMTP settings*. The SMTP server for the Fleet app. |
| port      | integer | body | *SMTP settings*. The SMTP port for the Fleet app. |
| authetication_type | string | body | *SMTP settings*. The authentication type used by the SMTP server. Options include `"authtype_username_and_password"` or `"none"`|
| username_name | string | body | *SMTP settings*. The username used to authenticate requests made to the SMTP server.|
| password | string | body | *SMTP settings*. The password used to authenticate requests made to the SMTP server.|
| enable_ssl_tls | boolean | body | *SMTP settings*. Whether or not SSL and TLS are enabled for the SMTP server.|
| authentication_method | string | body | *SMTP settings*. The authentication method used to make authenticate requests to SMTP server. Options include `"authmethod_plain"`, `"authmethod_cram_md5"`, and `"authmethod_login"`.|
| domain | string | body | *SMTP settings*. The domain for the SMTP server.|
| verify_ssl_certs | boolean | body | *SMTP settings*. Whether or not SSL certificates are verified by the SMTP server. Turn this off (not recommended) if you use a self-signed certificate. |
| enabled_start_tls | boolean | body | *SMTP settings*. Detects if STARTTLS is enabled in your SMTP server and starts to use it.|
| enabled_sso      | boolean | body | *SSO settings*. Whether or not SSO is enabled for the Fleet application. If this value is true, you must also include most of the SSO settings parameters below.|
| entity_id      | string | body | *SSO settings*. The required entity ID is a URI that you use to identify Fleet when configuring the identity provider. |
| issuer_uri      | string | body | *SSO settings*. The URI you provide here must exactly match the Entity ID field used in the identity provider configuration. |
| idp_image_url      | string | body | *SSO settings*. An optional link to an image such as a logo for the identity provider. |
| metadata      | string | body | *SSO settings*. Metadata provided by the identity provider. Either metadata or a metadata URL must be provided. |
| metadata_url      | string | body | *SSO settings*. A URL that references the identity provider metadata. If available from the identity provider, this is the preferred means of providing metadata. |
| host_expiry_enabled      | boolean | body | *Host expiry settings*. When enabled, allows automatic cleanup of hosts that have not communicated with Fleet in some number of days. |
| host_expiry_window      | integer | body | *Host expiry settings*. If a host has not communicated with Fleet in the specified number of days, it will be removed. |
| additional_queries      | boolean | body | Whether or not additional queries are enabled on hosts. |

#### Example

`PATCH /api/v1/kolide/config`

##### Request body

```
{
  "org_info": {
    "org_name": "Fleet Device Management",
    "org_logo_url": "https://fleetdm.com/logo.png"
  },
  "smtp_settings: {
    "enable_smtp": true,
    "server": "localhost",
    "port": "1025"
  }
}
```


##### Default response

`Status: 200`

```
{
  "org_info": {
    "org_name": "Fleet Device Management",
    "org_logo_url": "https://fleetdm.com/logo.png"
  },
  "server_settings": {
    "kolide_server_url": "https://localhost:8080",
    "live_query_disabled": false
  },
  "smtp_settings": {
    "enable_smtp": true,
    "configured": true,
    "sender_address": "",
    "server": "localhost",
    "port": 1025,
    "authentication_type": "authtype_username_none",
    "user_name": "",
    "password": "********",
    "enable_ssl_tls": true,
    "authentication_method": "authmethod_plain",
    "domain": "",
    "verify_ssl_certs": true,
    "enable_start_tls": true
  },
  "sso_settings": {
    "entity_id": "",
    "issuer_uri": "",
    "idp_image_url": "",
    "metadata": "",
    "metadata_url": "",
    "idp_name": "",
    "enable_sso": false
  },
  "host_expiry_settings": {
    "host_expiry_enabled": false,
    "host_expiry_window": 0
  },
  "host_settings": {
    "additional_queries": null
  }
}
```

### Get enroll secret(s)

Returns all the enroll secrets used by the Fleet server.

`GET /api/v1/kolide/spec/enroll_secret`

#### Parameters

None.

#### Example

`GET /api/v1/kolide/spec/enroll_secret`


##### Default response

`Status: 200`

```
{
  "specs": {
    "secrets": [
      {
        "name": "bar",
        "secret": "fTp52/twaxBU6gIi0J6PHp8o5Sm1k1kn",
        "active": true,
        "created_at": "2021-01-07T19:40:04Z"
      },
      {
        "name": "default",
        "secret": "fTp52/twaxBU6gIi0J6PHp8o5Sm1k1kn",
        "active": true,
        "created_at": "2021-01-04T21:18:07Z"
      },
      {
        "name": "foo",
        "secret": "fTp52/twaxBU6gIi0J6PHp8o5Sm1k1kn",
        "active": true,
        "created_at": "2021-01-07T19:40:04Z"
      }
    ]
  }
}
```

### Modify enroll secret(s)

Modifies and/or creates the specified enroll secret(s).

`POST /api/v1/kolide/spec/enroll_secret`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| name   | string  | body | **Required.** The name of the enroll secret              |
| secret   | string  | body | **Required.** The plain text string used as the enroll secret.               |
| active      | boolean  | body | Whether or not the enroll secret is active. Must be set to true for hosts to enroll using the enroll secret.          |

#### Example

##### Request body

```
{
  "spec": {
    "secrets": [
      {
        "name": "bar",
        "secret": "fTp52/twaxBU6gIi0J6PHp8o5Sm1k1kn",
        "active": false,
      },
    ]
  }
}
```

`POST /api/v1/kolide/spec/enroll_secret`


##### Default response

`Status: 200`

```
{}
```

### Create invite


`POST /api/v1/kolide/invites`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| admin   | boolean  | body | **Required.** Whether or not the invited user will be granted admin privileges.             |
| email   | string  | body | **Required.** The email of the invited user. This email will receive the invitation link.              |
| invited_by      | integer  | body | **Required.** The id of the user that is extending the invitation. See the [Get user information](#get-user-information) endpoint for how to retrieve a user's id.          |
| name     | string  | body | **Required.** The name of the invited user.         |
| sso_enabled     | boolean  | body | **Required.** Whether or not SSO will be enabled for the invited user.   |

#### Example

##### Request body

```
{
  "admin": false,
  "email": "john_appleseed@example.com",
  "invited_by": 1,
  "name": John,
  "sso_enabled": false
}
```

`POST /api/v1/kolide/invites`


##### Default response

`Status: 200`

```
{
  "invite": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 3,
    "invited_by": 1,
    "email": "john_appleseed@example.com",
    "admin": false,
    "name": "John",
    "sso_enabled": false
  }
}
```

### List invites

Returns a list of the active invitations in Fleet.

`GET /api/v1/kolide/invites`

#### Parameters

None.

#### Example

`GET /api/v1/kolide/invites`


##### Default response

`Status: 200`

```
{
  "invites": [
    {
      "created_at": "0001-01-01T00:00:00Z",
      "updated_at": "0001-01-01T00:00:00Z",
      "id": 3,
      "invited_by": 1,
      "email": "john_appleseed@example.com",
      "admin": false,
      "name": "John",
      "sso_enabled": false
    },
    {
      "created_at": "0001-01-01T00:00:00Z",
      "updated_at": "0001-01-01T00:00:00Z",
      "id": 4,
      "invited_by": 1,
      "email": "bob_marks@example.com",
      "admin": true,
      "name": "Bob",
      "sso_enabled": false
    },
  ]
}
```

### Delete invite

Delete the specified invite from Fleet.

`DELETE /api/v1/kolide/invites/{id}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| id   | integer  | path | **Required.** The user's id.            |

#### Example

`DELETE /api/v1/kolide/invites/{id}`


##### Default response

`Status: 200`

```
{}
```

### Verify invite

Verify the specified invite.

`GET /api/v1/kolide/invites/{token}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| token   | integer  | path | **Required.** The user's invite token.            |

#### Example

`GET /api/v1/kolide/invites/{token}`


##### Default response

`Status: 200`

```
{
    "invite": {
        "created_at": "2021-01-15T00:58:33Z",
        "updated_at": "2021-01-15T00:58:33Z",
        "id": 4,
        "invited_by": 1,
        "email": "steve@example.com",
        "admin": false,
        "name": "Steve",
        "sso_enabled": false
    }
}
```

##### Not found

`Status: 404`

```
{
    "message": "Resource Not Found",
    "errors": [
        {
            "name": "base",
            "reason": "Invite with token <token> was not found in the datastore"
        }
    ]
}
```
---

## Osquery options

- [Get osquery options spec](#get-osquery-options-spec)
- [Modify osquery options spec](#modify-osquery-options-spec)

### Get osquery options spec

Retrieve the osquery options configured via Fleet.

`GET /api/v1/kolide/spec/osquery_options`

#### Parameters

None.

#### Example

`GET /api/v1/kolide/spec/osquery_options`


##### Default response

`Status: 200`

```
{
  "spec": {
    "config": {
      "options": {
        "logger_plugin": "tls",
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
}
```

### Modify osquery options spec

Modifies the osquery options configuration set in Fleet.

`POST /api/v1/kolide/spec/osquery_options`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| spec   | JSON  | body | **Required.** The modified osquery spec.            |

#### Example

`POST /api/v1/kolide/spec/osquery_options`

##### Request body

```
{
  "spec": {
    "config": {
      "options": {
        "logger_plugin": "tls",
        "pack_delimiter": "/",
        "logger_tls_period": 10,
        "distributed_plugin": "tls",
        "disable_distributed": false,
        "logger_tls_endpoint": "/api/v1/osquery/log",
        "distributed_interval": 12,
        "distributed_tls_max_attempts": 4
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
}
```


##### Default response

`Status: 200`

```
{}
```

---
