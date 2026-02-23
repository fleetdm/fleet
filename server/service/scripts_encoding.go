package service

import (
	"encoding/base64"
	"net/http"
)

// ScriptsEncodedHeader is the HTTP header used to signal that script fields
// in the request body are base64-encoded. This is used to bypass WAF rules
// that may block requests containing shell/PowerShell script patterns.
const ScriptsEncodedHeader = "X-Fleet-Scripts-Encoded"

// decodeBase64Script decodes a base64-encoded script string.
// Returns empty string for empty input, which allows callers to pass through
// empty/unset script fields without modification.
func decodeBase64Script(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// isScriptsEncoded checks if the request has the scripts encoding header
// set to "base64", indicating that script fields should be decoded.
func isScriptsEncoded(r *http.Request) bool {
	return r.Header.Get(ScriptsEncodedHeader) == "base64"
}
