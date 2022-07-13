package s3

import (
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/stretchr/testify/require"
)

const (
	accessKeyID     = "minio"
	secretAccessKey = "minio123!"
	testEndpoint    = "localhost:9000"
)

func setupInstallerStore(tb testing.TB, bucket, prefix string) *InstallerStore {
	if _, ok := os.LookupEnv("FILE_STORAGE_TEST"); !ok {
		tb.Skip("set FILE_STORAGE_TEST environment variable to run S3-based tests")

	}

	store, err := NewInstallerStore(config.S3Config{
		Bucket:           bucket,
		Prefix:           prefix,
		Region:           "minio",
		EndpointURL:      testEndpoint,
		AccessKeyID:      accessKeyID,
		SecretAccessKey:  secretAccessKey,
		ForceS3PathStyle: true,
	})

	require.Nil(tb, err)

	return store
}
