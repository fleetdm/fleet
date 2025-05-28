# Manage software using GitOps

This guide is a walkthrough on how to manage software using [GitOps](https://fleetdm.com/docs/configuration/yaml-files#gitops). In Fleet, there are three types of software: [Fleet-maintained apps](#fleet-maintained-apps), [App Store (VPP) apps](#app-store-vpp-apps), and [custom packages](#custom-packages).

When using Gitops to manage Fleet, you can optionally put the UI in [GitOps mode](https://fleetdm.com/guides/gitops-mode). This prevents you from making changes in the UI that would be overridden by GitOps workflows.

## Fleet-maintained apps

Currently, managing [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps) is only supported using Fleet's UI or [API](https://fleetdm.com/docs/rest-api/rest-api) (Gitops support coming soon. Currently, Gitops won't delete/modify Fleet-maintained apps configured in the UI).

## App Store (VPP) apps

To configure App Store (VPP) apps, via GitOps, please see the [`app_store_apps`](https://github.com/fleetdm/fleet/blob/main/docs/Configuration/yaml-files.md#app_store_apps) of Fleet's best practice [GitOps documentation](https://github.com/fleetdm/fleet/blob/main/docs/Configuration/yaml-files.md#gitops). Note that VPP apps must first be added to [Apple Business Manager](https://business.apple.com).

## Custom packages

To configure custom packages via GitOps, please see the [`packages`](https://fleetdm.com/docs/configuration/yaml-files#packages) of Fleet's best practice [GitOps documentation](https://github.com/fleetdm/fleet/blob/main/docs/Configuration/yaml-files.md#gitops).

If you want to use Fleet to host custom packages instead of a third-party package hosting tool (ex. [Artifactory](https://jfrog.com/artifactory/)), first turn GitOps mode on in **Settings > Integration > Change management**
1. Navigate to **Software** and select a team. Then select **Add Software > Custom package**
2. Select a team and choose a file to upload and select **Add software**
3. A modal will appear with YAML instructions.
    1. Create a YAML file with the suggested filename and populate it with the contents below.
    2. Save this file to your repository.
    3. Make sure that the package YAML is referenced from your team YAML.
    4. Download the additional queries and scripts that are linked in the modal and add them to your repository. Make sure to use the paths listed in the contents area above.

To learn more about configuring custom packages using GitOps, [click here](https://github.com/fleetdm/fleet/blob/main/docs/Configuration/yaml-files.md#software).

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2025-04-30">
<meta name="articleTitle" value="Manage software in GitOps mode">
<meta name="description" value="Learn how to use Fleet's YAML to manage software in GitOps mode.">