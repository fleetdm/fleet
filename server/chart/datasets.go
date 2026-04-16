package chart

import (
	"context"
	"time"
)

// UptimeDataset implements Dataset for host uptime tracking.
type UptimeDataset struct{}

func (u *UptimeDataset) Name() string             { return "uptime" }
func (u *UptimeDataset) StorageType() StorageType { return StorageTypeBlob }

func (u *UptimeDataset) Collect(ctx context.Context, store DatasetStore, now time.Time) error {
	return store.CollectUptimeChartData(ctx, now)
}

func (u *UptimeDataset) ResolveFilters(_ context.Context, _ DatasetStore, _ map[string]string) ([]string, error) {
	return nil, nil
}

func (u *UptimeDataset) SupportedFilters() []FilterDef {
	return nil
}

func (u *UptimeDataset) DefaultVisualization() string {
	return "checkerboard"
}

func (u *UptimeDataset) HasEntityDimension() bool {
	return false
}

// CVEDataset implements Dataset for host CVE tracking.
type CVEDataset struct{}

func (u *CVEDataset) Name() string             { return "cve" }
func (u *CVEDataset) StorageType() StorageType { return StorageTypeSCD }

// Collect is a stub for now — CVE SCD data is recorded externally (by the
// charts-collect script) via the SCD record path, not via this in-process collector.
func (u *CVEDataset) Collect(_ context.Context, _ DatasetStore, _ time.Time) error {
	return nil
}

func (u *CVEDataset) ResolveFilters(_ context.Context, _ DatasetStore, _ map[string]string) ([]string, error) {
	return nil, nil
}

func (u *CVEDataset) SupportedFilters() []FilterDef {
	return nil
}

func (u *CVEDataset) DefaultVisualization() string {
	return "line"
}

func (u *CVEDataset) HasEntityDimension() bool {
	return true
}

// PolicyFailingDataset implements Dataset for per-policy host compliance tracking.
type PolicyFailingDataset struct{}

func (p *PolicyFailingDataset) Name() string             { return "policy_failing" }
func (p *PolicyFailingDataset) StorageType() StorageType { return StorageTypeSCD }

func (p *PolicyFailingDataset) Collect(ctx context.Context, store DatasetStore, now time.Time) error {
	return store.CollectPolicyFailingChartData(ctx, now)
}

func (p *PolicyFailingDataset) ResolveFilters(_ context.Context, _ DatasetStore, params map[string]string) ([]string, error) {
	if pid, ok := params["policy_id"]; ok && pid != "" {
		return []string{pid}, nil
	}
	return nil, nil
}

func (p *PolicyFailingDataset) SupportedFilters() []FilterDef {
	return []FilterDef{
		{
			Name:        "policy_id",
			Label:       "Policy",
			Type:        "multi_select",
			Description: "Restrict the chart to a specific policy",
		},
	}
}

func (p *PolicyFailingDataset) DefaultVisualization() string {
	return "stacked_bar"
}

func (p *PolicyFailingDataset) HasEntityDimension() bool {
	return true
}
