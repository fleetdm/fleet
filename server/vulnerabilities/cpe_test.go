package vulnerabilities

import (
	"errors"
	"net/http"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/dnaeon/go-vcr/v2/recorder"
	"github.com/facebookincubator/nvdtools/cpedict"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCpeFromSoftware(t *testing.T) {
	tempDir := os.TempDir()

	items, err := cpedict.Decode(strings.NewReader(XmlCPETestDict))
	require.NoError(t, err)

	dbPath := path.Join(tempDir, "cpe.sqlite")
	err = GenerateCPEDB(dbPath, items)
	require.NoError(t, err)

	db, err := sqliteDB(dbPath)
	require.NoError(t, err)

	// checking an non existent version returns empty
	cpe, err := CPEFromSoftware(db, &fleet.Software{Name: "Vendor Product.app", Version: "2.3.4", Source: "apps"})
	require.NoError(t, err)
	require.Equal(t, "", cpe)

	// checking a version that exists works
	cpe, err = CPEFromSoftware(db, &fleet.Software{Name: "Vendor Product.app", Version: "1.2.3", Source: "apps"})
	require.NoError(t, err)
	require.Equal(t, "cpe:2.3:a:vendor:product:1.2.3:*:*:*:*:macos:*:*", cpe)

	// follows many deprecations
	cpe, err = CPEFromSoftware(db, &fleet.Software{Name: "Vendor2 Product2.app", Version: "0.3", Source: "apps"})
	require.NoError(t, err)
	require.Equal(t, "cpe:2.3:a:vendor2:product4:999:*:*:*:*:macos:*:*", cpe)
}

func TestSyncCPEDatabase(t *testing.T) {
	// Disabling vcr because the resulting file exceeds the 100mb limit for github
	r, err := recorder.NewAsMode("fixtures/nvd-cpe-release", recorder.ModeDisabled, http.DefaultTransport)
	require.NoError(t, err)
	defer r.Stop()

	client := &http.Client{
		Transport: r,
	}

	tempDir := os.TempDir()
	dbPath := path.Join(tempDir, "cpe.sqlite")

	err = os.Remove(dbPath)
	if !errors.Is(err, os.ErrNotExist) {
		require.NoError(t, err)
	}

	// first time, db doesn't exist, so it downloads
	err = syncCPEDatabase(client, dbPath)
	require.NoError(t, err)

	db, err := sqliteDB(dbPath)
	require.NoError(t, err)

	// and this works afterwards
	software := &fleet.Software{Name: "1Password.app", Version: "7.2.3", Source: "apps"}
	cpe, err := CPEFromSoftware(db, software)
	require.NoError(t, err)
	require.Equal(t, "cpe:2.3:a:1password:1password:7.2.3:beta0:*:*:*:macos:*:*", cpe)

	// but now we truncate to make sure searching for cpe fails
	err = os.Truncate(dbPath, 0)
	require.NoError(t, err)
	_, err = CPEFromSoftware(db, software)
	require.Error(t, err)

	// and we make the db older than the release
	newTime := time.Date(2000, 01, 01, 01, 01, 01, 01, time.UTC)
	err = os.Chtimes(dbPath, newTime, newTime)
	require.NoError(t, err)

	// then it will download
	err = syncCPEDatabase(client, dbPath)
	require.NoError(t, err)

	// let's register the mtime for the db
	stat, err := os.Stat(dbPath)
	require.NoError(t, err)
	mtime := stat.ModTime()

	db.Close()
	db, err = sqliteDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	cpe, err = CPEFromSoftware(db, software)
	require.NoError(t, err)
	require.Equal(t, "cpe:2.3:a:1password:1password:7.2.3:beta0:*:*:*:macos:*:*", cpe)

	// let some time pass
	time.Sleep(2 * time.Second)

	// let's check it doesn't download because it's new enough
	err = syncCPEDatabase(client, dbPath)
	require.NoError(t, err)

	stat, err = os.Stat(dbPath)
	require.NoError(t, err)
	require.Equal(t, mtime, stat.ModTime())
}

type fakeSoftwareIterator struct {
	index     int
	softwares []*fleet.Software
	closed    bool
}

func (f *fakeSoftwareIterator) Next() bool {
	return f.index < len(f.softwares)
}

func (f *fakeSoftwareIterator) Value() (*fleet.Software, error) {
	s := f.softwares[f.index]
	f.index++
	return s, nil
}

func (f *fakeSoftwareIterator) Err() error   { return nil }
func (f *fakeSoftwareIterator) Close() error { f.closed = true; return nil }

func TestTranslateSoftwareToCPE(t *testing.T) {
	tempDir, err := os.MkdirTemp(os.TempDir(), "TestTranslateSoftwareToCPE-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ds := new(mock.Store)

	var cpes []string

	ds.AddCPEForSoftwareFunc = func(software fleet.Software, cpe string) error {
		cpes = append(cpes, cpe)
		return nil
	}

	iterator := &fakeSoftwareIterator{
		softwares: []*fleet.Software{
			{
				ID:      1,
				Name:    "Product",
				Version: "1.2.3",
				Source:  "apps",
			},
			{
				ID:      2,
				Name:    "Product2",
				Version: "0.3",
				Source:  "apps",
			},
		},
	}

	ds.AllSoftwareWithoutCPEIteratorFunc = func() (fleet.SoftwareIterator, error) {
		return iterator, nil
	}

	items, err := cpedict.Decode(strings.NewReader(XmlCPETestDict))
	require.NoError(t, err)

	dbPath := path.Join(tempDir, "cpe.sqlite")
	err = GenerateCPEDB(dbPath, items)
	require.NoError(t, err)

	err = TranslateSoftwareToCPE(ds, tempDir)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"cpe:2.3:a:vendor:product:1.2.3:*:*:*:*:macos:*:*",
		"cpe:2.3:a:vendor2:product4:999:*:*:*:*:macos:*:*",
	}, cpes)
	assert.True(t, iterator.closed)
}
