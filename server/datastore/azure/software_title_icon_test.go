package azure

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

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSoftwareTitleIcon(t *testing.T) {
	ctx := context.Background()
	store := setupTestSoftwareTitleIconStore(t, "software-title-icons-unit-test")

	// get a non-existing icon
	blob, length, err := store.Get(ctx, "no-such-icon")
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
	require.Nil(t, blob)
	require.Zero(t, length)

	exists, err := store.Exists(ctx, "no-such-icon")
	require.NoError(t, err)
	require.False(t, exists)

	createIconAndHash := func() ([]byte, string) {
		b := make([]byte, 1024)
		_, err = rand.Read(b)
		require.NoError(t, err)

		h := sha256.New()
		_, err = h.Write(b)
		require.NoError(t, err)
		iconID := hex.EncodeToString(h.Sum(nil))

		return b, iconID
	}

	getAndCheck := func(iconID string, expected []byte) {
		rc, sz, err := store.Get(ctx, iconID)
		require.NoError(t, err)
		require.EqualValues(t, len(expected), sz)
		defer rc.Close()

		got, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.Equal(t, expected, got)

		exists, err := store.Exists(ctx, iconID)
		require.NoError(t, err)
		require.True(t, exists)
	}

	// store an icon
	b0, id0 := createIconAndHash()
	err = store.Put(ctx, id0, bytes.NewReader(b0))
	require.NoError(t, err)

	// read it back, it should match
	getAndCheck(id0, b0)

	// store another one
	b1, id1 := createIconAndHash()
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

func TestSoftwareTitleIconCleanup(t *testing.T) {
	ctx := context.Background()
	store := setupTestSoftwareTitleIconStore(t, "software-title-icons-unit-test")

	assertExisting := func(want []string) {
		prefix := "software-title-icons/"
		pager := store.client.NewListBlobsFlatPager(store.container, nil)

		var got []string
		for pager.More() {
			page, err := pager.NextPage(ctx)
			require.NoError(t, err)
			if page.Segment == nil {
				continue
			}
			for _, item := range page.Segment.BlobItems {
				if item.Name == nil {
					continue
				}
				require.True(t, len(*item.Name) > len(prefix) && (*item.Name)[:len(prefix)] == prefix)
				got = append(got, path.Base(*item.Name))
			}
		}
		require.ElementsMatch(t, want, got)
	}

	// cleanup an empty store
	n, err := store.Cleanup(ctx, nil, time.Now())
	require.NoError(t, err)
	require.Equal(t, 0, n)

	// put an icon
	ic0 := uuid.NewString()
	err = store.Put(ctx, ic0, bytes.NewReader([]byte("icon0")))
	require.NoError(t, err)

	// cleanup but mark it as used
	n, err = store.Cleanup(ctx, []string{ic0}, time.Now())
	require.NoError(t, err)
	require.Equal(t, 0, n)

	assertExisting([]string{ic0})

	// cleanup but mark it as unused
	n, err = store.Cleanup(ctx, []string{}, time.Now())
	require.NoError(t, err)
	require.Equal(t, 1, n)

	assertExisting(nil)

	// put a few icons
	icons := []string{uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()}
	for i, ic := range icons {
		err = store.Put(ctx, ic, bytes.NewReader([]byte("icon"+fmt.Sprint(i))))
		require.NoError(t, err)
	}

	// cleanup with a time in the past, nothing gets removed
	n, err = store.Cleanup(ctx, []string{}, time.Now().Add(-time.Minute))
	require.NoError(t, err)
	require.Equal(t, 0, n)
	assertExisting([]string{icons[0], icons[1], icons[2], icons[3]})

	// cleanup in the future, all unused get removed
	n, err = store.Cleanup(ctx, []string{icons[0], icons[2]}, time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, 2, n)
	assertExisting([]string{icons[0], icons[2]})
}
