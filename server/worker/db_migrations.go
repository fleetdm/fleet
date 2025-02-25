package worker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

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
	DBMigrateVPPTokenTask DBMigrationTask = "migrate_vpp_token" //nolint: gosec
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
	// get the VPP token with an empty location, this is the one to migrate
	tok, err := m.Datastore.GetVPPTokenByLocation(ctx, "")
	if err != nil {
		if fleet.IsNotFound(err) {
			// nothing to migrate, exit successfully
			return nil
		}
		return ctxerr.Wrap(ctx, err, "get VPP token to migrate")
	}

	tokenData, didUpdate, err := extractVPPTokenFromMigration(tok)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "extract VPP token metadata")
	}
	if !didUpdate {
		// it should've updated, as the location, org name and renew date were all
		// dummy values after the DB migration. Log something, but otherwise
		// continue as retrying won't change the result.
		m.Log.Log("info", "VPP token metadata was not updated")
	}

	if _, err := m.Datastore.UpdateVPPToken(ctx, tok.ID, tokenData); err != nil {
		return ctxerr.Wrap(ctx, err, "update VPP token")
	}
	// the migated token should target "All teams"
	_, err = m.Datastore.UpdateVPPTokenTeams(ctx, tok.ID, []uint{})
	return ctxerr.Wrap(ctx, err, "update VPP token teams")
}

func extractVPPTokenFromMigration(migratedToken *fleet.VPPTokenDB) (tokData *fleet.VPPTokenData, didUpdateMetadata bool, err error) {
	var vppTokenData fleet.VPPTokenData
	if err := json.Unmarshal([]byte(migratedToken.Token), &vppTokenData); err != nil {
		return nil, false, fmt.Errorf("unmarshaling VPP token data: %w", err)
	}

	vppTokenRawBytes, err := base64.StdEncoding.DecodeString(vppTokenData.Token)
	if err != nil {
		return nil, false, fmt.Errorf("decoding raw vpp token data: %w", err)
	}

	var vppTokenRaw fleet.VPPTokenRaw
	if err := json.Unmarshal(vppTokenRawBytes, &vppTokenRaw); err != nil {
		return nil, false, fmt.Errorf("unmarshaling raw vpp token data: %w", err)
	}

	exp, err := time.Parse("2006-01-02T15:04:05Z0700", vppTokenRaw.ExpDate)
	if err != nil {
		return nil, false, fmt.Errorf("parsing vpp token expiration date: %w", err)
	}

	if vppTokenData.Location != migratedToken.Location {
		migratedToken.Location = vppTokenData.Location
		didUpdateMetadata = true
	}
	if vppTokenRaw.OrgName != migratedToken.OrgName {
		migratedToken.OrgName = vppTokenRaw.OrgName
		didUpdateMetadata = true
	}
	if !exp.Equal(migratedToken.RenewDate) {
		migratedToken.RenewDate = exp.UTC()
		didUpdateMetadata = true
	}

	return &vppTokenData, didUpdateMetadata, nil
}
