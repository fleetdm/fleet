package log

import "testing"

func TestLog(t *testing.T) {
	Options.SetWithCaller()
	Options.SetWithLevel()
	Info("Hello, world!", "key", "value")
}
