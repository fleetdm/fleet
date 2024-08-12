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

func TestBootstrapPackage(t *testing.T) {
	ctx := context.Background()
	store := SetupTestBootstrapPackageStore(t, "bootstrap-packages-unit-test", "prefix")

	// get a non-existing package
	blob, length, err := store.Get(ctx, "no-such-package")
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
	require.Nil(t, blob)
	require.Zero(t, length)

	exists, err := store.Exists(ctx, "no-such-package")
	require.NoError(t, err)
	require.False(t, exists)

	createPackageAndHash := func() ([]byte, string) {
		b := make([]byte, 1024)
		_, err = rand.Read(b)
		require.NoError(t, err)

		h := sha256.New()
		_, err = h.Write(b)
		require.NoError(t, err)
		fileID := hex.EncodeToString(h.Sum(nil))

		return b, fileID
	}

	getAndCheck := func(fileID string, expected []byte) {
		rc, sz, err := store.Get(ctx, fileID)
		require.NoError(t, err)
		require.EqualValues(t, len(expected), sz)
		defer rc.Close()

		got, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.Equal(t, expected, got)

		exists, err := store.Exists(ctx, fileID)
		require.NoError(t, err)
		require.True(t, exists)
	}

	// store a package
	b0, id0 := createPackageAndHash()
	err = store.Put(ctx, id0, bytes.NewReader(b0))
	require.NoError(t, err)

	// read it back, it should match
	getAndCheck(id0, b0)

	// store another one
	b1, id1 := createPackageAndHash()
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

func TestBootstrapPackageCleanup(t *testing.T) {
	ctx := context.Background()
	store := SetupTestBootstrapPackageStore(t, "bootstrap-packages-unit-test", "prefix")

	assertExisting := func(want []string) {
		prefix := path.Join(store.prefix, bootstrapPackagePrefix)
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

	// put a package
	ins0 := uuid.NewString()
	err = store.Put(ctx, ins0, bytes.NewReader([]byte("package0")))
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

	// put a few packages
	packages := []string{uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()}
	for i, ins := range packages {
		err = store.Put(ctx, ins, bytes.NewReader([]byte("package"+fmt.Sprint(i))))
		require.NoError(t, err)
	}

	// cleanup with a time in the past, nothing gets removed
	n, err = store.Cleanup(ctx, []string{}, time.Now().Add(-time.Minute))
	require.NoError(t, err)
	require.Equal(t, 0, n)
	assertExisting([]string{packages[0], packages[1], packages[2], packages[3]})

	// cleanup in the future, all unused get removed
	n, err = store.Cleanup(ctx, []string{packages[0], packages[2]}, time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, 2, n)
	assertExisting([]string{packages[0], packages[2]})
}
