# Which API endpoints to expose to the public internet?

This guide details which API endpoints to make publicly accessible.

## Managing hosts that can travel outside VPN or intranet

If you would like to manage hosts that can travel outside your VPN or intranet, we recommend only exposing the osquery endpoints to the public internet:

- `/api/osquery`
- `/api/v1/osquery`

## Using Fleet Desktop on remote devices

If you are using Fleet Desktop and want it to work on remote devices, the bare minimum API to expose is `/api/*/fleet/device/*/desktop`. This minimal endpoint will only provide the number of failing policies.

For full Fleet Desktop and scripts functionality, `/api/fleet/orbit/*` and`/api/fleet/device/ping` must also be exposed.

## Using fleetctl CLI from outside of your network

If you would like to use the fleetctl CLI from outside of your network, the following endpoints will also need to be exposed for `fleetctl`:

- `/api/setup`
- `/api/*/setup`
- `/api/*/fleet/*`

## Using Fleet's MDM features

### macOS

If you would like to use Fleet's macOS MDM features, the following endpoints need to be exposed:

- `/mdm/apple/scep` to allow hosts to obtain a SCEP certificate.
- `/mdm/apple/mdm` to allow hosts to reach the server using the MDM protocol.
- `/api/mdm/apple/enroll` to allow DEP-enrolled devices to get an enrollment profile.
- `/api/*/fleet/device/*` to give end users access to their **My device** page. This page is where they download their manual enrollment profile, rotate their disk encryption key, and use other features. For more information on these API endpoints see the documentation [here](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/API-for-contributors.md#device-authenticated-routes).
- `/api/*/fleet/mdm/sso` and `/api/*/fleet/mdm/sso/callback` if you use automatic enrollment (DEP) and you require [end user authentication](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#end-user-authentication-and-eula) during out-of-the-box macOS setup.
- `/api/*/fleet/mdm/setup/eula/*` if you use automatic enrollment (DEP) and you require that the end user agrees to an [End User License Agreement (EULA)](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#end-user-authentication-and-eula) during out-of-the-box macOS setup.
- `/api/*/fleet/mdm/bootstrap` if you use automatic enrollment (DEP) and you install a [bootstrap package](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#bootstrap-package) during out-of-the-box macOS setup.

> The `/mdm/apple/scep` and `/mdm/apple/mdm` endpoints are outside of the `/api` path because they
> are not RESTful and are not intended for use by API clients or browsers.

### Windows

If you would like to use Fleet's Windows MDM features, the following endpoints need to be exposed:

- `/api/mdm/microsoft/discovery` TODO
- `/api/mdm/microsoft/auth` TODO
- `/api/mdm/microsoft/policy` TODO
- `/api/mdm/microsoft/enroll` TODO
- `/api/mdm/microsoft/management` TODO
- `/api/mdm/microsoft/tos` - if you use automatic enrollment for Windows devices, this endpoints presents end users with the Terms of Service agreement during out-of-the-box Windows setup.

## Advanced

The following endpoints don't support mTLS:
- `/mdm/apple/scep`
- `/api/mdm/microsoft/discovery`
- `/api/mdm/microsoft/auth`
- `/api/mdm/microsoft/policy`
- `/api/mdm/microsoft/enroll`
- `/api/mdm/microsoft/management`
- `/api/mdm/microsoft/tos`

`/mdm/apple/mdm` and `/api/mdm/apple/enroll` support mTLS but require the [SCEP certificate issued by the Fleet server](https://fleetdm.com/docs/configuration/fleet-server-configuration#mdm-apple-scep-cert-bytes) to validate client certificates.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="mike-j-thomas">
<meta name="authorFullName" value="Mike Thomas">
<meta name="publishedOn" value="2023-11-13">
<meta name="articleTitle" value="Which API endpoints to expose to the public internet?">
