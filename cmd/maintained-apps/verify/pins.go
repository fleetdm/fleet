package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// recordPin writes the observed signing identity into the app's input JSON as
// a signature pin (trust-on-first-use backfill). It returns false without
// error when there is nothing to record: the input already has a pin, the
// signature wasn't observable in this environment, or the installer is
// unsigned (an "unsigned" pin needs a human-written justification, so it is
// never auto-recorded).
func recordPin(repoRoot string, av *appVerification) (bool, error) {
	if av.PinPresent || !av.SignatureObservable || av.ObservedUnsigned {
		return false, nil
	}

	sig := &maintained_apps.FMASignature{}
	switch av.Platform {
	case "darwin":
		if av.ObservedTeamID == "" {
			return false, nil
		}
		sig.AppleTeamID = av.ObservedTeamID
		sig.Notarized = av.ObservedNotarized
	case "windows":
		if len(av.ObservedSubjectCNs) == 0 {
			return false, nil
		}
		sig.SubjectCNs = av.ObservedSubjectCNs
	default:
		return false, nil
	}
	if err := sig.Validate(av.Platform); err != nil {
		return false, fmt.Errorf("observed identity does not form a valid pin: %w", err)
	}

	inputRelPath, err := maintained_apps.InputFilePathForSlug(av.Slug)
	if err != nil {
		return false, err
	}
	inputPath := filepath.Join(repoRoot, inputRelPath)

	content, err := os.ReadFile(inputPath)
	if err != nil {
		return false, fmt.Errorf("reading input file: %w", err)
	}

	updated, err := insertSignatureBlock(content, sig)
	if err != nil {
		return false, fmt.Errorf("inserting signature block into %s: %w", inputRelPath, err)
	}

	if err := os.WriteFile(inputPath, updated, 0o644); err != nil {
		return false, fmt.Errorf("writing input file: %w", err)
	}
	return true, nil
}

// insertSignatureBlock adds a "signature" property to the end of a JSON
// object document, preserving the file's existing formatting and key order
// (input JSONs are hand-maintained, so a full re-marshal would churn them).
func insertSignatureBlock(content []byte, sig *maintained_apps.FMASignature) ([]byte, error) {
	// Refuse to double-insert.
	var existing struct {
		Signature *maintained_apps.FMASignature `json:"signature"`
	}
	if err := json.Unmarshal(content, &existing); err != nil {
		return nil, fmt.Errorf("input file is not valid JSON: %w", err)
	}
	if existing.Signature != nil {
		return nil, errors.New("input file already has a signature block")
	}

	closeIdx := bytes.LastIndexByte(content, '}')
	if closeIdx < 0 {
		return nil, errors.New("input file has no closing brace")
	}

	prefix := bytes.TrimRight(content[:closeIdx], " \t\n\r")
	if !bytes.HasSuffix(prefix, []byte("{")) {
		prefix = append(prefix, ',')
	}

	sigJSON, err := json.MarshalIndent(sig, "  ", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling signature block: %w", err)
	}

	var buf bytes.Buffer
	buf.Write(prefix)
	buf.WriteString("\n  \"signature\": ")
	buf.Write(sigJSON)
	buf.WriteString("\n}\n")

	// Sanity-check the result still parses.
	var check map[string]any
	if err := json.Unmarshal(buf.Bytes(), &check); err != nil {
		return nil, fmt.Errorf("updated input file is not valid JSON: %w", err)
	}
	return buf.Bytes(), nil
}
