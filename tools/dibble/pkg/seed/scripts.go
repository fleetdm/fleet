package seed

import (
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/themes"
)

// Scripts seeds saved scripts (global + per-team). The endpoint accepts a
// multipart upload with the script content as a file plus `team_id` (or
// none for global) as a form field.
//
// Script content varies by extension: .sh / .ps1 / .zsh — so the seeded set
// covers all three platforms in the UI.
func Scripts(c Client, log Logger, theme themes.Theme, teams []Team, count int) Result {
	res := Result{Entity: "scripts"}

	exts := []string{".sh", ".ps1", ".zsh"}
	body := func(ext, name string) string {
		switch ext {
		case ".ps1":
			return fmt.Sprintf("# %s\nWrite-Host 'dibble was here'\n", name)
		case ".zsh":
			return fmt.Sprintf("#!/usr/bin/env zsh\n# %s\necho 'dibble was here'\n", name)
		default:
			return fmt.Sprintf("#!/usr/bin/env bash\n# %s\necho 'dibble was here'\n", name)
		}
	}

	postOne := func(scope string, teamID uint, i int) {
		n := themes.Pick(theme, "script", i)
		ext := exts[i%len(exts)]
		name := strings.TrimSuffix(n.Name, ".sh")
		name = strings.TrimSuffix(name, ".ps1")
		name = strings.TrimSuffix(name, ".zsh") + ext
		fields := map[string]string{}
		if teamID > 0 {
			// Fleet's renamed "team" → "fleet" — the multipart key is fleet_id.
			fields["fleet_id"] = fmt.Sprintf("%d", teamID)
		}
		// Form field must be exactly "script"; filename carries the extension.
		files := []MultipartFile{{FieldName: "script", Filename: name, Content: []byte(body(ext, n.Name))}}
		err := c.PostMultipart("/api/latest/fleet/scripts", fields, files, nil)
		switch {
		case err == nil:
			res.Created++
			log.Printf("script (%s) %s", scope, name)
		case IsAlreadyExists(err):
			res.Skipped++
		default:
			res.Errors = append(res.Errors, err)
		}
	}

	for i := 0; i < count; i++ {
		postOne("global", 0, i)
	}
	for _, t := range teams {
		postOne("team="+t.Name, t.ID, 0)
	}
	return res
}
