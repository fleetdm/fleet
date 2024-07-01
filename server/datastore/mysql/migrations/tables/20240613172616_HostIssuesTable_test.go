package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20240613172616(t *testing.T) {
	db := applyUpToPrev(t)

	res, err := db.Exec(
		`
    INSERT INTO policies (name, query, description, checksum)
    VALUES ('test_policy', "", "", "abc")`,
	)
	require.NoError(t, err)
	policyID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = db.Exec(
		`INSERT INTO policy_membership (policy_id, host_id, passes) VALUES (?, ?, ?)`,
		policyID, 1, 0,
	)
	require.NoError(t, err)

	applyNext(t, db)

	type issues struct {
		HostID                       uint      `db:"host_id"`
		FailingPoliciesCount         uint      `db:"failing_policies_count"`
		CriticalVulnerabilitiesCount uint      `db:"critical_vulnerabilities_count"`
		TotalIssuesCount             uint      `db:"total_issues_count"`
		CreatedAt                    time.Time `db:"created_at"`
		UpdatedAt                    time.Time `db:"updated_at"`
	}

	var result issues
	selectStmt := `SELECT * from host_issues WHERE host_id = ?`
	err = db.Get(&result, selectStmt, 1)
	require.NoError(t, err)
	assert.Equal(t, uint(1), result.HostID)
	assert.Equal(t, uint(1), result.FailingPoliciesCount)
	assert.Equal(t, uint(0), result.CriticalVulnerabilitiesCount)
	assert.Equal(t, uint(1), result.TotalIssuesCount)
	assert.NotZero(t, result.CreatedAt)
	assert.Equal(t, result.CreatedAt, result.UpdatedAt)

	hostID := uint(12)

	insertStmt := `INSERT INTO host_issues (host_id, failing_policies_count, critical_vulnerabilities_count, total_issues_count) VALUES (?, ?, ?, ?)`
	_, err = db.Exec(insertStmt, hostID, 1, 2, 3)
	require.NoError(t, err)
	_, err = db.Exec(insertStmt, hostID, 4, 5, 6)
	require.ErrorContains(t, err, "Error 1062")

	err = db.Get(&result, selectStmt, hostID)
	require.NoError(t, err)
	assert.Equal(t, hostID, result.HostID)
	assert.Equal(t, uint(1), result.FailingPoliciesCount)
	assert.Equal(t, uint(2), result.CriticalVulnerabilitiesCount)
	assert.Equal(t, uint(3), result.TotalIssuesCount)
	assert.NotZero(t, result.CreatedAt)
	assert.Equal(t, result.CreatedAt, result.UpdatedAt)
	created := result.CreatedAt

	time.Sleep(1 * time.Millisecond)
	_, err = db.Exec(`UPDATE host_issues SET total_issues_count = 4 WHERE host_id = ?`, hostID)
	require.NoError(t, err)

	result = issues{}
	err = db.Get(&result, selectStmt, hostID)
	require.NoError(t, err)
	assert.Equal(t, hostID, result.HostID)
	assert.Equal(t, uint(1), result.FailingPoliciesCount)
	assert.Equal(t, uint(2), result.CriticalVulnerabilitiesCount)
	assert.Equal(t, uint(4), result.TotalIssuesCount)
	assert.Equal(t, created, result.CreatedAt)
	assert.Greater(t, result.UpdatedAt, result.CreatedAt)
}
