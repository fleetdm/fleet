package tables

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261109155711(t *testing.T) {
	db := applyUpToPrev(t)

	type endpoint struct {
		Method string `db:"method"`
		Path   string `db:"path"`
	}
	const (
		oldByName = "/api/v1/fleet/reports/:name"
		oldByID   = "/api/v1/fleet/reports/id/:id"
		newByID   = "/api/v1/fleet/reports/:id"
	)
	del := func(path string) endpoint { return endpoint{Method: "DELETE", Path: path} }

	cases := []struct {
		name   string
		before []endpoint
		after  []endpoint
	}{
		{
			// An empty grant list means full access, so the only grant can't just be dropped.
			name:   "by name as the only grant becomes by ID",
			before: []endpoint{del(oldByName)},
			after:  []endpoint{del(newByID)},
		},
		{
			name:   "by ID keeps the deprecated path and gains the new by-ID path",
			before: []endpoint{del(oldByID)},
			after:  []endpoint{del(oldByID), del(newByID)},
		},
		{
			name:   "both",
			before: []endpoint{del(oldByName), del(oldByID)},
			after:  []endpoint{del(oldByID), del(newByID)},
		},
		{
			name: "non-catalog placeholder names, version placeholder, and lowercase method",
			before: []endpoint{
				{Method: "delete", Path: "/api/_version_/fleet/reports/{report_name}"},
				del("/api/v1/fleet/reports/id/{report_id}"),
			},
			after: []endpoint{del(oldByID), del(newByID)},
		},
		{
			// The old by-name grant's exact string equals the new by-ID path.
			name:   "by name saved with an :id placeholder",
			before: []endpoint{del("/api/v1/fleet/reports/:id")},
			after:  []endpoint{del(newByID)},
		},
		{
			name:   "by name saved with an :id placeholder alongside by ID",
			before: []endpoint{del("/api/v1/fleet/reports/:id"), del(oldByID)},
			after:  []endpoint{del(oldByID), del(newByID)},
		},
		{
			name: "unrelated grants are untouched",
			before: []endpoint{
				{Method: "GET", Path: "/api/v1/fleet/reports/:id"},
				{Method: "PATCH", Path: "/api/v1/fleet/reports/:id"},
				{Method: "POST", Path: "/api/v1/fleet/reports/delete"},
				del("/api/v1/fleet/labels/:name"),
				del("/api/v1/fleet/packs/id/:id"),
			},
			after: []endpoint{
				{Method: "GET", Path: "/api/v1/fleet/reports/:id"},
				{Method: "PATCH", Path: "/api/v1/fleet/reports/:id"},
				{Method: "POST", Path: "/api/v1/fleet/reports/delete"},
				del("/api/v1/fleet/labels/:name"),
				del("/api/v1/fleet/packs/id/:id"),
			},
		},
	}

	userIDs := make([]int64, len(cases))
	for i, tc := range cases {
		userIDs[i] = execNoErrLastID(t, db,
			`INSERT INTO users (email, password, salt, api_only) VALUES (?, 'x', 'x', 1)`,
			fmt.Sprintf("api-only-%d@example.com", i))
		for _, ep := range tc.before {
			execNoErr(t, db, `INSERT INTO user_api_endpoints (user_id, path, method) VALUES (?, ?, ?)`,
				userIDs[i], ep.Path, ep.Method)
		}
	}

	applyNext(t, db)

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []endpoint
			require.NoError(t, db.Select(&got, `SELECT method, path FROM user_api_endpoints WHERE user_id = ?`, userIDs[i]))
			require.ElementsMatch(t, tc.after, got)
		})
	}
}
