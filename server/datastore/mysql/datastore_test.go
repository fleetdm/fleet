package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore"
	_ "github.com/go-sql-driver/mysql"
)

func TestMySQL(t *testing.T) {
	RunTestsAgainstMySQL(t, datastore.TestFunctions)
}
