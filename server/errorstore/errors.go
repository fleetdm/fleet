// Package errorstore implements a Handler type that can be used to store
// deduplicated instances of errors in an ephemeral storage, and provides a
// Retrieve method to retrieve the list of errors with the option of clearing
// them at the same time. It provides a foundation to facilitate
// troubleshooting and building tooling for support while trying to keep the
// impact of storage to a minimum (ephemeral data, deduplication, flush on
// read).
package errorstore

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	redigo "github.com/gomodule/redigo/redis"
)

// Handler defines an error handler. Call Handler.Store to handle an error, and
// Handler.Retrieve to retrieve all stored errors and optionally clear them
// from the store. It is safe to call those methods concurrently.
type Handler struct {
	pool    fleet.RedisPool
	logger  kitlog.Logger
	ttl     time.Duration
	running int32 // accessed atomically
	errCh   chan error

	// for tests
	syncStore   bool        // if true, store error synchronously
	testOnStore func(error) // if set, called each time an error is stored
	testOnStart func()      // if set, called once the handler is running
}

// NewHandler creates an error handler using the provided pool and logger,
// storing unique instances of errors in Redis using the pool. It stops storing
// errors when ctx is cancelled. Errors are kept for the duration of ttl.
func NewHandler(ctx context.Context, pool fleet.RedisPool, logger kitlog.Logger, ttl time.Duration) *Handler {
	eh := &Handler{
		pool:   pool,
		logger: logger,
		ttl:    ttl,
	}
	if ttl >= 0 {
		runHandler(ctx, eh)
	}

	return eh
}

func newTestHandler(ctx context.Context, pool fleet.RedisPool, logger kitlog.Logger, ttl time.Duration, onStart func(), onStore func(error)) *Handler {
	eh := &Handler{
		pool:   pool,
		logger: logger,
		ttl:    ttl,

		syncStore:   true,
		testOnStart: onStart,
		testOnStore: onStore,
	}

	if ttl >= 0 {
		runHandler(ctx, eh)
	}
	return eh
}

func runHandler(ctx context.Context, eh *Handler) {
	ch := make(chan error, 1)
	eh.errCh = ch
	go eh.handleErrors(ctx)
}

// Retrieve retrieves all stored errors from Redis and returns them as a slice
// of JSON-encoded strings.
//
// If flush is `true`, performs a destructive read - the errors are removed
// from Redis on return.
func (h *Handler) Retrieve(flush bool) ([]*ctxerr.StoredError, error) {
	// scanning only the error:*:json keys as json and count are both tagged keys
	// and should hash to the same Redis slot
	errorKeys, err := redis.ScanKeys(h.pool, "error:*:json", 100)
	if err != nil {
		return nil, err
	}

	keysBySlot := redis.SplitKeysBySlot(h.pool, errorKeys...)
	var rawErrs []interface{}
	for _, qkeys := range keysBySlot {
		if len(qkeys) > 0 {
			gotErrors, err := h.collectBatchErrors(qkeys, flush)
			if err != nil {
				return nil, err
			}
			rawErrs = append(rawErrs, gotErrors...)
		}
	}

	errorList := make([]*ctxerr.StoredError, 0, len(rawErrs))
	if err := redigo.ScanSlice(rawErrs, &errorList); err != nil {
		return nil, err
	}
	return errorList, nil
}

func (h *Handler) collectBatchErrors(errorKeys []string, flush bool) ([]interface{}, error) {
	conn := redis.ConfigureDoer(h.pool, h.pool.Get())
	defer conn.Close()

	var args redigo.Args
	for _, ek := range errorKeys {
		// note: the order in which the keys are requested must match the order in
		// which the struct fields are declared (due to how redigo.ScanSlice works)
		args = args.Add(strings.Replace(ek, ":json", ":count", 1))
		args = args.Add(ek)
	}

	errorList, err := redigo.Values(conn.Do("MGET", args...))
	if err != nil {
		return nil, err
	}

	if flush {
		if _, err := conn.Do("DEL", args...); err != nil {
			return nil, err
		}
	}

	return errorList, nil
}

func sha256b64(s string) string {
	src := sha256.Sum256([]byte(s))
	return base64.URLEncoding.EncodeToString(src[:])
}

func hashError(err error) string {
	cause := ctxerr.Cause(err)
	ferr := ctxerr.FleetCause(err)

	var sb strings.Builder
	// hash the cause type and message (it might not be a FleetError)
	fmt.Fprintf(&sb, "%T\n%s\n", cause, cause.Error())

	// hash the stack trace of the root FleetError in the chain
	if ferr != nil {
		fmt.Fprint(&sb, strings.Join(ferr.Stack(), "\n"))
	}

	// hash the vital FleetError fields
	if fleetdErr, ok := cause.(fleet.FleetdError); ok {
		_, _ = fmt.Fprint(&sb,
			fleetdErr.ErrorSource,
			fleetdErr.ErrorSourceVersion,
			fleetdErr.ErrorAdditionalInfo,
		)
	}

	return sha256b64(sb.String())
}

func hashAndMarshalError(externalErr error) (errHash string, errAsJson string, err error) {
	bytes, err := ctxerr.MarshalJSON(externalErr)
	if err != nil {
		return "", "", err
	}
	return hashError(externalErr), string(bytes), nil
}

func (h *Handler) handleErrors(ctx context.Context) {
	atomic.StoreInt32(&h.running, 1)
	defer func() {
		atomic.StoreInt32(&h.running, 0)
	}()

	if h.testOnStart != nil {
		h.testOnStart()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-h.errCh:
			h.storeError(ctx, err)
		}
	}
}

func (h *Handler) storeError(ctx context.Context, err error) {
	errorHash, errorJson, err := hashAndMarshalError(err)
	if err != nil {
		level.Error(h.logger).Log("err", err, "msg", "hashErr failed")
		if h.testOnStore != nil {
			h.testOnStore(err)
		}
		return
	}

	// not using a connection that follows redirections here
	// in order to do pipeline commands
	conn := h.pool.Get()
	defer conn.Close()

	jsonKey := fmt.Sprintf("error:{%s}:json", errorHash)
	countKey := fmt.Sprintf("error:{%s}:count", errorHash)

	conn.Send("SET", jsonKey, errorJson) //nolint:errcheck
	conn.Send("INCR", countKey)          //nolint:errcheck

	if h.ttl > 0 {
		secs := int(h.ttl.Seconds())
		if secs <= 0 {
			secs = 1 // EXPIRE fails if ttl is <= 0
		}
		conn.Send("EXPIRE", jsonKey, secs)  //nolint:errcheck
		conn.Send("EXPIRE", countKey, secs) //nolint:errcheck
	}

	if _, err := conn.Do(""); err != nil {
		level.Error(h.logger).Log("err", err, "msg", "redis SET failed")
		if h.testOnStore != nil {
			h.testOnStore(err)
		}
		return
	}

	if h.testOnStore != nil {
		h.testOnStore(nil)
	}
}

// Store handles the provided error by storing it into Redis if the handler is
// still running.
//
// It waits for a predefined period of time to try to store the error but does
// so in a goroutine so the call returns immediately.
func (h *Handler) Store(err error) {
	exec := func() {
		if atomic.LoadInt32(&h.running) == 0 {
			return
		}

		timer := time.NewTimer(2 * time.Second)
		defer timer.Stop()
		select {
		case h.errCh <- err:
		case <-timer.C:
		}
	}

	if h.syncStore {
		exec()
	} else {
		go exec()
	}
}

// ServeHTTP implements an http.Handler that retrieves the errors stored
// by the Handler and returns them in the response as JSON.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	var flush bool
	opts := r.URL.Query()

	if opts.Has("flush") {
		var err error
		flush, err = strconv.ParseBool(opts.Get("flush"))

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	errors, err := h.Retrieve(flush)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bytes, err := json.Marshal(errors)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes) //nolint:errcheck
}
