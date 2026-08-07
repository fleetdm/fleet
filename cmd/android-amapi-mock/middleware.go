package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"
)

func simulateLatencyAndErrors(latencyMean time.Duration, errorRate float64, next http.HandlerFunc) http.HandlerFunc {
	if latencyMean == 0 && errorRate == 0 {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// Add random latency (50%-150% of mean)
		if latencyMean > 0 {
			jitter := time.Duration(float64(latencyMean) * (0.5 + rand.Float64())) // #nosec G404 -- load testing
			time.Sleep(jitter)
		}

		// Occasionally return errors
		if errorRate > 0 && rand.Float64() < errorRate { // #nosec G404 -- load testing
			w.Header().Set("Content-Type", "application/json")
			if rand.Float64() < 0.5 { // #nosec G404 -- load testing
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{
						"code":    429,
						"message": "simulated rate limit",
						"status":  "RESOURCE_EXHAUSTED",
					},
				})
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{
						"code":    500,
						"message": "simulated server error",
						"status":  "INTERNAL",
					},
				})
			}
			return
		}

		next(w, r)
	}
}

// forwardForRealDevice returns middleware that checks if a device-specific request
// targets a registered fake device. If not, it forwards to Google via the authenticated client.
func forwardForRealDevice(store *deviceStore, google *googleForwarder) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			name := deviceName(r)
			if store.getByName(name) != nil {
				next(w, r)
				return
			}
			// Real device — forward via Google SDK if available
			if google != nil {
				hasSeenRealDevice.Store(true)
				log.Printf("Real device request: %s %s (device: %q)", r.Method, r.URL.Path, name) // #nosec G706 -- load testing tool
				switch r.Method {
				case "GET":
					google.ForwardDevicesGet(w, r)
				case "PATCH":
					google.ForwardDevicesPatch(w, r)
				case "DELETE":
					google.ForwardDevicesDelete(w, r)
				case "POST":
					google.ForwardIssueCommand(w, r)
				default:
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusMethodNotAllowed)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"error": map[string]any{
							"code":    405,
							"message": "unsupported method " + r.Method + " for device forwarding",
							"status":  "METHOD_NOT_ALLOWED",
						},
					})
				}
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error":{"code":404,"message":"Device not found","status":"NOT_FOUND"}}`)
		}
	}
}

// forwardOrMock forwards to Google if credentials are configured,
// otherwise falls back to the local mock handler.
func forwardOrMock(google *googleForwarder, fallback http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if google == nil {
			fallback(w, r)
			return
		}
		// Route to the appropriate Google forwarder method based on the path
		path := r.URL.Path
		switch {
		case r.Method == "POST" && strings.Contains(path, "/enrollmentTokens"):
			google.ForwardEnrollmentTokenCreate(w, r)
		case r.Method == "GET" && strings.Contains(path, "/applications/"):
			google.ForwardApplicationsGet(w, r)
		case r.Method == "POST" && strings.Contains(path, "/webApps"):
			google.ForwardWebAppsCreate(w, r)
		case r.Method == "GET" && (path == "/v1/enterprises" || strings.HasSuffix(path, "/enterprises")):
			google.ForwardEnterprisesList(w, r)
		default:
			fallback(w, r)
		}
	}
}
