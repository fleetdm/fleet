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
	// handParsed maps a `url` tag value to the query parameters its parser reads by name in code, where no tag declares them.
	handParsed map[string][]string
	// skipPaths are request path segments, such as "/orbit/", whose requests the scan leaves alone.
	skipPaths []string
}

// maxRemovedNamesPresize caps how much buffer a request's Content-Length can reserve before any of its body has arrived.
const maxRemovedNamesPresize = 64 << 10

var removedNamesList atomic.Pointer[removedNames]

// SetRemovedNames sets the request names the scan rejects.
func SetRemovedNames(jsonKeys, queryParams []string, handParsed map[string][]string, skipPaths []string) {
	names := &removedNames{
		jsonKeys:    make(map[string]struct{}, len(jsonKeys)),
		queryParams: make(map[string]struct{}, len(queryParams)),
		handParsed:  handParsed,
		skipPaths:   skipPaths,
	}
	for _, k := range jsonKeys {
		names.jsonKeys[strings.ToLower(k)] = struct{}{}
	}
	for _, p := range queryParams {
		names.queryParams[p] = struct{}{}
	}
	removedNamesList.Store(names)
	queryNamesCache.Clear() // the declared names include the hand-parsed ones
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
// does not declare in a `query` tag.
func RejectRemovedQueryParams(query url.Values, t reflect.Type) error {
	names := removedNamesList.Load()
	if names == nil || len(names.queryParams) == 0 || len(query) == 0 {
		return nil
	}
	declared := declaredQueryNames(t, names.handParsed)
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

// declaredQueryNames returns the parameters t declares in `query` tags, plus those its hand-parsed fields read.
func declaredQueryNames(t reflect.Type, handParsed map[string][]string) map[string]struct{} {
	if cached, ok := queryNamesCache.Load(t); ok {
		return cached.(map[string]struct{})
	}
	declared := make(map[string]struct{})
	collectQueryNames(t, handParsed, declared)
	queryNamesCache.Store(t, declared)
	return declared
}

// collectQueryNames follows embedded structs by value only, like the decoder's allFields.
func collectQueryNames(t reflect.Type, handParsed map[string][]string, declared map[string]struct{}) {
	for sf := range t.Fields() {
		if tag, ok := sf.Tag.Lookup("query"); ok {
			name, _, _ := strings.Cut(tag, ",")
			declared[name] = struct{}{}
		}
		urlName, _, _ := strings.Cut(sf.Tag.Get("url"), ",")
		for _, param := range handParsed[urlName] {
			declared[param] = struct{}{}
		}
		if sf.Anonymous && sf.Type.Kind() == reflect.Struct {
			collectQueryNames(sf.Type, handParsed, declared)
		}
	}
}
