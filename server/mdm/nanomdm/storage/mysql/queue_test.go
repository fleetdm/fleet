package mysql

import (
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage/internal/test"

	_ "github.com/go-sql-driver/mysql"
)

func TestQueue(t *testing.T) {
	testDSN := os.Getenv("NANOMDM_MYSQL_STORAGE_TEST_DSN")
	if testDSN == "" {
		t.Skip("NANOMDM_MYSQL_STORAGE_TEST_DSN not set")
	}

	storage, err := New(WithDSN(testDSN), WithDeleteCommands())
	if err != nil {
		t.Fatal(err)
	}

	d, err := enrollTestDevice(storage)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("WithDeleteCommands()", func(t *testing.T) {
		test.TestQueue(t, d.UDID, storage)
	})

	storage, err = New(WithDSN(testDSN))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("normal", func(t *testing.T) {
		test.TestQueue(t, d.UDID, storage)
	})
}
