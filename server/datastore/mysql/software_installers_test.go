package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSoftwareInstallers(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"InsertSoftwareInstallRequest", testInsertSoftwareInstallRequest},
		{"GetSoftwareInstallResults", testGetSoftwareInstallResult},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testInsertSoftwareInstallRequest(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)

	cases := map[string]*uint{
		"no team": nil,
		"team":    &team.ID,
	}

	for tc, teamID := range cases {
		t.Run(tc, func(t *testing.T) {
			// non-existent installer and host does the installer check first
			err := ds.InsertSoftwareInstallRequest(ctx, 1, 1, teamID)
			var nfe fleet.NotFoundError
			require.ErrorAs(t, err, &nfe)

			// non-existent host
			installerID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
				Title:         "foo",
				Source:        "bar",
				InstallScript: "echo",
				TeamID:        teamID,
			})
			require.NoError(t, err)
			installerMeta, err := ds.GetSoftwareInstallerMetadata(ctx, installerID)
			require.NoError(t, err)

			err = ds.InsertSoftwareInstallRequest(ctx, 12, installerMeta.TitleID, teamID)
			require.ErrorAs(t, err, &nfe)

			// successful insert
			host, err := ds.NewHost(ctx, &fleet.Host{
				Hostname:      "macos-test" + tc,
				OsqueryHostID: ptr.String("osquery-macos" + tc),
				NodeKey:       ptr.String("node-key-macos" + tc),
				UUID:          uuid.NewString(),
				Platform:      "darwin",
				TeamID:        teamID,
			})
			require.NoError(t, err)
			err = ds.InsertSoftwareInstallRequest(ctx, host.ID, installerMeta.TitleID, teamID)
			require.NoError(t, err)
		})
	}
}

func testGetSoftwareInstallResult(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)
	teamID := team.ID

	for _, tc := range []struct {
		name                    string
		uuid                    string
		expectedStatus          fleet.SoftwareInstallerStatus
		postInstallScriptEC     *uint
		preInstallQueryOutput   *string
		installScriptEC         *uint
		postInstallScriptOutput *string
		installScriptOutput     *string
	}{
		{
			name:                    "pending install",
			uuid:                    "pending",
			expectedStatus:          fleet.SoftwareInstallerPending,
			postInstallScriptOutput: ptr.String("post install output"),
			installScriptOutput:     ptr.String("install output"),
		},
		{
			name:                    "failing install post install script",
			uuid:                    "fail_post_install_script",
			expectedStatus:          fleet.SoftwareInstallerFailed,
			postInstallScriptEC:     ptr.Uint(1),
			postInstallScriptOutput: ptr.String("post install output"),
			installScriptOutput:     ptr.String("install output"),
		},
		{
			name:                    "failing install install script",
			uuid:                    "fail_install_script",
			expectedStatus:          fleet.SoftwareInstallerFailed,
			installScriptEC:         ptr.Uint(1),
			postInstallScriptOutput: ptr.String("post install output"),
			installScriptOutput:     ptr.String("install output"),
		},
		{
			name:                    "failing install pre install query",
			uuid:                    "fail_pre_install_query",
			expectedStatus:          fleet.SoftwareInstallerFailed,
			preInstallQueryOutput:   ptr.String(""),
			postInstallScriptOutput: ptr.String("post install output"),
			installScriptOutput:     ptr.String("install output"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// create a host and software installer
			swFilename := "file_" + tc.name + ".pkg"
			installerID, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
				Title:         "foo" + tc.name,
				Source:        "bar" + tc.name,
				InstallScript: "echo " + tc.name,
				TeamID:        &teamID,
				Filename:      swFilename,
			})
			require.NoError(t, err)
			host, err := ds.NewHost(ctx, &fleet.Host{
				Hostname:      "macos-test-" + tc.name,
				ComputerName:  "macos-test-" + tc.name,
				OsqueryHostID: ptr.String("osquery-macos-" + tc.name),
				NodeKey:       ptr.String("node-key-macos-" + tc.name),
				UUID:          uuid.NewString(),
				Platform:      "darwin",
				TeamID:        &teamID,
			})
			require.NoError(t, err)

			// Need to insert manually so we have access to the UUID (it's generated in the DS method)
			query := `INSERT INTO host_software_installs (execution_id, host_id, software_installer_id, post_install_script_exit_code, install_script_exit_code, pre_install_query_output, install_script_output, post_install_script_output) VALUES (?,?,?,?,?,?,?,?)`
			_, err = ds.writer(ctx).ExecContext(ctx, query, tc.uuid, host.ID, installerID, tc.postInstallScriptEC, tc.installScriptEC, tc.preInstallQueryOutput, tc.installScriptOutput, tc.postInstallScriptOutput)
			require.NoError(t, err)

			res, err := ds.GetSoftwareInstallResults(ctx, tc.uuid)
			require.NoError(t, err)

			require.Equal(t, tc.uuid, res.InstallUUID)
			require.Equal(t, tc.expectedStatus, res.Status)
			require.Equal(t, swFilename, res.SoftwarePackage)
			require.Equal(t, host.ID, res.HostID)
			require.Equal(t, host.DisplayName(), res.HostDisplayName)
			expectedPreInstallQueryOutput := ""
			if tc.preInstallQueryOutput != nil {
				expectedPreInstallQueryOutput = *tc.preInstallQueryOutput
			}
			require.Equal(t, expectedPreInstallQueryOutput, res.PreInstallQueryOutput)

			expectedPostInstallScriptOutput := ""
			if tc.postInstallScriptOutput != nil {
				expectedPostInstallScriptOutput = *tc.postInstallScriptOutput
			}
			require.Equal(t, expectedPostInstallScriptOutput, res.PostInstallScriptOutput)
			expectedInstallScriptOutput := ""
			if tc.installScriptOutput != nil {
				expectedInstallScriptOutput = *tc.installScriptOutput
			}
			require.Equal(t, expectedInstallScriptOutput, res.Output)
		})
	}
}
