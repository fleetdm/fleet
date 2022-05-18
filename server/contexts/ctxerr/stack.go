package ctxerr

import (
	"fmt"
	"path/filepath"
	"runtime"
)

const (
	maxDepth = 10 // maximum number of stack frames to record
)

type stackTracer interface {
	List() []string
}

// stack holds a snapshot of program counters.
type stack []uintptr

// newStack captures a stack trace. skip specifies the number of frames to skip from
// a stack trace. skip=0 records stack.New call as the innermost frame.
func newStack(skip int) stack {
	pc := make([]uintptr, maxDepth+1)
	pc = pc[:runtime.Callers(skip+2, pc)]
	return stack(pc)
}

// List collects stack traces formatted as strings.
func (s stack) List() []string {
	var lines []string

	cf := runtime.CallersFrames(s)
	for {
		f, more := cf.Next()
		line := fmt.Sprintf("%s (%s:%d)", f.Function, filepath.Base(f.File), f.Line)
		lines = append(lines, line)

		if !more {
			break
		}
	}

	return lines
}
