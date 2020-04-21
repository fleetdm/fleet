package test

import (
	"reflect"
	"runtime"
	"strings"
)

// FunctionName returns the name of the function provided as the argument.
// Behavior is undefined if a non-function is passed.
func FunctionName(f interface{}) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	elements := strings.Split(fullName, ".")
	return elements[len(elements)-1]
}
