# Set a device hostname via the Fleet API

You can rename macOS, iOS, and iPadOS devices by sending an MDM command through Fleet's API. This is useful for enforcing naming conventions or identifying devices at a glance.

For more MDM commands and detailed guidance, see [MDM commands](https://fleetdm.com/guides/mdm-commands).


## Prerequisites

- A Fleet API token with write access
- The device's serial number
- The device must be enrolled in Fleet's MDM


## Get the host UUID

First, retrieve the host's UUID using its serial number.

**Endpoint:** `GET /api/v1/fleet/hosts/identifier/{serial}`

**Headers:**
- `Accept: application/json`
- `Authorization: Bearer {your_api_token}`

The response includes the host object. Extract the `uuid` field from `host.uuid`.


## Create the rename command

Build an XML payload using Apple's Settings MDM command. Replace `NEW-HOSTNAME` with your desired device name.

**Important:** The `CommandUUID` must be unique for each command you send. You can use a timestamp, UUID generator, or any unique identifier. For example: `Settings_20260127_143052` or a proper UUID like `A1B2C3D4-E5F6-7890-ABCD-EF1234567890`.

To generate a UUID on macOS:
```sh
uuidgen
```

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>RequestType</key>
        <string>Settings</string>
        <key>Settings</key>
        <array>
            <dict>
                <key>DeviceName</key>
                <string>NEW-HOSTNAME</string>
                <key>Item</key>
                <string>DeviceName</string>
            </dict>
        </array>
    </dict>
    <key>CommandUUID</key>
    <string>UUID-GOES-HERE</string>
</dict>
</plist>
```


## Base64 encode the command

The Fleet API requires MDM command payloads to be base64 encoded.

**macOS or Linux:**
```sh
cat command.xml | base64
```

**Windows (PowerShell):**
```powershell
[Convert]::ToBase64String((Get-Content -Path "command.xml" -Encoding byte))
```

Save the output for the next step.


## Send the command

Submit the encoded command to Fleet. For complete endpoint details, see the [Run MDM command](https://fleetdm.com/docs/rest-api/rest-api#run-mdm-command) API reference.

**Endpoint:** `POST /api/v1/fleet/commands/run`

**Headers:**
- `Content-Type: application/json`
- `Authorization: Bearer {your_api_token}`

**Body:**
```json
{
  "command": "{base64_encoded_command}",
  "host_uuids": ["{host_uuid}"]
}
```

You can include multiple UUIDs in the `host_uuids` array to rename several devices at once.

The device will update its hostname the next time it checks in with Fleet.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="authorFullName" value="Kitzy">
<meta name="publishedOn" value="2026-01-26">
<meta name="articleTitle" value="Set a device hostname via the Fleet API">
<meta name="description" value="Use Fleet's API to set device hostnames on macOS, iOS, and iPadOS. This guide walks through building and sending the MDM command to rename devices.">