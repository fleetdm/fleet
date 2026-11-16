package profilevalidation

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindingEncoding(t *testing.T) {
	out, err := json.Marshal(Finding{
		Code:        CodeTypeMismatch,
		PayloadType: "com.apple.applicationaccess",
		KeyPath:     "allowAirDrop",
		Expected:    []string{"boolean"},
		Actual:      "string",
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"code": "TYPE_MISMATCH",
		"payload_type": "com.apple.applicationaccess",
		"key_path": "allowAirDrop",
		"expected": ["boolean"],
		"actual": "string"
	}`, string(out))

	// An unknown type has no key or types to report.
	out, err = json.Marshal(Finding{Code: CodeUnknownType, PayloadType: "com.apple.example"})
	require.NoError(t, err)
	assert.JSONEq(t, `{"code": "UNKNOWN_TYPE", "payload_type": "com.apple.example"}`, string(out))
}
