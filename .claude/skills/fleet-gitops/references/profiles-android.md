# Android profiles

An Android profile is a JSON object containing fields of the Android Management API `Policy` resource. List it under `controls.android_settings.configuration_profiles` in a fleet file; the file name is the profile name and must be unique within the fleet.

```yaml
controls:
  android_settings:
    configuration_profiles:
      - path: ../lib/android/configuration-profiles/camera-disabled.json
        labels_include_any:
          - Field technicians
```

```json
{
  "cameraAccess": "CAMERA_ACCESS_DISABLED",
  "advancedSecurityOverrides": {
    "developerSettings": "DEVELOPER_SETTINGS_DISABLED"
  }
}
```

Keep one topic per file, and don't set the same key in two profiles that target the same hosts; which value wins isn't something you can see in the YAML.

## Keys and values come from the Policy schema

Google publishes the schema as a discovery document; the `Policy` object in it is the complete surface, with every field's type, the shape of nested objects, and for enum fields the exact allowed values:

```
https://androidmanagement.googleapis.com/$discovery/rest?version=v1
```

Fetch it (or `python3 <skill>/scripts/validate.py` fetches and caches it) and read the field before you set it. Prose descriptions: https://developers.google.com/android/management/reference/rest/v1/enterprises.policies.

Two things to get right that nothing local enforces:

- **Enum values are strings and aren't checked by Fleet.** `"camera_access_disabled"` and an invented value both pass the dry run and fail when Google receives the policy. Copy the enum out of the schema, case included, and quote it in your report.
- **Unknown keys nested inside an object are dropped, not rejected.** A typo at the top level fails by name; the same typo inside `advancedSecurityOverrides` is silently discarded and the setting never applies.

Fleet rejects, by name, the fields it manages elsewhere or doesn't support: `applications`, `appFunctions`, `playStoreMode`, `installAppsDisabled`, `uninstallAppsDisabled`, `blockApplicationsEnabled`, `appAutoUpdatePolicy` (software is managed under `software.app_store_apps`), `statusReportingSettings` (use host vitals), `kioskCustomLauncherEnabled`, `kioskCustomization`, `persistentPreferredActivities` (only personal hosts are supported today), `setupActions`, and `encryptionPolicy`. `systemUpdate` needs Fleet Premium. Per-app managed configuration (`managedConfiguration`, `workProfileWidgets`, `credentialProviderPolicy`) goes in the app's `configuration.path` under `app_store_apps`, not in a profile. Certificates go under `android_settings.certificates`.

## Validate

```bash
python3 -c 'import json,sys; json.load(open(sys.argv[1]))' path/to/profile.json
python3 <skill>/scripts/validate.py    # reserved keys, unknown top-level keys, enum values against the schema when online
```

Then the dry run: Fleet validates the JSON against Google's `Policy` type, so an unknown top-level key, a wrong type, a reserved field, or a Premium-only field fails with the field named. Enum values and nested keys remain yours to check against the schema.

## Verify on a device

Hosts > a host > OS settings shows the profile status. After a configuration change the install status reads Pending until the device syncs the new policy.
