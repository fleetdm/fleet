# Scripts

In Fleet you can run custom scripts to remediate an issue on your macOS, Windows, and Linux hosts.

Shell scripts are supported on macOS and Linux. By default, shell scripts will run in the host's (root) shell (`/bin/sh`). We also support `/bin/zsh` and `/bin/bash` interpreters.
Note: To run in `/bin/zsh` or `/bin/bash`, create `.sh` file (only supported extension) and add an interpreter at the first line.

PowerShell scripts are supported on Windows. Other types of scripts are not supported yet.

Script execution is disabled by default. Continue reading to learn how to enable scripts.

## Enable scripts

If you use Fleet's macOS MDM features, scripts are automatically enabled for macOS hosts that have MDM turned on. You're set!

If you don't use MDM features, to enable scripts, we'll [deploy a fleetd agent](https://fleetdm.com/guides/enroll-hosts) with scripts enabled:

1. Generate a new fleetd agent for macOS, Windows, or Linux using the `fleetctl package` command with the `--enable-scripts` flag. 

2. Deploy fleetd to your hosts. If your hosts already have fleetd installed, you can deploy the new fleetd on-top of the old installation.

## Manually run scripts

You can run a script in the Fleet UI, with Fleet API, or with the fleetctl command-line interface (CLI).

Fleet UI (single host):

1. In Fleet, head to the **Controls > Scripts** tab and upload your script.

2. Head to the **Hosts** page and select the host you want to run the script on.

3. On your target host's host details page, select the **Actions** dropdown and select **Run Script** to view the **Run Script** menu.

4. In the **Run Script** menu, select the **Actions** dropdown for the script you'd like to execute and choose the **Run** option.

Fleet UI (multiple hosts):

1. In Fleet, head to the **Controls > Scripts** tab and upload your script.

2. Head to the **Hosts** page. If you're on Fleet Premium, select a team (or "no team").

3. Click the checkbox next to one or more hosts you want to run the script on.

4. Click "Run script" in the table header.

5. In the popup modal, find the script you'd like to run, move the mouse pointer to that item in the list and click the "Run script" button that appears.

Scripts run from the Fleet UI will run the next time your host checks in with Fleet. You can view the status of the script execution as well as the output in the target host's activity feed.

When executing a script on more than one host, you can view the status of the batch of hosts by clicking on the related item in the global activity feed.

Fleet API: See our [REST API documentation](https://fleetdm.com/docs/rest-api/rest-api#run-script)

fleetctl CLI:

```sh
fleetctl run-script --script-path=/path/to/script --host=hostname
```

## Automatically run scripts

You can [automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script) using Fleet via policy automations.

## Batch execute scripts

You can execute a script on a large number of hosts at the same time using the Fleet UI or Fleet API.

Fleet UI:

1. In Fleet, go to the **Hosts** page, and select a team.

2. Select the hosts that you want to run the script on.

3. Click the **Run Script** button at the top of the list of hosts.

4. In the **Run Script** modal, mouse over the script you want to run and click **Run Script**.

5. Either close the modal or select another script to run.

Fleet API: See the [REST API documentation](https://fleetdm.com/docs/rest-api/rest-api#batch-run-script)

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-10-07">
<meta name="articleTitle" value="Scripts">
<meta name="description" value="Learn how to execute a custom script on macOS, Windows, and Linux hosts in Fleet.">
