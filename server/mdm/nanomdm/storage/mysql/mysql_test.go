package mysql

import (
	"context"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/test/e2e"
	_ "github.com/go-sql-driver/mysql"
)

func TestMySQL(t *testing.T) {
	testDSN := os.Getenv("NANOMDM_MYSQL_STORAGE_TEST_DSN")
	if testDSN == "" {
		t.Skip("NANOMDM_MYSQL_STORAGE_TEST_DSN not set")
	}

	s, err := New(WithDSN(testDSN), WithDeleteCommands())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("e2e-WithDeleteCommands()", func(t *testing.T) { e2e.TestE2E(t, context.Background(), s) })

	s, err = New(WithDSN(testDSN))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("e2e", func(t *testing.T) { e2e.TestE2E(t, context.Background(), s) })
}
