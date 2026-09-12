package fleetctl

import (
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

// scriptSetupTitle describes one Linux script-only title: how it is selected
// for the setup experience on each of its two possible targets, and what
// generate is expected to write for it.
type scriptSetupTitle struct {
	id       uint
	filename string
	native   bool // selected on Linux, its own platform
	cross    bool // selected on macOS
	multi    bool // title holds more than one package, so it takes the package-file path

	wantBoolean  bool   // expected setup_experience
	wantPlatform string // expected setup_experience_platform, empty for absent
}

type MockClientScriptSetup struct {
	MockClient
	titles []scriptSetupTitle
}

const scriptSetupTeamID = 91

func (c *MockClientScriptSetup) listed() []fleet.SoftwareTitleListResult {
	out := make([]fleet.SoftwareTitleListResult, 0, len(c.titles))
	for _, tt := range c.titles {
		out = append(out, fleet.SoftwareTitleListResult{
			ID:         tt.id,
			Name:       tt.filename,
			HashSHA256: new(tt.filename + "-hash"),
			SoftwarePackage: &fleet.SoftwarePackageOrApp{
				Name: tt.filename, Platform: "linux", Version: "1.0",
			},
		})
	}
	return out
}

func (c *MockClientScriptSetup) ListSoftwareTitles(query string) ([]fleet.SoftwareTitleListResult, error) {
	if query == fmt.Sprintf("available_for_install=1&fleet_id=%d&order_key=name", scriptSetupTeamID) {
		return c.listed(), nil
	}
	return c.MockClient.ListSoftwareTitles(query)
}

func (c *MockClientScriptSetup) GetSoftwareTitleByID(id uint, teamID *uint) (*fleet.SoftwareTitle, error) {
	for _, tt := range c.titles {
		if tt.id != id {
			continue
		}
		pkg := &fleet.SoftwareInstaller{
			InstallScript: "#!/usr/bin/env python3\nprint('x')",
			Platform:      "linux",
			URL:           "https://example.com/download/" + tt.filename,
			Name:          tt.filename,
		}
		title := &fleet.SoftwareTitle{ID: tt.id, SoftwarePackage: pkg}
		if tt.multi {
			title.Packages = []fleet.SoftwareInstaller{*pkg, *pkg}
		}
		return title, nil
	}
	return c.MockClient.GetSoftwareTitleByID(id, teamID)
}

// GetSetupExperienceSoftware mirrors the real endpoint: it lists every title
// selectable for the queried platform, and install_during_setup carries the
// selection. The darwin-only query reports the cross-table selection; the
// combined query reports the installer's own flag.
func (c *MockClientScriptSetup) GetSetupExperienceSoftware(platform string, teamID uint) ([]fleet.SoftwareTitleListResult, error) {
	if teamID != scriptSetupTeamID {
		return c.MockClient.GetSetupExperienceSoftware(platform, teamID)
	}
	out := c.listed()
	for i, tt := range c.titles {
		selected := tt.native
		if platform == "macos" {
			selected = tt.cross
		}
		out[i].SoftwarePackage.InstallDuringSetup = new(selected)
	}
	return out, nil
}

// TestGenerateSoftwareScriptPackageSetupExperience covers the four setup
// experience selection states of a Linux script-only package. An unselected
// package must not be written as cross-selected, and a package selected on
// both platforms must name both in setup_experience_platform.
func TestGenerateSoftwareScriptPackageSetupExperience(t *testing.T) {
	for _, ext := range []string{"sh", "py"} {
		t.Run(ext, func(t *testing.T) {
			for _, multi := range []bool{false, true} {
				name := "single package"
				if multi {
					name = "multiple packages"
				}
				t.Run(name, func(t *testing.T) {
					titles := []scriptSetupTitle{
						{id: 100, filename: "none." + ext, multi: multi},
						{id: 101, filename: "crossonly." + ext, multi: multi, cross: true, wantPlatform: "darwin"},
						{id: 102, filename: "nativeonly." + ext, multi: multi, native: true, wantBoolean: true},
						{id: 103, filename: "both." + ext, multi: multi, native: true, cross: true, wantPlatform: "darwin,linux"},
					}
					fleetClient := &MockClientScriptSetup{titles: titles}
					appConfig, err := fleetClient.GetAppConfig()
					require.NoError(t, err)
					cmd := &GenerateGitopsCommand{
						Client:       fleetClient,
						CLI:          cli.NewContext(cli.NewApp(), nil, nil),
						Messages:     Messages{},
						FilesToWrite: make(map[string]any),
						AppConfig:    appConfig,
						SoftwareList: make(map[uint]Software),
						ScriptList:   make(map[uint]string),
					}

					res, err := cmd.generateSoftware("fleets/t.yml", scriptSetupTeamID, "team-91", false)
					require.NoError(t, err)
					packages, ok := res["packages"].([]map[string]any)
					require.True(t, ok)
					require.Len(t, packages, len(titles))

					for i, tt := range titles {
						pkg := packages[i]
						boolean, hasBoolean := pkg["setup_experience"]
						platform, hasPlatform := pkg["setup_experience_platform"]

						if tt.wantBoolean {
							assert.Equal(t, true, boolean, "%s setup_experience", tt.filename)
						} else {
							assert.False(t, hasBoolean, "%s should not emit setup_experience", tt.filename)
						}
						if tt.wantPlatform == "" {
							assert.False(t, hasPlatform, "%s should not emit setup_experience_platform, got %v", tt.filename, platform)
							continue
						}
						assert.Equal(t, tt.wantPlatform, platform, "%s setup_experience_platform", tt.filename)
					}
				})
			}
		})
	}
}
