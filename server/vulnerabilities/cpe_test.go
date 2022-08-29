package vulnerabilities

import (
	"compress/gzip"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/facebookincubator/nvdtools/cpedict"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCPEFromSoftware(t *testing.T) {
	tempDir := t.TempDir()

	items, err := cpedict.Decode(strings.NewReader(XmlCPETestDict))
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "cpe.sqlite")

	err = GenerateCPEDB(dbPath, items)
	require.NoError(t, err)

	db, err := sqliteDB(dbPath)
	require.NoError(t, err)

	reCache := newRegexpCache()

	// checking an non existent version returns empty
	cpe, err := CPEFromSoftware(db, &fleet.Software{Name: "Vendor Product-1.app", Version: "2.3.4", BundleIdentifier: "vendor", Source: "apps"}, nil, reCache)
	require.NoError(t, err)
	require.Equal(t, "", cpe)

	// checking a version that exists works
	cpe, err = CPEFromSoftware(db, &fleet.Software{Name: "Vendor Product-1.app", Version: "1.2.3", BundleIdentifier: "vendor", Source: "apps"}, nil, reCache)
	require.NoError(t, err)
	require.Equal(t, "cpe:2.3:a:vendor:product-1:1.2.3:*:*:*:*:macos:*:*", cpe)

	// follows many deprecations
	cpe, err = CPEFromSoftware(db, &fleet.Software{Name: "Vendor2 Product2.app", Version: "0.3", BundleIdentifier: "vendor2", Source: "apps"}, nil, reCache)
	require.NoError(t, err)
	require.Equal(t, "cpe:2.3:a:vendor2:product4:999:*:*:*:*:macos:*:*", cpe)
}

func TestCPETranslations(t *testing.T) {
	tempDir := t.TempDir()

	items, err := cpedict.Decode(strings.NewReader(XmlCPETestDict))
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "cpe.sqlite")

	err = GenerateCPEDB(dbPath, items)
	require.NoError(t, err)

	db, err := sqliteDB(dbPath)
	require.NoError(t, err)

	translations := CPETranslations{
		{
			Match: CPETranslationMatch{ // (name = X OR Y) AND (source = apps)
				Name:   []string{"X", "Y"},
				Source: []string{"apps"},
			},
			Filter: CPETranslationFilter{
				Product: []string{"product-1"},
				Vendor:  []string{"vendor"},
			},
		},
	}

	tt := []struct {
		Name         string
		Translations CPETranslations
		Software     *fleet.Software
		Expected     string
	}{
		{
			Name: "simple match",
			Translations: CPETranslations{
				{
					Match: CPETranslationMatch{
						Name:   []string{"X"},
						Source: []string{"apps"},
					},
					Filter: CPETranslationFilter{
						Product: []string{"product-1"},
						Vendor:  []string{"vendor"},
					},
				},
			},
			Software: &fleet.Software{
				Name:    "X",
				Version: "1.2.3",
				Source:  "apps",
			},
			Expected: "cpe:2.3:a:vendor:product-1:1.2.3:*:*:*:*:macos:*:*",
		},
		{
			Name: "match name or",
			Translations: CPETranslations{
				{
					Match: CPETranslationMatch{
						Name:   []string{"X", "Y"},
						Source: []string{"apps"},
					},
					Filter: CPETranslationFilter{
						Product: []string{"product-1"},
						Vendor:  []string{"vendor"},
					},
				},
			},
			Software: &fleet.Software{
				Name:    "Y",
				Version: "1.2.3",
				Source:  "apps",
			},
			Expected: "cpe:2.3:a:vendor:product-1:1.2.3:*:*:*:*:macos:*:*",
		},
		{
			Name: "match name regexp",
			Translations: CPETranslations{
				{
					Match: CPETranslationMatch{
						Name:   []string{"/^[A-Z]$/"},
						Source: []string{"apps"},
					},
					Filter: CPETranslationFilter{
						Product: []string{"product-1"},
						Vendor:  []string{"vendor"},
					},
				},
			},
			Software: &fleet.Software{
				Name:    "Z",
				Version: "1.2.3",
				Source:  "apps",
			},
			Expected: "cpe:2.3:a:vendor:product-1:1.2.3:*:*:*:*:macos:*:*",
		},
	}

	reCache := newRegexpCache()

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			cpe, err := CPEFromSoftware(db, tc.Software, translations, reCache)
			require.NoError(t, err)
			require.Equal(t, tc.Expected, cpe)
		})
	}
}

func TestSyncCPEDatabase(t *testing.T) {
	nettest.Run(t)

	client := fleethttp.NewClient()

	tempDir := t.TempDir()

	// first time, db doesn't exist, so it downloads
	err := DownloadCPEDB(tempDir, client, "")
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "cpe.sqlite")
	db, err := sqliteDB(dbPath)
	require.NoError(t, err)

	// and this works afterwards
	reCache := newRegexpCache()

	software := &fleet.Software{Name: "1Password.app",
		Version:          "7.2.3",
		BundleIdentifier: "com.1password.1password",
		Source:           "apps",
	}
	cpe, err := CPEFromSoftware(db, software, nil, reCache)
	require.NoError(t, err)
	require.Equal(t, "cpe:2.3:a:1password:1password:7.2.3:beta0:*:*:*:macos:*:*", cpe)

	npmCPE, err := CPEFromSoftware(db, &fleet.Software{Name: "Adaltas Mixme 0.4.0 for Node.js", Version: "0.4.0", Source: "npm_packages"}, nil, reCache)
	require.NoError(t, err)
	assert.Equal(t, "cpe:2.3:a:adaltas:mixme:0.4.0:*:*:*:*:node.js:*:*", npmCPE)

	windowsCPE, err := CPEFromSoftware(db, &fleet.Software{Name: "HP Storage Data Protector 8.0 for Windows 8", Version: "8.0", Source: "programs"}, nil, reCache)
	require.NoError(t, err)
	assert.Equal(t, "cpe:2.3:a:hp:storage_data_protector:8.0:-:*:*:*:windows_7:*:*", windowsCPE)

	// but now we truncate to make sure searching for cpe fails
	err = os.Truncate(dbPath, 0)
	require.NoError(t, err)
	_, err = CPEFromSoftware(db, software, nil, reCache)
	require.Error(t, err)

	// and we make the db older than the release
	newTime := time.Date(2000, 1, 1, 1, 1, 1, 1, time.UTC)
	err = os.Chtimes(dbPath, newTime, newTime)
	require.NoError(t, err)

	// then it will download
	err = DownloadCPEDB(tempDir, client, "")
	require.NoError(t, err)

	// let's register the mtime for the db
	stat, err := os.Stat(dbPath)
	require.NoError(t, err)
	mtime := stat.ModTime()

	db.Close()
	db, err = sqliteDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	cpe, err = CPEFromSoftware(db, software, nil, reCache)
	require.NoError(t, err)
	require.Equal(t, "cpe:2.3:a:1password:1password:7.2.3:beta0:*:*:*:macos:*:*", cpe)

	// let some time pass
	time.Sleep(2 * time.Second)

	// let's check it doesn't download because it's new enough
	err = DownloadCPEDB(tempDir, client, "")
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
	nettest.Run(t)

	tempDir := t.TempDir()

	ds := new(mock.Store)

	var cpes []string

	ds.AddCPEForSoftwareFunc = func(ctx context.Context, software fleet.Software, cpe string) error {
		cpes = append(cpes, cpe)
		return nil
	}

	iterator := &fakeSoftwareIterator{
		softwares: []*fleet.Software{
			{
				ID:               1,
				Name:             "Product",
				Version:          "1.2.3",
				BundleIdentifier: "vendor",
				Source:           "apps",
			},
			{
				ID:               2,
				Name:             "Product2",
				Version:          "0.3",
				BundleIdentifier: "vendor2",
				Source:           "apps",
			},
		},
	}

	ds.AllSoftwareWithoutCPEIteratorFunc = func(ctx context.Context, excludedPlatforms []string) (fleet.SoftwareIterator, error) {
		return iterator, nil
	}

	items, err := cpedict.Decode(strings.NewReader(XmlCPETestDict))
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "cpe.sqlite")
	err = GenerateCPEDB(dbPath, items)
	require.NoError(t, err)

	err = TranslateSoftwareToCPE(context.Background(), ds, tempDir, kitlog.NewNopLogger())
	require.NoError(t, err)
	assert.Equal(t, []string{
		"cpe:2.3:a:vendor:product-1:1.2.3:*:*:*:*:macos:*:*",
		"cpe:2.3:a:vendor2:product4:999:*:*:*:*:macos:*:*",
	}, cpes)
	assert.True(t, iterator.closed)
}

func TestSyncsCPEFromURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zw := gzip.NewWriter(w)

		_, err := zw.Write([]byte("Hello world!"))
		require.NoError(t, err)
		err = zw.Close()
		require.NoError(t, err)
	}))
	defer ts.Close()

	client := fleethttp.NewClient()
	tempDir := t.TempDir()
	err := DownloadCPEDB(tempDir, client, ts.URL+"/hello-world.gz")
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "cpe.sqlite")
	stored, err := ioutil.ReadFile(dbPath)
	require.NoError(t, err)
	assert.Equal(t, "Hello world!", string(stored))
}

func TestLegacyCPEDB(t *testing.T) {

	// Older versions of fleet used "select * ..." when querying from the cpe and cpe_search tables
	// Ensure that this still works when generating the new cpe database.
	type IndexedCPEItem struct {
		ID         int     `json:"id" db:"rowid"`
		Title      string  `json:"title" db:"title"`
		Version    *string `json:"version" db:"version"`
		TargetSW   *string `json:"target_sw" db:"target_sw"`
		CPE23      string  `json:"cpe23" db:"cpe23"`
		Deprecated bool    `json:"deprecated" db:"deprecated"`
	}
	tempDir := t.TempDir()

	items, err := cpedict.Decode(strings.NewReader(XmlCPETestDict))
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "cpe.sqlite")

	err = GenerateCPEDB(dbPath, items)
	require.NoError(t, err)

	db, err := sqliteDB(dbPath)
	require.NoError(t, err)

	query := `SELECT rowid, * FROM cpe WHERE rowid in (
				  SELECT rowid FROM cpe_search WHERE title MATCH ?
				) and version = ? order by deprecated asc`

	var indexedCPEs []IndexedCPEItem
	err = db.Select(&indexedCPEs, query, "product", "1.2.3")
	require.NoError(t, err)

	require.Len(t, indexedCPEs, 1)
}
