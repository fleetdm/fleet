//go:build windows

package apps

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"golang.org/x/sys/windows/registry"
)

// uninstallSubKeys are the uninstall-entry paths to read under each hive root.
// The WOW6432Node variant only exists machine-wide, but reading a missing key
// just fails and is skipped, so the same pair is used for every root.
var uninstallSubKeys = []string{
	`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
	`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
}

type uninstallRoot struct {
	key   registry.Key
	sub   string
	scope string
}

// uninstallRoots lists every hive to search for uninstall entries.
//
// HKEY_USERS matters because the extension runs as NT AUTHORITY\SYSTEM: in that
// process CURRENT_USER resolves to SYSTEM's own effectively empty hive, never the
// interactive user's. Per-user Electron/Squirrel installers (Ollama, ChatGPT
// desktop, LM Studio) write only to the installing user's hive and offer no
// machine-wide option, so without walking HKEY_USERS they are invisible. Only
// hives of logged-in users are loaded there, which is the same liveness scope the
// homes.Home enumeration gives the other platforms.
//
// CURRENT_USER is still read so the collector keeps working when the extension
// runs unprivileged and cannot open other users' hives.
//
// Machine-wide roots come first so that an app installed both ways is reported
// with scope "system", which is how the collector already behaved.
func uninstallRoots() []uninstallRoot {
	var roots []uninstallRoot
	for _, sub := range uninstallSubKeys {
		roots = append(roots, uninstallRoot{registry.LOCAL_MACHINE, sub, "system"})
	}
	for _, sub := range uninstallSubKeys {
		roots = append(roots, uninstallRoot{registry.CURRENT_USER, sub, "user"})
	}

	for _, hive := range realUserHives() {
		for _, sub := range uninstallSubKeys {
			roots = append(roots, uninstallRoot{registry.USERS, hive + `\` + sub, "user"})
		}
	}
	return roots
}

// realUserHives returns the HKEY_USERS subkey names of the loaded hives that
// belong to a real user's account. Shared by the uninstall-key and MSIX scans,
// which both need to attribute an install to a person rather than to a service
// account.
func realUserHives() []string {
	hu, err := registry.OpenKey(registry.USERS, "", registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil
	}
	defer hu.Close()
	names, err := hu.ReadSubKeyNames(-1)
	if err != nil {
		return nil
	}
	var out []string
	for _, name := range names {
		if isRealUserHive(name) {
			out = append(out, name)
		}
	}
	return out
}

func scanApps(homesList []homes.Home) []App {
	c := newAppCollector()

	for _, r := range uninstallRoots() {
		k, err := registry.OpenKey(r.key, r.sub, registry.READ)
		if err != nil {
			continue
		}
		subKeys, _ := k.ReadSubKeyNames(-1)
		for _, name := range subKeys {
			sk, err := registry.OpenKey(r.key, r.sub+`\`+name, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			display, _, _ := sk.GetStringValue("DisplayName")
			version, _, _ := sk.GetStringValue("DisplayVersion")
			loc, _, _ := sk.GetStringValue("InstallLocation")
			pub, _, _ := sk.GetStringValue("Publisher")
			sk.Close()

			if display == "" {
				continue
			}
			c.add(appCandidate{
				MatchTokens: []string{display},
				DisplayName: display,
				Vendor:      pub,
				Version:     version,
				Path:        loc,
				Scope:       r.scope,
				Source:      "registry",
			})
		}
		k.Close()
	}

	// MSIX/Appx packages register nowhere near the uninstall keys, so they need a
	// separate pass over the package repository. It shares the collector, so an
	// app found both ways is reported once.
	scanAppx(c, homesList)
	return c.apps()
}
