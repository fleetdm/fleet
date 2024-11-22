package file

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/test/e2e"
)

func TestFileStorage(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("e2e", func(t *testing.T) { e2e.TestE2E(t, context.Background(), s) })
}
