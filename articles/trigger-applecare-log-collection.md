# Trigger AppleCare log collection with a custom MDM command

When you open an AppleCare case, Apple often needs diagnostic logs from the affected device. Collecting them used to mean physical access to the device and a user walking through the steps. On iOS 27 and iPadOS 27, you can start that collection remotely by sending Apple's `TriggerEnhancedLogCollection` command as a custom MDM command through Fleet. The device gathers the logs and uploads them directly to Apple, where they attach to your AppleCare ticket.

> **Note:** Fleet supports enhanced log collection on iOS and iPadOS. Support for macOS and tvOS isn't available yet.

## Prerequisites

Check these before you start:

- Fleet MDM turned on, with the target iPhone or iPad enrolled and supervised. Enhanced log collection doesn't work on unsupervised devices.
- iOS 27 or iPadOS 27 or later on the target device.
- An AppleCare token for the case. AppleCare issues this as part of the ticket, either an interactive or a non-interactive token (see the note below).
- `fleetctl` installed and logged in, or a [Fleet API token](https://fleetdm.com/docs/rest-api/rest-api#retrieve-your-api-token).
- An AppleCare Enterprise agreement. Apple requires one to use enhanced log collection, including for testing during the beta.

> **Note:** The token type controls what the user sees. An interactive token prompts the user to consent, and they can review what was collected and decline before anything uploads. A non-interactive (headless) token shows a notification and collects in the background. iPhone and iPad devices require interactive collection when a passcode is set or an account (such as iCloud or Mail) is configured. Shared iPad always uses non-interactive collection. Ask AppleCare for the token type that fits the device.

> **Warning:** The AppleCare token authorizes a diagnostic session on the device. Treat it like a secret. Don't commit it to a repository or paste it into a shared ticket.

## Get the AppleCare token

Open (or reference) the AppleCare case for the affected device and ask AppleCare for the enhanced log collection token. Keep it handy for the command payload in the next step. Each token is tied to a specific case, so use the one AppleCare issued for this device.

> **Note:** To test the workflow before you have a real case, Apple provides beta tokens that don't open a case or upload logs: `test-token-normal` (interactive), `test-token-normal-headless` (non-interactive), and `test-token-failed` (simulates a failure). You can still cancel or decline these sessions as normal.

## Create the command payload

Save the following as `trigger-enhanced-log-collection.xml`. Replace `APPLECARE-TOKEN` with the token from your case, and replace `UUID-GOES-HERE` with a unique identifier.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>RequestType</key>
        <string>TriggerEnhancedLogCollection</string>
        <key>AppleCareToken</key>
        <string>APPLECARE-TOKEN</string>
    </dict>
    <key>CommandUUID</key>
    <string>UUID-GOES-HERE</string>
</dict>
</plist>
```

Generate a unique `CommandUUID` so you can look up the result later:

```bash
uuidgen
```

## Send the command

Point the command at the affected host. Because an AppleCare case is about one device, target a single host.

### Using fleetctl

```bash
fleetctl mdm run-command --payload=trigger-enhanced-log-collection.xml --hosts=SERIAL_NUMBER
```

### Using the Fleet API

Base64 encode the payload first:

```bash
base64 < trigger-enhanced-log-collection.xml
```

Then send it. For endpoint details, see the [Run MDM command](https://fleetdm.com/docs/rest-api/rest-api#run-mdm-command) API reference.

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

The device starts the session the next time it checks in. On an interactive session, the user is prompted to consent before anything uploads.

## Verify

Confirm the command reached the device using the `CommandUUID` you set:

```bash
fleetctl get mdm-command-results --id=UUID-GOES-HERE
```

A `CommandUUID` you didn't set means Fleet generated one. Run `fleetctl get mdm-commands` to find it, then pass it to `--id`.

For an interactive session, confirm with the user that they consented on the device. Apple uploads the logs to your case, so the final confirmation is the logs appearing on the AppleCare ticket.

## Cancel a session

To stop an active collection session, send `CancelEnhancedLogCollection` the same way. Save this as `cancel-enhanced-log-collection.xml`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>RequestType</key>
        <string>CancelEnhancedLogCollection</string>
    </dict>
    <key>CommandUUID</key>
    <string>UUID-GOES-HERE</string>
</dict>
</plist>
```

```bash
fleetctl mdm run-command --payload=cancel-enhanced-log-collection.xml --hosts=SERIAL_NUMBER
```

## Troubleshoot

**The command result shows Failed or the device never responds.** Confirm the device runs iOS or iPadOS 27 or later and is supervised. Enhanced log collection is unavailable on earlier versions and on unsupervised devices.

**The user never sees a consent prompt.** The token is likely non-interactive, or the device qualifies for background collection. iPhone and iPad prompt only when a passcode or account is present, and Shared iPad never prompts. If you expected a prompt, ask AppleCare to reissue an interactive token.

**Logs don't appear on the AppleCare ticket.** The token may be expired or tied to a different case. Request a fresh token from AppleCare for this specific device, then resend the command.

## Related resources

- [MDM commands](https://fleetdm.com/guides/mdm-commands)
- [Run MDM command API reference](https://fleetdm.com/docs/rest-api/rest-api#run-mdm-command)
- [Apple: AppleCare log collection](https://support.apple.com/en-ca/guide/deployment/depd638aa061/1/web/1.0#dep99ac06bf1)
- [Apple: TriggerEnhancedLogCollection command](https://developer.apple.com/documentation/devicemanagement/triggerenhancedlogcollectioncommand)

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="jp-cpe">
<meta name="authorFullName" value="Jonathan Porter">
<meta name="publishedOn" value="2026-08-14">
<meta name="articleTitle" value="Trigger AppleCare log collection with a custom MDM command">
<meta name="description" value="Learn how to trigger AppleCare enhanced log collection on a supervised iPhone or iPad with a custom MDM command in Fleet.">
