package endpointer

import "encoding/json"

// RemoveJSONKeys removes the keys at paths, such as {"mdm", "setup_experience", "macos_script"},
// from a JSON object. A path whose parent is missing or isn't an object is skipped. Values that
// aren't on a path keep their original bytes, so numbers don't lose precision.
func RemoveJSONKeys(data []byte, paths ...[]string) ([]byte, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	changed, err := removeJSONKeys(obj, paths)
	if err != nil || !changed {
		return data, err
	}
	return json.Marshal(obj)
}

func removeJSONKeys(obj map[string]json.RawMessage, paths [][]string) (bool, error) {
	var changed bool
	nested := make(map[string][][]string)
	for _, path := range paths {
		switch len(path) {
		case 0:
		case 1:
			if _, ok := obj[path[0]]; ok {
				delete(obj, path[0])
				changed = true
			}
		default:
			nested[path[0]] = append(nested[path[0]], path[1:])
		}
	}
	for key, subpaths := range nested {
		raw, ok := obj[key]
		if !ok {
			continue
		}
		var child map[string]json.RawMessage
		// data was valid JSON, so an error here means the value isn't an object.
		if err := json.Unmarshal(raw, &child); err != nil || child == nil {
			continue
		}
		childChanged, err := removeJSONKeys(child, subpaths)
		if err != nil {
			return false, err
		}
		if !childChanged {
			continue
		}
		if obj[key], err = json.Marshal(child); err != nil {
			return false, err
		}
		changed = true
	}
	return changed, nil
}
