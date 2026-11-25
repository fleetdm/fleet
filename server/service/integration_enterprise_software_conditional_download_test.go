package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"

	ma "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A package replaced on one fleet keeps the URL and ETag of the original
// download. Another fleet adding that URL must still end up with the origin's
// bytes.
func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallersConditionalDownloadAfterPackageReplace() {
	t := s.T()
	ctx := context.Background()

	originBytes, err := os.ReadFile(filepath.Join("testdata", "software-installers", "ruby.deb"))
	require.NoError(t, err)
	originHash := sha256Hex(originBytes)
	// Same package name and extension, different bytes.
	replacedBytes := append(bytes.Clone(originBytes), []byte("replaced")...)
	replacedHash := sha256Hex(replacedBytes)
	require.NotEqual(t, originHash, replacedHash)

	const originETag = `"ruby-deb-origin"`
	var (
		mu           sync.Mutex
		ifNoneMatchs []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		ifNoneMatchs = append(ifNoneMatchs, r.Header.Get("If-None-Match"))
		mu.Unlock()
		w.Header().Set("ETag", originETag)
		if r.Header.Get("If-None-Match") == originETag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
		_, err := w.Write(originBytes)
		assert.NoError(t, err)
	}))
	t.Cleanup(srv.Close)
	pkgURL := srv.URL + "/ruby.deb"
	lastIfNoneMatch := func() string {
		mu.Lock()
		defer mu.Unlock()
		require.NotEmpty(t, ifNoneMatchs)
		return ifNoneMatchs[len(ifNoneMatchs)-1]
	}
	requestCount := func() int {
		mu.Lock()
		defer mu.Unlock()
		return len(ifNoneMatchs)
	}

	newFleet := func(suffix string) *fleet.Team {
		tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + suffix})
		require.NoError(t, err)
		return tm
	}
	fleetA := newFleet("-a")
	fleetEarly := newFleet("-early")
	fleetB := newFleet("-b")
	fleetPinned := newFleet("-pinned")
	fleetAlways := newFleet("-always")

	batch := func(tm *fleet.Team, payloads []*fleet.SoftwareInstallerPayload) []fleet.SoftwarePackageResponse {
		t.Helper()
		var resp batchSetSoftwareInstallersResponse
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: payloads},
			http.StatusAccepted, &resp, "fleet_name", tm.Name)
		return waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, resp.RequestUUID)
	}
	addURL := func(tm *fleet.Team, p fleet.SoftwareInstallerPayload) {
		t.Helper()
		p.URL = pkgURL
		pkgs := batch(tm, []*fleet.SoftwareInstallerPayload{&p})
		require.Len(t, pkgs, 1)
	}
	// storedHash hashes the bytes Fleet serves for the fleet's installer.
	storedHash := func(tm *fleet.Team, titleID uint) string {
		t.Helper()
		r := s.Do("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package?alt=media", titleID), nil,
			http.StatusOK, "fleet_id", fmt.Sprint(tm.ID))
		defer r.Body.Close()
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		return sha256Hex(b)
	}
	installerRow := func(tm *fleet.Team) (storageID, url, etag string) {
		t.Helper()
		var row struct {
			StorageID string  `db:"storage_id"`
			URL       string  `db:"url"`
			ETag      *string `db:"http_etag"`
		}
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &row,
				`SELECT storage_id, url, http_etag FROM software_installers WHERE global_or_team_id = ?`, tm.ID)
		})
		if row.ETag != nil {
			etag = *row.ETag
		}
		return row.StorageID, row.URL, etag
	}

	// Maintainer on fleet A only.
	maintainerA := &fleet.User{
		Name:  "Fleet A maintainer",
		Email: uuid.NewString() + "@example.com",
		Teams: []fleet.UserTeam{{Team: *fleetA, Role: fleet.RoleMaintainer}},
	}
	require.NoError(t, maintainerA.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(ctx, maintainerA)
	require.NoError(t, err)
	adminToken := s.token
	t.Cleanup(func() { s.token = s.getTestAdminToken() })

	// 1. Fleet A adds the URL and stores the origin's bytes and ETag.
	s.setTokenForTest(t, maintainerA.Email, test.GoodPassword)
	addURL(fleetA, fleet.SoftwareInstallerPayload{})
	storageA, urlA, etagA := installerRow(fleetA)
	require.Equal(t, originHash, storageA)
	require.Equal(t, pkgURL, urlA)
	require.Equal(t, originETag, etagA)
	titleID := getSoftwareTitleID(t, s.ds, "ruby", "deb_packages")

	// Before the replace, another fleet adding the URL gets the origin's bytes.
	s.token = adminToken
	addURL(fleetEarly, fleet.SoftwareInstallerPayload{})
	require.Equal(t, originETag, lastIfNoneMatch())
	require.Equal(t, originHash, storedHash(fleetEarly, titleID))

	// 2. Fleet A replaces the package. The row keeps its URL but drops the ETag,
	// which described the origin's bytes.
	s.setTokenForTest(t, maintainerA.Email, test.GoodPassword)
	tfr, err := fleet.NewTempFileReader(bytes.NewReader(replacedBytes), t.TempDir)
	require.NoError(t, err)
	defer tfr.Close()
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		Filename:      "ruby.deb",
		InstallerFile: tfr,
		TitleID:       titleID,
		TeamID:        &fleetA.ID,
	}, http.StatusOK, "")
	storageA, urlA, etagA = installerRow(fleetA)
	require.Equal(t, replacedHash, storageA)
	require.Equal(t, pkgURL, urlA)
	require.Empty(t, etagA)

	// Fleet A's maintainer can't add software to fleet B.
	var forbidden batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch",
		batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{{URL: pkgURL}}},
		http.StatusForbidden, &forbidden, "fleet_name", fleetB.Name)

	// Remove the earlier fleet's row so fleet A's is the only match for the URL.
	s.token = adminToken
	batch(fleetEarly, []*fleet.SoftwareInstallerPayload{})

	// 3 and 4. Fleet B adds the URL with the defaults: no hash, no always_download.
	// Fleet A's row has no ETag, so Fleet downloads the whole file.
	addURL(fleetB, fleet.SoftwareInstallerPayload{})
	require.Empty(t, lastIfNoneMatch())
	storageB, _, _ := installerRow(fleetB)
	assert.Equal(t, originHash, storageB, "fleet B must store the origin's bytes, got fleet A's replaced package")
	assert.Equal(t, originHash, storedHash(fleetB, titleID), "fleet B must serve the origin's bytes")

	// A pinned hash of the origin's bytes gets the origin's bytes.
	addURL(fleetPinned, fleet.SoftwareInstallerPayload{SHA256: originHash})
	require.Equal(t, originHash, storedHash(fleetPinned, titleID))

	// always_download skips the conditional request.
	before := requestCount()
	addURL(fleetAlways, fleet.SoftwareInstallerPayload{AlwaysDownload: true})
	require.Equal(t, before+1, requestCount())
	require.Empty(t, lastIfNoneMatch())
	require.Equal(t, originHash, storedHash(fleetAlways, titleID))
}

// Same as above for a Fleet-maintained app whose manifest has no hash and
// whose installer URL matches a replaced custom package on another fleet.
func (s *integrationEnterpriseTestSuite) TestBatchSetSoftwareInstallersConditionalDownloadAfterPackageReplaceFMA() {
	t := s.T()
	ctx := context.Background()

	originBytes, err := os.ReadFile(filepath.Join("testdata", "software-installers", "dummy_installer.pkg"))
	require.NoError(t, err)
	originHash := sha256Hex(originBytes)
	replacedBytes := append(bytes.Clone(originBytes), []byte("replaced")...)
	replacedHash := sha256Hex(replacedBytes)

	const originETag = `"fma-pkg-origin"`
	var (
		mu           sync.Mutex
		ifNoneMatchs []string
	)
	installerSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		ifNoneMatchs = append(ifNoneMatchs, r.Header.Get("If-None-Match"))
		mu.Unlock()
		w.Header().Set("ETag", originETag)
		if r.Header.Get("If-None-Match") == originETag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		_, err := w.Write(originBytes)
		assert.NoError(t, err)
	}))
	t.Cleanup(installerSrv.Close)
	pkgURL := installerSrv.URL + "/app.pkg"

	manifestSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(ma.FMAManifestFile{Versions: []*ma.FMAManifestApp{{
			Version:      "6.0",
			Queries:      ma.FMAQueries{Exists: "SELECT 1 FROM osquery_info;"},
			InstallerURL: pkgURL,
			SHA256:       "no_check",
		}}})
		assert.NoError(t, err)
	}))
	t.Cleanup(manifestSrv.Close)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", manifestSrv.URL, t)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_FALLBACK_BASE_URL", manifestSrv.URL, t)
	fma, err := s.ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:             "Conditional download app",
		Slug:             "conditional-download-app/darwin",
		Platform:         "darwin",
		UniqueIdentifier: "com.example.conditional-download-app",
	})
	require.NoError(t, err)

	fleetA, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-a"})
	require.NoError(t, err)
	fleetB, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-b"})
	require.NoError(t, err)
	maintainerA := &fleet.User{
		Name:  "Fleet A maintainer",
		Email: uuid.NewString() + "@example.com",
		Teams: []fleet.UserTeam{{Team: *fleetA, Role: fleet.RoleMaintainer}},
	}
	require.NoError(t, maintainerA.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(ctx, maintainerA)
	require.NoError(t, err)
	adminToken := s.token
	t.Cleanup(func() { s.token = s.getTestAdminToken() })

	batch := func(tm *fleet.Team, p *fleet.SoftwareInstallerPayload) fleet.SoftwarePackageResponse {
		t.Helper()
		var resp batchSetSoftwareInstallersResponse
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: []*fleet.SoftwareInstallerPayload{p}},
			http.StatusAccepted, &resp, "fleet_name", tm.Name)
		pkgs := waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, resp.RequestUUID)
		require.Len(t, pkgs, 1)
		return pkgs[0]
	}
	storageID := func(tm *fleet.Team) string {
		var id string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id, `SELECT storage_id FROM software_installers WHERE global_or_team_id = ?`, tm.ID)
		})
		return id
	}

	// Fleet A adds the FMA's installer URL as a custom package, then replaces it.
	s.setTokenForTest(t, maintainerA.Email, test.GoodPassword)
	pkgA := batch(fleetA, &fleet.SoftwareInstallerPayload{URL: pkgURL})
	require.Equal(t, originHash, storageID(fleetA))
	require.NotNil(t, pkgA.TitleID)
	tfr, err := fleet.NewTempFileReader(bytes.NewReader(replacedBytes), t.TempDir)
	require.NoError(t, err)
	defer tfr.Close()
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		Filename:      "app.pkg",
		InstallerFile: tfr,
		TitleID:       *pkgA.TitleID,
		TeamID:        &fleetA.ID,
	}, http.StatusOK, "")
	require.Equal(t, replacedHash, storageID(fleetA))

	// Fleet B adds the Fleet-maintained app.
	s.token = adminToken
	batch(fleetB, &fleet.SoftwareInstallerPayload{Slug: &fma.Slug})
	mu.Lock()
	require.NotEmpty(t, ifNoneMatchs)
	last := ifNoneMatchs[len(ifNoneMatchs)-1]
	mu.Unlock()
	require.Empty(t, last)
	assert.Equal(t, originHash, storageID(fleetB), "fleet B's Fleet-maintained app must store the origin's bytes, got fleet A's replaced package")
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
