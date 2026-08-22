package tables

import (
	"bytes"
	"compress/gzip"
	"database/sql/driver"
	"io"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func deadlock() error {
	return &mysql.MySQLError{Number: mysqlerr.ER_LOCK_DEADLOCK, Message: "Deadlock found when trying to get lock"}
}

// gzipOf matches a bound argument that gunzips to want, so the retried UPDATE is
// checked for the right payload rather than just for having happened.
type gzipOf struct {
	t    *testing.T
	want string
}

func (g gzipOf) Match(v driver.Value) bool {
	b, ok := v.([]byte)
	if !ok {
		return false
	}
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return false
	}
	defer zr.Close()
	out, err := io.ReadAll(zr)
	if err != nil {
		return false
	}
	return string(out) == g.want
}

// TestBackfillWindowsMDMResponsesGzRetriesReprepare drives the real backfill
// through sqlmock to prove the wiring, not just retryOnNeedReprepare in
// isolation: an ER_NEED_REPREPARE on the reported statement ("storing
// compressed response id ...", #51707) must be re-executed rather than abort
// the upgrade, and the row must still end up compressed exactly once.
func TestBackfillWindowsMDMResponsesGzRetriesReprepare(t *testing.T) {
	shrinkReprepareBackoff(t)

	const raw = "<SyncML><SyncBody><Status><Data>200</Data></Status></SyncBody></SyncML>"

	batchRows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"id", "raw_response"}).AddRow(int64(1), []byte(raw))
	}
	noRows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"id", "raw_response"})
	}

	t.Run("the reported update is retried and the walk completes", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, raw_response FROM windows_mdm_responses").WillReturnRows(batchRows())
		mock.ExpectExec("UPDATE windows_mdm_responses SET raw_response_gz").
			WithArgs(gzipOf{t, raw}, int64(1)).
			WillReturnError(needReprepare())
		mock.ExpectExec("UPDATE windows_mdm_responses SET raw_response_gz").
			WithArgs(gzipOf{t, raw}, int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))
		// The walk moves past id 1 and stops on the empty batch.
		mock.ExpectQuery("SELECT id, raw_response FROM windows_mdm_responses").WillReturnRows(noRows())

		tx, err := db.Begin()
		require.NoError(t, err)

		increments := 0
		require.NoError(t, backfillWindowsMDMResponsesGz(tx, func() { increments++ }))

		require.NoError(t, mock.ExpectationsWereMet())
		// The retry must not double-count the row it re-stored.
		require.Equal(t, 1, increments)
	})

	t.Run("the batch select is retried too", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, raw_response FROM windows_mdm_responses").WillReturnError(needReprepare())
		mock.ExpectQuery("SELECT id, raw_response FROM windows_mdm_responses").WillReturnRows(batchRows())
		mock.ExpectExec("UPDATE windows_mdm_responses SET raw_response_gz").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery("SELECT id, raw_response FROM windows_mdm_responses").WillReturnRows(noRows())

		tx, err := db.Begin()
		require.NoError(t, err)

		increments := 0
		require.NoError(t, backfillWindowsMDMResponsesGz(tx, func() { increments++ }))

		require.NoError(t, mock.ExpectationsWereMet())
		// The re-read batch is processed once. Note this does not exercise the
		// batch = batch[:0] reset: a 1615 arrives before any row does, so the
		// first attempt appends nothing. The reset is there for re-runnability,
		// not for this path.
		require.Equal(t, 1, increments)
	})

	// The negative control for the scope: an error the migration cannot retry in
	// place must still abort, with the original message intact.
	t.Run("a non-reprepare error still aborts", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT id, raw_response FROM windows_mdm_responses").WillReturnRows(batchRows())
		mock.ExpectExec("UPDATE windows_mdm_responses SET raw_response_gz").WillReturnError(deadlock())

		tx, err := db.Begin()
		require.NoError(t, err)

		err = backfillWindowsMDMResponsesGz(tx, func() {})
		require.Error(t, err)
		require.Contains(t, err.Error(), "storing compressed response id 1")
		require.Contains(t, err.Error(), "Deadlock found")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
