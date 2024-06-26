package file

import (
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage/internal/test"
)

func TestQueue(t *testing.T) {
	storage, err := New("test-db")
	if err != nil {
		t.Fatal(err)
	}
	test.TestQueue(t, "EA4E19F1-7F8B-493D-BEAB-264B33BCF4E6", storage)
	os.RemoveAll("test-db")
}
