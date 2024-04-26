// Package ctxlog allows logging data stored with a context.
package ctxlog

import (
	"context"
	"sync"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/log"
)

// CtxKVFunc creates logger key-value pairs from a context.
// CtxKVFuncs should aim to be be as efficient as possible—ideally only
// doing the minimum to read context values and generate KV pairs. Each
// associated CtxKVFunc is called every time we adapt a logger with
// Logger.
type CtxKVFunc func(context.Context) []interface{}

// ctxKeyFuncs is the context key for storing and retriveing
// a funcs{} struct on a context.
type ctxKeyFuncs struct{}

// funcs holds the associated CtxKVFunc functions to run.
type funcs struct {
	sync.RWMutex
	funcs []CtxKVFunc
}

// AddFunc associates a new CtxKVFunc function to a context.
func AddFunc(ctx context.Context, f CtxKVFunc) context.Context {
	if ctx == nil {
		return ctx
	}
	ctxFuncs, ok := ctx.Value(ctxKeyFuncs{}).(*funcs)
	if !ok || ctxFuncs == nil {
		ctxFuncs = &funcs{}
	}
	ctxFuncs.Lock()
	ctxFuncs.funcs = append(ctxFuncs.funcs, f)
	ctxFuncs.Unlock()
	return context.WithValue(ctx, ctxKeyFuncs{}, ctxFuncs)
}

// Logger runs the associated CtxKVFunc functions and returns a new
// logger with the results.
func Logger(ctx context.Context, logger log.Logger) log.Logger {
	if ctx == nil {
		return logger
	}
	ctxFuncs, ok := ctx.Value(ctxKeyFuncs{}).(*funcs)
	if !ok || ctxFuncs == nil {
		return logger
	}
	var acc []interface{}
	ctxFuncs.RLock()
	for _, f := range ctxFuncs.funcs {
		acc = append(acc, f(ctx)...)
	}
	ctxFuncs.RUnlock()
	return logger.With(acc...)
}

// SimpleStringFunc is a helper that generates a simple CtxKVFunc that
// returns a key-value pair if found on the context.
func SimpleStringFunc(logKey string, ctxKey interface{}) CtxKVFunc {
	return func(ctx context.Context) (out []interface{}) {
		v, _ := ctx.Value(ctxKey).(string)
		if v != "" {
			out = []interface{}{logKey, v}
		}
		return
	}
}
