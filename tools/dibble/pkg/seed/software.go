package seed

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// installerFiles bundles a curated set of installer fixtures into the dibble
// binary so `dibble software custom` can upload real package files without
// the user pointing at a checkout. Files are sourced from
// server/service/testdata/software-installers/ — the same fixtures Fleet's
// own tests use. .exe and vim.deb are excluded for size.
//
//go:embed data/installers/*
var installerFiles embed.FS

// extensionInstallers lists the curated 2-3 installer fixtures per
// extension. Order matters for display; the first entry per extension is
// uploaded first which keeps log output readable.
var extensionInstallers = map[string][]string{
	".pkg":    {"dummy_installer.pkg", "EchoApp.pkg", "no_version.pkg"},
	".deb":    {"emacs.deb", "ruby.deb", "ruby_arm64.deb"},
	".msi":    {"fleet-osquery.msi"}, // only fixture available
	".rpm":    {"ruby.rpm"},          // only fixture available
	".tar.gz": {"test.tar.gz"},       // only fixture available
	".ipa":    {"ipa_test.ipa", "ipa_test2.ipa"},
}

// extensionScripts maps an extension to the install / uninstall script form
// field values to send with the upload. Most extensions are left empty so
// Fleet auto-generates the commands; .tar.gz rejects uploads without explicit
// install AND uninstall scripts ("Install script is required for .tar.gz
// packages") so we ship placeholders that satisfy the validator.
var extensionScripts = map[string]struct {
	install   string
	uninstall string
}{
	".tar.gz": {
		install:   "#!/bin/sh\necho 'dibble seeded — replace with real install logic'\n",
		uninstall: "#!/bin/sh\necho 'dibble seeded — replace with real uninstall logic'\n",
	},
}

// SoftwareOptions configures the custom-package and Fleet-maintained-app
// seeders. TeamID == 0 targets "no team" (global); a non-zero value scopes
// the upload to that team.
type SoftwareOptions struct {
	// TeamID selects the team that uploaded installers and added maintained
	// apps land under. Zero means no team / global.
	TeamID uint

	// MaintainedAppCount is how many entries from /software/fleet_maintained_apps
	// to seed. Zero skips FMA entirely. Defaults to 3 when running the
	// "maintained" / "all" subcommands.
	MaintainedAppCount int
}

// loadInstaller reads a single embedded fixture by name.
func loadInstaller(name string) ([]byte, error) {
	full := path.Join("data/installers", name)
	return fs.ReadFile(installerFiles, full)
}

// sortedExtensions returns the supported extensions in a deterministic
// order so seeded output is stable across runs.
func sortedExtensions() []string {
	keys := make([]string, 0, len(extensionInstallers))
	for k := range extensionInstallers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// SoftwareCustom uploads the curated 2-3 installer fixtures per supported
// extension to the given team (or "no team" when opt.TeamID == 0). Each
// upload posts multipart to /api/latest/fleet/software/package, the same
// endpoint Fleet's UI calls when adding a custom package.
//
// Install / uninstall scripts are left empty; the server auto-generates
// defaults based on the file extension.
func SoftwareCustom(c Client, log Logger, opt SoftwareOptions) Result {
	res := Result{Entity: "software (custom)"}

	scope := "no team"
	teamField := ""
	if opt.TeamID > 0 {
		teamField = fmt.Sprintf("%d", opt.TeamID)
		scope = fmt.Sprintf("team=%d", opt.TeamID)
	}

	for _, ext := range sortedExtensions() {
		for _, fixture := range extensionInstallers[ext] {
			content, err := loadInstaller(fixture)
			if err != nil {
				res.Errors = append(res.Errors,
					fmt.Errorf("load %s: %w", fixture, err))
				continue
			}
			// Build the fields map per-fixture: extensions like .tar.gz
			// need an explicit install_script, others let Fleet
			// auto-generate one.
			fields := map[string]string{}
			if teamField != "" {
				fields["fleet_id"] = teamField
			}
			if scripts, ok := extensionScripts[ext]; ok {
				fields["install_script"] = scripts.install
				fields["uninstall_script"] = scripts.uninstall
			}
			files := []MultipartFile{{
				FieldName: "software",
				Filename:  fixture,
				Content:   content,
			}}
			err = c.PostMultipart("/api/latest/fleet/software/package", fields, files, nil)
			switch {
			case err == nil:
				res.Created++
				log.Printf("software (%s) %s [%s]", scope, fixture, ext)
			case IsAlreadyExists(err):
				res.Skipped++
				log.Printf("software (%s) %s already exists", scope, fixture)
			default:
				res.Errors = append(res.Errors,
					fmt.Errorf("%s: %w", fixture, err))
			}
		}
	}
	return res
}

// maintainedApp is the subset of fleet.MaintainedApp the seeder cares
// about. Decoded from the listFleetMaintainedApps response.
type maintainedApp struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
}

type listMaintainedAppsResponse struct {
	FleetMaintainedApps []maintainedApp `json:"fleet_maintained_apps"`
}

// SoftwareMaintained adds a handful of Fleet-maintained apps to the given
// team by:
//
//  1. GET /api/latest/fleet/software/fleet_maintained_apps to discover what
//     the server's catalog contains (the list is generated server-side).
//  2. POST /api/latest/fleet/software/fleet_maintained_apps with each
//     selected fleet_maintained_app_id.
//
// Adding FMAs is per-team; opt.TeamID == 0 means no team / global.
func SoftwareMaintained(c Client, log Logger, opt SoftwareOptions) Result {
	res := Result{Entity: "software (maintained)"}
	if opt.MaintainedAppCount <= 0 {
		return res
	}

	listPath := "/api/latest/fleet/software/fleet_maintained_apps"
	if opt.TeamID > 0 {
		listPath = fmt.Sprintf("%s?team_id=%d", listPath, opt.TeamID)
	}
	var list listMaintainedAppsResponse
	if err := c.Get(listPath, &list); err != nil {
		res.Errors = append(res.Errors,
			fmt.Errorf("list fleet-maintained apps: %w", err))
		return res
	}
	if len(list.FleetMaintainedApps) == 0 {
		log.Printf("software (maintained): server returned no maintained apps to add")
		return res
	}

	// Pick the first N from the server's list — the catalog is curated so
	// the head of the list is stable.
	n := opt.MaintainedAppCount
	if n > len(list.FleetMaintainedApps) {
		n = len(list.FleetMaintainedApps)
	}

	for i := 0; i < n; i++ {
		app := list.FleetMaintainedApps[i]
		body := map[string]any{
			"fleet_maintained_app_id": app.ID,
		}
		if opt.TeamID > 0 {
			body["fleet_id"] = opt.TeamID
		}
		err := c.Post("/api/latest/fleet/software/fleet_maintained_apps", body, nil)
		switch {
		case err == nil:
			res.Created++
			log.Printf("software (maintained) %s [%s] id=%d",
				app.Name, app.Platform, app.ID)
		case IsAlreadyExists(err) || isAlreadyAdded(err):
			res.Skipped++
		default:
			res.Errors = append(res.Errors,
				fmt.Errorf("add maintained app %s (id=%d): %w", app.Name, app.ID, err))
		}
	}
	return res
}

// isAlreadyAdded recognizes the "already added" error Fleet returns when a
// maintained app is re-added to the same team. The error isn't a generic
// "already exists" 409, so we have to sniff the message.
func isAlreadyAdded(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already") &&
		(strings.Contains(msg, "added") || strings.Contains(msg, "associated"))
}
