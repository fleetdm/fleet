package tables

import (
	"database/sql"
	"fmt"
	"regexp"
)

func init() {
	MigrationClient.AddMigration(Up_20260618124430, Down_20260618124430)
}

var fixEmbeddedTrailingNonWordChars = regexp.MustCompile(`\W+$`)

func Up_20260618124430(tx *sql.Tx) error {
	fmaNames, err := fixEmbeddedLoadFMANamesDarwin(tx)
	if err != nil {
		return fmt.Errorf("loading FMA names: %w", err)
	}

	rows, err := tx.Query(`
		SELECT st.id, st.name, st.bundle_identifier, s.name
		FROM software_titles st
		JOIN software s ON s.title_id = st.id
		WHERE st.source = 'apps'
		  AND st.bundle_identifier IS NOT NULL
		  AND st.bundle_identifier != ''
		ORDER BY st.id
	`)
	if err != nil {
		return fmt.Errorf("scanning macOS app titles: %w", err)
	}

	type titleInfo struct {
		currentName string
		bundleID    string
		siblings    map[string]struct{}
	}
	titles := make(map[int64]*titleInfo)

	for rows.Next() {
		var titleID int64
		var currentName, bundleID, softwareName string
		if err := rows.Scan(&titleID, &currentName, &bundleID, &softwareName); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scanning title row: %w", err)
		}
		t, ok := titles[titleID]
		if !ok {
			t = &titleInfo{currentName: currentName, bundleID: bundleID, siblings: make(map[string]struct{})}
			titles[titleID] = t
		}
		t.siblings[softwareName] = struct{}{}
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("closing title rows: %w", err)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating title rows: %w", err)
	}

	updateStmt, err := tx.Prepare(`UPDATE software_titles SET name = ? WHERE id = ? AND name != ?`)
	if err != nil {
		return fmt.Errorf("preparing update: %w", err)
	}
	defer updateStmt.Close()

	for titleID, t := range titles {
		if len(t.siblings) < 2 {
			continue
		}
		newName := fixEmbeddedPickTitleName(t.siblings, t.bundleID, fmaNames)
		if newName == "" || newName == t.currentName {
			continue
		}
		if _, err := updateStmt.Exec(newName, titleID, newName); err != nil {
			return fmt.Errorf("updating title %d: %w", titleID, err)
		}
	}
	return nil
}

func fixEmbeddedLoadFMANamesDarwin(tx *sql.Tx) (map[string]string, error) {
	rows, err := tx.Query(`SELECT unique_identifier, name FROM fleet_maintained_apps WHERE platform = 'darwin'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out[id] = name
	}
	return out, rows.Err()
}

// Mirrors preInsertSoftwareInventory's precedence in server/datastore/mysql/software.go.
func fixEmbeddedPickTitleName(siblings map[string]struct{}, bundleID string, fmaNames map[string]string) string {
	if name, ok := fmaNames[bundleID]; ok && name != "" {
		return name
	}
	names := make([]string, 0, len(siblings))
	for n := range siblings {
		names = append(names, n)
	}
	prefix := fixEmbeddedLongestCommonPrefix(names)
	prefix = fixEmbeddedTrailingNonWordChars.ReplaceAllString(prefix, "")
	if prefix != "" {
		return prefix
	}
	shortest := names[0]
	for _, n := range names[1:] {
		if len(n) < len(shortest) {
			shortest = n
		}
	}
	return shortest
}

func fixEmbeddedLongestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	firstLen := len(strs[0])
	i := 0
	for {
		if i >= firstLen {
			return strs[0]
		}
		c := strs[0][i]
		for _, s := range strs[1:] {
			if i >= len(s) || s[i] != c {
				return strs[0][:i]
			}
		}
		i++
	}
}

func Down_20260618124430(tx *sql.Tx) error {
	return nil
}
