package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20261111205515(t *testing.T) {
	for _, tc := range []struct {
		name string
		old  string
		want string
	}{
		{
			name: "old keys renamed",
			old: `{"apple_business_manager": [
					{"organization_name": "Org A", "macos_team": "Workstations", "ios_team": "", "ipados_team": "Tablets", "byod_team": "BYOD"},
					{"organization_name": "Org B", "default": true, "macos_team": "", "ios_team": "", "ipados_team": "", "byod_team": ""}],
				"volume_purchasing_program": [
					{"location": "Loc A", "teams": ["Workstations", "BYOD"]}, {"location": "Loc B", "teams": []}]}`,
			want: `{"apple_business_manager": [
					{"organization_name": "Org A", "macos_fleet": "Workstations", "ios_fleet": "", "ipados_fleet": "Tablets", "byod_fleet": "BYOD"},
					{"organization_name": "Org B", "default": true, "macos_fleet": "", "ios_fleet": "", "ipados_fleet": "", "byod_fleet": ""}],
				"volume_purchasing_program": [
					{"location": "Loc A", "fleets": ["Workstations", "BYOD"]}, {"location": "Loc B", "fleets": []}]}`,
		},
		{
			name: "new key wins over old",
			old: `{"apple_business_manager": [{"organization_name": "Org A", "macos_team": "Old", "macos_fleet": "New"}],
				"volume_purchasing_program": [{"location": "Loc A", "teams": ["Old"], "fleets": ["New"]}]}`,
			want: `{"apple_business_manager": [{"organization_name": "Org A", "macos_fleet": "New"}],
				"volume_purchasing_program": [{"location": "Loc A", "fleets": ["New"]}]}`,
		},
		{
			name: "null and missing lists",
			old:  `{"apple_business_manager": null}`,
			want: `{"apple_business_manager": null}`,
		},
		{
			name: "already migrated",
			old:  `{"apple_business_manager": [{"organization_name": "Org A", "macos_fleet": "Workstations"}]}`,
			want: `{"apple_business_manager": [{"organization_name": "Org A", "macos_fleet": "Workstations"}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := applyUpToPrev(t)
			execNoErr(t, db, `UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.mdm', CAST(? AS JSON)) WHERE id = 1`, tc.old)
			const otherSettings = `SELECT JSON_REMOVE(json_value, '$.mdm') FROM app_config_json WHERE id = 1`
			var othersBefore, othersAfter, mdm string
			require.NoError(t, db.Get(&othersBefore, otherSettings))

			applyNext(t, db)

			require.NoError(t, db.Get(&mdm, `SELECT json_value->'$.mdm' FROM app_config_json WHERE id = 1`))
			assert.JSONEq(t, tc.want, mdm)
			require.NoError(t, db.Get(&othersAfter, otherSettings))
			assert.JSONEq(t, othersBefore, othersAfter, "settings outside mdm are not touched")
		})
	}
}
