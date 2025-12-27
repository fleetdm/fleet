package ctxerr

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

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
