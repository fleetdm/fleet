// Package dataflatten contains tools to flatten complex data
// structures.
//
// On macOS, many plists use an array of maps, these can be tricky to
// filter. This package knows how to flatten that structure, as well
// as rewriting it as a nested array, or filtering it. It is akin to
// xpath, though simpler.
//
// This tool works primarily through string interfaces, so type
// information may be lost.
//
// # Query Syntax
//
// The query syntax handles both filtering and basic rewriting. It is
// not perfect. The idea behind it, is that we descend through an data
// structure, specifying what matches at each level.
//
// Each level of query can do:
//   - specify a filter, this is a simple string match with wildcard support. (prefix and/or postfix, but not infix)
//   - If the data is an array, specify an index
//   - For array-of-maps, specify a key to rewrite as a nested map
//
// Each query term has 3 parts: [#]string[=>kvmatch]
//
//  1. An optional `#` This denotes a key to rewrite an array-of-maps with
//
//  2. A search term. If this is an integer, it is interpreted as an array index.
//
//  3. a key/value match string. For a map, this is to match the value of a key.
//
//     Some examples:
//     *  data/users            Return everything under { data: { users: { ... } } }
//     *  data/users/0          Return the first item in the users array
//     *  data/users/name=>A*   Return users whose name starts with "A"
//     *  data/users/#id        Return the users, and rewrite the users array to be a map with the id as the key
//
// See the test suite for extensive examples.
// based on github.com/kolide/launcher/pkg/osquery/tables
package dataflatten

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/groob/plist"
	"github.com/rs/zerolog"

	howett "howett.net/plist"
)

// Flattener is an interface to flatten complex, nested, data
// structures. It recurses through them, and returns a simplified
// form. At the simplest level, this rewrites:
//
//	{ foo: { bar: { baz: 1 } } }
//
// To:
//
//	[ { path: foo/bar/baz, value: 1 } ]
//
// It can optionally filtering and rewriting.
type Flattener struct {
	debugLogging      bool
	expandNestedPlist bool
	includeNestedRaw  bool
	includeNils       bool
	logger            zerolog.Logger
	query             []string
	queryKeyDenoter   string
	queryWildcard     string
	rows              []Row
}

type FlattenOpts func(*Flattener)

// IncludeNulls indicates that Flatten should return null values,
// instead of skipping over them.
func IncludeNulls() FlattenOpts {
	return func(fl *Flattener) {
		fl.includeNils = true
	}
}

// WithNestedPlist indicates that nested plists should be expanded
func WithNestedPlist() FlattenOpts {
	return func(fl *Flattener) {
		fl.expandNestedPlist = true
	}
}

// WithLogger sets the logger to use
func WithLogger(logger zerolog.Logger) FlattenOpts {
	return func(fl *Flattener) {
		fl.logger = logger
	}
}

// WithDebugLogging enables debug logging. With debug logs,
// dataflatten is very verbose. This can overwhelm the other launcher
// logs. As we're not generally debugging this library, the default is
// to not enable debug logging.
func WithDebugLogging() FlattenOpts {
	return func(fl *Flattener) {
		fl.debugLogging = true
	}
}

// WithQuery Specifies a query to flatten with. This is used both for
// re-writing arrays into maps, and for filtering. See "Query
// Specification" for docs.
func WithQuery(q []string) FlattenOpts {
	if len(q) == 0 || (len(q) == 1 && q[0] == "") {
		return func(_ *Flattener) {}
	}

	return func(fl *Flattener) {
		fl.query = q
	}
}

// Flatten is the entry point to the Flattener functionality.
func Flatten(data interface{}, opts ...FlattenOpts) ([]Row, error) {
	fl := &Flattener{
		rows:            []Row{},
		logger:          zerolog.Nop(),
		queryWildcard:   `*`,
		queryKeyDenoter: `#`,
	}

	for _, opt := range opts {
		opt(fl)
	}

	if !fl.debugLogging {
		fl.logger = fl.logger.Level(zerolog.InfoLevel)
	}

	if err := fl.descend([]string{}, data, 0); err != nil {
		return nil, err
	}

	return fl.rows, nil
}

// descend recurses through a given data structure flattening along the way.
func (fl *Flattener) descend(path []string, data interface{}, depth int) error {
	queryTerm, isQueryMatched := fl.queryAtDepth(depth)
	logger := fl.logger.With().
		Str("caller", "descend").
		Int("depth", depth).
		Int("rows-so-far", len(fl.rows)).
		Str("query", queryTerm).
		Str("path", strings.Join(path, "/")).
		Logger()

	switch v := data.(type) {
	case []interface{}:
		for i, e := range v {
			pathKey := strconv.Itoa(i)
			logger.Debug().Str("indexStr", pathKey).Msg("checking an array")

			// If the queryTerm starts with
			// queryKeyDenoter, then we want to rewrite
			// the path based on it. Note that this does
			// no sanity checking. Multiple values will
			// re-write. If the value isn't there, you get
			// nothing. Etc.
			//
			// keyName == "name"
			// keyValue == "alex" (need to test this againsty queryTerm
			// pathKey == What we descend with
			if strings.HasPrefix(queryTerm, fl.queryKeyDenoter) {
				keyQuery := strings.SplitN(strings.TrimPrefix(queryTerm, fl.queryKeyDenoter), "=>", 2)
				keyName := keyQuery[0]

				innerlogger := logger.With().Str("arraykeyname", keyName).Logger()
				logger.Debug().Msg("attempting to coerce array into map")

				e, ok := e.(map[string]interface{})
				if !ok {
					innerlogger.Debug().Msg("can't coerce into map")
					continue
				}

				// Is keyName in this array?
				val, ok := e[keyName]
				if !ok {
					innerlogger.Debug().Msg("keyName not in map")
					continue
				}

				pathKey, ok = val.(string)
				if !ok {
					innerlogger.Debug().Msg("can't coerce pathKey val into string")
					continue
				}

				// Looks good to descend. we're overwritten both e and pathKey. Exit this conditional.
			}

			if !(isQueryMatched || fl.queryMatchArrayElement(e, i, queryTerm)) {
				logger.Debug().Msg("query not matched")
				continue
			}

			if err := fl.descend(append(path, pathKey), e, depth+1); err != nil {
				return fmt.Errorf("flattening array: %w", err)
			}
		}
	case map[string]interface{}:
		logger.Debug().Msg("checking a map")
		for k, e := range v {
			// Check that the key name matches. If not, skip this entire
			// branch of the map
			if !(isQueryMatched || fl.queryMatchString(k, queryTerm)) {
				continue
			}

			if err := fl.descend(append(path, k), e, depth+1); err != nil {
				return fmt.Errorf("flattening map: %w", err)
			}
		}
	case []map[string]interface{}:
		logger.Debug().Msg("checking an array of maps")
		for i, e := range v {
			if err := fl.descend(append(path, strconv.Itoa(i)), e, depth+1); err != nil {
				return fmt.Errorf("flattening array of maps: %w", err)
			}
		}
	case nil:
		// Because we want to filter nils out, we do _not_ examine isQueryMatched here
		if !(fl.queryMatchNil(queryTerm)) {
			logger.Debug().Msg("query not matched")
			return nil
		}
		fl.rows = append(fl.rows, NewRow(path, ""))
	case string:
		return fl.descendMaybePlist(path, []byte(v), depth)
	case []byte:
		// Most string like data comes in this way
		return fl.descendMaybePlist(path, v, depth)
	default:
		if err := fl.handleStringLike(logger, path, v, depth); err != nil {
			return fmt.Errorf("flattening at path %v: %w", path, err)
		}
	}
	return nil
}

// handleStringLike is called when we finally have an object we think
// can be converted to a string. It uses the depth to compare against
// the query, and returns a stringify'ed value
func (fl *Flattener) handleStringLike(logger zerolog.Logger, path []string, v interface{}, depth int) error {
	queryTerm, isQueryMatched := fl.queryAtDepth(depth)

	stringValue, err := stringify(v)
	if err != nil {
		return err
	}

	if !(isQueryMatched || fl.queryMatchString(stringValue, queryTerm)) {
		logger.Debug().Msg("query not matched")
		return nil
	}

	fl.rows = append(fl.rows, NewRow(path, stringValue))
	return nil
}

// descendMaybePlist optionally tries to decode []byte data as an
// embedded plist. In the case of failures, it falls back to treating
// it like a plain string.
func (fl *Flattener) descendMaybePlist(path []string, data []byte, depth int) error {
	logger := fl.logger.With().
		Str("caller", "descendMaybePlist").
		Int("depth", depth).
		Int("rows-so-far", len(fl.rows)).
		Str("path", strings.Join(path, "/")).
		Logger()

	// Skip if we're not expanding nested plists
	if !fl.expandNestedPlist {
		return fl.handleStringLike(logger, path, data, depth)
	}

	// Skip if this doesn't look like a plist.
	if !isPlist(data) {
		return fl.handleStringLike(logger, path, data, depth)
	}

	// Looks like a plist. Try parsing it
	logger.Debug().Msg("Parsing inner plist")

	var innerData interface{}

	if err := plist.Unmarshal(data, &innerData); err != nil {
		logger.Info().Err(err).Msg("plist parsing failed")
		return fl.handleStringLike(logger, path, data, depth)
	}

	// have a parsed plist. Descend and return from here.
	if fl.includeNestedRaw {
		if err := fl.handleStringLike(logger, append(path, "_raw"), data, depth); err != nil {
			logger.Error().Err(err).Msg("Failed to add _raw key")
		}
	}

	if err := fl.descend(path, innerData, depth); err != nil {
		return fmt.Errorf("flattening plist data: %w", err)
	}

	return nil
}

func (fl *Flattener) queryMatchNil(queryTerm string) bool {
	// TODO: If needed, we could use queryTerm for optional nil filtering
	return fl.includeNils
}

// queryMatchArrayElement matches arrays. This one is magic.
//
// Syntax:
//
//	#i -- Match index i. For example `#0`
//	k=>queryTerm -- If this is a map, it should have key k, that matches queryTerm
//
// We use `=>` as something that is reasonably intuitive, and not very
// likely to occur on it's own. Unfortunately, `==` shows up in base64
func (fl *Flattener) queryMatchArrayElement(data interface{}, arrIndex int, queryTerm string) bool {
	logger := fl.logger.With().
		Str("caller", "queryMatchArrayElement").
		Int("rows-so-far", len(fl.rows)).
		Str("query", queryTerm).
		Int("arrIndex", arrIndex).
		Logger()

	// strip off the key re-write denotation before trying to match
	queryTerm = strings.TrimPrefix(queryTerm, fl.queryKeyDenoter)

	if queryTerm == fl.queryWildcard {
		return true
	}

	// If the queryTerm is an int, then we expect to match the index
	if queryIndex, err := strconv.Atoi(queryTerm); err == nil {
		logger.Debug().Msg("using numeric index comparison")
		return queryIndex == arrIndex
	}

	logger.Debug().Msg("checking data type")

	switch dataCasted := data.(type) {
	case []interface{}:
		// fails. We can't match an array that has arrays as elements. Use a wildcard
		return false
	case map[string]interface{}:
		kvQuery := strings.SplitN(queryTerm, "=>", 2)

		// If this is one long, then we're testing for whether or not there's a key with this name,
		if len(kvQuery) == 1 {
			_, ok := dataCasted[kvQuery[0]]
			return ok
		}

		// Else see if the value matches
		for k, v := range dataCasted {
			// Since this needs to check against _every_
			// member, return true. Or fall through to the
			// false.
			if fl.queryMatchString(k, kvQuery[0]) && fl.queryMatchStringify(v, kvQuery[1]) {
				return true
			}
		}
		return false
	default:
		// non-iterable. stringify and be done
		return fl.queryMatchStringify(dataCasted, queryTerm)
	}
}

func (fl *Flattener) queryMatchStringify(data interface{}, queryTerm string) bool {
	// strip off the key re-write denotation before trying to match
	queryTerm = strings.TrimPrefix(queryTerm, fl.queryKeyDenoter)

	if queryTerm == fl.queryWildcard {
		return true
	}

	if data == nil {
		return fl.queryMatchNil(queryTerm)
	}

	stringValue, err := stringify(data)
	if err != nil {
		return false
	}

	return fl.queryMatchString(stringValue, queryTerm)
}

func (fl *Flattener) queryMatchString(v, queryTerm string) bool {
	if queryTerm == fl.queryWildcard {
		return true
	}

	// Some basic string manipulations to handle prefix and suffix operations
	switch {
	case strings.HasPrefix(queryTerm, fl.queryWildcard) && strings.HasSuffix(queryTerm, fl.queryWildcard):
		queryTerm = strings.TrimPrefix(queryTerm, fl.queryWildcard)
		queryTerm = strings.TrimSuffix(queryTerm, fl.queryWildcard)
		return strings.Contains(v, queryTerm)

	case strings.HasPrefix(queryTerm, fl.queryWildcard):
		queryTerm = strings.TrimPrefix(queryTerm, fl.queryWildcard)
		return strings.HasSuffix(v, queryTerm)

	case strings.HasSuffix(queryTerm, fl.queryWildcard):
		queryTerm = strings.TrimSuffix(queryTerm, fl.queryWildcard)
		return strings.HasPrefix(v, queryTerm)
	}

	return v == queryTerm
}

// queryAtDepth returns the query parameter for a given depth, and
// boolean indicating we've run out of queries. If we've run out of
// queries, than we can start checking, everything is a match.
func (fl *Flattener) queryAtDepth(depth int) (string, bool) {
	// if we're nil, there's an implied wildcard
	//
	// This works because:
	// []string   is len 0, and nil
	// []string{} is len 0, but not nil
	if fl.query == nil {
		return fl.queryWildcard, true
	}

	// If there's no query for this depth, then there's an implied
	// wildcard. This allows the query to specify prefixes.
	if depth+1 > len(fl.query) {
		return fl.queryWildcard, true
	}

	q := fl.query[depth]

	return q, q == fl.queryWildcard
}

// stringify takes an arbitrary piece of data, and attempst to coerce
// it into a string.
func stringify(data interface{}) (string, error) {
	switch v := data.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	case []byte:
		s := string(v)
		if utf8.ValidString(s) {
			return s, nil
		}
		return base64.StdEncoding.EncodeToString(v), nil
	case uint8:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case int:
		return strconv.Itoa(v), nil
	case int8:
		return strconv.FormatInt(int64(v), 10), nil
	case int16:
		return strconv.FormatInt(int64(v), 10), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case bool:
		return strconv.FormatBool(v), nil
	case time.Time:
		return strconv.FormatInt(v.Unix(), 10), nil
	case howett.UID:
		return strconv.FormatUint(uint64(v), 10), nil
	case fmt.Stringer:
		return v.String(), nil
	default:
		// spew.Dump(data)
		return "", fmt.Errorf("unknown type on %v", data)
	}
}

// isPlist returns whether or not something looks like it might be a
// plist. It uses Contains, instead of HasPrefix, as some encodings
// have a leading character.
func isPlist(data []byte) bool {
	var dataPrefix []byte
	if len(data) <= 30 {
		dataPrefix = data
	} else {
		dataPrefix = data[0:30]
	}

	if bytes.Contains(dataPrefix, []byte("bplist0")) {
		return true
	}

	if bytes.Contains(dataPrefix, []byte(`xml version="1.0"`)) && bytes.Contains(data, []byte(`<!DOCTYPE plist PUBLIC`)) {
		return true
	}

	return false
}
