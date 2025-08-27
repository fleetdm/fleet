package log

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/gitops-migrate/ansi"
)

const defaultSkip = 2

func Debug(msg string, pairs ...any) {
	log(LevelDebug, defaultSkip, msg, pairs...)
}

func Debugf(msg string, values ...any) {
	logf(LevelDebug, defaultSkip, msg, values...)
}

func Info(msg string, pairs ...any) {
	log(LevelInfo, defaultSkip, msg, pairs...)
}

func Infof(msg string, values ...any) {
	logf(LevelInfo, defaultSkip, msg, values...)
}

func Warn(msg string, pairs ...any) {
	log(LevelWarn, defaultSkip, msg, pairs...)
}

func Warnf(msg string, values ...any) {
	logf(LevelWarn, defaultSkip, msg, values...)
}

func Error(msg string, pairs ...any) {
	logf(LevelError, defaultSkip, msg, pairs...)
}

func Errorf(msg string, values ...any) {
	logf(LevelError, defaultSkip, msg, values...)
}

func Fatal(msg string, pairs ...any) {
	logf(LevelFatal, defaultSkip, msg, pairs...)
	os.Exit(1)
}

func Fatalf(msg string, values ...any) {
	logf(LevelFatal, defaultSkip, msg, values...)
	os.Exit(1)
}

func Panic(msg string, values ...any) {
	panic(fmt.Sprintf(msg, values...))
}

var builderPool = &sync.Pool{
	New: func() any {
		sb := new(strings.Builder)
		sb.Grow(4096)
		return sb
	},
}

const (
	brackL       = ansi.BoldBlack + "[" + ansi.Reset
	brackR       = ansi.BoldBlack + "]" + ansi.Reset
	arrow        = ansi.BoldMagenta + "=>" + ansi.Reset
	rowMiddle    = ansi.BoldMagenta + "┣━━ " + ansi.Reset
	rowBottom    = ansi.BoldMagenta + "┗━━ " + ansi.Reset
	valueMissing = "<NOVALUE>"
)

func log(l level, skip int, msg string, pairs ...any) {
	if l < Level {
		return
	}

	// Grab a string builder from the pool, defer it's reset and return to
	// the pool.
	b := builderPool.Get().(*strings.Builder)
	defer builderPool.Put(b)
	defer b.Reset()

	// Write the log level, if the appropriate configuration is set.
	writeLevel(b, l)

	// Write the caller if the appropriate configuration is set.
	writeCaller(b, skip+1)

	// Write the formatted message, followed by a newline.
	fmt.Fprintln(b, msg)

	// Produce all pairs.
	writePairs(b, pairs...)

	// Dump the buffer to the package writer.
	fmt.Fprint(output, b.String())
}

func writeLevel(b *strings.Builder, l level) {
	// Write the log level, if the appropriate configuration is set.
	if Options.WithLevel() {
		fmt.Fprintf(b, "%s ", l)
	}
}

func writeCaller(b *strings.Builder, skip int) {
	// Write the caller if the appropriate configuration is set.
	if Options.WithCaller() {
		// Init an array to send to the caller functions.
		pcs := [1]uintptr{}

		// Populate the array with the program counter we're after.
		runtime.Callers(skip+1, pcs[:])

		// Get the caller frame for the program counter we captured.
		frame, _ := runtime.CallersFrames(pcs[:]).Next()

		// Write the caller short file + line.
		file := frame.File
		line := frame.Line
		if i := strings.LastIndexByte(file, '/'); i >= 0 && i <= len(file)-1 {
			file = file[i+1:]
		}
		fmt.Fprintf(b, "%s[%s:%d]%s ", ansi.BoldWhite, file, line, ansi.Reset)
	}
}

func writePairs(b *strings.Builder, pairs ...any) {
	for i := 0; i < len(pairs); i += 2 {
		// Grab the key + value.
		key := fmt.Sprint(pairs[i])
		var val string
		if i+1 > len(pairs) {
			val = valueMissing
		} else {
			val = fmt.Sprint(pairs[i+1])
		}

		// Write the prefixed box characters.
		if i < len(pairs)-2 {
			b.WriteString(rowMiddle)
		} else {
			b.WriteString(rowBottom)
		}

		// Write the key.
		b.WriteString(brackL + ansi.BoldWhite + key + ansi.Reset + brackR)

		// Write the '=>'.
		b.WriteString(arrow)

		// Write the value, followed by a newline.
		b.WriteString(brackL + ansi.BoldWhite + val + ansi.Reset + brackR + "\n")
	}
}

func logf(l level, skip int, msg string, values ...any) {
	msg = fmt.Sprintf(msg, values...)
	log(l, skip+1, msg)
}
