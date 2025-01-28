// Package sync provides services to sync devices and assign profile UUIDs
// using the Apple DEP APIs.
package sync

import (
	"context"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/log"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/log/ctxlog"
)

// CursorStorage is where the device fetch and sync cursor can be stored and
// retrieved for a given DEP name.
type CursorStorage interface {
	RetrieveCursor(ctx context.Context, name string) (cursor string, modTime time.Time, err error)
	StoreCursor(ctx context.Context, name string, cursor string) error
}

// DeviceResponseCallback is called every time a fetch or sync operation completes.
type DeviceResponseCallback func(context.Context, bool, *godep.DeviceResponse) error

// Syncer performs the fetch and sync cursor operations to sync devices from
// the Apple DEP service. Depending on the options supplied it can perform the
// sync continuously on a duration or just once. See the various SyncerOptions
// for configuring the behavior of the syncer.
type Syncer struct {
	client   *godep.Client
	name     string
	store    CursorStorage
	logger   log.Logger
	duration time.Duration
	limitOpt godep.DeviceRequestOption
	callback DeviceResponseCallback
	// in "continuous" mode this is a channel that is selected on to interrupt
	// the duration wait to immediately perform the next sync operation(s).
	syncNow <-chan struct{}
}

type SyncerOption func(*Syncer)

// WithLogger configures logger for the syncer.
func WithLogger(logger log.Logger) SyncerOption {
	return func(syncer *Syncer) {
		syncer.logger = logger
	}
}

// WithDuration sets the "mode" of operation. If not set or set to 0 then
// the mode is "run once" and if a duration is provided then the syncer
// operates in "continuous" mode and will start a ticker for duration to wait
// beween sync cycles.
func WithDuration(duration time.Duration) SyncerOption {
	return func(syncer *Syncer) {
		syncer.duration = duration
	}
}

// WithSyncNow specifies the channel to select on which will advise the syncer
// to end its sync wait and perform the next immediate sync.
func WithSyncNow(syncNow <-chan struct{}) SyncerOption {
	return func(syncer *Syncer) {
		syncer.syncNow = syncNow
	}
}

// WithLimit sets the device sync limit for each fetch and sync.
func WithLimit(limit int) SyncerOption {
	return func(syncer *Syncer) {
		syncer.limitOpt = godep.WithLimit(limit)
	}
}

// WithCallback sets the callback function to call for each fetch and sync.
func WithCallback(cb DeviceResponseCallback) SyncerOption {
	return func(s *Syncer) {
		s.callback = cb
	}
}

// NewSyncer creates a new Syncer using client and uses store for cursor
// storage. DEP name is specified with name.
func NewSyncer(client *godep.Client, name string, store CursorStorage, opts ...SyncerOption) *Syncer {
	syncer := &Syncer{
		client: client,
		name:   name,
		store:  store,
		logger: log.NopLogger,
	}
	for _, opt := range opts {
		opt(syncer)
	}
	syncer.logger = syncer.logger.With("name", syncer.name)
	return syncer
}

// Run starts a device fetch and sync loop. Errors from the DEP API are
// generally ignored so that the sync can continue on (i.e. we assume API
// errors are transient). However if a cursor storage error or other "hard"
// error occurs then the loop will end. The loop will end if the context gets
// cancelled. The loop will also exit early if there is no duration option set
// (i.e. is in "run once" mode).
func (s *Syncer) Run(ctx context.Context) error {
	doFetch := true
	// phaseLabel is for logging based on the value of doFetch
	phaseLabel := map[bool]string{
		true:  "fetch",
		false: "sync",
	}
	var resp *godep.DeviceResponse
	cursor, _, err := s.store.RetrieveCursor(ctx, s.name)
	if err != nil {
		return err
	}
	logger := ctxlog.Logger(ctx, s.logger)

	// set our run mode (once vs. continuous)
	var ticker *time.Ticker
	if s.duration > 0 {
		ticker = time.NewTicker(s.duration)
		logger.Debug("msg", "starting timer", "duration", s.duration)
	}

	for {
		opts := make([]godep.DeviceRequestOption, 1, 2)
		opts[0] = godep.WithCursor(cursor)
		if s.limitOpt != nil {
			opts = append(opts, s.limitOpt)
		}
		if doFetch {
			resp, err = s.client.FetchDevices(ctx, s.name, opts...)
			if err != nil && godep.IsCursorExhausted(err) {
				logger.Debug(
					"msg", "cursor returned all devices previously",
					"phase", phaseLabel[doFetch],
					"cursor", cursor,
				)
				// we only see an exhausted cursor response on a fetch.
				// immediately move to a sync.
				doFetch = false
				continue
			}
		} else {
			resp, err = s.client.SyncDevices(ctx, s.name, opts...)
		}

		if err != nil {
			if godep.IsCursorExpired(err) || godep.IsCursorInvalid(err) {
				logger.Info(
					"msg", "cursor error, retrying with empty cursor",
					"phase", phaseLabel[doFetch],
					"cursor", cursor,
					"err", err,
				)
				// note: this will re-fetch the entire device list
				cursor = ""
				doFetch = true
				continue
			}
			logger.Info(
				"msg", "error syncing",
				"phase", phaseLabel[doFetch],
				"cursor", cursor,
				"err", err,
			)
			// errors are only logged and we just try again during the next cycle
		} else {
			logs := []interface{}{
				"msg", "device sync",
				"phase", phaseLabel[doFetch],
				"more", resp.MoreToFollow,
				"cursor", resp.Cursor,
				"devices", len(resp.Devices),
			}
			if !resp.FetchedUntil.IsZero() {
				// these just gunk up the logs if they're zero
				logs = append(logs, "fetched_until", resp.FetchedUntil)
			}
			logs = append(logs, logCountsForOpTypes(doFetch, resp.Devices)...)
			logger.Info(logs...)

			if s.callback != nil {
				err = s.callback(ctx, doFetch, resp)
				if err != nil {
					logger.Info("msg", "syncer callback", "err", err)
				}
			}

			if cursor != resp.Cursor {
				err = s.store.StoreCursor(ctx, s.name, resp.Cursor)
				if err != nil {
					return err
				}
				cursor = resp.Cursor
			}

			if resp.MoreToFollow {
				continue
			} else if doFetch {
				doFetch = false
				continue
			}
		}

		// if we're in "run once" mode then return after one cycle
		if ticker == nil {
			return nil
		}

		select {
		case <-ticker.C:
		case <-s.syncNow:
			logger.Debug("msg", "device sync: explicit sync requested")
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// logCountsForOpTypes tries to aggregate the various device "op_type"
// attributes so they can be logged.
func logCountsForOpTypes(isFetch bool, devices []godep.Device) []interface{} {
	opTypes := map[string]int{"added": 0, "modified": 0, "deleted": 0, "other": 0}
	var opType string
	for _, device := range devices {
		// normalize API input
		opType = strings.ToLower(device.OpType)
		if isFetch && opType == "" {
			// it seems no op_type is provided for a fetch sync
			continue
		}
		// we don't want to necessarily trust arbitrary op_types so restrict
		// our logging to some presets
		if _, ok := opTypes[opType]; !ok {
			opType = "other"
		}
		opTypes[opType] += 1
	}
	var logs []interface{}
	for k, v := range opTypes {
		if v > 0 {
			logs = append(logs, "op_type_"+k, v)
		}
	}
	return logs
}
