package service

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

var hostAISessionOrderKeys = []string{"started_at", "ended_at", "agent", "username", "model", "output_token_count"}

// hostAISessionsDefaultPerPage bounds a page when the caller doesn't, since a busy host has thousands of sessions and a
// busy session thousands of tool calls, each with a payload of up to 8 KB.
const hostAISessionsDefaultPerPage = 50

// authorizeHostAISessions checks that the user can read the host's AI sessions: only admins and maintainers, since
// sessions include commands and arguments that can contain secrets.
func (svc *Service) authorizeHostAISessions(ctx context.Context, hostID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}
	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get host lite")
	}
	return svc.authz.Authorize(ctx, &fleet.HostAISessionAuthz{TeamID: host.TeamID}, fleet.ActionRead)
}

func validateAISessionCategory(invalid *fleet.InvalidArgumentError, category fleet.AISessionCategory) {
	switch category {
	case "", fleet.AISessionCategoryCommand, fleet.AISessionCategoryFileAccess:
	default:
		invalid.Append("category", `must be "command" or "file_access"`)
	}
}

// validateAISessionAfter checks the cursor against the order key's column, since MySQL fails a query that compares a
// datetime column with a string it can't read. Timestamps are accepted as the API reports them or as MySQL reads them,
// and returned in the form MySQL reads, in UTC.
func validateAISessionAfter(invalid *fleet.InvalidArgumentError, orderKey, after string) string {
	switch orderKey {
	case "started_at", "ended_at":
		for _, layout := range []string{time.RFC3339Nano, time.DateTime, time.DateOnly} {
			if t, err := time.Parse(layout, after); err == nil {
				return t.UTC().Format("2006-01-02 15:04:05.999999")
			}
		}
		invalid.Append("after", "must be a timestamp, such as 2026-10-28T10:00:00Z, when ordering by "+orderKey)
	case "output_token_count":
		if _, err := strconv.ParseUint(after, 10, 64); err != nil {
			invalid.Append("after", "must be a non-negative integer when ordering by "+orderKey)
		}
	}
	return after
}

func (svc *Service) ListHostAISessions(ctx context.Context, hostID uint, opts fleet.HostAISessionListOptions) ([]*fleet.HostAISession, *fleet.PaginationMetadata, error) {
	if err := svc.authorizeHostAISessions(ctx, hostID); err != nil {
		return nil, nil, err
	}

	invalid := &fleet.InvalidArgumentError{}
	if opts.Agent != "" && !slices.Contains(fleet.AISessionAgents, opts.Agent) {
		invalid.Append("agent", "must be one of: "+strings.Join(fleet.AISessionAgents, ", "))
	}
	validateAISessionCategory(invalid, opts.Category)
	if opts.OrderKey == "" {
		opts.OrderKey = "ended_at"
	}
	if !slices.Contains(hostAISessionOrderKeys, opts.OrderKey) {
		invalid.Append("order_key", "must be one of: "+strings.Join(hostAISessionOrderKeys, ", "))
	} else if opts.After != "" {
		opts.After = validateAISessionAfter(invalid, opts.OrderKey, opts.After)
	}
	if invalid.HasErrors() {
		return nil, nil, ctxerr.Wrap(ctx, invalid)
	}

	opts.IncludeMetadata = true
	if opts.PerPage == 0 {
		opts.PerPage = hostAISessionsDefaultPerPage
	}
	sessions, meta, err := svc.ds.ListHostAISessions(ctx, hostID, opts)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "list host ai sessions")
	}
	return sessions, meta, nil
}

func (svc *Service) GetHostAISession(ctx context.Context, hostID, aiSessionID uint) (*fleet.HostAISession, error) {
	if err := svc.authorizeHostAISessions(ctx, hostID); err != nil {
		return nil, err
	}
	session, err := svc.ds.GetHostAISession(ctx, hostID, aiSessionID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host ai session")
	}
	return session, nil
}

func (svc *Service) ListHostAISessionEvents(ctx context.Context, hostID, aiSessionID uint, opts fleet.HostAISessionEventListOptions) ([]*fleet.HostAISessionEvent, *fleet.PaginationMetadata, error) {
	if err := svc.authorizeHostAISessions(ctx, hostID); err != nil {
		return nil, nil, err
	}

	invalid := &fleet.InvalidArgumentError{}
	validateAISessionCategory(invalid, opts.Category)
	if invalid.HasErrors() {
		return nil, nil, ctxerr.Wrap(ctx, invalid)
	}

	if _, err := svc.ds.GetHostAISession(ctx, hostID, aiSessionID); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get host ai session")
	}
	opts.IncludeMetadata = true
	if opts.PerPage == 0 {
		opts.PerPage = hostAISessionsDefaultPerPage
	}
	events, meta, err := svc.ds.ListHostAISessionEvents(ctx, aiSessionID, opts)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "list host ai session events")
	}
	return events, meta, nil
}
