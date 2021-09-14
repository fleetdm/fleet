package errors

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/mna/redisc"
	"github.com/rotisserie/eris"
)

var errCh chan error
var running int32

type errorHandler struct {
	pool   fleet.RedisPool
	logger kitlog.Logger
}

type ErrorFlusher interface {
	Flush() ([]string, error)
}

func NewHandler(ctx context.Context, pool fleet.RedisPool, logger kitlog.Logger) ErrorFlusher {
	errCh = make(chan error)

	e := &errorHandler{
		pool:   pool,
		logger: logger,
	}
	go e.handleErrors(ctx)
	return e
}

func (e *errorHandler) Flush() ([]string, error) {
	errorKeys, err := redis.ScanKeys(e.pool, "error:*")
	if err != nil {
		return nil, err
	}
	keysBySlot := redis.SplitRedisKeysBySlot(e.pool, errorKeys...)
	var errors []string
	for _, qkeys := range keysBySlot {
		gotErrors, err := e.collectBatchErrors(qkeys)
		errors = append(errors, gotErrors...)
		if err != nil {
			return nil, err
		}
	}
	return errors, nil
}

func (e *errorHandler) collectBatchErrors(errorKeys []string) ([]string, error) {
	var errors []string

	conn := e.pool.Get()
	defer conn.Close()

	if rc, err := redisc.RetryConn(conn, 3, 100*time.Millisecond); err == nil {
		conn = rc
	}
	for _, key := range errorKeys {
		errorJson, err := redigo.String(conn.Do("GET", key))
		if err != nil {
			return nil, err
		}
		errors = append(errors, errorJson)
		_, err = conn.Do("DEL", key)
		if err != nil {
			return nil, err
		}
	}

	return errors, nil
}

func sha256b64(s string) string {
	src := sha256.Sum256([]byte(s))
	return base64.URLEncoding.EncodeToString(src[:])
}

func hashErrorLocation(err error) string {
	unpackedErr := eris.Unpack(err)
	if unpackedErr.ErrExternal != nil {
		return sha256b64(unpackedErr.ErrExternal.Error())
	}

	if len(unpackedErr.ErrRoot.Stack) == 0 {
		return sha256b64(unpackedErr.ErrRoot.Msg)
	}

	lastFrame := unpackedErr.ErrRoot.Stack[0]
	return sha256b64(fmt.Sprintf("%s:%d", lastFrame.File, lastFrame.Line))
}

func hashErr(externalErr error) (string, string, error) {
	m := eris.ToJSON(externalErr, true)
	bytes, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", "", err
	}
	return hashErrorLocation(externalErr), string(bytes), nil
}

func (e errorHandler) handleErrors(ctx context.Context) {
	atomic.StoreInt32(&running, 1)
	defer func() {
		atomic.StoreInt32(&running, 0)
	}()

	conn := e.pool.Get()
	defer conn.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errCh:
			errorHash, errorJson, err := hashErr(err)
			if err != nil {
				// TODO: log
				continue
			}

			jsonKey := fmt.Sprintf("error:%s:json", errorHash)

			_, err = conn.Do("SET", jsonKey, errorJson, "EX", (24 * time.Hour).Seconds())
			if err != nil {
				// TODO: log
				continue
			}
		}
	}
}

func New(err error) error {
	if atomic.LoadInt32(&running) == 0 {
		return err
	}

	// TODO: wrap in eris error with timestamp and other metadata

	ticker := time.NewTicker(2 * time.Second)
	select {
	case errCh <- err:
	case <-ticker.C:
	}

	return err
}

func NewHttpHandler(eh ErrorFlusher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		errors, err := eh.Flush()
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		bytes, err := json.Marshal(errors)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.Write(bytes)
		w.WriteHeader(http.StatusOK)
	}
}
