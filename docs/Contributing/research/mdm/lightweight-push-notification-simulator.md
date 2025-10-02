# Lightweight push notification simulator (SSE-based)

## Overview

This document outlines the design of a lightweight push notification simulator built in Go, capable of broadcasting messages from a central server to up to 300,000 simulated clients using Server-Sent Events (SSE). It is optimized for one-way communication for load testing and infrastructure validation. This is the design for story [30816](https://github.com/fleetdm/fleet/issues/30816).

**Note:** This simulator models the APNs push behavior used by Apple MDM for both enterprise and BYOD deployments. While enrollment flows and management scopes differ, the push mechanism is a token-based, one-way wake-up notification that behaves the same in both cases.

## Goals

* Efficiently simulate 300K+ client connections
* Support one-way push messages from server to clients
* Allow optional features like message delay and targeted delivery to individual clients
* Keep memory and CPU usage minimal
* Simple to deploy and scale

---

## High-Level architecture

```
+----------------+
|  Fleet Server  |
+----------------+
        |
        v
+--------------------------------------------+
|             SSE Push Server                |
|               (Go, HTTP/1.1)               |
|                                            |
| - Accepts pushes                           |
| - Routes messages to individual clients    |
| - Supports delay, etc.                     |
+--------------------------------------------+
        |
   +----+-----+-----+
   |    |     |     |
   v    v     v     v
Client Client ... Client
         (Goroutines)
```

* Clients establish a long-lived SSE connection to the server
* The server sends events using `text/event-stream` format
* Fleet server can inject messages into the server via HTTP API

---

## Technologies

* Language: Go
* Protocol: HTTP/1.1 (SSE)
* Deployment: Single binary or Docker containers
* Optional: Redis for message queueing/fan-out (should not be needed if we only use 1 server)

---

## System tuning

### Resource requirements

The SSE server must efficiently handle up to 300,000 concurrent persistent HTTP connections. Resource planning should take into account memory and CPU usage under I/O-bound workloads:

* **CPU**: Multi-core system (≥8 cores recommended). Since connections are mostly I/O-bound, CPU usage is typically moderate, but spikes may occur during fan-out or burst pushes.
* **Memory**: Estimate \~4 KB per client connection (goroutine + buffers).

    * For 300K clients: 300,000 × 4 KB = \~1.2 GB RAM minimum.
    * Add overhead for routing structures, message buffers, and OS-level memory use.
    * Recommended: ≥4 GB RAM.

Additional system tuning helps optimize concurrency handling and socket limits.

### OS-level

```bash
# Increase maximum number of open file descriptors (required for many concurrent connections)
ulimit -n 1000000

# Max number of connections that can be queued for acceptance (note: only affects backlog for pending accepts, not the number of active concurrent connections)
sudo sysctl -w net.core.somaxconn=65535
```

---

## Fleet server changes

The Fleet server needs to add a development env var that switches the base https://api.push.apple.com URL to the URL of our SSE server. [Code that creates the URL](https://github.com/fleetdm/fleet/blob/e4df954b0f548d9d945fa56303aae7183d8b5d52/server/mdm/nanomdm/push/nanopush/provider.go#L73)

---

## SSE server (simplified)

```go
// Do NOT set ReadTimeout or IdleTimeout for SSE endpoints

var clients = make(map[string]chan string)
var mu sync.Mutex

func sendToClient(token string, message string) {
    mu.Lock()
    defer mu.Unlock()
    if ch, ok := clients[token]; ok {
        ch <- message
    }
}

func sseHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    // Extract device token
    deviceToken := r.URL.Query().Get("token")
    if deviceToken == "" {
        http.Error(w, "Missing device token", http.StatusBadRequest)
        return
    }

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
        return
    }

    // Register client
    clientChan := make(chan string)

    mu.Lock()
    clients[deviceToken] = clientChan
    mu.Unlock()

    defer func() {
        mu.Lock()
        delete(clients, deviceToken)
        mu.Unlock()
    }()

    for msg := range clientChan {
        fmt.Fprintf(w, "data: %s\n\n", msg)
        flusher.Flush()
    }
	
	// Optional: Send periodic keepalive messages
	// This prevents timeouts on intermediate proxies (e.g., load balancers)
}
```

---

## Client simulator (osquery-perf)

**Note:** In real MDM implementations, the APNs device token is sent from the device to the MDM server during enrollment via a `TokenUpdate` message. In this simulation, the client must retrieve and remember its device token (e.g., from `/checkin` as modeled in [Fleet's `mdmtest/apple.go`](https://github.com/fleetdm/fleet/blob/e4726d4410c8492dbb752bcc9443c133a726bb0a/pkg/mdm/mdmtest/apple.go#L745)). The client must include this device token when registering with the SSE server so the server knows which messages to deliver.

```go
// Ensure client doesn’t impose timeouts
// Don't use http.Client{Timeout: ...} for SSE

func runClient(deviceToken string) {
    url := fmt.Sprintf("http://localhost:8080/events?token=%s", deviceToken)
    resp, _ := http.Get(url)
    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
        fmt.Printf("Client %s got: %s
", deviceToken, scanner.Text())
    }
    resp.Body.Close()
}
```

In addition, the [`mdmCheckInTicker` should be replaced](https://github.com/fleetdm/fleet/blob/e4726d4410c8492dbb752bcc9443c133a726bb0a/cmd/osquery-perf/agent.go#L931) and only be triggered by the push notifications.

Run thousands of these in goroutines
