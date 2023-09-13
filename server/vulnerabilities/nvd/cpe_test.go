package nvd

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

	// checking a version that exists works
	cpe, err := CPEFromSoftware(db, &fleet.Software{Name: "Vendor Product-1.app", Version: "1.2.3", BundleIdentifier: "vendor", Source: "apps"}, nil, reCache)
	require.NoError(t, err)
	require.Equal(t, "cpe:2.3:a:vendor:product-1:1.2.3:*:*:*:*:macos:*:*", cpe)

	// follows many deprecations
	cpe, err = CPEFromSoftware(db, &fleet.Software{Name: "Vendor2 Product2.app", Version: "0.3", BundleIdentifier: "vendor2", Source: "apps"}, nil, reCache)
	require.NoError(t, err)
	require.Equal(t, "cpe:2.3:a:vendor2:product4:0.3:*:*:*:*:macos:*:*", cpe)
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
					Software: CPETranslationSoftware{
						Name:   []string{"X"},
						Source: []string{"apps"},
					},
					Filter: CPETranslation{
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
					Software: CPETranslationSoftware{
						Name:   []string{"X", "Y"},
						Source: []string{"apps"},
					},
					Filter: CPETranslation{
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
					Software: CPETranslationSoftware{
						Name:   []string{"/^[A-Z]$/"},
						Source: []string{"apps"},
					},
					Filter: CPETranslation{
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
			cpe, err := CPEFromSoftware(db, tc.Software, tc.Translations, reCache)
			require.NoError(t, err)
			require.Equal(t, tc.Expected, cpe)
		})
	}
}

func TestSyncCPEDatabase(t *testing.T) {
	nettest.Run(t)

	tempDir := t.TempDir()

	// first time, db doesn't exist, so it downloads
	err := DownloadCPEDBFromGithub(tempDir, "")
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "cpe.sqlite")
	db, err := sqliteDB(dbPath)
	require.NoError(t, err)

	// and this works afterwards
	reCache := newRegexpCache()

	software := &fleet.Software{
		Name:             "1Password.app",
		Version:          "7.2.3",
		BundleIdentifier: "com.1password.1password",
		Source:           "apps",
	}
	cpe, err := CPEFromSoftware(db, software, nil, reCache)
	require.NoError(t, err)
	require.Equal(t, "cpe:2.3:a:1password:1password:7.2.3:*:*:*:*:macos:*:*", cpe)

	npmCPE, err := CPEFromSoftware(db, &fleet.Software{Name: "Adaltas Mixme 0.4.0 for Node.js", Version: "0.4.0", Source: "npm_packages"}, nil, reCache)
	require.NoError(t, err)
	assert.Equal(t, "cpe:2.3:a:adaltas:mixme:0.4.0:*:*:*:*:node.js:*:*", npmCPE)

	windowsCPE, err := CPEFromSoftware(db, &fleet.Software{Name: "HP Storage Data Protector 8.0 for Windows 8", Version: "8.0", Source: "programs"}, nil, reCache)
	require.NoError(t, err)
	assert.Equal(t, "cpe:2.3:a:hp:storage_data_protector:8.0:*:*:*:*:windows:*:*", windowsCPE)

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
	err = DownloadCPEDBFromGithub(tempDir, "")
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
	require.Equal(t, "cpe:2.3:a:1password:1password:7.2.3:*:*:*:*:macos:*:*", cpe)

	// let some time pass
	time.Sleep(2 * time.Second)

	// let's check it doesn't download because it's new enough
	err = DownloadCPEDBFromGithub(tempDir, "")
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

func TestConsumeCPEBuffer(t *testing.T) {
	ctx := context.Background()

	t.Run("empty buffer", func(t *testing.T) {
		var upserted []fleet.SoftwareCPE
		var deleted []fleet.SoftwareCPE

		ds := new(mock.Store)
		ds.UpsertSoftwareCPEsFunc = func(ctx context.Context, cpes []fleet.SoftwareCPE) (int64, error) {
			upserted = append(upserted, cpes...)
			return int64(len(upserted)), nil
		}

		ds.DeleteSoftwareCPEsFunc = func(ctx context.Context, cpes []fleet.SoftwareCPE) (int64, error) {
			deleted = append(deleted, cpes...)
			return int64(len(deleted)), nil
		}
		err := consumeCPEBuffer(ctx, ds, nil, nil)
		require.NoError(t, err)
		require.Empty(t, upserted)
		require.Empty(t, deleted)
	})

	t.Run("inserts and deletes accordantly", func(t *testing.T) {
		var upserted []fleet.SoftwareCPE
		var deleted []fleet.SoftwareCPE

		ds := new(mock.Store)
		ds.UpsertSoftwareCPEsFunc = func(ctx context.Context, cpes []fleet.SoftwareCPE) (int64, error) {
			upserted = append(upserted, cpes...)
			return int64(len(upserted)), nil
		}

		ds.DeleteSoftwareCPEsFunc = func(ctx context.Context, cpes []fleet.SoftwareCPE) (int64, error) {
			deleted = append(deleted, cpes...)
			return int64(len(deleted)), nil
		}

		cpes := []fleet.SoftwareCPE{
			{
				SoftwareID: 1,
				CPE:        "",
			},
			{
				SoftwareID: 2,
				CPE:        "cpe-1",
			},
		}

		err := consumeCPEBuffer(ctx, ds, nil, cpes)
		require.NoError(t, err)
		require.Equal(t, len(upserted), 1)
		require.Equal(t, upserted[0].CPE, cpes[1].CPE)

		require.Equal(t, len(deleted), 1)
		require.Equal(t, deleted[0].CPE, cpes[0].CPE)
	})
}

func TestTranslateSoftwareToCPE(t *testing.T) {
	nettest.Run(t)

	tempDir := t.TempDir()

	ds := new(mock.Store)

	var cpes []string

	ds.UpsertSoftwareCPEsFunc = func(ctx context.Context, vals []fleet.SoftwareCPE) (int64, error) {
		for _, v := range vals {
			cpes = append(cpes, v.CPE)
		}
		return int64(len(vals)), nil
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
				GenerateCPE:      "something_wrong",
			},
			// For the following software entry, the matched cpe will match 'GenerateCPE', so we are
			// adding it to test that that 'UpsertSoftwareCPEs' will only be called iff software.GenerateCPE != detected CPE.
			{
				ID:               3,
				Name:             "Product2",
				Version:          "0.3",
				BundleIdentifier: "vendor2",
				Source:           "apps",
				GenerateCPE:      "cpe:2.3:a:vendor2:product4:0.3:*:*:*:*:macos:*:*",
			},
		},
	}

	ds.AllSoftwareIteratorFunc = func(ctx context.Context, q fleet.SoftwareIterQueryOptions) (fleet.SoftwareIterator, error) {
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
		"cpe:2.3:a:vendor2:product4:0.3:*:*:*:*:macos:*:*",
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

	tempDir := t.TempDir()
	err := DownloadCPEDBFromGithub(tempDir, ts.URL+"/hello-world.gz")
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

func TestCPEFromSoftwareIntegration(t *testing.T) {
	nettest.Run(t)

	testCases := []struct {
		software fleet.Software
		cpe      string
	}{
		{
			software: fleet.Software{
				Name:             "Adobe Acrobat Reader DC.app",
				Source:           "apps",
				Version:          "22.002.20191",
				Vendor:           "",
				BundleIdentifier: "com.adobe.Reader",
			},
			cpe: "cpe:2.3:a:adobe:acrobat_reader_dc:22.002.20191:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Adobe Lightroom.app",
				Source:           "apps",
				Version:          "5.5",
				Vendor:           "",
				BundleIdentifier: "com.adobe.mas.lightroomCC",
			}, cpe: "cpe:2.3:a:adobe:lightroom:5.5:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Finder.app",
				Source:           "apps",
				Version:          "12.5",
				Vendor:           "",
				BundleIdentifier: "com.apple.finder",
			}, cpe: "cpe:2.3:a:apple:finder:12.5:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Firefox.app",
				Source:           "apps",
				Version:          "105.0.1",
				Vendor:           "",
				BundleIdentifier: "org.mozilla.firefox",
			}, cpe: "cpe:2.3:a:mozilla:firefox:105.0.1:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Google Chrome.app",
				Source:           "apps",
				Version:          "105.0.5195.125",
				Vendor:           "",
				BundleIdentifier: "com.google.Chrome",
			}, cpe: "cpe:2.3:a:google:chrome:105.0.5195.125:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "iCloud.app",
				Source:           "apps",
				Version:          "1.0",
				Vendor:           "",
				BundleIdentifier: "com.apple.CloudKit.ShareBear",
			}, cpe: "cpe:2.3:a:apple:icloud:1.0:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Installer.app",
				Source:           "apps",
				Version:          "6.2.0",
				Vendor:           "",
				BundleIdentifier: "com.apple.installer",
			}, cpe: "cpe:2.3:a:apple:installer:6.2.0:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Mail.app",
				Source:           "apps",
				Version:          "16.0",
				Vendor:           "",
				BundleIdentifier: "com.apple.mail",
			}, cpe: "cpe:2.3:a:apple:mail:16.0:*:*:*:*:macos:*:*",
		},

		{
			software: fleet.Software{
				Name:             "Music.app",
				Source:           "apps",
				Version:          "1.2.5",
				Vendor:           "",
				BundleIdentifier: "com.apple.Music",
			}, cpe: "cpe:2.3:a:apple:music:1.2.5:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "OneDrive.app",
				Source:           "apps",
				Version:          "22.186.0904",
				Vendor:           "",
				BundleIdentifier: "com.microsoft.OneDrive-mac",
			}, cpe: "cpe:2.3:a:microsoft:onedrive:22.186.0904:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "osquery.app",
				Source:           "apps",
				Version:          "5.4.0",
				Vendor:           "",
				BundleIdentifier: "io.osquery.agent",
			}, cpe: "cpe:2.3:a:linuxfoundation:osquery:5.4.0:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Preview.app",
				Source:           "apps",
				Version:          "11.0",
				Vendor:           "",
				BundleIdentifier: "com.apple.Preview",
			}, cpe: "cpe:2.3:a:apple:preview:11.0:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Python.app",
				Source:           "apps",
				Version:          "3.8.9",
				Vendor:           "",
				BundleIdentifier: "com.apple.python3",
			}, cpe: "cpe:2.3:a:python:python:3.8.9:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Python.app",
				Source:           "apps",
				Version:          "3.10.7",
				Vendor:           "",
				BundleIdentifier: "org.python.python",
			}, cpe: "cpe:2.3:a:python:python:3.10.7:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Safari.app",
				Source:           "apps",
				Version:          "16.0",
				Vendor:           "",
				BundleIdentifier: "com.apple.Safari",
			}, cpe: "cpe:2.3:a:apple:safari:16.0:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Shortcuts.app",
				Source:           "apps",
				Version:          "5.0",
				Vendor:           "",
				BundleIdentifier: "com.apple.shortcuts",
			}, cpe: "cpe:2.3:a:apple:shortcuts:5.0:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Software Update.app",
				Source:           "apps",
				Version:          "6",
				Vendor:           "",
				BundleIdentifier: "com.apple.SoftwareUpdate",
			}, cpe: "cpe:2.3:a:apple:software_update:6:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Terminal.app",
				Source:           "apps",
				Version:          "2.12.7",
				Vendor:           "",
				BundleIdentifier: "com.apple.Terminal",
			}, cpe: "cpe:2.3:a:apple:terminal:2.12.7:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "TextEdit.app",
				Source:           "apps",
				Version:          "1.17",
				Vendor:           "",
				BundleIdentifier: "com.apple.TextEdit",
			}, cpe: "cpe:2.3:a:apple:textedit:1.17:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "zoom.us.app",
				Source:           "apps",
				Version:          "5.11.6 (9890)",
				Vendor:           "",
				BundleIdentifier: "us.zoom.xos",
			}, cpe: "cpe:2.3:a:zoom:meetings:5.11.6.9890:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "1Password – Password Manager",
				Source:           "chrome_extensions",
				Version:          "2.3.8",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:1password:1password:2.3.8:*:*:*:*:chrome:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Adblock Plus - free ad blocker",
				Source:           "chrome_extensions",
				Version:          "3.14.2",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:adblockplus:adblock_plus:3.14.2:*:*:*:*:chrome:*:*",
		},
		{
			software: fleet.Software{
				Name:             "AdBlock — best ad blocker",
				Source:           "chrome_extensions",
				Version:          "5.1.1",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:getadblock:adblock:5.1.1:*:*:*:*:chrome:*:*",
		},
		{
			software: fleet.Software{
				Name:             "AdBlock — best ad blocker",
				Source:           "chrome_extensions",
				Version:          "5.1.2",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:getadblock:adblock:5.1.2:*:*:*:*:chrome:*:*",
		},
		{
			software: fleet.Software{
				Name:             "uBlock Origin",
				Source:           "chrome_extensions",
				Version:          "1.44.4",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:ublockorigin:ublock_origin:1.44.4:*:*:*:*:chrome:*:*",
		},
		{
			software: fleet.Software{
				Name:             "uBlock Origin",
				Source:           "chrome_extensions",
				Version:          "1.44.2",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:ublockorigin:ublock_origin:1.44.2:*:*:*:*:chrome:*:*",
		},
		{
			software: fleet.Software{
				Name:             "uBlock Origin",
				Source:           "chrome_extensions",
				Version:          "1.44.0",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:ublockorigin:ublock_origin:1.44.0:*:*:*:*:chrome:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Bing",
				Source:           "firefox_addons",
				Version:          "1.3",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:microsoft:bing:1.3:*:*:*:*:firefox:*:*",
		},
		{
			software: fleet.Software{
				Name:             "DuckDuckGo",
				Source:           "firefox_addons",
				Version:          "1.1",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:duckduckgo:duckduckgo:1.1:*:*:*:*:firefox:*:*",
		},
		{
			software: fleet.Software{
				Name:             "node",
				Source:           "homebrew_packages",
				Version:          "18.9.0",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:nodejs:node.js:18.9.0:*:*:*:*:*:*:*",
		},
		{
			software: fleet.Software{
				Name:             "1Password",
				Source:           "programs",
				Version:          "8.9.5",
				Vendor:           "AgileBits Inc.",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:1password:1password:8.9.5:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "7-Zip 22.01 (x64)",
				Source:           "programs",
				Version:          "22.01",
				Vendor:           "Igor Pavlov",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:7-zip:7-zip:22.01:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Adobe Acrobat DC (64-bit)",
				Source:           "programs",
				Version:          "22.002.20212",
				Vendor:           "Adobe",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:adobe:acrobat_dc:22.002.20212:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Brave",
				Source:           "programs",
				Version:          "105.1.43.93",
				Vendor:           "Brave Software Inc",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:brave:brave:105.1.43.93:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Docker Desktop",
				Source:           "programs",
				Version:          "4.12.0",
				Vendor:           "Docker Inc.",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:docker:desktop:4.12.0:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Dropbox",
				Source:           "programs",
				Version:          "157.4.4808",
				Vendor:           "Dropbox, Inc.",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:dropbox:dropbox:157.4.4808:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Git",
				Source:           "programs",
				Version:          "2.37.1",
				Vendor:           "The Git Development Community",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:git-scm:git:2.37.1:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Google Chrome",
				Source:           "programs",
				Version:          "105.0.5195.127",
				Vendor:           "Google LLC",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:google:chrome:105.0.5195.127:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Microsoft Edge",
				Source:           "programs",
				Version:          "105.0.1343.50",
				Vendor:           "Microsoft Corporation",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:microsoft:edge:105.0.1343.50:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Microsoft OneDrive",
				Source:           "programs",
				Version:          "22.181.0828.0002",
				Vendor:           "Microsoft Corporation",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:microsoft:onedrive:22.181.0828.0002:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Microsoft Visual Studio Code (User)",
				Source:           "programs",
				Version:          "1.71.2",
				Vendor:           "Microsoft Corporation",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:microsoft:visual_studio_code:1.71.2:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Mozilla Firefox (x64 en-US)",
				Source:           "programs",
				Version:          "105.0.1",
				Vendor:           "Mozilla",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:mozilla:firefox:105.0.1:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Node.js",
				Source:           "programs",
				Version:          "16.16.0",
				Vendor:           "Node.js Foundation",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:nodejs:node.js:16.16.0:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Oracle VM VirtualBox 6.1.38",
				Source:           "programs",
				Version:          "6.1.38",
				Vendor:           "Oracle Corporation",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:oracle:vm_virtualbox:6.1.38:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Python 3.10.6 (64-bit)",
				Source:           "programs",
				Version:          "3.10.6150.0",
				Vendor:           "Python Software Foundation",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:python:python:3.10.6150.0:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Visual Studio Community 2022",
				Source:           "programs",
				Version:          "17.2.5",
				Vendor:           "Microsoft Corporation",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:microsoft:visual_studio_community:17.2.5:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "VLC media player",
				Source:           "programs",
				Version:          "3.0.17.4",
				Vendor:           "VideoLAN",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:videolan:vlc_media_player:3.0.17.4:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Zoom",
				Source:           "programs",
				Version:          "5.11.1 (6602)",
				Vendor:           "Zoom Video Communications, Inc.",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:zoom:zoom:5.11.1.6602:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:             "attrs",
				Source:           "python_packages",
				Version:          "21.2.0",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:attrs_project:attrs:21.2.0:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Brotli",
				Source:           "python_packages",
				Version:          "1.0.9",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:google:brotli:1.0.9:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "click",
				Source:           "python_packages",
				Version:          "8.0.3",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:click_project:click:8.0.3:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "cryptography",
				Source:           "python_packages",
				Version:          "3.4.8",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:cryptography.io:cryptography:3.4.8:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "decorator",
				Source:           "python_packages",
				Version:          "4.4.2",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:python:decorator:4.4.2:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "duplicity",
				Source:           "python_packages",
				Version:          "0.8.21",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:debian:duplicity:0.8.21:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "feedparser",
				Source:           "python_packages",
				Version:          "6.0.8",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:mark_pilgrim:feedparser:6.0.8:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "html5lib",
				Source:           "python_packages",
				Version:          "1.1",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:html5lib:html5lib:1.1:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "httplib2",
				Source:           "python_packages",
				Version:          "0.20.2",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:httplib2_project:httplib2:0.20.2:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "ipython",
				Source:           "python_packages",
				Version:          "7.31.1",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:ipython:ipython:7.31.1:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "keyring",
				Source:           "python_packages",
				Version:          "23.5.0",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:python:keyring:23.5.0:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "language-selector",
				Source:           "python_packages",
				Version:          "0.1",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:ubuntu_developers:language-selector:0.1:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "lxml",
				Source:           "python_packages",
				Version:          "4.8.0",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:lxml:lxml:4.8.0:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "lz4",
				Source:           "python_packages",
				Version:          "3.1.3+dfsg",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:lz4_project:lz4:3.1.3.dfsg:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Mako",
				Source:           "python_packages",
				Version:          "1.1.3",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:sqlalchemy:mako:1.1.3:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Markdown",
				Source:           "python_packages",
				Version:          "3.3.6",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:cebe:markdown:3.3.6:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "mechanize",
				Source:           "python_packages",
				Version:          "0.4.7",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:mechanize_project:mechanize:0.4.7:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "msgpack",
				Source:           "python_packages",
				Version:          "1.0.3",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:msgpack:msgpack:1.0.3:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "numpy",
				Source:           "python_packages",
				Version:          "1.21.5",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:numpy:numpy:1.21.5:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "oauthlib",
				Source:           "python_packages",
				Version:          "3.2.0",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:oauthlib_project:oauthlib:3.2.0:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "paramiko",
				Source:           "python_packages",
				Version:          "2.9.3",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:paramiko:paramiko:2.9.3:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "parso",
				Source:           "python_packages",
				Version:          "0.8.1",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:parso_project:parso:0.8.1:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Pillow",
				Source:           "python_packages",
				Version:          "9.0.1",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:python:pillow:9.0.1:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "pip",
				Source:           "python_packages",
				Version:          "22.2.1",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:pypa:pip:22.2.1:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "protobuf",
				Source:           "python_packages",
				Version:          "3.12.4",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:golang:protobuf:3.12.4:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Pygments",
				Source:           "python_packages",
				Version:          "2.11.2",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:pygments:pygments:2.11.2:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "PyJWT",
				Source:           "python_packages",
				Version:          "2.3.0",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:pyjwt_project:pyjwt:2.3.0:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "python-apt",
				Source:           "python_packages",
				Version:          "2.3.0+ubuntu2.1",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:debian:python-apt:2.3.0.ubuntu2.1:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "pyxdg",
				Source:           "python_packages",
				Version:          "0.27",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:python:pyxdg:0.27:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "PyYAML",
				Source:           "python_packages",
				Version:          "5.4.1",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:pyyaml:pyyaml:5.4.1:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "reportlab",
				Source:           "python_packages",
				Version:          "3.6.8",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:reportlab:reportlab:3.6.8:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "requests",
				Source:           "python_packages",
				Version:          "2.25.1",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:jenkins:requests:2.25.1:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "scipy",
				Source:           "python_packages",
				Version:          "1.8.0",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:scipy:scipy:1.8.0:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "setuptools",
				Source:           "python_packages",
				Version:          "63.2.0",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:python:setuptools:63.2.0:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "urllib3",
				Source:           "python_packages",
				Version:          "1.26.5",
				Vendor:           "",
				BundleIdentifier: "",
			}, cpe: "cpe:2.3:a:python:urllib3:1.26.5:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:             "UTM.app",
				Source:           "apps",
				Version:          "3.2.4",
				BundleIdentifier: "com.utmapp.UTM",
			}, cpe: "",
		},
		{
			software: fleet.Software{
				Name:             "Docs",
				Source:           "chrome_extensions",
				Version:          "0.10",
				BundleIdentifier: "",
			}, cpe: "",
		},
		// We don't use NVD to detect Mac Office vulnerabilities so all these should have an empty CPE
		{
			software: fleet.Software{
				Name:             "Microsoft PowerPoint.app",
				Source:           "apps",
				Version:          "16.69.1",
				BundleIdentifier: "com.microsoft.Powerpoint",
			}, cpe: "",
		},
		{
			software: fleet.Software{
				Name:             "Microsoft Word.app",
				Source:           "apps",
				Version:          "16.69.1",
				BundleIdentifier: "com.microsoft.Word",
			}, cpe: "",
		},
		{
			software: fleet.Software{
				Name:             "Microsoft Excel.app",
				Source:           "apps",
				Version:          "16.69.1",
				BundleIdentifier: "com.microsoft.Excel",
			}, cpe: "",
		},
		{
			software: fleet.Software{
				Name:             "Docker.app",
				Source:           "apps",
				Version:          "4.7.1",
				BundleIdentifier: "com.docker.docker",
			}, cpe: "cpe:2.3:a:docker:docker_desktop:4.7.1:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Docker Desktop.app",
				Source:           "apps",
				Version:          "4.16.2",
				BundleIdentifier: "com.electron.dockerdesktop",
			}, cpe: "cpe:2.3:a:docker:docker_desktop:4.16.2:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:             "Docker Desktop.app",
				Source:           "apps",
				Version:          "3.5.0",
				BundleIdentifier: "com.electron.docker-frontend",
			}, cpe: "cpe:2.3:a:docker:docker_desktop:3.5.0:*:*:*:*:macos:*:*",
		},
		// 2023-03-06: there are no entries for the docker python package at the NVD dataset.
		{
			software: fleet.Software{
				Name:    "docker",
				Source:  "python_packages",
				Version: "6.0.1",
			}, cpe: "",
		},
	}

	tempDir := t.TempDir()

	err := DownloadCPEDBFromGithub(tempDir, "")
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "cpe.sqlite")
	db, err := sqliteDB(dbPath)
	require.NoError(t, err)

	err = DownloadCPETranslationsFromGithub(tempDir, "")
	require.NoError(t, err)

	cpeTranslationsPath := filepath.Join(".", cpeTranslationsFilename)
	cpeTranslations, err := loadCPETranslations(cpeTranslationsPath)
	require.NoError(t, err)

	reCache := newRegexpCache()

	for _, tt := range testCases {
		tt := tt
		cpe, err := CPEFromSoftware(db, &tt.software, cpeTranslations, reCache)
		require.NoError(t, err)
		assert.Equal(t, tt.cpe, cpe, tt.software.Name)
	}
}
