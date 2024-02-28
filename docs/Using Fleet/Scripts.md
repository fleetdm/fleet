# Scripts

In Fleet you can execute a custom script to remediate an issue on your macOS, Windows, and Linux hosts.

Shell scripts are supported on macOS and Linux. All scripts will run in the host's (root) default shell (`/bin/sh`). Other interpreters are not supported yet.

PowerShell scripts are supported on Windows. Other types of scripts are not supported yet.

Script execution is disabled by default. Continue reading to learn how to enable scripts.

## Enable scripts

If you use Fleet's macOS MDM features, scripts are automatically enabled for macOS hosts that have MDM turned on. You're set!

If you don't use MDM features, to enable scripts, we'll deploy a fleetd agent with scripts enabled:

1. Generate a new fleetd agent for macOS, Windows, or Linux using the `fleetctl package` command with the `--enable-scripts` flag. 

2. Deploy fleetd to your hosts. If your hosts already have fleetd installed, you can deploy the new fleetd on-top of the old installation.

Learn more about generating a fleetd agent and deploying it [here](./enroll-hosts.md).

## Execute a script

You can execute a script in the Fleet UI, with Fleet API, or with the fleetctl command-line interface (CLI).

Fleet UI:

1. In Fleet, head to the **Controls > Scripts** tab and upload your script.

2. Head to the **Hosts** page and select the host you want to run the script on.

3. On your target host's host details page, select the **Scripts** tab and select **Actions** to run the script.

Fleet API: API documentation is [here](https://fleetdm.com/docs/rest-api/rest-api#run-script]

fleetctl CLI:

```sh
fleetctl run-script --script-path=/path/to/script --host=hostname
```

<meta name="pageOrderInSection" value="1508">
<meta name="title" value="Scripts">
<meta name="description" value="Learn how to execute a custom script on macOS, Windows, and Linux hosts in Fleet.">
<meta name="navSection" value="Device management">
