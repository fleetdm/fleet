// Package osqueryconfig_notify provides a fleet.Datastore decorator that tells
// connected agents to fetch their osquery config after a successful write that
// changes it.
//
// The writes that can change the osquery config are the ones hooked by
// server/datastore/etag_invalidate, plus hosts changing fleet (the ETag scope
// is resolved per request, so it needs no invalidation, but the host's config
// changes). Label membership changes are left out: they only change the
// config of hosts with label-scoped reports, and run continuously at fleet
// scale; those changes reach agents on their fallback poll.
//
// Notifications are scoped (each instance resolves the hosts it holds) and
// are delayed by the TTL of the in-memory caches the osquery config is built
// from (see pubsub.DelayedAgentNotifier).
//
// Missing a write here is not a correctness issue: connected agents still
// poll their osquery config as a fallback, only later.
package osqueryconfig_notify

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"reflect"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Datastore decorates a fleet.Datastore so that every successful write
// changing an osquery config input notifies the hosts whose config changed.
//
// ADDING A DATASTORE METHOD: declare it here if it can change what
// Service.GetClientConfig returns, notify after the inner call succeeds, and
// add a case to the hook cases in the tests (TestEveryWrappedMethodHasAHookCase
// requires one). TestCoversETagInvalidation requires every method hooked by
// etag_invalidate to be declared here too, or explicitly skipped.
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

func (d *Datastore) publishScope(ctx context.Context, scope fleet.AgentNotificationScope, reason, source string) {
	if err := d.notifier.NotifyOsqueryConfigScope(ctx, scope, reason); err != nil {
		d.logger.ErrorContext(ctx, "notify osquery config change", "scope", scope, "source", source, "err", err)
	}
}

func (d *Datastore) publishHosts(ctx context.Context, hostIDs []uint, reason, source string) {
	if len(hostIDs) == 0 {
		return
	}
	if err := d.notifier.NotifyOsqueryConfigHosts(ctx, hostIDs, reason); err != nil {
		d.logger.ErrorContext(ctx, "notify osquery config change", "source", source, "err", err)
	}
}

// freshCtx reads the state before a write from the primary, uncached: the
// comparison decides whether to notify.
func freshCtx(ctx context.Context) context.Context {
	return ctxdb.BypassCachedMysql(ctxdb.RequirePrimary(ctx, true), true)
}

// rawChanged reports whether two JSON values differ, ignoring formatting and
// object key order.
func rawChanged(before, after *json.RawMessage) bool {
	b, errB := decodeRaw(before)
	a, errA := decodeRaw(after)
	return errB != nil || errA != nil || !reflect.DeepEqual(a, b)
}

func decodeRaw(raw *json.RawMessage) (any, error) {
	if raw == nil {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(*raw))
	// Keeps large numbers exact.
	dec.UseNumber()
	var v any
	err := dec.Decode(&v)
	return v, err
}

// reportsScope returns the scope of the hosts running the reports of the
// given fleet, or of every host for global reports.
func reportsScope(teamID *uint) fleet.AgentNotificationScope {
	if teamID == nil {
		return fleet.AgentNotificationScopeGlobal
	}
	return fleet.AgentNotificationScopeTeam(*teamID)
}

func teamKey(teamID *uint) uint {
	if teamID == nil {
		// Fleet IDs start at 1.
		return 0
	}
	return *teamID
}

// reportsDigests returns the scheduled reports digest of each given fleet
// (nil for global reports). A fleet whose digest can't be read is left out,
// which reportsChanged treats as changed.
func (d *Datastore) reportsDigests(ctx context.Context, teamIDs []*uint) map[uint]string {
	digests := make(map[uint]string, len(teamIDs))
	for _, teamID := range teamIDs {
		digest, err := d.Datastore.ScheduledQueriesDigest(ctx, teamID)
		if err != nil {
			d.logger.ErrorContext(ctx, "read scheduled reports digest", "err", err)
			continue
		}
		digests[teamKey(teamID)] = digest
	}
	return digests
}

// withReportsDiff runs write and notifies the hosts of each given fleet whose
// scheduled reports it changed. GitOps applies every report on every run,
// mostly unchanged: comparing avoids notifying every host on every run.
//
// It compares even when write fails: ApplyQueries commits in batches, so a
// failure may follow committed changes.
func (d *Datastore) withReportsDiff(ctx context.Context, teamIDs []*uint, source string, write func() error) error {
	before := d.reportsDigests(freshCtx(ctx), teamIDs)
	err := write()
	after := d.reportsDigests(freshCtx(ctx), teamIDs)
	for _, teamID := range teamIDs {
		b, okB := before[teamKey(teamID)]
		a, okA := after[teamKey(teamID)]
		if !okB || !okA || a != b {
			d.publishScope(ctx, reportsScope(teamID), fleet.AgentWSReasonReports, source)
		}
	}
	return err
}

// ///////////////////////////////////////////////////////////////////////////
// Agent options. Hosts in a fleet with empty agent options get the global
// ones, so a global change notifies every host.
// ///////////////////////////////////////////////////////////////////////////

func (d *Datastore) SaveAppConfig(ctx context.Context, info *fleet.AppConfig) error {
	before, beforeErr := d.Datastore.AppConfig(freshCtx(ctx))
	if err := d.Datastore.SaveAppConfig(ctx, info); err != nil {
		return err
	}
	if beforeErr != nil || before == nil ||
		rawChanged(before.AgentOptions, info.AgentOptions) ||
		// Changes which reports are sent to agents.
		before.ServerSettings.QueryReportsDisabled != info.ServerSettings.QueryReportsDisabled {
		d.publishScope(ctx, fleet.AgentNotificationScopeGlobal, fleet.AgentWSReasonSettings, "SaveAppConfig")
	}
	return nil
}

func (d *Datastore) SaveTeam(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
	before, beforeErr := d.Datastore.TeamLite(freshCtx(ctx), team.ID)
	saved, err := d.Datastore.SaveTeam(ctx, team)
	if err != nil {
		return nil, err
	}
	if beforeErr != nil || before == nil || rawChanged(before.Config.AgentOptions, team.Config.AgentOptions) {
		d.publishScope(ctx, fleet.AgentNotificationScopeTeam(team.ID), fleet.AgentWSReasonSettings, "SaveTeam")
	}
	return saved, nil
}

// ///////////////////////////////////////////////////////////////////////////
// Hosts changing fleet get the new fleet's agent options and reports.
// ///////////////////////////////////////////////////////////////////////////

func (d *Datastore) DeleteTeam(ctx context.Context, tid uint) error {
	if err := d.Datastore.DeleteTeam(ctx, tid); err != nil {
		return err
	}
	// The fleet's hosts moved to "No fleet"; they are no longer in its scope.
	d.publishScope(ctx, fleet.AgentNotificationScopeTeam(0), fleet.AgentWSReasonSettings, "DeleteTeam")
	return nil
}

func (d *Datastore) AddHostsToTeam(ctx context.Context, params *fleet.AddHostsToTeamParams) error {
	err := d.Datastore.AddHostsToTeam(ctx, params)
	// Notify even on error: batches commit independently, so earlier ones may
	// have moved hosts. Hosts that didn't move just fetch an unchanged config.
	d.publishHosts(ctx, params.HostIDs, fleet.AgentWSReasonSettings, "AddHostsToTeam")
	return err
}

// ///////////////////////////////////////////////////////////////////////////
// Reports.
// ///////////////////////////////////////////////////////////////////////////

func (d *Datastore) ApplyQueries(ctx context.Context, authorID uint, queries []*fleet.Query, queriesToDiscardResults map[uint]struct{}) error {
	seen := make(map[uint]struct{})
	var teamIDs []*uint
	for _, q := range queries {
		if _, ok := seen[teamKey(q.TeamID)]; ok {
			continue
		}
		seen[teamKey(q.TeamID)] = struct{}{}
		teamIDs = append(teamIDs, q.TeamID)
	}
	return d.withReportsDiff(ctx, teamIDs, "ApplyQueries", func() error {
		return d.Datastore.ApplyQueries(ctx, authorID, queries, queriesToDiscardResults)
	})
}

// scheduled reports whether q is in its fleet's scheduled reports digest; a
// write leaving a report unscheduled skips the digests.
func scheduled(q *fleet.Query) bool {
	return q.Saved && q.Interval > 0
}

func (d *Datastore) NewQuery(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
	if !scheduled(query) {
		return d.Datastore.NewQuery(ctx, query, opts...)
	}
	var created *fleet.Query
	err := d.withReportsDiff(ctx, []*uint{query.TeamID}, "NewQuery", func() error {
		var err error
		created, err = d.Datastore.NewQuery(ctx, query, opts...)
		return err
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (d *Datastore) SaveQuery(ctx context.Context, query *fleet.Query, shouldDiscardResults bool, shouldDeleteStats bool) error {
	if !scheduled(query) {
		if stored, err := d.Datastore.Query(freshCtx(ctx), query.ID); err == nil && !scheduled(stored) {
			return d.Datastore.SaveQuery(ctx, query, shouldDiscardResults, shouldDeleteStats)
		}
	}
	return d.withReportsDiff(ctx, []*uint{query.TeamID}, "SaveQuery", func() error {
		return d.Datastore.SaveQuery(ctx, query, shouldDiscardResults, shouldDeleteStats)
	})
}

func (d *Datastore) DeleteQuery(ctx context.Context, teamID *uint, name string) error {
	return d.withReportsDiff(ctx, []*uint{teamID}, "DeleteQuery", func() error {
		return d.Datastore.DeleteQuery(ctx, teamID, name)
	})
}

// DeleteQueries notifies every host: the reports' fleets are unknown, and bulk
// deletes are infrequent.
func (d *Datastore) DeleteQueries(ctx context.Context, ids []uint) (uint, error) {
	n, err := d.Datastore.DeleteQueries(ctx, ids)
	if err != nil {
		return n, err
	}
	if n > 0 {
		d.publishScope(ctx, fleet.AgentNotificationScopeGlobal, fleet.AgentWSReasonReports, "DeleteQueries")
	}
	return n, nil
}

// ///////////////////////////////////////////////////////////////////////////
// 2017 "legacy" packs: rare, so every write notifies every host.
// ///////////////////////////////////////////////////////////////////////////

func (d *Datastore) ApplyPackSpecs(ctx context.Context, specs []*fleet.PackSpec) error {
	if err := d.Datastore.ApplyPackSpecs(ctx, specs); err != nil {
		return err
	}
	d.publishScope(ctx, fleet.AgentNotificationScopeGlobal, fleet.AgentWSReasonReports, "ApplyPackSpecs")
	return nil
}

func (d *Datastore) NewPack(ctx context.Context, pack *fleet.Pack, opts ...fleet.OptionalArg) (*fleet.Pack, error) {
	p, err := d.Datastore.NewPack(ctx, pack, opts...)
	if err != nil {
		return nil, err
	}
	d.publishScope(ctx, fleet.AgentNotificationScopeGlobal, fleet.AgentWSReasonReports, "NewPack")
	return p, nil
}

func (d *Datastore) SavePack(ctx context.Context, pack *fleet.Pack) error {
	if err := d.Datastore.SavePack(ctx, pack); err != nil {
		return err
	}
	d.publishScope(ctx, fleet.AgentNotificationScopeGlobal, fleet.AgentWSReasonReports, "SavePack")
	return nil
}

func (d *Datastore) DeletePack(ctx context.Context, name string) error {
	if err := d.Datastore.DeletePack(ctx, name); err != nil {
		return err
	}
	d.publishScope(ctx, fleet.AgentNotificationScopeGlobal, fleet.AgentWSReasonReports, "DeletePack")
	return nil
}

func (d *Datastore) NewScheduledQuery(ctx context.Context, sq *fleet.ScheduledQuery, opts ...fleet.OptionalArg) (*fleet.ScheduledQuery, error) {
	s, err := d.Datastore.NewScheduledQuery(ctx, sq, opts...)
	if err != nil {
		return nil, err
	}
	d.publishScope(ctx, fleet.AgentNotificationScopeGlobal, fleet.AgentWSReasonReports, "NewScheduledQuery")
	return s, nil
}

func (d *Datastore) SaveScheduledQuery(ctx context.Context, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
	s, err := d.Datastore.SaveScheduledQuery(ctx, sq)
	if err != nil {
		return nil, err
	}
	d.publishScope(ctx, fleet.AgentNotificationScopeGlobal, fleet.AgentWSReasonReports, "SaveScheduledQuery")
	return s, nil
}

func (d *Datastore) DeleteScheduledQuery(ctx context.Context, id uint) error {
	if err := d.Datastore.DeleteScheduledQuery(ctx, id); err != nil {
		return err
	}
	d.publishScope(ctx, fleet.AgentNotificationScopeGlobal, fleet.AgentWSReasonReports, "DeleteScheduledQuery")
	return nil
}

// ///////////////////////////////////////////////////////////////////////////
// Labels.
// ///////////////////////////////////////////////////////////////////////////

// DeleteLabel cascades away the label's query_labels rows, which can unscope
// a report (it then applies to every host of its fleet). Label deletion is
// infrequent, so it notifies every host.
func (d *Datastore) DeleteLabel(ctx context.Context, name string, filter fleet.TeamFilter) error {
	if err := d.Datastore.DeleteLabel(ctx, name, filter); err != nil {
		return err
	}
	d.publishScope(ctx, fleet.AgentNotificationScopeGlobal, fleet.AgentWSReasonLabels, "DeleteLabel")
	return nil
}
