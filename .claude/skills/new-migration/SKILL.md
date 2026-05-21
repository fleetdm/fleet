---
name: new-migration
description: Create a new Fleet database migration with timestamp naming, Up function, init registration, and test file.
allowed-tools: Bash(date *), Bash(make migration *), Bash(go build *), Bash(go test *), Bash(MYSQL_TEST*), Read, Write, Grep, Glob
model: sonnet
effort: medium
---

# Create a New Database Migration

Create a migration for: $ARGUMENTS

## Process

### 1. Generate Timestamp and Name
Use `make migration name=CamelCaseName` if available, or generate manually:
```bash
date +%Y%m%d%H%M%S
```
The migration name should be descriptive CamelCase (e.g., `AddRecoveryLockAutoRotateAt`, `CreateTableSoftwareInstallers`).

### 2. Create Migration File
Location: `server/datastore/mysql/migrations/tables/{TIMESTAMP}_{Name}.go`

```go
package tables

import "database/sql"

func init() {
	MigrationClient.AddMigration(Up_{TIMESTAMP}, Down_{TIMESTAMP})
}

func Up_{TIMESTAMP}(tx *sql.Tx) error {
	_, err := tx.Exec(`
		-- SQL statement here
	`)
	return err
}

func Down_{TIMESTAMP}(tx *sql.Tx) error {
	return nil
}
```

### 3. Create Test File
Location: `server/datastore/mysql/migrations/tables/{TIMESTAMP}_{Name}_test.go`

```go
package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_{TIMESTAMP}(t *testing.T) {
	db := applyUpToPrev(t)

	// Set up test data before migration if needed

	applyNext(t, db)

	// Verify migration applied correctly
	// e.g., check table exists, columns added, data migrated
}
```

### 4. Verify
- Run `go build ./server/datastore/mysql/migrations/...` to check compilation
- Run `MYSQL_TEST=1 go test -run TestUp_{TIMESTAMP} ./server/datastore/mysql/migrations/tables/` to test the migration

## Rules
- Every migration MUST have a test file
- Down migrations are always no-ops (`return nil`) — Fleet doesn't use rollback migrations
- Never modify existing migration files — create new ones
- Data migrations go in the `data/` subdirectory
