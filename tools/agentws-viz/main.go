// Command agentws-viz is a small development dashboard that visualizes the
// agent WebSocket connections held by a running Fleet server, live.
//
// It polls the server's /debug/agentws endpoint (which requires a global admin
// API token) and serves a self-refreshing dashboard page on localhost, showing
// each connection's host ID, remote address, uptime, last notification and
// counters, plus a running event log of connects, disconnects and
// notifications.
//
// Usage:
//
//	go run ./tools/agentws-viz -server https://localhost:8080 -token $(fleetctl config get token) -insecure
//
// Then open http://localhost:3001.
package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

func main() {
	var (
		serverURL = flag.String("server", "https://localhost:8080", "Fleet server URL")
		token     = flag.String("token", os.Getenv("FLEET_API_TOKEN"), "Fleet API token of a global admin (defaults to FLEET_API_TOKEN)")
		addr      = flag.String("addr", "127.0.0.1:3001", "address for the dashboard to listen on")
		insecure  = flag.Bool("insecure", false, "skip TLS certificate verification when talking to the Fleet server")
	)
	flag.Parse()

	if *token == "" {
		log.Fatal("missing Fleet API token: pass -token or set FLEET_API_TOKEN (get one with `fleetctl config get token` after `fleetctl login`)")
	}

	client := fleethttp.NewClient(
		fleethttp.WithTimeout(10*time.Second),
		fleethttp.WithTLSClientConfig(&tls.Config{
			InsecureSkipVerify: *insecure, //nolint:gosec // local development tool, opt-in flag
		}),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(dashboardHTML))
	})
	// Proxy the snapshot so the page needs no token and no CORS/TLS handling.
	mux.HandleFunc("/api/snapshot", func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, *serverURL+"/debug/agentws", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		req.Header.Set("Authorization", "Bearer "+*token)
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, fmt.Sprintf("fleet server unreachable: %v", err), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	})

	log.Printf("agentws-viz: watching %s/debug/agentws", *serverURL)
	log.Printf("agentws-viz: dashboard on http://%s", *addr)
	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
