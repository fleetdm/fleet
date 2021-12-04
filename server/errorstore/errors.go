// Package errorstore implements a Handler type that can be used to store
// deduplicated instances of errors in an ephemeral storage, and provides a
// Flush method to retrieve the list of errors while clearing it at the same
// time. It provides a foundation to facilitate troubleshooting and building
// tooling for support while trying to keep the impact of storage to a minimum
// (ephemeral data, deduplication, flush on read).
package errorstore

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/rotisserie/eris" //nolint:depguard
)

// Handler defines an error handler. Call Handler.Store to handle an error, and
// Handler.Flush to retrieve all stored errors and clear them from the store.
// It is safe to call those methods concurrently.
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
	runHandler(ctx, eh)

	// Clear out any records that exist.
	// Temporary mitigation for #3065.
	if _, err := eh.Flush(); err != nil {
		level.Error(eh.logger).Log("err", err, "msg", "failed to flush redis errors")
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
	runHandler(ctx, eh)
	return eh
}

func runHandler(ctx context.Context, eh *Handler) {
	ch := make(chan error, 1)
	eh.errCh = ch
	go eh.handleErrors(ctx)
}

// Flush retrieves all stored errors from Redis and returns them as a slice of
// JSON-encoded strings. It is a destructive read - the errors are removed from
// Redis on return.
func (h *Handler) Flush() ([]string, error) {
	errorKeys, err := redis.ScanKeys(h.pool, "error:*", 100)
	if err != nil {
		return nil, err
	}

	keysBySlot := redis.SplitKeysBySlot(h.pool, errorKeys...)
	var errors []string
	for _, qkeys := range keysBySlot {
		if len(qkeys) > 0 {
			gotErrors, err := h.collectBatchErrors(qkeys)
			if err != nil {
				return nil, err
			}
			errors = append(errors, gotErrors...)
		}
	}
	return errors, nil
}

func (h *Handler) collectBatchErrors(errorKeys []string) ([]string, error) {
	conn := redis.ConfigureDoer(h.pool, h.pool.Get())
	defer conn.Close()

	var args redigo.Args
	args = args.AddFlat(errorKeys)
	errorList, err := redigo.Strings(conn.Do("MGET", args...))
	if err != nil {
		return nil, err
	}

	if _, err := conn.Do("DEL", args...); err != nil {
		return nil, err
	}

	return errorList, nil
}

func sha256b64(s string) string {
	src := sha256.Sum256([]byte(s))
	return base64.URLEncoding.EncodeToString(src[:])
}

func hashError(err error) string {
	// Ok so the hashing process is as follows:
	//
	// a) we want to hash the type and error message of the *root* error (the
	// last unwrapped error) so that if by mistake the same error is sent to
	// Handler.Handle in multiple places in the code, after being wrapped
	// differently any number of times, it still hashes to the same value
	// (because the root is the same). The type is not sufficient because some
	// errors have the same type but variable parts (e.g.  a struct value that
	// implements the error interface and the message contains a file name that
	// caused the error and that is stored in a struct field).
	//
	// b) in addition that a), we also want to hash all locations in the stack
	// trace, so that the same error type and message (say, sql.ErrNoRows or
	// io.UnexpectedEOF) caused at two different places in the code are not
	// considered the same error. To get that location, the error must be wrapped
	// at some point by eris.Wrap (or must be a user-created error via eris.New).
	// We cannot hash only the leaf frame in the stack trace as that would all be
	// ctxerr.New or ctxerr.Wrap (i.e. whatever common helper function used to
	// create the eris error).
	//
	// c) if we call eris.Unpack on an error that is not *directly* an "eris"
	// error (i.e. an error value returned from eris.Wrap or eris.New), then
	// eris.Unpack will not return any location information. So if for example
	// the error was wrapped with the pkg/errors.Wrap or the stdlib's fmt.Errorf
	// calls at some point, eris.Unpack will not give us any location info. To
	// get around this, we look for an eris-created error in the wrapped chain,
	// and only give up hashing the location if we can't find any.
	//
	// d) there is no easy way to identify an "eris" error (i.e. we cannot simply
	// use errors.As(err, <some Eris error type>)) as eris does not export its
	// error type, and it actually uses 2 different internal error types. To get
	// around this, we look for an error that has the `StackFrames() []uintptr`
	// method, as both of eris internal errors implement that (see
	// https://github.com/rotisserie/eris/blob/v0.5.1/eris.go#L182).

	var sf interface{ StackFrames() []uintptr }
	if errors.As(err, &sf) {
		err = sf.(error)
	}

	unpackedErr := eris.Unpack(err)

	if unpackedErr.ErrExternal == nil &&
		len(unpackedErr.ErrRoot.Stack) == 0 &&
		len(unpackedErr.ErrChain) == 0 {
		return sha256b64(unpackedErr.ErrRoot.Msg)
	}

	var sb strings.Builder
	if unpackedErr.ErrExternal != nil {
		root := eris.Cause(unpackedErr.ErrExternal)
		fmt.Fprintf(&sb, "%T\n%s\n", root, root.Error())
	}

	if len(unpackedErr.ErrRoot.Stack) > 0 {
		for _, frame := range unpackedErr.ErrRoot.Stack {
			fmt.Fprintf(&sb, "%s:%d\n", frame.File, frame.Line)
		}
	} else if len(unpackedErr.ErrChain) > 0 {
		lastFrame := unpackedErr.ErrChain[0].Frame
		fmt.Fprintf(&sb, "%s:%d", lastFrame.File, lastFrame.Line)
	}
	return sha256b64(sb.String())
}

func hashAndMarshalError(externalErr error) (errHash string, errAsJson string, err error) {
	m := eris.ToJSON(externalErr, true)
	bytes, err := json.MarshalIndent(m, "", "  ")
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
	// Skip storing errors due to SCAN issues with Redis (see #3065).
	// if true here because otherwise we get linting errors for unreachable code.
	if true {
		return
	}

	errorHash, errorJson, err := hashAndMarshalError(err)
	if err != nil {
		level.Error(h.logger).Log("err", err, "msg", "hashErr failed")
		if h.testOnStore != nil {
			h.testOnStore(err)
		}
		return
	}
	jsonKey := fmt.Sprintf("error:%s:json", errorHash)

	conn := redis.ConfigureDoer(h.pool, h.pool.Get())
	defer conn.Close()

	secs := int(h.ttl.Seconds())
	if secs <= 0 {
		secs = 1 // SET EX fails if ttl is <= 0
	}
	if _, err := conn.Do("SET", jsonKey, errorJson, "EX", secs); err != nil {
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

// ServeHTTP implements an http.Handler that flushes the errors stored
// by the Handler and returns them in the response as JSON.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	errors, err := h.Flush()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// each string returned by eh.Flush is already JSON-encoded, so to prevent
	// double-marshaling while still marshaling the list of errors as a JSON
	// array, treat them as raw json messages.
	raw := make([]json.RawMessage, len(errors))
	for i, s := range errors {
		raw[i] = json.RawMessage(s)
	}

	bytes, err := json.Marshal(raw)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(bytes)
}
