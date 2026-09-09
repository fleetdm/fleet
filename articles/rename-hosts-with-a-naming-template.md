# Rename hosts with a naming template

_Available in Fleet Premium_

Set a naming convention once and Fleet renames every macOS, iOS, and iPadOS host in a fleet to match, both on the device and in Fleet. Instead of building an automation to send a custom MDM command to each host, you save a name template like `iPad $FLEET_VAR_HOST_HARDWARE_SERIAL` and Fleet resolves it per host, renames the device over MDM, and keeps its own record in sync.

This applies to Apple hosts (macOS, iOS, iPadOS) only. Windows and Android hosts are unaffected.

## Prerequisites

- Fleet Premium.
- Fleet's MDM [turned on](https://fleetdm.com/guides/macos-mdm-setup).
- Hosts enrolled in Fleet's MDM. Personally enrolled (BYOD) hosts are skipped and never renamed.
- iOS and iPadOS hosts must be supervised. Apple only applies a name change to supervised iPhones and iPads; unsupervised hosts receive the command once and land on **Failed**.

## Set a name template

1. In the top navigation, select **Controls**, then select a fleet (or **Unassigned** for hosts that aren't in a fleet).
2. Select **OS settings**, then **Host names**.
3. In **Name template**, enter your naming convention. Use plain text, built-in variables, custom variables, or a combination. For example: `Conference Room iPad $FLEET_VAR_HOST_HARDWARE_SERIAL`.
4. Select **Save**.

Fleet queues a rename for every eligible host in the fleet. The name you set becomes the host's name in Fleet and on the device itself.

> **Note:** Clearing the **Name template** field and saving stops enforcement but doesn't rename any host. Hosts keep their current name.

### Built-in variables

Use these variables in a template to give each host a unique name:

| Variable | Resolves to |
|---|---|
| `$FLEET_VAR_HOST_HARDWARE_SERIAL` | The host's hardware serial number. |
| `$FLEET_VAR_HOST_UUID` | The host's UUID. |
| `$FLEET_VAR_HOST_PLATFORM` | The host's platform: `macOS`, `iOS`, or `iPadOS`. |
| `$FLEET_VAR_HOST_END_USER_IDP_USERNAME` | The host end user's identity provider (IdP) username. |
| `$FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART` | The local part of the IdP username (before `@`). |
| `$FLEET_VAR_HOST_END_USER_IDP_GROUPS` | The end user's IdP groups, comma-separated. |
| `$FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT` | The end user's IdP department. |
| `$FLEET_VAR_HOST_END_USER_IDP_FULL_NAME` | The end user's IdP full name. |

Each variable also works in its `${FLEET_VAR_...}` form. For more on built-in variables, see [Built-in variables](https://fleetdm.com/guides/fleet-variables).

> **Note:** The IdP variables need the host to have end-user IdP data. If a host has no IdP user, or the field the template references is empty (for example, no department), that host lands on **Failed** — the same behavior as configuration profiles that use these variables. The identity variables (serial, UUID, platform) are always available.

> **Note:** A resolved host name can't be longer than 63 bytes (Apple's device name limit). Hosts whose resolved name exceeds this land on **Failed**.

### Custom variables

You can also use custom (`$FLEET_SECRET_*`) variables in a template, for example `$FLEET_SECRET_SITE-$FLEET_VAR_HOST_HARDWARE_SERIAL`. Custom variables are global, so a variable resolves to the same value for every fleet and host. See [Custom variables](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles).

The custom variable must already exist when you save the template, or the save fails. A custom variable used in a name template can't be deleted until you remove it from the template.

> **Important:** Unlike in scripts and configuration profiles, a custom variable used in a name template isn't kept hidden. Its value becomes the host's name in Fleet and on the device, so only use custom variables for values that are safe to display (for example, a site or location code), not for secrets.

### Variables that aren't supported

Certificate authority variables — for example `$FLEET_VAR_NDES_SCEP_CHALLENGE`, `$FLEET_VAR_DIGICERT_DATA_<CA>`, `$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA>`, `$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID`, and `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` — and the Platform SSO device registration token can't be used in a name template. They resolve to one-time challenges, proxy URLs, or certificate data: values that are meaningless as a device name and would expose secrets in a name that's visible on the device, in reports, and in the Fleet UI. Fleet rejects a template that references them.

The deprecated `$FLEET_VAR_HOST_END_USER_EMAIL_IDP` variable also isn't supported; use the `HOST_END_USER_IDP_*` variables above instead.

### Renaming when a variable's value changes

Fleet keeps host names in sync with their variables. When a host's IdP data changes (for example, the end user's department is updated) or a custom variable's value changes, Fleet resolves the template again and renames the hosts that use that variable.

## Set a name template with GitOps

Add `name_template` under `controls` in a fleet's YAML, or in `no_team.yml` or `default.yml` controls to apply it to "Unassigned" hosts:

```yaml
controls:
  name_template: "iPad $FLEET_VAR_HOST_HARDWARE_SERIAL" # Available in Fleet Premium
```

You can reference a custom (`$FLEET_SECRET_*`) variable in the template too:

```yaml
controls:
  name_template: "iPad $FLEET_SECRET_SITE" # Available in Fleet Premium
```

Removing the key clears the template. For all controls options, see the [YAML files reference](https://fleetdm.com/docs/configuration/yaml-files#controls).

## Verify

Open a host's **OS settings** to see its host name status:

1. Select **Hosts**, then select a host.
2. Select **Actions > Show details**, then open the **OS settings** modal.
3. Find the **Host name** row. Its status moves from **Enforcing** to **Verifying** (the device applied the name) to **Verified** (Fleet confirmed the name from the device).

Controls > OS settings also rolls host name statuses into the **Verified**, **Verifying**, **Pending**, and **Failed** aggregate cards.

## Troubleshoot

**A host's Host name row shows Failed.** The status is Failed when the device rejected the command, the resolved name was too long, the host is missing IdP data a variable in the template needs, a custom variable in the template is no longer defined, or an end user renamed the device off-template. The row's tooltip shows the error. Select **Resend** on the row to try again.

**An iPhone or iPad shows Failed with a supervision error.** Apple only applies MDM name changes to supervised iOS and iPadOS hosts. Supervise the host (for example, by enrolling it through Apple Business Manager), then select **Resend**.

**A host has no Host name row.** Fleet omits the row for hosts it doesn't enforce: hosts whose fleet (or "Unassigned") has no template, non-MDM hosts, and personally enrolled (BYOD) hosts.

## Further reading

- [Built-in variables](https://fleetdm.com/guides/fleet-variables)
- [Custom variables](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles)
- [YAML files reference](https://fleetdm.com/docs/configuration/yaml-files#controls)
- [Update host name template API](https://fleetdm.com/docs/rest-api/rest-api#update-host-name-template)

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="juan-fdz-hawa">
<meta name="authorFullName" value="Juan Fernandez">
<meta name="publishedOn" value="2026-07-14">
<meta name="articleTitle" value="Rename hosts with a naming template">
<meta name="description" value="Set a naming convention to rename macOS, iOS, and iPadOS hosts in Fleet and on the device with a name template.">
