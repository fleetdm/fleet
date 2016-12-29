package kolide

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionMarshaller(t *testing.T) {
	tests := []struct {
		value    interface{}
		typ      OptionType
		expected interface{}
	}{
		{23, OptionTypeInt, float64(23)},
		{true, OptionTypeBool, true},
		{"foobar", OptionTypeString, "foobar"},
	}

	for _, test := range tests {
		optIn := &Option{1, "foo", test.typ, OptionValue{test.value}, true}
		buff, err := json.Marshal(optIn)
		require.Nil(t, err)
		optOut := &Option{}
		err = json.Unmarshal(buff, optOut)
		require.Nil(t, err)
		assert.Equal(t, optIn.ID, optOut.ID)
		assert.Equal(t, optIn.Name, optOut.Name)
		assert.Equal(t, optIn.ReadOnly, optOut.ReadOnly)
		assert.Equal(t, optIn.Type, optOut.Type)
		assert.Equal(t, test.expected, optOut.Value.Val)

	}

	// test nil
	optIn := &Option{1, "bar", OptionTypeString, OptionValue{nil}, true}
	buff, err := json.Marshal(optIn)
	require.Nil(t, err)
	optOut := &Option{}
	err = json.Unmarshal(buff, optOut)
	require.Nil(t, err)
	assert.True(t, reflect.DeepEqual(optIn, optOut))

}
