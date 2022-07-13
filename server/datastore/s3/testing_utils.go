package s3

import (
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/fleetdm/fleet/v4/server/config"
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

	return store
}

func seedInstallerStore(tb testing.TB, store *InstallerStore, enrollSecret string) []*Installer {
	checkEnv(tb)
	installers := []*Installer{
		{enrollSecret, "pkg", false},
		{enrollSecret, "msi", false},
		{enrollSecret, "deb", false},
		{enrollSecret, "rpm", false},
		{enrollSecret, "pkg", true},
		{enrollSecret, "msi", true},
		{enrollSecret, "deb", true},
		{enrollSecret, "rpm", true},
	}

	for _, i := range installers {
		uploadMockInstaller(tb, store, i)
	}

	return installers
}

func uploadMockInstaller(tb testing.TB, store *InstallerStore, installer *Installer) {
	checkEnv(tb)
	_, err := store.s3client.PutObject(&s3.PutObjectInput{
		Bucket: &store.bucket,
		Body:   aws.ReadSeekCloser(strings.NewReader(mockInstallerContents)),
		Key:    aws.String(installer.key()),
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
	if _, ok := os.LookupEnv("FILE_STORAGE_TEST"); !ok {
		tb.Skip("set FILE_STORAGE_TEST environment variable to run S3-based tests")

	}
}
