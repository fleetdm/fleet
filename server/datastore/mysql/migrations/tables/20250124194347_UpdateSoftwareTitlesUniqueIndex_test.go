package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20250124194347(t *testing.T) {
	db := applyUpToPrev(t)

	var softwareTitles []struct {
		ColumnName string `db:"COLUMN_NAME"`
	}

	sel := `SELECT COLUMN_NAME 
			FROM information_schema.statistics 
			WHERE table_schema = DATABASE() 
			AND table_name = 'software_titles'
			AND index_name = 'idx_sw_titles'
			ORDER BY seq_in_index;`

	err := db.Select(&softwareTitles, sel)
	if err != nil {
		t.Fatalf("Failed to get index information: %v", err)
	}
	expected := []struct {
		ColumnName string `db:"COLUMN_NAME"`
	}{
		{ColumnName: "name"},
		{ColumnName: "source"},
		{ColumnName: "browser"},
	}
	require.Equal(t, expected, softwareTitles)

	applyNext(t, db)

	err = db.Select(&softwareTitles, sel)
	if err != nil {
		t.Fatalf("Failed to get index information: %v", err)
	}
	expected = []struct {
		ColumnName string `db:"COLUMN_NAME"`
	}{
		{ColumnName: "name"},
		{ColumnName: "source"},
		{ColumnName: "browser"},
		{ColumnName: "bundle_identifier"},
	}
	require.Equal(t, expected, softwareTitles)
}
