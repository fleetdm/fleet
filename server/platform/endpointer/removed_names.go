package endpointer

import (
	"bytes"
	"encoding"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
)

// The removed names scan rejects a request that sends a name Fleet 5 removed or renamed, such as team_id, instead of silently
// ignoring it. It is driven by one list of names (SetRemovedNames) and by the Go type the request decodes into: a name is only
// rejected where that type has no field of that name. That gives the scan three properties without any per-field tags:
//   - a type that still declares the old name (a surface not renamed yet) keeps accepting it, so surfaces switch over one by one;
//   - free-form JSON (maps, any, json.RawMessage, types with their own UnmarshalJSON) is never looked into, so a profile or agent
//     option that happens to contain "team" is not rejected;
//   - a request type that declares nothing about a removed name needs no change when a new endpoint is added.

type removedNames struct {
	jsonKeys    map[string]struct{} // lower case, JSON member names match case-insensitively
	queryParams map[string]struct{} // exact, query parameter names are case-sensitive
	// handParsed holds the `url` tag values whose parsers read query parameters by name in code, where no tag declares them.
	handParsed map[string]struct{}
}

var removedNamesList atomic.Pointer[removedNames]

// SetRemovedNames sets the request names the scan rejects. The platform layer has no list of its own: the service sets it from
// pkg/removednames, as it does for the removed key message. A request type with a field tagged `url:"<one of handParsedURLTags>"` is
// left out of the query check, since its parser reads parameters the scan cannot see; each one drops out of that list as its
// parser stops reading the removed name.
func SetRemovedNames(jsonKeys, queryParams, handParsedURLTags []string) {
	names := &removedNames{
		jsonKeys:    make(map[string]struct{}, len(jsonKeys)),
		queryParams: make(map[string]struct{}, len(queryParams)),
		handParsed:  make(map[string]struct{}, len(handParsedURLTags)),
	}
	for _, tag := range handParsedURLTags {
		names.handParsed[tag] = struct{}{}
	}
	for _, k := range jsonKeys {
		names.jsonKeys[strings.ToLower(k)] = struct{}{}
	}
	for _, p := range queryParams {
		names.queryParams[p] = struct{}{}
	}
	removedNamesList.Store(names)
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
		if !opaque {
			switch t.Kind() {
			case reflect.Struct:
				return scanObject(dec, jsonFields(t), removed)
			case reflect.Map:
				return scanMap(dec, t.Elem(), removed)
			}
		}
	case '[':
		if !opaque && (t.Kind() == reflect.Slice || t.Kind() == reflect.Array) {
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

// scanMap looks into the values of a map, whose keys are data and not field names.
func scanMap(dec *jsontext.Decoder, elem reflect.Type, removed map[string]struct{}) error {
	if _, err := dec.ReadToken(); err != nil { // {
		return err
	}
	for dec.PeekKind() != '}' {
		if _, err := dec.ReadToken(); err != nil { // the key
			return err
		}
		if err := scanValue(dec, elem, removed); err != nil {
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
	unmarshalerTypes = []reflect.Type{
		reflect.TypeFor[json.Unmarshaler](),
		reflect.TypeFor[jsonv2.UnmarshalerFrom](),
		reflect.TypeFor[encoding.TextUnmarshaler](),
	}
	jsonFieldsCache sync.Map // reflect.Type → map[string]reflect.Type
)

// isOpaque reports whether t decodes itself, so its members are not struct fields the scan can see.
func isOpaque(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for _, u := range unmarshalerTypes {
		if t.Implements(u) || reflect.PointerTo(t).Implements(u) {
			return true
		}
	}
	return false
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
			if _, dup := fields[strings.ToLower(name)]; !dup {
				fields[strings.ToLower(name)] = sf.Type
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
	declared, handParsed := declaredQueryNames(t, names.handParsed)
	if handParsed {
		return nil
	}
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

type queryNames struct {
	declared   map[string]struct{}
	handParsed bool
}

// declaredQueryNames returns the parameters t declares in `query` tags, and whether t has a hand-parsed field.
func declaredQueryNames(t reflect.Type, handParsed map[string]struct{}) (map[string]struct{}, bool) {
	if cached, ok := queryNamesCache.Load(t); ok {
		n := cached.(queryNames)
		return n.declared, n.handParsed
	}
	n := queryNames{declared: make(map[string]struct{})}
	visited := make(map[reflect.Type]struct{})
	var walk func(t reflect.Type)
	walk = func(t reflect.Type) {
		if _, ok := visited[t]; ok {
			return
		}
		visited[t] = struct{}{}
		for sf := range t.Fields() {
			if tag, ok := sf.Tag.Lookup("query"); ok {
				name, _, _ := strings.Cut(tag, ",")
				n.declared[name] = struct{}{}
			}
			if _, ok := handParsed[sf.Tag.Get("url")]; ok {
				n.handParsed = true
			}
			if ft := indirectType(sf.Type); ft.Kind() == reflect.Struct && sf.Anonymous {
				walk(ft)
			}
		}
	}
	walk(indirectType(t))
	queryNamesCache.Store(t, n)
	return n.declared, n.handParsed
}
