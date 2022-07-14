package s3

import (
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
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

func setupInstallerStore(tb testing.TB, bucket, prefix string) *InstallerStore {
	checkEnv(tb)

	store, err := NewInstallerStore(config.S3Config{
		Bucket:           bucket,
		Prefix:           prefix,
		Region:           "minio",
		EndpointURL:      testEndpoint,
		AccessKeyID:      accessKeyID,
		SecretAccessKey:  secretAccessKey,
		ForceS3PathStyle: true,
		DisableSSL:       true,
	})
	require.Nil(tb, err)

	store.s3client.CreateBucket(&s3.CreateBucketInput{
		Bucket:                    &bucket,
		CreateBucketConfiguration: &s3.CreateBucketConfiguration{},
	})

	tb.Cleanup(func() { cleanupStore(tb, store) })

	return store
}

func seedInstallerStore(tb testing.TB, store *InstallerStore, enrollSecret string) []*fleet.Installer {
	checkEnv(tb)
	installers := []*fleet.Installer{
		{EnrollSecret: enrollSecret, Kind: "pkg", Desktop: false},
		{EnrollSecret: enrollSecret, Kind: "msi", Desktop: false},
		{EnrollSecret: enrollSecret, Kind: "deb", Desktop: false},
		{EnrollSecret: enrollSecret, Kind: "rpm", Desktop: false},
		{EnrollSecret: enrollSecret, Kind: "pkg", Desktop: true},
		{EnrollSecret: enrollSecret, Kind: "msi", Desktop: true},
		{EnrollSecret: enrollSecret, Kind: "deb", Desktop: true},
		{EnrollSecret: enrollSecret, Kind: "rpm", Desktop: true},
	}

	for _, i := range installers {
		uploadMockInstaller(tb, store, i)
	}

	return installers
}

func uploadMockInstaller(tb testing.TB, store *InstallerStore, installer *fleet.Installer) {
	checkEnv(tb)
	_, err := store.s3client.PutObject(&s3.PutObjectInput{
		Bucket: &store.bucket,
		Body:   aws.ReadSeekCloser(strings.NewReader(mockInstallerContents)),
		Key:    aws.String(store.keyForInstaller(*installer)),
	})
	require.NoError(tb, err)
}

func cleanupStore(tb testing.TB, store *InstallerStore) {
	checkEnv(tb)
	resp, err := store.s3client.ListObjects(&s3.ListObjectsInput{
		Bucket: &store.bucket,
	})
	require.NoError(tb, err)

	var objs []*s3.ObjectIdentifier
	for _, o := range resp.Contents {
		objs = append(objs, &s3.ObjectIdentifier{Key: o.Key})
	}
	_, err = store.s3client.DeleteObjects(&s3.DeleteObjectsInput{
		Bucket: &store.bucket,
		Delete: &s3.Delete{
			Objects: objs,
		},
	})
	require.NoError(tb, err)

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
