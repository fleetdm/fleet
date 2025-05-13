# Software Updates Development Guide

This guide provides instructions for developing Software Updates functionality in Fleet.

## Introduction

Software Updates in Fleet enables the management and deployment of software updates across the device fleet. This guide covers the development and implementation of Software Updates features.

## Prerequisites

Before you begin developing Software Updates functionality, you should have:

- A development environment set up according to the [Building Fleet](../../getting-started/building-fleet.md) guide
- Basic understanding of software update mechanisms on different platforms
- Familiarity with Fleet's architecture
- Understanding of MDM capabilities for software updates

## Software Updates Architecture

Software Updates in Fleet follows a specific flow:

1. Fleet server identifies available updates for installed software
2. User initiates update deployment through the UI or API
3. Fleet server creates an update task
4. Fleet server distributes the update task to the target devices
5. Devices execute the update task
6. Devices report the update status back to the Fleet server
7. Fleet server updates the update status in the database

## Implementation

### Database Schema

Software Updates information is stored in the Fleet database:

```sql
CREATE TABLE software_updates (
  id INT AUTO_INCREMENT PRIMARY KEY,
  software_id INT NOT NULL,
  version VARCHAR(255) NOT NULL,
  release_notes TEXT,
  severity VARCHAR(255),
  update_url VARCHAR(255) NOT NULL,
  update_hash VARCHAR(255) NOT NULL,
  update_size BIGINT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (software_id) REFERENCES software(id),
  UNIQUE KEY (software_id, version)
);

CREATE TABLE software_update_tasks (
  id INT AUTO_INCREMENT PRIMARY KEY,
  update_id INT NOT NULL,
  status VARCHAR(255) NOT NULL,
  created_by INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (update_id) REFERENCES software_updates(id),
  FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE TABLE software_update_targets (
  id INT AUTO_INCREMENT PRIMARY KEY,
  task_id INT NOT NULL,
  type VARCHAR(255) NOT NULL,
  target_id INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (task_id) REFERENCES software_update_tasks(id)
);

CREATE TABLE host_software_updates (
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
  FOREIGN KEY (task_id) REFERENCES software_update_tasks(id),
  UNIQUE KEY (host_id, task_id)
);
```

### Update Identification

Implement update identification:

```go
func IdentifySoftwareUpdates(db *sql.DB) error {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Get all software
    softwareRows, err := tx.Query("SELECT id, name, version, publisher FROM software")
    if err != nil {
        return err
    }
    defer softwareRows.Close()
    
    // Process each software
    for softwareRows.Next() {
        var id int
        var name, version, publisher string
        err := softwareRows.Scan(&id, &name, &version, &publisher)
        if err != nil {
            return err
        }
        
        // Check for updates
        updates, err := CheckForUpdates(name, version, publisher)
        if err != nil {
            continue
        }
        
        // Process each update
        for _, update := range updates {
            // Check if the update already exists
            var updateID int
            err := tx.QueryRow(
                "SELECT id FROM software_updates WHERE software_id = ? AND version = ?",
                id, update.Version,
            ).Scan(&updateID)
            
            if err == sql.ErrNoRows {
                // Insert the update
                result, err := tx.Exec(
                    `INSERT INTO software_updates (
                        software_id, version, release_notes, severity, update_url, update_hash, update_size
                    ) VALUES (?, ?, ?, ?, ?, ?, ?)`,
                    id, update.Version, update.ReleaseNotes, update.Severity, update.UpdateURL, update.UpdateHash, update.UpdateSize,
                )
                if err != nil {
                    return err
                }
                
                // Get the update ID
                updateID64, err := result.LastInsertId()
                if err != nil {
                    return err
                }
                updateID = int(updateID64)
            } else if err != nil {
                return err
            } else {
                // Update the existing update
                _, err := tx.Exec(
                    `UPDATE software_updates SET
                        release_notes = ?,
                        severity = ?,
                        update_url = ?,
                        update_hash = ?,
                        update_size = ?,
                        updated_at = NOW()
                    WHERE id = ?`,
                    update.ReleaseNotes, update.Severity, update.UpdateURL, update.UpdateHash, update.UpdateSize, updateID,
                )
                if err != nil {
                    return err
                }
            }
        }
    }
    
    // Commit the transaction
    err = tx.Commit()
    if err != nil {
        return err
    }
    
    return nil
}

func CheckForUpdates(name, version, publisher string) ([]Update, error) {
    // This is a simplified implementation
    // In a real implementation, you would check various update sources
    
    // Example update sources:
    // - macOS: Apple Software Update catalog
    // - Windows: Windows Update API or WSUS
    // - Linux: Package manager repositories
    // - Third-party software: Vendor update APIs
    
    // For now, return an empty slice
    return []Update{}, nil
}
```

### Update Task Creation

Implement update task creation:

```go
func CreateUpdateTask(db *sql.DB, updateID, userID int, targets []Target) (int, error) {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return 0, err
    }
    defer tx.Rollback()
    
    // Create the task
    result, err := tx.Exec(
        "INSERT INTO software_update_tasks (update_id, status, created_by) VALUES (?, ?, ?)",
        updateID, "pending", userID,
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
            "INSERT INTO software_update_targets (task_id, type, target_id) VALUES (?, ?, ?)",
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

### Update Task Distribution

Implement update task distribution:

```go
func DistributeUpdateTask(db *sql.DB, taskID int) error {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Get the task
    var updateID int
    var status string
    err = tx.QueryRow(
        "SELECT update_id, status FROM software_update_tasks WHERE id = ?",
        taskID,
    ).Scan(&updateID, &status)
    if err != nil {
        return err
    }
    
    // Check if the task is pending
    if status != "pending" {
        return fmt.Errorf("task is not pending: %s", status)
    }
    
    // Get the update
    var softwareID int
    var version, updateURL, updateHash string
    var updateSize int64
    err = tx.QueryRow(
        `SELECT software_id, version, update_url, update_hash, update_size
         FROM software_updates WHERE id = ?`,
        updateID,
    ).Scan(&softwareID, &version, &updateURL, &updateHash, &updateSize)
    if err != nil {
        return err
    }
    
    // Get the software
    var name, currentVersion, platform string
    err = tx.QueryRow(
        "SELECT name, version, platform FROM software WHERE id = ?",
        softwareID,
    ).Scan(&name, &currentVersion, &platform)
    if err != nil {
        return err
    }
    
    // Get the target hosts
    rows, err := tx.Query(`
        SELECT DISTINCT h.id
        FROM hosts h
        JOIN host_software hs ON h.id = hs.host_id
        JOIN software_update_targets sut ON (
            (sut.type = 'host' AND sut.target_id = h.id) OR
            (sut.type = 'label' AND sut.target_id IN (
                SELECT label_id FROM host_labels WHERE host_id = h.id
            )) OR
            (sut.type = 'team' AND sut.target_id IN (
                SELECT team_id FROM host_teams WHERE host_id = h.id
            )) OR
            (sut.type = 'all')
        )
        WHERE sut.task_id = ? AND hs.software_id = ? AND h.platform = ?
    `, taskID, softwareID, platform)
    if err != nil {
        return err
    }
    defer rows.Close()
    
    // Create host updates
    for rows.Next() {
        var hostID int
        err := rows.Scan(&hostID)
        if err != nil {
            return err
        }
        
        // Check if the host update already exists
        var id int
        err = tx.QueryRow(
            "SELECT id FROM host_software_updates WHERE host_id = ? AND task_id = ?",
            hostID, taskID,
        ).Scan(&id)
        
        if err == sql.ErrNoRows {
            // Insert the host update
            _, err := tx.Exec(
                "INSERT INTO host_software_updates (host_id, task_id, status) VALUES (?, ?, ?)",
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
        "UPDATE software_update_tasks SET status = ?, updated_at = NOW() WHERE id = ?",
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

### Update Execution

Implement update execution on devices:

```go
func ExecuteUpdate(client *http.Client, serverURL, nodeKey string) error {
    // Get pending updates
    updates, err := GetPendingUpdates(client, serverURL, nodeKey)
    if err != nil {
        return err
    }
    
    // Execute each update
    for _, update := range updates {
        // Update update status
        err := UpdateUpdateStatus(client, serverURL, nodeKey, update.ID, "in_progress", "")
        if err != nil {
            return err
        }
        
        // Download the update
        updatePath, err := DownloadUpdate(update.UpdateURL, update.UpdateHash)
        if err != nil {
            UpdateUpdateStatus(client, serverURL, nodeKey, update.ID, "failed", err.Error())
            continue
        }
        
        // Install the update
        err = InstallUpdate(updatePath, update.SoftwareName, update.CurrentVersion, update.NewVersion, update.Platform)
        if err != nil {
            UpdateUpdateStatus(client, serverURL, nodeKey, update.ID, "failed", err.Error())
            continue
        }
        
        // Update update status
        err = UpdateUpdateStatus(client, serverURL, nodeKey, update.ID, "completed", "")
        if err != nil {
            return err
        }
    }
    
    return nil
}

func InstallUpdate(updatePath, softwareName, currentVersion, newVersion, platform string) error {
    switch platform {
    case "darwin":
        return InstallMacOSUpdate(updatePath, softwareName, currentVersion, newVersion)
    case "windows":
        return InstallWindowsUpdate(updatePath, softwareName, currentVersion, newVersion)
    case "linux":
        return InstallLinuxUpdate(updatePath, softwareName, currentVersion, newVersion)
    default:
        return fmt.Errorf("unsupported platform: %s", platform)
    }
}

func InstallMacOSUpdate(updatePath, softwareName, currentVersion, newVersion string) error {
    // Check if it's a system update
    if softwareName == "macOS" {
        cmd := exec.Command("softwareupdate", "--install", updatePath)
        return cmd.Run()
    }
    
    // Check if it's an App Store update
    if strings.HasPrefix(updatePath, "macappstore://") {
        appID := strings.TrimPrefix(updatePath, "macappstore://")
        cmd := exec.Command("softwareupdate", "--install", appID)
        return cmd.Run()
    }
    
    // Otherwise, treat it as a package
    cmd := exec.Command("installer", "-pkg", updatePath, "-target", "/")
    return cmd.Run()
}

func InstallWindowsUpdate(updatePath, softwareName, currentVersion, newVersion string) error {
    // Check if it's a Windows Update
    if softwareName == "Windows" {
        cmd := exec.Command("wusa", updatePath)
        return cmd.Run()
    }
    
    // Check if it's an MSI update
    if strings.HasSuffix(updatePath, ".msi") {
        cmd := exec.Command("msiexec", "/i", updatePath, "/qn")
        return cmd.Run()
    }
    
    // Otherwise, treat it as an executable
    cmd := exec.Command(updatePath, "/S")
    return cmd.Run()
}

func InstallLinuxUpdate(updatePath, softwareName, currentVersion, newVersion string) error {
    // Check if it's a system update
    if softwareName == "Linux" {
        // Determine the package manager
        if _, err := exec.LookPath("apt-get"); err == nil {
            cmd := exec.Command("apt-get", "upgrade", "-y")
            return cmd.Run()
        } else if _, err := exec.LookPath("yum"); err == nil {
            cmd := exec.Command("yum", "update", "-y")
            return cmd.Run()
        } else if _, err := exec.LookPath("dnf"); err == nil {
            cmd := exec.Command("dnf", "upgrade", "-y")
            return cmd.Run()
        } else {
            return fmt.Errorf("unsupported package manager")
        }
    }
    
    // Check if it's a DEB package
    if strings.HasSuffix(updatePath, ".deb") {
        cmd := exec.Command("dpkg", "-i", updatePath)
        return cmd.Run()
    }
    
    // Check if it's an RPM package
    if strings.HasSuffix(updatePath, ".rpm") {
        cmd := exec.Command("rpm", "-U", updatePath)
        return cmd.Run()
    }
    
    // Otherwise, treat it as a binary
    cmd := exec.Command("sh", "-c", fmt.Sprintf("chmod +x %s && %s", updatePath, updatePath))
    return cmd.Run()
}
```

### Update Status Reporting

Implement update status reporting:

```go
func UpdateUpdateStatus(client *http.Client, serverURL, nodeKey string, updateID int, status, errorMessage string) error {
    // Create the request body
    body := map[string]interface{}{
        "node_key": nodeKey,
        "update_id": updateID,
        "status": status,
        "error_message": errorMessage,
    }
    
    // Convert to JSON
    bodyJSON, err := json.Marshal(body)
    if err != nil {
        return err
    }
    
    // Create the request
    req, err := http.NewRequest("POST", serverURL+"/api/v1/fleet/software/updates/status", bytes.NewBuffer(bodyJSON))
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

Implement API endpoints for software updates:

### Get Available Updates

```go
func GetAvailableUpdatesHandler(w http.ResponseWriter, r *http.Request) {
    // Get query parameters
    hostID := r.URL.Query().Get("host_id")
    softwareID := r.URL.Query().Get("software_id")
    
    // Build the SQL query
    sqlQuery := `
        SELECT
            su.id,
            s.name,
            s.version AS current_version,
            su.version AS new_version,
            su.release_notes,
            su.severity,
            su.update_url,
            su.update_hash,
            su.update_size
        FROM software_updates su
        JOIN software s ON su.software_id = s.id
    `
    
    // Add conditions
    args := []interface{}{}
    if hostID != "" {
        sqlQuery += `
            JOIN host_software hs ON s.id = hs.software_id
            WHERE hs.host_id = ?
        `
        args = append(args, hostID)
    } else if softwareID != "" {
        sqlQuery += " WHERE s.id = ?"
        args = append(args, softwareID)
    }
    
    // Add order by
    sqlQuery += " ORDER BY su.created_at DESC"
    
    // Execute the query
    rows, err := db.Query(sqlQuery, args...)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting updates: %v", err), http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    
    // Process the results
    updates := []map[string]interface{}{}
    for rows.Next() {
        var id int
        var name, currentVersion, newVersion, releaseNotes, severity, updateURL, updateHash string
        var updateSize int64
        
        err := rows.Scan(
            &id,
            &name,
            &currentVersion,
            &newVersion,
            &releaseNotes,
            &severity,
            &updateURL,
            &updateHash,
            &updateSize,
        )
        if err != nil {
            http.Error(w, fmt.Sprintf("Error scanning update: %v", err), http.StatusInternalServerError)
            return
        }
        
        update := map[string]interface{}{
            "id": id,
            "name": name,
            "current_version": currentVersion,
            "new_version": newVersion,
            "release_notes": releaseNotes,
            "severity": severity,
            "update_url": updateURL,
            "update_hash": updateHash,
            "update_size": updateSize,
        }
        
        updates = append(updates, update)
    }
    
    // Return the updates
    json.NewEncoder(w).Encode(updates)
}
```

### Create Update Task

```go
func CreateUpdateTaskHandler(w http.ResponseWriter, r *http.Request) {
    // Parse the request body
    var req struct {
        UpdateID int      `json:"update_id"`
        Targets  []Target `json:"targets"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Get the user ID from the context
    userID := r.Context().Value("user_id").(int)
    
    // Create the task
    id, err := CreateUpdateTask(db, req.UpdateID, userID, req.Targets)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error creating task: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Distribute the task
    err = DistributeUpdateTask(db, id)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error distributing task: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the ID
    json.NewEncoder(w).Encode(map[string]int{"id": id})
}
```

### Get Pending Updates

```go
func GetPendingUpdatesHandler(w http.ResponseWriter, r *http.Request) {
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
    
    // Get pending updates for the host
    updates, err := GetPendingUpdatesForHost(db, hostID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting updates: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the updates
    json.NewEncoder(w).Encode(updates)
}
```

## Testing

### Manual Testing

1. Identify available updates for installed software
2. Create an update task
3. Verify the task is distributed to target devices
4. Check that devices execute the update
5. Verify the update status is reported back to the server

### Automated Testing

Fleet includes automated tests for Software Updates functionality:

```bash
# Run Software Updates tests
go test -v ./server/service/software_updates_test.go
```

## Debugging

### Update Identification Issues

- **Update Sources**: Verify the update sources are accessible
- **Version Comparison**: Ensure version comparison is correct
- **Update Metadata**: Check if update metadata is correctly retrieved

### Update Issues

- **Update Download**: Verify the update can be downloaded from the specified URL
- **Update Installation**: Ensure the update installation commands are correct
- **Error Handling**: Check if errors during update are properly handled

## Performance Considerations

Software Updates can impact system performance, especially for large updates or large fleets:

- **Update Size**: Consider the size of updates and their impact on network bandwidth
- **Update Timing**: Schedule updates during off-hours to minimize impact
- **Throttling**: Implement throttling to limit the number of concurrent updates
- **Caching**: Cache updates to reduce download time

## Related Resources

- [Software Updates Architecture](../../architecture/software/software-updates.md)
- [Software Product Group Documentation](../../product-groups/software/)
- [MDM Documentation](../../product-groups/mdm/)