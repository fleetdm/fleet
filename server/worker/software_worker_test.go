package worker

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
)

func TestSoftwareWorker(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	// call TruncateTables immediately as some DB migrations may create jobs
	mysql.TruncateTables(t, ds)

	mysql.SetTestABMAssets(t, ds, "fleet")

}
