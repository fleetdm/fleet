package datastore

import (
	"testing"

	"github.com/kolide/kolide/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testYARAStore(t *testing.T, ds kolide.Datastore) {
	ysg := &kolide.YARASignatureGroup{
		SignatureName: "sig1",
		Paths: []string{
			"path1",
			"path2",
		},
	}
	ysg, err := ds.NewYARASignatureGroup(ysg)
	require.Nil(t, err)
	require.True(t, ysg.ID > 0)
	fp := &kolide.FIMSection{
		SectionName: "fp1",
		Paths: []string{
			"path1",
			"path2",
			"path3",
		},
	}
	fp, err = ds.NewFIMSection(fp)
	require.Nil(t, err)
	assert.True(t, fp.ID > 0)

	err = ds.NewYARAFilePath("fp1", "sig1")
	require.Nil(t, err)
	yaraSection, err := ds.YARASection()
	require.Nil(t, err)
	require.Len(t, yaraSection.FilePaths, 1)
	assert.Len(t, yaraSection.FilePaths["fp1"], 1)
	require.Len(t, yaraSection.Signatures, 1)
	assert.Len(t, yaraSection.Signatures["sig1"], 2)
	ysg = &kolide.YARASignatureGroup{
		SignatureName: "sig2",
		Paths: []string{
			"path3",
		},
	}
	ysg, err = ds.NewYARASignatureGroup(ysg)
	require.Nil(t, err)
	yaraSection, err = ds.YARASection()
	require.Nil(t, err)
	assert.Len(t, yaraSection.Signatures["sig2"], 1)
}
