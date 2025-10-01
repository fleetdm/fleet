package log

import (
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// For all tests in this package, to make assertion life easier we hijack
// certain logging variables, such as symbols, box drawing characters and more.
// Considering this, 'setup' should be called at the top of any 'log' or 'logf'
// (or their callees) calls to ensure consistent behavior.
func setup(t *testing.T) testBuffer {
	t.Helper()

	// Hijack the package-level 'io.Writer' so we can observe the output produced
	// by our logging functions.
	buffer := new(strings.Builder)
	buffer.Grow(4096)
	SetOutput(buffer)

	// Hijack the various box-drawing characters, symbols and other
	// 'pairs'-related items so we don't have to muck with ANSI coloring in tests.
	arrow = "=>"
	brackL = "["
	brackR = "]"
	colorDBG = ""
	colorINF = ""
	colorWRN = ""
	colorERR = ""
	colorFTL = ""
	colorKey = ""
	colorVal = ""
	colorReset = ""
	colorCaller = ""

	// We just use 1/2 null bytes for these since they'll never collide with test
	// inputs... Probably™.
	rowMiddle = "\x00"
	rowBottom = "\x00\x00"

	return buffer
}

const (
	// Standard 'log' and 'logf' inputs.
	logBasic  = "hello, world!"
	logfBasic = "hello, %s!"
	logfValue = "world"
)

// Belongs with the constants above; these are sample 'log' pairs representing
// the main primitive data types.
var logPairs = []any{"key", "value", "key2", 2, "key3", true, "key4", 1.44}

func TestLog(t *testing.T) {
	buffer := setup(t)

	// Standard 'log' cases.
	Level = LevelDebug
	expectLog(t, buffer, LevelDebug, logBasic)
	expectLog(t, buffer, LevelInfo, logBasic)
	expectLog(t, buffer, LevelWarn, logBasic)
	expectLog(t, buffer, LevelError, logBasic)

	// Log with pairs (one of each primitive data type).
	expectLog(t, buffer, LevelInfo, logBasic, logPairs...)
	// Log with uneven number of pairs.
	expectLog(t, buffer, LevelInfo, logBasic, logPairs[len(logPairs)-1:])
}

func expectLog(t *testing.T, buffer testBuffer, l level, input string, pairs ...any) {
	t.Helper()
	defer buffer.Reset()

	// Call the logging function.
	log(l, defaultSkip, input, pairs...)

	if l < Level {
		// If the provided level is LOWER than the package level, we should expect
		// an empty buffer.
		require.Empty(t, buffer.String())
		return
	} else { //nolint:revive // 'else' block makes control flow more explicit here.
		// Otherwise, assert expected buffer contents.
		//
		// Split the lines we wrote.
		lines := strings.Split(buffer.String(), "\n")
		// Remove all empty lines.
		for i := range slices.Backward(lines) {
			if strings.TrimSpace(lines[i]) == "" {
				lines = slices.Delete(lines, i, i+1)
			}
		}

		// Assert the message first.
		require.Equal(t, linePrefix+" "+input, lines[0])

		// If we have no pairs we're done here.
		if len(pairs) == 0 {
			return
		}

		// Assert the pairs.
		require.Equal(t, len(pairs)/2+len(pairs)%2, len(lines[1:]))
		for i, line := range lines[1:] {
			// Since we're iterating by _line_, we need to '* 2' to get the actual
			// key index into 'pairs' since they're in... Pairs. xD
			keyIndex := i * 2
			valIndex := keyIndex + 1

			// Grab the key.
			key := fmt.Sprint(pairs[keyIndex])

			// Grab the value, using the default of 'valueMissing' if not present.
			val := valueMissing
			if valIndex < len(pairs) {
				val = fmt.Sprint(pairs[valIndex])
			}

			// Discern the expected box-drawing characters to start the row, depending
			// on whether we're at the final row.
			expectBox := rowMiddle
			if valIndex >= len(pairs)-1 {
				expectBox = rowBottom
			}

			// Format the logged value we expect.
			expect := fmt.Sprintf("%s[%s]=>[%s]", expectBox, key, val)

			// Zhu-li, do the thing!
			require.Equal(t, expect, line)
		}
	}
}

func TestLogf(t *testing.T) {
	buffer := setup(t)

	// Standard 'logf' cases.
	Level = LevelDebug
	expectLogf(t, buffer, LevelDebug, logfBasic, logfValue)
	expectLogf(t, buffer, LevelInfo, logfBasic, logfValue)
	expectLogf(t, buffer, LevelWarn, logfBasic, logfValue)
	expectLogf(t, buffer, LevelError, logfBasic, logfValue)

	// Verify output is suppressed at the appropriate log levels.
	//
	// Each of these tests should _not_ produce output.
	Level = LevelInfo
	expectLogf(t, buffer, LevelDebug, logfBasic, logfValue)
	Level = LevelWarn
	expectLogf(t, buffer, LevelInfo, logfBasic, logfValue)
	Level = LevelError
	expectLogf(t, buffer, LevelInfo, logfBasic, logfValue)
	Level = LevelFatal
	expectLogf(t, buffer, LevelError, logfBasic, logfValue)
}

func expectLogf(t *testing.T, buffer testBuffer, l level, input string, values ...any) {
	t.Helper()
	defer buffer.Reset()

	// Call the logging function.
	logf(l, defaultSkip, input, values...)

	if l < Level {
		// If the provided level is LOWER than the package level, we should expect
		// an empty buffer.
		require.Empty(t, buffer.String())
		return
	} else { //nolint:revive // 'else' block makes control flow more explicit here.
		// Produce the string, slicing off newlines.
		output := buffer.String()
		// Expect a trailing newline.
		require.True(t, strings.HasSuffix(output, "\n"))

		// Assert the output.
		//
		// Slice off the newline to simplify comparison below.
		output = strings.TrimRight(output, "\n")
		require.Equal(t, linePrefix+" "+fmt.Sprintf(input, values...), output)
	}
}

func TestWriteLevel(t *testing.T) {
	buffer := setup(t)

	// Disable log level output.
	Level = LevelDebug
	Options = 0

	// Verify the standard prefix ('>') is used instead of log level when the
	// appropriate option is disabled.
	require.Equal(t, "> ", doWriteLevel(t, buffer, LevelDebug))
	require.Equal(t, "> ", doWriteLevel(t, buffer, LevelWarn))
	require.Equal(t, "> ", doWriteLevel(t, buffer, LevelInfo))
	require.Equal(t, "> ", doWriteLevel(t, buffer, LevelError))

	// Enable log level output.
	Options = OptWithLevel

	// Verify expected log level output for each of these (meaning log level
	// strings _are_ written).
	Level = LevelDebug
	require.Equal(t, LevelDebug.String()+" ", doWriteLevel(t, buffer, LevelDebug))
	require.Equal(t, LevelInfo.String()+" ", doWriteLevel(t, buffer, LevelInfo))
	require.Equal(t, LevelWarn.String()+" ", doWriteLevel(t, buffer, LevelWarn))
	require.Equal(t, LevelError.String()+" ", doWriteLevel(t, buffer, LevelError))

	// Verify output is suppressed at the appropriate log levels.
	//
	// Debug
	Level = LevelDebug
	require.NotEmpty(t, doWriteLevel(t, buffer, LevelDebug))
	Level = LevelInfo
	require.Empty(t, doWriteLevel(t, buffer, LevelDebug))
	// Info
	require.NotEmpty(t, doWriteLevel(t, buffer, LevelInfo))
	Level = LevelWarn
	require.Empty(t, doWriteLevel(t, buffer, LevelInfo))
	// Warn
	require.NotEmpty(t, doWriteLevel(t, buffer, LevelWarn))
	Level = LevelError
	require.Empty(t, doWriteLevel(t, buffer, LevelWarn))
	// Error
	require.NotEmpty(t, doWriteLevel(t, buffer, LevelError))
	Level = LevelFatal
	require.Empty(t, doWriteLevel(t, buffer, LevelError))
}

func doWriteLevel(t *testing.T, buffer testBuffer, l level) string {
	t.Helper()
	defer buffer.Reset()

	writeLevel(buffer, l)

	// Stringify the result.
	return buffer.String()
}

func TestWriteCaller(t *testing.T) {
	buffer := setup(t).(*strings.Builder)

	// Expect no output if the 'WithCaller' option is not set.
	Options = 0
	expectCaller(t, buffer, "^$")

	// Expect a caller file + line number with the option set.
	Options = OptWithCaller
	expectCaller(t, buffer, fmt.Sprintf(`^\[%s:\d{1,3}\]`, callerFile))
}

// NOTE: If this _file_ ever gets renamed we'll need to update it here so this
// test passes!
const callerFile = "log_test.go"

func expectCaller(t *testing.T, buffer testBuffer, pattern string) {
	t.Helper()

	// Compile the provided pattern.
	r := regexp.MustCompile(pattern)

	// Write the caller info.
	writeCaller(buffer, 1)

	// Zhu-li, do the thing!
	require.True(t, r.MatchString(buffer.String()), "%s", buffer)
}

type testBuffer interface {
	io.Writer
	String() string
	Reset()
}
