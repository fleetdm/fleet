package apple_mdm

import (
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/micromdm/plist"
)

// ParseDeviceInformationResponse parses a DeviceInformation MDM command response plist
// into a flat map of "DeviceInformation.FieldName" -> string value.
func ParseDeviceInformationResponse(plistData []byte) (map[string]string, error) {
	var raw struct {
		QueryResponses map[string]interface{} `plist:"QueryResponses"`
	}
	if err := plist.Unmarshal(plistData, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal DeviceInformation response: %w", err)
	}

	result := make(map[string]string, len(raw.QueryResponses))
	flattenMap("DeviceInformation", raw.QueryResponses, result)
	return result, nil
}

// ParseSecurityInfoResponse parses a SecurityInfo MDM command response plist
// into a flat map of "SecurityInfo.FieldName" -> string value.
func ParseSecurityInfoResponse(plistData []byte) (map[string]string, error) {
	var raw struct {
		SecurityInfo map[string]interface{} `plist:"SecurityInfo"`
	}
	if err := plist.Unmarshal(plistData, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal SecurityInfo response: %w", err)
	}

	result := make(map[string]string, len(raw.SecurityInfo))
	flattenMap("SecurityInfo", raw.SecurityInfo, result)
	return result, nil
}

// ParseInstalledApplicationListResponse parses an InstalledApplicationList MDM command response
// into a flat map of "InstalledApplicationList.bundleID.FieldName" -> string value.
func ParseInstalledApplicationListResponse(plistData []byte) (map[string]string, error) {
	var raw struct {
		InstalledApplicationList []map[string]interface{} `plist:"InstalledApplicationList"`
	}
	if err := plist.Unmarshal(plistData, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal InstalledApplicationList response: %w", err)
	}

	result := make(map[string]string)
	for _, app := range raw.InstalledApplicationList {
		idVal, ok := app["Identifier"]
		if !ok {
			continue
		}
		identifier, ok := idVal.(string)
		if !ok || identifier == "" {
			continue
		}
		prefix := "InstalledApplicationList." + identifier
		flattenMap(prefix, app, result)
	}
	return result, nil
}

// toStringValue converts a plist value to its string representation.
func toStringValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int64:
		return strconv.FormatInt(val, 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case []byte:
		return base64.StdEncoding.EncodeToString(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// flattenMap recursively flattens a nested map into dot-notation keys.
func flattenMap(prefix string, m map[string]interface{}, result map[string]string) {
	for key, val := range m {
		fullKey := prefix + "." + key
		switch typedVal := val.(type) {
		case map[string]interface{}:
			flattenMap(fullKey, typedVal, result)
		default:
			result[fullKey] = toStringValue(val)
		}
	}
}
