//go:build windows

package apps

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"golang.org/x/sys/windows/registry"
)

func scanApps(_ []homes.Home) []App {
	seen := map[string]struct{}{}
	var out []App

	roots := []struct {
		key   registry.Key
		sub   string
		scope string
	}{
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, "system"},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`, "system"},
		{registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, "user"},
	}

	for _, r := range roots {
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
			ka, ok := matchKnown(display)
			if _, dup := seen[ka.name]; !ok || dup {
				continue
			}
			seen[ka.name] = struct{}{}
			out = append(out, App{
				Name:           ka.name,
				Vendor:         pub,
				Version:        version,
				Path:           loc,
				PlatformSource: "registry",
				Scope:          r.scope,
			})
		}
		k.Close()
	}
	return out
}
