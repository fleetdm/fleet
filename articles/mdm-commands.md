# MDM commands

MDM commands can be sent to macOS, iOS / iPadOS and Windows hosts managed in Fleet using the following general steps:

1. Create a payload that functions as the MDM command.
2. Choose a target host, or, a set of target hosts on which to run the MDM command.
3. Execute the MDM command by using the `fleetctl` command line interface (CLI) or by sending the payload in a Fleet API call.
4. If needed, verify the MDM command result with an additional `fleetctl` command or API call.

### Step 1: Create an MDM command payload

An MDM command payload can be created in mulitple ways.

For Apple devices, the payload is a  `.plist` that can be copied like this example from [Apple's developer documentation](https://developer.apple.com/documentation/devicemanagement/remove-profile-command), created with the [iMazing Profile Editor](https://imazing.com/profile-editor) or exported from a 3rd party MDM solution.

For Windows, the payload is standard `xml` and command options can be referenced in the [Microsoft CSP Policy docuementation](https://learn.microsoft.com/en-us/windows/client-management/mdm/policy-configuration-service-provider).

You can run any command supported by [Apple's MDM protocol](https://developer.apple.com/documentation/devicemanagement/commands_and_queries) or [Microsoft's MDM protocol](https://learn.microsoft.com/en-us/windows/client-management/mdm/).

The end result simply needs to be a standard, plain text file with the correct key / values for obtaining the intended result on the host device.

> Lock and wipe commands are only available in Fleet Premium.

### Examples

To restart a macOS host, we can use the ["Restart a Device" MDM command](https://developer.apple.com/documentation/devicemanagement/restart_a_device).

Below is the text to be used as the MDM command payload. Save it as a file and name it something like `apple-restart-device.xml`.

```
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

To restart a Windows host, we can use the ["Reboot" command](https://learn.microsoft.com/en-us/windows/client-management/mdm/reboot-csp).

Below is the text to be used as the MDM command payload. Save it as a file and name it something like `windows-restart-device.xml`.

```
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

To prepare an MDM command payload for use with the Fleet API, generate a UUID to be used as the `CommandUUID`, e.g.,

In Terminal, execute the following command:

```
% uuidgen
16F4301E-7A88-42AD-8523-A2F73F9D38FA
```

> It's not necessary to add the `CommandUUID` to the MDM command payload, but having it available makes it easier and quicker to verify the MDM command result if a check is needed. 

A `.plist` with the `CommandUUID` key / value added will look something like this:

```
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Command</key>
  <dict>
    <key>Identifier</key>
    <string>com.some.profile</string>
    <key>RequestType</key>
    <string>RemoveProfile</string>
  </dict>
  <key>CommandUUID</key>
  <string>16F4301E-7A88-42AD-8523-A2F73F9D38FA</string>
</dict>
</plist>
```

### Step 2: Choose a target host

Run the `fleetctl get hosts --mdm` command to get a list of hosts that are enrolled in Fleet and have MDM enabled. This may not be practical in Fleet environments with a large number of hosts without using command line tools to parse the output, e.g.,

Use something like `grep` with `fleetctl`:

```
% fleetctl get hosts --mdm | grep -i 'someSearchStringHere'                                                        
Client Version:   4.67.3
Server Version:  4.67.3
| 1B848BE8-some-uuid | someComputer | darwin | 5.17.0 | online  |
| 08C7634C-some-uuid | someOtherComputer | windows  | 5.17.0 | online  |
```

Or, something like `jq` for API output:

```
% curl -LSs \
--request GET \
--header 'Accept: application/json' \
--header "Authorization: Bearer $fleet_key" \
"$fleet_url/api/v1/fleet/hosts" | jq '.hosts[] | select(.computer_name | contains("someSearchStringHere"))'
```

> You will need a [Fleet API token](https://fleetdm.com/docs/rest-api/rest-api#retrieve-your-api-token) in your `fleetctl` configuration or for any interaction with the Fleet API to work.

### Step 3: Execute the MDM command

To deliver the MDM command payload with `fleetctl`, use something like the following that:

- references the MDM command payload file created above, and
- references the intended host targets, e.g.,

`fleetctl mdm run-command --payload='apple-restart-device.xml' --hosts='someHostname'`

For targeting multiple hosts, the `--hosts` option can be populated with comma-separated values.

To prepare the MDM command payload for execution in a Fleet API call, it must be base64-encoded. This is true for both Apple and Windows MDM command payloads. E.g., to encode the `.plist` in Terminal:

```
% echo '<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Command</key>
  <dict>
    <key>Identifier</key>
    <string>com.some.profile</string>
    <key>RequestType</key>
    <string>RemoveProfile</string>
  </dict>
  <key>CommandUUID</key>
  <string>16F4301E-7A88-42AD-8523-A2F73F9D38FA</string>
</dict>
</plist>' | base64
PD94bWwgdmVyc2lvbj0iMS4wIiBlSomeMorebase64blahblahblah...
```

Then, to deliver the MDM command payload via the Fleet API, use a command that conforms to the `curl` example below. (This can be achieved with any programmatic solution, e.g., python `requests` or `urllib.request`).

```
% fleet_key='yourfleetAPItoken'
% fleet_url='https://your.url.com'
% /usr/bin/curl -LSs \
--request POST \
--header 'Content-Type: application/json' \
--header "Authorization: Bearer $fleet_key" \
--data '{"command":"PD94bWwgdmVyc2lvbj0iMS4wIiBlSomeMorebase64blahblahblah...","host_uuids":["some-host-uuid"]}' \
"$fleet_url/api/v1/fleet/commands/run"
```

For targeting multiple hosts, the `"host_uuids"` key / value is a json array that can be populated with multiple host uuid values, e.g.,

`"host_uuids":["some-host-uuid-1","some-host-uuid-2","some-host-uuid-3"]`

### Step 4: Verify the MDM command result

To verify the MDM command result with `fleetctl`, use something like the command below:

`fleetctl get mdm-command-results --id=<insert-command-id>`

If you generated the `CommandUUID`, add that value in the `--id` field. If you did not generate a command ID, one will be added to the MDM command by Fleet and it should appear in a succesful response after execution or in the MDM command results stored in Fleet.

To verify the MDM command result with the Fleet API, use a command that conforms to the `curl` example below:

```
% /usr/bin/curl -LSs \
--request GET \
--header 'Accept: application/json' \
--header "Authorization: Bearer $fleet_key" \
"$fleet_url/api/v1/fleet/commands/results?command_uuid=16F4301E-7A88-42AD-8523-A2F73F9D38FA"
```

> The `?command_uuid=` parameter appended to the URL is populated with the same `CommandUUID` string that was used to populate the `CommandUUID` key / value in the base64-encoded `.plist` in Step 1.

## Troubleshooting

You can view a list of the 1000 most recent MDM commands executed in Fleet by running:

`fleetctl get mdm-commands`

The output will be sorted by the "most recent first" column and will include timestamp, targeted hostname, command type, execution status and command ID values.

The command ID can be used to view MDM command results as documented in Step 4.

You can also get this list of MDM commands from the Fleet API with something like:

```
% /usr/bin/curl -LSs \
--request GET \
--header 'Accept: application/json' \
--header "Authorization: Bearer $fleet_key" \
"$fleet_url/api/v1/fleet/commands/results"
```

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-06-12">
<meta name="articleTitle" value="MDM commands">
<meta name="description" value="Learn how to run custom MDM commands on hosts using Fleet.">
