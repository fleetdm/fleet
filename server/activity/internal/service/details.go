package service

import (
	"bytes"
	"encoding/json"
	"strings"
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

// translateStoredDetails renames the removed keys in the details of activities stored before Fleet 5, at any depth, and rewrites the
// removed team_id= query parameter of stored icon URLs. A row written while both names were stored keeps the new key's value. Details
// that cannot be decoded are returned unchanged.
func translateStoredDetails(activities []*api.Activity) {
	for _, a := range activities {
		if a.Details == nil || !bytes.Contains(*a.Details, []byte("team")) {
			continue
		}
		dec := json.NewDecoder(bytes.NewReader(*a.Details))
		dec.UseNumber()
		var details any
		if err := dec.Decode(&details); err != nil {
			continue
		}
		translated, err := json.Marshal(renameRemovedKeys(details))
		if err != nil {
			continue
		}
		raw := json.RawMessage(translated)
		a.Details = &raw
	}
}

func renameRemovedKeys(v any) any {
	switch v := v.(type) {
	case map[string]any:
		for key, value := range v {
			if newKey, removed := removedDetailsKeys()[key]; removed {
				delete(v, key)
				if _, exists := v[newKey]; !exists {
					v[newKey] = value
				}
			}
		}
		for key, value := range v {
			if s, ok := value.(string); ok && key == "software_icon_url" {
				v[key] = strings.ReplaceAll(s, "team_id=", "fleet_id=")
				continue
			}
			v[key] = renameRemovedKeys(value)
		}
	case []any:
		for i, value := range v {
			v[i] = renameRemovedKeys(value)
		}
	}
	return v
}
