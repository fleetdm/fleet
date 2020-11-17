# Fleet REST API endpoints

## Hosts

### List hosts

`GET /api/v1/kolide/hosts`

#### Parameters

| Name                    | Type    | In    | Description                                                                              |
|-------------------------|---------|-------|------------------------------------------------------------------------------------------|
| page                    | integer | query | Page number of the results to fetch.                                                     |
| per_page                | integer | query | Results per page.                                                                        |
| order_key               | string  | query | What to order results by. Can be any column in the hosts table.                          |
| status                  | string  | query | Indicates the status of the hosts to return. Can either be `new`, `online`, `offline`, or `mia`.|
| additional_info_filters | string  | query | A comma-delimited list of fields to include in each host's additional information object. See [Fleet Configuration Options](https://github.com/fleetdm/fleet/blob/master/docs/cli/file-format.md#fleet-configuration-options) for an example configuration with hosts' additional information.|

#### Example

`GET /api/v1/kolide/hosts?page=0&per_page=100&order_key=host_name`

Request query parameters

```
{
  "page": 0,
  "per_page": 100,
  "order_key": "host_name",
}
```

Default response

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
*******

## Entrance

### Log in

`POST /api/v1/kolide/login`

#### Parameters

| Name                    | Type    | In    | Description                                       |
|-------------------------|---------|-------|---------------------------------------------------|
| username                | string  | body  | **Required**. The user's email.              |
| password                | string  | body  | **Required**. The user's plain text password.|

#### Example

`POST /api/v1/kolide/login`

##### Request body

```
{
  username: "janedoe@example.com"
  passsword: "VArCjNW7CfsxGp67"
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
*******

### Log out

`POST /api/v1/kolide/logout`

#### Parameters

| Name                    | Type    | In     | Description                                                               |
|-------------------------|---------|--------|---------------------------------------------------------------------------|
| authorization           | string  | header | **Required**. The token received from the `kolide/login` response object. |

#### Example

`POST /api/v1/kolide/logout`

##### Request header

```
{
  "authentication": "Bearer {your token}"
}
```

##### Default response

`Status: 200`
*******

### Forgot password

`POST /api/v1/kolide/forgot_password`

#### Parameters

| Name                    | Type    | In     | Description                                                               |
|-------------------------|---------|--------|---------------------------------------------------------------------------|
| email                   | string  | body   | **Required**. The email of the user requesting the reset password link.   |

#### Example

`POST /api/v1/kolide/logout`

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
*******

<!-- ### Reset password

`POST /api/v1/kolide/reset_password`

#### Parameters

TODO

#### Example

`POST /api/v1/kolide/reset_password`

##### Request body

TODO

##### Default response

`Status: 200`

TODO
******* -->

### Change password

`POST /api/v1/kolide/change_password`

#### Parameters

| Name                    | Type    | In       | Description                                                                 |
|-------------------------|---------|----------|-----------------------------------------------------------------------------|
| authorization           | string  | header   | **Required**. The token received from the `kolide/login` response object.   |
| old_password            | string  | body     | **Required**. The user's old password.                                      |
| new_password            | string  | body     | **Required**. The user's new password.                                      |

#### Example

`POST /api/v1/kolide/change_password`

##### Request header

```
{
  "authentication": "Bearer {your token}"
}
```

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
*******

### Me

`POST /api/v1/kolide/me`

#### Parameters

| Name                    | Type    | In       | Description                                                                 |
|-------------------------|---------|----------|-----------------------------------------------------------------------------|
| authorization           | string  | header   | **Required**. The token received from the `kolide/login` response object.   |

#### Example

`POST /api/v1/kolide/me`

##### Request header

```
{
  "authentication": "Bearer {your token}"
}
```

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
*******

### Perform required password reset

`POST /api/v1/kolide/perform_require_password_reset`

#### Parameters

| Name                    | Type    | In       | Description                                                                 |
|-------------------------|---------|----------|-----------------------------------------------------------------------------|
| authorization           | string  | header   | **Required**. The token received from the `kolide/login` response object.   |

#### Example

`POST /api/v1/kolide/perform_required_password_reset`

##### Request header

```
{
  "authentication": "Bearer {your token}"
}
```

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
*******

### SSO config

`GET /api/v1/kolide/sso`

#### Parameters

| Name                    | Type    | In       | Description                                                                 |
|-------------------------|---------|----------|-----------------------------------------------------------------------------|
| relay_url               | string  | body     | **Required**. The relative url to be navigated to after succesful sign in.  |

#### Example

`GET /api/v1/kolide/sso`

##### Default response

`Status: 200`

```
{
  "settings": {
    "idp_name": "",
    "idp_image_url": "",
    "sso_enabled": false
  }
}
```
*******

### Initiate SSO

`POST /api/v1/kolide/sso`

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
*******

<!-- ### Callback SSO

`POST /api/v1/kolide/sso/callback`

#### Example

`POST /api/v1/kolide/sso/callback`

##### Request body

TODO

##### Default response

`Status: 200`

TODO -->
