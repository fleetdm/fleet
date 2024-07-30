package tables

import (
	"context"
	"errors"
	"github.com/VividCortex/mysqlerr"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUp_20240226082255(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	isDuplicate := func(err error) bool {
		err = ctxerr.Cause(err)
		var driverErr *mysql.MySQLError
		if errors.As(err, &driverErr) && driverErr.Number == mysqlerr.ER_DUP_ENTRY {
			return true
		}
		return false
	}

	// Insert 2 teams with emoji names
	_ = execNoErrLastID(t, db, "INSERT INTO teams (name) VALUES (?)", "🖥️")
	_ = execNoErrLastID(t, db, "INSERT INTO teams (name) VALUES (?)", "💿")
	// Try to insert a duplicate team name -- should error
	_, err := db.Exec("INSERT INTO teams (name) VALUES (?)", "🖥️")
	assert.True(t, isDuplicate(err))
	var count []uint
	err = db.SelectContext(context.Background(), &count, `SELECT COUNT(*) FROM teams`)
	assert.NoError(t, err)
	assert.Equal(t, uint(2), count[0])

}
