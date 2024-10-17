package test

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

type TestingT interface {
	assert.TestingT
	Helper()
}

// ElementsMatchSkipID asserts that the elements match, skipping any field with
// name "ID".
func ElementsMatchSkipID(t TestingT, listA, listB interface{}, msgAndArgs ...interface{}) (ok bool) {
	t.Helper()

	opt := cmp.FilterPath(func(p cmp.Path) bool {
		for _, ps := range p {
			if ps, ok := ps.(cmp.StructField); ok && strings.HasSuffix(ps.Name(), "ID") {
				return true
			}
		}
		return false
	}, cmp.Ignore())
	return ElementsMatchWithOptions(t, listA, listB, []cmp.Option{opt}, msgAndArgs)
}

// ElementsMatchSkipIDAndHostCount asserts that the elements match, skipping any field with
// name "ID" or "HostCount".
func ElementsMatchSkipIDAndHostCount(t TestingT, listA, listB interface{}, msgAndArgs ...interface{}) (ok bool) {
	t.Helper()

	opt := cmp.FilterPath(func(p cmp.Path) bool {
		for _, ps := range p {
			if ps, ok := ps.(cmp.StructField); ok && (ps.Name() == "ID" || ps.Name() == "HostCount") {
				return true
			}
		}
		return false
	}, cmp.Ignore())
	return ElementsMatchWithOptions(t, listA, listB, []cmp.Option{opt}, msgAndArgs)
}

// ElementsMatchSkipTimestampsID asserts that the elements match, skipping any field with
// name "ID", "CreatedAt", and "UpdatedAt". This is useful for comparing after DB insertion.
func ElementsMatchSkipTimestampsID(t TestingT, listA, listB interface{}, msgAndArgs ...interface{}) (ok bool) {
	t.Helper()

	opt := cmp.FilterPath(func(p cmp.Path) bool {
		for _, ps := range p {
			if ps, ok := ps.(cmp.StructField); ok {
				switch ps.Name() { //nolint:gocritic // ignore singleCaseSwitch
				case "ID", "UpdateCreateTimestamps", "CreateTimestamp", "UpdateTimestamp", "CreatedAt", "UpdatedAt":
					return true
				}
			}
		}
		return false
	}, cmp.Ignore())
	return ElementsMatchWithOptions(t, listA, listB, []cmp.Option{opt}, msgAndArgs)
}

// EqualSkipTimestampsID asserts that the structs are equal, skipping any field
// with name "ID", "CreatedAt", and "UpdatedAt". This is useful for comparing
// after DB insertion.
func EqualSkipTimestampsID(t TestingT, a, b interface{}, msgAndArgs ...interface{}) (ok bool) {
	t.Helper()

	opt := cmp.FilterPath(func(p cmp.Path) bool {
		for _, ps := range p {
			if ps, ok := ps.(cmp.StructField); ok {
				switch ps.Name() { //nolint:gocritic // ignore singleCaseSwitch
				case "ID", "UpdateCreateTimestamps", "CreateTimestamp", "UpdateTimestamp", "CreatedAt", "UpdatedAt":
					return true
				}
			}
		}
		return false
	}, cmp.Ignore())

	if !cmp.Equal(a, b, opt) {
		return assert.Fail(t, cmp.Diff(a, b, opt), msgAndArgs...)
	}
	return true
}

// The below functions adapted from
// https://github.com/stretchr/testify/blob/v1.7.0/assert/assertions.go#L895 by
// utilizing the options provided in github.com/google/go-cmp/cmp

// ElementsMatchWithOptions wraps the assert.ElementsMatch function with
// additional options as provided by the cmp package. This allows, for example,
// comparing structs while ignoring fields. See assert.ElementsMatch
// documentation for more details.
func ElementsMatchWithOptions(t TestingT, listA, listB interface{}, opts cmp.Options, msgAndArgs ...interface{}) (ok bool) {
	if isEmpty(listA) && isEmpty(listB) {
		return true
	}

	if !isList(t, listA, msgAndArgs...) || !isList(t, listB, msgAndArgs...) {
		return false
	}

	extraA, extraB := diffLists(listA, listB, opts)

	if len(extraA) == 0 && len(extraB) == 0 {
		return true
	}

	return assert.Fail(t, formatListDiff(listA, listB, extraA, extraB), msgAndArgs...)
}

// isEmpty gets whether the specified object is considered empty or not.
func isEmpty(object interface{}) bool {
	// get nil case out of the way
	if object == nil {
		return true
	}

	objValue := reflect.ValueOf(object)

	switch objValue.Kind() {
	// collection types are empty when they have no element
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		return objValue.Len() == 0
		// pointers are empty if nil or if the value they point to is empty
	case reflect.Ptr:
		if objValue.IsNil() {
			return true
		}
		deref := objValue.Elem().Interface()
		return isEmpty(deref)
		// for all other types, compare against the zero value
	default:
		zero := reflect.Zero(objValue.Type())
		return reflect.DeepEqual(object, zero.Interface())
	}
}

// isList checks that the provided value is array or slice.
func isList(t TestingT, list interface{}, msgAndArgs ...interface{}) (ok bool) {
	kind := reflect.TypeOf(list).Kind()
	if kind != reflect.Array && kind != reflect.Slice {
		return assert.Fail(t, fmt.Sprintf("%q has an unsupported type %s, expecting array or slice", list, kind),
			msgAndArgs...)
	}
	return true
}

// diffLists diffs two arrays/slices and returns slices of elements that are only in A and only in B.
// If some element is present multiple times, each instance is counted separately (e.g. if something is 2x in A and
// 5x in B, it will be 0x in extraA and 3x in extraB). The order of items in both lists is ignored.
func diffLists(listA, listB interface{}, opts cmp.Options) (extraA, extraB []interface{}) {
	aValue := reflect.ValueOf(listA)
	bValue := reflect.ValueOf(listB)

	aLen := aValue.Len()
	bLen := bValue.Len()

	// Mark indexes in bValue that we already used
	visited := make([]bool, bLen)
	for i := 0; i < aLen; i++ {
		element := aValue.Index(i).Interface()
		found := false
		for j := 0; j < bLen; j++ {
			if visited[j] {
				continue
			}
			if cmp.Equal(bValue.Index(j).Interface(), element, opts...) {
				visited[j] = true
				found = true
				break
			}
		}
		if !found {
			extraA = append(extraA, element)
		}
	}

	for j := 0; j < bLen; j++ {
		if visited[j] {
			continue
		}
		extraB = append(extraB, bValue.Index(j).Interface())
	}

	return
}

func formatListDiff(listA, listB interface{}, extraA, extraB []interface{}) string {
	spewConfig := spew.ConfigState{
		Indent:                  " ",
		DisablePointerAddresses: true,
		DisableCapacities:       true,
		SortKeys:                true,
		DisableMethods:          true,
		MaxDepth:                10,
	}

	var msg bytes.Buffer

	msg.WriteString("elements differ")
	if len(extraA) > 0 {
		msg.WriteString("\n\nextra elements in list A:\n")
		msg.WriteString(spewConfig.Sdump(extraA))
	}
	if len(extraB) > 0 {
		msg.WriteString("\n\nextra elements in list B:\n")
		msg.WriteString(spewConfig.Sdump(extraB))
	}
	msg.WriteString("\n\nlistA:\n")
	msg.WriteString(spewConfig.Sdump(listA))
	msg.WriteString("\n\nlistB:\n")
	msg.WriteString(spewConfig.Sdump(listB))

	return msg.String()
}

// QueryElementsMatch asserts that two queries slices match
func QueryElementsMatch(t TestingT, listA, listB interface{}, msgAndArgs ...interface{}) (ok bool) {
	t.Helper()

	opt := cmp.FilterPath(func(p cmp.Path) bool {
		for _, ps := range p {
			if ps, ok := ps.(cmp.StructField); ok {
				switch ps.Name() { //nolint:gocritic // ignore singleCaseSwitch
				case "ID",
					"UpdateCreateTimestamps",
					"AuthorID",
					"AuthorName",
					"AuthorEmail",
					"Packs",
					"Saved":
					return true
				}
			}
		}
		return false
	}, cmp.Ignore())
	return ElementsMatchWithOptions(t, listA, listB, []cmp.Option{opt}, msgAndArgs)
}

// QueriesMatch asserts that two queries 'match'.
func QueriesMatch(t TestingT, a, b interface{}, msgAndArgs ...interface{}) (ok bool) {
	t.Helper()

	opt := cmp.FilterPath(func(p cmp.Path) bool {
		for _, ps := range p {
			if ps, ok := ps.(cmp.StructField); ok {
				switch ps.Name() { //nolint:gocritic // ignore singleCaseSwitch
				case "ID",
					"UpdateCreateTimestamps",
					"AuthorID",
					"AuthorName",
					"AuthorEmail",
					"Packs",
					"Saved":
					return true
				}
			}
		}
		return false
	}, cmp.Ignore())

	if !cmp.Equal(a, b, opt) {
		return assert.Fail(t, cmp.Diff(a, b, opt), msgAndArgs...)
	}
	return true
}
