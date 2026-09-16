package s3

import (
	"context"
	"net/url"
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

		u, err := url.Parse(signed)
		require.NoError(t, err)
		require.Equal(t, "https", u.Scheme)
		require.Equal(t, "storage.googleapis.com", u.Host)
		// Path-style addressing puts the bucket and key in the path.
		require.Contains(t, u.Path, "test-bucket")
		require.Contains(t, u.Path, "abc123")

		q := u.Query()
		require.True(t,
			q.Get("X-Amz-Signature") != "" || q.Get("X-Goog-Signature") != "",
			"expected a presigned signature query param, got %s", signed)
		require.NotEmpty(t, q.Get("X-Amz-Algorithm"))
		require.Equal(t, "900", q.Get("X-Amz-Expires")) // 15 minutes
	})

	t.Run("signed url disabled and no cloudfront returns ErrNotConfigured", func(t *testing.T) {
		store, err := NewSoftwareInstallerStore(baseCfg())
		require.NoError(t, err)

		_, err = store.Sign(context.Background(), "abc123", 15*time.Minute)
		require.ErrorIs(t, err, fleet.ErrNotConfigured)
	})

	t.Run("signed url with gcs iam auth is rejected", func(t *testing.T) {
		// GCS IAM (bearer) auth is incompatible with SigV4 presigning, so store
		// initialization must fail rather than hand out unusable signed URLs.
		cfg := baseCfg()
		cfg.SoftwareInstallersSignedURL = true
		cfg.SoftwareInstallersGCSIAMAuth = true

		_, err := NewSoftwareInstallerStore(cfg)
		require.ErrorContains(t, err, "gcs iam auth")
	})

	t.Run("signed url with sts assume role is rejected", func(t *testing.T) {
		// STS assume-role swaps the HMAC credentials presigning needs for
		// temporary AWS credentials GCS can't verify, so store init must fail.
		cfg := baseCfg()
		cfg.SoftwareInstallersSignedURL = true
		cfg.SoftwareInstallersStsAssumeRoleArn = "arn:aws:iam::123456789012:role/test"

		_, err := NewSoftwareInstallerStore(cfg)
		require.ErrorContains(t, err, "sts assume role")
	})
}
