# Tines Stories

## Best practices

These are best practices to be followed by Fleeties when creating and sharing Stories.

- Include a note with what credentials/resources need to be set before running.
- Remove your email from the monitoring section.
- If actions are device-specific, prefix the Story with the platform. Examples:
  - `windows-story.json`
  - `mac-update.json`
  - `apple-device-location.json` — If the command applies to more than just Macs, use `apple` as the prefix.
- After exporting, open the .json file and:
  - Double-check to make sure your Fleet email (or any other confidential information) isn't in the contents.
  - Add an empty newline at the bottom of the file.


## [Apple device location](apple-device-location.json)

A custom MDM command sent through Fleet to get a device's location. It pauses for 30 seconds after sending the command, so that the device can report back its location before Fleet attempts to retrieve it.

A device needs to be [locked](#lock-commands) (known as "Lost Mode" in the Apple world) before the location can be requested.

Apple's documentation for this command can be found at [Device Location](https://developer.apple.com/documentation/devicemanagement/device-location-command).

Originally created for `customer-reedtimmer`.


## [Lock commands](lock-commands.json)

Fleet API commands to lock and unlock hosts.

Originally created for `customer-reedtimmer`.

## [MDM Migration (Jamf Pro)](mdm-migration-jamf-pro.json)

Used when migrating from Jamf Pro to Fleet. This receives the migration web hook from Fleet, looks up that serial number in Jamf Pro via Jamf's API, then sends an unenroll command from Jamf.

