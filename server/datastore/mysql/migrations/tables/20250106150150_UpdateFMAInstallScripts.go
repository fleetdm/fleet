package tables

import (
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20250106150150, Down_20250106150150)
}

func Up_20250106150150(tx *sql.Tx) error {
	var scriptContents []string
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	if err := txx.Select(&scriptContents, "SELECT sc.contents FROM fleet_library_apps fla JOIN script_contents sc ON fla.install_script_content_id = sc.id"); err != nil {
		return fmt.Errorf("selecting script contents: %w", err)
	}

	for _, sc := range scriptContents {
		fmt.Println("BEFORE:")
		fmt.Println(sc)
		fmt.Println()
		lines := strings.Split(sc, "\n")
		var index int
		for i, l := range lines {
			if strings.Contains(l, `sudo cp -R "$TMPDIR/`) {
				index = i
			}
		}
		lines = slices.Insert(lines, index, "YEEHAW")
		fmt.Println("AFTER")
		fmt.Println(strings.Join(lines, "\n"))

	}
	return nil
}

func Down_20250106150150(tx *sql.Tx) error {
	return nil
}
