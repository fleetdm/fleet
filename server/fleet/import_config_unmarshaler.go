package fleet

import (
	"encoding/json"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

var wrongTypeError = errors.New("argument missing or unexpected type")

// UnmarshalJSON custom unmarshaling for PackNameMap will determine whether
// the pack section of an osquery config file refers to a file path, or
// pack details.  Pack details are unmarshalled into into PackDetails structure
// as opposed to nested map[string]interface{}
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
			val, err := unmarshalPackDetails(t)
			if err != nil {
				return err
			}
			pnm[key] = val
		default:
			return errors.Errorf("can't unmarshal %s %v", key, val)
		}
	}
	return nil
}

func strptr(v interface{}) (*string, error) {
	if v == nil {
		return nil, nil
	}
	s, ok := v.(string)
	if !ok {
		return nil, wrongTypeError
	}
	return &s, nil
}

func boolptr(v interface{}) (*bool, error) {
	if v == nil {
		return nil, nil
	}
	b, ok := v.(bool)
	if !ok {
		return nil, wrongTypeError
	}
	return &b, nil
}

// We expect a float64 here because of the way JSON represents numbers
func uintptr(v interface{}) (*OsQueryConfigInt, error) {
	if v == nil {
		return nil, nil
	}
	i, err := unmarshalInteger(v)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func unmarshalPackDetails(v map[string]interface{}) (PackDetails, error) {
	var result PackDetails
	queries, err := unmarshalQueryDetails(v["queries"])
	if err != nil {
		return result, err
	}
	discovery, err := unmarshalDiscovery(v["discovery"])
	if err != nil {
		return result, err
	}
	platform := cast.ToString(v["platform"])
	shard, err := uintptr(v["shard"])
	if err != nil {
		return result, err
	}
	version, err := strptr(v["version"])
	if err != nil {
		return result, err
	}
	result = PackDetails{
		Queries:   queries,
		Shard:     shard,
		Version:   version,
		Platform:  platform,
		Discovery: discovery,
	}
	return result, nil
}

func unmarshalDiscovery(val interface{}) ([]string, error) {
	var result []string
	if val == nil {
		return result, nil
	}
	v, ok := val.([]interface{})
	if !ok {
		return result, wrongTypeError
	}
	for _, val := range v {
		query, err := cast.ToStringE(val)
		if err != nil {
			return result, err
		}
		result = append(result, query)
	}
	return result, nil
}

func unmarshalQueryDetails(v interface{}) (QueryNameToQueryDetailsMap, error) {
	var err error
	result := make(QueryNameToQueryDetailsMap)
	if v == nil {
		return result, nil
	}
	for qn, details := range v.(map[string]interface{}) {
		result[qn], err = unmarshalQueryDetail(details)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func unmarshalQueryDetail(val interface{}) (QueryDetails, error) {
	var result QueryDetails
	v, ok := val.(map[string]interface{})
	if !ok {
		return result, errors.New("argument was missing or the wrong type")
	}
	interval, err := unmarshalInteger(v["interval"])
	if err != nil {
		return result, err
	}
	query, err := cast.ToStringE(v["query"])
	if err != nil {
		return result, err
	}
	removed, err := boolptr(v["removed"])
	if err != nil {
		return result, err
	}
	platform, err := strptr(v["platform"])
	if err != nil {
		return result, err
	}
	version, err := strptr(v["version"])
	if err != nil {
		return result, err
	}
	shard, err := uintptr(v["shard"])
	if err != nil {
		return result, err
	}
	snapshot, err := boolptr(v["snapshot"])
	if err != nil {
		return result, nil
	}
	result = QueryDetails{
		Query:    query,
		Interval: OsQueryConfigInt(interval),
		Removed:  removed,
		Platform: platform,
		Version:  version,
		Shard:    shard,
		Snapshot: snapshot,
	}
	return result, nil
}

// It is valid for the interval can be a string that is convertable to an int,
// or an float64. The float64 is how all numbers in JSON are represented, so
// we need to convert to uint
func unmarshalInteger(val interface{}) (OsQueryConfigInt, error) {
	// if interval is nil return zero value
	if val == nil {
		return OsQueryConfigInt(0), nil
	}
	switch v := val.(type) {
	case string:
		i, err := strconv.ParseUint(v, 10, 64)
		return OsQueryConfigInt(i), err
	case float64:
		return OsQueryConfigInt(v), nil
	default:
		return OsQueryConfigInt(0), wrongTypeError
	}
}
