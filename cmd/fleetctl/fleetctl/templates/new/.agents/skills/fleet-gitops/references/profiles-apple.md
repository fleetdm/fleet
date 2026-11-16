# Apple profiles and declarations (macOS, iOS, iPadOS)

Both kinds are listed under `controls.apple_settings.configuration_profiles` in a fleet file (or `default.yml` for unassigned hosts), scoped with labels:

```yaml
controls:
  apple_settings:
    configuration_profiles:
      - path: ../lib/macos/configuration-profiles/screen-lock.mobileconfig
        labels_exclude_any:
          - macOS screen lock exclusions
      - path: ../lib/macos/declaration-profiles/Passcode settings.json
      - paths: ../lib/macos/configuration-profiles/*.mobileconfig   # options apply to every match
```

`name:` and `description:` on an entry override what Fleet shows; `activation: <path>` attaches a custom activation to a declaration. Fleet builds the default `com.apple.activation.simple` for every declaration, so you never write one.

## Choose the delivery method

- **Declaration (DDM `.json`)** when Apple publishes a `com.apple.configuration.*` type for the setting: passcode, software update settings, disk management, services/background tasks, app managed configuration, certificates and identities, Wi-Fi, VPN, DNS, security settings, and more. Declarations report their own status and take precedence over overlapping legacy payloads, so prefer them where available and don't ship the same setting both ways.
- **Configuration profile (`.mobileconfig`)** for everything else: MCX managed preferences for third-party apps, restrictions, PPPC, system extensions, login window, Safari, and most first-party payloads.
- Never write what Fleet manages for you: disk encryption (`enable_disk_encryption`), OS update enforcement (`macos_updates`, `ios_updates`, `ipados_updates`), the fleetd configuration profile, the Fleet root CA, or Setup Assistant (`setup_experience`). Fleet rejects their payload identifiers, names, and declaration types.

## Writing a `.mobileconfig`

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key><string>com.apple.systempolicy.control</string>
      <key>PayloadIdentifier</key><string>com.example.gatekeeper.systempolicy</string>
      <key>PayloadUUID</key><string>38554FF7-80E0-4EF1-BA44-7DE20B407797</string>
      <key>PayloadVersion</key><integer>1</integer>
      <key>PayloadDisplayName</key><string>Gatekeeper</string>
      <key>EnableAssessment</key><true/>
      <key>AllowIdentifiedDevelopers</key><true/>
    </dict>
  </array>
  <key>PayloadType</key><string>Configuration</string>
  <key>PayloadIdentifier</key><string>com.example.gatekeeper</string>
  <key>PayloadUUID</key><string>CA95CACF-8D79-4D6F-B804-653B90D7E437</string>
  <key>PayloadVersion</key><integer>1</integer>
  <key>PayloadDisplayName</key><string>Enable Gatekeeper</string>
  <key>PayloadDescription</key><string>Requires apps to be signed by identified developers.</string>
  <key>PayloadOrganization</key><string>Example Corp</string>
  <key>PayloadScope</key><string>System</string>
  <key>PayloadRemovalDisallowed</key><true/>
</dict>
</plist>
```

- The top-level `PayloadType` is `Configuration`. `PayloadDisplayName` is the name Fleet shows and must be unique within the fleet across all profile types. `PayloadIdentifier` is reverse-DNS with the repo's prefix and unique in the repo; two profiles sharing one replace each other on the device. Every `PayloadUUID` is a fresh `uuidgen` value. Each payload in `PayloadContent` needs its own `PayloadType`, `PayloadIdentifier`, `PayloadUUID`, and `PayloadVersion`.
- **Keys come from the schema, not from memory.** First-party payloads: https://github.com/apple/device-management/tree/release/mdm/profiles, one YAML per `PayloadType` (fetch the raw file, for example `.../release/mdm/profiles/com.apple.systempolicy.control.yaml`, and read `payloadkeys` for names, types, allowed values, and the OS versions they apply to). Third-party and app preference domains: https://github.com/ProfileManifests/ProfileManifests (`Manifests/ManifestsRoot/<domain>.plist`), the same manifests ProfileCreator and iMazing use. Third-party settings go in a `com.apple.ManagedClient.preferences` payload (`PayloadContent` → domain → `Forced` → `mcx_preference_settings`), the shape the repo's existing MCX profiles show.
- `PayloadScope: System` for device settings; `User` for user-channel payloads (Fleet delivers those to the managed user). iOS and iPadOS profiles have no scope.
- `$FLEET_VAR_HOST_END_USER_IDP_USERNAME`, `$FLEET_VAR_HOST_HARDWARE_SERIAL`, `$FLEET_SECRET_NAME`, and friends may appear in string values; Fleet substitutes and XML-escapes them at delivery. A profile with `$FLEET_SECRET_` is skipped by the dry run, and placeholders inside `<data>` make `plutil` and contour report an error you can ignore after confirming it's only the placeholder.

## Writing a declaration

```json
{
  "Type": "com.apple.configuration.passcode.settings",
  "Identifier": "com.example.config.passcode.settings",
  "Payload": {
    "RequireAlphanumericPasscode": true,
    "MinimumLength": 12,
    "MaximumInactivityInMinutes": 15
  }
}
```

- `Type` must start with `com.apple.configuration.` or `com.apple.management.`; status subscriptions, `com.apple.configuration.package`, watch enrollment, Google accounts, and server capabilities are rejected, and software update enforcement belongs to `macos_updates`. Asset declarations (`com.apple.asset.*`) are allowed and conventionally stored in an `assets/` folder.
- `Identifier` is reverse-DNS, unique across the repo, and at most 64 bytes. Tools that derive it from the type's last component collide (`passcode.settings` and `softwareupdate.settings` both become `…settings`), so set it yourself.
- `Payload` keys come from https://github.com/apple/device-management/tree/release/declarative/declarations (fetch the raw YAML for the type, for example `.../release/declarative/declarations/configurations/passcode.settings.yaml`). The name Fleet shows is the file name, so name the file for a human.
- `ServerToken` is managed by Fleet; leave it out.

## Validate

```bash
plutil -lint path/to/profile.mobileconfig                 # XML plist well-formed (macOS)
python3 -c 'import json,sys; json.load(open(sys.argv[1]))' "path/to/decl.json"
python3 <skill>/scripts/validate.py                       # structure, reserved ids, duplicates
contour profile validate --strict path/to/profile.mobileconfig   # payload keys and values vs Apple's schema
contour profile ddm validate path/to/declarations/        # declaration keys vs Apple's schema
```

contour (https://github.com/macadmins/contour, a community tool) embeds Apple's schema and is the only offline check for invented or mistyped payload keys; without `--strict` it accepts unknown keys. Use it when installed; otherwise read the schema file for every key you set and say in your report that key validation was manual. Then the dry run: Fleet checks the plist, reserved identifiers and names, declaration types, and name conflicts within the fleet.

Duplicates across the repo, which Fleet only catches per fleet:

```bash
for f in $(find . -name '*.mobileconfig'); do plutil -extract PayloadIdentifier raw -o - "$f"; done | sort | uniq -d
jq -r .Identifier $(find . -path '*declaration*' -name '*.json') | sort | uniq -d
```

## Verify on a device

Hosts > a host > OS settings shows each profile as Verified or Failed with Apple's error text. For a setting you can query, confirm it with `macos_profiles` (identifier installed) or `managed_policies` (MCX value applied).
