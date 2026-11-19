package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
)

// stubWebhookPolicyIDs gives fleet config reads and writes no policies with the webhook or ticket automation on.
//
// TODO(#58577): remove with policy_ids.
func stubWebhookPolicyIDs(ds *mock.Store) {
	ds.WebhooksAndTicketsPolicyIDsFunc = func(ctx context.Context, teamID *uint) ([]uint, error) { return nil, nil }
	ds.WebhooksAndTicketsPolicyIDsByTeamFunc = func(ctx context.Context, teamIDs []uint) (map[uint][]uint, error) { return nil, nil }
	ds.SetWebhooksAndTicketsPoliciesFunc = func(ctx context.Context, teamID *uint, policyIDs []uint) error { return nil }
	ds.PoliciesWithOtherAutomationsFunc = func(ctx context.Context, teamID *uint, policyIDs []uint) ([]*fleet.Policy, error) {
		return nil, nil
	}
}
