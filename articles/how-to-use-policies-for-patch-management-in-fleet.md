# How to use policies for patch management in Fleet

![How to use policies for patch management in Fleet](../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png)

Policies in Fleet enable IT admins to query devices and get quick yes or no answers about the status of their endpoints. Powered by the flexibility of osquery, the policies engine has become an invaluable part of the IT toolkit, simplifying the management of devices at scale.

Initially, Fleet’s policies allowed for automated responses, like firing a webhook on a policy failure or creating a ticket in your ITSM system. While effective, these actions were limited by the capabilities of your existing tools to process and act on these notifications.

## Enter patch management

Fleet’s policy capabilities have evolved beyond notification-based responses. With the release of Fleet v4.57, the policies engine now supports a game-changing feature: automated software installation on a policy failure. This addition transforms the policies engine into a dynamic tool for streamlined patch management.

In this article, we’ll explore how to leverage this new feature to automate patching across your environment. This will free up valuable IT resources to focus on high-impact tasks while enhancing end-user support.

## Why it matters

Around 60% of data breaches in 2023 involved vulnerabilities for which patches were available but not applied, underscoring the impact of delayed patch management.1

Regular updates often include bug fixes that improve stability and enhance user experience, allowing employees to work without disruptions. These updates also make sure compatibility with other applications, preventing integration issues that could impact workflows. 

Additionally, updated software often includes new features that can ultimately help teams work more efficiently and effectively.

## Let’s get started 

In this article, we will be using Google Chrome to demonstrate the functionality, and I already have the latest version’s .pkg downloaded locally.

Select the team you want the policy to run on. Navigate to **Software > Add Software**. Here you can use one of Fleet’s maintained apps, add from VPP or Custom Package. We will use Custom Package in this example and upload the Google Chrome.pkg mentioned previously. After upload, there are a couple of options for pre/post-install queries and scripts - you can read more about those options [here](https://fleetdm.com/guides/deploy-software-packages).

Navigate to **Policies**, select the team you want the policy to run in.

Assume we want to bring all devices in this team to the latest version of Chrome we uploaded to Fleet, which as of writing this, is 130.0.6723.70.

Your policy query would look something like this:

```sh
SELECT 1 FROM apps WHERE bundle_identifier = 'com.google.Chrome' AND bundle_short_version < '130.0.6723.70'
```

This means any evaluation of this policy, where the version is less than 130.0.6723.70, will result in a failure and thus kick off the automation.

Save and give it an intuitive name, we recommend something like:

_macOS - Update Google Chrome to Latest_

With the policy set, we can tie in the automation. Back on the main policy page, select **Manage automations > Install software**.

The module will show the policies available for that team. Check the box to turn on automation, and from the dropdown, select the software that will be installed on the failure. The dropdown will show all available software in that team - plus its supported OS and version.

And that’s it! Policies are evaluated across all online hosts every hour, or when a device is refetched manually. Any machine that fails this policy will install the Chrome version that was set in the policy.

## What else can we do?

This functionality unlocks many use cases for an IT admin to help manage their fleet. Another use case for this feature is to support a zero-touch deployment of devices and ensure that critical business and productivity software is installed from the first boot. 

A simple query like such: 

```sh
SELECT 1 FROM apps WHERE bundle_identifier = ‘com.tinyspeck.slackmacgap’
```

would deploy Slack to your endpoints the moment it comes out of the box, ensuring your users are ready to hit the ground running from day 1.

## Via the API

Fleet Premium customers can leverage the REST API to upload software packages and set policy automations using the software_title_id field. 

Info about the [Upload software](https://fleetdm.com/docs/rest-api/rest-api#add-package) and [Team policy](https://fleetdm.com/docs/rest-api/rest-api#add-team-policy) API docs are available in the documentation. 

## Curious about GitOps?

Fleet's flexible API and support for a GitOps life cycle means this entire process can be stored and managed in code, further unlocking audibility, collaboration, and security. Know who made changes, when, and why—without being tied to vendor-specific methods. 

Nest an **install_software** block in the policy you want to automate and ensure the path to the software matches the same path referenced in the team configuration file under the software block. Check out the [GitOps reference documentation](https://fleetdm.com/docs/configuration/yaml-files#policies) for more details.

## Want to know more?

Reach out for more information and a demo, or explore Fleet's detailed [documentation](https://fleetdm.com/docs/get-started/why-fleet). 

Sources

1. https://www.automox.com/blog/bad-cyber-hygiene-breaches-tied-to-unpatched-vulnerabilities

<meta name="articleTitle" value="How to use policies for patch management in Fleet">
<meta name="authorFullName" value="Harrison Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-11-07">
<meta name="articleImageUrl" value="../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png">
<meta name="description" value="This guide explores automating patching across your environment.">
