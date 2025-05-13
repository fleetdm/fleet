# Automated Device Enrollment Guide

This guide provides instructions for developing Automated Device Enrollment (ADE) functionality in Fleet's MDM.

## Introduction

Automated Device Enrollment (ADE) in Fleet's MDM allows for zero-touch deployment of devices, enabling organizations to automatically enroll and configure devices without manual intervention. This guide covers the development and implementation of ADE features for Apple devices (formerly known as DEP).

## Prerequisites

Before you begin developing ADE functionality, you should have:

- A development environment set up according to the [Building Fleet](../../getting-started/building-fleet.md) guide
- Basic understanding of Apple's Automated Device Enrollment
- Access to Apple Business Manager or Apple School Manager
- Understanding of Fleet's MDM architecture

## Apple Business Manager / Apple School Manager Setup

### Creating a Virtual Server

To develop ADE functionality, you need to create a virtual MDM server in Apple Business Manager:

1. Log in to [Apple Business Manager](https://business.apple.com)
2. Navigate to Settings > MDM Servers
3. Click "Add MDM Server"
4. Enter a name for your development MDM server
5. Download the server token

### Server Token

The server token is a signed XML file that authenticates your MDM server with Apple's services:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>consumer_key</key>
    <string>CK_your_consumer_key</string>
    <key>consumer_secret</key>
    <string>CS_your_consumer_secret</string>
    <key>access_token</key>
    <string>AT_your_access_token</string>
    <key>access_secret</key>
    <string>AS_your_access_secret</string>
    <key>access_token_expiry</key>
    <date>2023-12-31T12:00:00Z</date>
</dict>
</plist>
```

### Importing the Server Token

Import the server token into Fleet:

```go
func importServerToken(tokenData []byte) (*DEPToken, error) {
    // Parse the token plist
    var tokenPlist struct {
        ConsumerKey     string `plist:"consumer_key"`
        ConsumerSecret  string `plist:"consumer_secret"`
        AccessToken     string `plist:"access_token"`
        AccessSecret    string `plist:"access_secret"`
        AccessTokenExpiry time.Time `plist:"access_token_expiry"`
    }
    
    err := plist.Unmarshal(tokenData, &tokenPlist)
    if err != nil {
        return nil, err
    }
    
    // Create a DEP token
    token := &DEPToken{
        ConsumerKey:     tokenPlist.ConsumerKey,
        ConsumerSecret:  tokenPlist.ConsumerSecret,
        AccessToken:     tokenPlist.AccessToken,
        AccessSecret:    tokenPlist.AccessSecret,
        AccessTokenExpiry: tokenPlist.AccessTokenExpiry,
    }
    
    return token, nil
}
```

## Implementation

### Database Schema

ADE information is stored in the Fleet database:

```sql
CREATE TABLE dep_tokens (
  id INT AUTO_INCREMENT PRIMARY KEY,
  consumer_key VARCHAR(255) NOT NULL,
  consumer_secret VARCHAR(255) NOT NULL,
  access_token VARCHAR(255) NOT NULL,
  access_secret VARCHAR(255) NOT NULL,
  access_token_expiry TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE host_dep_assignments (
  id INT AUTO_INCREMENT PRIMARY KEY,
  serial_number VARCHAR(255) NOT NULL,
  model VARCHAR(255) NOT NULL,
  description VARCHAR(255),
  color VARCHAR(255),
  asset_tag VARCHAR(255),
  profile_uuid VARCHAR(255),
  assigned_date TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at TIMESTAMP NULL DEFAULT NULL,
  UNIQUE KEY (serial_number)
);
```

### DEP API Client

Implement a client for the DEP API:

```go
type DEPClient struct {
    ConsumerKey     string
    ConsumerSecret  string
    AccessToken     string
    AccessSecret    string
    AccessTokenExpiry time.Time
}

func NewDEPClient(token *DEPToken) *DEPClient {
    return &DEPClient{
        ConsumerKey:     token.ConsumerKey,
        ConsumerSecret:  token.ConsumerSecret,
        AccessToken:     token.AccessToken,
        AccessSecret:    token.AccessSecret,
        AccessTokenExpiry: token.AccessTokenExpiry,
    }
}

func (c *DEPClient) FetchDevices() ([]DEPDevice, error) {
    // Implement OAuth 1.0a authentication
    // Make API request to fetch devices
    // Parse and return the devices
}

func (c *DEPClient) AssignProfile(serialNumber, profileUUID string) error {
    // Implement OAuth 1.0a authentication
    // Make API request to assign profile
    // Handle response
}
```

### Enrollment Profiles

Create enrollment profiles for ADE devices:

```go
type DEPProfile struct {
    UUID                string
    Name                string
    Description         string
    OrganizationName    string
    Department          string
    SupportPhoneNumber  string
    SupportEmailAddress string
    Supervised          bool
    Mandatory           bool
    AwaitDeviceConfigured bool
    MDMRemovable        bool
    SetupItems          []string
}

func CreateDEPProfile(profile *DEPProfile) ([]byte, error) {
    // Create a profile in JSON format
    profileJSON, err := json.Marshal(profile)
    if err != nil {
        return nil, err
    }
    
    return profileJSON, nil
}
```

### Device Synchronization

Implement synchronization of devices from Apple Business Manager:

```go
func SyncDEPDevices(client *DEPClient, db *sql.DB) error {
    // Fetch devices from DEP API
    devices, err := client.FetchDevices()
    if err != nil {
        return err
    }
    
    // Process each device
    for _, device := range devices {
        // Check if device exists in database
        var id int
        err := db.QueryRow("SELECT id FROM host_dep_assignments WHERE serial_number = ?", device.SerialNumber).Scan(&id)
        if err == sql.ErrNoRows {
            // Insert new device
            _, err = db.Exec(
                "INSERT INTO host_dep_assignments (serial_number, model, description, color, asset_tag, profile_uuid, assigned_date) VALUES (?, ?, ?, ?, ?, ?, ?)",
                device.SerialNumber, device.Model, device.Description, device.Color, device.AssetTag, device.ProfileUUID, device.AssignedDate,
            )
            if err != nil {
                return err
            }
        } else if err != nil {
            return err
        } else {
            // Update existing device
            _, err = db.Exec(
                "UPDATE host_dep_assignments SET model = ?, description = ?, color = ?, asset_tag = ?, profile_uuid = ?, assigned_date = ?, updated_at = NOW() WHERE id = ?",
                device.Model, device.Description, device.Color, device.AssetTag, device.ProfileUUID, device.AssignedDate, id,
            )
            if err != nil {
                return err
            }
        }
    }
    
    return nil
}
```

### Cron Job

Implement a cron job to regularly synchronize devices:

```go
func DEPSyncerCronJob(client *DEPClient, db *sql.DB) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            err := SyncDEPDevices(client, db)
            if err != nil {
                log.Printf("Error syncing DEP devices: %v", err)
            }
        }
    }
}
```

## Enrollment Flow

### Device Assignment

Devices are assigned to Fleet in Apple Business Manager:

1. Log in to Apple Business Manager
2. Navigate to Devices
3. Select the devices to assign
4. Choose your MDM server
5. Confirm the assignment

### Profile Assignment

Profiles are assigned to devices in Fleet:

1. Create an enrollment profile in Fleet
2. Assign the profile to devices based on criteria (serial number, model, etc.)
3. Fleet assigns the profile to the devices in Apple Business Manager

### Device Enrollment

When a device is activated, it automatically enrolls with Fleet:

1. Device is powered on and connected to the internet
2. Device contacts Apple's activation servers
3. Activation servers direct the device to Fleet's MDM server
4. Device enrolls with Fleet's MDM server
5. Fleet sends initial configuration to the device

## Testing

### Manual Testing

1. Assign a test device to your MDM server in Apple Business Manager
2. Create and assign an enrollment profile
3. Erase and activate the device
4. Verify the device enrolls with Fleet

### Automated Testing

Fleet includes automated tests for ADE functionality:

```bash
# Run ADE tests
go test -v ./server/service/dep_test.go
```

## Debugging

### Synchronization Issues

- **Token Validity**: Verify the server token is valid and not expired
- **API Access**: Check if the DEP API is accessible
- **Error Handling**: Ensure errors from the DEP API are properly handled

### Enrollment Issues

- **Profile Assignment**: Verify the enrollment profile is correctly assigned to the device
- **MDM Server Configuration**: Check if the MDM server is properly configured
- **Device State**: Ensure the device is in a clean state (erased) before enrollment

## Special Cases

### Host Deletion

If a host is deleted in Fleet but exists in Apple Business Manager:

1. Soft delete the `host_dep_assignments` entry
2. Create a new host entry on the next sync
3. This allows IT admins to move the host between teams before it turns on MDM

## Related Resources

- [Automated Device Enrollment Architecture](../../architecture/mdm/automated-device-enrollment.md)
- [Apple Business Manager Documentation](https://support.apple.com/guide/apple-business-manager/welcome/web)
- [DEP API Documentation](https://developer.apple.com/documentation/devicemanagement/device_assignment)