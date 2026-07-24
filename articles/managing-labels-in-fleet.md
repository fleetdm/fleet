# Labels

In Fleet, labels organize hosts into groups you can target with [software](https://fleetdm.com/guides/deploy-software-packages), [policies](https://fleetdm.com/securing/what-are-fleet-policies), [reports](https://fleetdm.com/guides/queries), and [configuration profiles](https://fleetdm.com/guides/custom-os-settings). You can also use labels to filter the hosts view.

> We recommend labels, rather than separate fleets, as your primary way to target these features.

## Label types

- **Dynamic:** Query-based; auto-applied to any host returning a result for the label's SQL query. Optionally restrict to a platform (`darwin`, `windows`, `ubuntu`, `centos`).
- **Manual:** Applied to an explicit list of hosts, specified by `hardware_serial`, `uuid`, or Fleet host ID. Useful for one-off groupings (e.g., a pilot group).
- **Host vitals:** Auto-applied to hosts matching a host vital from your IdP. Supported criteria: `end_user_idp_group` and `end_user_idp_department`. Requires a connected IdP (Okta, Microsoft Entra ID, Google Workspace, authentik, or any SCIM provider; see [Foreign host vitals](https://fleetdm.com/guides/foreign-vitals-map-idp-users-to-hosts)).

> To change a dynamic label's query/platform or a host vitals label's criteria in the UI, you must delete and re-create it.

## Targeting with labels

Labels can target or exclude hosts using one scoping mode per item. Configuration Profiles support custom targeting via "Include any" and "Exclude any":

| Scope | Behavior | Available for |
| --- | --- | --- |
| **Include any** | Targets hosts with **any** of the labels | Software, policies, reports, configuration profiles |
| **Include all** | Targets hosts with **all** of the labels | Software, policies, reports, configuration profiles |
| **Exclude any** | Excludes hosts with **any** of the labels | Software, policies, configuration profiles |

## Label scope: global vs. fleet

A label's scope is set based on where it's created, not by its name:

- **Global:** Available across all fleets. Created by a global user in the UI, or defined in `default.yml`.
- **Fleet:** (Fleet Premium) Scoped to a single fleet and visible only alongside global labels for that fleet. Defined in that fleet's `fleets/fleet-name.yml`. Defining a label here scopes it to the fleet; it does **not** become global.

> **Tip:** Label names share one namespace, so creating a label whose name already exists (global or fleet) will fail. If multiple teams manage labels independently, prefix them to avoid collisions—either **by owner/fleet** (e.g. `[Workstations] Kiosk`, `ws-kiosk`) or by **centralizing all labels** in one place (e.g. a `labels/` directory referenced from `default.yml`) as the single source of truth, so collisions surface in a single PR.

## Managing labels

## Targeting with labels

_Available in Fleet Premium._

Fleet supports three targeting options:

- **Include all**: Only hosts that have **all** specified labels receive the profile (`labels_include_all`).
- **Include any**: Hosts that have **any** of the specified labels receive the profile (`labels_include_any`).
- **Exclude any**: Hosts that have **any** of the specified labels are excluded from receiving the profile (`labels_exclude_any`).

### Combining include and exclude

You can combine `labels_exclude_any` with either `labels_include_all` or `labels_include_any` on the same configuration profile. This lets you include a broad set of hosts and then carve out exceptions without writing a complex label query.

> `labels_include_all` and `labels_include_any` cannot be combined with each other on the same profile.

For example, to deliver a profile to all hosts in the "Engineering" or "Product" labels but skip hosts in the "Macs on Sequoia" label:

```yaml
controls:
  apple_settings:
    configuration_profiles:
      - path: ../lib/macos-profile.mobileconfig
        labels_include_any:
          - Engineering
          - Product
        labels_exclude_any:
          - Macs on Sequoia
```

Or, to deliver a profile only to hosts that have **both** the "Sonoma" and "Managed" labels while excluding hosts labeled "Contractors":

```yaml
controls:
  apple_settings:
    configuration_profiles:
      - path: ../lib/macos-profile.mobileconfig
        labels_include_all:
          - Sonoma
          - Managed
        labels_exclude_any:
          - Contractors
```

> Combining include and exclude labels is only supported for configuration profiles as of Fleet version 4.88.0.

## Label scope: global vs. fleet

A label's scope is set based on where it's created, not by its name:

- **Global:** Available across all fleets. Created by a global user in the UI, or defined in `default.yml`.
- **Fleet:** (Fleet Premium) Scoped to a single fleet and visible only alongside global labels for that fleet. Defined in that fleet's `fleets/fleet-name.yml`. Defining a label here scopes it to the fleet; it does **not** become global.

> **Tip:** Label names share one namespace, so creating a label whose name already exists (global or fleet) will fail. If multiple teams manage labels independently, prefix them to avoid collisions—either **by owner/fleet** (e.g. `[Workstations] Kiosk`, `ws-kiosk`) or by **centralizing all labels** in one place (e.g. a `labels/` directory referenced from `default.yml`) as the single source of truth, so collisions surface in a single PR.

If no label targeting is specified, the profile is delivered to all hosts on the specified platform.

## Managing labels

To add or edit a label in Fleet, select the avatar on the right side of the top navigation and select **Labels**.

You can also manage labels via [Fleet's API](https://fleetdm.com/docs/rest-api/rest-api#labels) or [best practice GitOps](https://fleetdm.com/docs/configuration/yaml-files#labels).


<meta name="articleTitle" value="Labels in Fleet">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-10-24">
<meta name="articleImageUrl" value="../website/assets/images/articles/managing-labels-in-fleet-1600x900@2x.png">
<meta name="description" value="Using labels in the Fleet">
