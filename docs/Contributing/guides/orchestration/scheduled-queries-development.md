# Scheduled Queries Development Guide

This guide provides instructions for developing Scheduled Queries functionality in Fleet.

## Introduction

Scheduled Queries in Fleet allow users to configure queries that run on a regular schedule, providing ongoing visibility into device status, configuration, and security posture. This guide covers the development and implementation of Scheduled Queries features.

## Prerequisites

Before you begin developing Scheduled Queries functionality, you should have:

- A development environment set up according to the [Building Fleet](../../getting-started/building-fleet.md) guide
- Basic understanding of SQL and osquery
- Familiarity with Fleet's architecture
- Understanding of configuration management

## Scheduled Query Architecture

Scheduled Queries in Fleet follow a specific flow:

1. User creates a scheduled query through the UI or API
2. Fleet server stores the query configuration in the database
3. Devices check in with the Fleet server and receive the configuration
4. Devices execute the queries according to the schedule
5. Devices return results to the Fleet server
6. Fleet server processes and stores the results

## Implementation

### Database Schema

Scheduled Query information is stored in the Fleet database:

```sql
CREATE TABLE scheduled_queries (
  id INT AUTO_INCREMENT PRIMARY KEY,
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
  FOREIGN KEY (query_id) REFERENCES queries(id)
);

CREATE TABLE scheduled_query_targets (
  id INT AUTO_INCREMENT PRIMARY KEY,
  scheduled_query_id INT NOT NULL,
  type VARCHAR(255) NOT NULL,
  target_id INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (scheduled_query_id) REFERENCES scheduled_queries(id)
);

CREATE TABLE scheduled_query_stats (
  id INT AUTO_INCREMENT PRIMARY KEY,
  scheduled_query_id INT NOT NULL,
  host_id INT NOT NULL,
  last_executed TIMESTAMP,
  last_error VARCHAR(255),
  executions INT NOT NULL DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (scheduled_query_id) REFERENCES scheduled_queries(id),
  FOREIGN KEY (host_id) REFERENCES hosts(id)
);
```

### Scheduled Query Creation

Implement scheduled query creation:

```go
func CreateScheduledQuery(db *sql.DB, name, description, queryString string, interval int, snapshot bool, platform, version string, targets []Target) (int, error) {
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
    
    // Create the scheduled query
    result, err = tx.Exec(
        "INSERT INTO scheduled_queries (query_id, name, description, interval, snapshot, platform, version) VALUES (?, ?, ?, ?, ?, ?, ?)",
        queryID, name, description, interval, snapshot, platform, version,
    )
    if err != nil {
        return 0, err
    }
    
    // Get the scheduled query ID
    scheduledQueryID, err := result.LastInsertId()
    if err != nil {
        return 0, err
    }
    
    // Add targets
    for _, target := range targets {
        _, err := tx.Exec(
            "INSERT INTO scheduled_query_targets (scheduled_query_id, type, target_id) VALUES (?, ?, ?)",
            scheduledQueryID, target.Type, target.ID,
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
    
    return int(scheduledQueryID), nil
}
```

### Configuration Generation

Implement configuration generation for devices:

```go
func GenerateOsqueryConfig(db *sql.DB, hostID int) (map[string]interface{}, error) {
    // Create the base configuration
    config := map[string]interface{}{
        "options": map[string]interface{}{
            "logger_tls_period": 10,
            "distributed_interval": 10,
            "config_tls_refresh": 10,
        },
        "schedule": map[string]interface{}{},
        "decorators": map[string]interface{}{
            "load": []string{
                "SELECT uuid AS host_uuid FROM system_info",
                "SELECT hostname AS hostname FROM system_info",
            },
        },
    }
    
    // Get scheduled queries for this host
    rows, err := db.Query(`
        SELECT sq.id, sq.name, q.query, sq.interval, sq.snapshot, sq.platform, sq.version
        FROM scheduled_queries sq
        JOIN queries q ON sq.query_id = q.id
        JOIN scheduled_query_targets sqt ON sq.id = sqt.scheduled_query_id
        WHERE sq.removed = FALSE
        AND (
            (sqt.type = 'host' AND sqt.target_id = ?)
            OR (sqt.type = 'label' AND sqt.target_id IN (
                SELECT label_id FROM host_labels WHERE host_id = ?
            ))
            OR (sqt.type = 'team' AND sqt.target_id IN (
                SELECT team_id FROM host_teams WHERE host_id = ?
            ))
            OR (sqt.type = 'all')
        )
        AND (sq.platform IS NULL OR sq.platform = (SELECT platform FROM hosts WHERE id = ?))
        AND (sq.version IS NULL OR sq.version = (SELECT os_version FROM hosts WHERE id = ?))
    `, hostID, hostID, hostID, hostID, hostID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    // Process the results
    schedule := config["schedule"].(map[string]interface{})
    for rows.Next() {
        var id int
        var name, query string
        var interval int
        var snapshot bool
        var platform, version sql.NullString
        
        err := rows.Scan(&id, &name, &query, &interval, &snapshot, &platform, &version)
        if err != nil {
            return nil, err
        }
        
        // Add the query to the schedule
        schedule[fmt.Sprintf("fleet_query_%d", id)] = map[string]interface{}{
            "query": query,
            "interval": interval,
            "snapshot": snapshot,
        }
    }
    
    return config, nil
}
```

### Result Processing

Implement result processing from devices:

```go
func ProcessScheduledQueryResults(db *sql.DB, hostID int, results map[string]interface{}) error {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Process each result
    for name, result := range results {
        // Extract the query ID from the name
        var queryID int
        _, err := fmt.Sscanf(name, "fleet_query_%d", &queryID)
        if err != nil {
            continue
        }
        
        // Update the stats
        _, err = tx.Exec(
            `INSERT INTO scheduled_query_stats (scheduled_query_id, host_id, last_executed, executions)
             VALUES (?, ?, NOW(), 1)
             ON DUPLICATE KEY UPDATE last_executed = NOW(), executions = executions + 1`,
            queryID, hostID,
        )
        if err != nil {
            return err
        }
        
        // Process the result rows
        resultMap, ok := result.(map[string]interface{})
        if !ok {
            continue
        }
        
        // Check for errors
        if status, ok := resultMap["status"].(float64); ok && status != 0 {
            message := resultMap["message"].(string)
            _, err = tx.Exec(
                "UPDATE scheduled_query_stats SET last_error = ? WHERE scheduled_query_id = ? AND host_id = ?",
                message, queryID, hostID,
            )
            if err != nil {
                return err
            }
            continue
        }
        
        // Process the rows
        rows, ok := resultMap["results"].([]interface{})
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
                "INSERT INTO scheduled_query_results (scheduled_query_id, host_id, result, timestamp) VALUES (?, ?, ?, NOW())",
                queryID, hostID, rowJSON,
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

## API Endpoints

Implement API endpoints for scheduled queries:

### Create Scheduled Query

```go
func CreateScheduledQueryHandler(w http.ResponseWriter, r *http.Request) {
    // Parse the request body
    var req struct {
        Name        string   `json:"name"`
        Description string   `json:"description"`
        Query       string   `json:"query"`
        Interval    int      `json:"interval"`
        Snapshot    bool     `json:"snapshot"`
        Platform    string   `json:"platform"`
        Version     string   `json:"version"`
        Targets     []Target `json:"targets"`
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
    
    // Create the scheduled query
    id, err := CreateScheduledQuery(db, req.Name, req.Description, req.Query, req.Interval, req.Snapshot, req.Platform, req.Version, req.Targets)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error creating scheduled query: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the ID
    json.NewEncoder(w).Encode(map[string]int{"id": id})
}
```

### Get Scheduled Queries

```go
func GetScheduledQueriesHandler(w http.ResponseWriter, r *http.Request) {
    // Get the scheduled queries
    queries, err := GetScheduledQueries(db)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting scheduled queries: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the queries
    json.NewEncoder(w).Encode(queries)
}
```

### Get Scheduled Query Results

```go
func GetScheduledQueryResultsHandler(w http.ResponseWriter, r *http.Request) {
    // Get the query ID from the URL
    vars := mux.Vars(r)
    queryID := vars["query_id"]
    
    // Get the results
    results, err := GetScheduledQueryResults(db, queryID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting results: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the results
    json.NewEncoder(w).Encode(results)
}
```

## Configuration File Format

The osquery configuration file format for scheduled queries:

```json
{
  "options": {
    "logger_tls_period": 10,
    "distributed_interval": 10,
    "config_tls_refresh": 10
  },
  "schedule": {
    "fleet_query_1": {
      "query": "SELECT * FROM processes",
      "interval": 60,
      "snapshot": false
    },
    "fleet_query_2": {
      "query": "SELECT * FROM users",
      "interval": 3600,
      "snapshot": true
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

1. Create a scheduled query through the API
2. Verify the query is included in the osquery configuration
3. Check that results are returned and stored
4. Test query targeting and filtering

### Automated Testing

Fleet includes automated tests for Scheduled Queries functionality:

```bash
# Run Scheduled Queries tests
go test -v ./server/service/scheduled_query_test.go
```

## Debugging

### Configuration Issues

- **Query Validation**: Verify the query is valid and can be executed by osquery
- **Configuration Generation**: Ensure the osquery configuration is correctly generated
- **Device Check-in**: Check if devices are retrieving the configuration

### Result Processing Issues

- **Result Format**: Verify the result format matches the expected format
- **Error Handling**: Ensure errors are properly captured and stored
- **Result Storage**: Check if results are correctly stored in the database

## Performance Considerations

Scheduled Queries can impact system performance, especially for frequent queries or large result sets:

- **Query Frequency**: More frequent queries consume more resources
- **Query Complexity**: Complex queries can consume significant CPU resources
- **Result Size**: Large result sets can consume significant storage space
- **Device Count**: Managing configurations for a large number of devices can impact server performance

## Related Resources

- [Scheduled Queries Architecture](../../architecture/orchestration/scheduled-queries.md)
- [osquery Documentation](https://osquery.readthedocs.io/)