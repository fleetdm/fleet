// Package ctxerr provides functions to wrap errors with annotations and
// stack traces, and to handle those errors such that unique instances of
// those errors will be stored for an amount of time so that it can be
// used to troubleshoot issues.
//
// Typical uses of this package should be to call New or Wrap[f] as close as
// possible from where the error is encountered (or where it needs to be
// created for New), and then to call Handle with the error only once, after it
// bubbled back to the top of the call stack (e.g. in the HTTP handler, or in
// the CLI command, etc.). It is fine to wrap the error with more annotations
// along the way, by calling Wrap[f].
package ctxerr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type key int

const errHandlerKey key = 0

// Defining here for testing purposes
var nowFn = time.Now

// FleetError is the error implementation used by this package.
type FleetError struct {
	msg   string          // error message to be prepended to cause
	stack StackTracer     // stack trace where this error was created
	cause error           // original error that caused this error if non-nil
	data  json.RawMessage // json-marshalled additional metadata about the error (timestamps, etc)
}

type fleetErrorJSON struct {
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Stack   []string        `json:"stack,omitempty"`
}

// Error implements the error interface.
func (e *FleetError) Error() string {
	if e.cause == nil {
		return e.msg
	}
	return fmt.Sprintf("%s: %s", e.msg, e.cause.Error())
}

// Unwrap implements the error Unwrap interface introduced in go1.13.
func (e *FleetError) Unwrap() error {
	return e.cause
}

// MarshalJSON implements the marshaller interface, giving us control on how
// errors are json-encoded
func (e *FleetError) MarshalJSON() ([]byte, error) {
	return json.Marshal(fleetErrorJSON{
		Message: e.msg,
		Data:    e.data,
		Stack:   e.stack.List(),
	})
}

// Stack returns a call stack for the error
func (e *FleetError) Stack() []string {
	return e.stack.List()
}

// setMetadata adds common metadata attributes to the `data` map provided.
//
// NOTE: this will mutate the data provided and override other values with the same keys.
func setMetadata(ctx context.Context, data map[string]interface{}) map[string]interface{} {
	if data == nil {
		data = map[string]interface{}{}
	}

	// TODO: add more metadata from ctx
	data["timestamp"] = nowFn().Format(time.RFC3339)

	return data
}

func encodeData(ctx context.Context, data map[string]interface{}, augment bool) json.RawMessage {
	if augment {
		data = setMetadata(ctx, data)
	}

	encoded, err := json.Marshal(data)
	if err != nil {
		msg := fmt.Sprintf(`{"error": "there was an error encoding additional data: %s"}`, err.Error())
		encoded = json.RawMessage(msg)
	}
	return encoded
}

func newError(ctx context.Context, msg string, cause error, data map[string]interface{}) *FleetError {
	stack := NewStack(2)
	edata := encodeData(ctx, data, true)
	return &FleetError{msg, stack, cause, edata}
}

func wrapError(ctx context.Context, msg string, cause error, data map[string]interface{}) *FleetError {
	stack := NewStack(2)
	_, isFleetError := cause.(*FleetError)

	// If the error is a FleetError, don't add the full stack trace as it should
	// already be present.
	if isFleetError {
		stack = stack[:1]
	}

	edata := encodeData(ctx, data, !isFleetError)
	return &FleetError{msg, stack, cause, edata}
}

// New creates a new error with the given message.
func New(ctx context.Context, msg string) *FleetError {
	return newError(ctx, msg, nil, nil)
}

func NewWithData(ctx context.Context, msg string, data map[string]interface{}) *FleetError {
	return newError(ctx, msg, nil, data)
}

// Errorf creates a new error with the given message.
func Errorf(ctx context.Context, format string, args ...interface{}) *FleetError {
	msg := fmt.Sprintf(format, args...)
	return newError(ctx, msg, nil, nil)
}

// Wrap creates a new error with the given message, wrapping another error.
func Wrap(ctx context.Context, cause error, msgs ...string) *FleetError {
	msg := strings.Join(msgs, " ")
	return wrapError(ctx, msg, cause, nil)
}

// WrapWithData creates a new error with the given message, wrapping another
// error and attaching the data provided to it.
func WrapWithData(ctx context.Context, cause error, msg string, data map[string]interface{}) *FleetError {
	return wrapError(ctx, msg, cause, data)
}

// Wrapf creates a new error with the given message, wrapping another error.
func Wrapf(ctx context.Context, cause error, format string, args ...interface{}) *FleetError {
	msg := fmt.Sprintf(format, args...)
	return wrapError(ctx, msg, cause, nil)
}

// Cause returns the root error in err's chain.
func Cause(err error) error {
	for {
		uerr := Unwrap(err)
		if uerr == nil {
			return err
		}
		err = uerr
	}
}

// FleetCause is similar to Cause, but returns the root-most
// FleetError in the chain
func FleetCause(err error) *FleetError {
	var ferr, aux *FleetError = nil, nil
	var ok bool

	for err != nil {
		if aux, ok = err.(*FleetError); ok {
			ferr = aux
		}
		err = Unwrap(err)
	}

	return ferr
}

// Unwrap is a wrapper of built-in errors.Unwrap. It returns the result of
// calling the Unwrap method on err, if err's type contains an Unwrap method
// returning error. Otherwise, Unwrap returns nil.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// MarshalJSON provides a JSON representation of a whole error chain.
func MarshalJSON(err error) ([]byte, error) {
	chain := make([]interface{}, 0)

	for err != nil {
		switch v := err.(type) {
		case json.Marshaler:
			chain = append(chain, v)
		default:
			chain = append(chain, map[string]interface{}{"message": err.Error()})
		}

		err = Unwrap(err)
	}

	// reverse the chain to present errors in chronological order.
	for i := len(chain)/2 - 1; i >= 0; i-- {
		opp := len(chain) - 1 - i
		chain[i], chain[opp] = chain[opp], chain[i]
	}

	return json.MarshalIndent(struct {
		Cause interface{}   `json:"cause"`
		Wraps []interface{} `json:"wraps,omitempty"`
	}{
		Cause: chain[0],
		Wraps: chain[1:],
	}, "", "  ")
}

type handler interface {
	Store(error)
}

// NewContext returns a context derived from ctx that contains the provided
// error handler.
func NewContext(ctx context.Context, eh handler) context.Context {
	return context.WithValue(ctx, errHandlerKey, eh)
}

func fromContext(ctx context.Context) handler {
	v, _ := ctx.Value(errHandlerKey).(handler)
	return v
}

// Handle handles err by passing it to the registered error handler,
// deduplicating it and storing it for a configured duration.
func Handle(ctx context.Context, err error) {
	if eh := fromContext(ctx); eh != nil {
		eh.Store(err)
	}
}
