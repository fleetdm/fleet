# Deploy security agents

![Deploy security agents](../website/assets/images/articles/deploy-security-agents-1600x900@2x.png)

Fleet [v4.50.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.50.0) introduced the ability to upload and deploy security agents to your hosts. Beyond a [bootstrap package](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#bootstrap-package) at enrollment, deploying security agents allows you to specify and verify device configuration using a pre-enrollment osquery query and customization of the install and post-install scripts, allowing for key and license deployment and configuration.  This guide will walk you through the steps to upload, configure, and install a security agent to hosts in your fleet.

## Prerequisites

* Fleet [v4.50.0](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.50.0).
* `fleetd` 1.25.0 deployed via MDM or built with the `--scripts-enabled` flag.
* An S3 bucket [configured](https://fleetdm.com/docs/configuration/fleet-server-configuration#s-3-software-installers-bucket) to store the installers.
* Increase any load balancer timeouts to at least 5 minutes for the following endpoints:
    * [Add software](https://fleetdm.com/docs/rest-api/rest-api#add-software).
    * [Batch-apply software](https://fleetdm.com/docs/rest-api/rest-api#add-software).

## Step-by-step instructions

### Access security agent installers

To access and manage security agents in Fleet:

* **Navigate to the Software page**: Click on the "Software" tab in the main navigation menu.
* **Select a team**: Click on the dropdown at the top left of the page.
* **Find your software**: using the filters on the top of the table, you can choose between:
    * “Available for install” filters software that can be installed on your hosts.
    * “Self-service” filters software that end users can install from Fleet Desktop.
* **Select security agent installer**: Click on a software package to view details and access additional actions for the agent installer.

### Add a security agent to a team

* **Navigate to the Software page**: Click on the "Software" tab in the main navigation menu.
* **Select a team**: Select a team or the "No team" team to add a security agent.

> Security agents cannot be added to "All teams"

* Click the “Add Software” button in the top right corner, and a modal will appear.
* Choose a file to upload. `.pkg`, `.msi`, `.exe`, or `.deb` files are supported.
* After selecting a file, a default install script will be pre-filled. If the security agent requires a custom installation process, this script can be edited.
* To allow users to install the software from Fleet Desktop, check the “Self-service” checkbox.
* To customize the conditions, click on “Advanced options”:
    * **Pre-install condition**: A pre-install condition is a valid osquery SQL statement that will be evaluated on the host before installing the software. If provided, the installation will proceed only if the query returns any value.
    * **Post-install script** A post-install script will run after the installation is complete, allowing you to configure the security agent right after installation. If this script returns a non-zero exit code, the installation will fail, and `fleetd` will attempt to uninstall the software.

### Install a security agent on a host

After an installer is added to a team, it can be installed on hosts via the UI.

* **Navigate to the Hosts page**: Click on the "Hosts" tab in the main navigation menu.
* **Navigate to the Host details page**: Click the host you want to install the security agent.
* **Navigate to the Host software tab**: In the host details, search for the tab named “Software”
* **Find your security agent**: Use the search bar and filters to search for your security agent.
* **Install the security agent on the host**: In the leftmost row of the table, click on “Actions” > “Install.”
* **Track installation status**: by either
    * Checking the “Install status” in the host software table.
    * Navigate to the “Details” tab on the host details page and check the activity log.

### Edit a security agent

Security agent installers can’t be edited via the UI. To modify an installer, remove it from the UI and add a new one.

### Remove a security agent from a team

* **Navigate to the Software page**: Click on the "Software" tab in the main navigation menu.
* **Select a team**: Select a team or the "No team" team to add a security agent.
* **Find your software**: using the filters on the top of the table, you can choose between:
    * “Available for install” filters software can be installed on your hosts.
    * “Self-service” filters software that users can install from Fleet Desktop.
* **Select security agent installer**: Click on a software package to view details.
* **Remove security agent installer**: From the Actions menu, select "Delete." Click the "Delete" button on the modal.

> Removing a security agent from a team will not uninstall the agent from the existing host(s).

### Manage security agents with the REST API

Fleet also provides a REST API for managing software programmatically. The API allows you to add, update, retrieve, list, and delete software. Detailed documentation on Fleet's [REST API is available](https://fleetdm.com/docs/rest-api/rest-api#software).

### Manage security agents with GitOps

Installers for security agents can be managed via `fleetctl` using [GitOps](https://fleetdm.com/docs/using-fleet/gitops).

Please refer to the documentation specific to [managing software with GitOps](https://fleetdm.com/docs/using-fleet/gitops#software). For a real-world example, [see how we manage software at Fleet](https://github.com/fleetdm/fleet/tree/main/it-and-security/teams).


## Conclusion

Deploying security agents with Fleet is straightforward and ensures your hosts are protected with the latest security measures. This guide has shown you how to access, add, and install security agents, as well as manage them using the REST API and `fleetctl`. Following these steps can effectively equip your fleet with the necessary security tools.

See Fleet's [documentation](https://fleetdm.com/docs/using-fleet) and additional [guides](https://fleetdm.com/guides) for more details on advanced setups, software features, and vulnerability detection.


<meta name="articleTitle" value="Deploy security agents">
<meta name="authorFullName" value="Roberto Dip">
<meta name="authorGitHubUsername" value="roperzh">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-08-05">
<meta name="articleImageUrl" value="../website/assets/images/articles/deploy-security-agents-1600x900@2x.png">
<meta name="description" value="This guide will walk you through adding apps to Fleet for user self-service.">
