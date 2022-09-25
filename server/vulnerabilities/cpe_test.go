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

func TestCPE(t *testing.T) {
	testCases := []struct {
		software          fleet.Software
		sanitizedName     string
		nameVariations    []string
		productVariations []string
		cpe               string
	}{
		{
			software: fleet.Software{
				Name:    "1Password",
				Version: "8.9.5",
				Vendor:  "AgileBits Inc.",
				Source:  "programs",
			},
			sanitizedName:     "1password",
			nameVariations:    nil,
			productVariations: nil,
			cpe:               "cpe:2.3:a:1password:1password:8.9.5:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:    "1Password – Password Manager",
				Version: "2.3.8",
				Source:  "chrome_extensions",
			},
			sanitizedName: "1password",
			cpe:           "cpe:2.3:a:1password:1password:8.9.5:*:*:*:*:chrome:*:*",
		},

		{
			software: fleet.Software{
				Name:    "Adblock Plus - free ad blocker",
				Version: "3.14.2",
				Source:  "chrome_extensions",
			},
			sanitizedName: "adblock plus",
			cpe:           "cpe:2.3:a:adblockplus:adblock_plus:3.14.2:*:*:*:*:chrome:*:*",
		},
		{
			software: fleet.Software{
				Name:    "AdBlock — best ad blocker",
				Version: "5.1.1",
				Vendor:  "",
				Source:  "chrome_extensions",
			},
			sanitizedName: "adblock",
			cpe:           "cpe:2.3:a:getadblock:adblock:5.1.1:*:*:*:*:chrome:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Adobe Acrobat DC (64-bit)",
				Version: "22.002.20212",
				Vendor:  "Adobe",
				Source:  "programs",
			},
			sanitizedName: "acrobat dc",
			cpe:           "cpe:2.3:a:adobe:acrobat_dc:22.002.20212:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Bing",
				Version: "1.3",
				Vendor:  "",
				Source:  "firefox_addons",
			},
			sanitizedName: "bing",
			cpe:           "cpe:2.3:a:microsoft:bing:1.3:*:*:*:*:firefox:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Brave",
				Version: "105.1.43.93",
				Vendor:  "Brave Software Inc",
				Source:  "programs",
			},
			sanitizedName: "brave",
			cpe:           "cpe:2.3:a:brave:brave:105.1.43.93:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Docker Desktop",
				Version: "4.12.0",
				Vendor:  "Docker Inc.",
				Source:  "programs",
			},
			sanitizedName: "docker desktop",
			cpe:           "cpe:2.3:a:docker:desktop:4.12.0:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Dropbox",
				Version: "157.4.4808",
				Vendor:  "Dropbox, Inc.",
				Source:  "programs",
			},
			sanitizedName: "dropbox",
			cpe:           "cpe:2.3:a:dropbox:dropbox:157.4.4808:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:    "DuckDuckGo",
				Version: "1.1",
				Vendor:  "",
				Source:  "firefox_addons",
			},
			sanitizedName: "duckduckgo",
			cpe:           "cpe:2.3:a:duckduckgo:duckduckgo:1.1:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Git",
				Version: "2.37.1",
				Vendor:  "The Git Development Community",
				Source:  "programs",
			},
			sanitizedName: "git",
			cpe:           "cpe:2.3:a:git:git:2.37.1:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Google Chrome",
				Version: "105.0.5195.127",
				Vendor:  "Google LLC",
				Source:  "programs",
			},
			sanitizedName: "google chrome",
			cpe:           "cpe:2.3:a:google:chrome:105.0.5195.127:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Microsoft Edge",
				Version: "105.0.1343.50",
				Vendor:  "Microsoft Corporation",
				Source:  "programs",
			},
			sanitizedName: "microsoft edge",
			cpe:           "cpe:2.3:a:microsoft:edge:105.0.1343.50:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Microsoft OneDrive",
				Version: "22.181.0828.0002",
				Vendor:  "Microsoft Corporation",
				Source:  "programs",
			},
			sanitizedName: "microsoft onedrive",
			cpe:           "cpe:2.3:a:microsoft:onedrive:22.181.0828.0002:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Microsoft Visual Studio Code (User)",
				Version: "1.71.2",
				Vendor:  "Microsoft Corporation",
				Source:  "programs",
			},
			sanitizedName: "microsoft visual studio code",
			cpe:           "cpe:2.3:a:microsoft:visual_studio_code:1.71.2:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Mozilla Firefox (x64 en-US)",
				Version: "105.0.1",
				Vendor:  "Mozilla",
				Source:  "programs",
			},
			sanitizedName: "firefox",
			cpe:           "cpe:2.3:a:mozilla:firefox:105.0.1:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Oracle VM VirtualBox 6.1.38",
				Version: "6.1.38",
				Vendor:  "Oracle Corporation",
				Source:  "programs",
			},
			sanitizedName: "oracle vm virtualbox",
			cpe:           "cpe:2.3:a:oracle:vm_virtualbox:6.1.38:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:    "pip",
				Version: "22.2.1",
				Vendor:  "",
				Source:  "python_packages",
			},
			sanitizedName: "pip",
			cpe:           "cpe:2.3:a:pypa:pip:22.2.1:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Python 3.10.6 (64-bit)",
				Version: "3.10.6150.0",
				Vendor:  "Python Software Foundation",
				Source:  "programs",
			},
			sanitizedName: "python",
			cpe:           "cpe:2.3:a:python:python:3.10.6150.0:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:    "setuptools",
				Version: "63.2.0",
				Vendor:  "",
				Source:  "python_packages",
			},
			sanitizedName: "setuptools",
			cpe:           "cpe:2.3:a:python:setuptools:63.2.0:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Steam",
				Version: "2.10.91.91",
				Vendor:  "Valve Corporation",
				Source:  "programs",
			},
			sanitizedName: "steam",
			cpe:           " ",
		},
		{
			software: fleet.Software{
				Name:    "Visual Studio Community 2022",
				Version: "17.2.5",
				Vendor:  "Microsoft Corporation",
				Source:  "programs",
			},
			sanitizedName: "visual studio community",
			cpe:           "cpe:2.3:a:microsoft:visual_studio_community:17.2.5:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:    "VLC media player",
				Version: "3.0.17.4",
				Vendor:  "VideoLAN",
				Source:  "programs",
			},
			sanitizedName: "vlc media player",
			cpe:           "cpe:2.3:a:videolan:vlc:3.0.17.4:*:*:*:*:windows:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Zoom",
				Version: "5.11.1 (6602)",
				Vendor:  "Zoom Video Communications, Inc.",
				Source:  "programs",
			},
			sanitizedName: "zoom",
			cpe:           "cpe:2.3:*:zoom:zoom:5.11.1.6602:*:*:*:*:*:*:*",
		},
		{
			software: fleet.Software{
				Name:    "attrs",
				Version: "21.2.0",
				Vendor:  "",
				Source:  "python_packages",
			},
			sanitizedName: "attrs",
			cpe:           "cpe:2.3:a:attrs_project:attrs:21.2.0:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Brotli",
				Version: "1.0.9",
				Vendor:  "",
				Source:  "python_packages",
			},
			sanitizedName: "brotli",
			cpe:           "cpe:2.3:a:google:brotli:1.0.9:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:    "cryptography",
				Version: "3.4.8",
				Vendor:  "",
				Source:  "python_packages",
			},
			sanitizedName: "cryptography",
			cpe:           "cpe:2.3:a:cryptography.io:cryptography:3.4.8:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:    "decorator",
				Version: "4.4.2",
				Vendor:  "",
				Source:  "python_packages",
			},
			sanitizedName: "decorator",
			cpe:           "cpe:2.3:*:python:decorator:4.4.2:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:    "duplicity",
				Version: "0.8.21",
				Vendor:  "",
				Source:  "python_packages",
			},
			sanitizedName: "duplicity",
			cpe:           "cpe:2.3:*:debian:duplicity:0.8.21:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:    "feedparser",
				Version: "6.0.8",
				Vendor:  "",
				Source:  "python_packages",
			},
			sanitizedName: "feedparser",
			cpe:           "cpe:2.3:*:mark_pilgrim:feedparser:6.0.8:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:    "html5lib",
				Version: "1.1",
				Vendor:  "",
				Source:  "python_packages",
			},

			sanitizedName: "html5lib",
			cpe:           "cpe:2.3:a:html5lib:html5lib:1.1:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:    "httplib2",
				Version: "0.20.2",
				Vendor:  "",
				Source:  "python_packages",
			},
			cpe:           "cpe:2.3:a:httplib2_project:httplib2:0.20.2:*:*:*:*:python:*:*",
			sanitizedName: "httplib2",
		},
		{
			software: fleet.Software{
				Name:    "ipython",
				Version: "7.31.1",
				Vendor:  "",
				Source:  "python_packages",
			},
			sanitizedName: "ipython",
			cpe:           "cpe:2.3:a:ipython:ipython:7.31.1:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:    "keyring",
				Version: "23.5.0",
				Vendor:  "",
				Source:  "python_packages",
			},
			sanitizedName: "keyring",
			cpe:           "cpe:2.3:*:python:keyring:23.5.0:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:    "lxml",
				Version: "4.8.0",
				Vendor:  "",
				Source:  "python_packages",
			},
			sanitizedName: "lxml",
			cpe:           "cpe:2.3:a:lxml:lxml:4.8.0:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:    "lz4",
				Version: "3.1.3+dfsg",
				Vendor:  "",
				Source:  "python_packages",
			},
			sanitizedName: "lz4",
			cpe:           "cpe:2.3:a:lz4_project:lz4:3.1.3.dfsg:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Mako",
				Version: "1.1.3",
				Vendor:  "",
				Source:  "python_packages",
			},
			sanitizedName: "mako",
			cpe:           "cpe:2.3:a:sqlalchemy:mako:1.1.3:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:    "mechanize",
				Version: "0.4.7",
				Vendor:  "",
				Source:  "python_packages",
			},
			sanitizedName: "mechanize",
			cpe:           "cpe:2.3:a:mechanize_project:mechanize:0.4.7:*:*:*:*:python:*:*",
		},
		{
			software: fleet.Software{
				Name:    "1Password 7.app",
				Version: "7.9.6",
				Vendor:  "",
				Source:  "apps",
			},
			sanitizedName: "1password",

			cpe: "cpe:2.3:a:1password:1password:7.9.6:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:    "AirDrop.app",
				Version: "1.0",
				Vendor:  "",
				Source:  "apps",
			},
			sanitizedName: "airdop",
			cpe:           "cpe:2.3:a:airdrop_project:airdrop:1.0:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Finder.app",
				Version: "12.5",
				Vendor:  "",
				Source:  "apps",
			},
			sanitizedName: "finder",
			cpe:           "cpe:2.3:a:apple:finder:12.5:*:*:*:*:macos:*:*",
		},
		{
			software: fleet.Software{
				Name:    "Firefox.app",
				Version: "105.0.1",
				Vendor:  "",
				Source:  "apps",
			},
			sanitizedName: "firefox",
			cpe:           "cpe:2.3:a:mozilla:firefox:105.0.1:*:*:*:*:macos:*:*",
		},
	}

	t.Run("sanitizedProductName", func(t *testing.T) {
		for _, tc := range testCases {
			actual := sanitizedProductName(&tc.software)
			require.Equal(t, tc.sanitizedName, actual)
		}
	})
}

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

	software := &fleet.Software{
		Name:             "1Password.app",
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
