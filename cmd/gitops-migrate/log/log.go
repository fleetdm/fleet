package log

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/fleetdm/fleet/v4/cmd/gitops-migrate/ansi"
)

// The default number of stack frames to skip when we grab the program counter
// and produce caller details.
//
// This package offers the ability to produce the 'caller' as part of the
// output. Considering we have paths with varying call stack depth to get to the
// point where we _actually generate_ the caller, we need to track the number of
// frames to skip when we get there.
const defaultSkip = 2

// Log an debug-level message, with optional variadic key-value pairs.
func Debug(msg string, pairs ...any) {
	log(LevelDebug, defaultSkip, msg, pairs...)
}

// Log a printf debug-level message.
func Debugf(msg string, values ...any) {
	logf(LevelDebug, defaultSkip, msg, values...)
}

// Log an info-level message, with optional variadic key-value pairs.
func Info(msg string, pairs ...any) {
	log(LevelInfo, defaultSkip, msg, pairs...)
}

// Log a printf info-level message.
func Infof(msg string, values ...any) {
	logf(LevelInfo, defaultSkip, msg, values...)
}

// Log an warn-level message, with optional variadic key-value pairs.
func Warn(msg string, pairs ...any) {
	log(LevelWarn, defaultSkip, msg, pairs...)
}

// Log a printf warn-level message.
func Warnf(msg string, values ...any) {
	logf(LevelWarn, defaultSkip, msg, values...)
}

// Log an error-level message, with optional variadic key-value pairs.
func Error(msg string, pairs ...any) {
	log(LevelError, defaultSkip, msg, pairs...)
}

// Log a printf error-level message.
func Errorf(msg string, values ...any) {
	logf(LevelError, defaultSkip, msg, values...)
}

// Log an fatal-level message, with optional variadic key-value pairs, followed
// by a call to 'os.Exit(1)'.
func Fatal(msg string, pairs ...any) {
	log(LevelFatal, defaultSkip, msg, pairs...)
	os.Exit(1)
}

// Log a printf fatal-level message, followed by a call to 'os.Exit(1)'.
func Fatalf(msg string, values ...any) {
	logf(LevelFatal, defaultSkip, msg, values...)
	os.Exit(1)
}

// Panic with the provided message and optional variadic key-value pairs.
func Panic(msg string, pairs ...any) {
	sb := builderPool.Get().(builder)
	defer builderPool.Put(sb)
	defer sb.Reset()
	sb.WriteString(msg)
	writePairs(sb, pairs...)
	panic(sb.String())
}

// Panic with the provided printf message.
func Panicf(msg string, values ...any) {
	panic(fmt.Sprintf(msg, values...))
}

// If we want to tweak the io.Writer returned by 'builderPool' (currently
// '*strings.Builder') we can simply update this alias, which is implemented
// at all the 'builderPool' call sites for the assertion when getting from the
// pool.
type builder = *strings.Builder

// To avoid races with concurrent calls to this package we use
// 'strings.Builder's to buffer our writes then send it to the package-level
// io.Writer ('output'). Creating and destroying buffers is expensive so,
// instead, we can grab and reinsert from this pool.
var builderPool = &sync.Pool{
	New: func() any {
		sb := new(strings.Builder)
		sb.Grow(4096)
		return sb
	},
}

var (
	brackL    = ansi.Blue + "[" + ansi.Reset
	brackR    = ansi.Blue + "]" + ansi.Reset
	arrow     = ansi.BoldBlack + "=>" + ansi.Reset
	rowMiddle = ansi.Magenta + "┣━ " + ansi.Reset
	rowBottom = ansi.Magenta + "┗━ " + ansi.Reset

	// The line prefix used when the 'WithLevel' option is _not_ set.
	linePrefix = ">"

	// Placeholder value for where len(pairs) % 2 != 0.
	valueMissing = "<NOVALUE>"
)

// log formats and writes a log entry to the package-level io.Writer ('output').
func log(l level, skip int, msg string, pairs ...any) {
	if l < Level {
		return
	}

	// Grab a string builder from the pool, defer its reset and return to
	// the pool.
	b := builderPool.Get().(builder)
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

// logf simply formats the printf message before sending to 'log'.
func logf(l level, skip int, msg string, values ...any) {
	msg = fmt.Sprintf(msg, values...)
	log(l, skip+1, msg)
}

// Write the log level (ex: 'INF').
func writeLevel(w io.Writer, l level) {
	if l < Level {
		return
	}
	// Write the log level, if the appropriate configuration is set, otherwise
	// just prefix the line with a caret.
	pfx := linePrefix
	if Options.WithLevel() {
		pfx = l.String()
	}
	var color string
	switch l {
	case LevelDebug:
		color = colorDBG
	case LevelInfo:
		color = colorINF
	case LevelWarn:
		color = colorWRN
	case LevelError:
		color = colorERR
	case LevelFatal:
		color = colorFTL
	default:
		color = colorDBG
	}
	fmt.Fprintf(w, "%s%s%s ", color, pfx, colorReset)
}

// Write the caller in 'short_file:line_number' format.
func writeCaller(w io.Writer, skip int) {
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
		fmt.Fprintf(w, "%s[%s:%d]%s ", colorCaller, file, line, colorReset)
	}
}

// Write variadic 'pairs' in a 'key=value' format.
//
// The standard log functions ('Info', 'Warn', etc.) allow for variadic pairs
// (like slog) which are treated as key-value pairs. Here we iterate them in
// groups of two and output them as formatted log artifact rows.
func writePairs(b *strings.Builder, pairs ...any) {
	for i := 0; i < len(pairs); i += 2 {
		// Grab the key + value.
		//
		// We default to 'valueMissing' for the value, only assigning the actual
		// value in the 'pairs' slice once we've successfully bounds checked the
		// index.
		key := fmt.Sprint(pairs[i])
		val := valueMissing
		if i+1 < len(pairs) {
			val = fmt.Sprint(pairs[i+1])
		}

		// Write the prefixed box characters.
		if i+1 >= len(pairs)-1 {
			b.WriteString(rowBottom)
		} else {
			b.WriteString(rowMiddle)
		}

		// Write the key.
		b.WriteString(brackL + colorKey + key + colorReset + brackR)

		// Write the '=>'.
		b.WriteString(arrow)

		// Write the value, followed by a newline.
		b.WriteString(brackL + colorVal + val + colorReset + brackR + "\n")
	}
}
