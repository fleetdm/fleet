# Disk encryption architecture

This document provides an overview of Fleet's Disk Encryption architecture for MDM.

## Introduction

Disk Encryption in Fleet's MDM allows for securing device data by encrypting the storage media. This document provides insights into the design decisions, system components, and interactions specific to the Disk Encryption functionality.

## Architecture overview

The Disk Encryption architecture leverages platform-specific encryption technologies (FileVault for macOS, BitLocker for Windows, LUKS via LVM for Linux) to encrypt device storage and securely manage recovery keys.

## Key components

- **Encryption Configuration**: Settings and policies for configuring disk encryption.
- **Key Management**: Secure storage and retrieval of encryption keys.
- **Verification**: Mechanisms to verify the encryption status of devices.
- **Recovery**: Processes for recovering access to encrypted devices.

## Architecture diagram

See platform-specific diagrams below

## Platform-specific implementation

### FileVault (macOS)

For macOS hosts without disk encryption enabled, encryption involves a two-step process:

1. Sending a profile with two payloads:
   - A Payload to configure how the disk is going to be encrypted
   - A Payload to configure the escrow of the encryption key

2. Retrieving the disk encryption key:
   - Via osquery, Fleet grabs the (encrypted) disk encryption key
   - In a cron job, Fleet verify that it is able to decrypt the key

```mermaid
sequenceDiagram
        actor Admin
        participant fleet as Fleet server
        participant host as macOS Host
        participant fleetd as orbit
        participant desktop as Fleet Desktop
        actor user as End User
        Admin->>fleet: Enable disk encryption
        host->>fleet: Enroll in Fleet MDM
        fleet->>host: Encryption and<br>escrow profile installed
        fleet->>host: Orbit/osquery installed
        host->>host: Enable Filevault
        desktop->>user: prompt user to logout
        user->>host: logout/login or create<br>initial user during setup
        host->>host: Store recovery key at <br>/var/db/filevaultprk.dat
        fleetd->>fleet: request vitals reports
        fleet->>fleetd: Return vitals reports including query<br>to read /var/db/filevaultprk.dat
        fleetd->>fleetd: execute reports
        fleetd->>fleet: return report data including recovery key
        fleet->>fleet: Verify that recovery key is decryptable<br>(hourly cron job)
```

If Fleet is not able to decrypt the key for a host, or there is no stored key for an already encrypted host, the key needs to be rotated. Rotation happens silently by:

1. The server sends a notification to orbit to rotate the disk encryption recovery key
2. orbit installs and enables an authorization plugin named [Escrow Buddy](https://github.com/macadmins/escrow-buddy) that performs the key rotation the next time the user logs in.
3. Escrow Buddy stores the encrypted key on disk after the next login
4. Fleet, via osquery, retrieves the (encrypted) recovery key
5. A cron job verifies that the retrieved key can be decrypted

```mermaid
sequenceDiagram
        actor Admin
        participant fleet as Fleet server
        participant host as macOS Host
        participant fleetd as orbit
        participant desktop as Fleet Desktop
        actor user as End User
        user->>host: Enable filevault
        Admin->>fleet: Enable disk encryption
        host->>fleet: Enroll in Fleet MDM
        fleet->>host: Encryption and<br>escrow profile installed
        fleet->>host: Orbit/osquery installed
        fleetd->>fleet: request vitals queries
        fleet->>fleetd: Return reports including encryption status
        fleetd->>fleet: return report data including encryption status
        fleet->>fleetd: Enable notifs.RunDiskEncryptionEscrow in orbit<br>config because Host is encrypted but no<br>key is escrowed
        fleetd->>host: Install Escrow Buddy
        fleetd->>host: Set Escrow Buddy<br>GenerateNewKey=true
        desktop->>user: prompt user to logout
        user->>host: logout/login
        host->>host: Store recovery key at <br>/var/db/filevaultprk.dat<br>(triggered by Escrow Buddy)
        fleetd->>fleet: request vitals reports
        fleet->>fleetd: Return vitals reports including query<br>to read /var/db/filevaultprk.dat
        fleetd->>fleetd: execute reports
        fleetd->>fleet: return report data including recovery key
        fleet->>fleetd: Disable notifs.RunDiskEncryptionEscrow in orbit<br>config because Host is encrypted and a<br>key is escrowed
        fleetd->>host: Set Escrow Buddy<br>GenerateNewKey=false
        fleet->>fleet: Verify that recovery key is decryptable<br>(hourly cron job)
```

#### Troubleshooting
The key stored in host_disk_encryption_keys for a given host will be deleted under the following circumstances:
- MDM re-enrollment or enrollment profile reinstallation, outside Fleet-initiated MDM SCEP certificate renewal
- Disk encryption disabled for a host's fleet
- Host moved to a fleet with disk encryption disabled

If the host is still in an encrypted fleet after the MDM re-enrollment, or in the case of a fleet
change, once the host is moved to a fleet with encryption enabled, Fleet will initiate one of the processes
described above, depending on the host's actual disk encryption status, to escrow a new encryption
key. Until the process is complete no key will be listed in Fleet, however in the event of an
emergency the latest key can be retrieved from the archive using the steps described in [Key Storage
and Security](#key-storage-and-security).

### BitLocker (Windows)

Disk encryption on Windows is performed entirely by orbit. 

When disk encryption is enabled, the server sends a notification to orbit, which calls the
[Win32_EncryptableVolume class](https://learn.microsoft.com/en-us/windows/win32/secprov/getencryptionmethod-win32-encryptablevolume)
to encrypt the used space of the disk with TPM and Numerical Password protectors and generate an encryption key.

If the disk is already encrypted, orbit rotates the recovery key: it adds a new Fleet-managed
Numerical Password protector, removes old recovery key protectors, and escrows the new key. The disk
is never decrypted. This matches how other MDM platforms handle pre-encrypted disks and avoids issues
with secondary drives that use BitLocker auto-unlock (which prevents decrypting the OS drive).

After the disk is encrypted (or the key is rotated), orbit sends the key back to the server using an
orbit-authenticated endpoint (`POST /api/fleet/orbit/disk_encryption_key`).

The server determines whether the disk is encrypted by checking both `conversion_status` (whether
the data is encrypted) and `protection_status` (whether the TPM protector is active) from the
osquery `bitlocker_info` table. If the disk is encrypted but protection is off (e.g., BitLocker
suspended for a BIOS update), the host shows "Action required" in the Fleet UI.

```mermaid
sequenceDiagram
        actor Admin
        participant fleet as Fleet server
        participant host as Windows Host
        participant fleetd as orbit
        participant desktop as Fleet Desktop
        actor user as End User
        Admin->>fleet: Enable disk encryption
        host->>fleet: Enroll in Fleet MDM
        fleet->>host: Orbit/osquery installed
        fleetd->>fleet: request vitals queries
        fleet->>fleetd: Return reports including encryption status<br>(protection_status and conversion_status)
        fleetd->>fleet: return report data including encryption status
        fleet->>fleetd: Enable notifs.EnforceBitLockerEncryption in orbit<br>config because Host is encrypted but no<br>key is escrowed or host is not encrypted
        alt Disk not encrypted
        fleetd->>host: Create TPM and Numerical Password protectors
        host->>fleetd: Return recovery key after creating protectors
        fleetd->>host: Encrypt OS volume
        else Disk already encrypted
        fleetd->>host: Add new Numerical Password protector
        host->>fleetd: Return new recovery key
        fleetd->>host: Remove old recovery key protectors
        fleetd->>host: Ensure TPM protector exists
        end
        fleetd->>fleet: Send recovery key
        fleetd->>fleet: request vitals reports
        fleet->>fleetd: Return vitals reports including query<br>to check encryption status
        fleetd->>fleetd: execute reports
        fleetd->>fleet: return report data including encryption status
        fleet->>fleetd: Disable notifs.EnforceBitLockerEncryption in orbit<br>config because Host is encrypted and a<br>key is escrowed
        fleet->>fleet: Verify that recovery key is decryptable<br>(hourly cron job)
```

#### Troubleshooting

A number of disk encryption related registry settings can interfere with Fleet's ability to encrypt the disk
and escrow the key in various ways, such as disallowing encryption methods used by Fleet or requiring
a non-standard TPM Platform Validation Profile. These settings can also potentially disallow disk
encryption entirely either explicitly or implicitly via conflicting settings. Because of the number
of settings and their possible values it is impractical to cover them all however, a good place to
begin investigating is the "Full Volume Encryption" policies key in the registry, which can be viewed
using the following query. Pay careful attention to keys like FDVEncryptionType and OSEncryptionType
which have been observed on customer systems causing conflicts:

`SELECT * FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\FVE\%%';`

Additionally, Fleet's disk encryption implementation requires a working TPM on Windows hosts. Hosts
without a TPM or with the TPM disabled via BIOS or other means will fail to have their disks
encrypted.

Finally, in some error scenarios the disk may be left in a state where Fleet has initialized the
encryption process but is unable to complete it. If you believe the conflicting settings have been
removed or properly modified the following command will ensure the disk is decrypted and in a clean
slate for orbit to attempt to encrypt the disk again:

`manage-bde -off C:`

### LUKS (Linux)
Fleet can escrow disk encryption keys for Ubuntu, Kubuntu, and Fedora Linux hosts that had LUKS2
encryption enabled during installation. Fleet will only initiate the escrow process on hosts running
eligible operating systems and for which encryption has already been enabled and will not prompt
users on ineligible hosts. Once the process is initiated, Fleet Desktop will prompt the user to
enter their encryption passphrase which is then used to create a new keyslot, the passphrase for
which orbit randomly generates and escrows on the server.

```mermaid
sequenceDiagram
        actor Admin
        participant fleet as Fleet server
        participant host as Linux Host
        participant fleetd as orbit
        participant desktop as Fleet Desktop
        actor user as End User
        user->>host: Encrypt disk during OS installation
        Admin->>fleet: Enable disk encryption
        host->>fleet: Enroll in Fleet
        fleet->>host: Orbit/osquery installed
        fleetd->>fleet: request vitals queries
        fleet->>fleetd: Return reports including encryption status
        fleetd->>fleet: return report data including encryption status
        fleet->>fleetd: Enable notifs.RunDiskEncryptionEscrow in orbit<br>config because Host is encrypted but no<br>key is escrowed or host is not encrypted
        fleetd->>desktop: Trigger user key escrow dialog
        desktop->>user: Prompt user to enter passphrase
        user->>desktop: Enter encryption passphrase
        desktop->>fleetd: Send user passphrase to orbit
        fleetd->>fleetd: Generate new random passphrase
        fleetd->>host: Use user passphrase to create new<br>keyslot with random passphrase
        fleetd->>fleet: Encrypt and send new passphrase
        fleet->>fleetd: Disable notifs.RunDiskEncryptionEscrow in orbit<br>config because Host is encrypted and a<br>key is escrowed
        fleet->>fleet: Verify that recovery key is decryptable<br>(hourly cron job)
```

#### TPM-backed FDE (Ubuntu 26 and later)

Newer Ubuntu releases offer TPM-backed full-disk encryption. The volume is still LUKS2/dm-crypt
underneath, but the LUKS key is sealed to the TPM and the disk unlocks automatically at boot, so by
default there is **no user passphrase**. snapd (via secboot) owns the LUKS key slots.

This means Fleet cannot escrow a key by adding a key slot with `cryptsetup luksAddKey` as it does for
the passphrase flow above: an externally added slot is untracked by snapd and can be removed when
snapd re-seals the key (for example on a kernel update). Instead, orbit detects that the volume is
snapd-managed (by inspecting the LUKS2 tokens) and escrows a dedicated, Fleet-owned snapd recovery
key created in its own named slot (`fleet-escrow`), leaving the user's install-time recovery key
untouched.

Because the disk auto-unlocks and there is no passphrase to collect, this happens **silently with no
end-user dialog**. orbit reports `key_type: "recovery_key"` in the escrow request
(`POST /api/fleet/orbit/luks_data`); recovery keys have no salt or key slot, so those columns are
empty/null in `host_disk_encryption_keys`. The escrowed recovery key is revealed to admins through
the same Host details flow as other platforms.

```mermaid
sequenceDiagram
        actor Admin
        participant fleet as Fleet server
        participant host as Linux Host (TPM FDE)
        participant fleetd as orbit
        participant snapd as snapd / snap-tpmctl
        actor user as End User
        user->>host: Enable TPM-backed FDE during OS installation
        Admin->>fleet: Enable disk encryption
        host->>fleet: Enroll in Fleet
        fleetd->>fleet: request vitals queries
        fleet->>fleetd: Return reports including encryption status
        fleetd->>fleet: return report data including encryption status
        fleet->>fleetd: Enable notifs.RunDiskEncryptionEscrow in orbit<br>config because Host is encrypted but no<br>key is escrowed
        fleetd->>snapd: Detect snapd-managed FDE (LUKS2 tokens)
        fleetd->>snapd: generate-recovery-key
        snapd->>fleetd: Return recovery key + transient key id
        fleetd->>snapd: add-recovery-key (enroll under fleet-escrow slot)
        fleetd->>snapd: check-recovery-key (validate before escrow)
        fleetd->>fleet: Encrypt and send recovery key<br>(key_type: recovery_key, no salt/key slot)
        fleet->>fleetd: Disable notifs.RunDiskEncryptionEscrow in orbit<br>config because Host is encrypted and a<br>key is escrowed
```

**Tooling.** orbit manages the snapd recovery key exclusively through the **snapd REST API over the
`/run/snapd.socket` unix socket** (`POST /v2/system-volumes`), which snapd 2.74 (shipping in Ubuntu
26.04) extended so management agents can enroll a dedicated, named recovery key. The flow is:
`generate-recovery-key` (synchronous; returns the key value and a transient key id) →
`add-recovery-key` (asynchronous; enrolls the key under the `fleet-escrow` name, falling back to
`replace-recovery-key` if the slot already exists) → `check-recovery-key` (validates before
escrow). These actions require root on the privileged socket, which orbit has, and the socket is
guaranteed present wherever TPM-backed FDE is in use — no network or snap store access needed.

orbit deliberately does **not** shell out to the
[`snap-tpmctl`](https://github.com/canonical/snap-tpmctl) CLI: that tool is itself just another
client of the same socket, is GPL-licensed (the socket is a stable, language-agnostic wire
protocol), is not installed by default, and would add a network/snap-store dependency. If a socket
operation fails, the escrow is reported to the server as a failure (the host shows "Action
required"/"Failed") rather than silently falling back to the passphrase dialog, which cannot work on
a host with no user passphrase. The socket is not on the boot/unlock path either — a TPM-backed host
boots and auto-unlocks via snapd's initramfs regardless.

snapd's `/v2/system-volumes` exposes no way to delete a recovery-key slot (as of snapd 2.74 — only
passphrase/PIN auth factors can be removed). A recovery key is retired by rotating it
(`replace-recovery-key`), which is what enrollment does on a retry. So if escrow to the Fleet server
fails after a key is enrolled, orbit does not (and cannot) delete it; the host stays pending escrow
and the next attempt regenerates and replaces the key in place, so escrow self-heals on retry. The
orphaned key is harmless because its secret was never stored anywhere.

## Key storage and security

Encryption keys are stored in the `host_disk_encryption_keys` table. The value for the key is
encrypted using Fleet's CA certificate, and thus can only be decrypted if you have the CA private
key. Additionally, a backup copy of any key that gets escrowed is stored in the
`host_disk_encryption_keys_archive` table. In the event that a host's encryption key is unavailable
due to a rotation or other event, it is possible to restore the host's most recently archived key by
executing the following query against the Fleet server's MySQL database, replacing HOST_ID with the
ID of the host in question:

```sql
INSERT INTO host_disk_encryption_keys (host_id, base64_encrypted, base64_encrypted_salt, key_slot, decryptable, created_at)
SELECT host_id, base64_encrypted, base64_encrypted_salt, key_slot, 1 AS decryptable, created_at FROM host_disk_encryption_keys_archive
WHERE host_id = HOST_ID ORDER BY created_at DESC LIMIT 1;
```

Please note that this key is not guaranteed to work and restoration should be a last resort option.
In many cases, Fleet triggers key rotation when the existing key may not work

## Related resources

- [MDM Product Group Documentation](../../product-groups/mdm/) - Documentation for the MDM product group
- [MDM Development Guides](../../guides/mdm/) - Guides for MDM development
- [Enforce disk encryption guide](https://fleetdm.com/guides/enforce-disk-encryption)
- [Bitlocker Policy Settings registry key guide by Geoff Chappell](https://www.geoffchappell.com/studies/windows/win32/fveapi/policy/index.htm)
