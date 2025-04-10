package e2e

import (
	"context"
	"reflect"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
)

type enrollDevice interface {
	IDer
	DoEnroll(context.Context) error
	GetPush() *mdm.Push
}

func enroll(t *testing.T, ctx context.Context, d enrollDevice, store storage.PushStore) {
	// enroll it
	err := d.DoEnroll(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// extract the push info for the given id
	pushInfos, err := store.RetrievePushInfo(ctx, []string{d.ID()})
	if err != nil {
		t.Fatal(err)
	}

	// test that we got the right push data data back
	if want, have := 1, len(pushInfos); want != have {
		t.Fatalf("len(pushInfos): want: %v, have: %v", want, have)
	}
	push := d.GetPush()
	if !reflect.DeepEqual(pushInfos[d.ID()], push) {
		t.Errorf("pushInfo have: %v, want: %v", pushInfos[d.ID()], push)
	}
}
