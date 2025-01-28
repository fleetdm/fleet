package rawjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// CombineRoots "concatenates" two JSON objects into a single object.
//
// By virtue of its implementation it:
//
//   - Doesn't take into account nested keys
//   - Assumes the JSON string is well formed and was marshaled by the standard
//     library
func CombineRoots(a, b json.RawMessage) (json.RawMessage, error) {
	if err := validate(a); err != nil {
		return nil, fmt.Errorf("validating first object: %w", err)
	}

	if err := validate(b); err != nil {
		return nil, fmt.Errorf("validating second object: %w", err)
	}

	emptyObject := []byte{'{', '}'}
	if bytes.Equal(a, emptyObject) {
		return b, nil
	}
	if bytes.Equal(b, emptyObject) {
		return a, nil
	}

	// remove '}' from the first object and add a trailing ','
	combined := a[:len(a)-1]
	combined = append(combined, ',')
	// remove '{' from the second object and combine the two
	combined = append(combined, b[1:]...)
	return combined, nil
}

func validate(j json.RawMessage) error {
	if len(j) < 2 {
		return errors.New("incomplete json object")
	}

	if j[0] != '{' || j[len(j)-1] != '}' {
		return errors.New("json object must be surrounded by '{' and '}'")
	}

	if len(j) > 2 && j[len(j)-2] == ',' {
		return errors.New("trailing comma at the end of the object")
	}

	return nil
}
