package ctxerr

import (
	"fmt"
	"path/filepath"
	"runtime"
)

const (
	maxDepth = 10 // maximum number of stack frames to record
)

type StackTracer interface {
	List() []string
}

// Stack holds a snapshot of program counters.
type Stack []uintptr

// NewStack captures a stack trace. skip specifies the number of frames to skip from
// a stack trace. skip=0 records stack.New call as the innermost frame.
func NewStack(skip int) Stack {
	pc := make([]uintptr, maxDepth+1)
	pc = pc[:runtime.Callers(skip+2, pc)]
	return Stack(pc)
}

// List collects stack traces formatted as strings.
func (s Stack) List() []string {
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
