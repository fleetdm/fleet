# Conditional access: PingFederate

_Available in Fleet Premium_

With Fleet's [conditional access](https://fleetdm.com/guides/conditional-access), PingFederate blocks sign-in from hosts that aren't managed by Fleet or are failing critical Fleet policies. It works on macOS, Windows, and Linux.

How it works:

1. Fleet deploys a certificate to each host. The certificate's common name (CN) is the host's UUID.
2. At sign-in, PingFederate's [X.509 Certificate Integration Kit](https://docs.pingidentity.com/integrations/x509/x509_certificate_integration_kit/pf_x509_certificate_ik.html) asks the browser for the certificate and checks that your CA issued it.
3. PingFederate looks up the UUID with Fleet's [Get host by identifier](https://fleetdm.com/docs/rest-api/rest-api#get-host-by-identifier) API, then gets the host's [device health report](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-device-health-report). It denies sign-in if the host isn't found or is failing a critical policy.

## Prerequisites

- A certificate authority (CA) connected to Fleet. See [supported CAs](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate). Best practice is a dedicated issuing CA used only for these certificates, because PingFederate trusts every certificate the CA issues.
- PingFederate with the X.509 Certificate Integration Kit [installed](https://docs.pingidentity.com/integrations/x509/x509_certificate_integration_kit/pf_x509_certificate_ik_deploying_the_integration_files.html), including setting `pf.secondary.https.port`.

## Step 1: Create a Fleet API user

Create an [API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user) with the **Observer** role. PingFederate uses its API token to look up hosts.

Select **Specific API endpoints** and add only the endpoints PingFederate calls:

- [Get host by identifier](https://fleetdm.com/docs/rest-api/rest-api#get-host-by-identifier)
- [Get host's device health report](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-device-health-report)

## Step 2: Mark critical policies

In Fleet, head to **Policies**. For each policy that should block sign-in, open the policy, select **Critical**, and select **Save**.

## Step 3: Deploy the certificate

For every platform, set the certificate's CN to `$FLEET_VAR_HOST_UUID` and include the TLS client authentication extended key usage.

### macOS

Deploy a SCEP [configuration profile](https://fleetdm.com/guides/custom-os-settings) for your CA. Set `PayloadScope` to `User` so the certificate lands in the login keychain, where browsers can use it.

### Windows

Deploy a SCEP profile for your CA using `./User/` paths so the certificate lands in the user certificate store. See [Enable Okta Verify on Windows](https://fleetdm.com/guides/enable-okta-verify-on-windows-using-a-scep-configuration-profile) for an example profile.

### Linux

1. Follow the [Hydrant](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#hydrant) or [EST](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#any-est-enrollment-over-secure-transport-ca) steps. In the script, set the CSR's subject to `/CN=$FLEET_VAR_HOST_UUID`.
2. Linux browsers don't read certificates from the filesystem. Append [`import-certificate-to-browsers.sh`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/linux/scripts/import-certificate-to-browsers.sh) to your script, updating the paths. It imports the certificate into each user's Chrome and Firefox stores, including snap installs. Hosts need `libnss3-tools` (Debian/Ubuntu) or `nss-tools` (RHEL).

## Step 4: Configure the X.509 adapter

1. In PingFederate, import your CA's certificate as a trusted CA.
2. Create an X.509 Certificate IdP Adapter instance.
3. Map the certificate's CN to an adapter attribute (for example, `hostUUID`).

## Step 5: Add Fleet as a data store

1. Go to **System > Data & Credential Stores > Data Stores** and select **Add New Data Store**.
2. For **Type**, select **Rest API**.
3. For **Base URL**, enter your Fleet server URL.
4. Add the Fleet API token from Step 1 as an `Authorization: Bearer <token>` header.
5. Add these attributes:

| Local attribute | JSON response attribute path |
|---|---|
| `fleetHostUUID` | `/host/uuid` |
| `fleetHostID` | `/host/id` |
| `failingCriticalPolicies` | `/health/failing_critical_policies_count` |

## Step 6: Build the authentication policy

1. Add the X.509 adapter from Step 4 to your authentication policy.
2. On the adapter's success path, map to a policy contract. On **Attribute Sources & User Lookup**, add the Fleet data store twice:
   - First, with the resource path `/api/v1/fleet/hosts/identifier/${hostUUID}?exclude_software=true`
   - Second, with the resource path `/api/v1/fleet/hosts/${fleetHostID}/health`
3. On **Issuance Criteria**, add:
   - `fleetHostUUID` equals `${hostUUID}` (host is in Fleet)
   - `failingCriticalPolicies` equals `0` (host is passing critical policies)

Fleet updates a host's policy results each time the host runs its policies, about once an hour. So a user who fixes a failing policy is still blocked until the host rechecks. If they select **Refetch** on their **My device** page, results usually update within a minute or so, and their next sign-in works right away.

## Step 7: Test

| Host | Expected result |
|---|---|
| Managed, passing critical policies | Sign-in succeeds |
| Managed, failing a non-critical policy | Sign-in succeeds |
| Managed, failing a critical policy | Sign-in denied |
| No certificate | Sign-in denied |

## Troubleshooting

- **Browser doesn't prompt for a certificate**: Check that the certificate is in the user store (macOS login keychain, Windows `Cert:\CurrentUser\My`, or Linux NSS). On Linux, run `certutil -L -d sql:$HOME/.pki/nssdb`.
- **Sign-in denied for a managed host**: Check that the certificate's CN matches the host's UUID in Fleet.
- **Sign-in still denied after fixing a policy**: Select **Refetch** on the **My device** page, wait for the policy results to update, then sign in again.
- **Data store errors**: Check the API token and that PingFederate can reach your Fleet server.

To skip the certificate picker in Chrome and Edge, deploy the `AutoSelectCertificateForUrls` policy for your PingFederate URL.

<meta name="articleTitle" value="Conditional access: PingFederate">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-09-29">
<meta name="description" value="Use PingFederate for conditional access, blocking sign-in from hosts that aren't managed by Fleet or are failing critical policies.">
