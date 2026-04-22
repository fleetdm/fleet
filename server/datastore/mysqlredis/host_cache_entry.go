package mysqlredis

import (
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// hostCacheEntry is the cached shape of a fleet.Host as returned by
// LoadHostByNodeKey. It exists because fleet.Host tags several critical fields
// with `json:"-"` — NodeKey, OrbitNodeKey, OsqueryHostID, HasHostIdentityCert,
// Platform — to keep them out of HTTP API responses. Using fleet.Host directly
// with encoding/json would silently drop those fields on cache round-trip and
// break auth (HasHostIdentityCert → nil would skip the httpsig check).
//
// The field set mirrors LoadHostByNodeKey's SELECT list in
// server/datastore/mysql/hosts.go. When that query gains a new column that
// downstream code reads, this struct and the conversion functions below must
// be updated in lockstep. TestHostCachePreservesLoadHostByNodeKeyFields (in
// host_cache_test.go) is intended to catch drift.
type hostCacheEntry struct {
	ID                          uint          `json:"id"`
	OsqueryHostID               *string       `json:"osquery_host_id,omitempty"`
	CreatedAt                   time.Time     `json:"created_at"`
	UpdatedAt                   time.Time     `json:"updated_at"`
	DetailUpdatedAt             time.Time     `json:"detail_updated_at"`
	NodeKey                     *string       `json:"node_key,omitempty"`
	Hostname                    string        `json:"hostname"`
	UUID                        string        `json:"uuid"`
	Platform                    string        `json:"platform"`
	OsqueryVersion              string        `json:"osquery_version"`
	OSVersion                   string        `json:"os_version"`
	Build                       string        `json:"build"`
	PlatformLike                string        `json:"platform_like"`
	CodeName                    string        `json:"code_name"`
	Uptime                      time.Duration `json:"uptime"`
	Memory                      int64         `json:"memory"`
	CPUType                     string        `json:"cpu_type"`
	CPUSubtype                  string        `json:"cpu_subtype"`
	CPUBrand                    string        `json:"cpu_brand"`
	CPUPhysicalCores            int           `json:"cpu_physical_cores"`
	CPULogicalCores             int           `json:"cpu_logical_cores"`
	HardwareVendor              string        `json:"hardware_vendor"`
	HardwareModel               string        `json:"hardware_model"`
	HardwareVersion             string        `json:"hardware_version"`
	HardwareSerial              string        `json:"hardware_serial"`
	ComputerName                string        `json:"computer_name"`
	PrimaryNetworkInterfaceID   *uint         `json:"primary_ip_id,omitempty"`
	DistributedInterval         uint          `json:"distributed_interval"`
	LoggerTLSPeriod             uint          `json:"logger_tls_period"`
	ConfigTLSRefresh            uint          `json:"config_tls_refresh"`
	PrimaryIP                   string        `json:"primary_ip"`
	PrimaryMac                  string        `json:"primary_mac"`
	LabelUpdatedAt              time.Time     `json:"label_updated_at"`
	LastEnrolledAt              time.Time     `json:"last_enrolled_at"`
	RefetchRequested            bool          `json:"refetch_requested"`
	RefetchCriticalQueriesUntil *time.Time    `json:"refetch_critical_queries_until,omitempty"`
	TeamID                      *uint         `json:"team_id,omitempty"`
	PolicyUpdatedAt             time.Time     `json:"policy_updated_at"`
	PublicIP                    string        `json:"public_ip"`
	OrbitNodeKey                *string       `json:"orbit_node_key,omitempty"`
	LastRestartedAt             time.Time     `json:"last_restarted_at"`
	TimeZone                    *string       `json:"timezone,omitempty"`
	GigsDiskSpaceAvailable      float64       `json:"gigs_disk_space_available"`
	GigsTotalDiskSpace          float64       `json:"gigs_total_disk_space"`
	PercentDiskSpaceAvailable   float64       `json:"percent_disk_space_available"`
	HasHostIdentityCert         *bool         `json:"has_host_identity_cert,omitempty"`
}

// hostCacheEntryFromHost copies the subset of fields that LoadHostByNodeKey
// populates into the cache-entry shape. Callers are expected to null-check the
// host before calling; this function does not guard against a nil argument.
func hostCacheEntryFromHost(h *fleet.Host) *hostCacheEntry {
	return &hostCacheEntry{
		ID:                          h.ID,
		OsqueryHostID:               h.OsqueryHostID,
		CreatedAt:                   h.CreatedAt,
		UpdatedAt:                   h.UpdatedAt,
		DetailUpdatedAt:             h.DetailUpdatedAt,
		NodeKey:                     h.NodeKey,
		Hostname:                    h.Hostname,
		UUID:                        h.UUID,
		Platform:                    h.Platform,
		OsqueryVersion:              h.OsqueryVersion,
		OSVersion:                   h.OSVersion,
		Build:                       h.Build,
		PlatformLike:                h.PlatformLike,
		CodeName:                    h.CodeName,
		Uptime:                      h.Uptime,
		Memory:                      h.Memory,
		CPUType:                     h.CPUType,
		CPUSubtype:                  h.CPUSubtype,
		CPUBrand:                    h.CPUBrand,
		CPUPhysicalCores:            h.CPUPhysicalCores,
		CPULogicalCores:             h.CPULogicalCores,
		HardwareVendor:              h.HardwareVendor,
		HardwareModel:               h.HardwareModel,
		HardwareVersion:             h.HardwareVersion,
		HardwareSerial:              h.HardwareSerial,
		ComputerName:                h.ComputerName,
		PrimaryNetworkInterfaceID:   h.PrimaryNetworkInterfaceID,
		DistributedInterval:         h.DistributedInterval,
		LoggerTLSPeriod:             h.LoggerTLSPeriod,
		ConfigTLSRefresh:            h.ConfigTLSRefresh,
		PrimaryIP:                   h.PrimaryIP,
		PrimaryMac:                  h.PrimaryMac,
		LabelUpdatedAt:              h.LabelUpdatedAt,
		LastEnrolledAt:              h.LastEnrolledAt,
		RefetchRequested:            h.RefetchRequested,
		RefetchCriticalQueriesUntil: h.RefetchCriticalQueriesUntil,
		TeamID:                      h.TeamID,
		PolicyUpdatedAt:             h.PolicyUpdatedAt,
		PublicIP:                    h.PublicIP,
		OrbitNodeKey:                h.OrbitNodeKey,
		LastRestartedAt:             h.LastRestartedAt,
		TimeZone:                    h.TimeZone,
		GigsDiskSpaceAvailable:      h.GigsDiskSpaceAvailable,
		GigsTotalDiskSpace:          h.GigsTotalDiskSpace,
		PercentDiskSpaceAvailable:   h.PercentDiskSpaceAvailable,
		HasHostIdentityCert:         h.HasHostIdentityCert,
	}
}

// toHost returns a fresh *fleet.Host populated from the entry. Not safe to
// call on a nil receiver; the internal callers always have a non-nil entry.
func (e *hostCacheEntry) toHost() *fleet.Host {
	h := &fleet.Host{
		ID:                          e.ID,
		OsqueryHostID:               e.OsqueryHostID,
		DetailUpdatedAt:             e.DetailUpdatedAt,
		NodeKey:                     e.NodeKey,
		Hostname:                    e.Hostname,
		UUID:                        e.UUID,
		Platform:                    e.Platform,
		OsqueryVersion:              e.OsqueryVersion,
		OSVersion:                   e.OSVersion,
		Build:                       e.Build,
		PlatformLike:                e.PlatformLike,
		CodeName:                    e.CodeName,
		Uptime:                      e.Uptime,
		Memory:                      e.Memory,
		CPUType:                     e.CPUType,
		CPUSubtype:                  e.CPUSubtype,
		CPUBrand:                    e.CPUBrand,
		CPUPhysicalCores:            e.CPUPhysicalCores,
		CPULogicalCores:             e.CPULogicalCores,
		HardwareVendor:              e.HardwareVendor,
		HardwareModel:               e.HardwareModel,
		HardwareVersion:             e.HardwareVersion,
		HardwareSerial:              e.HardwareSerial,
		ComputerName:                e.ComputerName,
		PrimaryNetworkInterfaceID:   e.PrimaryNetworkInterfaceID,
		DistributedInterval:         e.DistributedInterval,
		LoggerTLSPeriod:             e.LoggerTLSPeriod,
		ConfigTLSRefresh:            e.ConfigTLSRefresh,
		PrimaryIP:                   e.PrimaryIP,
		PrimaryMac:                  e.PrimaryMac,
		LabelUpdatedAt:              e.LabelUpdatedAt,
		LastEnrolledAt:              e.LastEnrolledAt,
		RefetchRequested:            e.RefetchRequested,
		RefetchCriticalQueriesUntil: e.RefetchCriticalQueriesUntil,
		TeamID:                      e.TeamID,
		PolicyUpdatedAt:             e.PolicyUpdatedAt,
		PublicIP:                    e.PublicIP,
		OrbitNodeKey:                e.OrbitNodeKey,
		LastRestartedAt:             e.LastRestartedAt,
		TimeZone:                    e.TimeZone,
		GigsDiskSpaceAvailable:      e.GigsDiskSpaceAvailable,
		GigsTotalDiskSpace:          e.GigsTotalDiskSpace,
		PercentDiskSpaceAvailable:   e.PercentDiskSpaceAvailable,
		HasHostIdentityCert:         e.HasHostIdentityCert,
	}
	h.CreatedAt = e.CreatedAt
	h.UpdatedAt = e.UpdatedAt
	return h
}
