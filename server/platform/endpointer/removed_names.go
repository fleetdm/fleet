package endpointer

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
)

// The removed names scan rejects a request that sends a name Fleet 5 removed or renamed, such as team_id. It is driven by one
// list of names (SetRemovedNames) and by the Go type the request decodes into: a name is only rejected where that type has no
// field of that name. That gives the scan three properties without any per-field tags:
//   - a type that still declares the old name (a surface not renamed yet) keeps accepting it, so surfaces switch over one by one;
//   - free-form JSON (maps, any, json.RawMessage, types with their own UnmarshalJSON) is never looked into, so a profile or agent
//     option that happens to contain "team" is not rejected;
//   - a request type that declares nothing about a removed name needs no change when a new endpoint is added.

type removedNames struct {
	jsonKeys    map[string]struct{} // lower case, JSON member names match case-insensitively
	queryParams map[string]struct{} // exact, query parameter names are case-sensitive
	sortKeys    map[string]struct{} // exact values of the order_key and inherited_order_key parameters
	// skipPaths are request path segments, such as "/orbit/", whose requests the scan leaves alone.
	skipPaths []string
}

// maxRemovedNamesPresize caps how much buffer a request's Content-Length can reserve before any of its body has arrived.
const maxRemovedNamesPresize = 64 << 10

// maxRemovedNamesScanSize is the largest body the scan buffers and checks, the default request limit. A larger body, which only
// the batch and spec routes allow, is streamed to the decoder without a scan, so these routes do not hold a whole body in memory.
const maxRemovedNamesScanSize = 1 << 20

var removedNamesList atomic.Pointer[removedNames]

// SetRemovedNames sets the request names the scan rejects.
func SetRemovedNames(jsonKeys, queryParams, sortKeys, skipPaths []string) {
	names := &removedNames{
		jsonKeys:    make(map[string]struct{}, len(jsonKeys)),
		queryParams: make(map[string]struct{}, len(queryParams)),
		sortKeys:    make(map[string]struct{}, len(sortKeys)),
		skipPaths:   skipPaths,
	}
	for _, k := range jsonKeys {
		names.jsonKeys[strings.ToLower(k)] = struct{}{}
	}
	for _, p := range queryParams {
		names.queryParams[p] = struct{}{}
	}
	for _, k := range sortKeys {
		names.sortKeys[k] = struct{}{}
	}
	removedNamesList.Store(names)
}

// SkipsRemovedNamesScan reports whether the request path is one of the skip paths.
func SkipsRemovedNamesScan(path string) bool {
	names := removedNamesList.Load()
	if names == nil {
		return false
	}
	for _, segment := range names.skipPaths {
		if strings.Contains(path, segment) {
			return true
		}
	}
	return false
}

// RejectRemovedJSONKeys returns a *RemovedKeyError if the JSON document data uses a removed name as the key of a field its
// destination type t does not declare. Input it cannot parse is not an error here: the decoder reports that.
func RejectRemovedJSONKeys(data []byte, t reflect.Type) error {
	names := removedNamesList.Load()
	if names == nil || len(names.jsonKeys) == 0 {
		return nil
	}
	dec := jsontext.NewDecoder(bytes.NewReader(data), jsontext.AllowDuplicateNames(true), jsontext.AllowInvalidUTF8(true))
	if rke, ok := errors.AsType[*RemovedKeyError](scanValue(dec, t, names.jsonKeys)); ok {
		return rke
	}
	return nil
}

func scanValue(dec *jsontext.Decoder, t reflect.Type, removed map[string]struct{}) error {
	opaque := isOpaque(t)
	t = indirectType(t)
	switch dec.PeekKind() {
	case '{':
		if !opaque && t.Kind() == reflect.Struct {
			return scanObject(dec, jsonFields(t), removed)
		}
	case '[':
		if !opaque && t.Kind() == reflect.Slice {
			return scanArray(dec, t.Elem(), removed)
		}
	}
	return dec.SkipValue()
}

func scanObject(dec *jsontext.Decoder, fields map[string]reflect.Type, removed map[string]struct{}) error {
	if _, err := dec.ReadToken(); err != nil { // {
		return err
	}
	for dec.PeekKind() != '}' {
		key, err := dec.ReadToken()
		if err != nil {
			return err
		}
		name := strings.ToLower(key.String())
		if ft, declared := fields[name]; declared {
			err = scanValue(dec, ft, removed)
		} else if _, isRemoved := removed[name]; isRemoved {
			return &RemovedKeyError{Key: key.String()}
		} else {
			err = dec.SkipValue()
		}
		if err != nil {
			return err
		}
	}
	_, err := dec.ReadToken() // }
	return err
}

func scanArray(dec *jsontext.Decoder, elem reflect.Type, removed map[string]struct{}) error {
	if _, err := dec.ReadToken(); err != nil { // [
		return err
	}
	for dec.PeekKind() != ']' {
		if err := scanValue(dec, elem, removed); err != nil {
			return err
		}
	}
	_, err := dec.ReadToken() // ]
	return err
}

var (
	unmarshalerType = reflect.TypeFor[json.Unmarshaler]()
	jsonFieldsCache sync.Map // reflect.Type → map[string]reflect.Type
)

// isOpaque reports whether t decodes itself, so its members are not struct fields the scan can see.
func isOpaque(t reflect.Type) bool {
	return reflect.PointerTo(indirectType(t)).Implements(unmarshalerType)
}

func indirectType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// jsonFields maps the lower-cased JSON name of every field encoding/json decodes into t, including the fields of embedded structs,
// to the field's type.
func jsonFields(t reflect.Type) map[string]reflect.Type {
	if cached, ok := jsonFieldsCache.Load(t); ok {
		return cached.(map[string]reflect.Type)
	}
	fields := make(map[string]reflect.Type)
	visited := make(map[reflect.Type]struct{})
	var walk func(t reflect.Type)
	walk = func(t reflect.Type) {
		if _, ok := visited[t]; ok {
			return
		}
		visited[t] = struct{}{}
		for sf := range t.Fields() {
			tag := sf.Tag.Get("json")
			name, _, _ := strings.Cut(tag, ",")
			if tag == "-" {
				continue
			}
			if sf.Anonymous && name == "" {
				if ft := indirectType(sf.Type); ft.Kind() == reflect.Struct {
					walk(ft)
					continue
				}
			}
			if !sf.IsExported() {
				continue
			}
			if name == "" {
				name = sf.Name
			}
			// The scan runs before the rewriter, so a field can arrive under its renameto name too.
			for _, n := range []string{name, sf.Tag.Get("renameto")} {
				if n != "" {
					fields[strings.ToLower(n)] = sf.Type
				}
			}
		}
	}
	walk(t)
	jsonFieldsCache.Store(t, fields)
	return fields
}

var queryNamesCache sync.Map // reflect.Type → map[string]struct{}

// RejectRemovedQueryParams returns a *RemovedKeyError if the query string uses a removed parameter name that the request type t
// does not declare in a `query` tag, or sorts by a removed order_key.
func RejectRemovedQueryParams(query url.Values, t reflect.Type) error {
	names := removedNamesList.Load()
	if names == nil || len(query) == 0 {
		return nil
	}
	if !sortsByAnyColumn(t) {
		for _, param := range []string{"order_key", "inherited_order_key"} {
			for _, value := range query[param] {
				if _, isRemoved := names.sortKeys[value]; isRemoved {
					return &RemovedKeyError{Key: value, Param: param}
				}
			}
		}
	}
	declared := declaredQueryNames(t)
	for param := range query {
		if _, isRemoved := names.queryParams[param]; !isRemoved {
			continue
		}
		if _, ok := declared[param]; !ok {
			return &RemovedKeyError{Key: param}
		}
	}
	return nil
}

// anyColumnSorter is implemented by request types whose order_key names a result column chosen by the user, which may be
// called team_id.
type anyColumnSorter interface{ SortsByAnyColumn() }

var anyColumnSorterType = reflect.TypeFor[anyColumnSorter]()

func sortsByAnyColumn(t reflect.Type) bool {
	return reflect.PointerTo(t).Implements(anyColumnSorterType)
}

// declaredQueryNames returns the parameters t declares in `query` tags.
func declaredQueryNames(t reflect.Type) map[string]struct{} {
	if cached, ok := queryNamesCache.Load(t); ok {
		return cached.(map[string]struct{})
	}
	declared := make(map[string]struct{})
	collectQueryNames(t, declared)
	queryNamesCache.Store(t, declared)
	return declared
}

// collectQueryNames follows embedded structs by value only, like the decoder's allFields.
func collectQueryNames(t reflect.Type, declared map[string]struct{}) {
	for sf := range t.Fields() {
		if tag, ok := sf.Tag.Lookup("query"); ok {
			name, _, _ := strings.Cut(tag, ",")
			declared[name] = struct{}{}
		}
		if sf.Anonymous && sf.Type.Kind() == reflect.Struct {
			collectQueryNames(sf.Type, declared)
		}
	}
}
