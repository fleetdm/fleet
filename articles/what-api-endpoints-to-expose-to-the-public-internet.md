# Which API endpoints to expose to the public internet?

This guide details which API endpoints to make publicly accessible.

## Managing hosts that can travel outside VPN or intranet

If you would like to manage hosts that can travel outside your VPN or intranet, we recommend only exposing the osquery endpoints to the public internet:

- `/api/osquery`
- `/api/v1/osquery`

## Using Fleet Desktop on remote devices

If you are using Fleet Desktop and want it to work on remote devices, the bare minimum API to expose is `/api/latest/fleet/device/*/desktop`. This minimal endpoint will only provide the number of failing policies.

For full Fleet Desktop and scripts functionality, `/api/fleet/orbit/*` and`/api/fleet/device/ping` must also be exposed.

## Using fleetctl CLI from outside of your network

If you would like to use the fleetctl CLI from outside of your network, the following endpoints will also need to be exposed for `fleetctl`:

- `/api/setup`
- `/api/v1/setup`
- `/api/latest/fleet/*`
- `/api/v1/fleet/*`

## Using Fleet's MDM features

If you would like to use Fleet's MDM features, the following endpoints need to be exposed:

- `/mdm/apple/scep` to allow hosts to obtain a SCEP certificate.
- `/mdm/apple/mdm` to allow hosts to reach the server using the MDM protocol.
- `/api/mdm/apple/enroll` to allow DEP-enrolled devices to get an enrollment profile.
- `/api/v1/fleet/device/*` to give end users access to their **My device** page. This page is where they download their manual enrollment profile, rotate their disk encryption key, and use other features. For more information on these API endpoints see the documentation [here](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/API-for-contributors.md#device-authenticated-routes).

> The `/mdm/apple/scep` and `/mdm/apple/mdm` endpoints are outside of the `/api` path because they
> are not RESTful and are not intended for use by API clients or browsers.

### Advanced

- `/mdm/apple/scep`, `/mdm/apple/mdm`, and `/api/mdm/apple/enroll` don't support mTLS.
- All `/api/v1/fleet/device/*` endpoints support mTLS.
- The base URL for all MDM features that use `/api/v1/fleet/device/*` API endpoints don't support `ORBIT_FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST`.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="mike-j-thomas">
<meta name="authorFullName" value="Mike Thomas">
<meta name="publishedOn" value="2023-11-13">
<meta name="articleTitle" value="Which API endpoints to expose to the public internet?">
