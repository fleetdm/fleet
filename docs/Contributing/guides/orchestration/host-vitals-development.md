# Host Vitals Development Guide

This guide provides instructions for developing Host Vitals functionality in Fleet.

## Introduction

Host Vitals in Fleet provide real-time and historical information about the health and status of devices, including CPU usage, memory usage, disk usage, and uptime. This guide covers the development and implementation of Host Vitals features.

## Prerequisites

Before you begin developing Host Vitals functionality, you should have:

- A development environment set up according to the [Building Fleet](../../getting-started/building-fleet.md) guide
- Basic understanding of system metrics and osquery
- Familiarity with Fleet's architecture
- Understanding of time-series data processing

## Host Vitals Architecture

Host Vitals in Fleet follow a specific flow:

1. Devices collect system metrics using osquery
2. Devices send the metrics to the Fleet server
3. Fleet server processes and stores the metrics
4. Fleet server provides API endpoints for retrieving metrics
5. Fleet UI displays the metrics in charts and tables

## Implementation

### Database Schema

Host Vitals information is stored in the Fleet database:

```sql
CREATE TABLE host_metrics (
  id INT AUTO_INCREMENT PRIMARY KEY,
  host_id INT NOT NULL,
  timestamp TIMESTAMP NOT NULL,
  cpu_usage FLOAT,
  memory_usage FLOAT,
  memory_total BIGINT,
  disk_usage FLOAT,
  disk_total BIGINT,
  uptime BIGINT,
  load_average_1m FLOAT,
  load_average_5m FLOAT,
  load_average_15m FLOAT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (host_id) REFERENCES hosts(id),
  INDEX (host_id, timestamp)
);

CREATE TABLE host_network_metrics (
  id INT AUTO_INCREMENT PRIMARY KEY,
  host_id INT NOT NULL,
  timestamp TIMESTAMP NOT NULL,
  interface VARCHAR(255) NOT NULL,
  rx_bytes BIGINT,
  tx_bytes BIGINT,
  rx_packets BIGINT,
  tx_packets BIGINT,
  rx_errors BIGINT,
  tx_errors BIGINT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (host_id) REFERENCES hosts(id),
  INDEX (host_id, timestamp, interface)
);
```

### Metrics Collection

Implement metrics collection using osquery:

```go
func CollectHostMetrics() (map[string]interface{}, error) {
    metrics := map[string]interface{}{}
    
    // Collect CPU usage
    cpuQuery := "SELECT user, system, idle, nice, iowait FROM cpu_time"
    cpuRows, err := osquery.Query(cpuQuery)
    if err != nil {
        return nil, err
    }
    
    if len(cpuRows) > 0 {
        user := cpuRows[0]["user"].(float64)
        system := cpuRows[0]["system"].(float64)
        idle := cpuRows[0]["idle"].(float64)
        nice := cpuRows[0]["nice"].(float64)
        iowait := cpuRows[0]["iowait"].(float64)
        
        total := user + system + idle + nice + iowait
        cpuUsage := (total - idle) / total * 100
        metrics["cpu_usage"] = cpuUsage
    }
    
    // Collect memory usage
    memoryQuery := "SELECT memory_total, memory_free, buffers, cached FROM memory_info"
    memoryRows, err := osquery.Query(memoryQuery)
    if err != nil {
        return nil, err
    }
    
    if len(memoryRows) > 0 {
        memoryTotal := memoryRows[0]["memory_total"].(int64)
        memoryFree := memoryRows[0]["memory_free"].(int64)
        buffers := memoryRows[0]["buffers"].(int64)
        cached := memoryRows[0]["cached"].(int64)
        
        memoryUsed := memoryTotal - memoryFree - buffers - cached
        memoryUsage := float64(memoryUsed) / float64(memoryTotal) * 100
        metrics["memory_usage"] = memoryUsage
        metrics["memory_total"] = memoryTotal
    }
    
    // Collect disk usage
    diskQuery := "SELECT device, blocks_size, blocks, blocks_free FROM mounts WHERE path = '/'"
    diskRows, err := osquery.Query(diskQuery)
    if err != nil {
        return nil, err
    }
    
    if len(diskRows) > 0 {
        blocksSize := diskRows[0]["blocks_size"].(int64)
        blocks := diskRows[0]["blocks"].(int64)
        blocksFree := diskRows[0]["blocks_free"].(int64)
        
        diskTotal := blocks * blocksSize
        diskUsed := (blocks - blocksFree) * blocksSize
        diskUsage := float64(diskUsed) / float64(diskTotal) * 100
        metrics["disk_usage"] = diskUsage
        metrics["disk_total"] = diskTotal
    }
    
    // Collect uptime
    uptimeQuery := "SELECT total_seconds FROM uptime"
    uptimeRows, err := osquery.Query(uptimeQuery)
    if err != nil {
        return nil, err
    }
    
    if len(uptimeRows) > 0 {
        uptime := uptimeRows[0]["total_seconds"].(int64)
        metrics["uptime"] = uptime
    }
    
    // Collect load average
    loadQuery := "SELECT '1m' AS period, average FROM load_average WHERE period = '1m'"
    loadRows, err := osquery.Query(loadQuery)
    if err != nil {
        return nil, err
    }
    
    if len(loadRows) > 0 {
        loadAverage1m := loadRows[0]["average"].(float64)
        metrics["load_average_1m"] = loadAverage1m
    }
    
    loadQuery = "SELECT '5m' AS period, average FROM load_average WHERE period = '5m'"
    loadRows, err = osquery.Query(loadQuery)
    if err != nil {
        return nil, err
    }
    
    if len(loadRows) > 0 {
        loadAverage5m := loadRows[0]["average"].(float64)
        metrics["load_average_5m"] = loadAverage5m
    }
    
    loadQuery = "SELECT '15m' AS period, average FROM load_average WHERE period = '15m'"
    loadRows, err = osquery.Query(loadQuery)
    if err != nil {
        return nil, err
    }
    
    if len(loadRows) > 0 {
        loadAverage15m := loadRows[0]["average"].(float64)
        metrics["load_average_15m"] = loadAverage15m
    }
    
    // Collect network metrics
    networkQuery := "SELECT interface, rx_bytes, tx_bytes, rx_packets, tx_packets, rx_errors, tx_errors FROM interface_details"
    networkRows, err := osquery.Query(networkQuery)
    if err != nil {
        return nil, err
    }
    
    networkMetrics := []map[string]interface{}{}
    for _, row := range networkRows {
        networkMetric := map[string]interface{}{
            "interface": row["interface"].(string),
            "rx_bytes": row["rx_bytes"].(int64),
            "tx_bytes": row["tx_bytes"].(int64),
            "rx_packets": row["rx_packets"].(int64),
            "tx_packets": row["tx_packets"].(int64),
            "rx_errors": row["rx_errors"].(int64),
            "tx_errors": row["tx_errors"].(int64),
        }
        networkMetrics = append(networkMetrics, networkMetric)
    }
    metrics["network"] = networkMetrics
    
    return metrics, nil
}
```

### Metrics Submission

Implement metrics submission to the Fleet server:

```go
func SubmitHostMetrics(client *http.Client, serverURL string, nodeKey string, metrics map[string]interface{}) error {
    // Create the request body
    body := map[string]interface{}{
        "node_key": nodeKey,
        "metrics": metrics,
    }
    
    // Convert to JSON
    bodyJSON, err := json.Marshal(body)
    if err != nil {
        return err
    }
    
    // Create the request
    req, err := http.NewRequest("POST", serverURL+"/api/v1/fleet/metrics", bytes.NewBuffer(bodyJSON))
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

### Metrics Processing

Implement metrics processing on the Fleet server:

```go
func ProcessHostMetrics(db *sql.DB, hostID int, metrics map[string]interface{}) error {
    // Start a transaction
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Insert the host metrics
    _, err = tx.Exec(
        `INSERT INTO host_metrics (
            host_id, timestamp, cpu_usage, memory_usage, memory_total,
            disk_usage, disk_total, uptime, load_average_1m, load_average_5m, load_average_15m
        ) VALUES (?, NOW(), ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
        hostID,
        metrics["cpu_usage"],
        metrics["memory_usage"],
        metrics["memory_total"],
        metrics["disk_usage"],
        metrics["disk_total"],
        metrics["uptime"],
        metrics["load_average_1m"],
        metrics["load_average_5m"],
        metrics["load_average_15m"],
    )
    if err != nil {
        return err
    }
    
    // Process network metrics
    networkMetrics, ok := metrics["network"].([]map[string]interface{})
    if ok {
        for _, networkMetric := range networkMetrics {
            _, err = tx.Exec(
                `INSERT INTO host_network_metrics (
                    host_id, timestamp, interface, rx_bytes, tx_bytes,
                    rx_packets, tx_packets, rx_errors, tx_errors
                ) VALUES (?, NOW(), ?, ?, ?, ?, ?, ?, ?)`,
                hostID,
                networkMetric["interface"],
                networkMetric["rx_bytes"],
                networkMetric["tx_bytes"],
                networkMetric["rx_packets"],
                networkMetric["tx_packets"],
                networkMetric["rx_errors"],
                networkMetric["tx_errors"],
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

### Metrics Retrieval

Implement metrics retrieval from the Fleet server:

```go
func GetHostMetrics(db *sql.DB, hostID int, start, end time.Time) ([]map[string]interface{}, error) {
    // Query the database
    rows, err := db.Query(
        `SELECT
            timestamp, cpu_usage, memory_usage, memory_total,
            disk_usage, disk_total, uptime, load_average_1m, load_average_5m, load_average_15m
        FROM host_metrics
        WHERE host_id = ? AND timestamp BETWEEN ? AND ?
        ORDER BY timestamp`,
        hostID, start, end,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    // Process the results
    metrics := []map[string]interface{}{}
    for rows.Next() {
        var timestamp time.Time
        var cpuUsage, memoryUsage, diskUsage, loadAverage1m, loadAverage5m, loadAverage15m sql.NullFloat64
        var memoryTotal, diskTotal, uptime sql.NullInt64
        
        err := rows.Scan(
            &timestamp,
            &cpuUsage,
            &memoryUsage,
            &memoryTotal,
            &diskUsage,
            &diskTotal,
            &uptime,
            &loadAverage1m,
            &loadAverage5m,
            &loadAverage15m,
        )
        if err != nil {
            return nil, err
        }
        
        metric := map[string]interface{}{
            "timestamp": timestamp.Unix(),
        }
        
        if cpuUsage.Valid {
            metric["cpu_usage"] = cpuUsage.Float64
        }
        
        if memoryUsage.Valid {
            metric["memory_usage"] = memoryUsage.Float64
        }
        
        if memoryTotal.Valid {
            metric["memory_total"] = memoryTotal.Int64
        }
        
        if diskUsage.Valid {
            metric["disk_usage"] = diskUsage.Float64
        }
        
        if diskTotal.Valid {
            metric["disk_total"] = diskTotal.Int64
        }
        
        if uptime.Valid {
            metric["uptime"] = uptime.Int64
        }
        
        if loadAverage1m.Valid {
            metric["load_average_1m"] = loadAverage1m.Float64
        }
        
        if loadAverage5m.Valid {
            metric["load_average_5m"] = loadAverage5m.Float64
        }
        
        if loadAverage15m.Valid {
            metric["load_average_15m"] = loadAverage15m.Float64
        }
        
        metrics = append(metrics, metric)
    }
    
    return metrics, nil
}
```

## API Endpoints

Implement API endpoints for host vitals:

### Submit Host Metrics

```go
func SubmitHostMetricsHandler(w http.ResponseWriter, r *http.Request) {
    // Parse the request body
    var req struct {
        NodeKey string                 `json:"node_key"`
        Metrics map[string]interface{} `json:"metrics"`
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
    
    // Process the metrics
    err = ProcessHostMetrics(db, hostID, req.Metrics)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error processing metrics: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return success
    w.WriteHeader(http.StatusOK)
}
```

### Get Host Metrics

```go
func GetHostMetricsHandler(w http.ResponseWriter, r *http.Request) {
    // Get the host ID from the URL
    vars := mux.Vars(r)
    hostID, err := strconv.Atoi(vars["host_id"])
    if err != nil {
        http.Error(w, "Invalid host ID", http.StatusBadRequest)
        return
    }
    
    // Get the time range from the query parameters
    startStr := r.URL.Query().Get("start")
    endStr := r.URL.Query().Get("end")
    
    var start, end time.Time
    if startStr != "" {
        startUnix, err := strconv.ParseInt(startStr, 10, 64)
        if err != nil {
            http.Error(w, "Invalid start time", http.StatusBadRequest)
            return
        }
        start = time.Unix(startUnix, 0)
    } else {
        start = time.Now().Add(-24 * time.Hour)
    }
    
    if endStr != "" {
        endUnix, err := strconv.ParseInt(endStr, 10, 64)
        if err != nil {
            http.Error(w, "Invalid end time", http.StatusBadRequest)
            return
        }
        end = time.Unix(endUnix, 0)
    } else {
        end = time.Now()
    }
    
    // Get the metrics
    metrics, err := GetHostMetrics(db, hostID, start, end)
    if err != nil {
        http.Error(w, fmt.Sprintf("Error getting metrics: %v", err), http.StatusInternalServerError)
        return
    }
    
    // Return the metrics
    json.NewEncoder(w).Encode(metrics)
}
```

## Metrics Visualization

Implement metrics visualization in the Fleet UI:

```javascript
function renderCPUChart(metrics) {
    const data = metrics.map(metric => ({
        x: new Date(metric.timestamp * 1000),
        y: metric.cpu_usage
    }));
    
    const chart = new Chart(document.getElementById('cpu-chart'), {
        type: 'line',
        data: {
            datasets: [{
                label: 'CPU Usage (%)',
                data: data,
                borderColor: '#007bff',
                backgroundColor: 'rgba(0, 123, 255, 0.1)',
                fill: true
            }]
        },
        options: {
            scales: {
                x: {
                    type: 'time',
                    time: {
                        unit: 'hour'
                    }
                },
                y: {
                    min: 0,
                    max: 100,
                    title: {
                        display: true,
                        text: 'CPU Usage (%)'
                    }
                }
            }
        }
    });
}
```

## Testing

### Manual Testing

1. Implement metrics collection on a test device
2. Submit metrics to the Fleet server
3. Verify metrics are stored in the database
4. Retrieve and visualize metrics in the UI

### Automated Testing

Fleet includes automated tests for Host Vitals functionality:

```bash
# Run Host Vitals tests
go test -v ./server/service/host_vitals_test.go
```

## Debugging

### Metrics Collection Issues

- **osquery Queries**: Verify the osquery queries are correctly retrieving system metrics
- **Data Types**: Ensure the data types of the metrics are correct
- **Error Handling**: Check if errors during metrics collection are properly handled

### Metrics Processing Issues

- **Database Schema**: Verify the database schema is correctly defined
- **Data Insertion**: Ensure metrics are correctly inserted into the database
- **Transaction Management**: Check if database transactions are properly managed

## Performance Considerations

Host Vitals can generate a significant amount of data, especially for large fleets:

- **Collection Frequency**: More frequent collection generates more data
- **Metrics Count**: Collecting more metrics generates more data
- **Data Retention**: Consider implementing data retention policies
- **Data Aggregation**: Consider implementing data aggregation for historical metrics

## Related Resources

- [Host Vitals Architecture](../../architecture/orchestration/host-vitals.md)
- [Understanding Host Vitals](../../product-groups/orchestration/understanding-host-vitals.md)
- [osquery Documentation](https://osquery.readthedocs.io/)