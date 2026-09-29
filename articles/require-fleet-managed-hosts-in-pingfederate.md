# Require Fleet-managed hosts in PingFederate

_Available in Fleet Premium_

This guide shows how to block PingFederate sign-in from hosts that aren't managed by Fleet, or that are failing Fleet policies. It works on macOS, Windows, and Linux.

How it works:

1. Fleet deploys a certificate to each host. The certificate's common name (CN) is the host's UUID.
2. At sign-in, PingFederate's [X.509 Certificate Integration Kit](https://docs.pingidentity.com/integrations/x509/x509_certificate_integration_kit/pf_x509_certificate_ik.html) asks the browser for the certificate and checks that your CA issued it.
3. PingFederate looks up the UUID with Fleet's [Get host by identifier](https://fleetdm.com/docs/rest-api/rest-api#get-host-by-identifier) API and denies sign-in if the host isn't found or is failing policies.


## Prerequisites

- A certificate authority (CA) connected to Fleet. See [supported CAs](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate). Best practice is a dedicated issuing CA used only for these certificates, because PingFederate trusts every certificate the CA issues.
- PingFederate with the X.509 Certificate Integration Kit [installed](https://docs.pingidentity.com/integrations/x509/x509_certificate_integration_kit/pf_x509_certificate_ik_deploying_the_integration_files.html), including setting `pf.secondary.https.port`.

## Step 1: Create a Fleet API user

Create an [API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user) with the **Observer** role. PingFederate uses its API token to look up hosts.

## Step 2: Deploy the certificate

For every platform, set the certificate's CN to `$FLEET_VAR_HOST_UUID` and include the TLS client authentication extended key usage.

### macOS

Deploy a SCEP [configuration profile](https://fleetdm.com/guides/custom-os-settings) for your CA. Set `PayloadScope` to `User` so the certificate lands in the login keychain, where browsers can use it.

### Windows

Deploy a SCEP profile for your CA using `./User/` paths so the certificate lands in the user certificate store. See [Enable Okta Verify on Windows](https://fleetdm.com/guides/enable-okta-verify-on-windows-using-a-scep-configuration-profile) for an example profile.

### Linux

1. Follow the [Hydrant](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#hydrant) or [EST](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#any-est-enrollment-over-secure-transport-ca) steps. In the script, set the CSR's subject to `/CN=$FLEET_VAR_HOST_UUID`.
2. Linux browsers don't read certificates from the filesystem. Append [`import-certificate-to-browsers.sh`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/linux/scripts/import-certificate-to-browsers.sh) to your script, updating the paths. It imports the certificate into each user's Chrome and Firefox stores, including snap installs. Hosts need `libnss3-tools` (Debian/Ubuntu) or `nss-tools` (RHEL).


## Step 3: Configure the X.509 adapter

1. In PingFederate, import your CA's certificate as a trusted CA.
2. Create an X.509 Certificate IdP Adapter instance.
3. Map the certificate's CN to an adapter attribute (for example, `hostUUID`).

## Step 4: Add Fleet as a data store

1. Go to **System > Data & Credential Stores > Data Stores** and select **Add New Data Store**.
2. For **Type**, select **Rest API**.
3. For **Base URL**, enter your Fleet server URL.
4. Add the Fleet API token from Step 1 as an `Authorization: Bearer <token>` header.
5. Add these attributes:

| Local attribute | JSON response attribute path |
|---|---|
| `fleetHostUUID` | `/host/uuid` |
| `failingPolicies` | `/host/issues/failing_policies_count` |

## Step 5: Build the authentication policy

1. Add the X.509 adapter from Step 3 to your authentication policy.
2. On the adapter's success path, map to a policy contract. On **Attribute Sources & User Lookup**, add the Fleet data store with the resource path `/api/v1/fleet/hosts/identifier/${hostUUID}?exclude_software=true`.
3. On **Issuance Criteria**, add:
   - `fleetHostUUID` equals `${hostUUID}` (host is in Fleet)
   - `failingPolicies` equals `0` (host is passing policies)

To only block on specific policies, gate on the policy's result in `/host/policies` instead.

## Step 6: Test

| Host | Expected result |
|---|---|
| Managed, passing policies | Sign-in succeeds |
| No certificate | Sign-in denied |
| Managed, failing a policy | Sign-in denied |

## Troubleshooting

- **Browser doesn't prompt for a certificate**: Check that the certificate is in the user store (macOS login keychain, Windows `Cert:\CurrentUser\My`, or Linux NSS). On Linux, run `certutil -L -d sql:$HOME/.pki/nssdb`.
- **Sign-in denied for a managed host**: Check that the certificate's CN matches the host's UUID in Fleet.
- **Data store errors**: Check the API token and that PingFederate can reach your Fleet server.

To skip the certificate picker in Chrome and Edge, deploy the `AutoSelectCertificateForUrls` policy for your PingFederate URL.

<meta name="articleTitle" value="Require Fleet-managed hosts in PingFederate">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-09-29">
<meta name="description" value="Block PingFederate sign-in from hosts that aren't managed by Fleet or are failing policies.">
