package profilevalidation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/variables"
	"howett.net/plist"
)

// decodeAppleProfile decodes a .mobileconfig into a value tree that keeps each plist element's type.
func decodeAppleProfile(data []byte) (map[string]any, error) {
	if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("<?xml")) {
		return nil, errors.New("signed profiles are not supported")
	}
	// DigiCert <data> placeholders aren't valid base64 until expansion; blank them as the upload
	// parser does.
	data = variables.ProfileDataVariableRegex.ReplaceAll(data, nil)
	var root any
	if _, err := plist.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("decode profile: %w", err)
	}
	dict, ok := root.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("decode profile: root is %s, not dictionary", kindOf(root))
	}
	return dict, nil
}

// decodeAppleDeclaration decodes a DDM declaration, keeping integers and reals apart.
func decodeAppleDeclaration(data []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var root any
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("decode declaration: %w", err)
	}
	dict, ok := root.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("decode declaration: root is %s, not dictionary", kindOf(root))
	}
	return dict, nil
}

// kindOf names a decoded value's type with the schema's type names.
func kindOf(v any) string {
	switch v := v.(type) {
	case string:
		return string(AppleTypeString)
	case bool:
		return string(AppleTypeBoolean)
	case int64, uint64:
		return string(AppleTypeInteger)
	case float64:
		return string(AppleTypeReal)
	case json.Number:
		if strings.ContainsAny(v.String(), ".eE") {
			return string(AppleTypeReal)
		}
		return string(AppleTypeInteger)
	case time.Time:
		return string(AppleTypeDate)
	case []byte:
		return string(AppleTypeData)
	case []any:
		return string(AppleTypeArray)
	case map[string]any:
		return string(AppleTypeDictionary)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", v)
	}
}
