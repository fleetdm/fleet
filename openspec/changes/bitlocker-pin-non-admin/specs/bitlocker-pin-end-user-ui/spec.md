## ADDED Requirements

### Requirement: Create PIN modal collects and validates the PIN
On the My device page for a Windows host with `disk_encryption.action_required = "create_pin"` and `fleetd_can_set_pin = true`, the **Create PIN** action SHALL open a modal titled "Create PIN" with the intro "Set a BitLocker PIN to protect your data if this host is lost or stolen. You'll need to enter it each time your host starts up.", a masked "BitLocker PIN" field with helper text "Must be 6–20 digits. Keep it somewhere safe. This PIN isn't saved by Fleet or your IT team.", a masked "Confirm PIN" field, a "Cancel" button and a primary "Save" button. Save SHALL be disabled until both fields contain the same 6 to 20 ASCII digits.

#### Scenario: Matching valid PINs
- **WHEN** the user enters "123456" in both fields
- **THEN** Save is enabled

#### Scenario: Mismatch
- **WHEN** the two fields differ
- **THEN** Save is disabled and the Confirm PIN field shows an error that the PINs do not match

#### Scenario: Invalid characters or length
- **WHEN** the BitLocker PIN field contains a non-digit, fewer than 6 or more than 20 characters
- **THEN** Save is disabled and the field shows an error restating the 6 to 20 digit rule

### Requirement: Save submits the PIN and waits for the agent
Clicking Save SHALL call `POST /api/_version_/fleet/device/{token}/disk_encryption_pin`, then show a waiting state ("Setting your PIN..." with the Save button in its loading state and both fields disabled) and poll `GET /api/_version_/fleet/device/{token}` every 3 seconds, reading `mdm.os_settings.disk_encryption.pin_request.status`, for up to 90 seconds.

#### Scenario: Agent sets the PIN
- **WHEN** polling observes `pin_request.status = "set"` or `action_required` is no longer `create_pin`
- **THEN** the modal closes, a success toast "Successfully set PIN." is shown, and the disk encryption banner is no longer rendered

#### Scenario: Agent reports failure
- **WHEN** polling observes `pin_request.status = "failed"` with `error` "PIN already set"
- **THEN** the modal stays open, shows the error toast "Couldn't set PIN. PIN already set. Try again or contact your IT admin.", and re-enables the fields and Save

#### Scenario: No result within 90 seconds
- **WHEN** polling reaches 90 seconds without `set` or `failed`
- **THEN** the modal shows "Couldn't set PIN. Fleet did not hear back from this device. Try again or contact your IT admin." and re-enables Save

#### Scenario: Submission rejected by the server
- **WHEN** the submit call returns 400 or 422
- **THEN** the modal shows "Couldn't set PIN. {server error}. Try again or contact your IT admin." and re-enables Save

### Requirement: Deep link opens the modal
When the My device page loads with `?create_pin=1` in the query string and the host's `action_required` is `create_pin`, the page SHALL open the Create PIN modal automatically (the legacy instructions modal when `fleetd_can_set_pin` is false). When `action_required` is not `create_pin`, the parameter SHALL be ignored.

#### Scenario: Toast button
- **WHEN** the user arrives from the Fleet Desktop toast at `/device/{token}?create_pin=1` and a PIN is still needed
- **THEN** the Create PIN modal is open on first render

### Requirement: Legacy instructions modal for old fleetd
When `fleetd_can_set_pin` is false, the **Create PIN** action SHALL open today's instructions modal (Manage BitLocker steps ending with Refetch) unchanged.

#### Scenario: Host on fleetd without the capability
- **WHEN** the device host response has `fleetd_can_set_pin = false`
- **THEN** clicking Create PIN shows the five-step instructions modal and no PIN form

### Requirement: Updated disk encryption banners
The My device banner for `action_required = "create_pin"` SHALL read "Disk encryption: Create a BitLocker PIN to protect your data if this host is lost or stolen." with a **Create PIN** link that opens the modal. The Host details (admin) banner for the same state SHALL read "Disk encryption: The end user needs to create a BitLocker PIN. Fleet Desktop prompts them at each login." with no action link. Both SHALL disappear once `action_required` is no longer `create_pin`.

#### Scenario: End user banner
- **WHEN** a Windows host on the My device page has `action_required = "create_pin"`
- **THEN** the yellow banner shows the new copy and a Create PIN link

#### Scenario: Admin banner
- **WHEN** an admin views Host details for the same host
- **THEN** the yellow banner shows the admin copy and no link

### Requirement: Activity feed renders the new activity
The global activity feed and the host activity tab SHALL render `created_disk_encryption_pin` as "**End user** created a disk encryption PIN for **{host_display_name}**." with the host name linking to the host when the host still exists.

#### Scenario: Feed row
- **WHEN** an activity `created_disk_encryption_pin` with `host_display_name = "MU-TH-UR"` is listed
- **THEN** the row reads "End user created a disk encryption PIN for MU-TH-UR." with the standard timestamp
