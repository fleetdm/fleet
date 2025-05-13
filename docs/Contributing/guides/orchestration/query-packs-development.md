# Query Packs Development Guide

This guide provides instructions for developing Query Packs functionality in Fleet.

## Introduction

Query Packs in Fleet allow users to group related queries together for easier management and distribution. This guide covers the development and implementation of Query Packs features.

## Prerequisites

Before you begin developing Query Packs functionality, you should have:

- A development environment set up according to the [Building Fleet](../../getting-started/building-fleet.md) guide
- Basic understanding of SQL and osquery
- Familiarity with Fleet's architecture
- Understanding of configuration management

## Query Packs Architecture

Query Packs in Fleet follow a specific flow:

1. User creates a query pack through the UI or API
2. Fleet server stores the pack configuration in the database
3. Devices check in with the Fleet server and receive the configuration
4. Devices execute the queries in the pack according to their schedules
5. Devices return results to the Fleet server
6. Fleet server processes and stores the results

## Implementation

### Database Schema

Query Pack information is stored in the Fleet database:

```sql
CREATE TABLE packs (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  platform VARCHAR(255),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE pack_targets (
  id INT AUTO_INCREMENT PRIMARY KEY,
  pack_id INT NOT NULL,
  type VARCHAR(255) NOT NULL,
  target_id INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (pack_id) REFERENCES packs(id)
);

CREATE TABLE pack_queries (
  id INT AUTO_INCREMENT PRIMARY KEY,
  pack_id INT NOT NULL,
  query_id INT NOT NULL,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  interval INT NOT NULL,
  snapshot BOOLEAN NOT NULL DEFAULT FALSE,
  removed BOOLEAN NOT NULL DEFAULT FALSE,
  platform VARCHAR(255),
  version VARCHAR(255),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (pack_id) REFERENCES packs(id),
  FOREIGN KEY (query_id) REFERENCES queries(id)
);

CREATE TABLE pack_stats (
  id INT AUTO_INCREMENT PRIMARY KEY,
  pack_id INT NOT NULL,
  host_id INT NOT NULL,
  last_executed TIMESTAMP,
  last_error VARCHAR(255),
  executions INT NOT NULL DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (pack_id) REFERENCES packs(id),
  FOREIGN KEY (host_id) REFERENCES hosts(id)
);
```

### Pack Creation

Implement pack creation:

```go
func CreatePack(db *sql.DB, name, description, platform string, targets []Target) (int, error) {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return 0, err
    }
    defer tx.Rollback()
    
    // Create the pack
    result, err := tx.Exec(
        "INSERT INTO packs (name, description, platform) VALUES (?, ?, ?)",
        name, description, platform,
    )
    if err != nil {
        return 0, err
    }
    
    // Get the pack ID
    packID, err := result.LastInsertId()
    if err != nil {
        return 0, err
    }
    
    // Add targets
    for _, target := range targets {
        _, err := tx.Exec(
            "INSERT INTO pack_targets (pack_id, type, target_id) VALUES (?, ?, ?)",
            packID, target.Type, target.ID,
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
    
    return int(packID), nil
}
```

### Adding Queries to a Pack

Implement adding queries to a pack:

```go
func AddQueryToPack(db *sql.DB, packID int, name, description, queryString string, interval int, snapshot bool, platform, version string) (int, error) {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return 0, err
    }
    defer tx.Rollback()
    
    // Create the query
    result, err := tx.Exec(
        "INSERT INTO queries (query) VALUES (?)",
        queryString,
    )
    if err != nil {
        return 0, err
    }
    
    // Get the query ID
    queryID, err := result.LastInsertId()
    if err != nil {
        return 0, err
    }
    
    // Add the query to the pack
    result, err = tx.Exec(
        "INSERT INTO pack_queries (pack_id, query_id, name, description, interval, snapshot, platform, version) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
        packID, queryID, name, description, interval, snapshot, platform, version,
    )
    if err != nil {
        return 0, err
    }
    
    // Get the pack query ID
    packQueryID, err := result.LastInsertId()
    if err != nil {
        return 0, err
    }
    
    // Commit the transaction
    err = tx.Commit()
    if err != nil {
        return 0, err
    }
    
    return int(packQueryID), nil
}
```

### Configuration Generation

Implement configuration generation for devices:

```go
func GenerateOsqueryConfigWithPacks(db *sql.DB, hostID int) (map[string]interface{}, error) {
    // Create the base configuration
    config := map[string]interface{}{
        "options": map[string]interface{}{
            "logger_tls_period": 10,
            "distributed_interval": 10,
            "config_tls_refresh": 10,
        },
        "schedule": map[string]interface{}{},
        "packs": map[string]interface{}{},
        "decorators": map[string]interface{}{
            "load": []string{
                "SELECT uuid AS host_uuid FROM system_info",
                "SELECT hostname AS hostname FROM system_info",
            },
        },
    }
    
    // Get packs for this host
    rows, err := db.Query(`
        SELECT p.id, p.name, p.description, p.platform
        FROM packs p
        JOIN pack_targets pt ON p.id = pt.pack_id
        WHERE (
            (pt.type = 'host' AND pt.target_id = ?)
            OR (pt.type = 'label' AND pt.target_id IN (
                SELECT label_id FROM host_labels WHERE host_id = ?
            ))
            OR (pt.type = 'team' AND pt.target_id IN (
                SELECT team_id FROM host_teams WHERE host_id = ?
            ))
            OR (pt.type = 'all')
        )
        AND (p.platform IS NULL OR p.platform = (SELECT platform FROM hosts WHERE id = ?))
    `, hostID, hostID, hostID, hostID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    // Process the packs
    packs := config["packs"].(map[string]interface{})
    for rows.Next() {
        var id int
        var name, description string
        var platform sql.NullString
        
        err := rows.Scan(&id, &name, &description, &platform)
        if err != nil {
            return nil, err
        }
        
        // Create the pack configuration
        pack := map[string]interface{}{
            "queries": map[string]interface{}{},
        }
        
        // Get queries for this pack
        queryRows, err := db.Query(`
            SELECT pq.name, q.query, pq.interval, pq.snapshot, pq.platform, pq.version
            FROM pack_queries pq
            JOIN queries q ON pq.query_id = q.id
            WHERE pq.pack_id = ?
            AND pq.removed = FALSE
            AND (pq.platform IS NULL OR pq.platform = (SELECT platform FROM hosts WHERE id = ?))
            AND (pq.version IS NULL OR pq.version = (SELECT os_version FROM hosts WHERE id = ?))
        `, id, hostID, hostID)
        if err != nil {
            return nil, err
        }
        
        // Process the queries
        queries := pack["queries"].(map[string]interface{})
        for queryRows.Next() {
            var name, query string
            var interval int
            var snapshot bool
            var platform, version sql.NullString
            
            err := queryRows.Scan(&name, &query, &interval, &snapshot, &platform, &version)
            if err != nil {
                return nil, err
            }
            
            // Add the query to the pack
            queries[name] = map[string]interface{}{
                "query": query,
                "interval": interval,
                "snapshot": snapshot,
            }
        }
        queryRows.Close()
        
        // Add the pack to the configuration
        packs[fmt.Sprintf("fleet_pack_%d", id)] = pack
    }
    
    return config, nil
}
```

### Result Processing

Implement result processing from devices:

```go
func ProcessPackQueryResults(db *sql.DB, hostID int, results map[string]interface{}) error {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Process each pack
    for packName, packResults := range results {
        // Extract the pack ID from the name
        var packID int
        _, err := fmt.Sscanf(packName, "fleet_pack_%d", &packID)
        if err != nil {
            continue
        }
        
        // Update the pack stats
        _, err = tx.Exec(
            `INSERT INTO pack_stats (pack_id, host_id, last_executed, executions)
             VALUES (?, ?, NOW(), 1)
             ON DUPLICATE KEY UPDATE last_executed = NOW(), executions = executions + 1`,
            packID, hostID,
        )
        if err != nil {
            return err
        }
        
        // Process the pack results
        packResultsMap, ok := packResults.(map[string]interface{})
        if !ok {
            continue
        }
        
        // Process each query in the pack
        for queryName, queryResult := range packResultsMap {
            // Get the query ID
            var queryID int
            err := tx.QueryRow(
                "SELECT id FROM pack_queries WHERE pack_id = ? AND name = ?",
                packID, queryName,
            ).Scan(&queryID)
            if err != nil {
                continue
            }
            
            // Process the query result
            queryResultMap, ok := queryResult.(map[string]interface{})
            if !ok {
                continue
            }
            
            // Check for errors
            if status, ok := queryResultMap["status"].(float64); ok && status != 0 {
                message := queryResultMap["message"].(string)
                _, err = tx.Exec(
                    "UPDATE pack_stats SET last_error = ? WHERE pack_id = ? AND host_id = ?",
                    message, packID, hostID,
                )
                if err != nil {
                    return err
                }
                continue
            }
            
            // Process the rows
            rows, ok := queryResultMap["results"].([]interface{})
            if !ok {
                continue
            }
            
            for _, row := range rows {
                // Convert row to JSON
                rowJSON, err := json.Marshal(row)
                if err != nil {
                    return err
                }
                
                // Save the row
                _, err = tx.Exec(
                    "INSERT INTO pack_query_results (pack_id, query_id, host_id, result, timestamp) VALUES (?, ?, ?, ?, NOW())",
                    packID, queryID, hostID, rowJSON,
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
```

## API Endpoints

Implement API endpoints for query packs:

### Create Pack

```go
func CreatePackHandler(w http.ResponseWriter, r *http.Request) {
    // Parse the request body
    var req struct {
        Name        string   `json:"name"`
        Description string   `json:"description"`
        Platform    string   `json:"platform"`
        Targets     []Target `json:"targets"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Create the pack
    id, err := CreatePack(db, req.Name, req.Description, req.Platform, req.Targets)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error creating pack: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the ID
    json.NewEncoder(w).Encode(map[string]int{"id": id})
}
```

### Add Query to Pack

```go
func AddQueryToPackHandler(w http.ResponseWriter, r *http.Request) {
    // Get the pack ID from the URL
    vars := mux.Vars(r)
    packID, err := strconv.Atoi(vars["pack_id"])
    if err != nil {
        http.Error(w, "Invalid pack ID", http.StatusBadRequest)
        return
    }
    
    // Parse the request body
    var req struct {
        Name        string `json:"name"`
        Description string `json:"description"`
        Query       string `json:"query"`
        Interval    int    `json:"interval"`
        Snapshot    bool   `json:"snapshot"`
        Platform    string `json:"platform"`
        Version     string `json:"version"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Validate the query
    if err := ValidateQuery(req.Query); err != nil {
        http.Error(w, fmt.Sprintf("Invalid query: %v", err), http.StatusBadRequest)
        return
    }
    
    // Add the query to the pack
    id, err := AddQueryToPack(db, packID, req.Name, req.Description, req.Query, req.Interval, req.Snapshot, req.Platform, req.Version)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error adding query to pack: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the ID
    json.NewEncoder(w).Encode(map[string]int{"id": id})
}
```

### Get Packs

```go
func GetPacksHandler(w http.ResponseWriter, r *http.Request) {
    // Get the packs
    packs, err := GetPacks(db)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting packs: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the packs
    json.NewEncoder(w).Encode(packs)
}
```

## Configuration File Format

The osquery configuration file format for query packs:

```json
{
  "options": {
    "logger_tls_period": 10,
    "distributed_interval": 10,
    "config_tls_refresh": 10
  },
  "schedule": {},
  "packs": {
    "fleet_pack_1": {
      "queries": {
        "process_events": {
          "query": "SELECT * FROM process_events",
          "interval": 60,
          "snapshot": false
        },
        "socket_events": {
          "query": "SELECT * FROM socket_events",
          "interval": 60,
          "snapshot": false
        }
      }
    },
    "fleet_pack_2": {
      "queries": {
        "user_accounts": {
          "query": "SELECT * FROM users",
          "interval": 3600,
          "snapshot": true
        },
        "installed_software": {
          "query": "SELECT * FROM programs",
          "interval": 3600,
          "snapshot": true
        }
      }
    }
  },
  "decorators": {
    "load": [
      "SELECT uuid AS host_uuid FROM system_info",
      "SELECT hostname AS hostname FROM system_info"
    ]
  }
}
```

## Testing

### Manual Testing

1. Create a query pack through the API
2. Add queries to the pack
3. Verify the pack is included in the osquery configuration
4. Check that results are returned and stored
5. Test pack targeting and filtering

### Automated Testing

Fleet includes automated tests for Query Packs functionality:

```bash
# Run Query Packs tests
go test -v ./server/service/packs_test.go
```

## Debugging

### Configuration Issues

- **Query Validation**: Verify the queries in the pack are valid and can be executed by osquery
- **Configuration Generation**: Ensure the osquery configuration with packs is correctly generated
- **Device Check-in**: Check if devices are retrieving the configuration

### Result Processing Issues

- **Result Format**: Verify the result format matches the expected format
- **Error Handling**: Ensure errors are properly captured and stored
- **Result Storage**: Check if results are correctly stored in the database

## Performance Considerations

Query Packs can impact system performance, especially for packs with many queries or frequent execution:

- **Query Count**: Packs with many queries consume more resources
- **Query Frequency**: More frequent queries consume more resources
- **Query Complexity**: Complex queries can consume significant CPU resources
- **Result Size**: Large result sets can consume significant storage space

## Related Resources

- [Query Packs Architecture](../../architecture/orchestration/query-packs.md)
- [osquery Documentation](https://osquery.readthedocs.io/)