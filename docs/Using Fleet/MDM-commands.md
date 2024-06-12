# Commands

In Fleet you can run MDM commands to take action on your macOS, iOS, iPadOS, and Windows hosts, like restarting the host, remotely.

## Custom commands

You can run custom commands and view a specific command's results using the `fleetctl` command-line interface.

To run a custom command, we will do the following steps:

1. Create a `.xml` with the request payload
2. Choose a target host
3. Run the command using `fleetctl`
4. View our command's results using `fleetctl`

### Step 1: Create an XML file

You can run any command supported by [Apple's MDM protocol](https://developer.apple.com/documentation/devicemanagement/commands_and_queries) or [Microsoft's MDM protocol](https://learn.microsoft.com/en-us/windows/client-management/mdm/).

> The lock and wipe commands are only available in Fleet Premium

For example, to restart a macOS host, we'll use the "Restart a Device" command documented by Apple [here](https://developer.apple.com/documentation/devicemanagement/restart_a_device#3384428). 

First, we'll need to create a `restart-device.xml` file locally with this payload:

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
</dict>
</plist>
```

To restart a Windows host, we'll use the "Reboot" command documented by Microsoft [here](https://learn.microsoft.com/en-us/windows/client-management/mdm/reboot-csp).

The `restart-device.xml` file will have this payload instead:

```xml
<Exec>
  <Item>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Reboot/RebootNow</LocURI>
    </Target>
    <Meta>
      <Format xmlns="syncml:metinf">null</Format>
      <Type>text/plain</Type>
    </Meta>
    <Data></Data>
  </Item>
</Exec>
```

### Step 2: Choose a target host

To run a command, we need to specify a target host by hostname.

1. Run the `fleetctl get hosts --mdm` command to get a list of hosts that are enrolled to Fleet and have MDM turned on.
2. Find your target host's hostname. You'll need this hostname to run the command.

### Step 3: Run the command

1. Run the `fleetctl mdm run-command --payload=restart-device.xml --hosts=hostname ` command.

> Replace the --payload and --hosts flags with your XML file and hostname respectively.

2. Look at the on-screen information. In the output you'll see the command to see results.

### Step 4: View the command's results

1. Run the `fleetctl get mdm-command-results --id=<insert-command-id>`
2. Look at the on-screen information.

## List recent commands

You can view a list of the 1,000 latest commands:

1. Run `fleetctl get mdm-commands`
2. View the list of latest commands, most recent first, along with the timestamp, targeted hostname, command type, execution status and command ID.

The command ID can be used to view command results as documented in [step 4 of the previous section](#step-4-view-the-commands-results). 

The possible statuses for macOS, iOS, and iPadOS hosts are the following:

* Pending: the command has yet to run on the host. The host will run the command the next time it comes online.
* NotNow: the host responded with "NotNow" status via the MDM protocol: the host received the command, but couldn’t execute it. The host will try to run the command the next time it comes online.
* Acknowledged: the host responded with "Acknowledged" status via the MDM protocol: the host processed the command successfully.
* Error: the host responded with "Error" status via the MDM protocol: an error occurred. Run the `fleetctl get mdm-command-results --id=<insert-command-id` to view the error.
* CommandFormatError: the host responded with "CommandFormatError" status via the MDM protocol: a protocol error occurred, which can result from a malformed command. Run the `fleetctl get mdm-command-results --id=<insert-command-id` to view the error.

The possible statuses for Windows hosts are documented in Microsoft's documentation [here](https://learn.microsoft.com/en-us/windows/client-management/oma-dm-protocol-support#syncml-response-status-codes).

<meta name="pageOrderInSection" value="1507">
<meta name="title" value="Commands">
<meta name="description" value="Learn how to run custom MDM commands on hosts using Fleet.">
<meta name="navSection" value="Device management">
