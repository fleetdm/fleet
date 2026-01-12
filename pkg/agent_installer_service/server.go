// Package agentinstallerservice provides an HTTP server for generating fleetd installers.
package agentinstallerservice

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// GenerateRequest represents the request body for POST /generate.
type GenerateRequest struct {
	Type         string `json:"type"`
	FleetDesktop bool   `json:"fleet_desktop"`
	Arch         string `json:"arch"`
	EnrollSecret string `json:"enroll_secret"`
	URL          string `json:"url"`
}

// GenerateResponse represents the response body for POST /generate.
type GenerateResponse struct {
	Token string `json:"token"`
}

// StatusResponse represents the response body for GET /status/{token}.
type StatusResponse struct {
	Status          string `json:"status"`
	Detail          string `json:"detail,omitempty"`
	OrbitVersion    string `json:"orbit_version,omitempty"`
	DesktopVersion  string `json:"desktop_version,omitempty"`
	OsquerydVersion string `json:"osqueryd_version,omitempty"`
}

// Server is the HTTP server for the agent installer service.
type Server struct {
	mux *http.ServeMux
}

// NewServer creates a new agent installer service server.
func NewServer() *Server {
	s := &Server{
		mux: http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// ServeHTTP implements the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("POST /generate", s.handleGenerate)
	s.mux.HandleFunc("GET /status/{token}", s.handleStatus)
	s.mux.HandleFunc("GET /download/{token}", s.handleDownload)
}

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Type == "" {
		http.Error(w, "type is required", http.StatusBadRequest)
		return
	}
	validTypes := []string{"pkg", "msi", "deb", "rpm", "pkg.tar.zst"}
	if !contains(validTypes, req.Type) {
		http.Error(w, "invalid type", http.StatusBadRequest)
		return
	}
	// For now, only MSI is supported.
	if strings.ToLower(req.Type) != "msi" {
		http.Error(w, "only msi type is currently supported", http.StatusBadRequest)
		return
	}
	if req.EnrollSecret == "" {
		http.Error(w, "enroll_secret is required", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}
	// Validate URL format
	if u, err := url.ParseRequestURI(req.URL); err != nil {
		http.Error(w, "invalid url format", http.StatusBadRequest)
		return
	} else if u.Scheme != "https" && u.Scheme != "http" {
		http.Error(w, "url must start with http or https", http.StatusBadRequest)
		return
	}

	// Return mock response
	resp := GenerateResponse{
		Token: "mock-token-12345",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		http.NotFound(w, r)
		return
	}

	// Return mock response
	resp := StatusResponse{
		Status:          "completed",
		OrbitVersion:    "1.0.0",
		DesktopVersion:  "1.0.0",
		OsquerydVersion: "5.0.0",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		http.NotFound(w, r)
		return
	}

	// Return mock response (empty body for now)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=installer")
	w.WriteHeader(http.StatusOK)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
