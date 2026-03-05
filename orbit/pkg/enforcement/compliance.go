package enforcement

import (
	"sync"
	"time"
)

// ComplianceRecord is a single compliance entry, read by the osquery table.
type ComplianceRecord struct {
	SettingName  string
	Category     string
	PolicyName   string
	CISRef       string
	DesiredValue string
	CurrentValue string
	Compliant    bool
	LastChecked  time.Time
}

// ComplianceCache holds the latest compliance results for osquery to read.
type ComplianceCache struct {
	mu      sync.RWMutex
	records []ComplianceRecord
}

// NewComplianceCache creates a new empty cache.
func NewComplianceCache() *ComplianceCache {
	return &ComplianceCache{}
}

// Update replaces the cache contents with new records.
func (c *ComplianceCache) Update(records []ComplianceRecord) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.records = records
}

// Records returns a copy of the current compliance records.
func (c *ComplianceCache) Records() []ComplianceRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]ComplianceRecord, len(c.records))
	copy(result, c.records)
	return result
}
