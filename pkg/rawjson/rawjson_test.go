package rawjson

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCombineRoots(t *testing.T) {
	tests := []struct {
		name    string
		a       json.RawMessage
		b       json.RawMessage
		want    json.RawMessage
		wantErr string
	}{
		{
			name: "both empty",
			a:    []byte("{}"),
			b:    []byte("{}"),
			want: []byte("{}"),
		},
		{
			name:    "first incomplete",
			a:       []byte("{"),
			b:       []byte("{}"),
			wantErr: "incomplete json object",
		},
		{
			name:    "second incomplete",
			a:       []byte("{}"),
			b:       []byte("{"),
			wantErr: "incomplete json object",
		},
		{
			name:    "first empty array",
			a:       []byte{},
			b:       []byte("{}"),
			wantErr: "incomplete json object",
		},
		{
			name:    "second empty array",
			a:       []byte("{}"),
			b:       []byte{},
			wantErr: "incomplete json object",
		},
		{
			name: "first empty",
			a:    []byte("{}"),
			b:    []byte(`{"key":"value"}`),
			want: []byte(`{"key":"value"}`),
		},
		{
			name: "second empty",
			a:    []byte(`{"key":"value"}`),
			b:    []byte("{}"),
			want: []byte(`{"key":"value"}`),
		},
		{
			name: "both with data",
			a:    []byte(`{"key1":"value1"}`),
			b:    []byte(`{"key2":"value2"}`),
			want: []byte(`{"key1":"value1","key2":"value2"}`),
		},
		{
			name:    "first incomplete",
			a:       []byte(`{"key1":"value1"`),
			b:       []byte(`{"key2":"value2"}`),
			wantErr: "json object must be surrounded by '{' and '}'",
		},
		{
			name:    "second incomplete",
			a:       []byte(`{"key2":"value2"}`),
			b:       []byte(`{"key1":"value1"`),
			wantErr: "json object must be surrounded by '{' and '}'",
		},
		{
			name:    "first trailing comma",
			a:       []byte(`{"key1":"value1",}`),
			b:       []byte(`{"key2":"value2"}`),
			wantErr: "trailing comma at the end of the object",
		},
		{
			name:    "second trailing comma",
			a:       []byte(`{"key1":"value1"}`),
			b:       []byte(`{"key2":"value2",}`),
			wantErr: "trailing comma at the end of the object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CombineRoots(tt.a, tt.b)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}
