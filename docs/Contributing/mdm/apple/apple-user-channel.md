# Apple MDM user channel (research)

The user channel in Apple MDM is a specific communication channel for user-level authentication and management. It is like having another device with its own commands. The user channel is primarily identified by the `UserAuthenticate` message type in the MDM protocol, which handles user authentication requests. Most MDMs use zero-length DigestChallenge by default (no additional authentication) unless advanced authentication is configured. The `UserAuthenticate` command is supported by the nanomdm code. See [demo image](https://github.com/fleetdm/fleet/issues/28798#issuecomment-2867571898).

## Check-in

The user channel is only active when the user is logged in. This means that's the only time any sort of user profiles can be sent. When the user first logs in, a user check-in message is sent to the MDM server.

Sample user check-in message:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>NotOnConsole</key>
	<false/>
	<key>Status</key>
	<string>Idle</string>
	<key>UDID</key>
	<string>980E3D0B-5A28-537C-B7CB-7AFD15AFB933</string>
	<key>UserID</key>
	<string>23B1AB9F-5A19-4919-8610-0A74091497E1</string>
	<key>UserLongName</key>
	<string>Victor Smith</string>
	<key>UserShortName</key>
	<string>victor-smith</string>
</dict>
</plist>
```

With the above check-in message, the MDM server sees the ID as UDID:UserID (980E3D0B-5A28-537C-B7CB-7AFD15AFB933:23B1AB9F-5A19-4919-8610-0A74091497E1), so it needs to link any commands or configuration profiles to that ID.

## User scoped profiles

The MDM command payload must be marked user-scoped and have a UserID, like:
```xml
    <key>PayloadScope</key>
    <string>User</string>
    <key>UserID</key>
    <string>23B1AB9F-5A19-4919-8610-0A74091497E1</string>
```

The top-level configuration profile must also include:
```xml
<key>PayloadScope</key>
<string>User</string>
```

## Pushing user commands to device

The user channel has its own push magic, which is delivered via `TokenUpdate` command. We are currently collecting and storing these in the `nano_enrollments` table. The main difference between device push and user push is the topic has a `.user` suffix, like `com.apple.mgmt.External.XXXXXXXX.user`

## Considerations for product

- Will we support multiple users?
  - No. We/Apple do not support multiple users managed by MDM.
- We plan to link IdP groups to labels. This means those labels/groups will apply to user-scoped profiles as well.
  - Yes
- Will we support [Shared iPad](https://support.apple.com/guide/deployment/shared-ipad-overview-dep9a34c2ba2/1/web/1.0)
  - Not right now. Shared iPad is mostly used in education and our customers do not require it right now.
- Will we support user-scoped DDM?
  - Initial focus is to support `.mobileconfig` since the primary use case is to deliver certificates.
- Will we allow IT admin to select in the UI whether a profile is device or user-scoped? Or rely on the PayloadScope of the top-level configuration profile?
  - Rely on PayloadScope since we recommend iMazing Profile Editor.
- What about security--do we need to guarantee that the cert is only delivered to a specific user?