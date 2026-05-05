# Labels


In Fleet, you can use labels to scope [software](https://fleetdm.com/guides/deploy-software-packages), [policies](https://fleetdm.com/securing/what-are-fleet-policies), [queries](https://fleetdm.com/guides/queries), and [configuration profiles](https://fleetdm.com/guides/custom-os-settings) for specific hosts, and filter the hosts view.

Labels can be one of the following types:
- **Dynamic**: A query-based label applied to any host that returns a result for the label's query.
> If you want to change the query or platform on a dynamic label, you must delete the existing label and create a new one.
- **Manual**: A manually assigned label used to filter selected hosts.
- **Host vitals**: A Fleet-generated label applied to hosts that match a specific host vital (currently IdP group and department on macOS, iOS, iPadOS, and Android).
> If you want to change the target of a host vitals label, you must delete the existing label and create a new one.

To add or edit a label in Fleet, select the avatar on the right side of the top navigation and select **Labels**.

You can also manage labels via [Fleet's API](https://fleetdm.com/docs/rest-api/rest-api#labels) or [best practice GitOps](https://fleetdm.com/docs/configuration/yaml-files#labels).

## Target configuration profiles with labels

_Available in Fleet Premium._

You can use labels to control which hosts receive a [configuration profile](https://fleetdm.com/guides/custom-os-settings). Fleet supports three targeting options:

- **Include all**: Only hosts that have **all** specified labels receive the profile (`labels_include_all`).
- **Include any**: Hosts that have **any** of the specified labels receive the profile (`labels_include_any`).
- **Exclude any**: Hosts that have **any** of the specified labels are excluded from receiving the profile (`labels_exclude_any`).

### Combining include and exclude

You can combine `labels_exclude_any` with either `labels_include_all` or `labels_include_any` on the same profile. This lets you include a broad set of hosts and then carve out exceptions without writing a complex label query.

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

If no label targeting is specified, the profile is delivered to all hosts on the specified platform.

You can also set label targets through the Fleet UI when adding or editing a configuration profile under **Controls > OS settings > Configuration profiles**, or via the [REST API](https://fleetdm.com/docs/rest-api/rest-api#create-configuration-profile).

## Activity logging

Fleet logs an activity when a label is added to or removed from a host. These activities appear in the host's activity feed:

- **added_label_to_host**: Generated when a label is added to a host.
- **removed_label_from_host**: Generated when a label is removed from a host.

Each activity includes the host ID, host display name, label ID, and label name. For more details, see [Audit logs](https://fleetdm.com/docs/contributing/reference/audit-logs).


<meta name="articleTitle" value="Labels in Fleet">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-10-24">
<meta name="articleImageUrl" value="../website/assets/images/articles/managing-labels-in-fleet-1600x900@2x.png">
<meta name="description" value="Using labels in the Fleet">
