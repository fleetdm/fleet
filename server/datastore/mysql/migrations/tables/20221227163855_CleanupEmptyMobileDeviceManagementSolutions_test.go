package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20221227163855(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `
INSERT INTO hosts
	(id, hostname, osquery_host_id)
	VALUES
		(1, 'foo', 'foo'),
		(2, 'bar', 'bar'),
		(3, 'zoo', 'zoo'),
		(4, 'no-mdm', 'no-mdm'),
		(5, 'with-valid-mdm', 'with-valid-mdm');`)
	execNoErr(t, db, `
INSERT INTO mobile_device_management_solutions
	(id, name, server_url)
	VALUES
		(1, 'foo', ''),
		(2, '', ''),
		(3, 'valid-name', 'valid.example.com'),
		(4, '', 'foo.example.com');`)
	execNoErr(t, db, `
INSERT INTO host_mdm
	(host_id, enrolled, server_url, installed_from_dep, mdm_id, is_server)
	VALUES
		(1, true, 'foo.example.com', true, 1, true),
		(2, true, '', false, 2, true),
		(3, true, '', true, 2, true),
		(5, true, 'valid.example.com', true, 3, true);`)

	applyNext(t, db)

	var solutions []fleet.AggregatedMDMSolutions
	err := db.Select(&solutions, `SELECT id, name, server_url FROM mobile_device_management_solutions;`)
	require.NoError(t, err)
	require.Len(t, solutions, 1)
	require.Equal(t, solutions[0].ServerURL, "valid.example.com")
	require.Equal(t, solutions[0].Name, "valid-name")

	var mdmHosts []fleet.HostMDM
	err = db.Select(&mdmHosts, `SELECT host_id, mdm_id FROM host_mdm;`)
	require.NoError(t, err)
	require.Len(t, mdmHosts, 1)
	require.Equal(t, mdmHosts[0].HostID, uint(5))
	require.NotNil(t, mdmHosts[0].MDMID)
	require.Equal(t, *mdmHosts[0].MDMID, uint(3))
}
