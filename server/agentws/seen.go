package agentws

import (
	"context"
	"log/slog"
	"time"
)

// SeenRecorder records that a host was seen (for online status). It is
// satisfied by the async task layer (*async.Task), which batches the updates
// exactly like the distributed/read auth path does.
type SeenRecorder interface {
	RecordHostLastSeen(ctx context.Context, hostID uint) error
}

// RecordSeenLoop marks every host holding a WebSocket connection on this
// instance as seen, every interval, until ctx is done.
//
// Fleet's online status is derived from seen_time freshness against the
// host's polling cadence (min(distributed_interval, config_tls_refresh) plus
// a buffer — ~70s with default agent options). Before the WebSocket transport
// (ADR-0011), osquery's 10s distributed/read poll kept seen_time fresh; with
// the transport active that poll never reaches the server, so without this
// loop hosts flap offline between osquery's 60s config refreshes. An
// authenticated open WebSocket is the strongest liveness signal the server
// has: dead connections are torn down by the keepalive read deadline, at
// which point their hosts age out of the online window naturally.
func RecordSeenLoop(ctx context.Context, hub *Hub, recorder SeenRecorder, interval time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for _, hostID := range hub.HeldHostIDs() {
				if err := recorder.RecordHostLastSeen(ctx, hostID); err != nil {
					logger.ErrorContext(ctx, "record last seen for websocket-connected host",
						"host_id", hostID, "err", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
