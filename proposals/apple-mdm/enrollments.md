# Enrollments

"Enrollments" hold settings for devices that will be enrolled to MDM.
The MDM "enrollments" will allow Fleet to automatically enroll devices to specific teams, which then allows for applying specific MDM settings (depending on the team).

For Dogfood-MVP, Fleet will allow creating global enrollments only (team support will be added at a subsequent iteration).
Users will be able to create the two following types of enrollments:
- Global manual enrollment
- Global DEP enrollment

We'll have a new `apple_enrollments` table with the following fields:
- `id` (used to deduce an "Enroll URL")
- `name`
- `config JSON`: holds enrollment config like "PayloadDisplayName", "AccessRights", TODO(lucas): Define with Guillaume if really necessary.
- `dep_config JSON`: holds DEP enrollment profile (`NULL` when enroll is manual).
- `team` (`0` for global)

## Fleetctl commands

Create automatic (DEP) enrollment:
`fleetctl apple-mdm enrollments create-automatic --name=Foo --config=<config.json> --profile=<dep_profile.json>`

TBD: `--config:` May not be necessary see TODO comment in `config JSON`. 

Here's a sample `dep_profile.json`:
```json
{
  "profile_name": "Acme Inc.",
  "allow`pairing": true,
  "auto_advance_setup": false,
  "await_device_configured": false,
  "configuration_web_url": "https://example.com", // <<<< Fleet will ignore and override this field.
  "url": "https://example.com",                   // <<<< Fleet will ignore and override this field.
  "department": "it@acme.com",
  "is_supervised": false,
  "is_multi_user": false,
  "is_mandatory": false,
  "is_mdm_removable": true,
  "language": "en",
  "org_magic": "1",
  "region": "US",
  "support_phone_number": "+1 408 555 1010",
  "support_email_address": "support@acme.com",
  "anchor_certs": [],
  "supervising_host_certs": [],
  "skip_setup_items": ["Accessibility", "Appearance", "AppleID", 
    "AppStore", "Biometric", "Diagnostics", "FileVault",
    "iCloudDiagnostics", "iCloudStorage", "Location", "Payment",
    "Privacy", "Restore", "ScreenTime", "Siri", "TermsOfAddress",
    "TOS", "UnlockWithWatch"
  ]
}
```

Create manual enrollment:
`fleetctl apple-mdm enrollments create-manual --name=Bar --config=<config.json>`
Returns the ID of the created enrollment.

Show enrollment settings:
`fleetctl apple-mdm enrollments get --id=<ENROLLMENT_ID>`

List enrollments:
`fleetctl apple-mdm enrollments list`

Delete enrollments:
`fleetctl apple-mdm enrollments delete --id=<ENROLLMENT_ID>`