package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20261117174518(t *testing.T) {
	db := applyUpToPrev(t)

	setServerSettings := func(t *testing.T, settings string) {
		execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.server_settings', CAST(? AS JSON)) WHERE id = 1`, settings)
	}
	serverSettings := func(t *testing.T) string {
		var settings string
		require.NoError(t, sqlx.Get(db, &settings, `SELECT json_value->'$.server_settings' FROM app_config_json WHERE id = 1`))
		return settings
	}
	runAgain := func(t *testing.T) {
		tx, err := db.Begin()
		require.NoError(t, err)
		require.NoError(t, Up_20261117174518(tx))
		require.NoError(t, tx.Commit())
	}

	setServerSettings(t, `{"server_url": "https://example.com", "live_query_disabled": true, "query_reports_disabled": true, "query_report_cap": 2500, "mcp_disabled": true}`)
	applyNext(t, db)
	require.JSONEq(t,
		`{"server_url": "https://example.com", "live_reporting_disabled": true, "discard_reports_data": true, "report_cap": 2500, "mcp_disabled": true}`,
		serverSettings(t))

	for _, tc := range []struct{ name, settings, want string }{
		{
			"zero values",
			`{"live_query_disabled": false, "query_reports_disabled": false, "query_report_cap": 0}`,
			`{"live_reporting_disabled": false, "discard_reports_data": false, "report_cap": 0}`,
		},
		{"missing", `{"server_url": "https://example.com"}`, `{"server_url": "https://example.com"}`},
		{"null", `null`, `null`},
		{
			// the old name holds the setting from before the upgrade
			"both names",
			`{"live_query_disabled": true, "live_reporting_disabled": false, "query_reports_disabled": false, "discard_reports_data": true, "query_report_cap": 50, "report_cap": 9}`,
			`{"live_reporting_disabled": true, "discard_reports_data": false, "report_cap": 50}`,
		},
		{
			"already renamed",
			`{"live_reporting_disabled": true, "discard_reports_data": true, "report_cap": 7}`,
			`{"live_reporting_disabled": true, "discard_reports_data": true, "report_cap": 7}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setServerSettings(t, tc.settings)
			runAgain(t)
			require.JSONEq(t, tc.want, serverSettings(t))

			// a second run finds nothing to move
			runAgain(t)
			require.JSONEq(t, tc.want, serverSettings(t))
		})
	}
}
