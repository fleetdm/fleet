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

```
[Placeholder for Disk Encryption Architecture Diagram]
```

## Platform-specific implementation

### FileVault (macOS)

For macOS, disk encryption involves a two-step process:

1. Sending a profile with two payloads:
   - A Payload to configure how the disk is going to be encrypted
   - A Payload to configure the escrow of the encryption key

2. Retrieving the disk encryption key:
   - Via osquery, we grab the (encrypted) disk encryption key
   - In a cron job, we verify that we're able to decrypt the key

If we're not able to decrypt the key for a host, the key needs to be rotated. Rotation happens silently by:

1. The server sends a notification to orbit, notifying that the key couldn't be decrypted.
2. orbit enables an authorization plugin named [Escrow Buddy](https://github.com/macadmins/escrow-buddy) that performs the key rotation the next time the user logs in.
3. Fleet retrieves and tries to validate the key again.

### BitLocker (Windows)

Disk encryption in Windows is performed entirely by orbit.

When disk encryption is enabled, the server sends a notification to orbit, who calls the [Win32_EncryptableVolume class](https://learn.microsoft.com/en-us/windows/win32/secprov/getencryptionmethod-win32-encryptablevolume) to encrypt/decrypt the disk and generate an encryption key.

After the disk is encrypted, orbit sends the key back to the server using an orbit-authenticated endpoint (`POST /api/fleet/orbit/disk_encryption_key`).

## Key storage and security

Encryption keys are stored in the `host_disk_encryption_keys` table. The value for the key is encrypted using Fleet's CA certificate, and thus can only be decrypted if you have the CA private key.

## Related resources

- [MDM Product Group Documentation](../../product-groups/mdm/) - Documentation for the MDM product group
- [MDM Development Guides](../../guides/mdm/) - Guides for MDM development