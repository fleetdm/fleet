package endpointer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveJSONKeys(t *testing.T) {
	paths := [][]string{
		{"scripts"},
		{"mdm", "volume_purchasing_program"},
		{"mdm", "setup_experience", "macos_bootstrap_package"},
		{"mdm", "setup_experience", "macos_script"},
	}

	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "removes top-level and nested keys",
			in:   `{"scripts":["a.sh"],"mdm":{"volume_purchasing_program":[],"setup_experience":{"macos_bootstrap_package":"https://example.com/b.pkg","macos_script":"s.sh","enable_release_device_manually":true}},"org_info":{"org_name":"Acme"}}`,
			want: `{"mdm":{"setup_experience":{"enable_release_device_manually":true}},"org_info":{"org_name":"Acme"}}`,
		},
		{
			name: "missing keys leave the input unchanged",
			in:   `{"org_info": {"org_name": "Acme"}, "big": 12345678901234567890}`,
			want: `{"org_info": {"org_name": "Acme"}, "big": 12345678901234567890}`,
		},
		{
			name: "skips parents that are null or not objects",
			in:   `{"mdm":null,"scripts":"x"}`,
			want: `{"mdm":null}`,
		},
		{
			name: "skips a nested parent that isn't an object",
			in:   `{"mdm":{"setup_experience":["macos_script"]}}`,
			want: `{"mdm":{"setup_experience":["macos_script"]}}`,
		},
		{
			name: "keeps a parent that ends up empty",
			in:   `{"mdm":{"setup_experience":{"macos_script":""}}}`,
			want: `{"mdm":{"setup_experience":{}}}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := RemoveJSONKeys([]byte(c.in), paths...)
			require.NoError(t, err)
			assert.JSONEq(t, c.want, string(got))
		})
	}

	t.Run("keeps the precision of other numbers", func(t *testing.T) {
		got, err := RemoveJSONKeys([]byte(`{"scripts":[],"mdm":{"big":12345678901234567890,"volume_purchasing_program":null}}`), paths...)
		require.NoError(t, err)
		assert.JSONEq(t, `{"mdm":{"big":12345678901234567890}}`, string(got))
		// JSONEq compares numbers as float64, which would miss lost digits.
		assert.Contains(t, string(got), "12345678901234567890")
	})

	t.Run("unchanged input is returned as is", func(t *testing.T) {
		in := []byte(`{ "a" : 1 }`)
		got, err := RemoveJSONKeys(in, []string{"b"})
		require.NoError(t, err)
		assert.Equal(t, string(in), string(got))
	})

	t.Run("not an object", func(t *testing.T) {
		_, err := RemoveJSONKeys([]byte(`[1, 2]`), []string{"a"})
		require.Error(t, err)
		_, err = RemoveJSONKeys([]byte(`{"a":`), []string{"a"})
		require.Error(t, err)
	})
}
