package software_ingestion

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// SoftwareIngestionService handles ingesting software data from multiple sources
// and updating the software-related database tables.
//
// This service owns the following database tables:
// - software: The software catalog
// - host_software: Links hosts to their installed software
// - host_software_installed_paths: Software installation paths per host
//
// Note: Vulnerability data (software_cve, software_cpe) is handled by a separate VulnerabilitiesService
type SoftwareIngestionService interface {
	// IngestOsquerySoftware processes software data reported by osquery agents
	IngestOsquerySoftware(ctx context.Context, hostID uint, host *fleet.Host, softwareRows []map[string]string) error

	// IngestMDMSoftware processes software data from MDM sources (iOS, iPadOS, macOS apps)
	IngestMDMSoftware(ctx context.Context, hostID uint, host *fleet.Host, software []fleet.Software) error
}

// Datastore defines the interface for software-related database operations
type Datastore interface {
	UpdateHostSoftware(ctx context.Context, hostID uint, software []fleet.Software) (*fleet.UpdateHostSoftwareDBResult, error)
	UpdateHostSoftwareInstalledPaths(ctx context.Context, hostID uint, reported map[string]struct{}, mutationResults *fleet.UpdateHostSoftwareDBResult) error
}

type service struct {
	ds     Datastore
	logger log.Logger
}

// NewService creates a new SoftwareIngestionService
func NewService(ds Datastore, logger log.Logger) SoftwareIngestionService {
	return &service{
		ds:     ds,
		logger: logger,
	}
}

// IngestOsquerySoftware implements the core logic from directIngestSoftware
func (s *service) IngestOsquerySoftware(ctx context.Context, hostID uint, host *fleet.Host, softwareRows []map[string]string) error {
	var software []fleet.Software
	installedPaths := map[string]struct{}{}

	for _, row := range softwareRows {
		// Validate and parse the software row
		parsedSoftware, err := s.parseSoftwareRow(ctx, host, row)
		if err != nil {
			level.Debug(s.logger).Log(
				"msg", "failed to parse software row",
				"host_id", hostID,
				"row", fmt.Sprintf("%+v", row),
				"err", err,
			)
			continue
		}

		// Apply platform-specific transformations
		s.applySoftwareTransformations(host, parsedSoftware)

		// Apply ingestion mutations (sanitization, normalization)
		applySoftwareMutations(parsedSoftware, s.logger)

		// Skip software that should be filtered out
		if s.shouldFilterSoftware(host, parsedSoftware) {
			continue
		}

		software = append(software, *parsedSoftware)

		// Collect installed paths for this software
		if path := s.extractInstalledPath(row, parsedSoftware); path != "" {
			installedPaths[path] = struct{}{}
		}
	}

	// Persist software data to database
	result, err := s.ds.UpdateHostSoftware(ctx, hostID, software)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update host software")
	}

	// Update installed paths
	if err := s.ds.UpdateHostSoftwareInstalledPaths(ctx, hostID, installedPaths, result); err != nil {
		return ctxerr.Wrap(ctx, err, "update software installed paths")
	}

	level.Debug(s.logger).Log(
		"msg", "ingested software for host",
		"host_id", hostID,
		"software_count", len(software),
		"paths_count", len(installedPaths),
	)

	return nil
}

// IngestMDMSoftware processes software data from MDM sources (iOS, iPadOS, macOS apps)
func (s *service) IngestMDMSoftware(ctx context.Context, hostID uint, host *fleet.Host, software []fleet.Software) error {
	if len(software) == 0 {
		level.Debug(s.logger).Log(
			"msg", "no MDM software to ingest",
			"host_id", hostID,
		)
		return nil
	}

	// Apply platform-specific transformations for MDM software
	var processedSoftware []fleet.Software
	for _, sw := range software {
		// Make a copy to avoid modifying the input
		processed := sw

		// Set the appropriate source based on platform
		processed.Source = s.determineMDMSoftwareSource(host.Platform)

		// Apply MDM-specific transformations
		s.applyMDMSoftwareTransformations(host, &processed)

		// Skip software that should be filtered out
		if s.shouldFilterSoftware(host, &processed) {
			continue
		}

		processedSoftware = append(processedSoftware, processed)
	}

	// Persist software data to database
	result, err := s.ds.UpdateHostSoftware(ctx, hostID, processedSoftware)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update host MDM software")
	}

	// MDM software typically doesn't have installed paths like osquery software,
	// but we'll call with empty paths to ensure consistency
	emptyPaths := map[string]struct{}{}
	if err := s.ds.UpdateHostSoftwareInstalledPaths(ctx, hostID, emptyPaths, result); err != nil {
		return ctxerr.Wrap(ctx, err, "update MDM software installed paths")
	}

	level.Debug(s.logger).Log(
		"msg", "ingested MDM software for host",
		"host_id", hostID,
		"platform", host.Platform,
		"software_count", len(processedSoftware),
	)

	return nil
}

// determineMDMSoftwareSource returns the appropriate source string for MDM software
func (s *service) determineMDMSoftwareSource(platform string) string {
	switch platform {
	case "ios":
		return "ios_apps"
	case "ipados":
		return "ipados_apps"
	case "darwin":
		return "app_store_apps" // macOS App Store apps
	default:
		return "mdm_apps" // fallback for other platforms
	}
}

// applyMDMSoftwareTransformations applies MDM-specific transformations
func (s *service) applyMDMSoftwareTransformations(host *fleet.Host, software *fleet.Software) {
	// MDM software typically comes with bundle identifiers already set
	// Apply any platform-specific normalization here

	// Ensure we have reasonable defaults for required fields
	if software.Vendor == "" && software.BundleIdentifier != nil {
		// Extract vendor from bundle identifier if possible
		if parts := strings.Split(*software.BundleIdentifier, "."); len(parts) >= 2 {
			software.Vendor = parts[1] // e.g., "com.apple.music" -> "apple"
		}
	}
}

// parseSoftwareRow converts an osquery row to fleet.Software
func (s *service) parseSoftwareRow(ctx context.Context, host *fleet.Host, row map[string]string) (*fleet.Software, error) {
	// Validate last_opened_at timestamp
	if _, err := fleet.ParseSoftwareLastOpenedAtRowValue(row["last_opened_at"]); err != nil {
		level.Debug(s.logger).Log(
			"msg", "host reported software with invalid last opened timestamp",
			"host_id", host.ID,
			"row", fmt.Sprintf("%+v", row),
		)
	}

	return fleet.SoftwareFromOsqueryRow(
		row["name"],
		row["version"],
		row["source"],
		row["vendor"],
		row["installed_path"],
		row["release"],
		row["arch"],
		row["bundle_identifier"],
		row["extension_id"],
		row["extension_for"],
		row["last_opened_at"],
	)
}

// applySoftwareTransformations applies platform-specific logic
func (s *service) applySoftwareTransformations(host *fleet.Host, software *fleet.Software) {
	// Mark Linux kernel packages
	if fleet.IsLinux(host.Platform) {
		if isKernelPackage(software.Name) {
			software.IsKernel = true
		}
	}

	// Add other platform-specific transformations here
}

// extractInstalledPath builds the installed path key for tracking
func (s *service) extractInstalledPath(row map[string]string, software *fleet.Software) string {
	installedPath := strings.TrimSpace(row["installed_path"])
	if installedPath == "" || strings.ToLower(installedPath) == "null" {
		return ""
	}

	// Truncate team identifier to max length
	teamIdentifier := truncateString(row["team_identifier"], fleet.SoftwareTeamIdentifierMaxLength)

	var cdhashSHA256 string
	if hash, ok := row["cdhash_sha256"]; ok {
		cdhashSHA256 = hash
	}

	// Build composite key for installed path tracking
	return fmt.Sprintf(
		"%s%s%s%s%s%s%s",
		installedPath,
		fleet.SoftwareFieldSeparator,
		teamIdentifier,
		fleet.SoftwareFieldSeparator,
		cdhashSHA256,
		fleet.SoftwareFieldSeparator,
		software.ToUniqueStr(),
	)
}

// shouldFilterSoftware determines if software should be excluded
func (s *service) shouldFilterSoftware(host *fleet.Host, software *fleet.Software) bool {
	// Implement the logic from shouldRemoveSoftware function
	// This would be extracted from the current implementation
	return false // placeholder
}

// Kernel detection regexes (extracted from original code)
const (
	linuxImageRegex       = `^linux-image-[[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+-[[:digit:]]+-[[:alnum:]]+`
	amazonLinuxKernelName = "kernel"
	rhelKernelName        = "kernel-core"
	archKernelName        = `^linux(?:-(?:lts|zen|hardened))?$`
)

var (
	kernelRegex     = regexp.MustCompile(linuxImageRegex)
	archKernelRegex = regexp.MustCompile(archKernelName)
)

func isKernelPackage(name string) bool {
	return kernelRegex.MatchString(name) ||
		name == amazonLinuxKernelName ||
		name == rhelKernelName ||
		archKernelRegex.MatchString(name)
}

func truncateString(str string, length int) string {
	runes := []rune(str)
	if len(runes) > length {
		return string(runes[:length])
	}
	return str
}

// applySoftwareMutations applies data sanitization and normalization
// This would extract the logic from MutateSoftwareOnIngestion
func applySoftwareMutations(software *fleet.Software, logger log.Logger) {
	// Implementation would be extracted from the current MutateSoftwareOnIngestion function
	// in server/service/osquery_utils/queries.go
}