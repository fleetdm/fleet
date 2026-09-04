# Set a naming convention for macOS, iOS, and iPadOS hosts

Fleet can automatically rename macOS, iOS, and iPadOS hosts as they enroll, so you don't have to rename each device by hand or script the [Fleet API](https://fleetdm.com/guides/set-device-hostname-via-fleet-api) for a whole fleet. Set a host name template once, and Fleet keeps every matching host's name in sync with it.

> **Note:** Host name templates are a Fleet Premium feature.


## Prerequisites

- A Fleet Premium license
- Global admin, team admin, or team maintainer access
- MDM turned on and configured for the hosts you want to rename
- The hosts enrolled in Fleet's MDM (macOS, iOS, and iPadOS only; other platforms aren't supported)


## Set the template

1. In the Fleet UI, select the team you want to apply the template to (or **No team**).
2. Head to **Controls > OS settings > Host names**.
3. In **Name template**, enter the name you want hosts to have. Combine fixed text with variables to make each host's name unique, for example: `iPad $FLEET_VAR_HOST_HARDWARE_SERIAL`.
4. Click **Save**.

> **Note:** If GitOps mode is turned on, the template is read-only in the Fleet UI. Set it in your GitOps YAML instead (see [Manage with GitOps](#manage-with-gitops) below).

Fleet applies the template the next time each matching host enrolls or checks in, and again whenever you change the template.


### Available variables

| Variable | Resolves to |
|----------|-------------|
| `$FLEET_VAR_HOST_HARDWARE_SERIAL` | The host's hardware serial number |
| `$FLEET_VAR_HOST_UUID` | The host's UUID |
| `$FLEET_VAR_HOST_PLATFORM` | The host's platform (macOS, iOS, or iPadOS) |
| `$FLEET_VAR_HOST_END_USER_IDP_USERNAME` | The end user's identity provider username |
| `$FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART` | The local part of the end user's IdP username (before the `@`) |
| `$FLEET_VAR_HOST_END_USER_IDP_GROUPS` | The end user's IdP groups |
| `$FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT` | The end user's IdP department |
| `$FLEET_VAR_HOST_END_USER_IDP_FULLNAME` | The end user's full name from the IdP |

You can also reference [secret variables](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles) and [custom host vitals](https://fleetdm.com/guides/custom-host-vitals) in a template.

> **Note:** A template can be at most 255 characters. Apple limits a device name to 63 bytes, so keep the fixed text short enough that the resolved name fits once variables expand. Fleet rejects a template whose fixed text alone already exceeds the limit.


## Verify the change

1. Go to **Hosts** and select a host the template applies to.
2. Open the host's **OS settings**.
3. Check the **Host name** row for its enforcement status:
   - **Enforcing**: Fleet has sent the rename command and is waiting for the host to check in.
   - **Verifying**: The host applied the command; Fleet is confirming the new name.
   - **Verified**: The host's name matches the template.
   - **Failed**: Fleet couldn't apply the template to this host (see Troubleshoot below).


## Troubleshoot

**A host shows a "Failed" status.** The name Fleet resolved for that host is likely longer than Apple's 63-byte device name limit, often because a variable (like an IdP username) expanded to more text than expected. Shorten the fixed text in your template, or check the value the variable resolved to for that host, then use **Resend** on the host's **Host name** row to try again.


## Manage with GitOps

If you manage controls with GitOps, set `name_template` under `controls` for a team (or under `controls` at the top level for **No team**):

```yaml
controls:
  name_template: "iPad $FLEET_VAR_HOST_HARDWARE_SERIAL"
```

An empty string clears the template without renaming any hosts.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="authorFullName" value="Kitzy">
<meta name="publishedOn" value="2026-08-29">
<meta name="articleTitle" value="Set a naming convention for macOS, iOS, and iPadOS hosts">
<meta name="description" value="Use Fleet's host name template to automatically rename macOS, iOS, and iPadOS hosts as they enroll.">
