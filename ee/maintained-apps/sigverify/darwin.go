package sigverify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ErrSkip is returned when a check cannot run for this installer format in
// this environment (e.g. a Windows-only archive format at ingest time).
var ErrSkip = errors.New("signature check skipped")

// Per-check timeouts so one hung subprocess (e.g. an hdiutil attach waiting
// on a license prompt, or spctl's network revocation check stalling) fails
// that app's check instead of hanging the whole run.
const (
	// commandTimeout bounds single-command checks (codesign, pkgutil, spctl,
	// osslsigncode).
	commandTimeout = 2 * time.Minute
	// containerTimeout bounds checks that first mount or extract a container
	// (dmg attach, zip extraction) before verifying its payload.
	containerTimeout = 5 * time.Minute
)

// DarwinResult is the outcome of macOS installer signature and notarization
// checks (pkgutil / codesign / spctl). These commands only exist on macOS;
// callers must gate on runtime.GOOS.
type DarwinResult struct {
	Verified    bool
	NoSignature bool
	// TeamID is the Apple Developer ID Team ID of the signer (e.g. "M683GB7CPW").
	TeamID string
	// Identity is the human-readable signing identity, e.g.
	// "Developer ID Installer: Box, Inc. (M683GB7CPW)".
	Identity string
	// NotarizationChecked is true when a Gatekeeper/notary assessment ran for
	// this installer format.
	NotarizationChecked bool
	Notarized           bool
	NotarizationDetail  string
	// Detail carries a short failure description for reporting.
	Detail string
}

// VerifyDarwinInstaller dispatches on the installer format. pkg files carry
// their own signature; for dmg and zip the meaningful signature usually lives
// on the payload (.app or .pkg) inside the container, so the container is
// mounted/extracted and the payload verified — the same sequence a human
// reviewer would run by hand. Returns ErrSkip for formats that cannot be
// checked.
func VerifyDarwinInstaller(ctx context.Context, installerPath string) (*DarwinResult, error) {
	switch strings.ToLower(filepath.Ext(installerPath)) {
	case ".pkg", ".mpkg":
		return VerifyPkgSignature(ctx, installerPath)
	case ".dmg":
		return VerifyDmgSignature(ctx, installerPath)
	case ".zip":
		return VerifyZipPayload(ctx, installerPath)
	default:
		return nil, ErrSkip
	}
}

// VerifyPkgSignature checks a pkg installer's signature (pkgutil) and its
// Gatekeeper install assessment (spctl), which requires notarization to be
// accepted.
func VerifyPkgSignature(ctx context.Context, pkgPath string) (*DarwinResult, error) {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "pkgutil", "--check-signature", pkgPath).CombinedOutput()
	res := ParsePkgutilOutput(string(out))
	if err != nil && !res.NoSignature {
		// pkgutil exits non-zero for unsigned packages (a result, not an
		// error); anything else unexpected is an error.
		if res.Detail == "" {
			return nil, fmt.Errorf("pkgutil --check-signature: %w: %s", err, strings.TrimSpace(string(out)))
		}
	}

	assessWithSpctl(ctx, res, "install", nil, pkgPath)

	return res, nil
}

// VerifyDmgSignature checks a dmg installer: the image's own signature when
// present, otherwise the payload (.app or .pkg) inside it.
func VerifyDmgSignature(ctx context.Context, dmgPath string) (*DarwinResult, error) {
	ctx, cancel := context.WithTimeout(ctx, containerTimeout)
	defer cancel()

	// If the disk image itself is signed, assess it directly: accepted +
	// "Notarized Developer ID" source means Apple's notary service vouches
	// for these exact bytes.
	container := VerifyCodeObject(ctx, dmgPath)
	if !container.NoSignature {
		assessWithSpctl(ctx, container, "open", []string{"--context", "context:primary-signature"}, dmgPath)
		return container, nil
	}

	// Unsigned dmg container (common: vendors sign the .app inside, not the
	// image): mount it read-only and verify the payload — the same files the
	// install script will copy to /Applications.
	mountPoint, err := attachDMG(ctx, dmgPath)
	if err != nil {
		return nil, fmt.Errorf("attaching dmg to verify payload: %w", err)
	}
	defer func() {
		// Best effort, but surface failures: a --all backfill run mounts
		// dozens of images, and silently leaked mounts add up.
		if out, err := exec.Command("hdiutil", "detach", mountPoint, "-force").CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to detach %s: %v: %s\n", mountPoint, err, strings.TrimSpace(string(out)))
		}
	}()

	res, err := verifyPayloadIn(ctx, mountPoint)
	if err != nil {
		return nil, err
	}
	res.Detail = strings.TrimSpace("dmg container unsigned; verified payload. " + res.Detail)
	return res, nil
}

// VerifyZipPayload extracts a zip installer and verifies the .app (or .pkg)
// payload inside; zip archives themselves cannot carry a code signature.
func VerifyZipPayload(ctx context.Context, zipPath string) (*DarwinResult, error) {
	ctx, cancel := context.WithTimeout(ctx, containerTimeout)
	defer cancel()

	dest, err := os.MkdirTemp(filepath.Dir(zipPath), "extract-")
	if err != nil {
		return nil, fmt.Errorf("creating extraction directory: %w", err)
	}
	defer os.RemoveAll(dest)

	// ditto preserves resource forks and extended attributes, which codesign
	// verification of app bundles depends on.
	if out, err := exec.CommandContext(ctx, "ditto", "-x", "-k", zipPath, dest).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("extracting zip: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return verifyPayloadIn(ctx, dest)
}

// verifyPayloadIn locates the installer payload (.app bundle or .pkg) at the
// top of dir (or one level down) and verifies it.
func verifyPayloadIn(ctx context.Context, dir string) (*DarwinResult, error) {
	if appPath := findPayload(dir, ".app"); appPath != "" {
		return VerifyAppBundle(ctx, appPath)
	}
	if pkgPath := findPayload(dir, ".pkg"); pkgPath != "" {
		return VerifyPkgSignature(ctx, pkgPath)
	}
	return nil, errors.New("no .app or .pkg payload found in installer container")
}

func findPayload(dir, ext string) string {
	for _, pattern := range []string{"*" + ext, "*/*" + ext} {
		matches, _ := filepath.Glob(filepath.Join(dir, pattern))
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

// VerifyAppBundle verifies an .app bundle's code signature and Gatekeeper
// assessment — the check the validator runs on the installed app before the
// quarantine exception is added.
func VerifyAppBundle(ctx context.Context, appPath string) (*DarwinResult, error) {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	res := VerifyCodeObject(ctx, appPath)

	assessWithSpctl(ctx, res, "execute", nil, appPath)

	return res, nil
}

// VerifyCodeObject runs codesign verification against any code object (app
// bundle or disk image) and extracts the signing identity.
func VerifyCodeObject(ctx context.Context, path string) *DarwinResult {
	res := &DarwinResult{}

	verifyOut, verifyErr := exec.CommandContext(ctx, "codesign", "--verify", "--deep", "--strict", path).CombinedOutput()
	if verifyErr != nil {
		if strings.Contains(string(verifyOut), "not signed at all") {
			res.NoSignature = true
			res.Detail = "no code signature"
			return res
		}
		res.Detail = strings.TrimSpace(string(verifyOut))
	} else {
		res.Verified = true
	}

	infoOut, _ := exec.CommandContext(ctx, "codesign", "-dvv", path).CombinedOutput()
	identity, teamID := ParseCodesignInfo(string(infoOut))
	res.Identity = identity
	res.TeamID = teamID

	return res
}

// attachDMG mounts a disk image read-only without Finder side effects and
// returns the mount point.
func attachDMG(ctx context.Context, dmgPath string) (string, error) {
	cmd := exec.CommandContext(ctx, "hdiutil", "attach", dmgPath, "-nobrowse", "-readonly", "-noautoopen", "-plist")
	// Some disk images embed a license agreement that prompts on attach;
	// answering Y keeps the run non-interactive.
	cmd.Stdin = strings.NewReader("Y\n")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("hdiutil attach: %w", err)
	}
	mountPoint := parseHdiutilMountPoint(string(out))
	if mountPoint == "" {
		return "", errors.New("hdiutil attach reported no mount point")
	}
	return mountPoint, nil
}

var mountPointPattern = regexp.MustCompile(`<key>mount-point</key>\s*<string>([^<]+)</string>`)

func parseHdiutilMountPoint(plistOut string) string {
	if m := mountPointPattern.FindStringSubmatch(plistOut); m != nil {
		return m[1]
	}
	return ""
}

var (
	// developerIDPattern matches certificate-chain leaf lines from pkgutil, e.g.:
	//	1. Developer ID Installer: Box, Inc. (M683GB7CPW)
	developerIDPattern = regexp.MustCompile(`\d+\.\s+((?:Developer ID|3rd Party Mac Developer|Apple) [^\n]*?)\s*(?:\(([0-9A-Z]{10})\))?\s*$`)
	// teamIdentifierPattern matches codesign -dvv output, e.g.:
	//	TeamIdentifier=XSYZ3E4B7D
	teamIdentifierPattern = regexp.MustCompile(`(?m)^TeamIdentifier=([0-9A-Z]{10})$`)
	// authorityPattern matches the leaf Authority line from codesign -dvv, e.g.:
	//	Authority=Developer ID Application: Ryan Hanson (XSYZ3E4B7D)
	authorityPattern = regexp.MustCompile(`(?m)^Authority=(.+)$`)
)

// ParsePkgutilOutput parses `pkgutil --check-signature` output.
func ParsePkgutilOutput(out string) *DarwinResult {
	res := &DarwinResult{}

	switch {
	case strings.Contains(out, "Status: no signature"):
		res.NoSignature = true
		res.Detail = "no signature"
		return res
	case strings.Contains(out, "Status: signed"):
		res.Verified = true
	default:
		res.Detail = firstLineContaining(out, "Status:")
		if res.Detail == "" {
			res.Detail = "pkgutil did not report a signature status"
		}
		return res
	}

	// Untrusted or revoked chains are reported as signed but flagged; treat
	// anything other than a clean trusted status as unverified.
	if strings.Contains(out, "untrusted") || strings.Contains(out, "revoked") || strings.Contains(out, "expired") {
		res.Verified = false
		res.Detail = firstLineContaining(out, "Status:")
	}

	// The first certificate-chain entry is the leaf (signing) certificate.
	for line := range strings.Lines(out) {
		if m := developerIDPattern.FindStringSubmatch(line); m != nil {
			res.Identity = strings.TrimSpace(m[1])
			if len(m) > 2 {
				res.TeamID = m[2]
			}
			if res.TeamID != "" {
				res.Identity = fmt.Sprintf("%s (%s)", res.Identity, res.TeamID)
			}
			break
		}
	}

	return res
}

// ParseCodesignInfo extracts the leaf signing identity and team ID from
// codesign -dvv output.
func ParseCodesignInfo(out string) (identity, teamID string) {
	if m := teamIdentifierPattern.FindStringSubmatch(out); m != nil {
		teamID = m[1]
	}
	if m := authorityPattern.FindStringSubmatch(out); m != nil {
		identity = strings.TrimSpace(m[1])
	}
	return identity, teamID
}

// SpctlAssessment is the parsed result of an `spctl --assess -vv` run.
type SpctlAssessment struct {
	// Assessed is true when spctl reached a verdict (accepted or rejected).
	// False means spctl never ran the assessment — a rejection is a result,
	// but no verdict at all must not be reported as one.
	Assessed bool
	Accepted bool
	Source   string
	Origin   string
}

// Summary renders the assessment for report output.
func (a *SpctlAssessment) Summary() string {
	verdict := "rejected"
	if a.Accepted {
		verdict = "accepted"
	}
	if a.Source == "" {
		return verdict
	}
	return fmt.Sprintf("%s; source=%s", verdict, a.Source)
}

// ParseSpctlOutput parses spctl --assess -vv output, e.g.:
//
//	/tmp/BoxDrive.pkg: accepted
//	source=Notarized Developer ID
//	origin=Developer ID Installer: Box, Inc. (M683GB7CPW)
func ParseSpctlOutput(out string) *SpctlAssessment {
	assess := &SpctlAssessment{}
	for line := range strings.Lines(out) {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasSuffix(trimmed, ": accepted"):
			assess.Accepted = true
			assess.Assessed = true
		case strings.Contains(trimmed, ": rejected"):
			assess.Assessed = true
		case strings.HasPrefix(trimmed, "source="):
			assess.Source = strings.TrimPrefix(trimmed, "source=")
		case strings.HasPrefix(trimmed, "origin="):
			assess.Origin = strings.TrimPrefix(trimmed, "origin=")
		}
	}
	return assess
}

// assessWithSpctl runs `spctl --assess -vv --type <assessType>` on path and
// applies the outcome to res. spctl exits non-zero for rejected assessments —
// that's a verdict, not an error — but when it produces no verdict at all
// (binary missing, timeout, crash) the assessment is marked as not run, so
// callers report "could not assess" instead of a false "not notarized".
func assessWithSpctl(ctx context.Context, res *DarwinResult, assessType string, extraArgs []string, path string) {
	args := append([]string{"--assess", "-vv", "--type", assessType}, extraArgs...)
	args = append(args, path)
	out, err := exec.CommandContext(ctx, "spctl", args...).CombinedOutput()
	assess := ParseSpctlOutput(string(out))
	if !assess.Assessed {
		res.NotarizationChecked = false
		res.NotarizationDetail = fmt.Sprintf("spctl --assess produced no verdict (err: %v): %s", err, strings.TrimSpace(string(out)))
		return
	}
	res.NotarizationChecked = true
	res.Notarized = assess.Accepted && strings.Contains(assess.Source, "Notarized")
	res.NotarizationDetail = assess.Summary()
}
