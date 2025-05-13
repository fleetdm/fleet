# Live Queries Development Guide

This guide provides instructions for developing Live Queries functionality in Fleet.

## Introduction

Live Queries in Fleet allow users to execute ad-hoc queries against devices in real-time, providing immediate visibility into device status, configuration, and security posture. This guide covers the development and implementation of Live Queries features.

## Prerequisites

Before you begin developing Live Queries functionality, you should have:

- A development environment set up according to the [Building Fleet](../../getting-started/building-fleet.md) guide
- Basic understanding of SQL and osquery
- Familiarity with Fleet's architecture
- Understanding of WebSockets for real-time communication

## Live Query Architecture

Live Queries in Fleet follow a specific flow:

1. User initiates a query through the UI or API
2. Fleet server creates a campaign for the query
3. Devices check in with the Fleet server and receive the query
4. Devices execute the query and return results
5. Fleet server processes and displays the results

## Implementation

### Database Schema

Live Query information is stored in the Fleet database:

```sql
CREATE TABLE distributed_query_campaigns (
  id INT AUTO_INCREMENT PRIMARY KEY,
  query_id INT NOT NULL,
  status VARCHAR(255) NOT NULL,
  user_id INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (query_id) REFERENCES queries(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE distributed_query_campaign_targets (
  id INT AUTO_INCREMENT PRIMARY KEY,
  campaign_id INT NOT NULL,
  type VARCHAR(255) NOT NULL,
  target_id INT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (campaign_id) REFERENCES distributed_query_campaigns(id)
);

CREATE TABLE distributed_query_executions (
  id INT AUTO_INCREMENT PRIMARY KEY,
  campaign_id INT NOT NULL,
  host_id INT NOT NULL,
  status VARCHAR(255) NOT NULL,
  error VARCHAR(255),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (campaign_id) REFERENCES distributed_query_campaigns(id),
  FOREIGN KEY (host_id) REFERENCES hosts(id)
);
```

### Campaign Creation

Implement campaign creation for live queries:

```go
func CreateLiveQueryCampaign(db *sql.DB, queryID, userID int, targets []Target) (int, error) {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return 0, err
    }
    defer tx.Rollback()
    
    // Create the campaign
    result, err := tx.Exec(
        "INSERT INTO distributed_query_campaigns (query_id, status, user_id) VALUES (?, ?, ?)",
        queryID, "new", userID,
    )
    if err != nil {
        return 0, err
    }
    
    // Get the campaign ID
    campaignID, err := result.LastInsertId()
    if err != nil {
        return 0, err
    }
    
    // Add targets
    for _, target := range targets {
        _, err := tx.Exec(
            "INSERT INTO distributed_query_campaign_targets (campaign_id, type, target_id) VALUES (?, ?, ?)",
            campaignID, target.Type, target.ID,
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
    
    return int(campaignID), nil
}
```

### Query Distribution

Implement query distribution to devices:

```go
func GetQueriesForHost(db *sql.DB, hostID int) ([]DistributedQuery, error) {
    // Query for campaigns targeting this host
    rows, err := db.Query(`
        SELECT dqc.id, q.query
        FROM distributed_query_campaigns dqc
        JOIN queries q ON dqc.query_id = q.id
        JOIN distributed_query_campaign_targets dqct ON dqc.id = dqct.campaign_id
        LEFT JOIN distributed_query_executions dqe ON dqc.id = dqe.campaign_id AND dqe.host_id = ?
        WHERE dqc.status = 'new'
        AND (
            (dqct.type = 'host' AND dqct.target_id = ?)
            OR (dqct.type = 'label' AND dqct.target_id IN (
                SELECT label_id FROM host_labels WHERE host_id = ?
            ))
            OR (dqct.type = 'team' AND dqct.target_id IN (
                SELECT team_id FROM host_teams WHERE host_id = ?
            ))
            OR (dqct.type = 'all')
        )
        AND dqe.id IS NULL
    `, hostID, hostID, hostID, hostID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    // Process the results
    queries := []DistributedQuery{}
    for rows.Next() {
        var query DistributedQuery
        err := rows.Scan(&query.ID, &query.Query)
        if err != nil {
            return nil, err
        }
        queries = append(queries, query)
    }
    
    return queries, nil
}
```

### Result Collection

Implement result collection from devices:

```go
func SaveQueryResults(db *sql.DB, hostID int, results []QueryResult) error {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Process each result
    for _, result := range results {
        // Update the execution status
        _, err := tx.Exec(
            "INSERT INTO distributed_query_executions (campaign_id, host_id, status) VALUES (?, ?, ?)",
            result.CampaignID, hostID, "complete",
        )
        if err != nil {
            return err
        }
        
        // Save the result rows
        for _, row := range result.Rows {
            // Convert row to JSON
            rowJSON, err := json.Marshal(row)
            if err != nil {
                return err
            }
            
            // Save the row
            _, err = tx.Exec(
                "INSERT INTO distributed_query_results (execution_id, result) VALUES (LAST_INSERT_ID(), ?)",
                rowJSON,
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

### WebSocket Notifications

Implement WebSocket notifications for real-time updates:

```go
func SetupWebSocketServer(server *http.Server) {
    // Create a WebSocket upgrader
    upgrader := websocket.Upgrader{
        ReadBufferSize:  1024,
        WriteBufferSize: 1024,
        CheckOrigin: func(r *http.Request) bool {
            // Allow all origins in development
            return true
        },
    }
    
    // Set up the WebSocket handler
    http.HandleFunc("/api/v1/fleet/results/websocket", func(w http.ResponseWriter, r *http.Request) {
        // Upgrade the HTTP connection to a WebSocket connection
        conn, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
            log.Printf("Error upgrading to WebSocket: %v", err)
            return
        }
        defer conn.Close()
        
        // Get the campaign ID from the query parameters
        campaignID := r.URL.Query().Get("campaign_id")
        if campaignID == "" {
            log.Printf("Missing campaign_id parameter")
            return
        }
        
        // Register the connection for this campaign
        RegisterConnection(campaignID, conn)
        defer UnregisterConnection(campaignID, conn)
        
        // Keep the connection alive
        for {
            // Read a message (just to detect disconnection)
            _, _, err := conn.ReadMessage()
            if err != nil {
                log.Printf("Error reading WebSocket message: %v", err)
                break
            }
        }
    })
}

func NotifyResultsAvailable(campaignID string, results []QueryResult) {
    // Get the connections for this campaign
    connections := GetConnections(campaignID)
    
    // Send the results to each connection
    for _, conn := range connections {
        // Convert results to JSON
        resultsJSON, err := json.Marshal(results)
        if err != nil {
            log.Printf("Error marshaling results: %v", err)
            continue
        }
        
        // Send the results
        err = conn.WriteMessage(websocket.TextMessage, resultsJSON)
        if err != nil {
            log.Printf("Error sending results: %v", err)
            continue
        }
    }
}
```

## API Endpoints

Implement API endpoints for live queries:

### Create Live Query

```go
func CreateLiveQueryHandler(w http.ResponseWriter, r *http.Request) {
    // Parse the request body
    var req struct {
        Query   string   `json:"query"`
        Targets []Target `json:"targets"`
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
    
    // Get the user ID from the context
    userID := r.Context().Value("user_id").(int)
    
    // Create the query
    queryID, err := CreateQuery(db, req.Query)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error creating query: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Create the campaign
    campaignID, err := CreateLiveQueryCampaign(db, queryID, userID, req.Targets)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error creating campaign: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the campaign ID
    json.NewEncoder(w).Encode(map[string]int{"campaign_id": campaignID})
}
```

### Get Live Query Results

```go
func GetLiveQueryResultsHandler(w http.ResponseWriter, r *http.Request) {
    // Get the campaign ID from the URL
    vars := mux.Vars(r)
    campaignID := vars["campaign_id"]
    
    // Get the results
    results, err := GetQueryResults(db, campaignID)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting results: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the results
    json.NewEncoder(w).Encode(results)
}
```

## Testing

### Manual Testing

1. Create a live query through the API
2. Verify the query is distributed to devices
3. Check that results are returned and displayed
4. Test WebSocket notifications

### Automated Testing

Fleet includes automated tests for Live Queries functionality:

```bash
# Run Live Queries tests
go test -v ./server/service/live_query_test.go
```

## Debugging

### Query Distribution Issues

- **Target Selection**: Verify the query is targeting the correct devices
- **Device Check-in**: Ensure devices are checking in with the Fleet server
- **Query Validation**: Check if the query is valid and can be executed by osquery

### Result Collection Issues

- **Execution Status**: Verify the query execution status is updated correctly
- **Result Storage**: Ensure results are stored in the database
- **WebSocket Notifications**: Check if WebSocket notifications are sent

## Performance Considerations

Live Queries can impact system performance, especially for complex queries or large result sets:

- **Query Complexity**: Complex queries can consume significant CPU resources on devices
- **Result Size**: Large result sets can consume significant memory and network bandwidth
- **Device Count**: Executing queries across a large number of devices can impact server performance

## Related Resources

- [Live Queries Architecture](../../architecture/orchestration/live-queries.md)
- [Troubleshooting Live Queries](../troubleshooting-live-queries.md)
- [osquery Documentation](https://osquery.readthedocs.io/)