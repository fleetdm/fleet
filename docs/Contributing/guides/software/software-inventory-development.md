# Software Inventory Development Guide

This guide provides instructions for developing Software Inventory functionality in Fleet.

## Introduction

Software Inventory in Fleet provides visibility into the software installed on devices across the fleet. This guide covers the development and implementation of Software Inventory features.

## Prerequisites

Before you begin developing Software Inventory functionality, you should have:

- A development environment set up according to the [Building Fleet](../../getting-started/building-fleet.md) guide
- Basic understanding of software inventory concepts
- Familiarity with Fleet's architecture
- Understanding of osquery and its capabilities

## Software Inventory Architecture

Software Inventory in Fleet follows a specific flow:

1. Devices collect software information using osquery
2. Devices send the information to the Fleet server
3. Fleet server processes and stores the information
4. Fleet server provides API endpoints for retrieving software inventory
5. Fleet UI displays the software inventory in tables and filters

## Implementation

### Database Schema

Software Inventory information is stored in the Fleet database:

```sql
CREATE TABLE software (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  version VARCHAR(255) NOT NULL,
  source VARCHAR(255),
  publisher VARCHAR(255),
  bundle_identifier VARCHAR(255),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY (name, version, source)
);

CREATE TABLE host_software (
  id INT AUTO_INCREMENT PRIMARY KEY,
  host_id INT NOT NULL,
  software_id INT NOT NULL,
  installed_path VARCHAR(255),
  installed_size BIGINT,
  installed_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (host_id) REFERENCES hosts(id),
  FOREIGN KEY (software_id) REFERENCES software(id),
  UNIQUE KEY (host_id, software_id)
);
```

### Software Collection

Implement software collection using osquery:

```go
func CollectSoftware() ([]map[string]interface{}, error) {
    software := []map[string]interface{}{}
    
    // Collect software based on platform
    switch runtime.GOOS {
    case "darwin":
        // Collect macOS applications
        appQuery := `
            SELECT
                name,
                bundle_identifier,
                bundle_short_version AS version,
                bundle_path AS installed_path,
                NULL AS publisher,
                'application' AS source
            FROM apps
        `
        appRows, err := osquery.Query(appQuery)
        if err != nil {
            return nil, err
        }
        software = append(software, appRows...)
        
        // Collect macOS packages
        pkgQuery := `
            SELECT
                name,
                NULL AS bundle_identifier,
                version,
                location AS installed_path,
                NULL AS publisher,
                'package' AS source
            FROM packages
        `
        pkgRows, err := osquery.Query(pkgQuery)
        if err != nil {
            return nil, err
        }
        software = append(software, pkgRows...)
        
    case "windows":
        // Collect Windows programs
        programQuery := `
            SELECT
                name,
                NULL AS bundle_identifier,
                version,
                install_location AS installed_path,
                publisher,
                'program' AS source
            FROM programs
        `
        programRows, err := osquery.Query(programQuery)
        if err != nil {
            return nil, err
        }
        software = append(software, programRows...)
        
        // Collect Windows app packages
        appPackageQuery := `
            SELECT
                name,
                package_full_name AS bundle_identifier,
                version,
                install_location AS installed_path,
                publisher,
                'appx' AS source
            FROM appcompat_shims
        `
        appPackageRows, err := osquery.Query(appPackageQuery)
        if err != nil {
            return nil, err
        }
        software = append(software, appPackageRows...)
        
    case "linux":
        // Collect Linux packages
        packageQuery := `
            SELECT
                name,
                NULL AS bundle_identifier,
                version,
                path AS installed_path,
                maintainer AS publisher,
                'package' AS source
            FROM deb_packages
            UNION
            SELECT
                name,
                NULL AS bundle_identifier,
                version,
                NULL AS installed_path,
                vendor AS publisher,
                'package' AS source
            FROM rpm_packages
        `
        packageRows, err := osquery.Query(packageQuery)
        if err != nil {
            return nil, err
        }
        software = append(software, packageRows...)
    }
    
    return software, nil
}
```

### Software Submission

Implement software submission to the Fleet server:

```go
func SubmitSoftware(client *http.Client, serverURL string, nodeKey string, software []map[string]interface{}) error {
    // Create the request body
    body := map[string]interface{}{
        "node_key": nodeKey,
        "software": software,
    }
    
    // Convert to JSON
    bodyJSON, err := json.Marshal(body)
    if err != nil {
        return err
    }
    
    // Create the request
    req, err := http.NewRequest("POST", serverURL+"/api/v1/fleet/software", bytes.NewBuffer(bodyJSON))
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

### Software Processing

Implement software processing on the Fleet server:

```go
func ProcessSoftware(db *sql.DB, hostID int, software []map[string]interface{}) error {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Get existing software for the host
    rows, err := tx.Query(`
        SELECT s.id, s.name, s.version, s.source
        FROM software s
        JOIN host_software hs ON s.id = hs.software_id
        WHERE hs.host_id = ?
    `, hostID)
    if err != nil {
        return err
    }
    defer rows.Close()
    
    // Build a map of existing software
    existingSoftware := map[string]int{}
    for rows.Next() {
        var id int
        var name, version, source string
        err := rows.Scan(&id, &name, &version, &source)
        if err != nil {
            return err
        }
        key := fmt.Sprintf("%s|%s|%s", name, version, source)
        existingSoftware[key] = id
    }
    
    // Process each software item
    processedSoftware := map[string]bool{}
    for _, item := range software {
        name := item["name"].(string)
        version := item["version"].(string)
        source := item["source"].(string)
        bundleIdentifier := item["bundle_identifier"]
        publisher := item["publisher"]
        installedPath := item["installed_path"]
        
        // Create a key for the software
        key := fmt.Sprintf("%s|%s|%s", name, version, source)
        processedSoftware[key] = true
        
        // Check if the software already exists
        softwareID, exists := existingSoftware[key]
        if !exists {
            // Insert the software
            var result sql.Result
            if bundleIdentifier != nil {
                result, err = tx.Exec(
                    "INSERT INTO software (name, version, source, bundle_identifier, publisher) VALUES (?, ?, ?, ?, ?)",
                    name, version, source, bundleIdentifier, publisher,
                )
            } else {
                result, err = tx.Exec(
                    "INSERT INTO software (name, version, source, publisher) VALUES (?, ?, ?, ?)",
                    name, version, source, publisher,
                )
            }
            if err != nil {
                return err
            }
            
            // Get the software ID
            softwareID64, err := result.LastInsertId()
            if err != nil {
                return err
            }
            softwareID = int(softwareID64)
        }
        
        // Insert or update the host_software relationship
        if installedPath != nil {
            _, err = tx.Exec(
                `INSERT INTO host_software (host_id, software_id, installed_path)
                 VALUES (?, ?, ?)
                 ON DUPLICATE KEY UPDATE installed_path = ?`,
                hostID, softwareID, installedPath, installedPath,
            )
        } else {
            _, err = tx.Exec(
                `INSERT INTO host_software (host_id, software_id)
                 VALUES (?, ?)
                 ON DUPLICATE KEY UPDATE software_id = software_id`,
                hostID, softwareID,
            )
        }
        if err != nil {
            return err
        }
    }
    
    // Remove software that is no longer installed
    for key, softwareID := range existingSoftware {
        if !processedSoftware[key] {
            _, err = tx.Exec(
                "DELETE FROM host_software WHERE host_id = ? AND software_id = ?",
                hostID, softwareID,
            )
            if err != nil {
                return err
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
```

### Software Retrieval

Implement software retrieval from the Fleet server:

```go
func GetHostSoftware(db *sql.DB, hostID int) ([]map[string]interface{}, error) {
    // Query the database
    rows, err := db.Query(`
        SELECT
            s.id,
            s.name,
            s.version,
            s.source,
            s.publisher,
            s.bundle_identifier,
            hs.installed_path,
            hs.installed_size,
            hs.installed_at
        FROM software s
        JOIN host_software hs ON s.id = hs.software_id
        WHERE hs.host_id = ?
    `, hostID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    // Process the results
    software := []map[string]interface{}{}
    for rows.Next() {
        var id int
        var name, version, source string
        var publisher, bundleIdentifier, installedPath sql.NullString
        var installedSize sql.NullInt64
        var installedAt sql.NullTime
        
        err := rows.Scan(
            &id,
            &name,
            &version,
            &source,
            &publisher,
            &bundleIdentifier,
            &installedPath,
            &installedSize,
            &installedAt,
        )
        if err != nil {
            return nil, err
        }
        
        item := map[string]interface{}{
            "id": id,
            "name": name,
            "version": version,
            "source": source,
        }
        
        if publisher.Valid {
            item["publisher"] = publisher.String
        }
        
        if bundleIdentifier.Valid {
            item["bundle_identifier"] = bundleIdentifier.String
        }
        
        if installedPath.Valid {
            item["installed_path"] = installedPath.String
        }
        
        if installedSize.Valid {
            item["installed_size"] = installedSize.Int64
        }
        
        if installedAt.Valid {
            item["installed_at"] = installedAt.Time.Unix()
        }
        
        software = append(software, item)
    }
    
    return software, nil
}
```

## API Endpoints

Implement API endpoints for software inventory:

### Submit Software

```go
func SubmitSoftwareHandler(w http.ResponseWriter, r *http.Request) {
    // Parse the request body
    var req struct {
        NodeKey  string                   `json:"node_key"`
        Software []map[string]interface{} `json:"software"`
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
    
    // Process the software
    err = ProcessSoftware(db, hostID, req.Software)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error processing software: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return success
    w.WriteHeader(http.StatusOK)
}
```

### Get Host Software

```go
func GetHostSoftwareHandler(w http.ResponseWriter, r *http.Request) {
    // Get the host ID from the URL
    vars := mux.Vars(r)
    hostID, err := strconv.Atoi(vars["host_id"])
    if err != nil {
        http.Error(w, "Invalid host ID", http.StatusBadRequest)
        return
    }
    
    // Get the software
    software, err := GetHostSoftware(db, hostID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting software: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the software
    json.NewEncoder(w).Encode(software)
}
```

## Testing

### Manual Testing

1. Implement software collection on a test device
2. Submit software to the Fleet server
3. Verify software is stored in the database
4. Retrieve and display software in the UI

### Automated Testing

Fleet includes automated tests for Software Inventory functionality:

```bash
# Run Software Inventory tests
go test -v ./server/service/software_inventory_test.go
```

## Debugging

### Software Collection Issues

- **osquery Queries**: Verify the osquery queries are correctly retrieving software information
- **Platform-Specific Logic**: Ensure the platform-specific collection logic is correct
- **Error Handling**: Check if errors during software collection are properly handled

### Software Processing Issues

- **Database Schema**: Verify the database schema is correctly defined
- **Data Insertion**: Ensure software information is correctly inserted into the database
- **Transaction Management**: Check if database transactions are properly managed

## Performance Considerations

Software Inventory can generate a significant amount of data, especially for large fleets:

- **Collection Frequency**: Consider the frequency of software collection
- **Data Deduplication**: Implement deduplication to reduce storage requirements
- **Indexing**: Ensure the database is properly indexed for efficient queries
- **Pagination**: Implement pagination for retrieving large software inventories

## Related Resources

- [Software Inventory Architecture](../../architecture/software/software-inventory.md)
- [Software Product Group Documentation](../../product-groups/software/)
- [osquery Documentation](https://osquery.readthedocs.io/)