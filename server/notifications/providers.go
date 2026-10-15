// Package notifications is the root package for the notifications bounded
// context. It contains public types that need to be shared with the ACL layer.
package notifications

import "context"

// ScriptQueueProvider queues a notification's script. Notifications reach a
// host through the shared script queue rather than the orbit config, and that
// queue lives in server/datastore/mysql.
type ScriptQueueProvider interface {
	// QueueScriptForHosts queues contents once and activates it next for each
	// host, returning the execution ID per host.
	QueueScriptForHosts(ctx context.Context, hostIDs []uint, contents string) (map[uint]string, error)
}

// DataProviders is everything this context needs from outside it, implemented
// by the ACL adapter.
type DataProviders interface {
	ScriptQueueProvider
}
