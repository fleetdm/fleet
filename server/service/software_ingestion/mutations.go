package software_ingestion

import (
	"fmt"
	"regexp"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
)

// applySoftwareMutations applies data sanitization and normalization to software entries
// This extracts the logic from MutateSoftwareOnIngestion in the original codebase
func applySoftwareMutations(software *fleet.Software, logger log.Logger) {
	// Apply basic app sanitizers
	for _, sanitizer := range basicAppSanitizers {
		if sanitizer.matches(software) {
			sanitizer.mutate(software, logger)
		}
	}

	// Add other mutation logic here as needed
}

// softwareSanitizer defines a rule for sanitizing/normalizing software data
type softwareSanitizer struct {
	matchBundleIdentifier string
	matchName             string
	mutate                func(*fleet.Software, log.Logger)
}

func (s *softwareSanitizer) matches(software *fleet.Software) bool {
	if s.matchBundleIdentifier != "" && software.BundleIdentifier != nil {
		return *software.BundleIdentifier == s.matchBundleIdentifier
	}
	if s.matchName != "" {
		return software.Name == s.matchName
	}
	return false
}

// dcvVersionFormat handles DCV Viewer version normalization
var dcvVersionFormat = regexp.MustCompile(`^(\d+\.\d+)\s*\(r(\d+)\)$`)

// basicAppSanitizers contains rules for normalizing common software entries
// This is extracted from the original codebase
var basicAppSanitizers = []softwareSanitizer{
	{
		matchBundleIdentifier: "com.nicesoftware.dcvviewer",
		mutate: func(s *fleet.Software, logger log.Logger) {
			if versionMatches := dcvVersionFormat.FindStringSubmatch(s.Version); len(versionMatches) == 3 {
				s.Version = fmt.Sprintf("%s.%s", versionMatches[1], versionMatches[2])
			}
		},
	},
	// Add more sanitizers as needed
}