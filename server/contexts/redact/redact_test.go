package redact

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScrubJSON(t *testing.T) {
	ctx := NewContext(t.Context())
	Record(ctx, "s3cr3t", `a"b<c>`, "s3cr3t-longer", "")

	var sb strings.Builder
	enc := json.NewEncoder(JSONWriter(ctx, &sb))
	require.NoError(t, enc.Encode(map[string]string{
		"reason": `invalid PayloadType: s3cr3t-longer, s3cr3t and a"b<c>`,
	}))

	out := sb.String()
	require.NotContains(t, out, "s3cr3t")
	require.NotContains(t, out, `a\"b<c>`)
	require.NotContains(t, out, "a\"b\u003c")
	require.Contains(t, out, "invalid PayloadType: [redacted], [redacted] and [redacted]")
}

func TestScrubJSONQuotedForms(t *testing.T) {
	for name, v := range map[string]string{
		"html and quote": `x&y<z>"q'`,
		"control char":   "a\x01b",
		"nbsp":           "a b",
		"zero width":     "a​b",
		"invalid utf8":   "a\xff\xfeb",
		"emoji":          "a🔑b",
	} {
		t.Run(name, func(t *testing.T) {
			ctx := NewContext(t.Context())
			Record(ctx, v)

			for _, verb := range []string{"%s", "%q", "%+q"} {
				var sb strings.Builder
				reason := fmt.Sprintf("bad "+verb, v)
				require.NoError(t, json.NewEncoder(JSONWriter(ctx, &sb)).Encode(map[string]string{"reason": reason}))

				var got map[string]string
				require.NoError(t, json.Unmarshal([]byte(sb.String()), &got))
				want := "bad " + Placeholder
				if verb != "%s" {
					want = `bad "` + Placeholder + `"`
				}
				require.Equal(t, want, got["reason"], verb)
			}
		})
	}
}

func TestRecordWithoutContext(t *testing.T) {
	ctx := t.Context()
	Record(ctx, "s3cr3t")
	require.Equal(t, []byte("s3cr3t"), ScrubJSON(ctx, []byte("s3cr3t")))
}
