// Package orbitconfig_notify provides a fleet.Datastore decorator that tells
// connected agents to fetch their orbit config after a successful write that
// changes it for a whole fleet, for every host, or for hosts changing fleet.
//
// Per-host inputs of the orbit config (pending scripts and installs, MDM and
// disk encryption state) are notified by the MySQL datastore itself, after
// the writing transaction commits (see Datastore.WithOrbitConfigNotifier).
// The decorator covers what lives on the app config and fleet records, which
// GetOrbitConfig reads through per-instance caches: those notifications are
// scoped (each instance resolves the hosts it holds) and are delayed by the
// cache TTL by the notifier (see pubsub.DelayedAgentNotifier).
//
// Missing a write here is not a correctness issue: connected agents still
// poll their orbit config as a fallback, only later.
package orbitconfig_notify

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Datastore decorates a fleet.Datastore so that every successful write
// changing an orbit config input shared by many hosts notifies them.
//
// ADDING A DATASTORE METHOD: declare it here if it can change what
// Service.GetOrbitConfig returns for hosts other than those it writes per-host
// rows for, notify after the inner call succeeds, and add a case to the hook
// cases in the tests (TestEveryWrappedMethodHasAHookCase requires one).
type Datastore struct {
	fleet.Datastore
	notifier fleet.AgentCheckInNotifier
	logger   *slog.Logger
}

func New(ds fleet.Datastore, notifier fleet.AgentCheckInNotifier, logger *slog.Logger) *Datastore {
	return &Datastore{
		Datastore: ds,
		notifier:  notifier,
		logger:    logger,
	}
}

func (d *Datastore) publishScope(ctx context.Context, scope fleet.AgentNotificationScope, source string) {
	if err := d.notifier.NotifyOrbitConfigScope(ctx, scope, fleet.AgentWSReasonSettings); err != nil {
		d.logger.ErrorContext(ctx, "notify orbit config change", "scope", scope, "source", source, "err", err)
	}
}

func (d *Datastore) publishHosts(ctx context.Context, hostIDs []uint, source string) {
	if len(hostIDs) == 0 {
		return
	}
	if err := d.notifier.NotifyOrbitConfigHosts(ctx, hostIDs, fleet.AgentWSReasonSettings); err != nil {
		d.logger.ErrorContext(ctx, "notify orbit config change", "source", source, "err", err)
	}
}

// freshCtx reads the state before a write from the primary, uncached: the
// comparison decides whether to notify.
func freshCtx(ctx context.Context) context.Context {
	return ctxdb.BypassCachedMysql(ctxdb.RequirePrimary(ctx, true), true)
}

// changed reports whether before and after differ. An unknown before (nil,
// e.g. it could not be read) counts as changed.
func changed[T any](before, after *T) bool {
	if before == nil {
		return true
	}
	b, errB := json.Marshal(before)
	a, errA := json.Marshal(after)
	return errB != nil || errA != nil || string(a) != string(b)
}

// appConfigGlobalInputs are the app config fields GetOrbitConfig reads for
// every host.
type appConfigGlobalInputs struct {
	MDMEnabled                     bool
	MacOSMigrationEnabled          bool
	WindowsMDMEnabled              bool
	EnableTurnOnWindowsMDMManually bool
	WindowsMigrationEnabled        bool
	ServerURL                      string
	ScriptsDisabled                bool
	// Hosts in a fleet fall back to the global script timeout.
	ScriptExecutionTimeout int
}

// appConfigNoTeamInputs are the app config fields GetOrbitConfig reads only
// for "No fleet" hosts.
type appConfigNoTeamInputs struct {
	AgentOptions   *json.RawMessage
	MacOSUpdates   fleet.AppleOSUpdateSettings
	DiskEncryption fleet.DiskEncryptionConfig
}

func appConfigOrbitInputs(cfg *fleet.AppConfig) (*appConfigGlobalInputs, *appConfigNoTeamInputs) {
	if cfg == nil {
		return nil, nil
	}
	var opts fleet.AgentOptions
	if cfg.AgentOptions != nil {
		// An unparsable value leaves the timeout at 0; AgentOptions still
		// compares raw below.
		_ = json.Unmarshal(*cfg.AgentOptions, &opts)
	}
	return &appConfigGlobalInputs{
			MDMEnabled:                     cfg.MDM.EnabledAndConfigured,
			MacOSMigrationEnabled:          cfg.MDM.MacOSMigration.Enable,
			WindowsMDMEnabled:              cfg.MDM.WindowsEnabledAndConfigured,
			EnableTurnOnWindowsMDMManually: cfg.MDM.EnableTurnOnWindowsMDMManually,
			WindowsMigrationEnabled:        cfg.MDM.WindowsMigrationEnabled,
			ServerURL:                      cfg.ServerSettings.ServerURL,
			ScriptsDisabled:                cfg.ServerSettings.ScriptsDisabled,
			ScriptExecutionTimeout:         opts.ScriptExecutionTimeout,
		}, &appConfigNoTeamInputs{
			AgentOptions:   cfg.AgentOptions,
			MacOSUpdates:   cfg.MDM.MacOSUpdates,
			DiskEncryption: cfg.MDM.DiskEncryptionConfig(),
		}
}

// teamConfigInputs are the fleet config fields GetOrbitConfig reads.
type teamConfigInputs struct {
	AgentOptions *json.RawMessage
	MDM          fleet.TeamMDM
}

func (d *Datastore) SaveAppConfig(ctx context.Context, info *fleet.AppConfig) error {
	// GitOps saves the app config on every run, mostly unchanged: only notify
	// on an actual change.
	// "No fleet" settings live on the app config too: notify every host only
	// for the fields read for every host.
	var beforeGlobal *appConfigGlobalInputs
	var beforeNoTeam *appConfigNoTeamInputs
	if cfg, err := d.Datastore.AppConfig(freshCtx(ctx)); err == nil {
		beforeGlobal, beforeNoTeam = appConfigOrbitInputs(cfg)
	}
	if err := d.Datastore.SaveAppConfig(ctx, info); err != nil {
		return err
	}
	afterGlobal, afterNoTeam := appConfigOrbitInputs(info)
	switch {
	case changed(beforeGlobal, afterGlobal):
		d.publishScope(ctx, fleet.AgentNotificationScopeGlobal, "SaveAppConfig")
	case changed(beforeNoTeam, afterNoTeam):
		d.publishScope(ctx, fleet.AgentNotificationScopeTeam(0), "SaveAppConfig")
	}
	return nil
}

func (d *Datastore) SaveTeam(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
	var before *teamConfigInputs
	if t, err := d.Datastore.TeamLite(freshCtx(ctx), team.ID); err == nil {
		before = &teamConfigInputs{AgentOptions: t.Config.AgentOptions, MDM: t.Config.MDM}
	}
	saved, err := d.Datastore.SaveTeam(ctx, team)
	if err != nil {
		return nil, err
	}
	if changed(before, &teamConfigInputs{AgentOptions: team.Config.AgentOptions, MDM: team.Config.MDM}) {
		d.publishScope(ctx, fleet.AgentNotificationScopeTeam(team.ID), "SaveTeam")
	}
	return saved, nil
}

func (d *Datastore) SaveDefaultTeamConfig(ctx context.Context, config *fleet.TeamConfig) error {
	var before *teamConfigInputs
	if cfg, err := d.Datastore.DefaultTeamConfig(freshCtx(ctx)); err == nil && cfg != nil {
		before = &teamConfigInputs{AgentOptions: cfg.AgentOptions, MDM: cfg.MDM}
	}
	if err := d.Datastore.SaveDefaultTeamConfig(ctx, config); err != nil {
		return err
	}
	if config != nil && changed(before, &teamConfigInputs{AgentOptions: config.AgentOptions, MDM: config.MDM}) {
		d.publishScope(ctx, fleet.AgentNotificationScopeTeam(0), "SaveDefaultTeamConfig")
	}
	return nil
}

func (d *Datastore) DeleteTeam(ctx context.Context, tid uint) error {
	if err := d.Datastore.DeleteTeam(ctx, tid); err != nil {
		return err
	}
	// The fleet's hosts moved to "No fleet"; they are no longer in its scope.
	d.publishScope(ctx, fleet.AgentNotificationScopeTeam(0), "DeleteTeam")
	return nil
}

func (d *Datastore) AddHostsToTeam(ctx context.Context, params *fleet.AddHostsToTeamParams) error {
	err := d.Datastore.AddHostsToTeam(ctx, params)
	// Notify even on error: batches commit independently, so earlier ones may
	// have moved hosts. Hosts that didn't move just fetch an unchanged config.
	d.publishHosts(ctx, params.HostIDs, "AddHostsToTeam")
	return err
}
