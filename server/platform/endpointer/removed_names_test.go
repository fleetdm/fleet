package endpointer

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/stretchr/testify/require"
)

type scanEmbedded struct {
	FleetID *uint `json:"fleet_id" query:"fleet_id"`
}

type scanSelfDecoding struct{ Inner string }

func (s *scanSelfDecoding) UnmarshalJSON(b []byte) error { s.Inner = string(b); return nil }

type scanItem struct {
	Name   string `json:"name"`
	TeamID *uint  `json:"team_id"` //nolint:apiparamcheck // a surface that has not been renamed yet
}

type scanRequest struct {
	scanEmbedded
	Name     string              `json:"name"`
	Skipped  string              `json:"-"`
	Item     *scanItem           `json:"item"`
	Items    []scanItem          `json:"items"`
	Renamed  []scanItem          `json:"queries" renameto:"reports"`
	ByName   map[string]scanItem `json:"by_name"`
	Settings map[string]any      `json:"settings"`
	Anything any                 `json:"anything"`
	Raw      json.RawMessage     `json:"raw"`
	Own      scanSelfDecoding    `json:"own"`
	Page     int                 `query:"page,optional"`
	OldKept  *uint               `query:"team_id,optional"` //nolint:apiparamcheck // a surface that has not been renamed yet
}

func TestRejectRemovedJSONKeys(t *testing.T) {
	SetRemovedNames([]string{"team_id", "teams", "Team"}, []string{"team_id", "team_name"}, nil)
	t.Cleanup(func() { SetRemovedNames(nil, nil, nil) })
	typ := reflect.TypeFor[scanRequest]()

	for _, tc := range []struct {
		name, body, wantKey string
	}{
		{name: "removed key at the top level", body: `{"name": "a", "teams": []}`, wantKey: "teams"},
		{name: "case does not matter, the message keeps the sent spelling", body: `{"TEAM_ID": 1}`, wantKey: "TEAM_ID"},
		{name: "old and new name together", body: `{"fleet_id": 1, "team_id": 1}`, wantKey: "team_id"},
		{name: "removed key in a nested struct, declared name next to it", body: `{"items": [{"name": "a", "teams": 1}]}`, wantKey: "teams"},
		{name: "removed key in a container sent under its renameto name", body: `{"reports": [{"teams": 1}]}`, wantKey: "teams"},
		{name: "removed key in a container sent under its json name", body: `{"queries": [{"teams": 1}]}`, wantKey: "teams"},
		{name: "removed key after unrelated unknown keys", body: `{"extra": 1, "another": {"x": 1}, "teams": []}`, wantKey: "teams"},

		{name: "new names only", body: `{"fleet_id": 1, "name": "a"}`},
		{name: "a removed name that the type still declares is accepted", body: `{"item": {"team_id": 3}, "items": [{"team_id": 3}]}`},
		{name: "free-form map is not looked into", body: `{"settings": {"teams": [1], "team_id": 2, "nested": {"Team": "x"}}}`},
		{name: "any is not looked into", body: `{"anything": {"team_id": 1}}`},
		{name: "raw JSON is not looked into", body: `{"raw": {"team_id": 1}}`},
		{name: "a type that decodes itself is not looked into", body: `{"own": {"team_id": 1}}`},
		{name: "a json:- field is not a name the type declares", body: `{"Skipped": "x"}`},
		{name: "maps are not looked into, their keys are data", body: `{"by_name": {"teams": {"teams": 1}}}`},
		{name: "removed names as values", body: `{"name": "team_id", "items": [{"name": "teams"}]}`},
		{name: "unknown keys that are not removed are left to the decoder", body: `{"unknown": {"team": 1}}`, wantKey: ""},
		{name: "scalar and array bodies", body: `[{"teams": 1}]`},
		{name: "malformed JSON is left to the decoder", body: `{"name": `},
		{name: "empty body", body: ``},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := RejectRemovedJSONKeys([]byte(tc.body), typ)
			if tc.wantKey == "" {
				require.NoError(t, err)
				return
			}
			rke, ok := err.(*RemovedKeyError)
			require.True(t, ok, "got %v", err)
			require.Equal(t, tc.wantKey, rke.Key)
		})
	}

	t.Run("the embedded struct's fields are the type's own", func(t *testing.T) {
		require.NoError(t, RejectRemovedJSONKeys([]byte(`{"fleet_id": 1}`), typ))
	})
	t.Run("no names set", func(t *testing.T) {
		SetRemovedNames(nil, nil, nil)
		require.NoError(t, RejectRemovedJSONKeys([]byte(`{"teams": 1}`), typ))
	})
}

type handParsedRequest struct {
	Opts struct{} `url:"host_options"`
}

func TestRejectRemovedQueryParams(t *testing.T) {
	SetRemovedNames(nil, []string{"team_id", "team_name"}, map[string][]string{"host_options": {"team_id"}})
	t.Cleanup(func() { SetRemovedNames(nil, nil, nil) })

	type renamed struct {
		FleetID *uint `query:"fleet_id,optional"`
	}
	for _, tc := range []struct {
		name    string
		typ     reflect.Type
		query   string
		wantKey string
	}{
		{name: "removed parameter", typ: reflect.TypeFor[renamed](), query: "team_id=1", wantKey: "team_id"},
		{name: "with the new name", typ: reflect.TypeFor[renamed](), query: "fleet_id=0&team_id=1", wantKey: "team_id"},
		{name: "the other removed parameter", typ: reflect.TypeFor[renamed](), query: "team_name=a", wantKey: "team_name"},
		{name: "new name only", typ: reflect.TypeFor[renamed](), query: "fleet_id=1"},
		{name: "a parameter the type still declares", typ: reflect.TypeFor[scanRequest](), query: "team_id=1"},
		{name: "declared through an embedded struct", typ: reflect.TypeFor[scanRequest](), query: "fleet_id=1&team_name=x", wantKey: "team_name"},
		{name: "parameter names are case sensitive, like the decoder", typ: reflect.TypeFor[renamed](), query: "Team_ID=1"},
		{name: "a hand-parsed field accepts the parameter its parser reads", typ: reflect.TypeFor[handParsedRequest](), query: "team_id=1"},
		{name: "a hand-parsed field does not accept the others", typ: reflect.TypeFor[handParsedRequest](), query: "team_name=a", wantKey: "team_name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values, err := url.ParseQuery(tc.query)
			require.NoError(t, err)
			err = RejectRemovedQueryParams(values, tc.typ)
			if tc.wantKey == "" {
				require.NoError(t, err)
				return
			}
			rke, ok := err.(*RemovedKeyError)
			require.True(t, ok, "got %v", err)
			require.Equal(t, tc.wantKey, rke.Key)
		})
	}
}

func BenchmarkRejectRemovedJSONKeys(b *testing.B) {
	SetRemovedNames([]string{"team_id", "teams", "team", "team_name"}, nil, nil)
	typ := reflect.TypeFor[scanRequest]()
	item := `{"name": "item", "team_id": 3}`
	for _, size := range []struct {
		name  string
		items int
	}{{"small", 3}, {"1MiB", 1 << 15}} {
		body := []byte(`{"name": "a", "items": [` + strings.TrimSuffix(strings.Repeat(item+",", size.items), ",") + `]}`)
		b.Run(size.name, func(b *testing.B) {
			b.SetBytes(int64(len(body)))
			for b.Loop() {
				if err := RejectRemovedJSONKeys(body, typ); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestMakeDecoderRemovedNames(t *testing.T) {
	SetRemovedNames([]string{"team_id"}, []string{"team_id"}, nil)
	t.Cleanup(func() { SetRemovedNames(nil, nil, nil) })

	type request struct {
		FleetID uint `json:"fleet_id"`
		Page    int  `query:"page,optional"`
	}
	decode := func(limit int64, target string, body string) error {
		r := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
		_, err := MakeDecoder(request{}, defaultJSONUnmarshal, nil, nil, nil, nil, limit)(t.Context(), r)
		return err
	}

	rke, ok := errors.AsType[*RemovedKeyError](decode(1<<20, "/", `{"team_id": 1}`))
	require.True(t, ok)
	require.Equal(t, "team_id", rke.Key)
	rke, ok = errors.AsType[*RemovedKeyError](decode(1<<20, "/?team_id=1", `{"fleet_id": 1}`))
	require.True(t, ok)
	require.Equal(t, "team_id", rke.Key)

	require.NoError(t, decode(1<<20, "/?page=1", `{"fleet_id": 1}`))
	// A body without a size limit (agent uploads) is not buffered or scanned.
	require.NoError(t, decode(-1, "/", `{"team_id": 1}`))
	// Malformed JSON is still reported by the decoder, and an oversize body keeps its error.
	require.Error(t, decode(1<<20, "/", `{"team_id": `))
	_, ok = errors.AsType[platform_http.PayloadTooLargeError](decode(8, "/", `{"fleet_id": 1, "x": "0123456789"}`))
	require.True(t, ok)
}

func TestSetRemovedNamesResetsDeclaredQueryCache(t *testing.T) {
	t.Cleanup(func() { SetRemovedNames(nil, nil, nil) })
	query := url.Values{"team_id": {"1"}}
	typ := reflect.TypeFor[handParsedRequest]()

	SetRemovedNames(nil, []string{"team_id"}, map[string][]string{"host_options": {"team_id"}})
	require.NoError(t, RejectRemovedQueryParams(query, typ))
	SetRemovedNames(nil, []string{"team_id"}, nil)
	require.Error(t, RejectRemovedQueryParams(query, typ))
}
