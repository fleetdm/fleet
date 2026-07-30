package seed

import (
	cryptorand "crypto/rand"
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

// mdmConfigSubset is the slice of GET /config we use to decide which
// profile uploads to attempt. The full /config response is huge — only the
// per-platform "enabled and configured" flags are relevant here.
type mdmConfigSubset struct {
	MDM struct {
		EnabledAndConfigured        bool `json:"enabled_and_configured"`         // Apple
		WindowsEnabledAndConfigured bool `json:"windows_enabled_and_configured"` // Windows
	} `json:"mdm"`
}

func Profiles(c Client, log Logger, theme themes.Theme, teams []Team, count int) Result {
	res := Result{Entity: "profiles"}

	// Skip platforms whose MDM stack isn't turned on — otherwise every
	// upload returns the same 400 "MDM features aren't turned on in Fleet"
	// and floods the run output.
	var cfg mdmConfigSubset
	if err := c.Get("/api/latest/fleet/config", &cfg); err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("check MDM config: %w", err))
		return res
	}
	appleOK := cfg.MDM.EnabledAndConfigured
	winOK := cfg.MDM.WindowsEnabledAndConfigured
	if !appleOK && !winOK {
		log.Printf("profiles: MDM not enabled (neither Apple nor Windows), skipping")
		return res
	}

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
				"{{UUID}}", randomUUIDv4(),
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

	// Global profiles, alternating Apple / Windows — but only for the
	// platforms whose MDM stack is enabled.
	for i := 0; i < count; i++ {
		if i%2 == 0 {
			if appleOK {
				postOne("apple", 0, i)
			}
		} else {
			if winOK {
				postOne("windows", 0, i)
			}
		}
	}
	// One Apple + one Windows per team, again gated by which stacks are on.
	for _, t := range teams {
		if appleOK {
			postOne("apple", t.ID, 100)
		}
		if winOK {
			postOne("windows", t.ID, 101)
		}
	}
	return res
}

// randomUUIDv4 returns a fresh RFC 4122 v4 UUID. Per-profile UUIDs prevent
// macOS from treating every seeded profile as the same payload (which would
// cause install/update collisions).
func randomUUIDv4() string {
	var b [16]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		// Vanishingly unlikely; fall back to a clearly-fake-but-unique-ish
		// value so callers can still spot seeded rows.
		return "00000000-0000-0000-0000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
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
