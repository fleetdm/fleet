package scripting

import "github.com/Shopify/go-lua"

type ScriptEngine struct {
	l *lua.State
}

func NewEngine() *ScriptEngine {
	l := lua.NewState()
	lua.Require(l)
	lua.OpenLibraries(l)
	return &ScriptEngine{l}
}

func (e *ScriptEngine) Execute(script string) error {
	return lua.DoString(e.l, script)
}
