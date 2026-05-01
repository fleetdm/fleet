package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type OSVData struct {
	SchemaVersion string     `json:"schema_version"`
	ID            string     `json:"id"`
	Published     string     `json:"published"`
	Modified      string     `json:"modified"`
	Details       string     `json:"details"`
	Affected      []Affected `json:"affected"`
	Upstream      []string   `json:"upstream,omitempty"`
	Related       []string   `json:"related,omitempty"`
}

type Affected struct {
	Package           Package        `json:"package"`
	Ranges            []Range        `json:"ranges"`
	Versions          []string       `json:"versions,omitempty"`
	EcosystemSpecific map[string]any `json:"ecosystem_specific,omitempty"`
	DatabaseSpecific  map[string]any `json:"database_specific,omitempty"`
}

type Package struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
	Purl      string `json:"purl,omitempty"`
}

type Range struct {
	Type   string  `json:"type"`
	Events []Event `json:"events"`
}

type Event struct {
	Introduced string `json:"introduced,omitempty"`
	Fixed      string `json:"fixed,omitempty"`
}

type ProcessedVuln struct {
	CVE        string   `json:"cve"`
	Published  string   `json:"published"`
	Modified   string   `json:"modified"`
	Introduced string   `json:"introduced,omitempty"`
	Fixed      string   `json:"fixed,omitempty"`
	Versions   []string `json:"versions,omitempty"`
}

type Config struct {
	Platform              string
	InputDir              string
	OutputDir             string
	Versions              string
	ExcludeVersions       string
	ChangedFilesToday     string
	ChangedFilesYesterday string
	DateStr               string
	YesterdayStr          string
	GeneratedTimestamp    string
	RunTime               time.Time
}

type ArtifactData struct {
	SchemaVersion   string                     `json:"schema_version"`
	UbuntuVersion   string                     `json:"ubuntu_version"`
	Generated       string                     `json:"generated"`
	TotalCVEs       int                        `json:"total_cves"`
	TotalPackages   int                        `json:"total_packages"`
	Vulnerabilities map[string][]ProcessedVuln `json:"vulnerabilities"`
}

type RHELArtifactData struct {
	SchemaVersion   string                     `json:"schema_version"`
	RHELVersion     string                     `json:"rhel_version"`
	Generated       string                     `json:"generated"`
	TotalCVEs       int                        `json:"total_cves"`
	TotalPackages   int                        `json:"total_packages"`
	Vulnerabilities map[string][]ProcessedVuln `json:"vulnerabilities"`
}

func main() {
	platform := flag.String("platform", "ubuntu", "Platform to process: ubuntu or rhel")
	inputDir := flag.String("input", "", "Input directory with OSV JSON files (default: /tmp/ubuntu-osv for ubuntu, /tmp/rhel-osv for rhel)")
	outputDir := flag.String("output", "./artifacts", "Output directory for artifacts")
	versions := flag.String("versions", "", "Comma-separated versions to process (inclusive)")
	excludeVersions := flag.String("exclude-versions", "", "Comma-separated versions to exclude (ignored if --versions is set)")
	changedFilesToday := flag.String("changed-files-today", "", "Path to file containing CVE files changed today (ubuntu only)")
	changedFilesYesterday := flag.String("changed-files-yesterday", "", "Path to file containing CVE files changed yesterday (ubuntu only)")
	flag.Parse()

	if *inputDir == "" {
		switch *platform {
		case "rhel":
			*inputDir = "/tmp/rhel-osv"
		default:
			*inputDir = "/tmp/ubuntu-osv"
		}
	}

	runTime := time.Now().UTC()

	cfg := Config{
		Platform:              *platform,
		InputDir:              *inputDir,
		OutputDir:             *outputDir,
		Versions:              *versions,
		ExcludeVersions:       *excludeVersions,
		ChangedFilesToday:     *changedFilesToday,
		ChangedFilesYesterday: *changedFilesYesterday,
		DateStr:               runTime.Format("2006-01-02"),
		YesterdayStr:          runTime.AddDate(0, 0, -1).Format("2006-01-02"),
		GeneratedTimestamp:    runTime.Format(time.RFC3339),
		RunTime:               runTime,
	}

	switch cfg.Platform {
	case "ubuntu":
		if err := run(cfg); err != nil {
			log.Fatalf("Error: %v", err)
		}
	case "rhel":
		if err := runRHEL(cfg); err != nil {
			log.Fatalf("Error: %v", err)
		}
	default:
		log.Fatalf("Unknown platform: %s (supported: ubuntu, rhel)", cfg.Platform)
	}
}

func run(cfg Config) error {
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build version filter
	targetVersions, excludedVersions := buildVersionFilter(cfg.Versions, cfg.ExcludeVersions)
	switch {
	case targetVersions != nil:
		log.Printf("Processing OSV files from %s for versions: %s", cfg.InputDir, cfg.Versions)
	case excludedVersions != nil:
		log.Printf("Processing OSV files from %s (auto-detecting, excluding: %s)", cfg.InputDir, cfg.ExcludeVersions)
	default:
		log.Printf("Processing OSV files from %s (auto-detecting all versions)", cfg.InputDir)
	}

	// Load changed CVE files for delta generation
	var todayCVEFiles, yesterdayCVEFiles map[string]struct{}
	generateTodayDeltas := cfg.ChangedFilesToday != ""
	generateYesterdayDeltas := cfg.ChangedFilesYesterday != ""

	if generateTodayDeltas {
		log.Printf("Loading today's changed CVE files from %s", cfg.ChangedFilesToday)
		var err error
		todayCVEFiles, err = loadChangedFiles(cfg.ChangedFilesToday)
		if err != nil {
			return fmt.Errorf("failed to load today's changed files: %w", err)
		}
		log.Printf("Found %d CVE files changed today", len(todayCVEFiles))
	}

	if generateYesterdayDeltas {
		log.Printf("Loading yesterday's changed CVE files from %s", cfg.ChangedFilesYesterday)
		var err error
		yesterdayCVEFiles, err = loadChangedFiles(cfg.ChangedFilesYesterday)
		if err != nil {
			return fmt.Errorf("failed to load yesterday's changed files: %w", err)
		}
		log.Printf("Found %d CVE files changed yesterday", len(yesterdayCVEFiles))
	}

	artifacts := make(map[string]*ArtifactData)
	todayArtifacts := make(map[string]*ArtifactData)
	yesterdayArtifacts := make(map[string]*ArtifactData)

	filesProcessed := 0
	filesSkipped := 0

	err := filepath.Walk(cfg.InputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		osvData, err := parseOSVFile(path)
		if err != nil {
			log.Printf("Failed to parse %s: %v", path, err)
			filesSkipped++
			return nil
		}

		inToday := false
		inYesterday := false
		if generateTodayDeltas {
			inToday = shouldIncludeInDelta(cfg.InputDir, path, todayCVEFiles)
		}
		if generateYesterdayDeltas {
			inYesterday = shouldIncludeInDelta(cfg.InputDir, path, yesterdayCVEFiles)
		}

		for _, affected := range osvData.Affected {
			ecosystem := affected.Package.Ecosystem
			packageName := affected.Package.Name

			ubuntuVer := extractUbuntuVersion(ecosystem)
			if ubuntuVer == "" {
				continue
			}

			// Filter versions based on flags
			if targetVersions != nil {
				// Inclusive mode: only process if in target list
				if !targetVersions[ubuntuVer] {
					continue
				}
			} else if excludedVersions != nil {
				// Exclusive mode: skip if in excluded list
				if excludedVersions[ubuntuVer] {
					continue
				}
			}
			// Otherwise auto-detect all versions (no filtering)

			cveID := extractCVEID(osvData)
			if cveID == "" {
				cveID = osvData.ID
			}

			introduced, fixed := extractVersionRange(affected.Ranges)

			vuln := ProcessedVuln{
				CVE:        cveID,
				Published:  osvData.Published,
				Modified:   osvData.Modified,
				Introduced: introduced,
				Fixed:      fixed,
				Versions:   affected.Versions,
			}

			// Apply any transformations/filters to modify the package name or cve
			packages, modifiedVuln := transformVuln(packageName, cveID, &vuln)
			if packages == nil {
				continue
			}
			// Use modified vulnerability if provided, otherwise use original
			vulnToUse := &vuln
			if modifiedVuln != nil {
				vulnToUse = modifiedVuln
			}

			for _, pkg := range packages {
				if _, exists := artifacts[ubuntuVer]; !exists {
					artifacts[ubuntuVer] = &ArtifactData{
						SchemaVersion:   "1.0",
						UbuntuVersion:   ubuntuVer,
						Vulnerabilities: make(map[string][]ProcessedVuln),
					}
				}
				artifacts[ubuntuVer].Vulnerabilities[pkg] = append(artifacts[ubuntuVer].Vulnerabilities[pkg], *vulnToUse)
			}

			// Add to today's delta artifact if this file was changed today
			if inToday {
				for _, pkg := range packages {
					if _, exists := todayArtifacts[ubuntuVer]; !exists {
						todayArtifacts[ubuntuVer] = &ArtifactData{
							SchemaVersion:   "1.0",
							UbuntuVersion:   ubuntuVer,
							Vulnerabilities: make(map[string][]ProcessedVuln),
						}
					}
					todayArtifacts[ubuntuVer].Vulnerabilities[pkg] = append(todayArtifacts[ubuntuVer].Vulnerabilities[pkg], *vulnToUse)
				}
			}

			// Add to yesterday's delta artifact if this file was changed yesterday
			if inYesterday {
				for _, pkg := range packages {
					if _, exists := yesterdayArtifacts[ubuntuVer]; !exists {
						yesterdayArtifacts[ubuntuVer] = &ArtifactData{
							SchemaVersion:   "1.0",
							UbuntuVersion:   ubuntuVer,
							Vulnerabilities: make(map[string][]ProcessedVuln),
						}
					}
					yesterdayArtifacts[ubuntuVer].Vulnerabilities[pkg] = append(yesterdayArtifacts[ubuntuVer].Vulnerabilities[pkg], *vulnToUse)
				}
			}
		}

		filesProcessed++
		if filesProcessed%1000 == 0 {
			log.Printf("Processed %d files...", filesProcessed)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error walking directory: %w", err)
	}

	log.Printf("Processed %d files, skipped %d files", filesProcessed, filesSkipped)
	log.Printf("Discovered %d Ubuntu versions", len(artifacts))

	// Write full artifacts
	for ver, artifact := range artifacts {
		artifact.Generated = cfg.GeneratedTimestamp
		artifact.TotalCVEs = countTotalCVEs(artifact)
		artifact.TotalPackages = len(artifact.Vulnerabilities)

		outputFile := filepath.Join(cfg.OutputDir, fmt.Sprintf("osv-ubuntu-%s-%s.json.gz",
			strings.ReplaceAll(ver, ".", ""),
			cfg.DateStr))

		if err := writeArtifact(outputFile, artifact); err != nil {
			return fmt.Errorf("failed to write artifact for Ubuntu %s: %w", ver, err)
		}

		log.Printf("Ubuntu %s: %d packages, %d CVEs -> %s",
			ver, artifact.TotalPackages, artifact.TotalCVEs, outputFile)
	}

	// Write delta artifacts (if any were generated)
	if generateTodayDeltas && len(todayArtifacts) > 0 {
		log.Printf("\nWriting today's delta artifacts (%s)...", cfg.DateStr)
		for ver, artifact := range todayArtifacts {
			artifact.Generated = cfg.GeneratedTimestamp
			artifact.TotalCVEs = countTotalCVEs(artifact)
			artifact.TotalPackages = len(artifact.Vulnerabilities)

			outputFile := filepath.Join(cfg.OutputDir, fmt.Sprintf("osv-ubuntu-%s-delta-%s.json.gz",
				strings.ReplaceAll(ver, ".", ""), cfg.DateStr))

			if err := writeArtifact(outputFile, artifact); err != nil {
				return fmt.Errorf("failed to write today's delta for Ubuntu %s: %w", ver, err)
			}

			log.Printf("Ubuntu %s (today): %d packages, %d CVEs -> %s",
				ver, artifact.TotalPackages, artifact.TotalCVEs, outputFile)
		}
	}

	if generateYesterdayDeltas && len(yesterdayArtifacts) > 0 {
		log.Printf("\nWriting yesterday's delta artifacts (%s)...", cfg.YesterdayStr)
		for ver, artifact := range yesterdayArtifacts {
			artifact.Generated = cfg.GeneratedTimestamp
			artifact.TotalCVEs = countTotalCVEs(artifact)
			artifact.TotalPackages = len(artifact.Vulnerabilities)

			outputFile := filepath.Join(cfg.OutputDir, fmt.Sprintf("osv-ubuntu-%s-delta-%s.json.gz",
				strings.ReplaceAll(ver, ".", ""), cfg.YesterdayStr))

			if err := writeArtifact(outputFile, artifact); err != nil {
				return fmt.Errorf("failed to write yesterday's delta for Ubuntu %s: %w", ver, err)
			}

			log.Printf("Ubuntu %s (yesterday): %d packages, %d CVEs -> %s",
				ver, artifact.TotalPackages, artifact.TotalCVEs, outputFile)
		}
	}

	return nil
}

func buildVersionFilter(versions, excludeVersions string) (targetVersions, excludedVersions map[string]bool) {
	if versions != "" {
		// Inclusive mode: only process specified versions
		targetVersions = make(map[string]bool)
		for ver := range strings.SplitSeq(versions, ",") {
			trimmed := strings.TrimSpace(ver)
			if trimmed != "" {
				targetVersions[trimmed] = true
			}
		}
		// If no valid versions were parsed, fall back to auto-detect
		if len(targetVersions) == 0 {
			return nil, nil
		}
		return targetVersions, nil
	}

	if excludeVersions != "" {
		// Exclusive mode: process all except specified versions
		excludedVersions = make(map[string]bool)
		for ver := range strings.SplitSeq(excludeVersions, ",") {
			trimmed := strings.TrimSpace(ver)
			if trimmed != "" {
				excludedVersions[trimmed] = true
			}
		}
		// If no valid versions were parsed, fall back to auto-detect
		if len(excludedVersions) == 0 {
			return nil, nil
		}
		return nil, excludedVersions
	}

	// Auto-detect all versions
	return nil, nil
}

func shouldIncludeInDelta(inputDir, filePath string, changedFiles map[string]struct{}) bool {
	relPath, err := filepath.Rel(inputDir, filePath)
	if err != nil {
		return false
	}

	normalizedRelPath := strings.TrimPrefix(filepath.ToSlash(relPath), "osv/cve/")
	fullRelPath := "osv/cve/" + normalizedRelPath

	_, exists := changedFiles[fullRelPath]
	return exists
}

func parseOSVFile(path string) (*OSVData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var osv OSVData
	if err := json.Unmarshal(data, &osv); err != nil {
		return nil, err
	}

	return &osv, nil
}

func extractUbuntuVersion(ecosystem string) string {
	// Example: "Ubuntu:24.04:LTS" -> "24.04"
	// Example: "Ubuntu:Pro:22.04:LTS" -> "22.04"
	for part := range strings.SplitSeq(ecosystem, ":") {
		// Look for version pattern like "24.04", "22.04", "20.04"
		if len(part) == 5 && strings.Contains(part, ".") {
			return part
		}
	}
	return ""
}

func extractCVEID(osv *OSVData) string {
	for _, upstream := range osv.Upstream {
		if strings.HasPrefix(upstream, "CVE-") {
			return upstream
		}
	}

	if strings.HasPrefix(osv.ID, "CVE-") {
		return osv.ID
	}

	if strings.HasPrefix(osv.ID, "UBUNTU-CVE-") {
		return strings.TrimPrefix(osv.ID, "UBUNTU-")
	}

	return ""
}

func extractVersionRange(ranges []Range) (introduced string, fixed string) {
	for _, r := range ranges {
		if r.Type == "ECOSYSTEM" {
			for _, event := range r.Events {
				if event.Introduced != "" && introduced == "" {
					introduced = event.Introduced
				}
				if event.Fixed != "" && fixed == "" {
					fixed = event.Fixed
				}
			}
		}
	}
	return
}

func countTotalCVEs(artifact *ArtifactData) int {
	seen := make(map[string]bool)
	for _, vulns := range artifact.Vulnerabilities {
		for _, vuln := range vulns {
			seen[vuln.CVE] = true
		}
	}
	return len(seen)
}

func writeArtifact(path string, artifact *ArtifactData) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); err == nil && cerr != nil {
			err = cerr
		}
	}()

	gzWriter := gzip.NewWriter(file)
	defer func() {
		if cerr := gzWriter.Close(); err == nil && cerr != nil {
			err = cerr
		}
	}()

	encoder := json.NewEncoder(gzWriter)

	if err = encoder.Encode(artifact); err != nil {
		return err
	}

	return nil
}

func loadChangedFiles(changedFilesPath string) (map[string]struct{}, error) {
	file, err := os.Open(changedFilesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open changed files list: %w", err)
	}
	defer file.Close()

	changedFiles := make(map[string]struct{})
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		changedFiles[line] = struct{}{}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading changed files: %w", err)
	}

	return changedFiles, nil
}

// extractRHELVersion extracts the major RHEL version from an ecosystem string.
// Only "enterprise_linux" ecosystems are supported; variants like rhel_e4s, rhel_eus,
// and rhel_software_collections are skipped.
//
// Repository suffixes (appstream, baseos, crb, nfv, realtime) and variant suffixes
// (server, workstation, client, computenode, fastdatapath, hypervisor) are stripped —
// all collapse to the same major version. For example, both
// "Red Hat:enterprise_linux:7::server" and "Red Hat:enterprise_linux:7::workstation"
// map to "7". Deduplication of CVE+package pairs across these variants happens in
// runRHEL.
//
// Examples:
//
//	"Red Hat:enterprise_linux:9::appstream"  -> "9"
//	"Red Hat:enterprise_linux:8::baseos"     -> "8"
//	"Red Hat:enterprise_linux:7::server"     -> "7"
//	"Red Hat:enterprise_linux:7::workstation"-> "7"
//	"Red Hat:enterprise_linux:10.0"          -> "10"
//	"Red Hat:enterprise_linux:10.1"          -> "10"
//	"Red Hat:rhel_e4s:8.8::appstream"        -> ""  (not enterprise_linux)
func extractRHELVersion(ecosystem string) string {
	parts := strings.Split(ecosystem, ":")
	if len(parts) < 3 || parts[0] != "Red Hat" {
		return ""
	}
	if parts[1] != "enterprise_linux" {
		return ""
	}
	// parts[2] is the version, possibly with minor: "9", "8", "10.0", "10.1"
	ver := parts[2]
	// Extract major version only
	if dotIdx := strings.Index(ver, "."); dotIdx >= 0 {
		ver = ver[:dotIdx]
	}
	return ver
}

// vulnKey is used for deduplication of CVE+package entries across ecosystems.
type vulnKey struct {
	pkg string
	cve string
}

func runRHEL(cfg Config) error {
	// Delta generation is not supported for RHEL — the data source is a full GCS zip
	// download with no git-based change tracking. Fail fast if callers pass delta flags.
	if cfg.ChangedFilesToday != "" || cfg.ChangedFilesYesterday != "" {
		return errors.New("--changed-files-today and --changed-files-yesterday are not supported with --platform rhel (no git-based change tracking for GCS data)")
	}

	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	targetVersions, excludedVersions := buildVersionFilter(cfg.Versions, cfg.ExcludeVersions)
	log.Printf("Processing RHEL OSV files from %s", cfg.InputDir)

	artifacts := make(map[string]*RHELArtifactData)
	// Track seen CVE+package pairs per version for deduplication across ecosystems
	seen := make(map[string]map[vulnKey]struct{})

	filesProcessed := 0
	filesSkipped := 0

	err := filepath.Walk(cfg.InputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		osvData, err := parseOSVFile(path)
		if err != nil {
			log.Printf("Failed to parse %s: %v", path, err)
			filesSkipped++
			return nil
		}

		// Extract all CVE IDs from this advisory
		cveIDs := extractCVEIDs(osvData)
		if len(cveIDs) == 0 {
			filesSkipped++
			return nil
		}

		for _, affected := range osvData.Affected {
			ecosystem := affected.Package.Ecosystem
			packageName := affected.Package.Name

			rhelVer := extractRHELVersion(ecosystem)
			if rhelVer == "" {
				continue
			}

			if targetVersions != nil {
				if !targetVersions[rhelVer] {
					continue
				}
			} else if excludedVersions != nil {
				if excludedVersions[rhelVer] {
					continue
				}
			}

			introduced, fixed := extractVersionRange(affected.Ranges)

			for _, cveID := range cveIDs {
				// Deduplicate: same CVE+package can appear in baseos, appstream, crb
				if seen[rhelVer] == nil {
					seen[rhelVer] = make(map[vulnKey]struct{})
				}
				key := vulnKey{pkg: packageName, cve: cveID}
				if _, exists := seen[rhelVer][key]; exists {
					continue
				}
				seen[rhelVer][key] = struct{}{}

				vuln := ProcessedVuln{
					CVE:        cveID,
					Published:  osvData.Published,
					Modified:   osvData.Modified,
					Introduced: introduced,
					Fixed:      fixed,
					Versions:   affected.Versions,
				}

				packages, modifiedVuln := transformVuln(packageName, cveID, &vuln)
				if packages == nil {
					continue
				}
				vulnToUse := &vuln
				if modifiedVuln != nil {
					vulnToUse = modifiedVuln
				}

				for _, pkg := range packages {
					if _, exists := artifacts[rhelVer]; !exists {
						artifacts[rhelVer] = &RHELArtifactData{
							SchemaVersion:   "1.0",
							RHELVersion:     rhelVer,
							Vulnerabilities: make(map[string][]ProcessedVuln),
						}
					}
					artifacts[rhelVer].Vulnerabilities[pkg] = append(artifacts[rhelVer].Vulnerabilities[pkg], *vulnToUse)
				}
			}
		}

		filesProcessed++
		if filesProcessed%1000 == 0 {
			log.Printf("Processed %d files...", filesProcessed)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error walking directory: %w", err)
	}

	log.Printf("Processed %d files, skipped %d files", filesProcessed, filesSkipped)
	log.Printf("Discovered %d RHEL versions", len(artifacts))

	for ver, artifact := range artifacts {
		artifact.Generated = cfg.GeneratedTimestamp
		artifact.TotalCVEs = countTotalRHELCVEs(artifact)
		artifact.TotalPackages = len(artifact.Vulnerabilities)

		outputFile := filepath.Join(cfg.OutputDir, fmt.Sprintf("osv-rhel-%s-%s.json.gz", ver, cfg.DateStr))

		if err := writeRHELArtifact(outputFile, artifact); err != nil {
			return fmt.Errorf("failed to write artifact for RHEL %s: %w", ver, err)
		}

		log.Printf("RHEL %s: %d packages, %d CVEs -> %s",
			ver, artifact.TotalPackages, artifact.TotalCVEs, outputFile)
	}

	return nil
}

// extractCVEIDs returns all CVE IDs from an OSV entry.
// RHEL advisories list CVEs in the "upstream" field (same as Ubuntu).
func extractCVEIDs(osv *OSVData) []string {
	var cves []string
	for _, upstream := range osv.Upstream {
		if strings.HasPrefix(upstream, "CVE-") {
			cves = append(cves, upstream)
		}
	}
	// Fallback: check Related field
	if len(cves) == 0 {
		for _, related := range osv.Related {
			if strings.HasPrefix(related, "CVE-") {
				cves = append(cves, related)
			}
		}
	}
	// Fallback: check ID itself
	if len(cves) == 0 {
		if strings.HasPrefix(osv.ID, "CVE-") {
			cves = append(cves, osv.ID)
		}
	}
	return cves
}

func countTotalRHELCVEs(artifact *RHELArtifactData) int {
	seen := make(map[string]bool)
	for _, vulns := range artifact.Vulnerabilities {
		for _, vuln := range vulns {
			seen[vuln.CVE] = true
		}
	}
	return len(seen)
}

func writeRHELArtifact(path string, artifact *RHELArtifactData) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); err == nil && cerr != nil {
			err = cerr
		}
	}()

	gzWriter := gzip.NewWriter(file)
	defer func() {
		if cerr := gzWriter.Close(); err == nil && cerr != nil {
			err = cerr
		}
	}()

	encoder := json.NewEncoder(gzWriter)

	if err = encoder.Encode(artifact); err != nil {
		return err
	}

	return nil
}
