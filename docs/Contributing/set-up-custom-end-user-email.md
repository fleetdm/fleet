# Provide end user email address w/o relying on end user

In #15057 we introduced a feature intended for internal usage that allows to
define the email under the "Used by" field in the host details page.

Supported operating systems are macOS and Windows, and the method use to define
the email varies.

### macOS via configuration profiles

When `fleetd` starts in macOS, it'll check for the user email in the enrollment profile.

The enrollment profile contains multiple `<dict>` elements, find the one with a
`PayloadIdentifier` equal to `com.fleetdm.fleet.mdm.apple.mdm`, and add the
following, replacing foo@example.com with your desired user email:


```xml
<key>EndUserEmail</key>
<string>foo@example.com</string>
```

For example, this is how the specific bit of the payload might look like:

```xml
<key>PayloadIdentifier</key>
<string>com.fleetdm.fleet.mdm.apple.mdm</string>
<key>EndUserEmail</key>
<string>foo@example.com</string>
```

### Windows via custom installers

For Windows, we built a hidden flag into `fleetctl` that allows you to define a custom user email.

If provided, `fleetd` will report the user email to the fleet server when it starts.

```
$ fleetctl package --type=msi --end-user-email=foo@example.com --fleet-url=https://test.example.com --enroll-secret=abc
```

You'll need to build a different package per user email.
