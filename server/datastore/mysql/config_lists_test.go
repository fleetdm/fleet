package mysql

import (
	"crypto/sha256"
	"encoding/json"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestConfigListsByTeamIDs(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := t.Context()

	tm1, err := ds.NewTeam(ctx, &fleet.Team{Name: "tm1"})
	require.NoError(t, err)
	tm2, err := ds.NewTeam(ctx, &fleet.Team{Name: "tm2"})
	require.NoError(t, err)
	tmEmpty, err := ds.NewTeam(ctx, &fleet.Team{Name: "empty"})
	require.NoError(t, err)

	newLabel := func(name string) *fleet.Label {
		lbl, err := ds.NewLabel(ctx, &fleet.Label{Name: name, Query: "select 1"})
		require.NoError(t, err)
		return lbl
	}
	lblAll := newLabel("include-all-1")
	lblAny := newLabel("include-any-1")
	lblExclude := newLabel("exclude-1")

	newAppleProfile := func(teamID *uint, name, identifier string, labels ...*fleet.Label) {
		cp := configProfileForTest(t, name, identifier, uuid.NewString(), labels...)
		cp.TeamID = teamID
		_, err := ds.NewMDMAppleConfigProfile(ctx, *cp, nil)
		require.NoError(t, err)
	}
	newDeclaration := func(teamID *uint, name, identifier string, labels ...*fleet.Label) {
		decl := declForTest(name, identifier, "", labels...)
		decl.TeamID = teamID
		_, err := ds.NewMDMAppleDeclaration(ctx, decl, nil)
		require.NoError(t, err)
	}
	newWindowsProfile := func(teamID *uint, name string, labels ...*fleet.Label) {
		cp := windowsConfigProfileForTest(t, name, "./Device/"+name, labels...)
		cp.TeamID = teamID
		_, err := ds.NewMDMWindowsConfigProfile(ctx, *cp, nil)
		require.NoError(t, err)
	}
	newAndroidProfile := func(teamID *uint, name string, labels ...*fleet.Label) {
		cp := androidConfigProfileForTest(t, name, nil, labels...)
		cp.TeamID = teamID
		_, err := ds.NewMDMAndroidConfigProfile(ctx, *cp, nil)
		require.NoError(t, err)
	}
	newScript := func(teamID *uint, name string) {
		_, err := ds.NewScript(ctx, &fleet.Script{TeamID: teamID, Name: name, ScriptContents: "echo " + name})
		require.NoError(t, err)
	}

	// "No fleet": every kind of list, plus Fleet-reserved profiles that must be left out.
	newAppleProfile(nil, "Wi-Fi", "com.example.wifi", lblAll)
	newAppleProfile(nil, "Disk encryption", mobileconfig.FleetFileVaultPayloadIdentifier)
	newDeclaration(nil, "Passcode", "passcode", lblExclude)
	newDeclaration(nil, mdm.FleetMacOSUpdatesProfileName, "osupdates")
	newWindowsProfile(nil, "Firewall", lblAny)
	newWindowsProfile(nil, mdm.FleetWindowsOSUpdatesProfileName)
	newAndroidProfile(nil, "Restrictions")
	newScript(nil, "b.sh")
	newScript(nil, "a.sh")
	_, err = ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{Name: "assistant.json", Profile: json.RawMessage("{}")})
	require.NoError(t, err)
	_, err = ds.SetSetupExperienceScript(ctx, &fleet.Script{Name: "setup.sh", ScriptContents: "echo setup"})
	require.NoError(t, err)
	pkg := []byte("bootstrap")
	sum := sha256.Sum256(pkg)
	err = ds.InsertMDMAppleBootstrapPackage(ctx, &fleet.MDMAppleBootstrapPackage{
		TeamID: 0, Name: "bootstrap.pkg", Sha256: sum[:], Bytes: pkg, Token: uuid.NewString(),
	}, nil)
	require.NoError(t, err)

	// Same profile and script names on other fleets must not leak into "No fleet".
	newAppleProfile(&tm1.ID, "Wi-Fi", "com.example.wifi", lblAny, lblExclude)
	newScript(&tm1.ID, "a.sh")
	_, err = ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{TeamID: &tm2.ID, Name: "tm2-assistant.json", Profile: json.RawMessage("{}")})
	require.NoError(t, err)

	newVPPToken := func(location string, teamIDs []uint) {
		data, err := test.CreateVPPTokenData(time.Now().Add(24*time.Hour), "org "+location, location)
		require.NoError(t, err)
		tok, err := ds.InsertVPPToken(ctx, data)
		require.NoError(t, err)
		if teamIDs != nil {
			_, err = ds.UpdateVPPTokenTeams(ctx, tok.ID, teamIDs)
			require.NoError(t, err)
		}
	}
	newVPPToken("specific", []uint{tm2.ID, 0, tm1.ID})
	newVPPToken("unassigned", nil)

	lists, err := ds.ConfigListsByTeamIDs(ctx, []uint{0, tm1.ID, tm2.ID, tmEmpty.ID})
	require.NoError(t, err)
	require.Len(t, lists, 4)

	require.Equal(t, &fleet.ConfigLists{
		AppleProfiles: []fleet.MDMProfileSpec{
			{Name: "Passcode", LabelsExcludeAny: []string{"exclude-1"}},
			{Name: "Wi-Fi", LabelsIncludeAll: []string{"include-all-1"}},
		},
		WindowsProfiles:  []fleet.MDMProfileSpec{{Name: "Firewall", LabelsIncludeAny: []string{"include-any-1"}}},
		AndroidProfiles:  []fleet.MDMProfileSpec{{Name: "Restrictions"}},
		Scripts:          []string{"a.sh", "b.sh"},
		BootstrapPackage: "bootstrap.pkg",
		SetupAssistant:   "assistant.json",
		SetupScript:      "setup.sh",
		VolumePurchasingProgram: []fleet.MDMAppleVolumePurchasingProgramInfo{
			{Location: "specific", Teams: []string{fleet.TeamNameNoTeam, "tm1", "tm2"}},
			{Location: "unassigned", Teams: []string{}},
		},
	}, lists[0])

	tm1Lists := fleet.NewConfigLists()
	tm1Lists.AppleProfiles = []fleet.MDMProfileSpec{
		{Name: "Wi-Fi", LabelsIncludeAny: []string{"include-any-1"}, LabelsExcludeAny: []string{"exclude-1"}},
	}
	tm1Lists.Scripts = []string{"a.sh"}
	require.Equal(t, tm1Lists, lists[tm1.ID])

	tm2Lists := fleet.NewConfigLists()
	tm2Lists.SetupAssistant = "tm2-assistant.json"
	require.Equal(t, tm2Lists, lists[tm2.ID])

	require.Equal(t, fleet.NewConfigLists(), lists[tmEmpty.ID])

	lists, err = ds.ConfigListsByTeamIDs(ctx, []uint{tm1.ID})
	require.NoError(t, err)
	require.Equal(t, map[uint]*fleet.ConfigLists{tm1.ID: tm1Lists}, lists)

	lists, err = ds.ConfigListsByTeamIDs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, lists)

	// A token assigned to all fleets lists "All fleets", which is what GitOps accepts.
	_, err = ds.writer(ctx).ExecContext(ctx, `DELETE FROM vpp_tokens`)
	require.NoError(t, err)
	newVPPToken("all", []uint{})
	lists, err = ds.ConfigListsByTeamIDs(ctx, []uint{0})
	require.NoError(t, err)
	require.NotNil(t, lists[0])
	require.Equal(t, []fleet.MDMAppleVolumePurchasingProgramInfo{{Location: "all", Teams: []string{fleet.DisplayNameAllTeams}}}, lists[0].VolumePurchasingProgram)
}
