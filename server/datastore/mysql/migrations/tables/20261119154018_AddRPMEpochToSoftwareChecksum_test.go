package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20261119154018(t *testing.T) {
	db := applyUpToPrev(t)

	prevBatchSize := rpmEpochChecksumBatchSize
	rpmEpochChecksumBatchSize = 2
	t.Cleanup(func() { rpmEpochChecksumBatchSize = prevBatchSize })

	rpm := func(name string, epoch *uint32) fleet.Software {
		return fleet.Software{
			Name: name, Version: "1.0", Release: "1.el9", Arch: "x86_64", Vendor: "Rocky", Source: "rpm_packages",
			Epoch: epoch,
		}
	}
	checksum := func(sw fleet.Software) []byte {
		sum, err := sw.ComputeRawChecksum()
		require.NoError(t, err)
		return sum
	}
	insertSoftware := func(sw fleet.Software, storedChecksum []byte) int64 {
		return execNoErrLastID(t, db,
			"INSERT INTO software (name, version, source, `release`, vendor, arch, checksum) VALUES (?, ?, ?, ?, ?, ?, ?)",
			sw.Name, sw.Version, sw.Source, sw.Release, sw.Vendor, sw.Arch, storedChecksum)
	}

	// Rows created while the epoch was not part of the checksum, so the checksum is that of the
	// software without its epoch.
	withEpoch := make(map[int64]fleet.Software)
	for _, sw := range []fleet.Software{
		rpm("tomcat", new(uint32(1))),
		rpm("mod_ssl", new(uint32(1))),
		rpm("bind-libs", new(uint32(32))),
	} {
		id := insertSoftware(sw, checksum(rpm(sw.Name, nil)))
		execNoErr(t, db, `INSERT INTO software_metadata (software_id, epoch) VALUES (?, ?)`, id, *sw.Epoch)
		withEpoch[id] = sw
	}
	withoutEpoch := rpm("httpd", nil)
	withoutEpochID := insertSoftware(withoutEpoch, checksum(withoutEpoch))

	// Epoch 0 is the same as no epoch, so the checksum stays.
	epoch0 := rpm("curl", nil)
	epoch0ID := insertSoftware(epoch0, checksum(epoch0))
	execNoErr(t, db, `INSERT INTO software_metadata (software_id, epoch) VALUES (?, 0)`, epoch0ID)

	// A name ending in a NUL and an epoch, which a host could report before reported fields were
	// checked for NUL, doesn't take the checksum the real package gets.
	forged := rpm("tomcat\x00epoch:1", nil)
	forgedID := insertSoftware(forged, checksum(forged))

	applyNext(t, db)

	storedChecksum := func(id int64) []byte {
		var sum []byte
		require.NoError(t, db.Get(&sum, `SELECT checksum FROM software WHERE id = ?`, id))
		return sum
	}
	// Rows with a stored epoch now have the checksum ingestion computes for the software reported
	// with that epoch, so hosts keep linking to the same software IDs.
	for id, sw := range withEpoch {
		require.Equal(t, checksum(sw), storedChecksum(id), sw.Name)
	}
	require.Equal(t, checksum(withoutEpoch), storedChecksum(withoutEpochID))
	require.Equal(t, checksum(epoch0), storedChecksum(epoch0ID))
	require.Equal(t, checksum(forged), storedChecksum(forgedID))
}
