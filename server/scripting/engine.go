package scripting

import (
	"bytes"
	"context"
	"fmt"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
)

func Execute(ctx context.Context, script string, reader mysql.DBReader) (string, error) {
	scr := tengo.NewScript([]byte(script))

	stdio := new(bytes.Buffer)

	moduleMap := tengo.NewModuleMap()
	moduleMap.AddMap(stdlib.GetModuleMap(stdlib.AllModuleNames()...))
	moduleMap.AddBuiltinModule("db", map[string]tengo.Object{
		"select": &tengo.UserFunction{
			Name: "select",
			Value: func(args ...tengo.Object) (ret tengo.Object, err error) {
				if len(args) < 1 {
					return nil, tengo.ErrWrongNumArguments
				}

				// TODO: make this a safer conversion
				query := args[0].(*tengo.String).Value
				var queryArgs []interface{}
				for _, arg := range args[1:] {
					queryArgs = append(queryArgs, tengo.ToInterface(arg))
				}

				rows, err := reader.QueryContext(ctx, query, queryArgs...)
				if err != nil {
					return &tengo.Error{Value: &tengo.String{Value: err.Error()}}, nil
				}
				defer rows.Close()

				cols, err := rows.Columns()
				if err != nil {
					return &tengo.Error{Value: &tengo.String{Value: err.Error()}}, nil
				}
				var res []interface{}
				for rows.Next() {
					columns := make([]string, len(cols))
					columnPointers := make([]interface{}, len(cols))
					for i := range columns {
						columnPointers[i] = &columns[i]
					}

					err = rows.Scan(columnPointers...)
					if err != nil {
						return &tengo.Error{Value: &tengo.String{Value: err.Error()}}, nil
					}
					mapRow := make(map[string]interface{})
					for i, col := range cols {
						val := columnPointers[i].(*string)
						mapRow[col] = *val
					}
					res = append(res, mapRow)
				}
				o, err := tengo.FromInterface(res)
				if err != nil {
					return &tengo.Error{
						Value: &tengo.String{Value: err.Error()}}, nil
				}
				return o, nil
			},
		},
	})
	scr.SetImports(moduleMap)

	err := scr.Add("printf", &tengo.UserFunction{Value: func(args ...tengo.Object) (ret tengo.Object, err error) {
		// TODO: make this a safer conversion
		formatString := args[0].(*tengo.String).Value
		var interfaceArgs []interface{}
		for _, arg := range args[1:] {
			interfaceArgs = append(interfaceArgs, tengo.ToInterface(arg))
		}
		res := fmt.Sprintf(formatString, interfaceArgs...)
		stdio.WriteString(res)
		return nil, nil
	}})
	if err != nil {
		return "", err
	}
	err = scr.Add("println", &tengo.UserFunction{Value: func(args ...tengo.Object) (ret tengo.Object, err error) {
		// TODO: make this a safer conversion
		var interfaceArgs []interface{}
		for _, arg := range args {
			interfaceArgs = append(interfaceArgs, tengo.ToInterface(arg))
		}
		res := fmt.Sprintln(interfaceArgs...)
		stdio.WriteString(res)
		return nil, nil
	}})
	if err != nil {
		return "", err
	}

	_, err = scr.RunContext(ctx)
	return stdio.String(), err
}
