# Commands

In Fleet you can run MDM commands to take some action on your macOS hosts, like restart the host, remotely.

If a host is offline when you run a command, the host will run the command the next time it comes online.

## Custom commands

You can run a custom commands and view a specific command's results using the `fleetctl` command-line interface.

To run a custom command, we will do the following steps:
1. Create a `.xml` with the request payload
2. Choose a target host
3. Run the command using `fleetctl`
4. View our command's results using `fleetctl`

### Step 1: create a `.xml` file

You can run any command supported by Apple's MDM protocol as a custom command in Fleet. To see the list of possible commands, head to [Apple's Commands and Queries documentation](https://developer.apple.com/documentation/devicemanagement/commands_and_queries).

> The "Erase a device" and "Lock a device" commands are only available in Fleet Premium

Each command has example request payloads in XML format. For example, if we want to restart a host, we'll use the "Restart a Device" request payload documented by Apple [here](https://developer.apple.com/documentation/devicemanagement/restart_a_device#3384428).

To run the "Restart a device" command, we'll need to create a `restart-device.xml` file locally and copy and paste the request payload into this `.xml` file:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>RequestType</key>
        <string>RestartDevice</string>
    </dict>
    <key>CommandUUID</key>
    <string>0001_RestartDevice</string>
</dict>
</plist>
```

### Step 2: choose a target host

To run a command, we need to specify a target host by hostname. Commands can only be run on a single host in Fleet.

To find a host's hostname, choose the "Fleet UI" or "fleetctl" method and follow the steps below.

Fleet UI:

1. Head to the **Hosts** page in Fleet and find your target host.
2. Make sure the **Hostname** column is visible (select **Edit columns** if not) and find your host's hostname. You'll need this hostname to run the command.

> A host must be enrolled to Fleet and have MDM turned on to run an command against it.

`fleetctl` CLI:

1. Run the `fleetctl get hosts --mdm` command to get a list of hosts that are enrolled to Fleet and have MDM turned on.
2. Find your host's hostname. You'll need this hostname to run the command.

### Step 3: run the command

1. Run the `fleetctl mdm run-command --payload=restart-device.xml --host=hostname `
> Replace the --payload and --host flags with your `.xml` file and hostname respectively.

2. Look at the on-screen information. In the output you'll see the command required to see results. Be sure to copy this command. If you don't, it will be difficult to view command results later.

### Step 4: View the command's results

1. Run the `fleetctl get mdm-command-results --id=<insert-command-id>`

2. Look at the on-screen information.

<meta name="pageOrderInSection" value="1506">
<meta name="title" value="MDM commands">
