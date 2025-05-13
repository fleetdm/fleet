# Software Installation Development Guide

This guide provides instructions for developing Software Installation functionality in Fleet.

## Introduction

Software Installation in Fleet enables the deployment and installation of software packages across the device fleet. This guide covers the development and implementation of Software Installation features.

## Prerequisites

Before you begin developing Software Installation functionality, you should have:

- A development environment set up according to the [Building Fleet](../../getting-started/building-fleet.md) guide
- Basic understanding of software installation mechanisms on different platforms
- Familiarity with Fleet's architecture
- Understanding of MDM capabilities for software installation

## Software Installation Architecture

Software Installation in Fleet follows a specific flow:

1. User initiates software installation through the UI or API
2. Fleet server creates an installation task
3. Fleet server distributes the installation task to the target devices
4. Devices execute the installation task
5. Devices report the installation status back to the Fleet server
6. Fleet server updates the installation status in the database

## Implementation

### Database Schema

Software Installation information is stored in the Fleet database:

```sql
CREATE TABLE software_packages (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  version VARCHAR(255) NOT NULL,
  description TEXT,
  platform VARCHAR(255) NOT NULL,
  package_type VARCHAR(255) NOT NULL,
  package_url VARCHAR(255) NOT NULL,
  package_hash VARCHAR(255) NOT NULL,
  package_size BIGINT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY (name, version, platform)
);

CREATE TABLE software_installation_tasks (
  id INT AUTO_INCREMENT PRIMARY KEY,
  package_id INT NOT NULL,
  status VARCHAR(255) NOT NULL,
  created_by INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (package_id) REFERENCES software_packages(id),
  FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE software_installation_targets (
  id INT AUTO_INCREMENT PRIMARY KEY,
  task_id INT NOT NULL,
  type VARCHAR(255) NOT NULL,
  target_id INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (task_id) REFERENCES software_installation_tasks(id)
);

CREATE TABLE host_software_installations (
  id INT AUTO_INCREMENT PRIMARY KEY,
  host_id INT NOT NULL,
  task_id INT NOT NULL,
  status VARCHAR(255) NOT NULL,
  error_message TEXT,
  started_at TIMESTAMP NULL,
  completed_at TIMESTAMP NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (host_id) REFERENCES hosts(id),
  FOREIGN KEY (task_id) REFERENCES software_installation_tasks(id),
  UNIQUE KEY (host_id, task_id)
);
```

### Package Management

Implement package management:

```go
func CreateSoftwarePackage(db *sql.DB, name, version, description, platform, packageType, packageURL, packageHash string, packageSize int64) (int, error) {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return 0, err
    }
    defer tx.Rollback()
    
    // Check if the package already exists
    var id int
    err = tx.QueryRow(
        "SELECT id FROM software_packages WHERE name = ? AND version = ? AND platform = ?",
        name, version, platform,
    ).Scan(&id)
    
    if err == sql.ErrNoRows {
        // Insert the package
        result, err := tx.Exec(
            `INSERT INTO software_packages (
                name, version, description, platform, package_type, package_url, package_hash, package_size
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
            name, version, description, platform, packageType, packageURL, packageHash, packageSize,
        )
        if err != nil {
            return 0, err
        }
        
        // Get the package ID
        id64, err := result.LastInsertId()
        if err != nil {
            return 0, err
        }
        id = int(id64)
    } else if err != nil {
        return 0, err
    } else {
        // Update the package
        _, err := tx.Exec(
            `UPDATE software_packages SET
                description = ?,
                package_type = ?,
                package_url = ?,
                package_hash = ?,
                package_size = ?,
                updated_at = NOW()
            WHERE id = ?`,
            description, packageType, packageURL, packageHash, packageSize, id,
        )
        if err != nil {
            return 0, err
        }
    }
    
    // Commit the transaction
    err = tx.Commit()
    if err != nil {
        return 0, err
    }
    
    return id, nil
}
```

### Installation Task Creation

Implement installation task creation:

```go
func CreateInstallationTask(db *sql.DB, packageID, userID int, targets []Target) (int, error) {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return 0, err
    }
    defer tx.Rollback()
    
    // Create the task
    result, err := tx.Exec(
        "INSERT INTO software_installation_tasks (package_id, status, created_by) VALUES (?, ?, ?)",
        packageID, "pending", userID,
    )
    if err != nil {
        return 0, err
    }
    
    // Get the task ID
    taskID64, err := result.LastInsertId()
    if err != nil {
        return 0, err
    }
    taskID := int(taskID64)
    
    // Add targets
    for _, target := range targets {
        _, err := tx.Exec(
            "INSERT INTO software_installation_targets (task_id, type, target_id) VALUES (?, ?, ?)",
            taskID, target.Type, target.ID,
        )
        if err != nil {
            return 0, err
        }
    }
    
    // Commit the transaction
    err = tx.Commit()
    if err != nil {
        return 0, err
    }
    
    return taskID, nil
}
```

### Installation Task Distribution

Implement installation task distribution:

```go
func DistributeInstallationTask(db *sql.DB, taskID int) error {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Get the task
    var packageID int
    var status string
    err = tx.QueryRow(
        "SELECT package_id, status FROM software_installation_tasks WHERE id = ?",
        taskID,
    ).Scan(&packageID, &status)
    if err != nil {
        return err
    }
    
    // Check if the task is pending
    if status != "pending" {
        return fmt.Errorf("task is not pending: %s", status)
    }
    
    // Get the package
    var name, version, platform, packageType, packageURL, packageHash string
    var packageSize int64
    err = tx.QueryRow(
        `SELECT name, version, platform, package_type, package_url, package_hash, package_size
         FROM software_packages WHERE id = ?`,
        packageID,
    ).Scan(&name, &version, &platform, &packageType, &packageURL, &packageHash, &packageSize)
    if err != nil {
        return err
    }
    
    // Get the target hosts
    rows, err := tx.Query(`
        SELECT DISTINCT h.id
        FROM hosts h
        JOIN software_installation_targets sit ON (
            (sit.type = 'host' AND sit.target_id = h.id) OR
            (sit.type = 'label' AND sit.target_id IN (
                SELECT label_id FROM host_labels WHERE host_id = h.id
            )) OR
            (sit.type = 'team' AND sit.target_id IN (
                SELECT team_id FROM host_teams WHERE host_id = h.id
            )) OR
            (sit.type = 'all')
        )
        WHERE sit.task_id = ? AND h.platform = ?
    `, taskID, platform)
    if err != nil {
        return err
    }
    defer rows.Close()
    
    // Create host installations
    for rows.Next() {
        var hostID int
        err := rows.Scan(&hostID)
        if err != nil {
            return err
        }
        
        // Check if the host installation already exists
        var id int
        err = tx.QueryRow(
            "SELECT id FROM host_software_installations WHERE host_id = ? AND task_id = ?",
            hostID, taskID,
        ).Scan(&id)
        
        if err == sql.ErrNoRows {
            // Insert the host installation
            _, err := tx.Exec(
                "INSERT INTO host_software_installations (host_id, task_id, status) VALUES (?, ?, ?)",
                hostID, taskID, "pending",
            )
            if err != nil {
                return err
            }
        } else if err != nil {
            return err
        }
    }
    
    // Update the task status
    _, err = tx.Exec(
        "UPDATE software_installation_tasks SET status = ?, updated_at = NOW() WHERE id = ?",
        "in_progress", taskID,
    )
    if err != nil {
        return err
    }
    
    // Commit the transaction
    err = tx.Commit()
    if err != nil {
        return err
    }
    
    return nil
}
```

### Installation Execution

Implement installation execution on devices:

```go
func ExecuteInstallation(client *http.Client, serverURL, nodeKey string) error {
    // Get pending installations
    installations, err := GetPendingInstallations(client, serverURL, nodeKey)
    if err != nil {
        return err
    }
    
    // Execute each installation
    for _, installation := range installations {
        // Update installation status
        err := UpdateInstallationStatus(client, serverURL, nodeKey, installation.ID, "in_progress", "")
        if err != nil {
            return err
        }
        
        // Download the package
        packagePath, err := DownloadPackage(installation.PackageURL, installation.PackageHash)
        if err != nil {
            UpdateInstallationStatus(client, serverURL, nodeKey, installation.ID, "failed", err.Error())
            continue
        }
        
        // Install the package
        err = InstallPackage(packagePath, installation.PackageType)
        if err != nil {
            UpdateInstallationStatus(client, serverURL, nodeKey, installation.ID, "failed", err.Error())
            continue
        }
        
        // Update installation status
        err = UpdateInstallationStatus(client, serverURL, nodeKey, installation.ID, "completed", "")
        if err != nil {
            return err
        }
    }
    
    return nil
}

func InstallPackage(packagePath, packageType string) error {
    switch runtime.GOOS {
    case "darwin":
        switch packageType {
        case "pkg":
            return InstallMacOSPkg(packagePath)
        case "dmg":
            return InstallMacOSDmg(packagePath)
        default:
            return fmt.Errorf("unsupported package type for macOS: %s", packageType)
        }
    case "windows":
        switch packageType {
        case "msi":
            return InstallWindowsMsi(packagePath)
        case "exe":
            return InstallWindowsExe(packagePath)
        default:
            return fmt.Errorf("unsupported package type for Windows: %s", packageType)
        }
    case "linux":
        switch packageType {
        case "deb":
            return InstallLinuxDeb(packagePath)
        case "rpm":
            return InstallLinuxRpm(packagePath)
        default:
            return fmt.Errorf("unsupported package type for Linux: %s", packageType)
        }
    default:
        return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
    }
}

func InstallMacOSPkg(packagePath string) error {
    cmd := exec.Command("installer", "-pkg", packagePath, "-target", "/")
    return cmd.Run()
}

func InstallMacOSDmg(packagePath string) error {
    // Mount the DMG
    mountCmd := exec.Command("hdiutil", "attach", packagePath)
    mountOutput, err := mountCmd.Output()
    if err != nil {
        return err
    }
    
    // Parse the mount point
    mountPoint := ""
    scanner := bufio.NewScanner(bytes.NewReader(mountOutput))
    for scanner.Scan() {
        line := scanner.Text()
        if strings.Contains(line, "/Volumes/") {
            parts := strings.Split(line, "/Volumes/")
            mountPoint = "/Volumes/" + parts[1]
            break
        }
    }
    
    if mountPoint == "" {
        return fmt.Errorf("failed to find mount point")
    }
    
    // Find the .app bundle
    appPath := ""
    err = filepath.Walk(mountPoint, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if strings.HasSuffix(path, ".app") && info.IsDir() {
            appPath = path
            return filepath.SkipDir
        }
        return nil
    })
    if err != nil {
        return err
    }
    
    if appPath == "" {
        return fmt.Errorf("failed to find .app bundle")
    }
    
    // Copy the .app bundle to /Applications
    copyCmd := exec.Command("cp", "-R", appPath, "/Applications/")
    err = copyCmd.Run()
    if err != nil {
        return err
    }
    
    // Unmount the DMG
    unmountCmd := exec.Command("hdiutil", "detach", mountPoint)
    return unmountCmd.Run()
}

func InstallWindowsMsi(packagePath string) error {
    cmd := exec.Command("msiexec", "/i", packagePath, "/qn")
    return cmd.Run()
}

func InstallWindowsExe(packagePath string) error {
    cmd := exec.Command(packagePath, "/S")
    return cmd.Run()
}

func InstallLinuxDeb(packagePath string) error {
    cmd := exec.Command("dpkg", "-i", packagePath)
    return cmd.Run()
}

func InstallLinuxRpm(packagePath string) error {
    cmd := exec.Command("rpm", "-i", packagePath)
    return cmd.Run()
}
```

### Installation Status Reporting

Implement installation status reporting:

```go
func UpdateInstallationStatus(client *http.Client, serverURL, nodeKey string, installationID int, status, errorMessage string) error {
    // Create the request body
    body := map[string]interface{}{
        "node_key": nodeKey,
        "installation_id": installationID,
        "status": status,
        "error_message": errorMessage,
    }
    
    // Convert to JSON
    bodyJSON, err := json.Marshal(body)
    if err != nil {
        return err
    }
    
    // Create the request
    req, err := http.NewRequest("POST", serverURL+"/api/v1/fleet/software/installations/status", bytes.NewBuffer(bodyJSON))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")
    
    // Send the request
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // Check the response
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }
    
    return nil
}
```

## API Endpoints

Implement API endpoints for software installation:

### Create Software Package

```go
func CreateSoftwarePackageHandler(w http.ResponseWriter, r *http.Request) {
    // Parse the request body
    var req struct {
        Name        string `json:"name"`
        Version     string `json:"version"`
        Description string `json:"description"`
        Platform    string `json:"platform"`
        PackageType string `json:"package_type"`
        PackageURL  string `json:"package_url"`
        PackageHash string `json:"package_hash"`
        PackageSize int64  `json:"package_size"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Create the package
    id, err := CreateSoftwarePackage(
        db,
        req.Name,
        req.Version,
        req.Description,
        req.Platform,
        req.PackageType,
        req.PackageURL,
        req.PackageHash,
        req.PackageSize,
    )
    if err != nil {
        http.Error(w, fmt.Sprintf("Error creating package: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the ID
    json.NewEncoder(w).Encode(map[string]int{"id": id})
}
```

### Create Installation Task

```go
func CreateInstallationTaskHandler(w http.ResponseWriter, r *http.Request) {
    // Parse the request body
    var req struct {
        PackageID int      `json:"package_id"`
        Targets   []Target `json:"targets"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Get the user ID from the context
    userID := r.Context().Value("user_id").(int)
    
    // Create the task
    id, err := CreateInstallationTask(db, req.PackageID, userID, req.Targets)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error creating task: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Distribute the task
    err = DistributeInstallationTask(db, id)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error distributing task: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the ID
    json.NewEncoder(w).Encode(map[string]int{"id": id})
}
```

### Get Pending Installations

```go
func GetPendingInstallationsHandler(w http.ResponseWriter, r *http.Request) {
    // Get the node key from the request
    nodeKey := r.URL.Query().Get("node_key")
    if nodeKey == "" {
        http.Error(w, "Missing node_key parameter", http.StatusBadRequest)
        return
    }
    
    // Get the host ID from the node key
    hostID, err := GetHostIDFromNodeKey(db, nodeKey)
    if err != nil {
        http.Error(w, "Invalid node key", http.StatusUnauthorized)
        return
    }
    
    // Get pending installations for the host
    installations, err := GetPendingInstallationsForHost(db, hostID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting installations: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the installations
    json.NewEncoder(w).Encode(installations)
}
```

### Update Installation Status

```go
func UpdateInstallationStatusHandler(w http.ResponseWriter, r *http.Request) {
    // Parse the request body
    var req struct {
        NodeKey       string `json:"node_key"`
        InstallationID int    `json:"installation_id"`
        Status        string `json:"status"`
        ErrorMessage  string `json:"error_message"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Get the host ID from the node key
    hostID, err := GetHostIDFromNodeKey(db, req.NodeKey)
    if err != nil {
        http.Error(w, "Invalid node key", http.StatusUnauthorized)
        return
    }
    
    // Update the installation status
    err = UpdateHostInstallationStatus(db, hostID, req.InstallationID, req.Status, req.ErrorMessage)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error updating status: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return success
    w.WriteHeader(http.StatusOK)
}
```

## Testing

### Manual Testing

1. Create a software package
2. Create an installation task
3. Verify the task is distributed to target devices
4. Check that devices execute the installation
5. Verify the installation status is reported back to the server

### Automated Testing

Fleet includes automated tests for Software Installation functionality:

```bash
# Run Software Installation tests
go test -v ./server/service/software_installation_test.go
```

## Debugging

### Package Management Issues

- **Package Validation**: Verify the package is valid and can be installed
- **Package Download**: Ensure the package can be downloaded from the specified URL
- **Package Hash**: Check if the package hash matches the expected hash

### Installation Issues

- **Platform Compatibility**: Verify the package is compatible with the target platform
- **Installation Commands**: Ensure the installation commands are correct
- **Error Handling**: Check if errors during installation are properly handled

## Performance Considerations

Software Installation can impact system performance, especially for large packages or large fleets:

- **Package Size**: Consider the size of packages and their impact on network bandwidth
- **Installation Timing**: Schedule installations during off-hours to minimize impact
- **Throttling**: Implement throttling to limit the number of concurrent installations
- **Caching**: Cache packages to reduce download time

## Related Resources

- [Software Installation Architecture](../../architecture/software/software-installation.md)
- [Software Product Group Documentation](../../product-groups/software/)
- [MDM Documentation](../../product-groups/mdm/)