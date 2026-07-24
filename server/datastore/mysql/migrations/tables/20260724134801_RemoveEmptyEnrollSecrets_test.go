package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260724134801(t *testing.T) {
	db := applyUpToPrev(t)

	// Start from a clean enroll_secrets table so the assertions below are exact.
	_, err := db.Exec(`DELETE FROM enroll_secrets`)
	require.NoError(t, err)

	// Seed whitespace-only secrets (must be removed) alongside secrets that
	// contain real content (must be kept). CHAR(... USING utf8mb4) keeps the
	// exact bytes explicit. The secret column is a PADSPACE primary key, so the
	// empty string and a spaces-only secret share the same key; '   ' stands in
	// for both. The tab/newline/NBSP cases are the ones MySQL's TRIM would miss.
	_, err = db.Exec(`
		INSERT INTO enroll_secrets (secret, team_id) VALUES
			(CHAR(32, 32, 32 USING utf8mb4), NULL),                                              -- spaces only
			(CHAR(9 USING utf8mb4), NULL),                                                       -- tab only
			(CHAR(10 USING utf8mb4), NULL),                                                      -- newline only
			(CONCAT(CHAR(13 USING utf8mb4), CHAR(10 USING utf8mb4)), NULL),                      -- CRLF only
			(CONCAT(CHAR(9 USING utf8mb4), CHAR(32 USING utf8mb4), CHAR(10 USING utf8mb4)), NULL), -- mixed tab/space/newline
			(_utf8mb4 0xC2A0, NULL),                                                             -- non-breaking space (Unicode)
			('validSecret', NULL),                                                               -- kept
			('has spaces inside', NULL),                                                         -- kept (inner spaces)
			(CONCAT('a', CHAR(9 USING utf8mb4), 'b'), NULL)                                      -- kept (tab between content)
	`)
	require.NoError(t, err)

	applyNext(t, db)

	var remaining []string
	require.NoError(t, db.Select(&remaining, `SELECT secret FROM enroll_secrets`))
	require.ElementsMatch(t, []string{
		"validSecret",
		"has spaces inside",
		"a\tb",
	}, remaining)
}
