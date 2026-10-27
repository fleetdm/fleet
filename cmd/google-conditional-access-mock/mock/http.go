package mock

import (
	"bytes"
	"encoding/json"
	"html/template"
	"math/rand/v2"
	"net/http"
	"time"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError answers fleetdm.com routes, which use a flat error string.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// writeGoogleError answers in Google's error shape, so googleapi.CheckResponse returns a
// *googleapi.Error with the status code.
func writeGoogleError(w http.ResponseWriter, code int, status, msg string) {
	writeJSON(w, code, map[string]any{
		"error": map[string]any{"code": code, "message": msg, "status": status},
	})
}

func writeOAuthError(w http.ResponseWriter, code, description string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": code, "error_description": description})
}

func renderHTML(w http.ResponseWriter, status int, tmpl *template.Template, data any) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

// simulateLatencyAndErrors delays requests by 50% to 150% of latencyMean and fails a
// fraction of them with a 429 or 500.
func simulateLatencyAndErrors(latencyMean time.Duration, errorRate float64, next http.HandlerFunc) http.HandlerFunc {
	if latencyMean == 0 && errorRate == 0 {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if latencyMean > 0 {
			time.Sleep(time.Duration(float64(latencyMean) * (0.5 + rand.Float64()))) //nolint:gosec // simulated latency
		}
		if errorRate > 0 && rand.Float64() < errorRate { //nolint:gosec // simulated errors
			if rand.Float64() < 0.5 { //nolint:gosec // simulated errors
				writeGoogleError(w, http.StatusTooManyRequests, "RESOURCE_EXHAUSTED", "simulated rate limit")
			} else {
				writeGoogleError(w, http.StatusInternalServerError, "INTERNAL", "simulated server error")
			}
			return
		}
		next(w, r)
	}
}
