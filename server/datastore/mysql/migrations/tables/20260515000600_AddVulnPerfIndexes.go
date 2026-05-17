package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260515000600, Down_20260515000600)
}

// Up_20260515000600 adds three indexes to speed up the /api/v1/fleet/vulnerabilities
// and /api/v1/fleet/software/versions endpoints, which were doing full-table scans
// for filter and scope predicates. All three are created with ALGORITHM=INPLACE,
// LOCK=NONE so they can be applied online without blocking writers.
//
//   - idx_cve_meta_exploit (cisa_known_exploit, cve):
//     CVE listing filters by cm.cisa_known_exploit = 1; that column has no index
//     today, forcing a full scan of cve_meta. cisa_known_exploit is highly
//     selective (a few thousand out of 200k+ CVEs), so this index turns the
//     filter into an index range scan. The trailing cve makes the index
//     covering for the join back to vulnerability_host_counts.
//
//   - idx_cve_meta_cvss_score (cvss_score, cve):
//     /software/versions filters by c.cvss_score >= ? on cve_meta with no
//     supporting index. Kept separate from idx_cve_meta_exploit because mixing
//     an equality column with a range column in a single composite would
//     prevent independent use of either filter.
//
//   - idx_vhc_scope_cve (global_stats, team_id, host_count, cve):
//     vulnerability_host_counts only has UNIQUE KEY (cve, team_id, global_stats),
//     which leads with cve and is useless for the scope filter shape used by
//     ListVulnerabilities/CountVulnerabilities (global_stats = ?, team_id = ?,
//     host_count > 0). Leading with the scope columns and including cve last
//     makes this index covering for the inner query in the refactored
//     ListVulnerabilities path.
func Up_20260515000600(tx *sql.Tx) error {
	stmts := []struct {
		name string
		sql  string
	}{
		{
			name: "idx_cve_meta_exploit",
			sql:  `ALTER TABLE cve_meta ADD INDEX idx_cve_meta_exploit (cisa_known_exploit, cve), ALGORITHM=INPLACE, LOCK=NONE`,
		},
		{
			name: "idx_cve_meta_cvss_score",
			sql:  `ALTER TABLE cve_meta ADD INDEX idx_cve_meta_cvss_score (cvss_score, cve), ALGORITHM=INPLACE, LOCK=NONE`,
		},
		{
			name: "idx_vhc_scope_cve",
			sql:  `ALTER TABLE vulnerability_host_counts ADD INDEX idx_vhc_scope_cve (global_stats, team_id, host_count, cve), ALGORITHM=INPLACE, LOCK=NONE`,
		},
	}

	for _, s := range stmts {
		if _, err := tx.Exec(s.sql); err != nil {
			return fmt.Errorf("failed to add %s: %w", s.name, err)
		}
	}
	return nil
}

func Down_20260515000600(tx *sql.Tx) error {
	return nil
}
