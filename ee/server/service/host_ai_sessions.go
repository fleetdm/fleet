package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
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

func validateAISessionPerPage(invalid *fleet.InvalidArgumentError, perPage uint) {
	if perPage > platform_http.MaxPerPage {
		invalid.Append("per_page", fmt.Sprintf("must be no greater than %d", platform_http.MaxPerPage))
	}
}

func validateAISessionCategory(invalid *fleet.InvalidArgumentError, category fleet.AISessionCategory) {
	switch category {
	case "", fleet.AISessionCategoryCommand, fleet.AISessionCategoryFileAccess:
	default:
		invalid.Append("category", `must be "command" or "file_access"`)
	}
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
	}
	// The order keys aren't unique, so a cursor on the key alone would skip the sessions tied with it.
	if opts.After != "" {
		invalid.Append("after", "is not supported, use page instead")
	}
	validateAISessionPerPage(invalid, opts.PerPage)
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
	validateAISessionPerPage(invalid, opts.PerPage)
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
