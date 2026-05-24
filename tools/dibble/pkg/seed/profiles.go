package seed

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/themes"
)

// Profiles seeds MDM configuration profiles. The same endpoint
// (/api/latest/fleet/configuration_profiles) accepts both Apple
// (.mobileconfig) and Windows (.xml) payloads — Fleet detects the platform
// from the content.
//
// The seeded profile content is parameterized: the PayloadIdentifier (Apple)
// or LocURI (Windows) is filled with a themed string so each profile is
// unique per seed run.

//go:embed templates/profile.mobileconfig.tmpl
var appleProfileTmpl string

//go:embed templates/profile.windows.xml.tmpl
var windowsProfileTmpl string

func Profiles(c Client, log Logger, theme themes.Theme, teams []Team, count int) Result {
	res := Result{Entity: "profiles"}

	postOne := func(platform string, teamID uint, idx int) {
		n := themes.Pick(theme, "policy", idx) // reuse policy names — they make solid profile names
		safeName := sanitizeProfileName(n.Name)
		var content string
		var filename string
		switch platform {
		case "apple":
			content = strings.NewReplacer(
				"{{NAME}}", safeName,
				"{{IDENT}}", "dev.dibble."+safeName,
			).Replace(appleProfileTmpl)
			filename = safeName + ".mobileconfig"
		case "windows":
			content = strings.NewReplacer(
				"{{NAME}}", safeName,
				"{{LOCURI}}", "./Vendor/Dibble/"+safeName,
			).Replace(windowsProfileTmpl)
			filename = safeName + ".xml"
		}
		fields := map[string]string{}
		if teamID > 0 {
			// Fleet's renamed "team" → "fleet" — the multipart key is fleet_id.
			fields["fleet_id"] = fmt.Sprintf("%d", teamID)
		}
		// Form field is "profile"; Fleet detects platform from the filename's extension.
		files := []MultipartFile{{FieldName: "profile", Filename: filename, Content: []byte(content)}}
		err := c.PostMultipart("/api/latest/fleet/configuration_profiles", fields, files, nil)
		switch {
		case err == nil:
			res.Created++
			log.Printf("profile (%s, %s) %s", platform, scopeLabel(teamID), filename)
		case IsAlreadyExists(err):
			res.Skipped++
		default:
			res.Errors = append(res.Errors, err)
		}
	}

	// Global profiles, alternating Apple / Windows.
	for i := 0; i < count; i++ {
		if i%2 == 0 {
			postOne("apple", 0, i)
		} else {
			postOne("windows", 0, i)
		}
	}
	// One Apple + one Windows per team.
	for _, t := range teams {
		postOne("apple", t.ID, 100)
		postOne("windows", t.ID, 101)
	}
	return res
}

func sanitizeProfileName(s string) string {
	r := strings.NewReplacer(
		" ", "-", "/", "-", "\\", "-", ":", "-",
		"'", "", `"`, "", ".", "-",
	)
	return strings.Trim(r.Replace(strings.ToLower(s)), "-")
}

func scopeLabel(teamID uint) string {
	if teamID == 0 {
		return "global"
	}
	return fmt.Sprintf("team=%d", teamID)
}
