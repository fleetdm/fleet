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

- `/mdm/apple/scep`: Allows hosts to obtain a SCEP certificate.
- `/mdm/apple/mdm`: Allows hosts to reach the server using the MDM protocol.
- `/api/mdm/apple/enroll`: If you use automatic enrollment, allows hosts to get an enrollment profile.
- `/api/*/fleet/device/*`: Provides end users access to their **My device** page.
  - This page is where they download their manual enrollment profile, rotate their disk encryption key, and use other features. For more information on these API endpoints see the documentation [here](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/API-for-contributors.md#device-authenticated-routes).
- `/api/*/fleet/mdm/sso` and `/api/*/fleet/mdm/sso/callback`: If you use automatic enrollment and you require [end user authentication](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#end-user-authentication-and-eula) during out-of-the-box macOS setup, allows end users to authenticate with your IdP.
- `/api/*/fleet/mdm/setup/eula/*`: If you use automatic enrollment and you require that the end user agrees to an [End User License Agreement (EULA)](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#end-user-authentication-and-eula) during out-of-the-box macOS setup, allows end user to see the EULA.
- `/api/*/fleet/mdm/bootstrap`: If you use automatic enrollment and you install a [bootstrap package](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#bootstrap-package) during out-of-the-box macOS setup, installs the bootstrap package.

> The `/mdm/apple/scep` and `/mdm/apple/mdm` endpoints are outside of the `/api` path because they
> are not RESTful and are not intended for use by API clients or browsers.

### Windows

If you would like to use Fleet's Windows MDM features, the following endpoints need to be exposed:

- `/api/mdm/microsoft/management`: Allows host to get MDM commands and profiles once the host.
  - See the [Mobile Device Management Protocol specification](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mdm/33769a92-ac31-47ef-ae7b-dc8501f7104f).
- `/api/mdm/microsoft/discovery`: Allows hosts to get information from the MDM server.
  - See the [section 3.1 on the MS-MDE2 specification](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/2681fd76-1997-4557-8963-cf656ab8d887) for more details.
- `/api/mdm/microsoft/policy`: Delivers the enrollment policies required to issue identity certificates to hosts.
  - See the [section 3.3 on the MS-MDE2 specification](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-xcep/08ec4475-32c2-457d-8c27-5a176660a210) for more details.
- `/api/mdm/microsoft/enroll`: Delivers WS-Trust X.509v3 Token Enrollment (MS-WSTEP) functionality.
  - See the [section 3.4 on the MS-MDE2 specification](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-wstep/4766a85d-0d18-4fa1-a51f-e5cb98b752ea) for more details.
- `/api/mdm/microsoft/tos`: Presents end users with the Terms of Service agreement during out-of-the-box Windows setup. Required for automatic enrollment.
- `/api/mdm/microsoft/auth`: If you use automatic enrollment, authenticates end users during out-of-the-box Windows setup. 
  - See the [section 3.2 on the MS-MDE2 specification](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-mde2/27ed8c2c-0140-41ce-b2fa-c3d1a793ab4a) for more details.

### SCEP proxy

If you would like to use Fleet as a SCEP proxy, the following endpoint needs to be exposed:

- `/mdm/scep/proxy/*`: Allows hosts to obtain a SCEP certificate from a configured SCEP server.

## Advanced

The `/api/*/fleet/*` endpoints accessed by the fleetd agent can use mTLS with the certificate provided via the `--fleet-tls-client-certificate` flag in the `fleetctl package` command.

The `/mdm/apple/mdm` and `/api/mdm/apple/enroll` endpoints can use mTLS with the SCEP certificate issued by the Fleet server.

These endpoints don't use mTLS:
- `/mdm/apple/scep`
- `/api/mdm/microsoft/discovery`
- `/api/mdm/microsoft/auth`
- `/api/mdm/microsoft/policy`
- `/api/mdm/microsoft/enroll`
- `/api/mdm/microsoft/management`
- `/api/mdm/microsoft/tos`

For macOS and Windows, the MDM client on the host will send the client certificate in a header. The Fleet server always does additional verification of this certificate.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="mike-j-thomas">
<meta name="authorFullName" value="Mike Thomas">
<meta name="publishedOn" value="2023-11-13">
<meta name="articleTitle" value="Which API endpoints to expose to the public internet?">
