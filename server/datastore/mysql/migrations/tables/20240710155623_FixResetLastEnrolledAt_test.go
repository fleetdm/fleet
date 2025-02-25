package tables

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20240710155623(t *testing.T) {
	db := applyUpToPrev(t)

	i := uint(1)
	newHost := func(platform, lastEnrolledAt string, hostDisk bool) uint {
		id := fmt.Sprintf("%d", i)
		i++
		hostID := uint(execNoErrLastID(t, db, //nolint:gosec // dismiss G115
			`INSERT INTO hosts (osquery_host_id, node_key, uuid, platform, last_enrolled_at) VALUES (?, ?, ?, ?, ?);`,
			id, id, id, platform, lastEnrolledAt,
		))
		if hostDisk {
			execNoErr(t, db,
				`INSERT INTO host_disks (host_id) VALUES (?);`,
				hostID,
			)
		}
		return hostID
	}
	neverDate := "2000-01-01 00:00:00"
	ubuntuHostID := newHost("ubuntu", neverDate, true)                 // non-darwin hosts should not be updated.
	validMacOSHostID := newHost("darwin", "2024-07-08 18:00:53", true) // host without the issue, should not be updated.
	pendingMacOSDEPHostID := newHost("darwin", neverDate, false)       // host without the issue (e.g. DEP pending, not enrolled), should not be updated.
	invalidMacOSHostID := newHost("darwin", neverDate, true)           // host with the issue, should be updated.

	const getColumnsQuery = `
		SELECT h.last_enrolled_at, h.updated_at, hd.created_at AS host_disks_created_at
		FROM hosts h LEFT JOIN host_disks hd ON h.id=hd.host_id WHERE h.id = ?;`
	type hostTimestamps struct {
		LastEnrolledAt     time.Time  `db:"last_enrolled_at"`
		UpdatedAt          time.Time  `db:"updated_at"`
		HostDisksCreatedAt *time.Time `db:"host_disks_created_at"`
	}
	var ubuntuTimestampsBefore hostTimestamps
	err := db.Get(&ubuntuTimestampsBefore, getColumnsQuery, ubuntuHostID)
	require.NoError(t, err)
	require.NotZero(t, ubuntuTimestampsBefore.UpdatedAt)
	require.Equal(t, ubuntuTimestampsBefore.LastEnrolledAt.Format("2006-01-02 15:04:05"), neverDate)
	require.NotNil(t, ubuntuTimestampsBefore.HostDisksCreatedAt)
	require.NotZero(t, *ubuntuTimestampsBefore.HostDisksCreatedAt)
	var validMacOSTimestampsBefore hostTimestamps
	err = db.Get(&validMacOSTimestampsBefore, getColumnsQuery, validMacOSHostID)
	require.NoError(t, err)
	require.NotZero(t, validMacOSTimestampsBefore.UpdatedAt)
	require.Equal(t, validMacOSTimestampsBefore.LastEnrolledAt.Format("2006-01-02 15:04:05"), "2024-07-08 18:00:53")
	require.NotNil(t, validMacOSTimestampsBefore.HostDisksCreatedAt)
	require.NotZero(t, *validMacOSTimestampsBefore.HostDisksCreatedAt)
	var pendingMacOSDEPTimestampsBefore hostTimestamps
	err = db.Get(&pendingMacOSDEPTimestampsBefore, getColumnsQuery, pendingMacOSDEPHostID)
	require.NoError(t, err)
	require.NotZero(t, pendingMacOSDEPTimestampsBefore.UpdatedAt)
	require.Equal(t, pendingMacOSDEPTimestampsBefore.LastEnrolledAt.Format("2006-01-02 15:04:05"), neverDate)
	require.Nil(t, pendingMacOSDEPTimestampsBefore.HostDisksCreatedAt)
	var invalidMacOSTimestampsBefore hostTimestamps
	err = db.Get(&invalidMacOSTimestampsBefore, getColumnsQuery, invalidMacOSHostID)
	require.NoError(t, err)
	require.NotZero(t, invalidMacOSTimestampsBefore.UpdatedAt)
	require.Equal(t, invalidMacOSTimestampsBefore.LastEnrolledAt.Format("2006-01-02 15:04:05"), neverDate)
	require.NotNil(t, invalidMacOSTimestampsBefore.HostDisksCreatedAt)
	require.NotZero(t, *invalidMacOSTimestampsBefore.HostDisksCreatedAt)

	// Apply current migration.
	applyNext(t, db)

	var ubuntuTimestampsAfter hostTimestamps
	err = db.Get(&ubuntuTimestampsAfter, getColumnsQuery, ubuntuHostID)
	require.NoError(t, err)
	require.Equal(t, ubuntuTimestampsBefore, ubuntuTimestampsAfter)
	var validMacOSTimestampsAfter hostTimestamps
	err = db.Get(&validMacOSTimestampsAfter, getColumnsQuery, validMacOSHostID)
	require.NoError(t, err)
	require.Equal(t, validMacOSTimestampsBefore, validMacOSTimestampsAfter)
	var pendingMacOSDEPTimestampsAfter hostTimestamps
	err = db.Get(&pendingMacOSDEPTimestampsAfter, getColumnsQuery, pendingMacOSDEPHostID)
	require.NoError(t, err)
	require.Equal(t, pendingMacOSDEPTimestampsBefore.UpdatedAt, pendingMacOSDEPTimestampsAfter.UpdatedAt)    // updated_at is unmodified
	require.Equal(t, neverDate, pendingMacOSDEPTimestampsAfter.LastEnrolledAt.Format("2006-01-02 15:04:05")) // last_enrolled_at was not modified
	var invalidMacOSTimestampsAfter hostTimestamps
	err = db.Get(&invalidMacOSTimestampsAfter, getColumnsQuery, invalidMacOSHostID)
	require.NoError(t, err)
	require.Equal(t, invalidMacOSTimestampsBefore.UpdatedAt, invalidMacOSTimestampsAfter.UpdatedAt) // updated_at is unmodified
	require.NotNil(t, invalidMacOSTimestampsAfter.HostDisksCreatedAt)
	require.Equal(t, *invalidMacOSTimestampsAfter.HostDisksCreatedAt, invalidMacOSTimestampsAfter.LastEnrolledAt) // last_enrolled_at was updated to host_disks date
}
