# Policies in Fleet for software patching

Fleet [v4.57](https://fleetdm.com/releases/fleet-4.57.0) introduced a powerful new feature that enables automatic software installation based on policy failures. This guide will walk you through configuring this functionality in Fleet, available for macOS, Windows, and Linux platforms.

## Step-by-step instructions	

We will be using Google Chrome in this example but any of these supported software types can be used - `.pkg, .msi, .exe, .deb, or .rpm`

1. **Add the software:** Select the team you want the policy to run on. Navigate to **Software > Add Software.** Here you have the option to use one of Fleet’s maintained apps, VPP or Custom Package. 

    After upload, there are a few options for pre/post-install queries and scripts - you can read more about those options [here](https://fleetdm.com/guides/deploy-software-packages).

2. **Add a policy:** Navigate to **Policies**, select the team you want the policy to run in. Assume we want to bring all devices in this team to **Chrome 129.0.6668.90**, which was uploaded in the previous step.

    Your policy query would look something like this:

    ```
    SELECT 1 FROM apps WHERE bundle_identifier = 'com.google.Chrome' AND bundle_short_version < '129.0..6668.90'
    ```

3. Save this and give it an intuitive name, we recommend something like:

    _macOS - Update Google Chrome to Latest_

With the policy set, we now tie in the automation. 

4. Back on the main policy page for the team, select **Manage automations > Install software**.

The module will show all available policies for the selected team. Check the box to turn on **automation** and from the dropdown, select the software from the previous step. The dropdown will show all available software in that team - plus its supported OS and version.

![Policy automation module](..website/assets/images/articles/policy-automation-in-fleet-for-software-patching-512x87@2x.png)

Next time a host fails this policy, Fleet will install the selected software on the endpoint. Admins can view the status of this action through the Activity feed.

Policies are evaluated across all online hosts every hour by default, or when a device is re fetched manually. 

## Via the API
Fleet Premium customers can leverage the REST API to upload software packages as well as set policy automations using the software_title_id field. 

Info about the [Upload software](https://fleetdm.com/docs/rest-api/rest-api#add-package) and [Team policy](https://fleetdm.com/docs/rest-api/rest-api#add-team-policy) API docs are available in the documentation. 	 

## Via GitOps

Software install can also leverage a GitOps workflow. Nest an **install_software** block in the policy you want to automate and ensure the path to the software matches the same path referenced in the team configuration file under the **software** block. Check out the [GitOps reference documentation](https://fleetdm.com/docs/configuration/yaml-files#policies) for more details.

<meta name="articleTitle" value="Policies in Fleet for software patching">
<meta name="authorFullName" value="Harrison Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-11-13">
<meta name="articleImageUrl" value="../website/assets/images/articles/sysadmin-diaries-1600x900@2x.png">
<meta name="description" value="This guide explores using policies to automatet software patching across your environment.">
