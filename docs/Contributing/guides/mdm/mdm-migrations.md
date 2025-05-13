# MDM Migrations Guide

This guide provides instructions for developing MDM Migration functionality in Fleet.

## Introduction

MDM Migrations in Fleet allow organizations to migrate devices from other MDM solutions to Fleet. This guide covers the development and implementation of MDM migration features for both Apple and Windows devices.

## Prerequisites

Before you begin developing MDM Migration functionality, you should have:

- A development environment set up according to the [Building Fleet](../../getting-started/building-fleet.md) guide
- Basic understanding of MDM protocols for Apple and Windows
- Access to source MDM solutions for testing (e.g., MicroMDM, Jamf, Intune)
- Understanding of Fleet's MDM architecture

## Migration Types

Fleet supports two main types of MDM migrations:

### Regular Migration

The regular migration flow involves `fleetd` guiding the user through the migration process:

1. User installs `fleetd` on the device
2. `fleetd` detects the existing MDM enrollment
3. `fleetd` guides the user through the migration steps
4. Device is unenrolled from the source MDM
5. Device is enrolled in Fleet MDM

### Seamless Migration

The seamless migration flow allows for migration without user intervention:

1. Organization sets up a migration proxy
2. Organization extracts data from the source MDM
3. Devices communicate with the migration proxy
4. Proxy redirects devices to Fleet MDM
5. Devices are automatically enrolled in Fleet MDM

## Implementation

### Database Schema

MDM migration information is stored in the Fleet database:

```sql
CREATE TABLE mdm_migrations (
  id INT AUTO_INCREMENT PRIMARY KEY,
  source_mdm VARCHAR(255) NOT NULL,
  migration_type VARCHAR(255) NOT NULL,
  status VARCHAR(255) NOT NULL,
  started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  completed_at TIMESTAMP NULL DEFAULT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE mdm_migration_devices (
  id INT AUTO_INCREMENT PRIMARY KEY,
  migration_id INT NOT NULL,
  device_id INT NOT NULL,
  serial_number VARCHAR(255) NOT NULL,
  status VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (migration_id) REFERENCES mdm_migrations(id),
  FOREIGN KEY (device_id) REFERENCES hosts(id)
);
```

### Regular Migration Implementation

#### Detecting Existing MDM

Implement detection of existing MDM enrollments:

```go
func DetectExistingMDM() (string, error) {
    // For macOS
    if runtime.GOOS == "darwin" {
        // Check for profiles with MDM payload
        cmd := exec.Command("profiles", "-L")
        output, err := cmd.Output()
        if err != nil {
            return "", err
        }
        
        if strings.Contains(string(output), "com.apple.mdm") {
            // Parse the output to determine the MDM vendor
            // This is a simplified example
            if strings.Contains(string(output), "Jamf") {
                return "Jamf", nil
            } else if strings.Contains(string(output), "MicroMDM") {
                return "MicroMDM", nil
            } else {
                return "Unknown MDM", nil
            }
        }
    }
    
    // For Windows
    if runtime.GOOS == "windows" {
        // Check for MDM enrollment
        cmd := exec.Command("powershell", "-Command", "Get-MDMDeviceStatus")
        output, err := cmd.Output()
        if err != nil {
            return "", err
        }
        
        if strings.Contains(string(output), "EnrollmentState: Enrolled") {
            // Parse the output to determine the MDM vendor
            // This is a simplified example
            if strings.Contains(string(output), "Intune") {
                return "Intune", nil
            } else {
                return "Unknown MDM", nil
            }
        }
    }
    
    return "", nil
}
```

#### Guiding the User

Implement user guidance for migration:

```go
func GuideMDMMigration(sourceMDM string) []MigrationStep {
    steps := []MigrationStep{}
    
    // Common steps
    steps = append(steps, MigrationStep{
        Title: "Backup your device",
        Description: "Before proceeding with the migration, make sure to backup your device.",
        Action: "backup",
    })
    
    // Source MDM specific steps
    if sourceMDM == "Jamf" {
        steps = append(steps, MigrationStep{
            Title: "Remove Jamf MDM profile",
            Description: "Go to System Preferences > Profiles and remove the Jamf MDM profile.",
            Action: "remove_profile",
        })
    } else if sourceMDM == "MicroMDM" {
        steps = append(steps, MigrationStep{
            Title: "Remove MicroMDM profile",
            Description: "Go to System Preferences > Profiles and remove the MicroMDM profile.",
            Action: "remove_profile",
        })
    } else if sourceMDM == "Intune" {
        steps = append(steps, MigrationStep{
            Title: "Remove Intune enrollment",
            Description: "Go to Settings > Accounts > Access work or school and remove the Intune enrollment.",
            Action: "remove_enrollment",
        })
    }
    
    // Fleet enrollment steps
    steps = append(steps, MigrationStep{
        Title: "Enroll in Fleet MDM",
        Description: "Follow the instructions to enroll in Fleet MDM.",
        Action: "enroll_fleet",
    })
    
    return steps
}
```

### Seamless Migration Implementation

#### Migration Proxy

Implement a migration proxy for seamless migrations:

```go
type MigrationProxy struct {
    SourceMDM     string
    SourceURL     string
    FleetURL      string
    CertPath      string
    KeyPath       string
}

func NewMigrationProxy(sourceMDM, sourceURL, fleetURL, certPath, keyPath string) *MigrationProxy {
    return &MigrationProxy{
        SourceMDM:     sourceMDM,
        SourceURL:     sourceURL,
        FleetURL:      fleetURL,
        CertPath:      certPath,
        KeyPath:       keyPath,
    }
}

func (p *MigrationProxy) Start() error {
    // Set up HTTP server
    http.HandleFunc("/", p.handleRequest)
    
    // Start HTTPS server
    return http.ListenAndServeTLS(":8443", p.CertPath, p.KeyPath, nil)
}

func (p *MigrationProxy) handleRequest(w http.ResponseWriter, r *http.Request) {
    // Log the request
    log.Printf("Received request: %s %s", r.Method, r.URL.Path)
    
    // Check if this is an MDM request
    if strings.HasPrefix(r.URL.Path, "/mdm") {
        // Redirect to Fleet MDM
        newURL := p.FleetURL + r.URL.Path
        http.Redirect(w, r, newURL, http.StatusTemporaryRedirect)
        return
    }
    
    // Forward other requests to source MDM
    p.forwardRequest(w, r)
}

func (p *MigrationProxy) forwardRequest(w http.ResponseWriter, r *http.Request) {
    // Create a new request to the source MDM
    proxyReq, err := http.NewRequest(r.Method, p.SourceURL+r.URL.Path, r.Body)
    if err != nil {
        http.Error(w, "Error creating proxy request", http.StatusInternalServerError)
        return
    }
    
    // Copy headers
    for key, values := range r.Header {
        for _, value := range values {
            proxyReq.Header.Add(key, value)
        }
    }
    
    // Send the request
    client := &http.Client{}
    resp, err := client.Do(proxyReq)
    if err != nil {
        http.Error(w, "Error forwarding request", http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()
    
    // Copy response headers
    for key, values := range resp.Header {
        for _, value := range values {
            w.Header().Add(key, value)
        }
    }
    
    // Copy response status code
    w.WriteHeader(resp.StatusCode)
    
    // Copy response body
    io.Copy(w, resp.Body)
}
```

#### Data Extraction

Implement data extraction from source MDM:

```go
func ExtractMicroMDMData(dbPath string) ([]Device, error) {
    // Open the MicroMDM database
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, err
    }
    defer db.Close()
    
    // Query devices
    rows, err := db.Query("SELECT serial_number, udid, enrolled FROM devices")
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    // Extract device data
    devices := []Device{}
    for rows.Next() {
        var device Device
        err := rows.Scan(&device.SerialNumber, &device.UDID, &device.Enrolled)
        if err != nil {
            return nil, err
        }
        devices = append(devices, device)
    }
    
    return devices, nil
}
```

## Migration Flows

### Apple MDM Migration

#### Regular Flow

1. User installs `fleetd` on the macOS device
2. `fleetd` detects the existing MDM enrollment
3. `fleetd` guides the user to remove the existing MDM profile
4. User removes the MDM profile
5. `fleetd` guides the user to install the Fleet MDM profile
6. User installs the Fleet MDM profile

#### Seamless Flow

1. Organization sets up a migration proxy with the same domain as the source MDM
2. Organization extracts device data from the source MDM
3. Organization imports the device data into Fleet
4. Devices continue to communicate with the same domain
5. Proxy redirects MDM traffic to Fleet
6. Fleet sends a new enrollment profile to the devices

### Windows MDM Migration

#### Regular Flow

1. User installs `fleetd` on the Windows device
2. `fleetd` detects the existing MDM enrollment
3. `fleetd` guides the user to remove the existing MDM enrollment
4. User removes the MDM enrollment
5. `fleetd` guides the user to enroll in Fleet MDM
6. User enrolls in Fleet MDM

## Testing

### Manual Testing

1. Set up a test device with a source MDM
2. Implement the migration flow
3. Verify the device successfully migrates to Fleet MDM
4. Check that all device data is preserved

### Automated Testing

Fleet includes automated tests for MDM Migration functionality:

```bash
# Run MDM Migration tests
go test -v ./server/service/mdm_migration_test.go
```

## Debugging

### Regular Migration Issues

- **Detection**: Verify the existing MDM is correctly detected
- **User Guidance**: Ensure the user is provided with clear instructions
- **Enrollment**: Check if the Fleet MDM enrollment is successful

### Seamless Migration Issues

- **Proxy Configuration**: Verify the migration proxy is correctly configured
- **Data Extraction**: Ensure device data is correctly extracted from the source MDM
- **Redirection**: Check if devices are properly redirected to Fleet MDM

## Related Resources

- [MDM Migration Documentation](https://fleetdm.com/guides/mdm-migration)
- [Seamless MDM Migration Documentation](https://fleetdm.com/guides/seamless-mdm-migration)
- [MicroMDM Migration Tool](../../tools/mdm/migration/micromdm/touchless/)
- [MDM Proxy Tool](../../tools/mdm/migration/mdmproxy/)