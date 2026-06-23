package mysql

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

// MasterStatus is a struct that holds the file and position of the master,
// retrieved by SHOW MASTER STATUS / SHOW BINARY LOG STATUS.
type MasterStatus struct {
	File     string
	Position uint64
}

// MasterStatus returns the current master file and position for this
// datastore's writer.
func (ds *Datastore) MasterStatus(ctx context.Context, mysqlVersion string) (MasterStatus, error) {
	stmt := "SHOW BINARY LOG STATUS"
	if strings.HasPrefix(mysqlVersion, "8.0") {
		stmt = "SHOW MASTER STATUS"
	}

	rows, err := ds.writer(ctx).QueryContext(ctx, stmt)
	if err != nil {
		return MasterStatus{}, ctxerr.Wrap(ctx, err, stmt)
	}
	defer rows.Close()

	// Since we don't control the column names, and we want to be future compatible,
	// we only scan for the columns we care about.
	ms := MasterStatus{}
	// Get the column names from the query
	columns, err := rows.Columns()
	if err != nil {
		return ms, ctxerr.Wrap(ctx, err, "get columns")
	}
	numberOfColumns := len(columns)
	for rows.Next() {
		cols := make([]any, numberOfColumns)
		for i := range cols {
			cols[i] = new(string)
		}
		err := rows.Scan(cols...)
		if err != nil {
			return ms, ctxerr.Wrap(ctx, err, "scan row")
		}
		for i, col := range cols {
			switch columns[i] {
			case "File":
				ms.File = *col.(*string)
			case "Position":
				ms.Position, err = strconv.ParseUint(*col.(*string), 10, 64)
				if err != nil {
					return ms, ctxerr.Wrap(ctx, err, "parse Position")
				}

			}
		}
	}
	if err := rows.Err(); err != nil {
		return ms, ctxerr.Wrap(ctx, err, "rows error")
	}
	if ms.File == "" || ms.Position == 0 {
		return ms, ctxerr.New(ctx, "missing required fields in master status")
	}
	return ms, nil
}

// ReplicaStatus returns the SHOW REPLICA STATUS row as a map keyed by column
// name.
func (ds *Datastore) ReplicaStatus(ctx context.Context) (map[string]any, error) {
	rows, err := ds.reader(ctx).QueryContext(ctx, "SHOW REPLICA STATUS")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "show replica status")
	}
	defer rows.Close()

	// Get the column names from the query
	columns, err := rows.Columns()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get columns")
	}
	numberOfColumns := len(columns)
	result := make(map[string]any, numberOfColumns)
	for rows.Next() {
		cols := make([]any, numberOfColumns)
		for i := range cols {
			cols[i] = &sql.NullString{}
		}
		err = rows.Scan(cols...)
		if err != nil {
			return result, ctxerr.Wrap(ctx, err, "scan row")
		}
		for i, col := range cols {
			colValue := col.(*sql.NullString)
			if colValue.Valid {
				result[columns[i]] = colValue.String
			} else {
				result[columns[i]] = nil
			}
		}
	}
	if err := rows.Err(); err != nil {
		return result, ctxerr.Wrap(ctx, err, "rows error")
	}
	return result, nil
}
