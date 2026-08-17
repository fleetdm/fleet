# Automatically resend configuration profiles

Fleet can automatically resend a configuration profile to a host when it fails a policy check. This is useful for re-enforcing settings like certificate renewal, Wi-Fi configuration, Santa daemon health, or CIS Benchmark compliance.

## Prerequisites

- MDM turned on for the fleet and hosts enrolled.
- The configuration profile must already be added to the fleet's **Controls** > **OS settings** > **Configuration profiles** > **Profiles** and assigned to the host.

## Step-by-step instructions

1. **Add a configuration profile**: Navigate to **Controls** > **OS settings** > **Configuration profiles** > **Profiles**, select the fleet, and add the configuration profile you want to resend. Learn how in the [custom OS settings guide](https://fleetdm.com/guides/custom-os-settings).

2. **Add a policy**: Navigate to **Policies**, select the fleet, and click **Add policy**. Write a policy query that fails when the configuration needs to be re-enforced and click **Save**. For example, a policy that checks if FileVault is enabled:

```sql
SELECT 1 FROM disk_encryption WHERE encrypted = 1;
```

3. **Set the automation**: In the **Save policy** modal, click **Add automations**, then click the checkbox next to **Resend configuration profile**. From the dropdown, select the profile you want to resend. Click **Save**.

When a host fails the selected policy, Fleet will resend the configuration profile to the host.

If you need to retrigger the automation on hosts that had previously failed, deselect the policy in the **Policies > Manage automations** modal, click Save, and then reselect the policy. This will reset the policy's host passing and failing host counts and retrigger the resend.

## How does it work?
- Online hosts report policy status on a configurable cadence, with hourly default.
- Fleet will resend the configuration profile on the first policy failure or if a policy goes from "Pass" to "Fail". By default, policies that remain failing for a host in consecutive reports will not trigger a resend.
- If the profile is already pending or verifying delivery, Fleet will skip the resend to avoid interrupting an in-flight delivery.
- If the profile is not assigned to the host, Fleet will skip the resend (no error is raised).

> When the configuration profile automation on a policy is added or changed, the policy's status will reset for associated hosts. This allows the resend to trigger on hosts that had previously failed the policy.

## Via the API
Configuration profile policy automation can be managed by setting the `profile_uuid` field on the Fleet REST API's [Add team policy](https://fleetdm.com/docs/rest-api/rest-api#add-team-policy) or [Edit team policy](https://fleetdm.com/docs/rest-api/rest-api#edit-team-policy) endpoints.

## Via GitOps
To configure configuration profile policy automation via GitOps, nest a `resend_configuration_profile` entry under the policy you want to automate, using the `name` of a profile defined in the same fleet's `controls` section. See the GitOps reference documentation for an example.

```yaml
policies:
  - name: "macOS - FileVault enabled"
    query: "SELECT 1 FROM disk_encryption WHERE encrypted = 1;"
    platform: darwin
    resend_configuration_profile:
      name: "Passcode requirements"
```

<meta name="articleTitle" value="Automatically resend configuration profiles">
<meta name="authorFullName" value="Fleet">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-08-13">
<meta name="description" value="A guide to automatically resending configuration profiles when hosts fail a policy in Fleet.">
