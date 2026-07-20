package maintained_apps

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// FMASignature pins the expected signing identity for a maintained app's
// installer. Pins live in the app's input JSON (ee/maintained-apps/inputs/**)
// and are verified at ingest time (cmd/maintained-apps/verify) and again in
// the validator (cmd/maintained-apps/validate) before the install script
// runs. Identity pins survive certificate renewals because they pin the
// subject identity (Team ID / subject CN), not the leaf certificate: an
// identity change (vendor rebrand, certificate transfer) fails verification
// and requires a human to update the pin in a reviewed PR.
type FMASignature struct {
	// AppleTeamID is the Apple Developer ID Team ID expected to have signed
	// the macOS installer (e.g. "M683GB7CPW"). Darwin apps only.
	AppleTeamID string `json:"apple_team_id,omitempty"` //nolint:apiparamcheck // "Team ID" is Apple's term for the developer account, not Fleet's teams concept
	// Notarized indicates the macOS installer is expected to carry an Apple
	// notarization ticket. Darwin apps only.
	Notarized bool `json:"notarized,omitempty"`
	// SubjectCNs lists the accepted Authenticode leaf subject CommonNames for
	// the Windows installer (e.g. ["Box, Inc."]). An array because vendors
	// sometimes ship installers signed with more than one certificate.
	// Windows apps only.
	SubjectCNs []string `json:"subject_cns,omitempty"`
	// Unsigned acknowledges that the vendor ships an unsigned installer, so
	// signature verification is expected to find no signature. It must be
	// accompanied by Justification and is mutually exclusive with the
	// identity pins above.
	Unsigned bool `json:"unsigned,omitempty"`
	// Justification documents why an unsigned installer is acceptable for
	// this app. Required when Unsigned is true.
	Justification string `json:"justification,omitempty"`
}

var appleTeamIDPattern = regexp.MustCompile(`^[A-Z0-9]{10}$`)

// Validate checks that the signature pin is internally consistent for an app
// on the given platform ("darwin" or "windows").
func (s *FMASignature) Validate(platform string) error {
	if s.Unsigned {
		if s.Justification == "" {
			return errors.New(`signature pin with "unsigned": true requires a justification`)
		}
		if s.AppleTeamID != "" || s.Notarized || len(s.SubjectCNs) > 0 {
			return errors.New(`signature pin with "unsigned": true cannot also pin a signing identity`)
		}
		return nil
	}

	switch platform {
	case "darwin":
		if len(s.SubjectCNs) > 0 {
			return errors.New(`"subject_cns" is a Windows pin; use "apple_team_id" for darwin apps`)
		}
		if s.AppleTeamID == "" {
			return errors.New(`signature pin for a darwin app must set "apple_team_id" (or "unsigned": true)`)
		}
		if !appleTeamIDPattern.MatchString(s.AppleTeamID) {
			return fmt.Errorf("invalid apple_team_id %q: expected 10 uppercase alphanumeric characters", s.AppleTeamID)
		}
	case "windows":
		if s.AppleTeamID != "" || s.Notarized {
			return errors.New(`"apple_team_id"/"notarized" are darwin pins; use "subject_cns" for windows apps`)
		}
		if len(s.SubjectCNs) == 0 {
			return errors.New(`signature pin for a windows app must set "subject_cns" (or "unsigned": true)`)
		}
		for _, cn := range s.SubjectCNs {
			if strings.TrimSpace(cn) == "" {
				return errors.New(`"subject_cns" entries cannot be empty`)
			}
		}
	default:
		return fmt.Errorf("unknown platform %q for signature pin", platform)
	}

	return nil
}

// MatchesSubjectCN reports whether the observed Authenticode leaf subject
// CommonName matches one of the pinned subject CNs.
func (s *FMASignature) MatchesSubjectCN(observed string) bool {
	return slices.Contains(s.SubjectCNs, observed)
}

// InputFilePathForSlug returns the path of the input JSON file for the given
// slug ("<token>/<platform>") relative to the repo root, e.g.
// "ee/maintained-apps/inputs/homebrew/box-drive.json" for "box-drive/darwin".
func InputFilePathForSlug(slug string) (string, error) {
	parts := strings.Split(slug, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid slug format: %s, expected <name>/<platform>", slug)
	}

	var platformDir string
	switch parts[1] {
	case "darwin":
		platformDir = "homebrew"
	case "windows":
		platformDir = "winget"
	default:
		return "", fmt.Errorf("unknown platform %q in slug %s", parts[1], slug)
	}

	return filepath.Join("ee", "maintained-apps", "inputs", platformDir, parts[0]+".json"), nil
}

// SignaturePinForSlug reads the signature pin for the given slug from the
// app's input JSON under repoRoot. It returns (nil, nil) when the input file
// has no signature block.
func SignaturePinForSlug(repoRoot, slug string) (*FMASignature, error) {
	inputPath, err := InputFilePathForSlug(slug)
	if err != nil {
		return nil, err
	}

	fileBytes, err := os.ReadFile(filepath.Join(repoRoot, inputPath))
	if err != nil {
		return nil, fmt.Errorf("reading app input file: %w", err)
	}

	var input struct {
		Signature *FMASignature `json:"signature"`
	}
	if err := json.Unmarshal(fileBytes, &input); err != nil {
		return nil, fmt.Errorf("unmarshal app input file: %w", err)
	}

	return input.Signature, nil
}
