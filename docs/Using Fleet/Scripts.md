# Scripts

_Available in Fleet Premium_

In Fleet you can execute a custom script to remediate an issue on your macOS, Windows, and Linux hosts.

Shell scripts are supported on macOS and Linux. All scripts will run in the host's (root) default shell (`/bin/sh`). Other interpreters are not supported yet.

PowerShell scripts are supported on Windows. Other types of scripts are not supported yet.

Script execution is disabled by default. Continue reading to learn how to enable scripts.

## Execute a script

You can execute a script using the `fleetctl` command-line interface.

To execute a script, we will do the following steps:
1. Enable script execution
2. Write a script
3. Run the script
4. View script activities in the UI

### Step 1: Enable script execution

If you use Fleet's macOS MDM features, scripts are automatically enabled for macOS hosts that have MDM turned on. You're set!

If you don't use MDM features, to enable scripts, we'll deploy a fleetd agent with scripts enabled:

1. Generate a new fleetd agent for macOS, Windows, or Linux using the `fleetctl package` command with the `--script-execution` flag. 

2. Deploy fleetd to your hosts. If your hosts already have fleetd installed, you can deploy the new fleetd on-top of the old installation.

Learn more about generating a fleetd agent and deploying it [here](./enroll-hosts.md#enroll-hosts-with-fleetd).

### Step 2: write a script

As an example, we'll write a shell script for a macOS host that downloads a Fleet wallpaper and set the host's wallpaper to it.

To run the script, we'll need to create a `set-wallpaper-to-fleet.sh` file locally and copy and paste this script into this `.sh` file:

```sh
wallpaper="/tmp/wallpaper.png" 

curl --fail https://fleetdm.com/images/wallpaper-cloud-city-1920x1080.png -o $wallpaper

osascript -e 'tell application "Finder" to set desktop picture to POSIX file "'"$wallpaper"'"' 
```

### Step 3: run the script

1. Run the `fleetctl run-script --script_path=set-wallpaper-to-fleet.sh --host=hostname` command.

> Replace --host flag with your target host's hostname respectively.

2. Look at the on-screen information. In the output you'll see the script's exit code and output.

### Step 4: view script activities in the UI

Each time a Fleet user runs a script an entry is created in Fleet's activity feed. This entry includes the user's name, script content, script exit code, script output, and a timestamp of when the script was run.

To view the activity in the UI, click the Fleet icon in the top navigation bar and locate the **Activity** section.

You can optionally send all activity feed entries to your log destination. Learn more [here](./Audit-logs.md).

## Security considerations

Script execution can only be enabled by someone with root access to the host.

Turning MDM on for a macOS host or pushing a new fleetd agent qualify as root access.

<meta name="pageOrderInSection" value="1506">
<meta name="title" value="Scripts">
<meta name="description" value="Learn how to execute a custom script on macOS, Windows, and Linux hosts in Fleet.">
<meta name="navSection" value="Device management">
