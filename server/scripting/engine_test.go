package scripting

import "testing"

func TestRunScript(t *testing.T) {
	e := NewEngine()

	e.Execute(`
	print("hello world")
`)
}
