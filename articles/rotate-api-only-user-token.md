# Rotate an API-only user's token

API-only user tokens don't expire, and getting a new token from the [log in API](https://fleetdm.com/docs/rest-api/rest-api#log-in) doesn't revoke the old one. To rotate a token so that the old one stops working, create a new API-only user, move your integrations to its token, then delete the old user. Deleting a user revokes all of its tokens. This guide covers API-only users only, not regular users who sign in to the Fleet UI.

## Step 1: Record the old user's access

The new user needs the same access as the old one, so record it first.

1. Go to **Settings > Users**. Only Fleet users with the global admin role have access to this page.
2. Find the old API-only user. API-only users show **API** next to their name.
3. Note its role and, with Fleet Premium, its fleets and API endpoint restrictions.

To get the same details from the REST API, find the user with [List all users](https://fleetdm.com/docs/rest-api/rest-api#list-all-users):

```sh
curl "https://fleet.example.com/api/v1/fleet/users?query=test-api-user" \
  -H "Authorization: Bearer <admin_token>"
```

Note the old user's `id` from the response. You'll need it to delete the user later.

## Create the new API-only user

Give the new user a name that tells the two users apart, for example `test-api-user (2026-10)`. Create it with the Fleet UI, fleetctl, or the REST API.

### Fleet UI

1. Go to **Settings > Users** and select **Create user**.
2. Select the **API-only** option.
3. Enter a name and assign the same role or fleets as the old user.
4. With Fleet Premium, add the same API endpoint restrictions as the old user, if it had any.
5. Create the user and copy its API token.

### fleetctl

```sh
fleetctl user create --name 'test-api-user (2026-10)' --api-only --global-role 'observer'
```

With Fleet Premium, use `--fleet <fleet_id>:<role>` instead of `--global-role` to limit the user to a fleet:

```sh
fleetctl user create --name 'test-api-user (2026-10)' --api-only --fleet 4:gitops
```

`--email` and `--password` are optional. If you omit them, Fleet generates an email address, and the user authenticates with its API token only. fleetctl prints the token once, so copy it before you close the terminal.

### REST API

Use [Create API-only user](https://fleetdm.com/docs/rest-api/rest-api#create-api-only-user):

```sh
curl -X POST "https://fleet.example.com/api/v1/fleet/users/api_only" \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"test-api-user (2026-10)","global_role":"observer"}'
```

The response includes the new user's `token`. Specify either `global_role` or `fleets`, not both. With Fleet Premium, add `api_endpoints` to restrict the user to specific endpoints.

> **Note:** Use the `api_only` endpoint, not [Create a user](https://fleetdm.com/docs/rest-api/rest-api#create-a-user) (`/api/v1/fleet/users/admin`). A user created through `/users/admin` may be flagged for a password reset, and its token returns `password reset required` errors until the reset is done.

## Update your integrations

Replace the old token with the new one everywhere it's used: secrets managers, CI variables, identity provider settings (for example, Okta SCIM provisioning), and fleetctl contexts.

For a fleetctl context, edit `~/.fleet/config` and replace the `token` value. Editing the file, rather than passing the token on the command line, keeps it out of your shell history.

## Verify the new token

Before you delete the old user, confirm the new token works:

```sh
curl "https://fleet.example.com/api/v1/fleet/me" \
  -H "Authorization: Bearer <new_token>"
```

If the token works, Fleet returns the new user's details, including its `name` and `"api_only": true`. If Fleet returns an error message instead, the token isn't valid. Check that you copied the whole token.

Then trigger each integration once and confirm it succeeds.

## Delete the old API-only user

> **Warning:** Deleting the old user immediately revokes its token. Any integration still using that token stops working. Move every integration to the new token before you delete the old user.

### Fleet UI

1. Go to **Settings > Users**.
2. Find the old API-only user.
3. Select **Actions > Delete**, then confirm.

### REST API

Use [Delete user](https://fleetdm.com/docs/rest-api/rest-api#delete-user) with the old user's `id`:

```sh
curl -X DELETE "https://fleet.example.com/api/v1/fleet/users/<old_user_id>" \
  -H "Authorization: Bearer <admin_token>"
```

## Confirm the old token is revoked

Confirm that the old token no longer works and the new one does.

1. Run the `curl` command from [Verify the new token](#verify-the-new-token) with the old token. Fleet should return an error message instead of user details.
2. Run it again with the new token. Fleet should return the new user's details.

## Troubleshoot

**The new token returns a `password reset required` error**

The user was created through `/api/v1/fleet/users/admin` and is flagged for a password reset. Delete it and create the user again with the `api_only` endpoint, fleetctl, or the Fleet UI.

**The new token returns a permission error on some endpoints**

The new user's role, fleets, or API endpoint restrictions don't match the old user's. Compare them with the details you recorded and update the new user in **Settings > Users**.

**An integration started failing after you deleted the old user**

That integration was still using the old token. Update it with the new token. The old token can't be restored.

**You can't find the old user by email**

API-only users created without `--email` have a generated email address. Search for the user by name in **Settings > Users** or with the `query` parameter in [List all users](https://fleetdm.com/docs/rest-api/rest-api#list-all-users).

## Further reading

- [Using fleetctl with an API-only user](https://fleetdm.com/guides/fleetctl#using-fleetctl-with-an-api-only-user)
- [Create API-only user REST API reference](https://fleetdm.com/docs/rest-api/rest-api#create-api-only-user)
- [Log in REST API reference](https://fleetdm.com/docs/rest-api/rest-api#log-in)
- [Role-based access in Fleet](https://fleetdm.com/guides/role-based-access)

<meta name="articleTitle" value="Rotate an API-only user's token">
<meta name="authorFullName" value="Matthew Thomas">
<meta name="authorGitHubUsername" value="adriant172">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-21">
<meta name="description" value="Rotate an API-only user's token by creating a new API-only user and deleting the old one.">
