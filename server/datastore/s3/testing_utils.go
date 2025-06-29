package s3

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/stretchr/testify/require"
)

const (
	accessKeyID     = "minio"
	secretAccessKey = "minio123!"
	testEndpoint    = "http://localhost:9000"
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

type testBucketCreator interface {
	CreateTestBucket(ctx context.Context, name string) error
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

	err = store.CreateTestBucket(context.Background(), bucket)
	require.NoError(tb, err)

	return store
}

func cleanupStore(tb testing.TB, store *s3store) {
	checkEnv(tb)

	ctx := context.Background()
	resp, err := store.s3Client.ListObjects(ctx, &s3.ListObjectsInput{
		Bucket: &store.bucket,
	})
	var noSuchBucket *types.NoSuchBucket
	if errors.As(err, &noSuchBucket) {
		// OK, nothing to clean-up if the bucket no longer exists, no error
		return
	}
	require.NoError(tb, err)

	var objs []types.ObjectIdentifier
	for _, o := range resp.Contents {
		objs = append(objs, types.ObjectIdentifier{
			Key: o.Key,
		})
	}
	if len(objs) > 0 {
		_, err = store.s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: &store.bucket,
			Delete: &types.Delete{
				Objects: objs,
			},
		})
		require.NoError(tb, err)
	}

	_, err = store.s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: &store.bucket,
	})
	require.NoError(tb, err)
}

func checkEnv(tb testing.TB) {
	if _, ok := os.LookupEnv("MINIO_STORAGE_TEST"); !ok {
		tb.Skip("set MINIO_STORAGE_TEST environment variable to run S3-based tests")
	}
}
