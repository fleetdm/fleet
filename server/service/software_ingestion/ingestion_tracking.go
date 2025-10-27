package software_ingestion

import (
	"context"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// IngestionTracker monitors software ingestion frequency per host to ensure
// each host gets its software updated approximately once per hour
type IngestionTracker struct {
	mu                    sync.RWMutex
	hostLastIngestion     map[uint]time.Time     // hostID -> last ingestion time
	hostIngestionCount    map[uint]int64         // hostID -> total ingestion count
	hostIngestionHistory  map[uint][]time.Time   // hostID -> recent ingestion times (for rate calculation)
	expectedInterval      time.Duration          // Expected ingestion interval (1 hour)
	alertThreshold        time.Duration          // Alert if no ingestion for this long (1.5 hours)
	logger                log.Logger

	// Metrics for monitoring
	staleHosts            int64                  // Hosts that haven't ingested recently
	overActiveHosts       int64                  // Hosts ingesting too frequently
	totalActiveHosts      int64                  // Total hosts with recent activity
}

// HostIngestionStatus represents the ingestion status for a single host
type HostIngestionStatus struct {
	HostID              uint          `json:"host_id"`
	LastIngestion       time.Time     `json:"last_ingestion"`
	TimeSinceLastUpdate time.Duration `json:"time_since_last_update"`
	TotalIngestions     int64         `json:"total_ingestions"`
	IngestionRate       float64       `json:"ingestion_rate_per_hour"`
	Status              IngestionHealthStatus `json:"status"`
	LastSoftwareCount   int           `json:"last_software_count"`
}

type IngestionHealthStatus string

const (
	IngestionHealthy     IngestionHealthStatus = "healthy"      // Regular ingestion within expected window
	IngestionStale       IngestionHealthStatus = "stale"        // No ingestion for > 1.5 hours
	IngestionOverActive  IngestionHealthStatus = "over_active"  // Ingesting too frequently (> 2x/hour)
	IngestionFirstTime   IngestionHealthStatus = "first_time"   // First ingestion recorded
)

// NewIngestionTracker creates a tracker for monitoring software ingestion frequency
func NewIngestionTracker(logger log.Logger) *IngestionTracker {
	return &IngestionTracker{
		hostLastIngestion:    make(map[uint]time.Time),
		hostIngestionCount:   make(map[uint]int64),
		hostIngestionHistory: make(map[uint][]time.Time),
		expectedInterval:     1 * time.Hour,
		alertThreshold:       90 * time.Minute, // 1.5 hours
		logger:               logger,
	}
}

// RecordIngestion records a software ingestion event for a host
func (it *IngestionTracker) RecordIngestion(hostID uint, softwareCount int, ingestionType string) {
	it.mu.Lock()
	defer it.mu.Unlock()

	now := time.Now()

	// Record basic tracking info
	lastIngestion, existed := it.hostLastIngestion[hostID]
	it.hostLastIngestion[hostID] = now
	it.hostIngestionCount[hostID]++

	// Maintain ingestion history (keep last 24 hours for rate calculation)
	history := it.hostIngestionHistory[hostID]
	cutoff := now.Add(-24 * time.Hour)

	// Remove old entries
	var recentHistory []time.Time
	for _, t := range history {
		if t.After(cutoff) {
			recentHistory = append(recentHistory, t)
		}
	}
	recentHistory = append(recentHistory, now)
	it.hostIngestionHistory[hostID] = recentHistory

	// Log the ingestion event
	level.Debug(it.logger).Log(
		"msg", "software ingestion recorded",
		"host_id", hostID,
		"software_count", softwareCount,
		"type", ingestionType,
		"time_since_last", func() string {
			if existed {
				return time.Since(lastIngestion).String()
			}
			return "first_time"
		}(),
	)

	// Check for concerning patterns
	if existed {
		timeSinceLast := now.Sub(lastIngestion)

		// Alert if ingesting too frequently (multiple times in short period)
		if timeSinceLast < 30*time.Minute && len(recentHistory) > 1 {
			level.Warn(it.logger).Log(
				"msg", "host ingesting software very frequently",
				"host_id", hostID,
				"time_since_last", timeSinceLast,
				"recent_ingestions", len(recentHistory),
			)
		}
	}
}

// GetHostStatus returns the ingestion status for a specific host
func (it *IngestionTracker) GetHostStatus(hostID uint) HostIngestionStatus {
	it.mu.RLock()
	defer it.mu.RUnlock()

	lastIngestion, exists := it.hostLastIngestion[hostID]
	if !exists {
		return HostIngestionStatus{
			HostID: hostID,
			Status: IngestionFirstTime,
		}
	}

	timeSinceLast := time.Since(lastIngestion)
	totalIngestions := it.hostIngestionCount[hostID]

	// Calculate ingestion rate (ingestions per hour over last 24 hours)
	history := it.hostIngestionHistory[hostID]
	ingestionRate := float64(len(history)) // This is for last 24 hours, so divide by 24 to get per hour
	if len(history) > 0 {
		ingestionRate = float64(len(history)) / 24.0
	}

	// Determine status
	var status IngestionHealthStatus
	switch {
	case timeSinceLast > it.alertThreshold:
		status = IngestionStale
	case ingestionRate > 2.0: // More than 2 ingestions per hour
		status = IngestionOverActive
	default:
		status = IngestionHealthy
	}

	return HostIngestionStatus{
		HostID:              hostID,
		LastIngestion:       lastIngestion,
		TimeSinceLastUpdate: timeSinceLast,
		TotalIngestions:     totalIngestions,
		IngestionRate:       ingestionRate,
		Status:              status,
	}
}

// GetStaleHosts returns hosts that haven't ingested software recently
func (it *IngestionTracker) GetStaleHosts() []HostIngestionStatus {
	it.mu.RLock()
	defer it.mu.RUnlock()

	var staleHosts []HostIngestionStatus
	now := time.Now()

	for hostID, lastIngestion := range it.hostLastIngestion {
		if now.Sub(lastIngestion) > it.alertThreshold {
			staleHosts = append(staleHosts, it.getHostStatusUnsafe(hostID))
		}
	}

	return staleHosts
}

// GetOverActiveHosts returns hosts that are ingesting too frequently
func (it *IngestionTracker) GetOverActiveHosts() []HostIngestionStatus {
	it.mu.RLock()
	defer it.mu.RUnlock()

	var overActiveHosts []HostIngestionStatus

	for hostID, history := range it.hostIngestionHistory {
		// Check if host has ingested more than 2 times in the last hour
		oneHourAgo := time.Now().Add(-1 * time.Hour)
		recentCount := 0
		for _, t := range history {
			if t.After(oneHourAgo) {
				recentCount++
			}
		}

		if recentCount > 2 {
			overActiveHosts = append(overActiveHosts, it.getHostStatusUnsafe(hostID))
		}
	}

	return overActiveHosts
}

// getHostStatusUnsafe is an internal helper that doesn't acquire locks
func (it *IngestionTracker) getHostStatusUnsafe(hostID uint) HostIngestionStatus {
	lastIngestion := it.hostLastIngestion[hostID]
	timeSinceLast := time.Since(lastIngestion)
	totalIngestions := it.hostIngestionCount[hostID]

	history := it.hostIngestionHistory[hostID]
	ingestionRate := float64(len(history)) / 24.0

	var status IngestionHealthStatus
	switch {
	case timeSinceLast > it.alertThreshold:
		status = IngestionStale
	case ingestionRate > 2.0:
		status = IngestionOverActive
	default:
		status = IngestionHealthy
	}

	return HostIngestionStatus{
		HostID:              hostID,
		LastIngestion:       lastIngestion,
		TimeSinceLastUpdate: timeSinceLast,
		TotalIngestions:     totalIngestions,
		IngestionRate:       ingestionRate,
		Status:              status,
	}
}

// GetIngestionSummary returns an overview of ingestion health across all hosts
func (it *IngestionTracker) GetIngestionSummary() IngestionSummary {
	it.mu.RLock()
	defer it.mu.RUnlock()

	summary := IngestionSummary{
		TotalHosts: len(it.hostLastIngestion),
		Timestamp:  time.Now(),
	}

	now := time.Now()

	for hostID := range it.hostLastIngestion {
		status := it.getHostStatusUnsafe(hostID)

		switch status.Status {
		case IngestionHealthy:
			summary.HealthyHosts++
		case IngestionStale:
			summary.StaleHosts++
		case IngestionOverActive:
			summary.OverActiveHosts++
		}

		// Count hosts that have ingested in the last hour (active)
		if status.TimeSinceLastUpdate < 1*time.Hour {
			summary.ActiveHosts++
		}

		// Track ingestion rates
		if status.IngestionRate > summary.MaxIngestionRate {
			summary.MaxIngestionRate = status.IngestionRate
		}

		summary.AverageIngestionRate += status.IngestionRate
	}

	if summary.TotalHosts > 0 {
		summary.AverageIngestionRate /= float64(summary.TotalHosts)
		summary.HealthPercentage = float64(summary.HealthyHosts) / float64(summary.TotalHosts) * 100
	}

	return summary
}

type IngestionSummary struct {
	TotalHosts            int       `json:"total_hosts"`
	HealthyHosts          int       `json:"healthy_hosts"`
	StaleHosts            int       `json:"stale_hosts"`
	OverActiveHosts       int       `json:"over_active_hosts"`
	ActiveHosts           int       `json:"active_hosts"`          // Hosts ingested in last hour
	HealthPercentage      float64   `json:"health_percentage"`     // % of hosts with healthy ingestion
	AverageIngestionRate  float64   `json:"average_ingestion_rate"` // Average ingestions per hour
	MaxIngestionRate      float64   `json:"max_ingestion_rate"`     // Highest ingestion rate
	Timestamp             time.Time `json:"timestamp"`
}

// CleanupOldData removes tracking data for hosts that haven't been seen in a long time
func (it *IngestionTracker) CleanupOldData(maxAge time.Duration) {
	it.mu.Lock()
	defer it.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)

	for hostID, lastIngestion := range it.hostLastIngestion {
		if lastIngestion.Before(cutoff) {
			delete(it.hostLastIngestion, hostID)
			delete(it.hostIngestionCount, hostID)
			delete(it.hostIngestionHistory, hostID)

			level.Debug(it.logger).Log(
				"msg", "cleaned up old ingestion tracking data",
				"host_id", hostID,
				"last_seen", lastIngestion,
			)
		}
	}
}

// StartPeriodicReporting starts a background goroutine that logs ingestion summaries
func (it *IngestionTracker) StartPeriodicReporting(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			summary := it.GetIngestionSummary()

			level.Info(it.logger).Log(
				"msg", "software ingestion summary",
				"total_hosts", summary.TotalHosts,
				"healthy_hosts", summary.HealthyHosts,
				"stale_hosts", summary.StaleHosts,
				"over_active_hosts", summary.OverActiveHosts,
				"health_percentage", summary.HealthPercentage,
				"avg_ingestion_rate", summary.AverageIngestionRate,
			)

			// Alert if too many stale hosts
			if summary.TotalHosts > 0 {
				stalePercentage := float64(summary.StaleHosts) / float64(summary.TotalHosts) * 100
				if stalePercentage > 10.0 { // More than 10% of hosts are stale
					level.Warn(it.logger).Log(
						"msg", "high percentage of stale hosts detected",
						"stale_percentage", stalePercentage,
						"stale_count", summary.StaleHosts,
						"total_count", summary.TotalHosts,
					)
				}
			}

			// Cleanup old data every hour
			it.CleanupOldData(7 * 24 * time.Hour) // Keep data for 7 days
		}
	}
}