package e2e

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
)

type tokenTallyDevice interface {
	DoTokenUpdate(context.Context) error
	IDer
}

// tally tests to make sure the TokenUpdate tally functions nominally.
func tally(t *testing.T, ctx context.Context, d tokenTallyDevice, store storage.TokenUpdateTallyStore, initial int) {
	// retrieve the tally
	tally, err := store.RetrieveTokenUpdateTally(ctx, d.ID())
	if err != nil {
		t.Fatal()
	}

	// make sure it's what we want
	if have, want := tally, initial; have != want {
		t.Errorf("token update tally: have: %v, want: %v", have, want)
	}

	// perform a TokenUpdate (should increase the tally)
	err = d.DoTokenUpdate(ctx)
	if err != nil {
		t.Fatal()
	}

	// retrieve the tally again
	tally, err = store.RetrieveTokenUpdateTally(ctx, d.ID())
	if err != nil {
		t.Fatal()
	}

	// make sure it's what we want (+1)
	if have, want := tally, initial+1; have != want {
		t.Errorf("token update tally (2nd): have: %v, want: %v", have, want)
	}
}
