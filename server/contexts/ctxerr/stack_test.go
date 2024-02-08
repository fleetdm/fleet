package ctxerr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.elastic.co/apm/v2"
	"go.elastic.co/apm/v2/apmtest"
)

type mockStack struct {
	trace []string
}

func (s mockStack) List() []string {
	return s.trace
}

func buildStack(depth int) stack {
	if depth == 0 {
		return newStack(0)
	}
	return buildStack(depth - 1)
}

func TestStack(t *testing.T) {
	trace := buildStack(maxDepth)
	lines := trace.List()

	require.Equal(t, len(lines), len(trace))

	re := regexp.MustCompile(`server/contexts/ctxerr\.buildStack \(stack_test.go:\d+\)$`)
	for i, line := range lines {
		require.Regexpf(t, re, line, "expected line %d to match %q, got %q", i, re, line)
	}
}

func TestElasticStack(t *testing.T) {
	ctx := context.Background()

	var wrap = errors.New("wrap")
	errFn := func(fn func() error) error { // func1
		if err := fn(); err != nil {
			if err == wrap {
				return Wrap(ctx, err, "wrapped")
			}
			return err
		}
		return nil
	}

	cases := []struct {
		desc               string
		chain              func() error
		causeStackContains []string
		leafStackContains  []string
	}{
		{
			desc: "depth 2, wrap in errFn",
			chain: func() error {
				// gets wrapped in errFn, so top of the stack is func1
				return errFn(func() error { return wrap })
			},
			causeStackContains: []string{
				"/ctxerr.TestElasticStack.func1 ",
			},
		},
		{
			desc: "depth 2, wrap immediately",
			chain: func() error {
				// gets wrapped immediately when returned, so top of the stack is funcX.1
				return errFn(func() error { return Wrap(ctx, wrap) })
			},
			causeStackContains: []string{
				"/ctxerr.TestElasticStack.func3.1 ",
				"/ctxerr.TestElasticStack.func1 ", // errFn
			},
		},
		{
			desc: "depth 3, ctxerr.New",
			chain: func() error {
				// gets wrapped directly in the call to New, so top of the stack is X.1.1
				return errFn(func() error { return func() error { return New(ctx, "new") }() })
			},
			causeStackContains: []string{
				"/ctxerr.TestElasticStack.func4.1.1 ",
				"/ctxerr.TestElasticStack.func4.1 ",
				"/ctxerr.TestElasticStack.func1 ", // errFn
			},
		},
		{
			desc: "depth 4, ctxerr.New",
			chain: func() error {
				// stacked capture in New, so top of the stack is X.1.1.1
				return errFn(func() error {
					return func() error {
						return func() error {
							return New(ctx, "new")
						}()
					}()
				})
			},
			causeStackContains: []string{
				"/ctxerr.TestElasticStack.func5.1.1.1 ",
				"/ctxerr.TestElasticStack.func5.1.1 ",
				"/ctxerr.TestElasticStack.func5.1 ",
				"/ctxerr.TestElasticStack.func1 ", // errFn
			},
		},
		{
			desc: "depth 4, ctxerr.New always wrapped",
			chain: func() error {
				// stacked capture in New, so top of the stack is X.1.1.1
				return errFn(func() error {
					return Wrap(ctx, func() error {
						return Wrap(ctx, func() error {
							return New(ctx, "new")
						}())
					}())
				})
			},
			causeStackContains: []string{
				"/ctxerr.TestElasticStack.func6.1.1.1 ",
				"/ctxerr.TestElasticStack.func6.1.1 ",
				"/ctxerr.TestElasticStack.func6.1 ",
				"/ctxerr.TestElasticStack.func1 ", // errFn
			},
			leafStackContains: []string{
				// only a single stack trace is collected when wrapping another
				// FleetError.
				"/ctxerr.TestElasticStack.func6.1 ",
			},
		},
		{
			desc: "depth 4, wrapped only at the end",
			chain: func() error {
				return errFn(func() error {
					return Wrap(ctx, func() error {
						return func() error {
							return io.EOF
						}()
					}())
				})
			},
			causeStackContains: []string{
				// since it wraps a non-FleetError, the full stack is collected
				"/ctxerr.TestElasticStack.func7.1 ",
				"/ctxerr.TestElasticStack.func1 ", // errFn
			},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			checkStack := func(stack, contains []string) {
				stackStr := strings.Join(stack, "\n")
				lastIx := -1
				for _, want := range contains {
					ix := strings.Index(stackStr, want)
					require.True(t, ix > -1, "expected stack %v to contain %q", stackStr, want)
					require.True(t, ix > lastIx, "expected %q to be after last check in %v", want, stackStr)
					lastIx = ix
				}
			}

			err := c.chain()
			require.Error(t, err)
			var ferr *FleetError
			require.ErrorAs(t, err, &ferr)

			leafStack := ferr.Stack()
			cause := FleetCause(err)
			causeStack := cause.Stack()

			// if the fleet root error != fleet leaf error, then separate leaf +
			// cause stacks must be provided.
			if cause != ferr {
				require.True(t, len(c.causeStackContains) > 0)
				require.True(t, len(c.leafStackContains) > 0)
			} else {
				// otherwise use the same stack expectations for both
				if len(c.causeStackContains) == 0 {
					c.causeStackContains = c.leafStackContains
				}
				if len(c.leafStackContains) == 0 {
					c.leafStackContains = c.causeStackContains
				}
			}

			checkStack(causeStack, c.causeStackContains)
			checkStack(leafStack, c.leafStackContains)

			// run in a test APM transaction, recording the sent events
			_, _, apmErrs := apmtest.NewRecordingTracer().WithTransaction(func(ctx context.Context) {
				// APM should be able to capture that error (we use the FleetCause error,
				// so that it gets the most relevant stack trace and not that of the
				// wrapped ones) and its cause is the cause.
				apm.CaptureError(ctx, cause).Send()
			})
			require.Len(t, apmErrs, 1)
			apmErr := apmErrs[0]
			require.NotNil(t, apmErr)

			// the culprit should be the function name of the top of the stack of the
			// cause error.
			fnName := strings.TrimSpace(c.causeStackContains[0][strings.Index(c.causeStackContains[0], "TestElasticStack"):])
			require.Equal(t, fnName, apmErr.Culprit)

			// the APM stack should match the cause stack (i.e. APM should have
			// grabbed the stacktrace that we provided). If it didn't, it would have
			// a stacktrace with a function name indicating the WithTransaction
			// function literal where CaptureError is called.
			var apmStack []string
			for _, st := range apmErr.Exception.Stacktrace {
				apmStack = append(apmStack, st.Module+"."+st.Function+" ("+st.File+":"+fmt.Sprint(st.Line)+")")
			}
			checkStack(apmStack, c.causeStackContains)
		})
	}
}
