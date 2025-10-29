# Provide end user email address w/o relying on end user

In issue #15057, we introduced a feature primarily intended for internal use
that allows the specification of the email for the "Used by" field on the host
details page. This feature supports macOS and Windows, but the method to define
the email differs for each operating system.

### macOS via configuration profiles

When `fleetd` starts in macOS, it'll check for the user email in the enrollment profile.

The enrollment profile contains multiple `<dict>` elements. Locate the one with a
`PayloadIdentifier` equal to `com.fleetdm.fleet.mdm.apple.mdm`, and add the
following, replacing foo@example.com with the desired user email:

```xml
<key>EndUserEmail</key>
<string>foo@example.com</string>
```

For instance, the specific part of the payload might look like this:

```xml
<key>PayloadIdentifier</key>
<string>com.fleetdm.fleet.mdm.apple.mdm</string>
<key>EndUserEmail</key>
<string>foo@example.com</string>
```

### Windows via custom installers

For Windows, we implemented a hidden flag in `fleetctl` that allows you to
define a custom user email when a package is built.

If provided, `fleetd` will report the user email to the fleet server when it starts.

```
$ fleetctl package --type=msi --end-user-email=foo@example.com --fleet-url=https://test.example.com --enroll-secret=abc
```

Note: You need to build a different package per user email.

<meta name="pageOrderInSection" value="1202">
