package worker

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
)

// Name of the DB migration job as registered in the worker. Note that although
// it is a single job, it can process a number of different-but-related tasks,
// identified by the Task field in the job's payload. This is deliberately
// general so other one-off migration tasks that can't be done during the
// "fleet prepare db" command can reuse this job.
const dbMigrationJobName = "db_migration"

type DBMigrationTask string

// List of supported tasks.
const (
	DBMigrateVPPTokenTask DBMigrationTask = "migrate_vpp_token"
)

// DBMigration is the job processor for the db_migration job.
type DBMigration struct {
	Datastore fleet.Datastore
	Log       kitlog.Logger
}

// Name returns the name of the job.
func (m *DBMigration) Name() string {
	return dbMigrationJobName
}

// dbMigrationArgs is the payload for the DB migration job.
type dbMigrationArgs struct {
	Task DBMigrationTask `json:"task"`
}

// Run executes the db_migration job. Note that unlike for other worker jobs,
// there is no QueueDBMigrationJob function - it is expected that this job will
// always be enqueued by a database migration, which use a direct INSERT into
// the jobs table to avoid depending on code that may change over time.
func (m *DBMigration) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args dbMigrationArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	switch args.Task {
	case DBMigrateVPPTokenTask:
		err := m.migrateVPPToken(ctx)
		return ctxerr.Wrap(ctx, err, "running migrate VPP token task")

	default:
		return ctxerr.Errorf(ctx, "unknown task: %v", args.Task)
	}
}

func (m *DBMigration) migrateVPPToken(ctx context.Context) error {
	panic("unimplemented")
}
