## ADDED Requirements

### Requirement: Toast shown once per login while a PIN is needed
On Windows, when the Fleet Desktop summary returns `notifications.needs_bitlocker_pin = true`, Fleet Desktop SHALL show a Windows toast at most once per Fleet Desktop process lifetime. Because orbit starts Fleet Desktop for each interactive login, this yields one toast per login until the PIN is set. When the summary omits the flag, Fleet Desktop SHALL NOT show the toast.

#### Scenario: First summary after login
- **WHEN** Fleet Desktop starts, fetches its summary, and `needs_bitlocker_pin` is true
- **THEN** exactly one toast is shown for this process

#### Scenario: Flag stays true across the 5 minute poll
- **WHEN** later summary polls in the same process still return `needs_bitlocker_pin = true`
- **THEN** no additional toast is shown

#### Scenario: PIN set
- **WHEN** a summary omits `needs_bitlocker_pin`
- **THEN** no toast is shown and any existing Fleet PIN toast is removed from Action Center

### Requirement: Toast content and action
The toast SHALL display the heading "Set your BitLocker PIN to protect this device", the body "Your IT team requires a BitLocker PIN on this device. Set yours now to keep your files safe if your device is ever lost.", and one button labelled "Create PIN" that uses protocol activation to open the My device URL for the current device token with `?create_pin=1` appended. The header row SHALL show the Fleet Desktop name and icon from the registered AppUserModelID.

#### Scenario: User clicks Create PIN
- **WHEN** the user clicks the toast button
- **THEN** the default browser opens `https://<fleet>/device/<current token>?create_pin=1` and no Fleet Desktop code runs for the click

### Requirement: Toast survives device token rotation
The toast SHALL carry a fixed tag and group so re-posting replaces the previous instance, SHALL be re-posted with the new URL when Fleet Desktop observes that the device token changed while `needs_bitlocker_pin` is still true, and SHALL carry an expiration so an ignored toast is removed from Action Center before its URL can go stale.

#### Scenario: Token rotates with the toast still pending
- **WHEN** the device token file changes and the last summary said a PIN is needed
- **THEN** Fleet Desktop posts the toast again with the new token and the earlier toast is replaced rather than duplicated

### Requirement: Toast delivery mechanism and identity
Fleet Desktop SHALL post the toast through WinRT `Windows.UI.Notifications.ToastNotificationManager` invoked from a hidden, non-interactive PowerShell process, using the `FleetDM.FleetDesktop` AppUserModelID when its registry key exists and falling back to the built-in Windows PowerShell AppUserModelID otherwise. Failure to show the toast SHALL be logged and SHALL NOT affect any other Fleet Desktop behavior.

#### Scenario: AUMID registered by orbit
- **WHEN** `HKLM\Software\Classes\AppUserModelId\FleetDM.FleetDesktop` exists
- **THEN** the toast appears under "Fleet Desktop" with its icon

#### Scenario: AUMID missing
- **WHEN** the key does not exist
- **THEN** the toast is posted under the Windows PowerShell AppUserModelID and still appears

#### Scenario: PowerShell blocked
- **WHEN** the PowerShell process fails or is blocked by policy
- **THEN** Fleet Desktop logs the error at warn level and continues; the My device banner remains the path to the modal

### Requirement: No toast on other platforms or without the capability
Fleet Desktop on macOS and Linux SHALL ignore `needs_bitlocker_pin`. The server SHALL only set the flag for hosts whose fleetd advertised `windows_bitlocker_pin`, so hosts on older fleetd never receive a toast.

#### Scenario: Old fleetd
- **WHEN** the host's Fleet Desktop is new but its orbit never advertised the capability
- **THEN** the summary omits `needs_bitlocker_pin` and no toast is shown
