package s3

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSoftwareInstaller(t *testing.T) {
	ctx := context.Background()
	store := SetupTestSoftwareInstallerStore(t, "software-installers-unit-test", "prefix")

	// get a non-existing installer
	blob, length, err := store.Get(ctx, "no-such-installer")
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
	require.Nil(t, blob)
	require.Zero(t, length)

	exists, err := store.Exists(ctx, "no-such-installer")
	require.NoError(t, err)
	require.False(t, exists)

	createInstallerAndHash := func() ([]byte, string) {
		b := make([]byte, 1024)
		_, err = rand.Read(b)
		require.NoError(t, err)

		h := sha256.New()
		_, err = h.Write(b)
		require.NoError(t, err)
		installerID := hex.EncodeToString(h.Sum(nil))

		return b, installerID
	}

	getAndCheck := func(installerID string, expected []byte) {
		rc, sz, err := store.Get(ctx, installerID)
		require.NoError(t, err)
		require.EqualValues(t, len(expected), sz)
		defer rc.Close()

		got, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.Equal(t, expected, got)

		exists, err := store.Exists(ctx, installerID)
		require.NoError(t, err)
		require.True(t, exists)
	}

	// store an installer
	b0, id0 := createInstallerAndHash()
	err = store.Put(ctx, id0, bytes.NewReader(b0))
	require.NoError(t, err)

	// read it back, it should match
	getAndCheck(id0, b0)

	// store another one
	b1, id1 := createInstallerAndHash()
	err = store.Put(ctx, id1, bytes.NewReader(b1))
	require.NoError(t, err)

	// read it back, it should match
	getAndCheck(id1, b1)

	// replace the first one
	err = store.Put(ctx, id0, bytes.NewReader(b0))
	require.NoError(t, err)

	// read it back, it should still match
	getAndCheck(id0, b0)
}

func TestSoftwareInstallerCleanup(t *testing.T) {
	ctx := context.Background()
	store := SetupTestSoftwareInstallerStore(t, "software-installers-unit-test", "prefix")

	assertExisting := func(want []string) {
		prefix := path.Join(store.prefix, softwareInstallersPrefix)
		page, err := store.s3client.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket: &store.bucket,
			Prefix: &prefix,
		})
		require.NoError(t, err)

		got := make([]string, 0, len(page.Contents))
		for _, item := range page.Contents {
			got = append(got, path.Base(*item.Key))
		}
		require.ElementsMatch(t, want, got)
	}

	// cleanup an empty store
	n, err := store.Cleanup(ctx, nil, time.Now())
	require.NoError(t, err)
	require.Equal(t, 0, n)

	// put an installer
	ins0 := uuid.NewString()
	err = store.Put(ctx, ins0, bytes.NewReader([]byte("installer0")))
	require.NoError(t, err)

	// cleanup but mark it as used
	n, err = store.Cleanup(ctx, []string{ins0}, time.Now())
	require.NoError(t, err)
	require.Equal(t, 0, n)

	assertExisting([]string{ins0})

	// cleanup but mark it as unused
	n, err = store.Cleanup(ctx, []string{}, time.Now())
	require.NoError(t, err)
	require.Equal(t, 1, n)

	assertExisting(nil)

	// put a few installers
	installers := []string{uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()}
	for i, ins := range installers {
		err = store.Put(ctx, ins, bytes.NewReader([]byte("installer"+fmt.Sprint(i))))
		require.NoError(t, err)
	}

	// cleanup with a time in the past, nothing gets removed
	n, err = store.Cleanup(ctx, []string{}, time.Now().Add(-time.Minute))
	require.NoError(t, err)
	require.Equal(t, 0, n)
	assertExisting([]string{installers[0], installers[1], installers[2], installers[3]})

	// cleanup in the future, all unused get removed
	n, err = store.Cleanup(ctx, []string{installers[0], installers[2]}, time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, 2, n)
	assertExisting([]string{installers[0], installers[2]})
}
