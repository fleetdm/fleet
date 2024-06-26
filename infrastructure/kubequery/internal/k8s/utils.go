/**
 * Copyright (c) 2020-present, The kubequery authors
 *
 * This source code is licensed as defined by the LICENSE file found in the
 * root directory of this source tree.
 *
 * SPDX-License-Identifier: (Apache-2.0 OR GPL-2.0-only)
 */

package k8s

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/Uptycs/basequery-go/plugin/table"
	"github.com/iancoleman/strcase"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var replacements = map[string]string{
	"IPs":   "Ips",
	"URLs":  "Urls",
	"CIDRs": "Cidrs",
	"WWIDs": "WwIds",
	"WWNs":  "WwNs",
}

func makeKey(name string) string {
	for k, v := range replacements {
		name = strings.Replace(name, k, v, 1)
	}
	return strcase.ToSnake(name)
}

func getFieldValue(field reflect.Value) string {
	tp := field.Type()
	kind := tp.Kind()

	if kind == reflect.Ptr {
		if field.IsNil() {
			return ""
		}

		tp = field.Type().Elem()
		kind = tp.Kind()
		field = field.Elem()
	}

	if tp.PkgPath() == "k8s.io/apimachinery/pkg/apis/meta/v1" && tp.Name() == "Time" {
		i := field.Interface()
		if i.(metav1.Time).UTC().IsZero() {
			return "0"
		}
		return strconv.FormatInt(i.(metav1.Time).Unix(), 10)
	}

	switch kind {
	case reflect.Map, reflect.Slice:
		if !field.IsNil() {
			bytes, _ := json.Marshal(field.Interface())
			if bytes != nil {
				return string(bytes)
			}
		}
	case reflect.Struct:
		bytes, _ := json.Marshal(field.Interface())
		if bytes != nil {
			return string(bytes)
		}
	case reflect.String:
		return string(field.String())
	case reflect.Bool:
		if field.Bool() {
			return "1"
		}
		return "0"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(field.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(field.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%f", field.Float())
	default:
		panic(fmt.Sprintf("Type not supported: %s", kind))
	}

	return ""
}

// ToMap returns object fields as key/value map. Field names are converted to snake case.
// Values are converted to string. Complex value types like structures are serialized as JSON.
func ToMap(obj interface{}) map[string]string {
	item := make(map[string]string)
	val := reflect.ValueOf(obj)
	if kind := val.Kind(); kind == reflect.Interface || kind == reflect.Ptr {
		val = val.Elem()
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		name := val.Type().Field(i).Name

		if val.Type().Field(i).Anonymous {
			if field.Type().Kind() == reflect.Ptr {
				panic(fmt.Sprintf("Embedded pointer to a struct not supported: %s", name))
			}

			for k, v := range ToMap(field.Interface()) {
				item[k] = v
			}
		} else {
			key := makeKey(name)
			str := getFieldValue(val.Field(i))
			if str != "" {
				item[key] = str
			}
		}
	}

	return item
}

func getFieldSchema(name string, field reflect.Value) table.ColumnDefinition {
	tp := field.Type()
	kind := tp.Kind()
	key := makeKey(name)

	if kind == reflect.Ptr {
		tp = field.Type().Elem()
		kind = tp.Kind()
	}
	if tp.PkgPath() == "k8s.io/apimachinery/pkg/apis/meta/v1" && tp.Name() == "Time" {
		return table.BigIntColumn(key)
	}

	switch kind {
	case reflect.Map, reflect.Slice, reflect.Struct, reflect.String:
		return table.TextColumn(key)
	case reflect.Float32, reflect.Float64:
		return table.DoubleColumn(key)
	case reflect.Int64, reflect.Uint64:
		return table.BigIntColumn(key)
	case reflect.Bool, reflect.Int, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint16, reflect.Uint32:
		return table.IntegerColumn(key)
	default:
		panic(fmt.Sprintf("Type not supported: %s", kind))
	}
}

// GetSchema takes a object and returns Osquery table column definitions.
// Object field names are converted to snake case.
// The object fields including anonymous ones are identified appropriate column definitions are identified.
func GetSchema(obj interface{}) []table.ColumnDefinition {
	schema := make([]table.ColumnDefinition, 0)
	val := reflect.ValueOf(obj)
	if kind := val.Kind(); kind == reflect.Interface || kind == reflect.Ptr {
		val = val.Elem()
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		name := val.Type().Field(i).Name

		if val.Type().Field(i).Anonymous {
			if field.Type().Kind() == reflect.Ptr {
				panic(fmt.Sprintf("Embedded pointer to a struct not supported: %s", name))
			}

			s := GetSchema(field.Interface())
			schema = append(schema, s...)
		} else {
			schema = append(schema, getFieldSchema(name, field))
		}
	}

	return schema
}
