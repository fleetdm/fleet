// Package notifications is the root package for the notifications bounded
// context. It contains public types that need to be shared with the ACL layer.
package notifications

import "context"

// ScriptQueueProvider queues the script that delivers a notification to a
// host. Notifications are delivered by queueing a script rather than by orbit
// acting on the config, so they share the unified script queue with
// everything else the host runs; that queue is owned by server/datastore/mysql.
type ScriptQueueProvider interface {
	// QueueScriptForHosts queues contents once and activates it next for each
	// host, returning the execution ID it was queued under per host.
	QueueScriptForHosts(ctx context.Context, hostIDs []uint, contents string) (map[uint]string, error)
}

// DataProviders combines all external dependency interfaces for the
// notifications bounded context. The ACL adapter implements this single
// interface.
type DataProviders interface {
	ScriptQueueProvider
}
