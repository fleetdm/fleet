# Conditional access: PingFederate

_Available in Fleet Premium_

With Fleet's [conditional access](https://fleetdm.com/guides/conditional-access), PingFederate blocks sign-in from hosts that aren't managed by Fleet or are failing critical Fleet policies. It works on macOS, Windows, and Linux.

How it works:

1. Fleet deploys a certificate to each host. The certificate's common name (CN) is the host's UUID.
2. At sign-in, PingFederate's [X.509 Certificate Integration Kit](https://docs.pingidentity.com/integrations/x509/x509_certificate_integration_kit/pf_x509_certificate_ik.html) asks the browser for the certificate and checks that your CA issued it.
3. PingFederate looks up the UUID with Fleet's [Get host by identifier](https://fleetdm.com/docs/rest-api/rest-api#get-host-by-identifier) API, then gets the host's [device health report](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-device-health-report). It denies sign-in if the host isn't found or is failing a critical policy.

## Prerequisites

- A certificate authority (CA) connected to Fleet. See [supported CAs](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate). Best practice is a dedicated issuing CA used only for these certificates, because PingFederate trusts every certificate the CA issues.
- Hosts that trust the certificate PingFederate presents. If your own CA issued it, deploy that CA's root certificate to your hosts with a certificate profile. Browsers show a security warning otherwise.
- PingFederate with the X.509 Certificate Integration Kit [installed](https://docs.pingidentity.com/integrations/x509/x509_certificate_integration_kit/pf_x509_certificate_ik_deploying_the_integration_files.html), including setting `pf.secondary.https.port`.

## Step 1: Create a Fleet API user

Create an [API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user) with the **Observer** role. PingFederate uses its API token to look up hosts.

Select **Specific API endpoints** and add only the endpoints PingFederate calls:

- [Get host by identifier](https://fleetdm.com/docs/rest-api/rest-api#get-host-by-identifier)
- [Get host's device health report](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-device-health-report)

## Step 2: Mark critical policies

In Fleet, head to **Policies**. For each policy that should block sign-in, open the policy, select **Critical**, and select **Save**.

## Step 3: Deploy the certificate

For every platform, set the certificate's CN to `$FLEET_VAR_HOST_UUID`. The certificate also needs the TLS client authentication extended key usage (EKU). Your CA sets the EKU, not the profile, so check your CA's certificate template.

### macOS

Deploy a SCEP [configuration profile](https://fleetdm.com/guides/custom-os-settings) for your CA. Set `PayloadScope` to `User` so the certificate lands in the login keychain, where browsers can use it. In the SCEP payload, set `AllowAllAppsAccess` to `true` and `KeyIsExtractable` to `false`. Even then, macOS asks the end user for their login keychain password the first time Safari, Chrome or Firefox uses the certificate. They select **Always Allow** once for each browser. See the [custom SCEP example](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate) for a profile.

### Windows

Deploy a SCEP profile for your CA using `./User/` paths so the certificate lands in the user certificate store. See [Enable Okta Verify on Windows](https://fleetdm.com/guides/enable-okta-verify-on-windows-using-a-scep-configuration-profile) for an example profile. The example sets `SubjectName` to the host's serial number. Change it to `CN=$FLEET_VAR_HOST_UUID,OU=$FLEET_VAR_CERTIFICATE_RENEWAL_ID` so the CN is the host's UUID and Fleet can renew the certificate.

### Linux

1. Follow the [Hydrant](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#hydrant) or [EST](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#any-est-enrollment-over-secure-transport-ca) steps. In the script, set the CSR's subject to `/CN=$FLEET_VAR_HOST_UUID`.
2. Linux browsers don't read certificates from the filesystem. Append [`import-certificate-to-browsers.sh`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/linux/scripts/import-certificate-to-browsers.sh) to your script, updating the paths. It imports the certificate into each user's Chrome and Firefox stores, including snap installs. Hosts need `libnss3-tools` (Debian/Ubuntu) or `nss-tools` (RHEL). The script only imports into browser stores that exist. Run it again after a user first opens Firefox, and for users added later.

## Step 4: Configure the X.509 adapter

1. In PingFederate, import your CA's certificate as a trusted CA.
2. Create an X.509 Certificate IdP Adapter instance.
3. Set **Client Auth Port** to your secondary HTTPS port (`pf.secondary.https.port`). PingFederate requires it. Set **Client Auth Hostname** to the host name browsers use to reach PingFederate.
4. The adapter's `CN` attribute holds the certificate's common name, which is the host's UUID. You use it in Step 6.

## Step 5: Add Fleet as a data store

1. Go to **System > Data & Credential Stores > Data Stores** and select **Add New Data Store**.
2. For **Type**, select **Rest API**.
3. For **Base URL**, enter your Fleet server URL.
4. Add the Fleet API token from Step 1 as an `Authorization: Bearer <token>` header.
5. For **Test Connection URL**, enter `/api/v1/fleet/hosts/identifier/<a host's UUID>` and select **Test Connection**. The API user can call this endpoint.
6. Add these attributes:

| Local attribute | JSON response attribute path |
|---|---|
| `fleetHostID` | `/host/id` |
| `failingCriticalPolicies` | `/health/failing_critical_policies_count` |

## Step 6: Build the authentication policy

PingFederate looks up the host in Fleet in two calls. The first finds the host by the certificate's UUID. The second checks that host's critical policies.

1. Create an authentication policy contract. The default `subject` attribute is enough.
2. Add the X.509 adapter from Step 4 to your authentication policy. On its success path, map to the policy contract.
3. On **Attribute Sources & User Lookup**, add the Fleet data store twice:
   - First, with the resource path `/api/v1/fleet/hosts/identifier/${ad.<adapter ID>.CN}?exclude_software=true`
   - Second, with the resource path `/api/v1/fleet/hosts/${ds.<first source ID>.fleetHostID}/health`

   Replace `<adapter ID>` with the ID of the X.509 adapter instance (for example, `${ad.x509.CN}`) and `<first source ID>` with the ID you gave the first Fleet data source. `${hostUUID}` and `${fleetHostID}` don't resolve in a lookup path.
4. On **Contract Fulfillment**, map `subject` to the adapter's `CN`. PingFederate requires every contract attribute to have a value.
5. On **Issuance Criteria**, add `failingCriticalPolicies` equals `0`. This denies sign-in if the host is failing critical policies. It also denies sign-in if the host isn't in Fleet, because the second lookup then returns nothing.

Fleet updates a host's policy results each time the host runs its policies, about once an hour. So a user who fixes a failing policy is still blocked until the host rechecks. If they select **Refetch** on their **My device** page, results usually update within a few minutes, and their next sign-in works right away.

## Step 7: Test

| Host | Expected result |
|---|---|
| Managed, passing critical policies | Sign-in succeeds |
| Managed, failing a non-critical policy | Sign-in succeeds |
| Managed, failing a critical policy | Sign-in denied |
| No certificate | Sign-in denied |

## Troubleshooting

- **Browser doesn't prompt for a certificate**: Check that the certificate is in the user store (macOS login keychain, Windows `Cert:\CurrentUser\My`, or Linux NSS). On Linux, run `certutil -L -d sql:$HOME/.pki/nssdb`.
- **Browser asks for the login keychain password (macOS)**: Expected the first time Safari, Chrome or Firefox uses the certificate. The end user enters the password and selects **Always Allow**. If the prompt is refused, the browser sends no certificate and sign-in fails with "Authentication failed", the same as when there's no certificate. For Firefox, quit and reopen it before trying again, because it remembers the answer for the session. If you changed `AllowAllAppsAccess` after the certificate was issued, Fleet doesn't resend the profile to a host that already has a certificate. Resend it from the host's **Host details** page, the end user's **My device** page, or the [API](https://fleetdm.com/docs/rest-api/rest-api#resend-configuration-profile). Or wait for the certificate to renew.
- **Sign-in denied for a managed host**: Check that the certificate's CN matches the host's UUID in Fleet.
- **Sign-in still denied after fixing a policy**: Select **Refetch** on the **My device** page, wait for the policy results to update, then sign in again.
- **Data store errors**: Check the API token and that PingFederate can reach your Fleet server.

## Skip the certificate picker

- **Chrome and Edge**: Deploy the `AutoSelectCertificateForUrls` policy. Use the host name and port that ask for the certificate, not the sign-in URL's port. For example: `{"pattern":"https://ping.example.com:9032","filter":{"ISSUER":{"CN":"<your CA's common name>"}}}`. On macOS, deploy it in a configuration profile with the payload type `com.google.Chrome` (`com.microsoft.Edge` for Edge).
- **Firefox**: Set `security.default_personal_cert` to `Select Automatically` with an [enterprise policy](https://mozilla.github.io/policy-templates/). Firefox asks every time otherwise.
- **Safari**: Always asks. There's no policy to skip it.

<meta name="articleTitle" value="Conditional access: PingFederate">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-09-29">
<meta name="description" value="Use PingFederate for conditional access, blocking sign-in from hosts that aren't managed by Fleet or are failing critical policies.">
