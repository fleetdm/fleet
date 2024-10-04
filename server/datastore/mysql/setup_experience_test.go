package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestSetupExperience(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"ListSetupExperienceStatusResults", testSetupExperienceStatusResults},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testSetupExperienceStatusResults(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	hostUUID := uuid.NewString()
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname: "host1",
		UUID:     hostUUID,
		Platform: "darwin",
	})
	require.NoError(t, err)
	user, err := ds.NewUser(ctx, &fleet.User{Name: "Foo", Email: "foo@example.com", GlobalRole: ptr.String("admin"), Password: []byte("12characterslong!")})
	require.NoError(t, err)
	id, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{Filename: "test.app", Version: "1.0.0", UserID: user.ID})
	require.NoError(t, err)
	_, err = ds.InsertSoftwareInstallRequest(ctx, host.ID, id, false)
	require.NoError(t, err)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO setup_experience_status_results (host_uuid, type, name, status, host_software_installs_id) VALUES (?, ?, ?, ?, ?)`,
			hostUUID, fleet.TypeSoftwareInstall, "software", fleet.StatusPending, 1)
		require.NoError(t, err)
		return nil
	})

	cmdUUID := uuid.NewString()
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO setup_experience_status_results (host_uuid, type, name, status, nano_command_uuid) VALUES (?, ?, ?, ?, ?)`,
			hostUUID, fleet.TypeBootstrapPackage, "bootstrap", fleet.StatusPending, cmdUUID)
		require.NoError(t, err)
		return nil
	})

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO setup_experience_status_results (host_uuid, type, name, status, script_execution_id) VALUES (?, ?, ?, ?, ?)`,
			hostUUID, fleet.TypePostInstallScript, "script", fleet.StatusPending, 1)
		require.NoError(t, err)
		return nil
	})

	res, err := ds.ListSetupExperienceResultsByHostUUID(ctx, hostUUID)
	require.NoError(t, err)
	require.Len(t, res, 3)
	r := res[0]
	require.Equal(t, hostUUID, r.HostUUID)
	require.Equal(t, fleet.TypeSoftwareInstall, r.Type)
	require.Equal(t, "software", r.Name)
	require.Equal(t, fleet.StatusPending, r.Status)
	require.Equal(t, ptr.Uint(uint(1)), r.HostSoftwareInstallsID)
	require.Nil(t, r.Error)
}
