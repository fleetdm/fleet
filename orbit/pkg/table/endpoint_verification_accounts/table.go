// Package endpoint_verification_accounts implements the osquery virtual table
// `endpoint_verification_accounts`, which surfaces every signed-in Google
// Workspace identity on the device that's bound to a Cloud Identity
// deviceUser via Google's Endpoint Verification (EV) Chrome extension and
// native helper.
//
// The table is the primary resolution mechanism for Fleet's Cloud Identity
// ClientState integration: each row provides the
// `resource_id` (rawResourceId) and `email` Fleet needs to PATCH per-device
// per-user compliance signals into Google's CAA evaluator.
//
// See docs/Contributing/research/security-compliance/google-cloud-identity-conditional-access-design.md
// for the broader design rationale; the *Endpoint Verification as the
// resolution mechanism* section describes the file layout this table reads.
//
// Implemented for macOS in v1. Windows and Linux paths are stubbed pending
// verification of EV's actual on-disk layout on those platforms (Google's
// REST docs name `~/.secureConnect/context_aware_config.json` but the macOS
// path is already stale; Linux/Windows paths warrant confirmation against a
// live EV install before shipping).
package endpoint_verification_accounts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog"
)

const tableName = "endpoint_verification_accounts"

// Table implements the osquery extension table plugin.
type Table struct {
	name   string
	logger zerolog.Logger
	// userLister returns the home directories of local interactive users.
	// Injectable for testing.
	userLister func() ([]userHome, error)
}

type userHome struct {
	uid      string
	username string
	homeDir  string
}

// TablePlugin constructs the osquery extension table.
func TablePlugin(logger zerolog.Logger) *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("uid"),
		table.TextColumn("username"),
		table.TextColumn("gaia_id"),
		table.TextColumn("resource_id"),
		table.TextColumn("email"),
		table.TextColumn("last_sync"),
	}
	t := &Table{
		name:       tableName,
		logger:     logger.With().Str("table", tableName).Logger(),
		userLister: listLocalUsers,
	}
	return table.NewPlugin(t.name, columns, t.generate)
}

// accountsFile mirrors the on-disk structure of EV's accounts.json:
//
//	{
//	  "<gaia_user_id>": {
//	    "device": { "resourceId": "<rawResourceId>", "lastSync": "<RFC3339>" },
//	    "user":   { "email": "<workspace_email>" }
//	  }, ...
//	}
//
// On macOS the file is at
// ~/Library/Application Support/Google/Endpoint Verification/accounts.json
// — note this is NOT the path Google's REST reference documents
// (~/.secureConnect/context_aware_config.json on macOS). The
// `.secureConnect/` path is checked as a fallback for older EV installs.
type accountsFile map[string]accountEntry

type accountEntry struct {
	Device deviceEntry `json:"device"`
	User   userEntry   `json:"user"`
}

type deviceEntry struct {
	ResourceID string `json:"resourceId"`
	LastSync   string `json:"lastSync"`
}

type userEntry struct {
	Email string `json:"email"`
}

func (t *Table) generate(_ context.Context, _ table.QueryContext) ([]map[string]string, error) {
	users, err := t.userLister()
	if err != nil {
		t.logger.Debug().Err(err).Msg("list local users failed")
		// Return empty rather than error — osquery treats nil error+empty
		// rows as "table queried successfully but no data."
		return []map[string]string{}, nil
	}

	rows := make([]map[string]string, 0, len(users))
	for _, u := range users {
		paths := candidatePaths(u.homeDir)
		for _, p := range paths {
			entries, ok := readAccounts(p)
			if !ok {
				continue
			}
			for gaiaID, ent := range entries {
				if ent.Device.ResourceID == "" || ent.User.Email == "" {
					continue
				}
				rows = append(rows, map[string]string{
					"uid":         u.uid,
					"username":    u.username,
					"gaia_id":     gaiaID,
					"resource_id": ent.Device.ResourceID,
					"email":       ent.User.Email,
					"last_sync":   ent.Device.LastSync,
				})
			}
			// Stop after the first file that parsed successfully — don't
			// double-emit if both the current and legacy paths exist.
			break
		}
	}
	return rows, nil
}

// candidatePaths returns the list of accounts.json locations to try for a
// given user's home directory, in preferred order. Current EV macOS path
// first, then the legacy `.secureConnect/` location Google's REST docs
// reference.
func candidatePaths(homeDir string) []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			filepath.Join(homeDir, "Library", "Application Support", "Google", "Endpoint Verification", "accounts.json"),
			filepath.Join(homeDir, ".secureConnect", "context_aware_config.json"),
		}
	case "linux":
		// EV layout on Linux is unverified against a live install as of
		// 2026-05-29. The Google REST docs name .secureConnect/ but those
		// docs are stale for macOS, so caveat emptor on Linux too. See the
		// "Open questions" section of the design doc for tracking.
		return []string{filepath.Join(homeDir, ".secureConnect", "context_aware_config.json")}
	case "windows":
		// Same caveat as Linux.
		return []string{filepath.Join(homeDir, ".secureConnect", "context_aware_config.json")}
	default:
		return nil
	}
}

func readAccounts(path string) (accountsFile, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		// Silently skip on any read failure (file missing, permissions,
		// etc.) — we don't want to noisy-log on every host that lacks EV.
		return nil, false
	}
	var entries accountsFile
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, false
	}
	return entries, true
}

// listLocalUsers is the platform-specific implementation; only macOS is
// implemented in v1. The other platforms return an error which generate()
// swallows into an empty result set.
var listLocalUsers = func() ([]userHome, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("endpoint_verification_accounts: %s not yet implemented", runtime.GOOS)
	}
	return listLocalUsersDarwin()
}
