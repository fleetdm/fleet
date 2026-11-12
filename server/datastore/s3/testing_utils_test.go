package s3

import (
	"context"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/stretchr/testify/require"
)

// These helpers mirror server/datastore/s3/s3test but live in package s3 so
// internal tests can use them without forming an import cycle
// (s3test imports s3, so s3's own _test.go files can't import s3test).
// External callers should use s3test instead.

const (
	testAccessKeyID     = "locals3"
	testSecretAccessKey = "locals3"
)

var testEndpoint = func() string {
	if port := os.Getenv("FLEET_S3_PORT"); port != "" {
		return "http://localhost:" + port
	}
	return "http://localhost:9000"
}()

func setupTestSoftwareInstallerStore(tb testing.TB, bucket, prefix string) *SoftwareInstallerStore {
	return setupTestStore(tb, bucket, prefix, NewSoftwareInstallerStore)
}

func setupTestBootstrapPackageStore(tb testing.TB, bucket, prefix string) *BootstrapPackageStore {
	return setupTestStore(tb, bucket, prefix, NewBootstrapPackageStore)
}

type testStoreIface interface {
	CreateTestBucket(ctx context.Context, name string) error
	CleanupTestBucket(ctx context.Context) error
}

func setupTestStore[T testStoreIface](tb testing.TB, bucket, prefix string, newFn func(config.S3Config) (T, error)) T {
	checkTestEnv(tb)

	store, err := newFn(config.S3Config{
		SoftwareInstallersBucket:           bucket,
		SoftwareInstallersPrefix:           prefix,
		SoftwareInstallersRegion:           "localhost",
		SoftwareInstallersEndpointURL:      testEndpoint,
		SoftwareInstallersAccessKeyID:      testAccessKeyID,
		SoftwareInstallersSecretAccessKey:  testSecretAccessKey,
		SoftwareInstallersForceS3PathStyle: true,
		SoftwareInstallersDisableSSL:       true,

		CarvesBucket:           bucket,
		CarvesPrefix:           prefix,
		CarvesRegion:           "localhost",
		CarvesEndpointURL:      testEndpoint,
		CarvesAccessKeyID:      testAccessKeyID,
		CarvesSecretAccessKey:  testSecretAccessKey,
		CarvesForceS3PathStyle: true,
		CarvesDisableSSL:       true,
	})
	require.NoError(tb, err)

	err = store.CreateTestBucket(context.Background(), bucket)
	require.NoError(tb, err)

	tb.Cleanup(func() {
		if err := store.CleanupTestBucket(context.Background()); err != nil {
			tb.Errorf("cleanup s3 bucket %q: %v", bucket, err)
		}
	})

	return store
}

func checkTestEnv(tb testing.TB) {
	if _, ok := os.LookupEnv("S3_STORAGE_TEST"); !ok {
		tb.Skip("set S3_STORAGE_TEST environment variable to run S3-based tests")
	}
}
