package s3

import (
	"bytes"
	"io"
	"net/http"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestStagedUploadStore(t *testing.T) {
	ctx := t.Context()
	const bucket, prefix = "staged-uploads-unit-test", "prefix"
	staged := setupTestStore(t, bucket, prefix, func(cfg config.S3Config) (*StagedUploadStore, error) {
		cfg.SoftwareInstallersSignedURL = true
		return NewStagedUploadStore(cfg)
	})
	installers := setupTestSoftwareInstallerStore(t, bucket, prefix)
	bootstrap := setupTestBootstrapPackageStore(t, bucket, prefix)
	icons := setupTestStore(t, bucket, prefix, NewSoftwareTitleIconStore)

	put := func(url string, body string, size int64) int {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, strings.NewReader(body))
		require.NoError(t, err)
		req.ContentLength = size
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			// the server may drop the connection on a length mismatch
			return 0
		}
		resp.Body.Close()
		return resp.StatusCode
	}

	t.Run("presigned put pins the size", func(t *testing.T) {
		url, err := staged.PresignPut(ctx, "up1", 5, time.Hour)
		require.NoError(t, err)
		require.NotEqual(t, http.StatusOK, put(url, "too long", 8))
		_, _, err = staged.Get(ctx, "up1")
		require.True(t, fleet.IsNotFound(err))

		require.Equal(t, http.StatusOK, put(url, "hello", 5))
		rc, size, err := staged.Get(ctx, "up1")
		require.NoError(t, err)
		b, err := io.ReadAll(rc)
		rc.Close()
		require.NoError(t, err)
		require.Equal(t, "hello", string(b))
		require.EqualValues(t, 5, size)

		require.NoError(t, staged.Delete(ctx, "up1"))
		_, _, err = staged.Get(ctx, "up1")
		require.True(t, fleet.IsNotFound(err))
		require.NoError(t, staged.Delete(ctx, "up1"))
	})

	t.Run("sweeps are isolated by prefix", func(t *testing.T) {
		require.NoError(t, staged.Put(ctx, "up2", bytes.NewReader([]byte("staged"))))
		require.NoError(t, installers.Put(ctx, "ins", bytes.NewReader([]byte("installer"))))
		require.NoError(t, bootstrap.Put(ctx, "bp", bytes.NewReader([]byte("bootstrap"))))
		require.NoError(t, icons.Put(ctx, "icon", bytes.NewReader([]byte("icon"))))

		listAll := func() []string {
			page, err := staged.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: &staged.bucket})
			require.NoError(t, err)
			var keys []string
			for _, item := range page.Contents {
				keys = append(keys, *item.Key)
			}
			return keys
		}
		all := []string{
			path.Join(prefix, "uploads/up2"),
			path.Join(prefix, "software-installers/ins"),
			path.Join(prefix, "bootstrap-packages/bp"),
			path.Join(prefix, "software-title-icons/icon"),
		}
		require.ElementsMatch(t, all, listAll())

		future := time.Now().Add(time.Minute)
		_, err := bootstrap.Cleanup(ctx, nil, future)
		require.NoError(t, err)
		_, err = installers.Cleanup(ctx, nil, future)
		require.NoError(t, err)
		_, err = icons.Cleanup(ctx, nil, future)
		require.NoError(t, err)
		require.ElementsMatch(t, all[:1], listAll())

		require.NoError(t, installers.Put(ctx, "ins", bytes.NewReader([]byte("installer"))))
		n, err := staged.Cleanup(ctx, nil, future)
		require.NoError(t, err)
		require.Equal(t, 1, n)
		require.ElementsMatch(t, all[1:2], listAll())
	})
}
