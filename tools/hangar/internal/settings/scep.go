package settings

import (
	"fmt"
	"path/filepath"
)

// ScepProfile is one saved SCEP server launch configuration. The in-repo
// scepserver binary (a fork of micromdm/scep) is shared across profiles; each
// profile differs by its depot (CA), port, and challenge, so several can run
// side by side and expose multiple Custom SCEP CAs to Fleet at once. Profiles
// are the reusable ("starred") configs — every profile is persisted, so the
// saved list *is* the set of reusable selections.
type ScepProfile struct {
	ID   string `json:"id"`   // stable, e.g. "scep1" (never reused/renumbered)
	Name string `json:"name"` // user-facing label, e.g. "windows" / "temp"
	// DepotPath is the CA folder (ca.pem/ca.key/index.txt/serial/...). Empty
	// means the managed default <ScepDepotsDir>/<ID>; see ResolveDepotPath.
	DepotPath string `json:"depot_path"`
	Port      uint16 `json:"port"`
	Challenge string `json:"challenge"`
	// AllowRenew is the number of days before expiry a renewal is allowed
	// (scepserver -allowrenew). 0 = always allow.
	AllowRenew int  `json:"allow_renew"`
	Debug      bool `json:"debug"` // scepserver -debug
	// ExtraFlags are additional scepserver args, whitespace-separated, for
	// anything not covered by the fields above.
	ExtraFlags string `json:"extra_flags"`
}

// defaultScepStartPort is where auto-assigned SCEP ports begin. Matches the
// port block Andrey's existing QA CAs use (2016/2017/2018).
const defaultScepStartPort = 2016

// defaultScepProfile builds a fresh profile for the i-th slot: a unique id, the
// next free port starting at 2016, and the challenge/renew defaults from
// Andrey's QA setup (challenge "secret", -allowrenew 0, -debug on).
func defaultScepProfile(existing []ScepProfile, i int) ScepProfile {
	return ScepProfile{
		ID:         fmt.Sprintf("scep%d", i+1),
		Name:       fmt.Sprintf("scep %d", i+1),
		Port:       nextFreeScepPort(existing),
		Challenge:  "secret",
		AllowRenew: 0,
		Debug:      true,
	}
}

// NextScepProfile returns a fresh profile whose id and port don't collide with
// any existing one. Unlike servers there's no hard cap — the user decides how
// many CAs to run concurrently.
func NextScepProfile(existing []ScepProfile) ScepProfile {
	i := len(existing)
	// Bump past any id already in use (e.g. a middle profile was removed and
	// re-added) so ids are never reused.
	for hasScepID(existing, fmt.Sprintf("scep%d", i+1)) {
		i++
	}
	return defaultScepProfile(existing, i)
}

func hasScepID(profiles []ScepProfile, id string) bool {
	for _, p := range profiles {
		if p.ID == id {
			return true
		}
	}
	return false
}

// nextFreeScepPort returns the lowest port at/above defaultScepStartPort not
// already claimed by an existing profile.
func nextFreeScepPort(existing []ScepProfile) uint16 {
	used := make(map[uint16]struct{}, len(existing))
	for _, p := range existing {
		used[p.Port] = struct{}{}
	}
	port := uint16(defaultScepStartPort)
	for {
		if _, taken := used[port]; !taken {
			return port
		}
		port++
	}
}

// ResolveDepotPath returns the profile's depot directory: its explicit
// DepotPath when set, otherwise the managed default <depotsDir>/<ID>.
func ResolveDepotPath(depotsDir string, p ScepProfile) string {
	if p.DepotPath != "" {
		return p.DepotPath
	}
	return filepath.Join(depotsDir, p.ID)
}
