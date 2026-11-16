# Roll out changes with deployment rings

A new app version, configuration profile, or OS update can break things for everyone at once. Deployment rings limit that risk: a change reaches a small group first, then larger groups, and you only move to the next ring once the last one looks healthy. Fleet doesn't have a built-in ring feature, but labels and label targeting give you the same result. This guide shows two ways to build rings, how to scope software, profiles, and OS updates to them, and how to promote a change from one ring to the next.

## Choose how to build rings

Fleet supports two ring designs. Both put each host in exactly one ring, and both use the same promotion steps, so you can switch later without changing your workflow.

- IdP groups (recommended): Each ring is a group in your identity provider (IdP), like Okta or Microsoft Entra ID. You decide which people are in which ring, so your early adopters are people who've agreed to test changes and know how to report problems. Rings follow the person to every host they use, including iPhones, iPads, and Android devices.
- Rollout waves: Each ring is a dynamic label that assigns hosts to a wave based on their hardware UUID. The team that manages Fleet controls the rings end to end, with no IdP changes. Use this when you can't create or manage IdP groups, or when most of your hosts have no end user assigned (servers, kiosks, shared devices).

## Prerequisites

- Fleet Premium.
- Admin or maintainer access to the fleet you're rolling out to.
- For IdP rings: your IdP connected to Fleet, with groups sent to Fleet and end users mapped to hosts. Learn how in [Foreign vitals: map IdP users to hosts](https://fleetdm.com/guides/foreign-vitals-map-idp-users-to-hosts).
- For rollout waves: hosts running Fleet's agent (fleetd). Waves rely on a query, so they don't cover iOS, iPadOS, or Android hosts.

This guide shows GitOps YAML for each step. Everything also works in the Fleet UI: create the labels under **Labels**, and set targets in each item's **Target** settings.

## Option 1: Build rings from IdP groups

### Create the ring groups in your IdP

1. In your IdP, create one group per ring. For example:
   - `Fleet ring 1 - IT`
   - `Fleet ring 2 - Early adopters`
   - `Fleet ring 3 - Pilot`
2. Add people to the groups. Put each person in only one ring group.
3. Send the groups to Fleet. In Okta, add them under **Push Groups** on your Fleet SCIM app. In Entra ID, assign them to your Fleet SCIM app.

You don't need a group for the final ring. The last step of every rollout targets all hosts, which covers everyone who isn't in a ring group.

> **Note:** Hosts without an IdP user aren't in any ring. They get the change at the final step, along with everyone else. If you have many of these hosts, use [rollout waves](#option-2-build-rings-from-rollout-waves) instead, or add them to a [pinned hosts](#optional-pin-specific-hosts-into-the-first-ring) label.

### Create a label for each ring

Create a host vitals label for each ring group. Fleet adds a host to the label when the host's end user is a member of the group.

`lib/labels/deployment-rings.labels.yml`:

```yaml
- name: Ring 1 - IT
  description: Hosts whose end user is in the "Fleet ring 1 - IT" IdP group.
  label_membership_type: host_vitals
  criteria:
    vital: end_user_idp_group
    value: Fleet ring 1 - IT
- name: Ring 2 - Early adopters
  description: Hosts whose end user is in the "Fleet ring 2 - Early adopters" IdP group.
  label_membership_type: host_vitals
  criteria:
    vital: end_user_idp_group
    value: Fleet ring 2 - Early adopters
- name: Ring 3 - Pilot
  description: Hosts whose end user is in the "Fleet ring 3 - Pilot" IdP group.
  label_membership_type: host_vitals
  criteria:
    vital: end_user_idp_group
    value: Fleet ring 3 - Pilot
```

Reference the file from `default.yml` and run `fleetctl gitops`:

```yaml
labels:
  - path: ./lib/labels/deployment-rings.labels.yml
```

In the UI, go to **Labels > Add label**, select **Host vitals**, choose **IdP group**, and enter the group name.

> **Warning:** The `value` must match the IdP group name exactly. If someone renames the group in your IdP, the label empties and the ring stops receiving changes. Update the label's `value` whenever a ring group is renamed.

## Option 2: Build rings from rollout waves

Rollout waves split hosts into five waves using two characters of each host's hardware UUID. A host's wave never changes, and new hosts land in a wave shortly after they enroll.

| Wave | Share of hosts |
| --- | --- |
| Wave 1 - Canary (0-1%) | ~1% |
| Wave 2 - Early (1-5%) | ~4% |
| Wave 3 - Pilot (5-25%) | ~20% |
| Wave 4 - Broad (25-75%) | ~50% |
| Wave 5 - Tail (75-100%) | ~25% |

1. Download [`rollout-waves.labels.yml`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/labels/rollout-waves.labels.yml) and save it to `lib/labels/rollout-waves.labels.yml` in your GitOps repository.
2. Reference it from `default.yml`:

   ```yaml
   labels:
     - path: ./lib/labels/rollout-waves.labels.yml
   ```

3. Run `fleetctl gitops`.
4. Wait about an hour before you target the waves. Fleet adds hosts to dynamic labels as hosts check in, so a brand-new wave looks smaller than it is.

To create the waves in the UI, go to **Labels > Add label**, select **Dynamic**, and copy each wave's name and query from the file.

> **Note:** Hosts that share a hardware UUID, like VMs cloned from a template without a new UUID, all land in the same wave. If one wave is much bigger than its expected share, look for duplicate UUIDs.

To change the wave sizes, edit the thresholds in each query. The waves cover the values 0-255, so a threshold of `< 3` is about 1% of hosts and `< 64` is about 25%.

## Optional: Pin specific hosts into the first ring

Some hosts should always get changes first, like your own test Mac or a QA VM. Add them to a manual label and list it alongside your first ring.

1. Download [`pinned-canaries.labels.yml`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/labels/pinned-canaries.labels.yml) and save it to `lib/labels/pinned-canaries.labels.yml`.
2. Replace the example `hosts` with your hosts' serial numbers, UUIDs, or Fleet host IDs.
3. Reference the file from `default.yml` next to your ring labels.

In the UI, go to **Labels > Add label**, select **Manual**, and add the hosts.

## Scope a change to the first ring

Put the ring labels on whatever delivers the change. Add only the first ring (and `Pinned canaries`, if you use it) to start.

The examples below use IdP ring names. For rollout waves, use the wave names instead, like `Wave 1 - Canary (0-1%)`.

### New software

For software that isn't on your hosts yet, target the software itself. Policies that install the software follow its targets, so you don't need to target the policy too.

`fleets/workstations.yml`:

```yaml
software:
  packages:
    - url: https://example.com/downloads/example-app-2.0.0.pkg
      labels_include_any:
        - Ring 1 - IT
        - Pinned canaries
```

### A new version of a custom package

Add the new version to the same software, next to the current version. Target the new version to the rings and exclude the rings from the current version, so each host matches exactly one package.

`lib/software/example-app.package.yml`:

```yaml
# Current version. Keep it first in the list.
- url: https://example.com/downloads/example-app-1.9.0.pkg
  labels_exclude_any:
    - Ring 1 - IT
    - Pinned canaries
# New version
- url: https://example.com/downloads/example-app-2.0.0.pkg
  labels_include_any:
    - Ring 1 - IT
    - Pinned canaries
```

If a policy installs this software automatically, point it at the new package with `install_software.hash_sha256`. Hosts outside the new package's targets don't get the install.

> **Note:** Software can use only one of `labels_include_any`, `labels_include_all`, or `labels_exclude_any`. That's why the current version uses an exclude list that mirrors the new version's include list. Keep the two lists in sync at every step.

### A new version of a Fleet-maintained app

Fleet-maintained apps update to the latest version every time GitOps runs, unless you pin a `version`. Pin the version, then target the patch policy that installs updates.

`fleets/workstations.yml`:

```yaml
software:
  fleet_maintained_apps:
    - slug: slack/darwin
      version: "4.47.65"

policies:
  - name: macOS - Slack up to date
    type: patch
    fleet_maintained_app_slug: slack/darwin
    install_software: true
    labels_include_any:
      - Ring 1 - IT
      - Pinned canaries
```

> **Note:** Hosts outside the rings don't get automatic updates, but end users can still install the new version from **Fleet Desktop > Self-service**.

### Configuration profiles and OS updates

Target the profile. This works for configuration profiles, DDM declarations, and Windows CSP profiles.

Fleet's built-in OS update settings in **Controls > OS updates** apply to every host in a fleet, so they can't follow rings. To roll out an OS update in rings, deploy your own update profile instead: a `softwareupdate.enforcement.specific` declaration for macOS, iOS, and iPadOS, or an Update CSP profile for Windows. Learn more in [Enforce macOS updates per major version](https://fleetdm.com/guides/enforce-macos-updates-per-major-version).

> **Warning:** Don't combine a custom Apple OS update declaration with the minimum version set in **Controls > OS updates**. Both send a `softwareupdate.enforcement.specific` declaration and conflict on the host. Clear the built-in setting for that platform first.

`fleets/workstations.yml`:

```yaml
controls:
  apple_settings:
    configuration_profiles:
      - path: ../lib/macos/declarations/macos-26-update.json
        labels_include_any:
          - Ring 1 - IT
          - Pinned canaries
```

`lib/macos/declarations/macos-26-update.json`:

```json
{
  "Type": "com.apple.configuration.softwareupdate.enforcement.specific",
  "Identifier": "com.example.softwareupdate.macos26",
  "Payload": {
    "TargetOSVersion": "26.1",
    "TargetLocalDateTime": "2026-12-15T17:00:00"
  }
}
```

Run `fleetctl gitops` to start the rollout.

## Promote to the next ring

When the current ring looks healthy, add the next ring to every list you set up for this rollout. Keep the earlier rings in the list. Each host is in only one ring, so removing a ring would stop the rollout for those hosts.

```yaml
      labels_include_any:
        - Ring 1 - IT
        - Pinned canaries
        - Ring 2 - Early adopters
```

For a new version of a custom package, add the ring to both lists: the new version's `labels_include_any` and the current version's `labels_exclude_any`.

Run `fleetctl gitops`. In the UI, select the next ring in the item's **Target** settings and save.

How long you wait between rings is up to you. A few days per ring is common for apps, and a week or more for OS updates. Fleet doesn't promote on its own, so nothing moves to the next ring until someone makes the change.

## Finish the rollout

After the last ring, roll out to everyone, including hosts that aren't in any ring:

- New software, profiles, and patch policies: Remove `labels_include_any`. In the UI, set **Target** to **All hosts**.
- A new version of a custom package: Remove the old version's entry from the package YAML, then remove `labels_include_any` from the new version.

For the next release, start again from the first ring.

## Pause or roll back

- Pause: Stop adding rings. Hosts that already have the change keep it, and no new rings get it.
- Roll back a ring: Remove the most recent ring from the list. Fleet removes a profile from hosts that are no longer targeted. Software that already installed stays installed, so uninstall it or install the previous version on those hosts.
- Skip one host: Cancel the pending install on the host's **Host details > Activity > Upcoming** tab. For profiles and policies, add a manual label with that host to `labels_exclude_any`.

> **Note:** Removing an OS update declaration doesn't undo an update that already installed.

## Verify

1. Go to **Hosts** and filter by a ring label. Check that the host count is what you expect.
2. For software, open the software's page and check the **Installed** and **Failed** counts.
3. For profiles, go to **Controls > OS settings** and check the profile's **Verified** and **Failed** counts.
4. Before you promote, check that hosts in the next ring don't have the change yet.

## Troubleshoot

**A ring label has no hosts**

For IdP rings, check that the group name in the label matches the IdP group exactly and that your IdP is sending the group to Fleet. Open a host in the ring and check **Groups (IdP)** on its **Host details** page. For rollout waves, wait an hour after you create the labels.

**A host is in a ring group but doesn't get the change**

The host may not have an IdP user. Check **Host details** for the host's IdP username, and add it if it's missing.

**A host got both package versions, or neither**

The new version's include list and the current version's exclude list don't match. Make them identical. If a host matches both packages, Fleet installs the one added first.

## Further reading

- [Labels in Fleet](https://fleetdm.com/guides/managing-labels-in-fleet)
- [Foreign vitals: map IdP users to hosts](https://fleetdm.com/guides/foreign-vitals-map-idp-users-to-hosts)
- [Deploy software packages](https://fleetdm.com/guides/deploy-software-packages)
- [Automatic software install in Fleet](https://fleetdm.com/guides/automatic-software-install-in-fleet)
- [GitOps reference: labels](https://fleetdm.com/docs/configuration/yaml-files#labels)

<meta name="articleTitle" value="Roll out changes with deployment rings">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-11-16">
<meta name="category" value="guides">
<meta name="description" value="Roll out software, profiles, and OS updates in rings with IdP group or rollout wave labels, and promote changes one ring at a time.">
