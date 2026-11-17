package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/datastore/s3/s3test"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TestIntegrationsStreamingInstallers runs the installer download and staged
// upload flows with cloud_storage.stream_installer_downloads enabled, against
// the local S3-compatible store.
func TestIntegrationsStreamingInstallers(t *testing.T) {
	if _, ok := os.LookupEnv("S3_STORAGE_TEST"); !ok {
		t.Skip("set S3_STORAGE_TEST to run streaming installer tests")
	}
	testingSuite := new(integrationStreamingInstallersTestSuite)
	testingSuite.withServer.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type integrationStreamingInstallersTestSuite struct {
	withServer
	suite.Suite
	installerStore    *countingStreamingStore
	stagedUploadStore *s3.StagedUploadStore
}

// countingStreamingStore counts the installers streamed to a staging object,
// to prove the flows don't fall back to temp files.
type countingStreamingStore struct {
	*s3.SoftwareInstallerStore
	putStaging atomic.Int32
}

func (c *countingStreamingStore) PutStaging(ctx context.Context, stagingID string, r io.Reader) error {
	c.putStaging.Add(1)
	return c.SoftwareInstallerStore.PutStaging(ctx, stagingID, r)
}

const streamingTestBucket = "integration-tests-streaming"

func (s *integrationStreamingInstallersTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationStreamingInstallersTestSuite")

	s.installerStore = &countingStreamingStore{SoftwareInstallerStore: s3test.SetupSoftwareInstallerStore(s.T(), streamingTestBucket, "")}
	s.stagedUploadStore = s3test.SetupStagedUploadStore(s.T(), streamingTestBucket, "")

	fleetCfg := config.TestConfig()
	fleetCfg.CloudStorage.StreamInstallerDownloads = true
	fleetCfg.S3.SoftwareInstallersSignedURL = true
	fleetCfg.S3.SoftwareInstallersBucket = streamingTestBucket

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		logger = slog.New(slog.DiscardHandler)
	}
	users, server := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{
		License:              &fleet.LicenseInfo{Tier: fleet.TierPremium},
		Pool:                 redistest.SetupRedis(s.T(), s.T().Name(), false, false, false),
		Rs:                   pubsub.NewInmemQueryResults(),
		Logger:               logger,
		SoftwareInstallStore: s.installerStore,
		StagedUploadStore:    s.stagedUploadStore,
		DBConns:              s.dbConns,
		FleetConfig:          &fleetCfg,
	})
	s.server = server
	s.users = users
	s.token = s.getTestAdminToken()
	s.cachedTokens = make(map[string]string)
}

func (s *integrationStreamingInstallersTestSuite) TearDownTest() {
	s.withServer.commonTearDownTest(s.T())
}

// requireNoStagingObjects fails if a flow left a staging object behind.
func (s *integrationStreamingInstallersTestSuite) requireNoStagingObjects(t *testing.T) {
	n, err := s.stagedUploadStore.Cleanup(t.Context(), nil, time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.Zero(t, n, "staging objects left behind")
}

func (s *integrationStreamingInstallersTestSuite) storageIDOf(t *testing.T, filename string) string {
	var storageID string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &storageID,
			`SELECT storage_id FROM software_installers WHERE filename = ? ORDER BY id DESC LIMIT 1`, filename)
	})
	return storageID
}

func (s *integrationStreamingInstallersTestSuite) TestBatchSetSoftwareInstallers() {
	t := s.T()
	ctx := t.Context()

	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	fixtures := []string{"ruby.deb", "ruby.rpm", "fleet-osquery.msi", "hello-world-installer.exe", "dummy_installer.pkg"}
	contents := map[string][]byte{}
	for _, f := range fixtures {
		b, err := os.ReadFile(filepath.Join("testdata", "software-installers", f))
		require.NoError(t, err)
		contents["/"+f] = b
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, ok := contents[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(b)
	}))
	t.Cleanup(srv.Close)

	apply := func(software []*fleet.SoftwareInstallerPayload, dryRun bool) string {
		var resp batchSetSoftwareInstallersResponse
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: software},
			http.StatusAccepted, &resp, "fleet_name", tm.Name, "dry_run", jsonBool(dryRun))
		return resp.RequestUUID
	}
	payloads := func() []*fleet.SoftwareInstallerPayload {
		var p []*fleet.SoftwareInstallerPayload
		for _, f := range fixtures {
			p = append(p, &fleet.SoftwareInstallerPayload{
				URL:             srv.URL + "/" + f,
				InstallScript:   "install",
				UninstallScript: "uninstall",
			})
		}
		return p
	}

	t.Run("dry run stores nothing", func(t *testing.T) {
		before := s.installerStore.putStaging.Load()
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, apply(payloads(), true))
		require.EqualValues(t, len(fixtures), s.installerStore.putStaging.Load()-before)
		s.requireNoStagingObjects(t)
		for _, f := range fixtures {
			sum := sha256.Sum256(contents["/"+f])
			exists, err := s.installerStore.Exists(ctx, hex.EncodeToString(sum[:]))
			require.NoError(t, err)
			require.False(t, exists, f)
		}
	})

	t.Run("hash mismatch fails without leftovers", func(t *testing.T) {
		p := payloads()[:1]
		p[0].SHA256 = hex.EncodeToString(make([]byte, 32))
		msg := waitBatchSetSoftwareInstallersFailed(t, &s.withServer, tm.Name, apply(p, false))
		require.Contains(t, msg, "hash does not match")
		s.requireNoStagingObjects(t)
	})

	t.Run("apply streams and stores every installer", func(t *testing.T) {
		before := s.installerStore.putStaging.Load()
		packages := waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, apply(payloads(), false))
		require.Len(t, packages, len(fixtures))
		require.EqualValues(t, len(fixtures), s.installerStore.putStaging.Load()-before)
		s.requireNoStagingObjects(t)

		for _, f := range fixtures {
			sum := sha256.Sum256(contents["/"+f])
			wantID := hex.EncodeToString(sum[:])
			require.Equal(t, wantID, s.storageIDOf(t, f), f)

			rc, _, err := s.installerStore.Get(ctx, wantID)
			require.NoError(t, err)
			got, err := io.ReadAll(rc)
			rc.Close()
			require.NoError(t, err)
			require.True(t, bytes.Equal(contents["/"+f], got), f)
		}

		// Metadata parsed from the staging objects with ranged reads.
		var titles []struct {
			Name    string `db:"name"`
			Version string `db:"version"`
		}
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &titles,
				`SELECT st.name, si.version FROM software_installers si JOIN software_titles st ON st.id = si.title_id
				 WHERE si.global_or_team_id = ? ORDER BY st.name`, tm.ID)
		})
		require.Len(t, titles, len(fixtures))
		for _, title := range titles {
			require.NotEmpty(t, title.Name)
		}
	})
}

func (s *integrationStreamingInstallersTestSuite) TestStagedSoftwareUpload() {
	t := s.T()

	content, err := os.ReadFile(filepath.Join("testdata", "software-installers", "ruby.deb"))
	require.NoError(t, err)

	var presignResp createStagedUploadResponse
	res := s.Do("POST", "/api/latest/fleet/staged_upload",
		createStagedUploadRequest{Target: fleet.StagedUploadTargetSoftwarePackage, Size: int64(len(content))}, http.StatusOK)
	require.NoError(t, json.NewDecoder(res.Body).Decode(&presignResp))
	req, err := http.NewRequest(http.MethodPut, presignResp.URL, bytes.NewReader(content))
	require.NoError(t, err)
	putResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	putResp.Body.Close()
	require.Equal(t, http.StatusOK, putResp.StatusCode)

	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		StagedUploadID: presignResp.UploadID, Filename: "ruby.deb", InstallScript: "install", TeamID: new(uint(0)),
	}, http.StatusOK, "")

	// Promoted with a server-side copy: the staged upload is gone and the
	// installer is stored under its hash.
	_, _, err = s.stagedUploadStore.Get(t.Context(), presignResp.UploadID)
	require.True(t, fleet.IsNotFound(err))
	sum := sha256.Sum256(content)
	wantID := hex.EncodeToString(sum[:])
	require.Equal(t, wantID, s.storageIDOf(t, "ruby.deb"))
	exists, err := s.installerStore.Exists(t.Context(), wantID)
	require.NoError(t, err)
	require.True(t, exists)
	s.requireNoStagingObjects(t)
}

func jsonBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
