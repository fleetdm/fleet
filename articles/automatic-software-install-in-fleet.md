# Automatic policy-based installation of software on hosts

![Top Image](../website/assets/images/articles/automatic-software-install-top-image.png)

Fleet [v4.57.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.57.0) introduces the ability to automatically and remotely install software on hosts based on predefined policy failures. This guide will walk you through the process of configuring fleet for automatic installation of software on hosts using uploaded installation images and based on programmed policies.  You'll learn how to configure and use this feature, as well as understand how the underlying mechanism works.

Fleet allows its users to upload trusted software installation files to be installed and used on hosts. This installation could be conditioned on a failure of a specific Fleet Policy.

## Prerequisites

* Fleet premium with Admin permissions.
* Fleet [v4.57.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.57.0) or greater.

## Step-by-step instructions

1. **Adding software**: Add any software to be available for installation. Follow the [deploying software](https://fleetdm.com/guides/deploy-security-agents) document with instructions how to do it. Note that all installation steps (pre-install query, install script, and post-install script) will be executed as configured, regardless of the policy that triggers the installation.


![Add software](../website/assets/images/articles/automatic-software-install-add-software.png)

Current supported software deployment formats:
- macOS: .pkg
- Windows: .msi, .exe
- Linux: .deb

Coming soon:
- VPP for iOS and iPadOS

2. **Add a policy**: In Fleet, add a policy that failure to pass will trigger the required installation. Go to Policies tab --> Press the top right "Add policy" button. --> Click "create your own policy" --> Enter your policy SQL --> Save --> Fill in details in the Save modal and Save.

```
SELECT 1 FROM apps WHERE name = 'Adobe Acrobat Reader.app' AND version_compare(bundle_short_version, '23.001.20687') >= 0;
```

Note: In order to know the exact application name to put in the query (e.g. "Adobe Acrobat Reader.app" in the query above) you can manually install it on a canary/test host and then query SELECT * from apps;


3. **Manage automation**: Open Manage Automations: Policies Tab --> top right "Manage automations" --> "Install software".

![Manage policies](../website/assets/images/articles/automatic-software-install-policies-manage.png)

4. **Select policy**: Select (click the check box of) your newly created policy. To the right of it select from the
   drop-down list the software you would like to be installed upon failure of this policy.

![Install software modal](../website/assets/images/articles/automatic-software-install-install-software.png)

Upon failure of the selected policy, the selected software installation will be triggered.

## How does it work?

* After configuring Fleet to auto-install a specific software the rest will be done automatically.
* The policy check mechanism runs on a typical 1 hour cadence on all online hosts. 
* Fleet will send install requests to the hosts on the first policy failure (first "No" result for the host) or if a policy goes from "Yes" to "No". On this iteration it will not send a install request if a policy is already failing and continues to fail ("No" -> "No"). See the following flowchart for details.

![Flowchart](../website/assets/images/articles/automatic-software-install-workflow.png)
*Detailed flowchart*

## Using the REST API for self-service software packages

Fleet provides a REST API for managing software packages, including self-service software packages.  Learn more about Fleet's [REST API](https://fleetdm.com/docs/rest-api/rest-api#add-team-policy).

## Managing self-service software packages with GitOps

To manage self-service software packages using Fleet's best practice GitOps, check out the `software` key in the [GitOps reference documentation](https://fleetdm.com/docs/configuration/yaml-files#policies).

## Conclusion

Software deployment can be time-consuming and risky. This guide presents Fleet's ability to mass deploy software to your fleet in a simple and safe way. Starting with uploading a trusted installer and ending with deploying it to the proper set of machines answering the exact policy defined by you.

Leveraging Fleet’s ability to install and upgrade software on your hosts, you can streamline the process of controlling your hosts, replacing old versions of software and having the up-to-date info on what's installed on your fleet.

By automating software deployment, you can gain greater control over what's installed on your machines and have better oversight of version upgrades, ensuring old software with known issues is replaced.

<meta name="articleTitle" value="Automatic installation of software on hosts">
<meta name="authorFullName" value="Sharon Katz">
<meta name="authorGitHubUsername" value="sharon-fdm">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-09-23">
<meta name="description" value="A guide to workflows using automatic software installation in Fleet.">
