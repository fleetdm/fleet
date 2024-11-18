package file

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage/test"
)

func TestFileStorage(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	test.TestQueue(t, "EA4E19F1-7F8B-493D-BEAB-264B33BCF4E6", s)
	test.TestRetrievePushInfo(t, context.Background(), s)
}
