package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
)

func TestRedactOperationSensitiveFields(t *testing.T) {
	t.Run("nil metadata is a no-op", func(t *testing.T) {
		op := &androidmanagement.Operation{Name: "enterprises/E/devices/D/operations/1", Done: true}
		redactOperationSensitiveFields(op)
		assert.Nil(t, op.Metadata)
	})

	t.Run("empty metadata is a no-op", func(t *testing.T) {
		op := &androidmanagement.Operation{
			Name:     "enterprises/E/devices/D/operations/1",
			Done:     true,
			Metadata: googleapi.RawMessage{},
		}
		redactOperationSensitiveFields(op)
		assert.Empty(t, op.Metadata)
	})

	t.Run("metadata without newPassword is unchanged", func(t *testing.T) {
		meta := `{"@type":"type.googleapis.com/google.android.devicemanagement.v1.Command","type":"LOCK","createTime":"2026-01-01T00:00:00Z"}`
		op := &androidmanagement.Operation{
			Name:     "enterprises/E/devices/D/operations/1",
			Done:     true,
			Metadata: googleapi.RawMessage(meta),
		}
		redactOperationSensitiveFields(op)

		var m map[string]any
		require.NoError(t, json.Unmarshal(op.Metadata, &m))
		assert.Equal(t, "LOCK", m["type"])
		assert.NotContains(t, m, "newPassword")
	})

	t.Run("newPassword is removed from metadata", func(t *testing.T) {
		meta := `{"@type":"type.googleapis.com/google.android.devicemanagement.v1.Command","type":"RESET_PASSWORD","newPassword":"s3cret!","createTime":"2026-01-01T00:00:00Z","userName":"enterprises/E/users/U"}`
		op := &androidmanagement.Operation{
			Name:     "enterprises/E/devices/D/operations/1",
			Done:     true,
			Metadata: googleapi.RawMessage(meta),
		}
		redactOperationSensitiveFields(op)

		var m map[string]any
		require.NoError(t, json.Unmarshal(op.Metadata, &m))
		assert.NotContains(t, m, "newPassword", "newPassword should be deleted from metadata")
		assert.Equal(t, "RESET_PASSWORD", m["type"], "other fields should be preserved")
		assert.Equal(t, "enterprises/E/users/U", m["userName"], "other fields should be preserved")
	})

	t.Run("invalid JSON without sensitive key is left unchanged", func(t *testing.T) {
		badJSON := googleapi.RawMessage(`{not valid json}`)
		op := &androidmanagement.Operation{
			Name:     "enterprises/E/devices/D/operations/1",
			Done:     true,
			Metadata: badJSON,
		}
		redactOperationSensitiveFields(op)
		assert.Equal(t, googleapi.RawMessage(`{not valid json}`), op.Metadata)
	})

	t.Run("invalid JSON containing sensitive key is scrubbed via regex", func(t *testing.T) {
		badJSON := googleapi.RawMessage(`{not valid "newPassword": "leaked"}`)
		op := &androidmanagement.Operation{
			Name:     "enterprises/E/devices/D/operations/1",
			Done:     true,
			Metadata: badJSON,
		}
		redactOperationSensitiveFields(op)
		assert.NotContains(t, string(op.Metadata), "newPassword")
		assert.NotContains(t, string(op.Metadata), "leaked")
	})

	t.Run("unicode-escaped newPassword key in invalid JSON is scrubbed", func(t *testing.T) {
		badJSON := googleapi.RawMessage(`[{"\u006eewPassword": "secret"}]`)
		op := &androidmanagement.Operation{
			Name:     "enterprises/E/devices/D/operations/1",
			Done:     true,
			Metadata: badJSON,
		}
		redactOperationSensitiveFields(op)
		assert.NotContains(t, string(op.Metadata), "secret")
	})
}
