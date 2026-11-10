package endpointer

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

// RemovedKeyError reports a request parameter that Fleet 5 removed or renamed. Its message comes from
// SetRemovedKeyMessage. It renders as the usual 422 validation error, {"message", "errors": [{"name", "reason"}]}.
type RemovedKeyError struct {
	Key string
}

var removedKeyMessage atomic.Pointer[func(key string) string]

// SetRemovedKeyMessage sets how RemovedKeyError words its message. The platform layer has no wording of its own: the
// service sets the Fleet 5 one from pkg/removednames, as it does for removed paths (RemovedPath.Message).
func SetRemovedKeyMessage(f func(key string) string) { removedKeyMessage.Store(&f) }

func (e *RemovedKeyError) Error() string {
	if f := removedKeyMessage.Load(); f != nil && *f != nil {
		return (*f)(e.Key)
	}
	return fmt.Sprintf("The %q parameter is not available anymore.", e.Key)
}

func (e *RemovedKeyError) Invalid() []map[string]string {
	return []map[string]string{{"name": e.Key, "reason": e.Error()}}
}

func (e *RemovedKeyError) IsClientError() bool { return true }

// RemovedKeyRule rejects Key. Scope works as in AliasRule.
type RemovedKeyRule struct {
	Key   string
	Scope []string
}

// matches reports whether key, sitting directly inside an object reached through enclosingKey, is the removed name.
// Both are compared ignoring case, as the JSON decoder matches field names, so a differently cased old name is
// rejected instead of silently dropped.
func (r RemovedKeyRule) matches(key, enclosingKey string) bool {
	if !strings.EqualFold(r.Key, key) {
		return false
	}
	return len(r.Scope) == 0 || slices.ContainsFunc(r.Scope, func(s string) bool { return strings.EqualFold(s, enclosingKey) })
}

var removedKeyRulesCache sync.Map // reflect.Type → []RemovedKeyRule

// ExtractRemovedKeyRules builds the rules from `removedkey` struct tags, found recursively like ExtractAliasRules. The
// tag sits on the field that carries the new name and lists the removed name(s), comma-separated:
//
//	RequireAllSoftware bool `json:"macos_require_all_software" removedkey:"require_all_software_macos"`
//
// Unlike `renameto`, the old name is rejected instead of accepted.
func ExtractRemovedKeyRules(iface any) []RemovedKeyRule {
	if iface == nil {
		return nil
	}
	t := reflect.TypeOf(iface)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	if cached, ok := removedKeyRulesCache.Load(t); ok {
		return cached.([]RemovedKeyRule)
	}

	var rules []RemovedKeyRule
	if deleted := deletedKeyRules.Load(); deleted != nil {
		rules = slices.Clone(*deleted)
	}

	seen := make(map[string]struct{})
	visited := make(map[reflect.Type]struct{})
	var walk func(t reflect.Type)
	walk = func(t reflect.Type) {
		if _, ok := visited[t]; ok {
			return
		}
		visited[t] = struct{}{}
		for sf := range t.Fields() {
			newKey, _, _ := strings.Cut(sf.Tag.Get("json"), ",")
			if removed := sf.Tag.Get("removedkey"); removed != "" && newKey != "" && newKey != "-" {
				scope := parseRenameScope(sf.Tag)
				for oldKey := range strings.SplitSeq(removed, ",") {
					oldKey = strings.TrimSpace(oldKey)
					id := strings.Join(append([]string{oldKey, newKey}, scope...), "\x00")
					if _, dup := seen[id]; oldKey == "" || dup {
						continue
					}
					seen[id] = struct{}{}
					rules = append(rules, RemovedKeyRule{Key: oldKey, Scope: scope})
				}
			}
			if ft := elemType(sf.Type); ft.Kind() == reflect.Struct {
				walk(ft)
			}
		}
	}
	walk(t)
	removedKeyRulesCache.Store(t, rules)
	return rules
}

// CheckRemovedKeys returns a *RemovedKeyError if data uses a removed key as an object key. It is for request bodies
// captured as json.RawMessage, which the request decoder does not inspect. Input it cannot parse returns the parse
// error rather than nil, since a more lenient decoder could still read a removed key past where parsing stopped.
func CheckRemovedKeys(data []byte, rules []RemovedKeyRule) error {
	if len(rules) == 0 || len(data) == 0 {
		return nil
	}
	return newJSONKeyRewriter(nil, rules, "").rewrite(bytes.NewReader(data), io.Discard)
}

var deletedKeyRules atomic.Pointer[[]RemovedKeyRule]

func SetDeletedKeyRules(rules []RemovedKeyRule) { deletedKeyRules.Store(&rules) }
