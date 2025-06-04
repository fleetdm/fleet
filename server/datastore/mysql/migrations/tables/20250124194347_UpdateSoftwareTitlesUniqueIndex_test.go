package tables

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20250124194347(t *testing.T) {
	db := applyUpToPrev(t)

	insertSql := `INSERT INTO software_titles (name, source, browser, bundle_identifier) VALUES (?, ?, ?, ?);`
	_, err := db.Exec(insertSql, "name1", "", "", "com.fleet1")
	require.NoError(t, err)
	_, err = db.Exec(insertSql, "name1", "", "", "com.fleet2")
	require.Error(t, err, "Expected software insert to fail because of unique key")

	applyNext(t, db)

	_, err = db.Exec(insertSql, "name2", "", "", "com.fleetdm1")
	require.NoError(t, err)
	_, err = db.Exec(insertSql, "name2", "", "", "com.fleetdm2")
	require.NoError(t, err, "Expected software insert to succeed")

	insertSql = `INSERT INTO software_titles (name, source, browser, bundle_identifier) VALUES`
	var valueStrings []string
	var valueArgs []interface{}

	for i := 0; i < 10; i++ {
		valueStrings = append(valueStrings, "(?, ?, ?, ?)")
		source := ""
		if i%2 == 0 {
			source = "app"
		} else {
			source = ""
		}
		valueArgs = append(valueArgs, fmt.Sprintf("name_%d", i), source, "", fmt.Sprintf("bundle_%d", i))
	}
	_, err = db.Exec(insertSql+strings.Join(valueStrings, ","), valueArgs...)
	require.NoError(t, err)

	result := struct {
		ID           int     `db:"id"`
		SelectType   string  `db:"select_type"`
		Table        string  `db:"table"`
		Type         string  `db:"type"`
		PossibleKeys *string `db:"possible_keys"`
		Key          *string `db:"key"`
		KeyLen       *int    `db:"key_len"`
		Ref          *string `db:"ref"`
		Rows         int     `db:"rows"`
		Filtered     float64 `db:"filtered"`
		Extra        *string `db:"Extra"`
		Partitions   *string `db:"partitions"`
	}{}

	err = db.Get(
		&result, `EXPLAIN SELECT id from software_titles WHERE name = ? and source = ?`,
		"name1", "app",
	)
	require.NoError(t, err)
	require.Equal(t, *result.Key, "idx_sw_titles")
}
