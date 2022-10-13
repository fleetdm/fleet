package tables

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
)

func init() {
	MigrationClient.AddMigration(Up_20221003154220, Down_20221003154220)
}

func powerSet(original []string) [][]string {
	powerSetSize := int(math.Pow(2, float64(len(original))))
	result := make([][]string, 0, powerSetSize)

	var index int
	for index < powerSetSize {
		var subSet []string

		for j, elem := range original {
			if index&(1<<uint(j)) > 0 {
				subSet = append(subSet, elem)
			}
		}
		result = append(result, subSet)
		index++
	}
	return result
}

func Up_20221003154220(tx *sql.Tx) error {
	nFeatures := 50

	columns := []string{
		"some_date",
		"some_enum_str",
		"some_str",
		"some_bool",
		"some_decimal",
		"some_number",
	}

	for i := 1; i <= nFeatures; i++ {
		stm := `
CREATE TABLE IF NOT EXISTS host_feature_%d (
id int(10) unsigned NOT NULL AUTO_INCREMENT,
host_id INT unsigned NOT NULL,
some_date timestamp NULL,
some_enum_str CHAR(10),
some_str VARCHAR(255),
some_bool TINYINT,
some_decimal DECIMAL(12,4),
some_number INT,
PRIMARY KEY (id),
%s
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
		`

		var indexStms []string
		for j, set := range powerSet(columns) {
			if len(set) > 0 {
				indexStms = append(indexStms, fmt.Sprintf(
					"INDEX host_feature_%d_%d_idx (host_id, %s)",
					i,
					j,
					strings.Join(set, ","),
				))
			}
		}

		stm = fmt.Sprintf(stm, i, strings.Join(indexStms, ",\n"))

		_, err := tx.Exec(stm)
		if err != nil {
			return err
		}
	}

	return nil
}

func Down_20221003154220(tx *sql.Tx) error {
	return nil
}
