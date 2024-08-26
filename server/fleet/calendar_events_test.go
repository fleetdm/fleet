package fleet

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveDataItems(t *testing.T) {
	t.Parallel()

	var event CalendarEvent

	assert.Equal(t, "", event.GetBodyTag())

	bodyTag := "bodyTag"
	require.NoError(t, event.SaveDataItems("body_tag", bodyTag))
	assert.Equal(t, bodyTag, event.GetBodyTag())

	testMap := make(map[string]any, 5)
	oldBodyTag := "oldBodyTag"
	testMap["body_tag"] = oldBodyTag
	testMap["key1"] = 1.2 // All JSON values are float type
	testMap["key2"] = "abc"
	data, err := json.Marshal(testMap)
	require.NoError(t, err)
	event.Data = data
	assert.Equal(t, oldBodyTag, event.GetBodyTag())

	require.NoError(t, event.SaveDataItems("body_tag", bodyTag))
	assert.Equal(t, bodyTag, event.GetBodyTag())

	// Make sure data was not modified
	require.NoError(t, event.SaveDataItems("body_tag", oldBodyTag))
	var result map[string]any
	require.NoError(t, json.Unmarshal(event.Data, &result))
	assert.Equal(t, testMap, result)
}
