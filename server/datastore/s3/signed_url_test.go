package s3

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// TestSignGCSPresignedURL verifies the GCS presigned-URL download path added on
// top of the upstream CloudFront-only Sign(). It runs fully offline:
// PresignGetObject computes the URL locally without contacting the bucket, and
// a non-empty region avoids the GetBucketRegion network lookup in newS3Store.
func TestSignGCSPresignedURL(t *testing.T) {
	baseCfg := func() config.S3Config {
		return config.S3Config{
			SoftwareInstallersBucket:           "test-bucket",
			SoftwareInstallersRegion:           "auto",
			SoftwareInstallersEndpointURL:      "https://storage.googleapis.com",
			SoftwareInstallersAccessKeyID:      "GOOG-test",
			SoftwareInstallersSecretAccessKey:  "secret",
			SoftwareInstallersForceS3PathStyle: true,
		}
	}

	t.Run("signed url enabled returns GCS presigned URL", func(t *testing.T) {
		cfg := baseCfg()
		cfg.SoftwareInstallersSignedURL = true
		store, err := NewSoftwareInstallerStore(cfg)
		require.NoError(t, err)

		signed, err := store.Sign(context.Background(), "abc123", 15*time.Minute)
		require.NoError(t, err)
		require.Contains(t, signed, "storage.googleapis.com")
		require.Contains(t, signed, "test-bucket")
		require.True(t,
			strings.Contains(signed, "X-Amz-Signature") || strings.Contains(signed, "X-Goog-Signature"),
			"expected a presigned signature query param, got %s", signed)
	})

	t.Run("signed url disabled and no cloudfront returns ErrNotConfigured", func(t *testing.T) {
		store, err := NewSoftwareInstallerStore(baseCfg())
		require.NoError(t, err)

		_, err = store.Sign(context.Background(), "abc123", 15*time.Minute)
		require.True(t, errors.Is(err, fleet.ErrNotConfigured), "expected ErrNotConfigured, got %v", err)
	})

	t.Run("signed url with gcs iam auth is rejected", func(t *testing.T) {
		// GCS IAM (bearer) auth is incompatible with SigV4 presigning, so store
		// initialization must fail rather than hand out unusable signed URLs.
		cfg := baseCfg()
		cfg.SoftwareInstallersSignedURL = true
		cfg.SoftwareInstallersGCSIAMAuth = true

		_, err := NewSoftwareInstallerStore(cfg)
		require.Error(t, err)
	})
}
