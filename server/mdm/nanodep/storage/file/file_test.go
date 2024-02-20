package file

import (
	"testing"

	"github.com/micromdm/nanodep/storage"
	"github.com/micromdm/nanodep/storage/storagetest"
)

func TestFileStorage(t *testing.T) {
	storagetest.Run(t, func(t *testing.T) storage.AllStorage {
		s, err := New(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		return s
	})
}
