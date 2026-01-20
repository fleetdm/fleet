package tables

import (
	"bytes"
	"database/sql"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicMigrationStep(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	t.Run("success", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("ALTER TABLE foo ADD COLUMN bar INT").WillReturnResult(sqlmock.NewResult(0, 0))

		tx, err := db.Begin()
		require.NoError(t, err)

		step := basicMigrationStep("ALTER TABLE foo ADD COLUMN bar INT", "failed to add column")
		err = step(tx)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("ALTER TABLE foo ADD COLUMN bar INT").WillReturnError(errors.New("syntax error"))

		tx, err := db.Begin()
		require.NoError(t, err)

		step := basicMigrationStep("ALTER TABLE foo ADD COLUMN bar INT", "failed to add column")
		err = step(tx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to add column")
		assert.Contains(t, err.Error(), "syntax error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIncrementalMigrationStep(t *testing.T) {
	// Save original values and restore after test
	originalOutputTo := outputTo
	originalProgressInterval := progressInterval
	defer func() {
		outputTo = originalOutputTo
		progressInterval = originalProgressInterval
	}()

	t.Run("zero count skips execution", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()

		tx, err := db.Begin()
		require.NoError(t, err)

		executeCalled := false
		step := incrementalMigrationStep(
			func(tx *sql.Tx) (uint, error) {
				return 0, nil
			},
			func(tx *sql.Tx, increment func()) error {
				executeCalled = true
				return nil
			},
		)

		err = step(tx)
		require.NoError(t, err)
		assert.False(t, executeCalled, "executor should not be called when count is 0")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count error is returned", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()

		tx, err := db.Begin()
		require.NoError(t, err)

		var wasExecutorCalled bool
		expectedErr := errors.New("count query failed")
		step := incrementalMigrationStep(
			func(tx *sql.Tx) (uint, error) {
				return 0, expectedErr
			},
			func(tx *sql.Tx, increment func()) error {
				wasExecutorCalled = true
				return nil
			},
		)

		err = step(tx)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		require.False(t, wasExecutorCalled, "executor should not be called when count call errors")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("executor error is returned", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()

		tx, err := db.Begin()
		require.NoError(t, err)

		expectedErr := errors.New("executor failed")
		step := incrementalMigrationStep(
			func(tx *sql.Tx) (uint, error) {
				return 5, nil
			},
			func(tx *sql.Tx, increment func()) error {
				return expectedErr
			},
		)

		err = step(tx)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("progress output", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()

		tx, err := db.Begin()
		require.NoError(t, err)

		// Override output and progress interval for testing
		var buf bytes.Buffer
		outputTo = &buf
		progressInterval = 10 * time.Millisecond

		step := incrementalMigrationStep(
			func(tx *sql.Tx) (uint, error) {
				return 10, nil
			},
			func(tx *sql.Tx, increment func()) error {
				// Simulate work with increments
				for range 10 {
					increment()
					time.Sleep(5 * time.Millisecond)
				}
				return nil
			},
		)

		err = step(tx)
		require.NoError(t, err)

		// Verify progress output was written
		assert.Equal(t, "20% complete\n40% complete\n60% complete\n80% complete\n100% complete\n", buf.String())

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("increment updates progress", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()

		tx, err := db.Begin()
		require.NoError(t, err)

		// Override output and progress interval for testing
		var buf bytes.Buffer
		outputTo = &buf
		progressInterval = 20 * time.Millisecond

		incrementCount := 0
		step := incrementalMigrationStep(
			func(tx *sql.Tx) (uint, error) {
				return 50, nil
			},
			func(tx *sql.Tx, increment func()) error {
				// Call increment multiple times
				for range 50 {
					increment()
					incrementCount++
				}
				// Allow time for progress ticker
				time.Sleep(30 * time.Millisecond)
				return nil
			},
		)

		err = step(tx)
		require.NoError(t, err)
		assert.Equal(t, 50, incrementCount)
		require.Equal(t, "100% complete\n", buf.String())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestWithSteps(t *testing.T) {
	// Save original values and restore after test
	originalOutputTo := outputTo
	defer func() {
		outputTo = originalOutputTo
	}()

	t.Run("empty steps succeeds", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		var buf bytes.Buffer
		outputTo = &buf

		mock.ExpectBegin()

		tx, err := db.Begin()
		require.NoError(t, err)

		err = withSteps([]migrationStep{}, tx)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())

		require.Empty(t, buf.String())
	})

	t.Run("single step no step output", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()

		tx, err := db.Begin()
		require.NoError(t, err)

		var buf bytes.Buffer
		outputTo = &buf

		stepCalled := false
		steps := []migrationStep{
			func(tx *sql.Tx) error {
				stepCalled = true
				return nil
			},
		}

		err = withSteps(steps, tx)
		require.NoError(t, err)
		assert.True(t, stepCalled)

		// Single step should not output step number
		output := buf.String()
		assert.Empty(t, output)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("multiple steps with output", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()

		tx, err := db.Begin()
		require.NoError(t, err)

		var buf bytes.Buffer
		outputTo = &buf

		var callOrder []int
		steps := []migrationStep{
			func(tx *sql.Tx) error {
				callOrder = append(callOrder, 1)
				return nil
			},
			func(tx *sql.Tx) error {
				callOrder = append(callOrder, 2)
				return nil
			},
			func(tx *sql.Tx) error {
				callOrder = append(callOrder, 3)
				return nil
			},
		}

		err = withSteps(steps, tx)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, callOrder)

		// Multiple steps should output step numbers
		assert.Equal(t, "Step 1 of 3\nStep 2 of 3\nStep 3 of 3\n", buf.String())

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error stops execution", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()

		tx, err := db.Begin()
		require.NoError(t, err)

		var buf bytes.Buffer
		outputTo = &buf

		expectedErr := errors.New("step 2 failed")
		var callOrder []int
		steps := []migrationStep{
			func(tx *sql.Tx) error {
				callOrder = append(callOrder, 1)
				return nil
			},
			func(tx *sql.Tx) error {
				callOrder = append(callOrder, 2)
				return expectedErr
			},
			func(tx *sql.Tx) error {
				callOrder = append(callOrder, 3)
				return nil
			},
		}

		err = withSteps(steps, tx)
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, []int{1, 2}, callOrder, "step 3 should not be called after step 2 fails")
		require.Equal(t, "Step 1 of 3\nStep 2 of 3\n", buf.String())

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("integration with basicMigrationStep", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectBegin()
		mock.ExpectExec("ALTER TABLE foo ADD COLUMN a INT").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("ALTER TABLE foo ADD COLUMN b INT").WillReturnResult(sqlmock.NewResult(0, 0))

		tx, err := db.Begin()
		require.NoError(t, err)

		var buf bytes.Buffer
		outputTo = &buf

		steps := []migrationStep{
			basicMigrationStep("ALTER TABLE foo ADD COLUMN a INT", "failed to add column a"),
			basicMigrationStep("ALTER TABLE foo ADD COLUMN b INT", "failed to add column b"),
		}

		err = withSteps(steps, tx)
		require.NoError(t, err)

		require.Equal(t, buf.String(), "Step 1 of 2\nStep 2 of 2\n")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestOutputToAndProgressIntervalDefaults verifies the default values of the package variables
func TestOutputToAndProgressIntervalDefaults(t *testing.T) {
	// Note: These tests verify the defaults are sensible
	// The actual io.Writer interface check is sufficient
	var _ io.Writer = outputTo
	assert.Equal(t, 5*time.Second, progressInterval)
}
