# Authentication

## Retrieve your API token

All API requests to the Fleet server require API token authentication unless noted in the documentation. API tokens are tied to your Fleet user account.

To get an API token, retrieve it from "My account" > "Get API token" in the Fleet UI (`/profile`). Or, you can send a request to the [login API endpoint](#log-in) to get your token.

Then, use that API token to authenticate all subsequent API requests by sending it in the "Authorization" request header, prefixed with "Bearer ":

```http
Authorization: Bearer <your token>
```

> For SSO users, email/password login is disabled. The API token can instead be retrieved from the "My account" page in the UI (/profile). On this page, choose "Get API token".

## Log in

Authenticates the user with the specified credentials. Use the token returned from this endpoint to authenticate further API requests.

`POST /api/v1/fleet/login`

> This API endpoint is not available to SSO users, since email/password login is disabled for SSO users. To get an API token for an SSO user, you can use the Fleet UI.

### Parameters

| Name     | Type   | In   | Description                                   |
| -------- | ------ | ---- | --------------------------------------------- |
| email    | string | body | **Required**. The user's email.               |
| password | string | body | **Required**. The user's plain text password. |

### Example

`POST /api/v1/fleet/login`

#### Request body

```json
{
  "email": "janedoe@example.com",
  "password": "VArCjNW7CfsxGp67"
}
```

#### Default response

`Status: 200`

```json
{
  "user": {
    "created_at": "2020-11-13T22:57:12Z",
    "updated_at": "2020-11-13T22:57:12Z",
    "id": 1,
    "name": "Jane Doe",
    "email": "janedoe@example.com",
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false,
    "global_role": "admin",
    "teams": []
  },
  "token": "{your token}"
}
```

#### Authentication failed

`Status: 401 Unauthorized`

```json
{
  "message": "Authentication failed",
  "errors": [
    {
      "name": "base",
      "reason": "Authentication failed"
    }
  ],
  "uuid": "1272014b-902b-4b36-bcdb-75fde5eac1fc"
}
```

#### Too many requests / Rate limiting

`Status: 429 Too Many Requests`
`Header: retry-after: N`

> This response includes a header `retry-after` that indicates how many more seconds you are blocked before you can try again.

```json
{
  "message": "limit exceeded, retry after: Ns",
  "errors": [
    {
      "name": "base",
      "reason": "limit exceeded, retry after: Ns"
    }
  ]
}
```

---

## Log out

Logs out the authenticated user.

`POST /api/v1/fleet/logout`

### Example

`POST /api/v1/fleet/logout`

#### Default response

`Status: 200`

---

## Forgot password

Sends a password reset email to the specified email. Requires that SMTP or SES is configured for your Fleet server.

`POST /api/v1/fleet/forgot_password`

### Parameters

| Name  | Type   | In   | Description                                                             |
| ----- | ------ | ---- | ----------------------------------------------------------------------- |
| email | string | body | **Required**. The email of the user requesting the reset password link. |

### Example

`POST /api/v1/fleet/forgot_password`

#### Request body

```json
{
  "email": "janedoe@example.com"
}
```

#### Default response

`Status: 200`

#### Unknown error

`Status: 500`

```json
{
  "message": "Unknown Error",
  "errors": [
    {
      "name": "base",
      "reason": "email not configured"
    }
  ]
}
```

---

## Change password

`POST /api/v1/fleet/change_password`

Changes the password for the authenticated user.

### Parameters

| Name         | Type   | In   | Description                            |
| ------------ | ------ | ---- | -------------------------------------- |
| old_password | string | body | **Required**. The user's old password. |
| new_password | string | body | **Required**. The user's new password. |

### Example

`POST /api/v1/fleet/change_password`

#### Request body

```json
{
  "old_password": "VArCjNW7CfsxGp67",
  "new_password": "zGq7mCLA6z4PzArC"
}
```

#### Default response

`Status: 200`

#### Validation failed

`Status: 422 Unprocessable entity`

```json
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

## Reset password

Resets a user's password. Which user is determined by the password reset token used. The password reset token can be found in the password reset email sent to the desired user.

`POST /api/v1/fleet/reset_password`

### Parameters

| Name                      | Type   | In   | Description                                                               |
| ------------------------- | ------ | ---- | ------------------------------------------------------------------------- |
| new_password              | string | body | **Required**. The new password.                                           |
| new_password_confirmation | string | body | **Required**. Confirmation for the new password.                          |
| password_reset_token      | string | body | **Required**. The token provided to the user in the password reset email. |

### Example

`POST /api/v1/fleet/reset_password`

#### Request body

```json
{
  "new_password": "abc123",
  "new_password_confirmation": "abc123",
  "password_reset_token": "UU5EK0JhcVpsRkY3NTdsaVliMEZDbHJ6TWdhK3oxQ1Q="
}
```

#### Default response

`Status: 200`


---

## Me

Retrieves the user data for the authenticated user.

`GET /api/v1/fleet/me`

### Example

`GET /api/v1/fleet/me`

#### Default response

`Status: 200`

```json
{
  "user": {
    "created_at": "2020-11-13T22:57:12Z",
    "updated_at": "2020-11-16T23:49:41Z",
    "id": 1,
    "name": "Jane Doe",
    "email": "janedoe@example.com",
    "global_role": "admin",
    "enabled": true,
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false,
    "teams": []
  }
}
```

---

## Perform required password reset

Resets the password of the authenticated user. Requires that `force_password_reset` is set to `true` prior to the request.

`POST /api/v1/fleet/perform_required_password_reset`

### Example

`POST /api/v1/fleet/perform_required_password_reset`

#### Request body

```json
{
  "new_password": "sdPz8CV5YhzH47nK"
}
```

#### Default response

`Status: 200`

```json
{
  "user": {
    "created_at": "2020-11-13T22:57:12Z",
    "updated_at": "2020-11-17T00:09:23Z",
    "id": 1,
    "name": "Jane Doe",
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

---

## SSO config

Gets the current SSO configuration.

`GET /api/v1/fleet/sso`

### Example

`GET /api/v1/fleet/sso`

#### Default response

`Status: 200`

```json
{
  "settings": {
    "idp_name": "IDP Vendor 1",
    "idp_image_url": "",
    "sso_enabled": false
  }
}
```

---

## Initiate SSO

`POST /api/v1/fleet/sso`

### Parameters

| Name      | Type   | In   | Description                                                                 |
| --------- | ------ | ---- | --------------------------------------------------------------------------- |
| relay_url | string | body | **Required**. The relative url to be navigated to after successful sign in. |

### Example

`POST /api/v1/fleet/sso`

#### Request body

```json
{
  "relay_url": "/hosts/manage"
}
```

#### Default response

`Status: 200`

#### Unknown error

`Status: 500`

```json
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

## SSO callback

This is the callback endpoint that the identity provider will use to send security assertions to Fleet. This is where Fleet receives and processes the response from the identify provider.

`POST /api/v1/fleet/sso/callback`

### Parameters

| Name         | Type   | In   | Description                                                 |
| ------------ | ------ | ---- | ----------------------------------------------------------- |
| SAMLResponse | string | body | **Required**. The SAML response from the identity provider. |

### Example

`POST /api/v1/fleet/sso/callback`

#### Request body

```json
{
  "SAMLResponse": "<SAML response from IdP>"
}
```

#### Default response

`Status: 200`

<meta name="description" value="Documentation for Fleet's authentication REST API endpoints.">
<meta name="pageOrderInSection" value="20">