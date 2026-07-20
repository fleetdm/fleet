package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/ee/maintained-apps/sigverify"
)

// checkStatus is the outcome of one verification check.
type checkStatus string

const (
	// statusPass: the check ran and matched expectations.
	statusPass checkStatus = "pass"
	// statusFail: the check ran and contradicts the manifest or the pin.
	statusFail checkStatus = "fail"
	// statusWarn: the check ran but the result needs human attention.
	statusWarn checkStatus = "warn"
	// statusRecorded: the check ran with nothing to compare against (no pin /
	// no_check hash); the observed value is recorded for the reviewer.
	statusRecorded checkStatus = "recorded"
	// statusSkipped: the check could not run in this environment (missing
	// tool, wrong OS, uncheckable installer format).
	statusSkipped checkStatus = "skipped"
	// statusError: the check was attempted but errored.
	statusError checkStatus = "error"
)

type checkResult struct {
	Status checkStatus `json:"status"`
	Detail string      `json:"detail,omitempty"`
}

// appVerification is the verification result for one (slug, version) entry.
type appVerification struct {
	Slug          string   `json:"slug"`
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	Platform      string   `json:"platform"`
	InstallerURL  string   `json:"installer_url"`
	ManifestPath  string   `json:"manifest_path"`
	IsNew         bool     `json:"is_new,omitempty"`
	ChangedFields []string `json:"changed_fields,omitempty"`

	ClaimedSHA256  string `json:"claimed_sha256"`
	ObservedSHA256 string `json:"observed_sha256,omitempty"`

	Hash         checkResult `json:"hash"`
	Signature    checkResult `json:"signature"`
	Notarization checkResult `json:"notarization,omitzero"`

	PinPresent bool `json:"pin_present"`

	// Observations available for pin recording (backfill).
	ObservedTeamID      string   `json:"observed_team_id,omitempty"` //nolint:apiparamcheck // "Team ID" is Apple's term for the developer account, not Fleet's teams concept
	ObservedNotarized   bool     `json:"observed_notarized,omitempty"`
	ObservedSubjectCNs  []string `json:"observed_subject_cns,omitempty"`
	ObservedUnsigned    bool     `json:"observed_unsigned,omitempty"`
	SignatureObservable bool     `json:"-"`

	// Failures are hard-fail conditions per the failure policy (enforced in
	// --enforce mode; report-only otherwise). Warnings never fail the run.
	Failures []string `json:"failures,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

func (av *appVerification) fail(format string, args ...any) {
	av.Failures = append(av.Failures, fmt.Sprintf(format, args...))
}

func (av *appVerification) warn(format string, args ...any) {
	av.Warnings = append(av.Warnings, fmt.Sprintf(format, args...))
}

func verifyApp(ctx context.Context, cfg *config, dl *downloader, t targetApp) *appVerification {
	av := &appVerification{
		Slug:          t.Slug,
		Name:          t.Name,
		Version:       t.Manifest.Version,
		Platform:      t.Manifest.Platform(),
		InstallerURL:  t.Manifest.InstallerURL,
		ManifestPath:  t.ManifestPath,
		IsNew:         t.IsNew,
		ChangedFields: t.ChangedFields,
		ClaimedSHA256: t.Manifest.SHA256,
	}

	pin, err := maintained_apps.SignaturePinForSlug(cfg.repoRoot, t.Slug)
	if err != nil {
		av.warn("loading signature pin: %v", err)
	}
	av.PinPresent = pin != nil

	installerPath, observedSHA, err := dl.fetch(ctx, t.Manifest.InstallerURL)
	if err != nil {
		av.Hash = checkResult{Status: statusError, Detail: err.Error()}
		av.Signature = checkResult{Status: statusSkipped, Detail: "installer download failed"}
		av.fail("downloading installer: %v", err)
		return av
	}
	av.ObservedSHA256 = observedSHA

	verifyHash(av)
	verifySignature(ctx, av, installerPath, pin)

	return av
}

// verifyHash implements Layer 1 (hash provenance): the recomputed SHA256 of
// the actual installer bytes must match the hash upstream claims in the
// manifest. no_check apps (unversioned URLs re-released in place) cannot be
// pinned; the observed hash is recorded and the signature pin is their
// primary integrity control.
func verifyHash(av *appVerification) {
	if av.ClaimedSHA256 == "no_check" {
		av.Hash = checkResult{
			Status: statusRecorded,
			Detail: fmt.Sprintf("no_check URL; observed %s", av.ObservedSHA256),
		}
		return
	}
	if !strings.EqualFold(av.ClaimedSHA256, av.ObservedSHA256) {
		av.Hash = checkResult{
			Status: statusFail,
			Detail: fmt.Sprintf("manifest claims %s but downloaded bytes hash to %s", av.ClaimedSHA256, av.ObservedSHA256),
		}
		av.fail("SHA256 mismatch: manifest claims %s, downloaded bytes hash to %s", av.ClaimedSHA256, av.ObservedSHA256)
		return
	}
	av.Hash = checkResult{Status: statusPass, Detail: "recomputed hash matches manifest"}
}

// verifySignature implements the ingest-stage part of Layer 2 (signing
// identity pinning). Windows Authenticode verifies on any OS via
// osslsigncode; macOS signatures verify only when this tool runs on macOS —
// otherwise they are deferred to the validator, which runs the authoritative
// check on a real macOS runner.
func verifySignature(ctx context.Context, av *appVerification, installerPath string, pin *maintained_apps.FMASignature) {
	switch av.Platform {
	case "windows":
		verifyWindowsSignature(ctx, av, installerPath, pin)
	case "darwin":
		if runtime.GOOS != "darwin" {
			av.Signature = checkResult{Status: statusSkipped, Detail: "macOS signature verification requires a macOS host; deferred to validator"}
			return
		}
		verifyDarwinSignature(ctx, av, installerPath, pin)
	default:
		av.Signature = checkResult{Status: statusSkipped, Detail: fmt.Sprintf("unsupported platform %q", av.Platform)}
	}
}

func verifyWindowsSignature(ctx context.Context, av *appVerification, installerPath string, pin *maintained_apps.FMASignature) {
	detailPrefix := ""
	switch ext := strings.ToLower(filepath.Ext(installerPath)); ext {
	case ".exe", ".msi", ".dll", ".cab":
		// osslsigncode-supported Authenticode containers.
	case ".zip":
		// The Authenticode signature lives on the .msi/.exe payload inside
		// the archive; the zip container itself can never be signed.
		dest, err := os.MkdirTemp(filepath.Dir(installerPath), "extract-")
		if err != nil {
			av.Signature = checkResult{Status: statusError, Detail: fmt.Sprintf("creating extraction directory: %v", err)}
			av.warn("extracting zip installer: %v", err)
			return
		}
		defer os.RemoveAll(dest)
		payload, err := sigverify.ExtractZipPayload(installerPath, dest, []string{".msi", ".exe"})
		if err != nil {
			av.Signature = checkResult{Status: statusError, Detail: fmt.Sprintf("extracting zip payload: %v", err)}
			av.warn("extracting zip installer payload: %v", err)
			return
		}
		if payload == "" {
			av.Signature = checkResult{Status: statusSkipped, Detail: "no Authenticode payload (.msi/.exe) found in zip; deferred to validator"}
			return
		}
		detailPrefix = fmt.Sprintf("zip payload %s: ", filepath.Base(payload))
		installerPath = payload
	default:
		av.Signature = checkResult{Status: statusSkipped, Detail: fmt.Sprintf("cannot verify %s Authenticode at ingest; deferred to validator", filepath.Ext(installerPath))}
		return
	}

	res := sigverify.VerifyAuthenticode(ctx, installerPath)
	if !res.Available {
		av.Signature = checkResult{Status: statusSkipped, Detail: "osslsigncode not installed; deferred to validator"}
		return
	}

	av.SignatureObservable = true
	av.ObservedSubjectCNs = res.SubjectCNs
	av.ObservedUnsigned = res.NoSignature

	switch {
	case pin != nil && pin.Unsigned:
		switch {
		case res.NoSignature:
			av.Signature = checkResult{Status: statusPass, Detail: detailPrefix + "unsigned, as pinned (justification: " + pin.Justification + ")"}
		case !res.Verified:
			// A formerly-unsigned installer now carrying a broken or
			// untrusted signature is a tamper indicator, not a vendor
			// starting to sign.
			av.Signature = checkResult{Status: statusFail, Detail: detailPrefix + "pinned unsigned but installer now carries an invalid signature: " + res.Detail}
			av.fail("pin says unsigned but installer now carries an invalid signature: %s", res.Detail)
		default:
			av.Signature = checkResult{Status: statusWarn, Detail: detailPrefix + fmt.Sprintf("pinned unsigned but installer is validly signed by %v; update the pin", res.SubjectCNs)}
			av.warn("installer is now validly signed by %v but the pin says unsigned; update the pin", res.SubjectCNs)
		}
	case pin != nil:
		switch {
		case res.NoSignature:
			av.Signature = checkResult{Status: statusFail, Detail: detailPrefix + fmt.Sprintf("installer has no Authenticode signature but pin expects %v", pin.SubjectCNs)}
			av.fail("installer is unsigned but the pin expects signer %v", pin.SubjectCNs)
		case !res.Verified:
			av.Signature = checkResult{Status: statusFail, Detail: detailPrefix + "Authenticode signature verification failed: " + res.Detail}
			av.fail("Authenticode signature verification failed: %s", res.Detail)
		case !anyCNMatches(pin, res.SubjectCNs):
			av.Signature = checkResult{Status: statusFail, Detail: detailPrefix + fmt.Sprintf("signer %v does not match pinned %v", res.SubjectCNs, pin.SubjectCNs)}
			av.fail("signer identity changed: observed %v, pinned %v", res.SubjectCNs, pin.SubjectCNs)
		default:
			av.Signature = checkResult{Status: statusPass, Detail: detailPrefix + fmt.Sprintf("signed by %v (matches pin)", res.SubjectCNs)}
		}
	default: // no pin
		switch {
		case res.NoSignature:
			av.Signature = checkResult{Status: statusWarn, Detail: detailPrefix + `installer has no Authenticode signature and no "unsigned" pin`}
			av.warn(`installer is unsigned and the app has no "unsigned" signature pin; with a no_check hash this app would have no integrity control`)
		case !res.Verified:
			av.Signature = checkResult{Status: statusFail, Detail: detailPrefix + "Authenticode signature verification failed: " + res.Detail}
			av.fail("Authenticode signature verification failed: %s", res.Detail)
		default:
			av.Signature = checkResult{Status: statusRecorded, Detail: detailPrefix + fmt.Sprintf("signed by %v (no pin yet)", res.SubjectCNs)}
		}
	}
}

func anyCNMatches(pin *maintained_apps.FMASignature, observed []string) bool {
	return slices.ContainsFunc(observed, pin.MatchesSubjectCN)
}

func verifyDarwinSignature(ctx context.Context, av *appVerification, installerPath string, pin *maintained_apps.FMASignature) {
	res, err := sigverify.VerifyDarwinInstaller(ctx, installerPath)
	if err != nil {
		if errors.Is(err, sigverify.ErrSkip) {
			av.Signature = checkResult{Status: statusSkipped, Detail: fmt.Sprintf("cannot verify %s at ingest; deferred to validator", filepath.Ext(installerPath))}
			return
		}
		av.Signature = checkResult{Status: statusError, Detail: err.Error()}
		av.warn("macOS signature verification errored: %v", err)
		return
	}

	av.SignatureObservable = true
	av.ObservedTeamID = res.TeamID
	av.ObservedNotarized = res.Notarized
	av.ObservedUnsigned = res.NoSignature

	switch {
	case pin != nil && pin.Unsigned:
		switch {
		case res.NoSignature:
			av.Signature = checkResult{Status: statusPass, Detail: "unsigned, as pinned (justification: " + pin.Justification + ")"}
		case !res.Verified:
			// A formerly-unsigned installer now carrying a broken or
			// untrusted signature is a tamper indicator, not a vendor
			// starting to sign.
			av.Signature = checkResult{Status: statusFail, Detail: "pinned unsigned but installer now carries an invalid signature: " + res.Detail}
			av.fail("pin says unsigned but installer now carries an invalid signature: %s", res.Detail)
		default:
			av.Signature = checkResult{Status: statusWarn, Detail: fmt.Sprintf("pinned unsigned but installer is validly signed by %s; update the pin", res.Identity)}
			av.warn("installer is now validly signed by %s but the pin says unsigned; update the pin", res.Identity)
		}
	case pin != nil:
		switch {
		case res.NoSignature:
			av.Signature = checkResult{Status: statusFail, Detail: fmt.Sprintf("installer has no signature but pin expects team ID %s", pin.AppleTeamID)}
			av.fail("installer is unsigned but the pin expects team ID %s", pin.AppleTeamID)
		case !res.Verified:
			av.Signature = checkResult{Status: statusFail, Detail: "signature verification failed: " + res.Detail}
			av.fail("macOS signature verification failed: %s", res.Detail)
		case res.TeamID != pin.AppleTeamID:
			av.Signature = checkResult{Status: statusFail, Detail: fmt.Sprintf("signer team ID %s does not match pinned %s", res.TeamID, pin.AppleTeamID)}
			av.fail("signer identity changed: observed team ID %s, pinned %s", res.TeamID, pin.AppleTeamID)
		default:
			av.Signature = checkResult{Status: statusPass, Detail: fmt.Sprintf("signed by %s (matches pin)", res.Identity)}
		}
	default: // no pin
		switch {
		case res.NoSignature:
			av.Signature = checkResult{Status: statusWarn, Detail: `installer has no signature and no "unsigned" pin`}
			av.warn(`installer is unsigned and the app has no "unsigned" signature pin`)
		case !res.Verified:
			av.Signature = checkResult{Status: statusFail, Detail: "signature verification failed: " + res.Detail}
			av.fail("macOS signature verification failed: %s", res.Detail)
		default:
			av.Signature = checkResult{Status: statusRecorded, Detail: fmt.Sprintf("signed by %s (no pin yet)", res.Identity)}
		}
	}

	// Notarization (Layer 3 on macOS: notarization means Apple's automated
	// malware analysis ran on these bytes, and spctl consults XProtect and
	// Apple's revocation service at assessment time).
	if !res.NotarizationChecked {
		av.Notarization = checkResult{Status: statusSkipped, Detail: "notarization not assessable for this format at ingest"}
		return
	}
	switch {
	case pin != nil && pin.Notarized && !res.Notarized:
		av.Notarization = checkResult{Status: statusFail, Detail: "pin expects a notarization ticket but Gatekeeper did not assess the installer as notarized: " + res.NotarizationDetail}
		av.fail("installer is not notarized but the pin expects notarization")
	case pin != nil && pin.Notarized:
		av.Notarization = checkResult{Status: statusPass, Detail: res.NotarizationDetail}
	case res.Notarized:
		av.Notarization = checkResult{Status: statusRecorded, Detail: res.NotarizationDetail + " (no pin yet)"}
	default:
		av.Notarization = checkResult{Status: statusWarn, Detail: "not notarized: " + res.NotarizationDetail}
		av.warn("installer was not assessed as notarized: %s", res.NotarizationDetail)
	}
}
