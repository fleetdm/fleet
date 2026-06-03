package cron

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/fleetdm/fleet/v4/ee/server/idp/googleworkspace"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
)

// NewGoogleWorkspaceSchedule registers the cron that syncs IdP host vitals
// (users and groups) from Google Workspace into the SCIM tables.
func NewGoogleWorkspaceSchedule(
	ctx context.Context,
	instanceID string,
	ds fleet.Datastore,
	serverConfig config.GoogleWorkspaceConfig,
	logger *slog.Logger,
) (*schedule.Schedule, error) {
	const name = string(fleet.CronGoogleWorkspace)
	logger = logger.With("cron", name)
	s := schedule.New(
		ctx, name, instanceID, serverConfig.Periodicity, ds, ds,
		schedule.WithAltLockID("google_workspace"),
		schedule.WithLogger(logger),
		schedule.WithJob(
			"google_workspace_idp_sync",
			func(ctx context.Context) error {
				return cronGoogleWorkspaceSync(ctx, ds, logger)
			},
		),
	)
	return s, nil
}

func cronGoogleWorkspaceSync(ctx context.Context, ds fleet.Datastore, logger *slog.Logger) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return fmt.Errorf("load app config: %w", err)
	}

	if len(appConfig.Integrations.GoogleWorkspace) == 0 {
		return nil
	}
	intg := appConfig.Integrations.GoogleWorkspace[0]

	syncErr := googleworkspace.Sync(ctx, ds, intg, logger)

	// Record the outcome in the SCIM last-request status, which is what surfaces
	// IdP sync health in the UI (Google Workspace writes into the SCIM tables).
	lastRequest := &fleet.ScimLastRequest{Status: "success"}
	if syncErr != nil {
		lastRequest.Status = "error"
		details := syncErr.Error()
		if len(details) > fleet.SCIMMaxFieldLength {
			details = details[:fleet.SCIMMaxFieldLength]
		}
		lastRequest.Details = details
	}
	if err := ds.UpdateScimLastRequest(ctx, lastRequest); err != nil {
		logger.WarnContext(ctx, "failed to update google workspace sync last request", "err", err)
	}

	return syncErr
}
