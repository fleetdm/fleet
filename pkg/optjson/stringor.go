package optjson

import (
	"bytes"
	"encoding/json"
)

// StringOr is a JSON value that can be a string or a different type of object
// (e.g. somewhat common for a string or an array of strings, but can also be
// a string or an object, etc.).
type StringOr[T any] struct {
	String  string
	Other   T
	IsOther bool
}

func (s StringOr[T]) MarshalJSON() ([]byte, error) {
	if s.IsOther {
		return json.Marshal(s.Other)
	}
	return json.Marshal(s.String)
}

func (s *StringOr[T]) UnmarshalJSON(data []byte) error {
	if bytes.HasPrefix(data, []byte(`"`)) {
		s.IsOther = false
		return json.Unmarshal(data, &s.String)
	}
	s.IsOther = true
	return json.Unmarshal(data, &s.Other)
}
