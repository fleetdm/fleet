package pgsql

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

// RetrievePushInfo retreives push info for identifiers ids.
//
// Note that we may return fewer results than input. The user of this
// method needs to reconcile that with their requested ids.
func (s *PgSQLStorage) RetrievePushInfo(ctx context.Context, ids []string) (map[string]*mdm.Push, error) {
	if len(ids) < 1 {
		return nil, errors.New("no ids provided")
	}

	// previous: `SELECT id, topic, push_magic, token_hex FROM enrollments WHERE id IN (`+qs+`);`,
	// refactor all strings concatenations with strings.Builder which is more efficient
	var qs strings.Builder

	qs.WriteString(`SELECT id, topic, push_magic, token_hex FROM enrollments WHERE id IN (`)
	args := make([]interface{}, len(ids))
	for i, v := range ids {
		args[i] = v
		if i > 0 {
			qs.WriteString(",")
		}
		// can be a bit faster than fmt.Fprintf(&qs, "$%d", i+1)
		qs.WriteString("$")
		qs.WriteString(strconv.Itoa(i + 1))
	}
	qs.WriteString(`);`)

	rows, err := s.db.QueryContext(ctx, qs.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pushInfos := make(map[string]*mdm.Push)
	for rows.Next() {
		push := new(mdm.Push)
		var id, token string
		if err := rows.Scan(&id, &push.Topic, &push.PushMagic, &token); err != nil {
			return nil, err
		}
		// convert from hex
		if err := push.SetTokenString(token); err != nil {
			return nil, err
		}
		pushInfos[id] = push
	}
	return pushInfos, rows.Err()
}
