package scripting

import (
	"context"

	"github.com/Shopify/go-lua"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
)

type ScriptEngine struct {
	l *lua.State
}

func fleetFunction(reader mysql.DBReader) lua.Function {
	return func(l *lua.State) int {
		lua.NewLibrary(l, []lua.RegistryFunction{
			{"db", func(l *lua.State) int {
				query := lua.CheckString(l, 1)
				i := 2
				var args []interface{}
				for !l.IsNone(i) {
					args = append(args, l.ToValue(i))
					i++
				}

				rows, err := reader.QueryContext(context.Background(), query, args...)
				if err != nil {
					lua.Errorf(l, err.Error())
				}
				defer rows.Close()

				var res []map[string]interface{}
				for rows.Next() {
					row := make(map[string]interface{})
					err := rows.Scan(row)
					if err != nil {
						lua.Errorf(l, err.Error())
					}
					res = append(res, row)
				}
				l.PushUserData()
				return 1
			}},
		})
		return 1
	}
}

func NewEngine(reader mysql.DBReader) *ScriptEngine {
	l := lua.NewState()
	// TODO: OpenLibraries registers everything, too much, just add the few that we need
	lua.OpenLibraries(l)
	lua.Require(l, "fleet", fleetFunction(reader), true)
	return &ScriptEngine{l}
}

func (e *ScriptEngine) Execute(script string) error {
	return lua.DoString(e.l, script)
}
