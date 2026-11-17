package endpointer

import (
	"bytes"
	"encoding/json/jsontext"
	"errors"
	"io"
)

// RemoveJSONKeys removes the keys at paths, such as {"mdm", "setup_experience", "macos_script"},
// from a JSON object. A path whose parent is missing or isn't an object is skipped. Everything else
// is copied as is, so numbers keep their precision and keys keep their order.
func RemoveJSONKeys(data []byte, paths ...[]string) ([]byte, error) {
	dec := jsontext.NewDecoder(bytes.NewReader(data), jsontext.AllowDuplicateNames(true))
	switch dec.PeekKind() {
	case '{':
	case 'n':
		if _, err := dec.ReadToken(); err != nil {
			return nil, err
		}
		if err := requireEndOfInput(dec); err != nil {
			return nil, err
		}
		return data, nil
	default:
		if _, err := dec.ReadToken(); err != nil {
			return nil, err
		}
		return nil, errors.New("JSON value is not an object")
	}

	var buf bytes.Buffer
	enc := jsontext.NewEncoder(&buf, jsontext.AllowDuplicateNames(true))
	removed, err := copyObjectWithoutKeys(dec, enc, paths)
	if err != nil {
		return nil, err
	}
	if err := requireEndOfInput(dec); err != nil {
		return nil, err
	}
	if !removed {
		return data, nil
	}
	return bytes.TrimSpace(buf.Bytes()), nil
}

// copyObjectWithoutKeys copies the object dec is at to enc, leaving out the keys at paths.
func copyObjectWithoutKeys(dec *jsontext.Decoder, enc *jsontext.Encoder, paths [][]string) (bool, error) {
	if err := copyToken(dec, enc); err != nil {
		return false, err
	}
	var removed bool
	for dec.PeekKind() != '}' {
		key, err := dec.ReadToken()
		if err != nil {
			return false, err
		}
		var remove bool
		var nested [][]string
		for _, path := range paths {
			switch {
			case len(path) == 0 || path[0] != key.String():
			case len(path) == 1:
				remove = true
			default:
				nested = append(nested, path[1:])
			}
		}
		if remove {
			if err := dec.SkipValue(); err != nil {
				return false, err
			}
			removed = true
			continue
		}
		if err := enc.WriteToken(key); err != nil {
			return false, err
		}
		if len(nested) > 0 && dec.PeekKind() == '{' {
			nestedRemoved, err := copyObjectWithoutKeys(dec, enc, nested)
			if err != nil {
				return false, err
			}
			removed = removed || nestedRemoved
			continue
		}
		value, err := dec.ReadValue()
		if err != nil {
			return false, err
		}
		if err := enc.WriteValue(value); err != nil {
			return false, err
		}
	}
	return removed, copyToken(dec, enc)
}

func requireEndOfInput(dec *jsontext.Decoder) error {
	if _, err := dec.ReadToken(); err != io.EOF {
		if err == nil {
			err = errors.New("unexpected data after the JSON value")
		}
		return err
	}
	return nil
}

func copyToken(dec *jsontext.Decoder, enc *jsontext.Encoder) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	return enc.WriteToken(tok)
}
