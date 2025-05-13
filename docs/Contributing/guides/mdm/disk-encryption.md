# Disk Encryption Guide

This guide provides instructions for developing Disk Encryption functionality in Fleet's MDM.

## Introduction

Disk Encryption in Fleet's MDM allows for securing device data by encrypting the storage media. This guide covers the development and implementation of disk encryption features for both macOS (FileVault) and Windows (BitLocker).

## Prerequisites

Before you begin developing Disk Encryption functionality, you should have:

- A development environment set up according to the [Building Fleet](../../getting-started/building-fleet.md) guide
- Basic understanding of disk encryption technologies (FileVault, BitLocker)
- Access to macOS and Windows devices for testing
- Understanding of Fleet's MDM architecture

## Disk Encryption Overview

### macOS (FileVault)

FileVault is Apple's full-disk encryption feature for macOS:

- Uses XTS-AES-128 encryption with a 256-bit key
- Encrypts the entire system volume
- Requires a recovery key for emergency access
- Can be managed through MDM

### Windows (BitLocker)

BitLocker is Microsoft's full-disk encryption feature for Windows:

- Uses AES encryption with 128-bit or 256-bit keys
- Encrypts entire volumes
- Supports various authentication methods (TPM, PIN, USB key)
- Can be managed through MDM

## Implementation

### Database Schema

Disk encryption keys are stored in the Fleet database:

```sql
CREATE TABLE host_disk_encryption_keys (
  id INT AUTO_INCREMENT PRIMARY KEY,
  host_id INT NOT NULL,
  encrypted_key TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (host_id) REFERENCES hosts(id)
);
```

### Key Encryption

Encryption keys are encrypted before storage:

1. Fleet's CA certificate is used to encrypt the disk encryption key
2. The encrypted key is stored in the database
3. The key can only be decrypted with the CA private key

Example key encryption code:

```go
func encryptKey(key []byte, cert *x509.Certificate) ([]byte, error) {
    // Extract public key from certificate
    publicKey := cert.PublicKey.(*rsa.PublicKey)
    
    // Encrypt the key using RSA-OAEP
    encryptedKey, err := rsa.EncryptOAEP(
        sha256.New(),
        rand.Reader,
        publicKey,
        key,
        nil,
    )
    if err != nil {
        return nil, err
    }
    
    return encryptedKey, nil
}
```

### Key Decryption

Encrypted keys are decrypted when needed:

```go
func decryptKey(encryptedKey []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
    // Decrypt the key using RSA-OAEP
    key, err := rsa.DecryptOAEP(
        sha256.New(),
        rand.Reader,
        privateKey,
        encryptedKey,
        nil,
    )
    if err != nil {
        return nil, err
    }
    
    return key, nil
}
```

## macOS (FileVault) Implementation

### Enabling FileVault

FileVault is enabled through an MDM profile:

1. Create a configuration profile with FileVault settings
2. Include a payload for escrow of the recovery key
3. Send the profile to the device using MDM

Example FileVault configuration profile:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadContent</key>
    <array>
        <!-- FileVault Configuration Payload -->
        <dict>
            <key>PayloadType</key>
            <string>com.apple.MCX</string>
            <key>PayloadIdentifier</key>
            <string>com.fleetdm.mdm.filevault.config</string>
            <key>PayloadUUID</key>
            <string>00000000-0000-0000-0000-000000000000</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
            <key>PayloadEnabled</key>
            <true/>
            <key>dontAllowFDEDisable</key>
            <true/>
            <key>EnableFDERecoveryKey</key>
            <true/>
        </dict>
        <!-- FileVault Escrow Payload -->
        <dict>
            <key>PayloadType</key>
            <string>com.apple.security.FDERecoveryKeyEscrow</string>
            <key>PayloadIdentifier</key>
            <string>com.fleetdm.mdm.filevault.escrow</string>
            <key>PayloadUUID</key>
            <string>00000000-0000-0000-0000-000000000001</string>
            <key>PayloadVersion</key>
            <integer>1</integer>
            <key>PayloadEnabled</key>
            <true/>
            <key>Location</key>
            <string>https://fleet.example.com/api/v1/fleet/mdm/filevault/escrow</string>
            <key>EncryptCertificate</key>
            <data><!-- Base64-encoded certificate data --></data>
        </dict>
    </array>
    <key>PayloadDisplayName</key>
    <string>FileVault Configuration</string>
    <key>PayloadIdentifier</key>
    <string>com.fleetdm.mdm.filevault</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>00000000-0000-0000-0000-000000000002</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
</dict>
</plist>
```

### Retrieving FileVault Recovery Key

The FileVault recovery key is retrieved through osquery:

1. Use osquery to query the encrypted recovery key
2. Verify the key can be decrypted with Fleet's CA private key
3. Store the verified key in the database

Example osquery query:

```sql
SELECT encrypted_recovery_key FROM plist WHERE path = '/var/db/FileVaultPRK.dat'
```

### Key Rotation

If the key cannot be decrypted, it needs to be rotated:

1. Server sends a notification to orbit
2. orbit installs the Escrow Buddy plugin
3. Escrow Buddy rotates the key on next user login
4. The new key is retrieved and verified

## Windows (BitLocker) Implementation

### Enabling BitLocker

BitLocker is enabled through orbit:

1. Server sends a notification to orbit
2. orbit uses the Win32_EncryptableVolume class to enable BitLocker
3. orbit generates a recovery key
4. orbit sends the recovery key back to the server

Example orbit code for enabling BitLocker:

```go
func enableBitLocker(volumePath string) (string, error) {
    // Connect to WMI
    wmi, err := connectToWMI()
    if err != nil {
        return "", err
    }
    
    // Get the encryptable volume
    volume, err := wmi.GetEncryptableVolume(volumePath)
    if err != nil {
        return "", err
    }
    
    // Enable BitLocker
    err = volume.EnableBitLocker()
    if err != nil {
        return "", err
    }
    
    // Generate recovery key
    recoveryKey, err := volume.GenerateRecoveryKey()
    if err != nil {
        return "", err
    }
    
    return recoveryKey, nil
}
```

### Retrieving BitLocker Recovery Key

The BitLocker recovery key is sent to the server by orbit:

1. orbit retrieves the recovery key after enabling BitLocker
2. orbit sends the key to the server using an authenticated endpoint
3. Server encrypts and stores the key in the database

Example API endpoint for receiving the key:

```go
func handleBitLockerKey(w http.ResponseWriter, r *http.Request) {
    // Authenticate the request
    host, err := authenticateOrbitRequest(r)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Parse the request body
    var req struct {
        RecoveryKey string `json:"recovery_key"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Encrypt and store the key
    encryptedKey, err := encryptKey([]byte(req.RecoveryKey), caCert)
    if err != nil {
        http.Error(w, "Error encrypting key", http.StatusInternalServerError)
        return
    }
    
    err = storeEncryptedKey(host.ID, encryptedKey)
    if err != nil {
        http.Error(w, "Error storing key", http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusOK)
}
```

## Testing

### Manual Testing

1. Enable disk encryption on a test device
2. Verify the recovery key is retrieved and stored
3. Test key retrieval and decryption
4. Test key rotation (for FileVault)

### Automated Testing

Fleet includes automated tests for Disk Encryption functionality:

```bash
# Run Disk Encryption tests
go test -v ./server/service/disk_encryption_test.go
```

## Debugging

### FileVault Issues

- **Profile Installation**: Verify the FileVault configuration profile is installed
- **Key Escrow**: Check if the key escrow payload is correctly configured
- **Key Retrieval**: Ensure osquery can retrieve the encrypted recovery key
- **Key Decryption**: Verify the key can be decrypted with Fleet's CA private key

### BitLocker Issues

- **WMI Access**: Ensure orbit has access to WMI
- **TPM Availability**: Check if TPM is available and enabled
- **Key Generation**: Verify BitLocker can generate a recovery key
- **Key Transmission**: Ensure the key is securely transmitted to the server

## Related Resources

- [Disk Encryption Architecture](../../architecture/mdm/disk-encryption.md)
- [Apple FileVault Documentation](https://support.apple.com/guide/deployment/intro-to-filevault-dep82064ec40/web)
- [Microsoft BitLocker Documentation](https://learn.microsoft.com/en-us/windows/security/operating-system-security/data-protection/bitlocker/)
- [Win32_EncryptableVolume Class](https://learn.microsoft.com/en-us/windows/win32/secprov/win32-encryptablevolume)