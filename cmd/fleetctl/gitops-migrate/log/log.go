package log

import (
	"fmt"
	"os"
	"runtime"
	"strings"

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

func log(l level, skip int, msg string, pairs ...any) {
	if l < Level {
		return
	}

	// Write the log level, if the appropriate configuration is set.
	if Options.WithLevel() {
		fmt.Fprintf(output, "%s ", l)
	}

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
		fmt.Fprintf(output, "%s[%s:%d]%s ", ansi.BoldWhite, file, line, ansi.Reset)
	}

	// Write the formatted message, followed by a newline.
	fmt.Fprintln(output, msg)

	// Produce all pairs.
	longestKey := 0
	longestVal := 0
	fmtPairs := make([][2]string, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key := fmt.Sprint(pairs[i])
		if len(key) > longestKey {
			longestKey = len(key)
		}

		var val string
		if i+1 < len(pairs) {
			val = fmt.Sprint(pairs[i+1])
		} else {
			val = "<MISSING>"
		}
		if len := len(val); len > longestVal {
			longestVal = len
		}

		fmtPairs = append(fmtPairs, [2]string{key, val})
	}
	for _, pair := range fmtPairs {
		key, val := pair[0], pair[1]
		fmt.Fprintf(output, "  %s>>%s %s%s%s", ansi.BoldMagenta, ansi.Reset, ansi.BoldWhite, key, ansi.Reset)
		for range longestKey - len(key) + 1 {
			fmt.Fprint(output, " ")
		}
		fmt.Fprintf(output, "%s=>%s ", ansi.BoldMagenta, ansi.Reset)
		fmt.Fprintf(output, "%s%s%s", ansi.BoldWhite, val, ansi.Reset)
		fmt.Fprintln(output)
	}

	// Flush the buffered writer.
	output.Flush()
}

func logf(l level, skip int, msg string, values ...any) {
	msg = fmt.Sprintf(msg, values...)
	log(l, skip+1, msg)
}
