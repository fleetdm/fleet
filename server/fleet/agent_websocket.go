package fleet

import (
	"context"
	"encoding/json"
	"fmt"
)

// AgentWSMessage is the envelope of messages sent from the server to a fleetd
// agent over the agent WebSocket notification channel (ADR-0011). The channel
// carries only tiny "check now" notifications: no query content, config
// payloads, or results ever travel over it. Agents ignore message types they
// don't recognize, so new types can be added without breaking older agents.
type AgentWSMessage struct {
	Type string `json:"type"`
	// Reason says what triggered the notification (see the AgentWSReason
	// constants and AgentWSReasonLiveQuery). It is informational, for
	// debugging/troubleshooting only: agents must not change behavior based on
	// it, and the server picks one reason even when several kinds of work are
	// due at once.
	Reason string `json:"reason,omitempty"`
	// Data is reserved for future message types that need a payload; it is
	// empty for the notification types that exist today.
	Data json.RawMessage `json:"data,omitempty"`
}

// AgentWSMessageTypeDistributedRead tells the agent there is work waiting on
// the distributed/read endpoint (a live query, or label/policy/detail queries
// due per their update intervals).
const AgentWSMessageTypeDistributedRead = "distributed/read"

// Reasons for a distributed/read notification triggered by interval work,
// mirroring the gates of the distributed/read endpoint itself.
const (
	AgentWSReasonLabel   = "label"   // label queries due per LabelUpdateInterval
	AgentWSReasonPolicy  = "policy"  // policy queries due per PolicyUpdateInterval
	AgentWSReasonDetail  = "detail"  // detail (host vitals) queries due per DetailUpdateInterval
	AgentWSReasonRefetch = "refetch" // a host refetch was requested (or critical queries are being refetched)

	// AgentWSReasonHostNotFound is returned by ListHostIDsDueForDistributedRead
	// for IDs with no hosts row (the host was deleted while its agent held a
	// connection). It is never sent to agents: the interval check job closes
	// such connections so the agent reconnects with its new identity.
	AgentWSReasonHostNotFound = "host-not-found"
)

// AgentWSReasonLiveQuery is the reason for a distributed/read notification
// triggered by a live query campaign targeting the host.
func AgentWSReasonLiveQuery(campaignID uint) string {
	return AgentWSReasonLiveQueryName(fmt.Sprint(campaignID))
}

// AgentWSReasonLiveQueryName is the AgentWSReasonLiveQuery variant for callers
// holding the campaign's live query store name, which is its ID in decimal
// (see the LiveQueryStore.RunQuery callers).
func AgentWSReasonLiveQueryName(name string) string {
	return "live-" + name
}

// AgentCheckInNotifier publishes cross-instance wake-ups for connected agents.
// The server instance that creates a live query campaign is generally not the
// instance holding a targeted agent's WebSocket connection, so the wake-up is
// published to all instances (via Redis pub/sub); each instance then notifies
// the targeted agents whose connections it holds.
type AgentCheckInNotifier interface {
	NotifyAgentsForLiveQuery(ctx context.Context, hostIDs []uint, campaignID uint) error
}
