package software_ingestion

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// MonitoringHandler provides HTTP endpoints for monitoring software ingestion frequency
type MonitoringHandler struct {
	tracker *IngestionTracker
	logger  log.Logger
}

func NewMonitoringHandler(tracker *IngestionTracker, logger log.Logger) *MonitoringHandler {
	return &MonitoringHandler{
		tracker: tracker,
		logger:  logger,
	}
}

func (h *MonitoringHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/v1/fleet/software_ingestion/tracking/summary":
		h.handleSummary(w, r)
	case "/api/v1/fleet/software_ingestion/tracking/stale_hosts":
		h.handleStaleHosts(w, r)
	case "/api/v1/fleet/software_ingestion/tracking/over_active_hosts":
		h.handleOverActiveHosts(w, r)
	case "/api/v1/fleet/software_ingestion/tracking/host_status":
		h.handleHostStatus(w, r)
	case "/api/v1/fleet/software_ingestion/tracking/alerts":
		h.handleAlerts(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleSummary returns an overview of ingestion health across all hosts
func (h *MonitoringHandler) handleSummary(w http.ResponseWriter, r *http.Request) {
	summary := h.tracker.GetIngestionSummary()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summary); err != nil {
		level.Error(h.logger).Log("msg", "failed to encode summary response", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleStaleHosts returns hosts that haven't ingested software recently
func (h *MonitoringHandler) handleStaleHosts(w http.ResponseWriter, r *http.Request) {
	staleHosts := h.tracker.GetStaleHosts()

	response := struct {
		StaleHosts []HostIngestionStatus `json:"stale_hosts"`
		Count      int                   `json:"count"`
		Timestamp  time.Time             `json:"timestamp"`
	}{
		StaleHosts: staleHosts,
		Count:      len(staleHosts),
		Timestamp:  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		level.Error(h.logger).Log("msg", "failed to encode stale hosts response", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleOverActiveHosts returns hosts that are ingesting too frequently
func (h *MonitoringHandler) handleOverActiveHosts(w http.ResponseWriter, r *http.Request) {
	overActiveHosts := h.tracker.GetOverActiveHosts()

	response := struct {
		OverActiveHosts []HostIngestionStatus `json:"over_active_hosts"`
		Count           int                   `json:"count"`
		Timestamp       time.Time             `json:"timestamp"`
	}{
		OverActiveHosts: overActiveHosts,
		Count:           len(overActiveHosts),
		Timestamp:       time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		level.Error(h.logger).Log("msg", "failed to encode over active hosts response", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleHostStatus returns ingestion status for a specific host
func (h *MonitoringHandler) handleHostStatus(w http.ResponseWriter, r *http.Request) {
	hostIDStr := r.URL.Query().Get("host_id")
	if hostIDStr == "" {
		http.Error(w, "host_id parameter required", http.StatusBadRequest)
		return
	}

	hostID, err := strconv.ParseUint(hostIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid host_id parameter", http.StatusBadRequest)
		return
	}

	status := h.tracker.GetHostStatus(uint(hostID))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		level.Error(h.logger).Log("msg", "failed to encode host status response", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleAlerts returns active alerts based on ingestion patterns
func (h *MonitoringHandler) handleAlerts(w http.ResponseWriter, r *http.Request) {
	summary := h.tracker.GetIngestionSummary()
	staleHosts := h.tracker.GetStaleHosts()
	overActiveHosts := h.tracker.GetOverActiveHosts()

	alerts := []IngestionAlert{}

	// Generate alerts based on thresholds
	if summary.TotalHosts > 0 {
		stalePercentage := float64(summary.StaleHosts) / float64(summary.TotalHosts) * 100

		// Critical: More than 25% of hosts are stale
		if stalePercentage > 25.0 {
			alerts = append(alerts, IngestionAlert{
				Level:       "critical",
				Type:        "high_stale_percentage",
				Message:     "High percentage of hosts have stale software ingestion",
				Value:       stalePercentage,
				Threshold:   25.0,
				AffectedHosts: len(staleHosts),
				Timestamp:   time.Now(),
			})
		} else if stalePercentage > 10.0 {
			// Warning: More than 10% of hosts are stale
			alerts = append(alerts, IngestionAlert{
				Level:       "warning",
				Type:        "moderate_stale_percentage",
				Message:     "Moderate percentage of hosts have stale software ingestion",
				Value:       stalePercentage,
				Threshold:   10.0,
				AffectedHosts: len(staleHosts),
				Timestamp:   time.Now(),
			})
		}

		// Warning: Hosts ingesting too frequently
		if len(overActiveHosts) > 0 {
			overActivePercentage := float64(len(overActiveHosts)) / float64(summary.TotalHosts) * 100
			alerts = append(alerts, IngestionAlert{
				Level:       "warning",
				Type:        "over_active_hosts",
				Message:     "Hosts are ingesting software too frequently",
				Value:       overActivePercentage,
				Threshold:   5.0, // Alert if more than 5% are over-active
				AffectedHosts: len(overActiveHosts),
				Timestamp:   time.Now(),
			})
		}

		// Info: Low overall activity
		if summary.ActiveHosts < summary.TotalHosts/2 {
			activePercentage := float64(summary.ActiveHosts) / float64(summary.TotalHosts) * 100
			alerts = append(alerts, IngestionAlert{
				Level:       "info",
				Type:        "low_activity",
				Message:     "Low overall software ingestion activity",
				Value:       activePercentage,
				Threshold:   50.0,
				AffectedHosts: summary.TotalHosts - summary.ActiveHosts,
				Timestamp:   time.Now(),
			})
		}
	}

	response := struct {
		Alerts    []IngestionAlert `json:"alerts"`
		Count     int              `json:"count"`
		Summary   IngestionSummary `json:"summary"`
		Timestamp time.Time        `json:"timestamp"`
	}{
		Alerts:    alerts,
		Count:     len(alerts),
		Summary:   summary,
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		level.Error(h.logger).Log("msg", "failed to encode alerts response", "err", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

type IngestionAlert struct {
	Level         string    `json:"level"`          // "critical", "warning", "info"
	Type          string    `json:"type"`           // Alert type identifier
	Message       string    `json:"message"`        // Human-readable message
	Value         float64   `json:"value"`          // Current value
	Threshold     float64   `json:"threshold"`      // Threshold that triggered alert
	AffectedHosts int       `json:"affected_hosts"` // Number of hosts affected
	Timestamp     time.Time `json:"timestamp"`      // When alert was generated
}

// GetTracker returns the ingestion tracker for use by other parts of the service
func (s *service) GetTracker() *IngestionTracker {
	return s.tracker
}

// StartTrackingReports starts periodic reporting of ingestion metrics
func (s *service) StartTrackingReports(ctx context.Context) {
	go s.tracker.StartPeriodicReporting(ctx, 15*time.Minute) // Report every 15 minutes
}