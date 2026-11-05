package service

import (
	"bytes"
	"encoding/json"
	"slices"
	"sync"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/activity/api"
)

// removedDetailsKeys maps the removed keys of stored activity details to their replacements.
var removedDetailsKeys = sync.OnceValue(func() map[string]string {
	keys := make(map[string]string)
	for _, e := range removednames.Filter(removednames.KindJSONKey, removednames.ScopeActivityDetails) {
		keys[e.Old] = e.New
	}
	return keys
})

// removedKeyNeedles holds the quoted removed keys, and their common prefix as a cheap first check that most rows fail.
var removedKeyNeedles = sync.OnceValues(func() ([]byte, [][]byte) {
	var prefix []byte
	var exact [][]byte
	for key := range removedDetailsKeys() {
		needle := []byte(`"` + key + `"`)
		exact = append(exact, needle)
		if prefix == nil {
			prefix = needle
		}
		for !bytes.HasPrefix(needle, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix, exact
})

// mayHaveRemovedKeys reports whether the details contain a removed key. It can only say yes wrongly if a value is the same quoted text.
func mayHaveRemovedKeys(details []byte) bool {
	prefix, exact := removedKeyNeedles()
	return bytes.Contains(details, prefix) && slices.ContainsFunc(exact, func(needle []byte) bool { return bytes.Contains(details, needle) })
}

// translateStoredDetails renames the removed top-level keys in the details of activities stored before Fleet 5. Only the top level holds
// them: nested values, such as the app-owned configuration, are left as stored. A row written while both names were stored keeps the new
// key's value. Details that cannot be decoded are returned unchanged.
func translateStoredDetails(activities []*api.Activity) {
	for _, a := range activities {
		if a.Details == nil || !mayHaveRemovedKeys(*a.Details) {
			continue
		}
		var details map[string]json.RawMessage
		if err := json.Unmarshal(*a.Details, &details); err != nil || details == nil {
			continue
		}
		for key, value := range details {
			if newKey, removed := removedDetailsKeys()[key]; removed {
				delete(details, key)
				if _, exists := details[newKey]; !exists {
					details[newKey] = value
				}
			}
		}
		translated, err := json.Marshal(details)
		if err != nil {
			continue
		}
		raw := json.RawMessage(translated)
		a.Details = &raw
	}
}
