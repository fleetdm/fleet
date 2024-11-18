package test

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
)

func TestRetrievePushInfo(t *testing.T, ctx context.Context, s storage.PushStore) {
	t.Run("TestRetrievePushInfo", func(t *testing.T) {
		_, err := s.RetrievePushInfo(ctx, []string{"INVALID"})
		if err != nil {
			// should NOT recieve a "global" error for an enrollment that
			// is merely invalid (or not enrolled yet, or not fully enrolled)
			t.Errorf("should NOT have errored: %v", err)
		}
	})
}
