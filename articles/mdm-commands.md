# MDM commands

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

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-06-12">
<meta name="articleTitle" value="MDM commands">
<meta name="description" value="Learn how to run custom MDM commands on hosts using Fleet.">
