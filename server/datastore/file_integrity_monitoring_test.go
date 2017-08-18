package datastore

import (
	"testing"

	"github.com/kolide/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testFileIntegrityMonitoring(t *testing.T, ds kolide.Datastore) {
	fp := &kolide.FIMSection{
		SectionName: "fp1",
		Paths: []string{
			"path1",
			"path2",
			"path3",
		},
	}
	fp, err := ds.NewFIMSection(fp)
	require.Nil(t, err)
	assert.True(t, fp.ID > 0)
	fp = &kolide.FIMSection{
		SectionName: "fp2",
		Paths: []string{
			"path4",
			"path5",
		},
	}
	_, err = ds.NewFIMSection(fp)
	require.Nil(t, err)

	actual, err := ds.FIMSections()
	require.Nil(t, err)
	assert.Len(t, actual, 2)
	assert.Len(t, actual["fp1"], 3)
	assert.Len(t, actual["fp2"], 2)

	err = ds.ClearFIMSections()
	require.Nil(t, err)
	fs, err := ds.FIMSections()
	assert.Nil(t, err)
	assert.Len(t, fs, 0)

}
