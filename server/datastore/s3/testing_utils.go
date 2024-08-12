package s3

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

const (
	accessKeyID           = "minio"
	secretAccessKey       = "minio123!"
	testEndpoint          = "localhost:9000"
	mockInstallerContents = "mock"
)

func SetupTestSoftwareInstallerStore(tb testing.TB, bucket, prefix string) *SoftwareInstallerStore {
	store := setupTestStore(tb, bucket, prefix, NewSoftwareInstallerStore)
	tb.Cleanup(func() { cleanupStore(tb, store.s3store) })
	return store
}

func SetupTestBootstrapPackageStore(tb testing.TB, bucket, prefix string) *BootstrapPackageStore {
	store := setupTestStore(tb, bucket, prefix, NewBootstrapPackageStore)
	tb.Cleanup(func() { cleanupStore(tb, store.s3store) })
	return store
}

// SetupTestInstallerStore creates a new store with minio as a back-end
// for local testing
func SetupTestInstallerStore(tb testing.TB, bucket, prefix string) *InstallerStore {
	store := setupTestStore(tb, bucket, prefix, NewInstallerStore)
	tb.Cleanup(func() { cleanupStore(tb, store.s3store) })
	return store
}

type testBucketCreator interface {
	CreateTestBucket(name string) error
}

func setupTestStore[T testBucketCreator](tb testing.TB, bucket, prefix string, newFn func(config.S3Config) (T, error)) T {
	checkEnv(tb)

	store, err := newFn(config.S3Config{
		SoftwareInstallersBucket:           bucket,
		SoftwareInstallersPrefix:           prefix,
		SoftwareInstallersRegion:           "minio",
		SoftwareInstallersEndpointURL:      testEndpoint,
		SoftwareInstallersAccessKeyID:      accessKeyID,
		SoftwareInstallersSecretAccessKey:  secretAccessKey,
		SoftwareInstallersForceS3PathStyle: true,
		SoftwareInstallersDisableSSL:       true,

		CarvesBucket:           bucket,
		CarvesPrefix:           prefix,
		CarvesRegion:           "minio",
		CarvesEndpointURL:      testEndpoint,
		CarvesAccessKeyID:      accessKeyID,
		CarvesSecretAccessKey:  secretAccessKey,
		CarvesForceS3PathStyle: true,
		CarvesDisableSSL:       true,
	})
	require.Nil(tb, err)

	err = store.CreateTestBucket(bucket)
	require.NoError(tb, err)

	return store
}

// SeedTestInstallerStore adds mock installers to the given store
func SeedTestInstallerStore(tb testing.TB, store *InstallerStore, enrollSecret string) []fleet.Installer {
	checkEnv(tb)
	installers := []fleet.Installer{
		mockInstaller(enrollSecret, "pkg", true),
		mockInstaller(enrollSecret, "msi", true),
		mockInstaller(enrollSecret, "deb", true),
		mockInstaller(enrollSecret, "rpm", true),
		mockInstaller(enrollSecret, "pkg", false),
		mockInstaller(enrollSecret, "msi", false),
		mockInstaller(enrollSecret, "deb", false),
		mockInstaller(enrollSecret, "rpm", false),
	}

	for _, i := range installers {
		_, err := store.Put(context.Background(), i)
		require.NoError(tb, err)
	}

	return installers
}

func mockInstaller(secret, kind string, desktop bool) fleet.Installer {
	return fleet.Installer{
		EnrollSecret: secret,
		Kind:         kind,
		Desktop:      desktop,
		Content:      aws.ReadSeekCloser(strings.NewReader(mockInstallerContents)),
	}
}

func cleanupStore(tb testing.TB, store *s3store) {
	checkEnv(tb)

	resp, err := store.s3client.ListObjects(&s3.ListObjectsInput{
		Bucket: &store.bucket,
	})
	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == s3.ErrCodeNoSuchBucket {
			// fine, nothing to clean-up if the bucket no longer exists, no error
			return
		}
	}
	require.NoError(tb, err)

	var objs []*s3.ObjectIdentifier
	for _, o := range resp.Contents {
		objs = append(objs, &s3.ObjectIdentifier{Key: o.Key})
	}
	if len(objs) > 0 {
		_, err = store.s3client.DeleteObjects(&s3.DeleteObjectsInput{
			Bucket: &store.bucket,
			Delete: &s3.Delete{
				Objects: objs,
			},
		})
		require.NoError(tb, err)
	}

	_, err = store.s3client.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: &store.bucket,
	})
	require.NoError(tb, err)
}

func checkEnv(tb testing.TB) {
	if _, ok := os.LookupEnv("MINIO_STORAGE_TEST"); !ok {
		tb.Skip("set MINIO_STORAGE_TEST environment variable to run S3-based tests")
	}
}
