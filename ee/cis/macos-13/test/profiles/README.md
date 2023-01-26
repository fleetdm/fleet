# CIS Profiles

On this directory we store the profiles for each CIS benchmark check that will allow us to apply them automatically on macOS VMs.

## How to create one

Let's assume you are creating a profile for CIS 1.6, "Ensure Install Security Responses and System Files Is Enabled".

1. Copy an existing profile:
```sh
cp compliance/profiles/2.1.1.3.mobileconfig compliance/profiles/1.6.mobileconfig
```

2. Generate two unique UUIDs:
```sh
$ uuidgen
380B8EF9-B5E8-4967-A102-52F78EA03AB9
$ uuidgen
3C4F942C-C716-48F3-A2E9-52AD7DBE55E0
```

3. Open the created copy with a text editor and modify the following fields:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadDisplayName</key>
			<string>test</string>
			<key>PayloadType</key>
			<string><!--- Domain of the setting, e.g. com.apple.SoftwareUpdate --></string>
			<key>PayloadIdentifier</key>
			<string>com.fleetdm.cis-1.6.check <!--- This must be unique and a sub domain of the main profile, thus we add the cis number at the end + ".check" --></string>
			<key>PayloadUUID</key>
			<string><!--- Paste one of the generated UUID here, in this case 380B8EF9-B5E8-4967-A102-52F78EA03AB9 --></string>
			<key><!--- Setting, e.g. CriticalUpdateInstall --></key>
			<false/>
		</dict>
	</array>
	<key>PayloadDescription</key>
	<string>test</string>
	<key>PayloadDisplayName</key>
	<string><!-- Title of the CIS here, e.g. Ensure Install Security Responses and System Files Is Enabled --></string>
	<key>PayloadIdentifier</key>
	<string>com.fleetdm.cis-1.6</string> <!--- This must be unique, thus we add the cis number at the end -->
	<key>PayloadRemovalDisallowed</key>
	<false/>
	<key>PayloadScope</key>
	<string>System</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string><!--- Paste the other generated UUID here, in this case 3C4F942C-C716-48F3-A2E9-52AD7DBE55E0 --></string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
```

4. Place the `.mobileconfig` on the VM and double click the profile.
5. Go to `System Settings > Profiles` and then install the profile.