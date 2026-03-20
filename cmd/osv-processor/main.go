package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
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
	Package           Package                `json:"package"`
	Ranges            []Range                `json:"ranges"`
	Versions          []string               `json:"versions,omitempty"`
	EcosystemSpecific map[string]interface{} `json:"ecosystem_specific,omitempty"`
	DatabaseSpecific  map[string]interface{} `json:"database_specific,omitempty"`
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

type ArtifactData struct {
	SchemaVersion   string                     `json:"schema_version"`
	UbuntuVersion   string                     `json:"ubuntu_version"`
	Generated       string                     `json:"generated"`
	TotalCVEs       int                        `json:"total_cves"`
	TotalPackages   int                        `json:"total_packages"`
	Vulnerabilities map[string][]ProcessedVuln `json:"vulnerabilities"`
}

func main() {
	inputDir := flag.String("input", "/tmp/ubuntu-osv", "Input directory with OSV JSON files")
	outputDir := flag.String("output", "./artifacts", "Output directory for artifacts")
	versions := flag.String("versions", "", "Comma-separated Ubuntu versions to process (inclusive)")
	excludeVersions := flag.String("exclude-versions", "", "Comma-separated Ubuntu versions to exclude (ignored if --versions is set)")
	changedFilesToday := flag.String("changed-files-today", "", "Path to file containing CVE files changed today (generates today's deltas)")
	changedFilesYesterday := flag.String("changed-files-yesterday", "", "Path to file containing CVE files changed yesterday (generates yesterday's deltas)")
	flag.Parse()

	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Build version filter
	targetVersions, excludedVersions := buildVersionFilter(*versions, *excludeVersions)
	switch {
	case targetVersions != nil:
		log.Printf("Processing OSV files from %s for versions: %s", *inputDir, *versions)
	case excludedVersions != nil:
		log.Printf("Processing OSV files from %s (auto-detecting, excluding: %s)", *inputDir, *excludeVersions)
	default:
		log.Printf("Processing OSV files from %s (auto-detecting all versions)", *inputDir)
	}

	// Load changed CVE files for delta generation
	var todayCVEFiles, yesterdayCVEFiles map[string]bool
	generateTodayDeltas := *changedFilesToday != ""
	generateYesterdayDeltas := *changedFilesYesterday != ""

	if generateTodayDeltas {
		log.Printf("Loading today's changed CVE files from %s", *changedFilesToday)
		var err error
		todayCVEFiles, err = loadChangedFiles(*changedFilesToday)
		if err != nil {
			log.Fatalf("Failed to load today's changed files: %v", err)
		}
		log.Printf("Found %d CVE files changed today", len(todayCVEFiles))
	}

	if generateYesterdayDeltas {
		log.Printf("Loading yesterday's changed CVE files from %s", *changedFilesYesterday)
		var err error
		yesterdayCVEFiles, err = loadChangedFiles(*changedFilesYesterday)
		if err != nil {
			log.Fatalf("Failed to load yesterday's changed files: %v", err)
		}
		log.Printf("Found %d CVE files changed yesterday", len(yesterdayCVEFiles))
	}

	artifacts := make(map[string]*ArtifactData)
	todayArtifacts := make(map[string]*ArtifactData)
	yesterdayArtifacts := make(map[string]*ArtifactData)

	filesProcessed := 0
	filesSkipped := 0

	err := filepath.Walk(*inputDir, func(path string, info os.FileInfo, err error) error {
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
		if generateTodayDeltas || generateYesterdayDeltas {
			relPath, err := filepath.Rel(*inputDir, path)
			if err == nil {
				fullRelPath := filepath.Join("osv/cve", relPath)
				if generateTodayDeltas {
					inToday = todayCVEFiles[fullRelPath]
				}
				if generateYesterdayDeltas {
					inYesterday = yesterdayCVEFiles[fullRelPath]
				}
			}
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
						Generated:       time.Now().UTC().Format(time.RFC3339),
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
							Generated:       time.Now().UTC().Format(time.RFC3339),
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
							Generated:       time.Now().UTC().Format(time.RFC3339),
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
		log.Fatalf("Error walking directory: %v", err)
	}

	log.Printf("Processed %d files, skipped %d files", filesProcessed, filesSkipped)
	log.Printf("Discovered %d Ubuntu versions", len(artifacts))

	// Write full artifacts
	for ver, artifact := range artifacts {
		artifact.TotalCVEs = countTotalCVEs(artifact)
		artifact.TotalPackages = len(artifact.Vulnerabilities)

		outputFile := filepath.Join(*outputDir, fmt.Sprintf("osv-ubuntu-%s-%s.json.gz",
			strings.Replace(ver, ".", "", -1),
			time.Now().Format("2006-01-02")))

		if err := writeArtifact(outputFile, artifact); err != nil {
			log.Fatalf("Failed to write artifact for Ubuntu %s: %v", ver, err)
		}

		log.Printf("Ubuntu %s: %d packages, %d CVEs -> %s",
			ver, artifact.TotalPackages, artifact.TotalCVEs, outputFile)
	}

	// Write delta artifacts (if any were generated)
	if generateTodayDeltas && len(todayArtifacts) > 0 {
		today := time.Now().UTC().Format("2006-01-02")
		log.Printf("\nWriting today's delta artifacts (%s)...", today)
		for ver, artifact := range todayArtifacts {
			artifact.TotalCVEs = countTotalCVEs(artifact)
			artifact.TotalPackages = len(artifact.Vulnerabilities)

			outputFile := filepath.Join(*outputDir, fmt.Sprintf("osv-ubuntu-%s-delta-%s.json.gz",
				strings.Replace(ver, ".", "", -1), today))

			if err := writeArtifact(outputFile, artifact); err != nil {
				log.Fatalf("Failed to write today's delta for Ubuntu %s: %v", ver, err)
			}

			log.Printf("Ubuntu %s (today): %d packages, %d CVEs -> %s",
				ver, artifact.TotalPackages, artifact.TotalCVEs, outputFile)
		}
	}

	if generateYesterdayDeltas && len(yesterdayArtifacts) > 0 {
		yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
		log.Printf("\nWriting yesterday's delta artifacts (%s)...", yesterday)
		for ver, artifact := range yesterdayArtifacts {
			artifact.TotalCVEs = countTotalCVEs(artifact)
			artifact.TotalPackages = len(artifact.Vulnerabilities)

			outputFile := filepath.Join(*outputDir, fmt.Sprintf("osv-ubuntu-%s-delta-%s.json.gz",
				strings.Replace(ver, ".", "", -1), yesterday))

			if err := writeArtifact(outputFile, artifact); err != nil {
				log.Fatalf("Failed to write yesterday's delta for Ubuntu %s: %v", ver, err)
			}

			log.Printf("Ubuntu %s (yesterday): %d packages, %d CVEs -> %s",
				ver, artifact.TotalPackages, artifact.TotalCVEs, outputFile)
		}
	}
}

func buildVersionFilter(versions, excludeVersions string) (targetVersions, excludedVersions map[string]bool) {
	if versions != "" {
		// Inclusive mode: only process specified versions
		targetVersions = make(map[string]bool)
		for _, ver := range strings.Split(versions, ",") {
			targetVersions[strings.TrimSpace(ver)] = true
		}
		return targetVersions, nil
	}

	if excludeVersions != "" {
		// Exclusive mode: process all except specified versions
		excludedVersions = make(map[string]bool)
		for _, ver := range strings.Split(excludeVersions, ",") {
			excludedVersions[strings.TrimSpace(ver)] = true
		}
		return nil, excludedVersions
	}

	// Auto-detect all versions
	return nil, nil
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
	parts := strings.Split(ecosystem, ":")
	for _, part := range parts {
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

func writeArtifact(path string, artifact *ArtifactData) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	encoder := json.NewEncoder(gzWriter)
	encoder.SetIndent("", "  ")

	return encoder.Encode(artifact)
}

func loadChangedFiles(changedFilesPath string) (map[string]bool, error) {
	file, err := os.Open(changedFilesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open changed files list: %w", err)
	}
	defer file.Close()

	changedFiles := make(map[string]bool)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		changedFiles[line] = true
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading changed files: %w", err)
	}

	return changedFiles, nil
}
