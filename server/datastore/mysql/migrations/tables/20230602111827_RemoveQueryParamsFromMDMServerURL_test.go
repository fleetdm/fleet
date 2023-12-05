package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230602111827(t *testing.T) {
	db := applyUpToPrev(t)
	insertMDMSolutionStmt := `INSERT INTO mobile_device_management_solutions (id, name, server_url) VALUES (?, ?, ?)`
	_, err := db.Exec(insertMDMSolutionStmt, 1, "foo", "https://test.example.com?test=1")
	require.NoError(t, err)
	_, err = db.Exec(insertMDMSolutionStmt, 2, "foo", "https://test.example.com")
	require.NoError(t, err)
	_, err = db.Exec(insertMDMSolutionStmt, 3, "bar", "https://test.example.com/abc")
	require.NoError(t, err)
	_, err = db.Exec(insertMDMSolutionStmt, 4, "bar", "https://test.example.com/abc?test=1")
	require.NoError(t, err)
	_, err = db.Exec(insertMDMSolutionStmt, 5, "baz", "https://foo.bar.com")
	require.NoError(t, err)

	insertHostMDMStmt := `INSERT INTO host_mdm (host_id, server_url, mdm_id) VALUES (?, ?, ?)`
	_, err = db.Exec(insertHostMDMStmt, 1, "https://test.example.com?test=1", 1)
	require.NoError(t, err)
	_, err = db.Exec(insertHostMDMStmt, 2, "https://test.example.com", 2)
	require.NoError(t, err)
	_, err = db.Exec(insertHostMDMStmt, 3, "https://test.example.com/abc", 3)
	require.NoError(t, err)
	_, err = db.Exec(insertHostMDMStmt, 4, "https://test.example.com/abc?test=1", 4)
	require.NoError(t, err)
	_, err = db.Exec(insertHostMDMStmt, 5, "https://foo.bar.com", 5)
	require.NoError(t, err)
	_, err = db.Exec(insertHostMDMStmt, 6, "https://test.example.com?test=1", 1)
	require.NoError(t, err)
	_, err = db.Exec(insertHostMDMStmt, 7, "https://test.example.com", 2)
	require.NoError(t, err)

	applyNext(t, db)

	type hostMDM struct {
		ServerURL string `db:"server_url"`
		MDMID     uint   `db:"mdm_id"`
	}
	var hostMDMs []hostMDM
	err = db.Select(&hostMDMs, "SELECT server_url, mdm_id FROM host_mdm GROUP BY server_url, mdm_id")
	require.NoError(t, err)
	require.Len(t, hostMDMs, 3)
	require.ElementsMatch(t, []hostMDM{
		{"https://test.example.com", 1},
		{"https://test.example.com/abc", 3},
		{"https://foo.bar.com", 5},
	}, hostMDMs)

	var mdmSolutions []string
	err = db.Select(&mdmSolutions, "SELECT server_url FROM mobile_device_management_solutions")
	require.NoError(t, err)
	require.Len(t, mdmSolutions, 3)
	require.ElementsMatch(t, []string{"https://test.example.com", "https://test.example.com/abc", "https://foo.bar.com"}, mdmSolutions)
}
