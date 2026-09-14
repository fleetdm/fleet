## ADDED Requirements

### Requirement: Orbit advertises the PIN capability on Windows
Orbit on Windows SHALL include `windows_bitlocker_pin` in the `X-Fleet-Capabilities` header of every request to the Fleet server. Orbit on other platforms SHALL NOT.

#### Scenario: Windows orbit polls config
- **WHEN** a Windows orbit with this change sends its config request
- **THEN** the capabilities header contains `windows_bitlocker_pin`

### Requirement: Orbit fetches and applies a pending PIN under the BitLocker mutex
When the orbit config carries `bitlocker_pin_request_pending = true`, the Windows BitLocker config receiver SHALL acquire the same mutex used by the encryption and protection-restore paths, fetch the PIN from `POST /api/fleet/orbit/disk_encryption_pin/request`, apply it, and report the outcome to `POST /api/fleet/orbit/disk_encryption_pin`. If the mutex is held, the receiver SHALL skip this poll and try again on the next one.

#### Scenario: Pending request while the volume is idle
- **WHEN** the config flags a pending request and no other BitLocker work holds the mutex
- **THEN** orbit fetches the PIN, applies it, reports the outcome, and returns without blocking other receivers

#### Scenario: Encryption in progress
- **WHEN** the config flags a pending request while the encryption path holds the mutex
- **THEN** orbit does not fetch the PIN on this poll and leaves the request `pending` for the next poll

### Requirement: Preconditions before touching the volume
Before adding a protector, orbit SHALL verify all of: the host is not Windows Server; volume `C:` reports `FullyEncrypted`; protection status is on; no key protector of type 4 (TPM+PIN) or 6 (TPM+PIN+startup key) exists. If any check fails, orbit SHALL report `failed` with a reason naming the failed check and SHALL NOT call `ProtectKeyWithTPMAndPIN`.

#### Scenario: PIN already present
- **WHEN** a type 4 protector already exists on `C:`
- **THEN** orbit reports `failed` with "PIN already set" and makes no changes

#### Scenario: Protection suspended
- **WHEN** protection status on `C:` is off
- **THEN** orbit reports `failed` with a reason that protection is off and makes no changes

### Requirement: Apply the TPM+PIN protector and make it effective
Orbit SHALL call `Win32_EncryptableVolume.ProtectKeyWithTPMAndPIN` on `C:` with a nil friendly name, the default platform validation profile, and the fetched PIN; SHALL verify that `GetKeyProtectors(4)` now returns at least one ID; SHALL then delete every protector of type 1 (TPM only) by ID; and SHALL verify that no type 1 protector remains before reporting `set`.

#### Scenario: Volume with TPM and recovery password protectors
- **WHEN** `C:` has a type 1 and a type 3 protector and a valid PIN is applied
- **THEN** the volume ends with a type 4 and the type 3 protector, no type 1, and orbit reports `set`

#### Scenario: Add succeeds but TPM-only deletion fails
- **WHEN** `ProtectKeyWithTPMAndPIN` succeeds but deleting the type 1 protector returns an error
- **THEN** orbit deletes the type 4 protector it just added, verifies the protector list matches the pre-call state, and reports `failed` with the deletion error

### Requirement: Windows error codes map to actionable reasons
Orbit SHALL map `FVE_E_POLICY_INVALID_PIN_LENGTH` (0x80310068), `FVE_E_INVALID_PIN_CHARS` (0x8031009A), `FVE_E_PROTECTOR_EXISTS` (0x80310031), `TBS_E_SERVICE_NOT_RUNNING` (0x80284008), `FVE_E_LOCKED_VOLUME` (0x80310000), `FVE_E_BOOTABLE_CDDVD` (0x80310030) and `FVE_E_FOREIGN_VOLUME` (0x80310023) to human-readable reasons in `client_error`, and SHALL fall back to the existing `fveErrorCode` formatting for any other code.

#### Scenario: Policy requires a longer PIN
- **WHEN** `ProtectKeyWithTPMAndPIN` returns 0x80310068
- **THEN** orbit reports `failed` with a reason stating the PIN is shorter than the minimum length required by policy, including the code

### Requirement: The PIN never leaves memory except to the WMI call
Orbit SHALL hold the PIN only in memory for the duration of the apply sequence, SHALL zero it afterwards, and SHALL NOT log it at any level, include it in error strings, or write it to disk.

#### Scenario: Debug logging enabled
- **WHEN** orbit runs with debug logging during a PIN apply
- **THEN** no log line contains the PIN value

### Requirement: Orbit registers the Fleet Desktop AppUserModelID
Orbit on Windows SHALL, at startup and idempotently, ensure the registry key `HKLM\Software\Classes\AppUserModelId\FleetDM.FleetDesktop` exists with `DisplayName = "Fleet Desktop"` and `IconUri` set to the installed Fleet Desktop icon path. Failure to write the key SHALL be logged at warn level and SHALL NOT prevent orbit from starting.

#### Scenario: Fresh install
- **WHEN** orbit starts on a Windows host where the key does not exist
- **THEN** the key and both values are created

#### Scenario: Key already present
- **WHEN** orbit starts and the key already has the expected values
- **THEN** orbit makes no registry writes
