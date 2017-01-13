package kolide

import (
	"encoding/json"
	"errors"
)

// UnmarshalJSON custom unmarshaling for PackNameMap will determine whether
// the pack section of an osquery config file refers to a file path, or
// pack details.  Pack details are unmarshalled into into PackDetails structure
// as oppossed to nested map[string]interface{}
func (pnm PackNameMap) UnmarshalJSON(b []byte) error {
	var temp map[string]interface{}
	err := json.Unmarshal(b, &temp)
	if err != nil {
		return err
	}
	for key, val := range temp {
		switch t := val.(type) {
		case string:
			pnm[key] = t
		case map[string]interface{}:
			pnm[key] = unmarshalPackDetails(t)
		default:
			return errors.New("can't unmarshal json")
		}
	}
	return nil
}

func strptr(v interface{}) *string {
	if v == nil {
		return nil
	}
	s := new(string)
	*s = v.(string)
	return s
}

func boolptr(v interface{}) *bool {
	if v == nil {
		return nil
	}
	b := new(bool)
	*b = v.(bool)
	return b
}

func uintptr(v interface{}) *uint {
	if v == nil {
		return nil
	}
	i := new(uint)
	*i = uint(v.(float64))
	return i
}

func unmarshalPackDetails(v map[string]interface{}) PackDetails {
	return PackDetails{
		Queries:   unmarshalQueryDetails(v["queries"]),
		Shard:     uintptr(v["shard"]),
		Version:   strptr(v["version"]),
		Platform:  v["platform"].(string),
		Discovery: unmarshalDiscovery(v["discovery"]),
	}
}

func unmarshalDiscovery(val interface{}) []string {
	var result []string
	if val == nil {
		return result
	}
	v := val.([]interface{})
	for _, val := range v {
		result = append(result, val.(string))
	}
	return result
}

func unmarshalQueryDetails(v interface{}) QueryNameToQueryDetailsMap {
	result := make(QueryNameToQueryDetailsMap)
	if v == nil {
		return result
	}
	for qn, details := range v.(map[string]interface{}) {
		result[qn] = unmarshalQueryDetail(details)
	}
	return result
}

func unmarshalQueryDetail(val interface{}) QueryDetails {
	v := val.(map[string]interface{})
	return QueryDetails{
		Query:    v["query"].(string),
		Interval: uint(v["interval"].(float64)),
		Removed:  boolptr(v["removed"]),
		Platform: strptr(v["platform"]),
		Version:  strptr(v["version"]),
		Shard:    uintptr(v["shard"]),
		Snapshot: boolptr(v["snapshot"]),
	}
}
