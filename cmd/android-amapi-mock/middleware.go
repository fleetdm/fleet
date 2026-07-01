package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

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
				log.Printf("Forwarding to Google AMAPI: %s %s", r.Method, r.URL.Path)
				switch r.Method {
				case "GET":
					google.ForwardDevicesGet(w, r)
				case "PATCH":
					google.ForwardDevicesPatch(w, r)
				case "DELETE":
					google.ForwardDevicesDelete(w, r)
				case "POST":
					google.ForwardIssueCommand(w, r)
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

// discardResponseWriter is an http.ResponseWriter that discards the response.
// Used for fire-and-forget forwarding where we don't need the response.
type discardResponseWriter struct{}

func (discardResponseWriter) Header() http.Header         { return http.Header{} }
func (discardResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (discardResponseWriter) WriteHeader(int)              {}
