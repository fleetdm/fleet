// Package scep drives a local SCEP server for QA: it builds the in-repo
// scepserver binary (Fleet's fork of micromdm/scep) once and caches it, reads a
// depot's CA identity (thumbprint / issuer DN), initializes new CA depots, and
// builds the launch args the process engine runs. One shared binary serves many
// depot-based profiles (see internal/settings ScepProfile), so several CAs can
// run side by side and expose multiple Custom SCEP CAs to Fleet at once.
//
// Pure helpers (arg builders, depot parsing, proc/channel ids) take explicit
// inputs so they're unit-testable; the service layer resolves real paths and
// the process manager.
package scep

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // SHA-1 is the standard X.509 cert thumbprint format (Windows CAThumbprint / SCEP profiles expect it).
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
	"github.com/fleetdm/fleet/tools/hangar/internal/shellpath"
)

// scepserverPkg is the in-repo main package built into the cached binary. It's
// a fork of micromdm/scep with the same CLI (`ca -init`, serve flags) and the
// same file-depot format, so it reads depots created by either binary.
const scepserverPkg = "./server/mdm/scep/cmd/scepserver"

// ProcID is the process-engine id for a profile's running server. Namespaced
// under "scep" so it never collides with fleet-serve / docker ids.
func ProcID(profileID string) string { return "scep:" + profileID }

// LogChannel is the structured-log ring + on-disk file for a profile's server,
// mirroring the fleet-serve-<id> scheme.
func LogChannel(profileID string) string { return "scep-" + profileID }

// BinaryInfo describes the cached scepserver binary.
type BinaryInfo struct {
	Path    string `json:"path"`
	Exists  bool   `json:"exists"`
	BuiltAt string `json:"built_at"` // RFC3339 mtime, "" when absent
}

// DepotInfo is a depot's CA identity, read from <depot>/ca.pem. Exists is false
// (with Error set on a real failure) when the depot has no parseable CA yet —
// the caller should prompt the user to Init a CA.
type DepotInfo struct {
	DepotPath  string `json:"depot_path"`
	Exists     bool   `json:"exists"`
	Thumbprint string `json:"thumbprint"` // SHA-1, uppercase hex, no separators
	IssuerDN   string `json:"issuer_dn"`
	SubjectDN  string `json:"subject_dn"`
	NotAfter   string `json:"not_after"` // RFC3339
	Error      string `json:"error"`
}

// InitCAParams are the DN + key inputs for `scepserver ca -init`.
type InitCAParams struct {
	CommonName         string `json:"common_name"`
	Organization       string `json:"organization"`
	OrganizationalUnit string `json:"organizational_unit"`
	Country            string `json:"country"`
	KeySize            int    `json:"key_size"`
	Years              int    `json:"years"`
	KeyPassword        string `json:"key_password"`
}

// ServeArgs builds the scepserver launch args for a profile against an
// already-resolved depot path. Pure; unit-tested.
func ServeArgs(depot string, p settings.ScepProfile) []string {
	args := []string{
		"-depot", depot,
		"-port", strconv.Itoa(int(p.Port)),
		"-allowrenew", strconv.Itoa(p.AllowRenew),
	}
	if p.Challenge != "" {
		args = append(args, "-challenge", p.Challenge)
	}
	if p.Debug {
		args = append(args, "-debug")
	}
	if extra := strings.Fields(p.ExtraFlags); len(extra) > 0 {
		args = append(args, extra...)
	}
	return args
}

// InitCAArgs builds the `ca -init` args against a resolved depot path. Pure;
// unit-tested. KeySize/Years fall back to the binary's defaults (4096 / 10)
// when non-positive; -key-password is only passed when set.
func InitCAArgs(depot string, params InitCAParams) []string {
	keySize := params.KeySize
	if keySize <= 0 {
		keySize = 4096
	}
	years := params.Years
	if years <= 0 {
		years = 10
	}
	args := []string{
		"ca", "-init",
		"-depot", depot,
		"-common_name", params.CommonName,
		"-organization", params.Organization,
		"-organizational_unit", params.OrganizationalUnit,
		"-country", params.Country,
		"-keySize", strconv.Itoa(keySize),
		"-years", strconv.Itoa(years),
	}
	if params.KeyPassword != "" {
		args = append(args, "-key-password", params.KeyPassword)
	}
	return args
}

// ParseDepot reads <depot>/ca.pem and returns its CA identity. A missing ca.pem
// yields Exists=false with no Error (depot simply isn't initialized yet); a
// read/parse failure sets Error.
func ParseDepot(depotPath string) DepotInfo {
	info := DepotInfo{DepotPath: depotPath}
	caPath := filepath.Join(depotPath, "ca.pem")
	raw, err := os.ReadFile(caPath)
	if errors.Is(err, os.ErrNotExist) {
		return info // not initialized yet — not an error
	}
	if err != nil {
		info.Error = fmt.Sprintf("read ca.pem: %v", err)
		return info
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		info.Error = "ca.pem is not valid PEM"
		return info
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		info.Error = fmt.Sprintf("parse ca.pem: %v", err)
		return info
	}
	info.Exists = true
	info.Thumbprint = thumbprint(cert.Raw)
	info.IssuerDN = cert.Issuer.String()
	info.SubjectDN = cert.Subject.String()
	info.NotAfter = cert.NotAfter.UTC().Format(time.RFC3339)
	return info
}

// thumbprint is the SHA-1 fingerprint of a cert's DER bytes, uppercase hex with
// no separators — the form Windows CAThumbprint / SCEP profiles use.
func thumbprint(der []byte) string {
	sum := sha1.Sum(der) //nolint:gosec // standard cert thumbprint, not a security primitive here
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// BinaryPath is the cached scepserver location under the data dir.
func BinaryPath(dataDir string) string {
	return filepath.Join(dataDir, "bin", "scepserver")
}

// StatBinary reports the cached binary's presence and build time without
// building it.
func StatBinary(dataDir string) BinaryInfo {
	p := BinaryPath(dataDir)
	info := BinaryInfo{Path: p}
	if st, err := os.Stat(p); err == nil {
		info.Exists = true
		info.BuiltAt = st.ModTime().UTC().Format(time.RFC3339)
	}
	return info
}

// BuildBinary compiles the in-repo scepserver from repoPath into the cached
// location using the login-shell PATH (so `go` resolves in a .app launch). It
// overwrites any existing binary.
func BuildBinary(ctx context.Context, repoPath, dataDir string) (BinaryInfo, error) {
	if repoPath == "" {
		return BinaryInfo{}, errors.New("no primary repo configured — set server 1's repo first")
	}
	binPath := BinaryPath(dataDir)
	if err := os.MkdirAll(filepath.Dir(binPath), 0o755); err != nil {
		return BinaryInfo{}, fmt.Errorf("create bin dir: %w", err)
	}
	cmd := shellpath.CommandContext(ctx, "go", "build", "-o", binPath, scepserverPkg)
	cmd.Dir = repoPath
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return BinaryInfo{}, fmt.Errorf("go build scepserver: %s", msg)
	}
	return StatBinary(dataDir), nil
}

// InitCA runs `scepserver ca -init` (binPath) against depot to create a fresh
// CA. The binary refuses to overwrite an existing CA, so callers should check
// the depot is empty first for a friendlier message.
func InitCA(ctx context.Context, binPath, depot string, params InitCAParams) error {
	cmd := shellpath.CommandContext(ctx, binPath, InitCAArgs(depot, params)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("ca -init: %s", msg)
	}
	return nil
}

// LanIP returns the host's primary outbound IPv4 (what Fleet SCEP URLs should
// use), matching `ipconfig getifaddr en0` in practice. Empty on failure.
func LanIP() string {
	// A UDP "connect" picks the source IP via the routing table without sending
	// any packets.
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return addr.IP.String()
	}
	return ""
}
