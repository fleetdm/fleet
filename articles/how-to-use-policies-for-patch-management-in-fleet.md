# How to use policies for patch management in Fleet

![How to use policies for patch management in Fleet](../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png)

Policies in Fleet enable IT admins to report on devices and get quick yes or no answers about the status of their endpoints. Powered by the flexibility of osquery, the policies engine has become an invaluable part of the IT toolkit, simplifying the management of devices at scale. This guide covers two approaches: **patch policies**, which automate everything for Fleet-maintained apps, and **manual policies** for custom packages.

Initially, Fleet’s policies allowed for automated responses, like firing a webhook on a policy failure or creating a ticket in your ITSM system. While effective, these actions were limited by the capabilities of your existing tools to process and act on these notifications.

## Enter patch management

Fleet’s policy capabilities have evolved beyond notification-based responses. With the release of Fleet v4.57, the policies engine now supports a game-changing feature: automated software installation on a policy failure. This addition transforms the policies engine into a dynamic tool for streamlined patch management.

With Fleet v4.83, patch management gets even simpler with **patch policies** for [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps). Patch policies eliminate the need to write osquery queries — Fleet auto-generates the correct query for each app. When managed via GitOps, the query automatically updates to include the latest version each time specs are applied.

In this article, we’ll explore how to leverage these features to automate patching across your environment. This will free up valuable IT resources to focus on high-impact tasks while enhancing end-user support.

## Why it matters

Around [60% of data breaches in 2023](https://www.automox.com/blog/bad-cyber-hygiene-breaches-tied-to-unpatched-vulnerabilities) involved vulnerabilities for which patches were available but not applied, underscoring the impact of delayed patch management.

Regular updates often include bug fixes that improve stability and enhance user experience, allowing employees to work without disruptions. These updates also make sure compatibility with other applications, preventing integration issues that could impact workflows. 

Additionally, updated software often includes new features that can ultimately help teams work more efficiently and effectively.

## Patch policies for Fleet-maintained apps

_Available in Fleet Premium_

A patch policy automatically checks whether a Fleet-maintained app is up to date on your hosts. Unlike manual policies, you don’t need to write or update osquery queries — Fleet handles it for you.

Key benefits:
- **Automatic query generation** — Fleet creates the correct query for the app and platform.
- **Fail only if outdated** — The policy only fails if the app IS installed AND running an older version. Hosts without the app installed pass the policy.

### In the Fleet UI

1. Navigate to **Software** and select your fleet.
2. Click on a Fleet-maintained app to open its details.
3. From the **Actions** dropdown, select **Patch**.
4. Click **Add** in the confirmation modal.

To automatically install updates when the policy fails, navigate to **Policies > Manage automations > Install software** and enable the automation for the new patch policy.

### Via GitOps

Add a policy with `type: patch` and specify the `fleet_maintained_app_slug`. With GitOps, the patch policy query automatically updates to include the latest version each time specs are applied:

```yaml
policies:
  - name: Zoom up to date
    type: patch
    fleet_maintained_app_slug: zoom/darwin
```

For all available options, see the [GitOps reference documentation](https://fleetdm.com/docs/configuration/yaml-files#patch-policy).

### Via the API

Set `type` to `"patch"` and provide `patch_software_title_id` when [adding a fleet policy](https://fleetdm.com/docs/rest-api/rest-api#create-fleet-policy).

## Manual policies for custom packages

If you’re deploying custom software packages (not Fleet-maintained apps), you can write your own policy query and pair it with install automation.

In this example, we will be using Google Chrome to demonstrate the functionality, and I already have the latest version’s .pkg downloaded locally.

Select the fleet you want the policy to run on. Navigate to **Software > Add Software**. Here you can use one of Fleet’s maintained apps, add from VPP or Custom Package. We will use Custom Package in this example and upload the Google Chrome.pkg mentioned previously. After upload, there are a couple of options for pre/post-install queries and scripts - you can read more about those options in our [guide on deploying software](https://fleetdm.com/guides/deploy-software-packages).

Navigate to **Policies**, select the fleet you want the policy to run in.

Assume we want to bring all devices in this fleet to the latest version of Chrome we uploaded to Fleet, which as of writing this, is 130.0.6723.70.

Your policy query would look something like this:

```sh
SELECT 1 FROM apps WHERE bundle_identifier = 'com.google.Chrome' AND bundle_short_version < '130.0.6723.70'
```

This means any evaluation of this policy, where the version is less than 130.0.6723.70, will result in a failure and thus kick off the automation.

Save and give it an intuitive name, we recommend something like:

_macOS - Update Google Chrome to Latest_

With the policy set, we can tie in the automation. Back on the main policy page, select **Manage automations > Install software**.

The module will show the policies available for that fleet. Check the box to turn on automation, and from the dropdown, select the software that will be installed on the failure. The dropdown will show all available software in that fleet - plus its supported OS and version.

And that’s it! Policies are evaluated across all online hosts every hour, or when a device is refetched manually. Any machine that fails this policy will install the Chrome version that was set in the policy.

## When to use each approach

| | Patch policies | Manual policies |
| --- | --- | --- |
| **Best for** | Fleet-maintained apps | Custom packages, VPP apps |
| **Query management** | Automatic | You write and maintain the query |
| **Version updates** | Automatic with GitOps; re-create via UI for new versions | Manual |
| **Behavior when app is missing** | Policy passes | Depends on your query |
| **Platforms** | macOS, Windows | macOS, Windows, Linux |

## What else can we do?

This functionality unlocks many use cases for an IT admin to help manage their fleet. Another use case for this feature is to support a zero-touch deployment of devices and ensure that critical business and productivity software is installed from the first boot.

A simple query like such: 

```sh
SELECT 1 FROM apps WHERE bundle_identifier = ‘com.tinyspeck.slackmacgap’
```

would deploy Slack to your endpoints the moment it comes out of the box, ensuring your users are ready to hit the ground running from day 1.

> If the app is available as a Fleet-maintained app (like Slack), you can also add a [patch policy](#patch-policies-for-fleet-maintained-apps) to keep it updated automatically — no query maintenance required.

## Via the API

Fleet Premium customers can leverage the REST API for both approaches:

- **Patch policies**: Set `type` to `"patch"` with `patch_software_title_id` when [adding a fleet policy](https://fleetdm.com/docs/rest-api/rest-api#create-fleet-policy).
- **Manual policies**: Use `software_title_id` to link a policy to software that installs on failure.

See the [Upload software](https://fleetdm.com/docs/rest-api/rest-api#add-package) and [Create fleet policy](https://fleetdm.com/docs/rest-api/rest-api#create-fleet-policy) API docs.

## Curious about GitOps?

Fleet's flexible API and support for a GitOps life cycle means this entire process can be stored and managed in code, further unlocking audibility, collaboration, and security. Know who made changes, when, and why—without being tied to vendor-specific methods. 

For manual policies, nest an **install_software** block in the policy you want to automate and ensure the path to the software matches the same path referenced in the fleet configuration file under the software block. For patch policies, set `type` to `patch` and specify `fleet_maintained_app_slug`. Check out the [GitOps reference documentation](https://fleetdm.com/docs/configuration/yaml-files#policies) for more details.

## Want to know more?

Reach out for more information and a demo, or explore Fleet's detailed [documentation](https://fleetdm.com/docs/get-started/why-fleet). 


<meta name="articleTitle" value="How to use policies for patch management in Fleet">
<meta name="authorFullName" value="Harrison Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-03-27">
<meta name="articleImageUrl" value="../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png">
<meta name="description" value="This guide explores automating patching using patch policies for Fleet-maintained apps and manual policies for custom packages.">
