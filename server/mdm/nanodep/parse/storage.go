package parse

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage/file"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage/mysql"
	_ "github.com/go-sql-driver/mysql"
)

// Storage parses a storage name and dsn to determine which and return a storage backend.
func Storage(storageName, dsn string) (storage.AllDEPStorage, error) {
	var store storage.AllDEPStorage
	var err error
	switch storageName {
	case "file":
		if dsn == "" {
			dsn = "db"
		}
		store, err = file.New(dsn)
	case "mysql":
		store, err = mysql.New(mysql.WithDSN(dsn))
	default:
		return nil, fmt.Errorf("unknown storage: %q", storageName)
	}
	return store, err
}
