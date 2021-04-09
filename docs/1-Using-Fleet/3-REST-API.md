# REST API
- [Overview](#overview)
  - [fleetctl](#fleetctl)
  - [Current API](#current-api)
- [Authentication](#authentication)
- [Hosts](#hosts)
- [Labels](#labels)
- [Users](#users)
- [Sessions](#sessions)
- [Queries](#queries)
- [Packs](#packs)
- [Targets](#targets)
- [Fleet configuration](#fleet-configuration)
- [Osquery options](#osquery-options)
- [File carving](#file-carving)

## Overview

Fleet is powered by a Go API server which serves three types of endpoints:

- Endpoints starting with `/api/v1/osquery/` are osquery TLS server API endpoints. All of these endpoints are used for talking to osqueryd agents and that's it.
- Endpoints starting with `/api/v1/fleet/` are endpoints to interact with the Fleet data model (packs, queries, scheduled queries, labels, hosts, etc) as well as application endpoints (configuring settings, logging in, session management, etc).
- All other endpoints are served by the React single page application bundle.
  The React app uses React Router to determine whether or not the URI is a valid
  route and what to do.
  
Note: We have deprecated `/api/v1/kolide/` routes and will remove them in the Fleet 4.0 release. Please migrate all routes to `/api/v1/fleet/`. 

### fleetctl

Many of the operations that a user may wish to perform with an API are currently best performed via the [fleetctl](./2-fleetctl-CLI.md) tooling. These CLI tools allow updating of the osquery configuration entities, as well as performing live queries.

### Current API

The general idea with the current API is that there are many entities throughout the Fleet application, such as:

- Queries
- Packs
- Labels
- Hosts

Each set of objects follows a similar REST access pattern.

- You can `GET /api/v1/fleet/packs` to get all packs
- You can `GET /api/v1/fleet/packs/1` to get a specific pack.
- You can `DELETE /api/v1/fleet/packs/1` to delete a specific pack.
- You can `POST /api/v1/fleet/packs` (with a valid body) to create a new pack.
- You can `PATCH /api/v1/fleet/packs/1` (with a valid body) to modify a specific pack.

Queries, packs, scheduled queries, labels, invites, users, sessions all behave this way. Some objects, like invites, have additional HTTP methods for additional functionality. Some objects, such as scheduled queries, are merely a relationship between two other objects (in this case, a query and a pack) with some details attached.

All of these objects are put together and distributed to the appropriate osquery agents at the appropriate time. At this time, the best source of truth for the API is the [HTTP handler file](https://github.com/fleetdm/fleet/blob/master/server/service/handler.go) in the Go application. The REST API is exposed via a transport layer on top of an RPC service which is implemented using a micro-service library called [Go Kit](https://github.com/go-kit/kit). If using the Fleet API is important to you right now, being familiar with Go Kit would definitely be helpful.



## Authentication

- [Log in](#log-in)
- [Log out](#log-out)
- [Forgot password](#forgot-password)
- [Change password](#change-password)
- [Reset password](#reset-password)
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

`POST /api/v1/fleet/login`

#### Parameters

| Name     | Type   | In   | Description                                   |
| -------- | ------ | ---- | --------------------------------------------- |
| username | string | body | **Required**. The user's email.               |
| password | string | body | **Required**. The user's plain text password. |

#### Example

`POST /api/v1/fleet/login`

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

`POST /api/v1/fleet/logout`

#### Example

`POST /api/v1/fleet/logout`

##### Default response

`Status: 200`

---

### Forgot password

Sends a password reset email to the specified email. Requires that SMTP is configured for your Fleet server.

`POST /api/v1/fleet/forgot_password`

#### Parameters

| Name  | Type   | In   | Description                                                             |
| ----- | ------ | ---- | ----------------------------------------------------------------------- |
| email | string | body | **Required**. The email of the user requesting the reset password link. |

#### Example

`POST /api/v1/fleet/forgot_password`

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

`POST /api/v1/fleet/change_password`

Changes the password for the authenticated user.

#### Parameters

| Name         | Type   | In   | Description                            |
| ------------ | ------ | ---- | -------------------------------------- |
| old_password | string | body | **Required**. The user's old password. |
| new_password | string | body | **Required**. The user's new password. |

#### Example

`POST /api/v1/fleet/change_password`

##### Request body

```
{
  "old_password": "VArCjNW7CfsxGp67",
  "new_password": "zGq7mCLA6z4PzArC"
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

### Reset password

Resets a user's password. Which user is determined by the password reset token used. The password reset token can be found in the password reset email sent to the desired user.

`POST /api/v1/fleet/reset_password`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| new_password   | string | body | **Required**. The new password. |
| new_password_confirmation   | string | body | **Required**. Confirmation for the new password. |
| password_reset_token   | string | body | **Required**. The token provided to the user in the password reset email. |

#### Example

`POST /api/v1/fleet/reset_password`

##### Request body

```
{
  "new_password": "abc123"
  "new_password_confirmation": "abc123"
  "password_reset_token": "UU5EK0JhcVpsRkY3NTdsaVliMEZDbHJ6TWdhK3oxQ1Q="
}
```

##### Default response

`Status: 200`

```
{}
```

---

### Me

Retrieves the user data for the authenticated user.

`POST /api/v1/fleet/me`

#### Example

`POST /api/v1/fleet/me`

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

`POST /api/v1/fleet/perform_require_password_reset`

#### Example

`POST /api/v1/fleet/perform_required_password_reset`

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

`GET /api/v1/fleet/sso`

#### Example

`GET /api/v1/fleet/sso`

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

`POST /api/v1/fleet/sso`

#### Parameters

| Name      | Type   | In   | Description                                                                |
| --------- | ------ | ---- | -------------------------------------------------------------------------- |
| relay_url | string | body | **Required**. The relative url to be navigated to after successful sign in. |

#### Example

`POST /api/v1/fleet/sso`

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

`GET /api/v1/fleet/hosts`

#### Parameters

| Name                    | Type    | In    | Description                                                                                                                                                                                                                                                                                    |
| ----------------------- | ------- | ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| page                    | integer | query | Page number of the results to fetch.                                                                                                                                                                                                                                                           |
| per_page                | integer | query | Results per page.                                                                                                                                                                                                                                                                              |
| order_key               | string  | query | What to order results by. Can be any column in the hosts table.                                                                                                                                                                                                                                |
| order_direction               | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`.                                                                                                                                                                                                 |
| status                  | string  | query | Indicates the status of the hosts to return. Can either be `new`, `online`, `offline`, or `mia`.                                                                                                                                                                                               |
| query                  | string  | query | Search query keywords. Searchable fields include `hostname`, `machine_serial`, `uuid`, and `ipv4`.                                                                                                                                                                                               |
| additional_info_filters | string  | query | A comma-delimited list of fields to include in each host's additional information object. See [Fleet Configuration Options](https://github.com/fleetdm/fleet/blob/master/docs/1-Using-Fleet/2-fleetctl-CLI.md#fleet-configuration-options) for an example configuration with hosts' additional information. |

#### Example

`GET /api/v1/fleet/hosts?page=0&per_page=100&order_key=host_name&query=2ce`

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
  ]
}
```

### Get hosts summary

Returns the count of all hosts organized by status. `online_count` includes all hosts currently enrolled in Fleet. `offline_count` includes all hosts that haven't checked into Fleet recently. `mia_count` includes all hosts that haven't been seen by Fleet in more than 30 days. `new_count` includes the hosts that have been enrolled to Fleet in the last 24 hours.

`GET /api/v1/fleet/host_summary`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/host_summary`


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

`GET /api/v1/fleet/hosts/{id}`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| id                 | integer | path | **Required**. The host's id.                    |

#### Example

`GET /api/v1/fleet/hosts/121`


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

`GET /api/v1/fleet/hosts/identifier/{identifier}`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| identifier                 | integer or string | path | **Required**. The host's `uuid`, `osquery_host_id`, `hostname`, or `node_key`|

#### Example

`GET /api/v1/fleet/hosts/identifier/f01c4390-0000-0000-a1e5-14346a5724dc`


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

`DELETE /api/v1/fleet/hosts/{id}`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| id                 | integer | path | **Required**. The host's id.                    |

#### Example

`DELETE /api/v1/fleet/hosts/121`


##### Default response

`Status: 200`

```
{}
```

---

## Labels

- [Create label](#create-label)
- [Modify label](#modify-label)
- [Get label](#get-label)
- [List labels](#list-labels)
- [List hosts in a label](#list-hosts-in-a-label)
- [Delete label](#delete-label)
- [Delete label by ID](#delete-label-by-id)
- [Apply labels specs](#apply-labels-specs)
- [Get labels specs](#get-labels-specs)
- [Get label spec](#get-label-spec)

### Create label

Creates a dynamic label.

`POST /api/v1/fleet/labels`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| name                 | string | body | **Required**. The label's name.                    |
| description                 | string | body | The label's description.                    |
| query                 | string | body | **Required**. The query in SQL syntax used to filter the hosts.                    |
| platform                 | string | body | The specific platform for the label to target. Provides an additional filter. Choices for platform are `darwin`, `windows`, `ubuntu`, and `centos`. All platforms are included by default and this option is represented by an empty string.|

#### Example

`POST /api/v1/fleet/labels`

##### Request body

```
{
  "name": "Ubuntu hosts",
  "description": "Filters ubuntu hosts",
  "query": "select 1 from os_version where platform = 'ubuntu';",
  "platform": ""
}
```

##### Default response

`Status: 200`

```
{
  "label": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 1,
    "name": "Ubuntu hosts",
    "description": "Filters ubuntu hosts",
    "query": "select 1 from os_version where platform = 'ubuntu';",
    "label_type": "regular",
    "label_membership_type": "dynamic",
    "display_text": "Ubuntu hosts",
    "count": 0,
    "host_ids": null
  }
}
```

### Modify label

Modifies the specified label. Note: Label queries are immutable. To change the query, you must delete the label and create a new label.

`PATCH /api/v1/fleet/labels/{id}`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| id                 | integer | path | **Required**. The label's id.                    |
| name                 | string | body | The label's name.                    |
| description                 | string | body | The label's description.                    |
| platform                 | string | body | The specific platform for the label to target. Provides an additional filter. Choices for platform are `darwin`, `windows`, `ubuntu`, and `centos`. All platforms are included by default and this option is represented by an empty string.|

#### Example

`PATCH /api/v1/fleet/labels/1`

##### Request body

```
{
  "name": "macOS label",
  "description": "Now this label only includes macOS machines",
  "platform": "darwin"
}
```

##### Default response

`Status: 200`

```
{
  "label": {
    "created_at": "0001-01-01T00:00:00Z",
    "updated_at": "0001-01-01T00:00:00Z",
    "id": 1,
    "name": "Ubuntu hosts",
    "description": "Filters ubuntu hosts",
    "query": "select 1 from os_version where platform = 'ubuntu';",
    "platform": "darwin",
    "label_type": "regular",
    "label_membership_type": "dynamic",
    "display_text": "Ubuntu hosts",
    "count": 0,
    "host_ids": null
  }
}
```

### Get label

Returns the specified label.

`GET /api/v1/fleet/labels/{id}`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| id                 | integer | path | **Required**. The label's id.                    |

#### Example

`GET /api/v1/fleet/labels/1`

##### Default response

`Status: 200`

```
{
  "label": {
    "created_at": "2021-02-09T22:09:43Z",
    "updated_at": "2021-02-09T22:15:58Z",
    "id": 12,
    "name": "Ubuntu",
    "description": "Filters ubuntu hosts",
    "query": "select 1 from os_version where platform = 'ubuntu';",
    "label_type": "regular",
    "label_membership_type": "dynamic",
    "display_text": "Ubuntu",
    "count": 0,
    "host_ids": null
  }
}
```

### List labels

Returns a list of all the labels in Fleet.

`GET /api/v1/fleet/labels`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| id                 | integer | path | **Required**. The label's id.                    |
| order_key               | string  | query | What to order results by. Can be any column in the labels table.                                                                                                                                                                                                                                |
| order_direction               | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`.   |

#### Example

`GET /api/v1/fleet/labels`

##### Default response

`Status: 200`

```
{
  "labels": [
    {
      "created_at": "2021-02-02T23:55:25Z",
      "updated_at": "2021-02-02T23:55:25Z",
      "id": 6,
      "name": "All Hosts",
      "description": "All hosts which have enrolled in Fleet",
      "query": "select 1;",
      "label_type": "builtin",
      "label_membership_type": "dynamic",
      "host_count": 7,
      "display_text": "All Hosts",
      "count": 7,
      "host_ids": null
    },
    {
      "created_at": "2021-02-02T23:55:25Z",
      "updated_at": "2021-02-02T23:55:25Z",
      "id": 7,
      "name": "macOS",
      "description": "All macOS hosts",
      "query": "select 1 from os_version where platform = 'darwin';",
      "platform": "darwin",
      "label_type": "builtin",
      "label_membership_type": "dynamic",
      "host_count": 1,
      "display_text": "macOS",
      "count": 1,
      "host_ids": null
    },
    {
      "created_at": "2021-02-02T23:55:25Z",
      "updated_at": "2021-02-02T23:55:25Z",
      "id": 8,
      "name": "Ubuntu Linux",
      "description": "All Ubuntu hosts",
      "query": "select 1 from os_version where platform = 'ubuntu';",
      "platform": "ubuntu",
      "label_type": "builtin",
      "label_membership_type": "dynamic",
      "host_count": 3,
      "display_text": "Ubuntu Linux",
      "count": 3,
      "host_ids": null
    },
    {
      "created_at": "2021-02-02T23:55:25Z",
      "updated_at": "2021-02-02T23:55:25Z",
      "id": 9,
      "name": "CentOS Linux",
      "description": "All CentOS hosts",
      "query": "select 1 from os_version where platform = 'centos' or name like '%centos%'",
      "label_type": "builtin",
      "label_membership_type": "dynamic",
      "host_count": 3,
      "display_text": "CentOS Linux",
      "count": 3,
      "host_ids": null
    },
    {
      "created_at": "2021-02-02T23:55:25Z",
      "updated_at": "2021-02-02T23:55:25Z",
      "id": 10,
      "name": "MS Windows",
      "description": "All Windows hosts",
      "query": "select 1 from os_version where platform = 'windows';",
      "platform": "windows",
      "label_type": "builtin",
      "label_membership_type": "dynamic",
      "display_text": "MS Windows",
      "count": 0,
      "host_ids": null
    },
  ]
}
```

### List hosts in a label

Returns a list of the hosts that belong to the specified label.

`GET /api/v1/fleet/labels/{id}/hosts`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| id                 | integer | path | **Required**. The label's id.                    |
| order_key          | string  | query | What to order results by. Can be any column in the hosts table.                                                                                                                                                                                                                                |
| order_direction   | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`.   |
| query              | string  | query | Search query keywords. Searchable fields include `hostname`, `machine_serial`, `uuid`, and `ipv4`. |                                                                        

#### Example

`GET /api/v1/fleet/labels/6/hosts&query=floobar`

##### Default response

`Status: 200`

```
{
  "hosts": [
    {
      "created_at": "2021-02-03T16:11:43Z",
      "updated_at": "2021-02-03T21:58:19Z",
      "id": 2,
      "detail_updated_at": "2021-02-03T21:58:10Z",
      "label_updated_at": "2021-02-03T21:58:10Z",
      "last_enrolled_at": "2021-02-03T16:11:43Z",
      "seen_time": "2021-02-03T21:58:20Z",
      "hostname": "floobar42",
      "uuid": "a2064cef-0000-0000-afb9-283e3c1d487e",
      "platform": "ubuntu",
      "osquery_version": "4.5.1",
      "os_version": "Ubuntu 20.4.0",
      "build": "",
      "platform_like": "debian",
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
      "computer_name": "e2e7f8d8983d",
      "primary_ip": "172.20.0.2",
      "primary_mac": "02:42:ac:14:00:02",
      "distributed_interval": 10,
      "config_tls_refresh": 10,
      "logger_tls_period": 10,
      "additional": {},
      "enroll_secret_name": "default",
      "status": "offline",
      "display_text": "e2e7f8d8983d"
    },
  ]
}
```

### Delete label

Deletes the label specified by name.

`DELETE /api/v1/fleet/labels/{name}`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| name                 | string | path | **Required**. The label's name.                    |

#### Example

`DELETE /api/v1/fleet/labels/ubuntu_label`

##### Default response

`Status: 200`

```
{}
```

### Delete label by ID

Deletes the label specified by ID.

`DELETE /api/v1/fleet/labels/id/{id}`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| id                 | integer | path | **Required**. The label's id.                    |

#### Example

`DELETE /api/v1/fleet/labels/id/13`

##### Default response

`Status: 200`

```
{}
```

### Apply labels specs

Applies the supplied labels specs to Fleet. Each label requires the `name`, and `label_membership_type` properties.

If the `label_membership_type` is set to `dynamic`, the `query` property must also be specified with the value set to a query in SQL syntax.

If the `label_membership_type` is set to `manual`, the `hosts` property must also be specified with the value set to a list of hostnames.

`POST /api/v1/fleet/specs/labels`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| specs                 | list | path | A list of the label to apply. Each label requires the `name`, `query`, and `label_membership_type` properties|

#### Example

`POST /api/v1/fleet/specs/labels`

##### Request body

```
{
  "specs": [
    {
      "name": "Ubuntu",
      "description": "Filters ubuntu hosts",
      "query": "select 1 from os_version where platform = 'ubuntu';",
      "label_membership_type": "dynamic"
    },
    {
      "name": "local_machine",
      "description": "Includes only my local machine",
      "label_membership_type": "manual",
      "hosts": [
        "snacbook-pro.local"
      ]
    }
  ]
}
```

##### Default response

`Status: 200`

```
{}
```

### Get labels specs

`GET /api/v1/fleet/specs/labels`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/specs/labels`

##### Default response

`Status: 200`

```
{
  "specs": [
    {
      "ID": 0,
      "name": "All Hosts",
      "description": "All hosts which have enrolled in Fleet",
      "query": "select 1;",
      "label_type": "builtin",
      "label_membership_type": "dynamic"
    },
    {
      "ID": 0,
      "name": "macOS",
      "description": "All macOS hosts",
      "query": "select 1 from os_version where platform = 'darwin';",
      "platform": "darwin",
      "label_type": "builtin",
      "label_membership_type": "dynamic"
    },
    {
      "ID": 0,
      "name": "Ubuntu Linux",
      "description": "All Ubuntu hosts",
      "query": "select 1 from os_version where platform = 'ubuntu';",
      "platform": "ubuntu",
      "label_type": "builtin",
      "label_membership_type": "dynamic"
    },
    {
      "ID": 0,
      "name": "CentOS Linux",
      "description": "All CentOS hosts",
      "query": "select 1 from os_version where platform = 'centos' or name like '%centos%'",
      "label_type": "builtin",
      "label_membership_type": "dynamic"
    },
    {
      "ID": 0,
      "name": "MS Windows",
      "description": "All Windows hosts",
      "query": "select 1 from os_version where platform = 'windows';",
      "platform": "windows",
      "label_type": "builtin",
      "label_membership_type": "dynamic"
    },
    {
      "ID": 0,
      "name": "Ubuntu",
      "description": "Filters ubuntu hosts",
      "query": "select 1 from os_version where platform = 'ubuntu';",
      "label_membership_type": "dynamic"
    }
  ]
}
```

### Get label spec

Returns the spec for the label specified by name.

`GET /api/v1/fleet/specs/labels/{name}`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/specs/labels/local_machine`

##### Default response

`Status: 200`

```
{
  "specs": {
    "ID": 0,
    "name": "local_machine",
    "description": "Includes only my local machine",
    "query": "",
    "label_membership_type": "manual",
    "hosts": [
        "snacbook-pro.local"
    ]
  }
}
```

---

## Users

- [List all users](#list-all-users)
- [Create a user account with an invitation](#create-a-user-account-with-an-invitation)
- [Create a user account without an invitation](#create-a-user-account-without-an-invitation)
- [Get user information](#get-user-information)
- [Modify user](#modify-user)
- [Enable or disable user](#enable-or-disable-user)
- [Promote or demote user](#promote-or-demote-user)
- [Require password reset](#require-password-reset)
- [List a user's sessions](#list-a-users-sessions)
- [Delete a user's sessions](#delete-a-users-sessions)

The Fleet server exposes a handful of API endpoints that handles common user management operations. All the following endpoints require prior authentication meaning you must first log in successfully before calling any of the endpoints documented below.

### List all users

Returns a list of all enabled users

`GET /api/v1/fleet/users`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| order_key               | string  | query | What to order results by. Can be any column in the users table.                                                                                                                                                                                                                                |
| order_direction               | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`.   |

#### Example

`GET /api/v1/fleet/users`

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

`POST /api/v1/fleet/users`

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

`POST /api/v1/fleet/users`

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

`POST /api/v1/fleet/users/admin`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| username   | string  | body | **Required**. The user's username.               |
| email      | string  | body | **Required**. The user's email address.          |
| password   | string  | body | **Required**. The user's password.               |
| invited_by | integer | body | **Required**. ID of the admin creating the user. |
| admin      | boolean | body | **Required**. Whether the user has admin privileges. |

#### Example

`POST /api/v1/fleet/users/admin`

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

`GET /api/v1/fleet/users/{id}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required**. The user's id. |

#### Example

`GET /api/v1/fleet/users/2`

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

### Modify user

`PATCH /api/v1/fleet/users/{id}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required**. The user's id. |
| name   | string | body | The user's name. |
| username   | string | body | The user's username. |
| position   | string | body | The user's position. |
| email   | string | body | The user's email. |
| sso_enabled   | boolean | body | Whether or not SSO is enabled for the user. |

#### Example

`PATCH /api/v1/fleet/users/2`

##### Request body

```
{
  "name": "Jane Doe",
  "position": "Incident Response Engineer"
}
```

##### Default response

`Status: 200`

```
{
  "user": {
    "created_at": "2021-02-03T16:11:06Z",
    "updated_at": "2021-02-03T16:11:06Z",
    "id": 2,
    "username": "jdoe",
    "name": "Jane Doe",
    "email": "janedoe@example.com",
    "admin": true,
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "position": "Incident Response Engineer",
    "sso_enabled": false
  }
}
```

### Enable or disable user

Revokes or renews the selected user's access to Fleet. Returns the user object.

`POST /api/v1/fleet/users/{id}/enable`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required**. The user's id. |
| enabled   | boolean | body | **Required**. Whether or not the user can access Fleet. |

#### Example

`POST /api/v1/fleet/users/2/enable`

#### Default body

```
{
  "enabled": false
}
```

##### Default response

`Status: 200`

```
{
  "user": {
    "created_at": "2021-02-23T22:23:34Z",
    "updated_at": "2021-02-23T22:23:34Z",
    "id": 2,
    "username": "janedoe",
    "name": "Jane Doe",
    "email": "janedoe@example.com",
    "admin": false,
    "enabled": false,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false
  }
}
```

### Promote or demote user

Promotes or demotes the selected user's level of access as an admin in Fleet. Admins in Fleet have the ability to invite new users, edit settings, and edit osquery options across hosts. Returns the user object.

`POST /api/v1/fleet/users/{id}/admin`

#### Parameters

| Name  | Type    | In    | Description                  |
| ----- | ------- | ----- | ---------------------------- |
| id    | integer | path | **Required**. The user's id. |
| admin | boolean | body | **Required**. Whether or not the user is an admin. |

#### Example

`POST /api/v1/fleet/users/2/admin`

#### Default body

```
{
  "admin": true
}
```

##### Default response

`Status: 200`

```
{
  "user": {
    "created_at": "2021-02-23T22:23:34Z",
    "updated_at": "2021-02-23T22:28:41Z",
    "id": 2,
    "username": "janedoe",
    "name": "Jane Doe",
    "email": "janedoe@example.com",
    "admin": true,
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false
  }
}
```

### Require password reset

The selected user is logged out of Fleet and required to reset their password during the next attempt to log in. This also revokes all active Fleet API tokens for this user. Returns the user object.

`POST /api/v1/fleet/users/{id}/require_password_reset`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required**. The user's id. |
| reset   | boolean | body | Whether or not the user is required to reset their password during the next attempt to log in. |

#### Example

`POST /api/v1/fleet/users/{id}/require_password_reset`

##### Request body

```
{
  "require": true
}
```

##### Default response

`Status: 200`

```
{
  "user": {
    "created_at": "2021-02-23T22:23:34Z",
    "updated_at": "2021-02-23T22:28:52Z",
    "id": 2,
    "username": "janedoe",
    "name": "Jane Doe",
    "email": "janedoe@example.com",
    "admin": false,
    "enabled": true,
    "force_password_reset": true,
    "gravatar_url": "",
    "sso_enabled": false
  }
}
```

### List a user's sessions

Returns a list of the user's sessions in Fleet.

`GET /api/v1/fleet/users/{id}/sessions`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/users/1/sessions`

##### Default response

`Status: 200`

```
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

### Delete a user's sessions

Deletes the selected user's sessions in Fleet. Also deletes the user's API token.

`DELETE /api/v1/fleet/users/{id}/sessions`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| id   | integer  | path | **Required**. The ID of the desired user.               |

#### Example

`DELETE /api/v1/fleet/users/1/sessions`

##### Default response

`Status: 200`

```
{}
```

---

## Sessions
- [Get session info](#get-session-info)
- [Delete session](#delete-session)

### Get session info

Returns the session information for the session specified by ID.

`GET /api/v1/fleet/sessions/{id}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| id   | integer  | path | **Required**. The ID of the desired session.               |

#### Example

`GET /api/v1/fleet/sessions/1`

##### Default response

`Status: 200`

```
{
  "session_id": 1,
  "user_id": 1,
  "created_at": "2021-03-02T18:41:34Z"
}
```

### Delete session

Deletes the session specified by ID. When the user associated with the session next attempts to access Fleet, they will be asked to log in.

`DELETE /api/v1/fleet/sessions/{id}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| id   | integer  | path | **Required**. The id of the desired session.               |

#### Example

`DELETE /api/v1/fleet/sessions/1`


##### Default response

`Status: 200`

```
{}
```

---

## Queries

- [Get query](#get-query)
- [List queries](#list-queries)
- [Create query](#create-query)
- [Modify query](#modify-query)
- [Delete query](#delete-query)
- [Delete query by ID](#delete-query-by-id)
- [Delete queries](#delete-queries)
- [Get queries specs](#get-queries-specs)
- [Get query spec](#get-query-spec)
- [Apply queries specs](#apply-queries-specs)
- [Run live query](#run-live-query)
- [Run live query by name](#run-live-query-by-name)
- [Retrieve live query results (standard WebSocket API)](#retrieve-live-query-results-standard-websocket-api)
- [Retrieve live query results (SockJS)](#retrieve-live-query-results-sockjs)

### Get query

Returns the query specified by ID.

`GET /api/v1/fleet/queries/{id}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| id   | integer  | path | **Required**. The id of the desired query.               |

#### Example

`GET /api/v1/fleet/queries/31`


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

`GET /api/v1/fleet/queries`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| order_key               | string  | query | What to order results by. Can be any column in the queries table.                                                                                                                                                                                                                                |
| order_direction               | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`.   |

#### Example

`GET /api/v1/fleet/queries`


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

`POST /api/v1/fleet/queries`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| name   | string  | body | **Required**. The name of the query.               |
| query   | string  | body | **Required**. The query in SQL syntax.              |
| description   | string  | body | The query's description.               |

#### Example

`POST /api/v1/fleet/queries`

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

`PATCH /api/v1/fleet/queries/{id}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| id   | integer  | path | **Required.** The ID of the query.               |
| name   | string  | body | The name of the query.               |
| query   | string  | body | The query in SQL syntax.              |
| description   | string  | body | The query's description.               |

#### Example

`PATCH /api/v1/fleet/queries/2`

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

`DELETE /api/v1/fleet/queries/{name}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| name   | string  | path | **Required.** The name of the query.               |

#### Example

`DELETE /api/v1/fleet/queries/{name}`

##### Default response

`Status: 200`

```
{}
```

### Delete query by ID

Deletes the query specified by ID.

`DELETE /api/v1/fleet/queries/id/{id}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| id   | integer  | path | **Required.** The ID of the query.               |

#### Example

`DELETE /api/v1/fleet/queries/id/28`

##### Default response

`Status: 200`

```
{}
```

### Delete queries

Deletes the queries specified by ID. Returns the count of queries successfully deleted.

`POST /api/v1/fleet/queries/delete`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| ids   | list  | body | **Required.** The IDs of the queries.               |

#### Example

`POST /api/v1/fleet/queries/delete`

##### Request body

```
{
  "ids": [
    2, 24, 25
  ]
}
```

##### Default response

`Status: 200`

```
{
  "deleted": 3
}
```

### Get queries specs

Returns a list of all queries in the Fleet instance. Each item returned includes the name, description, and SQL of the query.

`GET /api/v1/fleet/spec/queries`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/spec/queries`

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

`GET /api/v1/fleet/spec/queries/{name}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| name   | string  | path | **Required.** The name of the query.               |

#### Example

`GET /api/v1/fleet/spec/queries/query1`

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

`POST /api/v1/fleet/spec/queries`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| specs   | list  | body | **Required.** The list of the queries to be created or modified.               |

#### Example

`POST /api/v1/fleet/spec/queries`

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

`POST /api/v1/fleet/queries/run`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| query   | string  | body | **Required.** The SQL of the query.               |
| selected   | object  | body | **Required.** The desired targets for the query specified by ID. This object can contain `hosts` and/or `labels` properties. See examples below.     |

#### Example with one host targeted by ID

`POST /api/v1/fleet/queries/run`

##### Request body

```
{
  "query": "select instance_id from system_info",
  "selected": { 
    "hosts": [171]
  }
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

```
{
  "query": "select instance_id from system_info;",
  "selected": { 
    "labels": [7]
  }
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

Runs the specified query as a live query on the specified hosts or group of hosts. Returns a new live query campaign. Individual hosts must be specified with the host's hostname. Groups of hosts are specified by label name.

`POST /api/v1/fleet/queries/run_by_names`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| query   | string  | body | **Required.** The SQL of the query.              |
| selected   | object  | body | **Required.** The desired targets for the query specified by name. This object can contain `hosts` and/or `labels` properties. See examples below.     |

#### Example with one host targeted by hostname

`POST /api/v1/fleet/queries/run_by_names`

##### Request body

```
{
  "query": "select instance_id from system_info",
  "selected": { 
    "hosts": [
      "macbook-pro.local", 
    ]
  }
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

```
{
  "query": "select instance_id from system_info",
  "selected": { 
    "labels": [
      "All Hosts"
    ]
  }
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

Before you retrieve the live query results, you must create a live query campaign by running the live query. See the documentation for the [Run live query](#run-live-query) endpoint to create a live query campaign.

`/api/v1/fleet/results/websockets`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| token  | string  |  | **Required.** The token used to authenticate with the Fleet API. |
| campaignID   | integer  |  | **Required.** The ID of the live query campaign. |

#### Example

##### Example script to handle request and response

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

##### Detailed request and response walkthrough with example data

##### `webSocket.onopen()`

###### Response data

```
o
```

##### `webSocket.send()`

###### Request data

```
[
  { 
    "type": "auth", 
    "data": { "token": <insert_token_here> } 
  }
]
```

```
[
  {
    "type": "select_campaign", 
    "data": { "campaign_id": 12 }
  }
]
```

##### `webSocket.onmessage()`

###### Response data

```
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

```
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

```
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

```
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

`/api/v1/fleet/results/`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| token  | string  |  | **Required.** The token used to authenticate with the Fleet API. |
| campaignID   | integer  |  | **Required.** The ID of the live query campaign. |

#### Example

##### Example script to handle request and response

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

##### `socket.onopen()`

###### Response data

```
o
```

##### `socket.send()`

###### Request data

```
[
  { 
    "type": "auth", 
    "data": { "token": <insert_token_here> } 
  }
]
```

```
[
  {
    "type": "select_campaign", 
    "data": { "campaign_id": 12 }
  }
]
```

##### `socket.onmessage()`

###### Response data

```
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

```
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

```
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

```
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

`POST /api/v1/fleet/packs`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| name   | string | body | **Required**. The pack's name. |
| description   | string | body | The pack's description. |
| host_ids   | list | body | A list containing the targeted host IDs. |
| label_ids   | list | body | A list containing the targeted label's IDs. |

#### Example

`POST /api/v1/fleet/packs`

##### Request query parameters

```
{
  "description": "Collects osquery data.",
  "host_ids": [],
  "label_ids": [6],
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

`PATCH /api/v1/fleet/packs/{id}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required.** The pack's id. |
| name   | string | body | The pack's name. |
| description   | string | body | The pack's description. |
| host_ids   | list | body | A list containing the targeted host IDs. |
| label_ids   | list | body | A list containing the targeted label's IDs. |

#### Example

`PATCH /api/v1/fleet/packs/{id}`

##### Request query parameters

```
{
  "description": "MacOS hosts are targeted",
  "host_ids": [],
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

`GET /api/v1/fleet/packs/{id}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required.** The pack's id. |

#### Example

`GET /api/v1/fleet/packs/17`

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

`GET /api/v1/fleet/packs`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| order_key               | string  | query | What to order results by. Can be any column in the packs table.                                                                                                                                                                                                                                |
| order_direction               | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`.   |

#### Example

`GET /api/v1/fleet/packs`

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

`DELETE /api/v1/fleet/packs/{name}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| name   | string | path | **Required.** The pack's name. |

#### Example

`DELETE /api/v1/fleet/packs/pack_number_one`

##### Default response

`Status: 200`

```
{}
```

### Delete pack by ID

`DELETE /api/v1/fleet/packs/id/{id}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required.** The pack's ID. |

#### Example

`DELETE /api/v1/fleet/packs/id/1`

##### Default response

`Status: 200`

```
{}
```

### Get scheduled queries in a pack

`GET /api/v1/fleet/packs/{id}/scheduled`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required.** The pack's ID. |

#### Example

`GET /api/v1/fleet/packs/1/scheduled`

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

`POST /api/v1/fleet/schedule`

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

`POST /api/v1/fleet/schedule`

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

`GET /api/v1/fleet/schedule/{id}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required.** The scheduled query's ID. |

#### Example

`GET /api/v1/fleet/schedule/56`

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

`PATCH /api/v1/fleet/schedule/{id}`

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

`PATCH /api/v1/fleet/schedule/56`

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

`DELETE /api/v1/fleet/schedule/{id}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| id   | integer | path | **Required.** The scheduled query's ID. |

#### Example

`DELETE /api/v1/fleet/schedule/56`

##### Default response

`Status: 200`

```
{}
```

### Get packs specs

Returns the specs for all packs in the Fleet instance.

`GET /api/v1/fleet/spec/packs`

#### Example

`GET /api/v1/fleet/spec/packs`

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

`POST /api/v1/fleet/spec/packs`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| specs   | list | body | **Required.** A list that includes the specs for each pack to be added to the Fleet instance. |

#### Example

`POST /api/v1/fleet/spec/packs`

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

`GET /api/v1/fleet/spec/packs/{name}`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| name   | string | path | **Required.** The pack's name. |

#### Example

`GET /api/v1/fleet/spec/packs/pack_1`

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

## Targets

In Fleet, targets are used to run queries against specific hosts or groups of hosts. Labels are used to create groups in Fleet.

### Search targets

The search targets endpoint returns two lists. The first list includes the possible target hosts in Fleet given the search query provided and the hosts already selected as targets. The second list includes the possible target labels in Fleet given the search query provided and the labels already selected as targets.

`POST /api/v1/fleet/targets`

#### Parameters

| Name | Type    | In    | Description                  |
| ---- | ------- | ----- | ---------------------------- |
| query   | string | body | The search query. Searchable items include a host's hostname or IPv4 address and labels. |
| selected   | object | body | The targets already selected. The object includes a `hosts` property which contains a list of host IDs and a `labels` property which contains a list of label IDs.|

#### Example

`POST /api/v1/fleet/targets`

##### Request body

```
{
  "query": "172",
  "selected": {
    "hosts": [], 
    "labels": [7]
  }
}
```

##### Default response

```
{
  "targets": {
    "hosts": [
      {
        "created_at": "2021-02-03T16:11:43Z",
        "updated_at": "2021-02-03T21:58:19Z",
        "id": 3,
        "detail_updated_at": "2021-02-03T21:58:10Z",
        "label_updated_at": "2021-02-03T21:58:10Z",
        "last_enrolled_at": "2021-02-03T16:11:43Z",
        "seen_time": "2021-02-03T21:58:20Z",
        "hostname": "7a2f41482833",
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
        "computer_name": "7a2f41482833",
        "primary_ip": "172.20.0.3",
        "primary_mac": "02:42:ac:14:00:03",
        "distributed_interval": 10,
        "config_tls_refresh": 10,
        "logger_tls_period": 10,
        "additional": {},
        "enroll_secret_name": "default",
        "status": "offline",
        "display_text": "7a2f41482833"
      },
      {
        "created_at": "2021-02-03T16:11:43Z",
        "updated_at": "2021-02-03T21:58:19Z",
        "id": 4,
        "detail_updated_at": "2021-02-03T21:58:10Z",
        "label_updated_at": "2021-02-03T21:58:10Z",
        "last_enrolled_at": "2021-02-03T16:11:43Z",
        "seen_time": "2021-02-03T21:58:20Z",
        "hostname": "78c96e72746c",
        "uuid": "a2064cef-0000-0000-afb9-283e3c1d487e",
        "platform": "ubuntu",
        "osquery_version": "4.5.1",
        "os_version": "Ubuntu 16.4.0",
        "build": "",
        "platform_like": "debian",
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
        "computer_name": "78c96e72746c",
        "primary_ip": "172.20.0.7",
        "primary_mac": "02:42:ac:14:00:07",
        "distributed_interval": 10,
        "config_tls_refresh": 10,
        "logger_tls_period": 10,
        "additional": {},
        "enroll_secret_name": "default",
        "status": "offline",
        "display_text": "78c96e72746c"
      }
    ],
    "labels": [
      {
        "created_at": "2021-02-02T23:55:25Z",
        "updated_at": "2021-02-02T23:55:25Z",
        "id": 6,
        "name": "All Hosts",
        "description": "All hosts which have enrolled in Fleet",
        "query": "select 1;",
        "label_type": "builtin",
        "label_membership_type": "dynamic",
        "host_count": 5,
        "display_text": "All Hosts",
        "count": 5
      }
    ]
  },
  "targets_count": 1,
  "targets_online": 1,
  "targets_offline": 0,
  "targets_missing_in_action": 0
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
- [Version](#version)

The Fleet server exposes a handful of API endpoints that handle the configuration of Fleet as well as endpoints that manage invitation and enroll secret operations. All the following endpoints require prior authentication meaning you must first log in successfully before calling any of the endpoints documented below.

### Get certificate

Returns the Fleet certificate.

`GET /api/v1/fleet/config/certificate`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/config/certificate`


##### Default response

`Status: 200`

```
{
  "certificate_chain": <certificate_chain>
}
```

### Get configuration

Returns all information about the Fleet's configuration.

`GET /api/v1/fleet/config`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/config`


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

`PATCH /api/v1/fleet/config`

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
| authentication_type | string | body | *SMTP settings*. The authentication type used by the SMTP server. Options include `"authtype_username_and_password"` or `"none"`|
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

`PATCH /api/v1/fleet/config`

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

`GET /api/v1/fleet/spec/enroll_secret`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/spec/enroll_secret`


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

`POST /api/v1/fleet/spec/enroll_secret`

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

`POST /api/v1/fleet/spec/enroll_secret`


##### Default response

`Status: 200`

```
{}
```

### Create invite


`POST /api/v1/fleet/invites`

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

`POST /api/v1/fleet/invites`


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

`GET /api/v1/fleet/invites`

#### Parameters

| Name                  | Type   | In   | Description                                                     |
| --------------------- | ------ | ---- | --------------------------------------------------------------- |
| order_key               | string  | query | What to order results by. Can be any column in the invites table.                                                                                                                                                                                                                                |
| order_direction               | string  | query | **Requires `order_key`**. The direction of the order given the order key. Options include `asc` and `desc`. Default is `asc`.   |

#### Example

`GET /api/v1/fleet/invites`


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

`DELETE /api/v1/fleet/invites/{id}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| id   | integer  | path | **Required.** The user's id.            |

#### Example

`DELETE /api/v1/fleet/invites/{id}`


##### Default response

`Status: 200`

```
{}
```

### Verify invite

Verify the specified invite.

`GET /api/v1/fleet/invites/{token}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| token   | integer  | path | **Required.** Token provided to the user in the invitation email.|

#### Example

`GET /api/v1/fleet/invites/{token}`


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

### Change email

Changes the email specified by token.

`GET /api/v1/fleet/email/change/{token}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| token   | integer  | path | **Required.** The token provided to the user in the email change confirmation email.|

#### Example

`GET /api/v1/fleet/invites/{token}`


##### Default response

`Status: 200`

```
{
  "new_email": janedoe@example.com
}
```
---

### Version

Get version and build information from the Fleet server.

`GET /api/v1/fleet/version`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/version`

##### Default response

`Status: 200`

```
{
  "version": "3.9.0-93-g1b67826f-dirty",
  "branch": "version",
  "revision": "1b67826fe4bf40b2f45ec53e01db9bf467752e74",
  "go_version": "go1.15.7",
  "build_date": "2021-03-27T00:28:48Z",
  "build_user": "zwass"
}
```
---

## Osquery options

- [Get osquery options spec](#get-osquery-options-spec)
- [Modify osquery options spec](#modify-osquery-options-spec)

### Get osquery options spec

Retrieve the osquery options configured via Fleet.

`GET /api/v1/fleet/spec/osquery_options`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/spec/osquery_options`


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

`POST /api/v1/fleet/spec/osquery_options`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| spec   | JSON  | body | **Required.** The modified osquery spec.            |

#### Example

`POST /api/v1/fleet/spec/osquery_options`

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

## File carving

- [List carves](#list-carves)
- [Get carve](#get-carve)
- [Get carve block](#get-carve-block)

Fleet supports osquery's file carving functionality as of Fleet 3.3.0. This allows the Fleet server to request files (and sets of files) from osquery agents, returning the full contents to Fleet.

To initiate a file carve using the Fleet API, you can use the [live query](#run-live-query) or [scheduled query](#add-scheduled-query-to-a-pack) endpoints to run a query against the `carves` table. 

For more information on executing a file carve in Fleet, go to the [File carving with Fleet docs](../1-Using-Fleet/2-fleetctl-CLI.md#file-carving-with-fleet).

### List carves

Retrieves a list of the non expired carves. Carve contents remain available for 24 hours after the first data is provided from the osquery client.

`GET /api/v1/fleet/carves`

#### Parameters

None.

#### Example

`GET /api/v1/fleet/carves`

##### Default response

`Status: 200`

```
{
  "carves": [
    {
      "id": 1,
      "created_at": "2021-02-23T22:52:01Z",
      "host_id": 7,
      "name": "macbook-pro.local-2021-02-23T22:52:01Z-fleet_distributed_query_30",
      "block_count": 1,
      "block_size": 2000000,
      "carve_size": 2048,
      "carve_id": "c6958b5f-4c10-4dc8-bc10-60aad5b20dc8",
      "request_id": "fleet_distributed_query_30",
      "session_id": "065a1dc3-40ad-441c-afff-80c2ad7dac28",
      "expired": false,
      "max_block": 0
    },
    {
      "id": 2,
      "created_at": "2021-02-23T22:53:03Z",
      "host_id": 7,
      "name": "macbook-pro.local-2021-02-23T22:53:03Z-fleet_distributed_query_31",
      "block_count": 2,
      "block_size": 2000000,
      "carve_size": 3400704,
      "carve_id": "2b9170b9-4e11-4569-a97c-2f18d18bec7a",
      "request_id": "fleet_distributed_query_31",
      "session_id": "f73922ed-40a4-4e98-a50a-ccda9d3eb755",
      "expired": false,
      "max_block": 1
    }
  ]
}
```

### Get carve

Retrieves the specified carve.

`GET /api/v1/fleet/carves/{id}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| id   | integer  | path | **Required.** The desired carve's ID.            |

#### Example

`GET /api/v1/fleet/carves/1`

##### Default response

`Status: 200`

```
{
  "carve": {
    "id": 1,
    "created_at": "2021-02-23T22:52:01Z",
    "host_id": 7,
    "name": "macbook-pro.local-2021-02-23T22:52:01Z-fleet_distributed_query_30",
    "block_count": 1,
    "block_size": 2000000,
    "carve_size": 2048,
    "carve_id": "c6958b5f-4c10-4dc8-bc10-60aad5b20dc8",
    "request_id": "fleet_distributed_query_30",
    "session_id": "065a1dc3-40ad-441c-afff-80c2ad7dac28",
    "expired": false,
    "max_block": 0
  }
}
```

### Get carve block

Retrieves the specified carve block. This endpoint retrieves the data that was carved.

`GET /api/v1/fleet/carves/{id}/block/{block_id}`

#### Parameters

| Name       | Type    | In   | Description                                      |
| ---------- | ------- | ---- | ------------------------------------------------ |
| id   | integer  | path | **Required.** The desired carve's ID.            |
| block_id   | integer  | path | **Required.** The desired carve block's ID.            |

#### Example

`GET /api/v1/fleet/carves/1/block/0`

##### Default response

`Status: 200`

```
{
    "data": "aG9zdHMAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA..."
}
```

---
