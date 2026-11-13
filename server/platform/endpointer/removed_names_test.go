package endpointer

import (
	"cmp"
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
	OldKept  *uint               `query:"team_id,optional"` //nolint:apiparamcheck // a surface that has not been renamed yet
}

// requireRejected checks that err is a *RemovedKeyError for wantKey, or nil when wantKey is empty.
type columnSortRequest struct {
	OrderKey string `query:"order_key,optional"`
}

func (columnSortRequest) SortsByAnyColumn() {}

func requireRejected(t *testing.T, err error, wantKey string) {
	t.Helper()
	if wantKey == "" {
		require.NoError(t, err)
		return
	}
	rke, ok := errors.AsType[*RemovedKeyError](err)
	require.True(t, ok, "got %v", err)
	require.Equal(t, wantKey, rke.Key)
}

func TestRejectRemovedJSONKeys(t *testing.T) {
	SetRemovedNames([]string{"team_id", "teams", "Team"}, nil, nil, nil)
	t.Cleanup(func() { SetRemovedNames(nil, nil, nil, nil) })
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

		{name: "new names only, one through an embedded struct", body: `{"fleet_id": 1, "name": "a"}`},
		{name: "a removed name that the type still declares is accepted", body: `{"item": {"team_id": 3}, "items": [{"team_id": 3}]}`},
		{name: "free-form map is not looked into", body: `{"settings": {"teams": [1], "team_id": 2, "nested": {"Team": "x"}}}`},
		{name: "any is not looked into", body: `{"anything": {"team_id": 1}}`},
		{name: "raw JSON is not looked into", body: `{"raw": {"team_id": 1}}`},
		{name: "a type that decodes itself is not looked into", body: `{"own": {"team_id": 1}}`},
		{name: "a json:- field is not a name the type declares", body: `{"Skipped": "x"}`},
		{name: "maps are not looked into, their keys are data", body: `{"by_name": {"teams": {"teams": 1}}}`},
		{name: "removed names as values", body: `{"name": "team_id", "items": [{"name": "teams"}]}`},
		{name: "unknown keys that are not removed are left to the decoder", body: `{"unknown": {"team": 1}}`},
		{name: "an array body for an object type is left to the decoder", body: `[{"teams": 1}]`},
		{name: "malformed JSON is left to the decoder", body: `{"name": `},
		{name: "empty body", body: ``},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireRejected(t, RejectRemovedJSONKeys([]byte(tc.body), typ), tc.wantKey)
		})
	}
}

func TestRejectRemovedQueryParams(t *testing.T) {
	SetRemovedNames(nil, []string{"team_id", "team_name"}, []string{"team_name"}, nil)
	t.Cleanup(func() { SetRemovedNames(nil, nil, nil, nil) })

	type renamed struct {
		FleetID *uint `query:"fleet_id,optional"`
	}
	for _, tc := range []struct {
		name, query, wantKey, wantParam string
		typ                             reflect.Type // renamed when unset
	}{
		{name: "removed parameter", query: "team_id=1", wantKey: "team_id"},
		{name: "with the new name", query: "fleet_id=0&team_id=1", wantKey: "team_id"},
		{name: "new name only", query: "fleet_id=1"},
		{name: "parameter names are case sensitive, like the decoder", query: "Team_ID=1"},
		{name: "a parameter the type still declares", typ: reflect.TypeFor[scanRequest](), query: "team_id=1"},
		{name: "declared through an embedded struct", typ: reflect.TypeFor[scanRequest](), query: "fleet_id=1&team_name=x", wantKey: "team_name"},
		{name: "removed sort key", query: "order_key=team_name", wantKey: "team_name", wantParam: "order_key"},
		{name: "removed inherited sort key", query: "order_key=name&inherited_order_key=team_name", wantKey: "team_name", wantParam: "inherited_order_key"},
		{name: "sort keys are matched as values of the order key parameters only", query: "order_key=fleet_name&sort=team_name"},
		{name: "a type that sorts by any result column", typ: reflect.TypeFor[columnSortRequest](), query: "order_key=team_name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values, err := url.ParseQuery(tc.query)
			require.NoError(t, err)
			err = RejectRemovedQueryParams(values, cmp.Or(tc.typ, reflect.TypeFor[renamed]()))
			requireRejected(t, err, tc.wantKey)
			if tc.wantParam != "" {
				require.Equal(t, tc.wantParam, err.(*RemovedKeyError).Param)
			}
		})
	}
}

func TestMakeDecoderRemovedNames(t *testing.T) {
	SetRemovedNames([]string{"team_id"}, []string{"team_id"}, nil, []string{"/orbit/", "/device/"})
	t.Cleanup(func() { SetRemovedNames(nil, nil, nil, nil) })

	type request struct {
		FleetID uint `json:"fleet_id"`
		Page    int  `query:"page,optional"`
	}
	decode := func(limit int64, target, body string) error {
		r := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
		_, err := MakeDecoder(request{}, defaultJSONUnmarshal, nil, nil, nil, nil, limit)(t.Context(), r)
		return err
	}

	requireRejected(t, decode(1<<20, "/", `{"team_id": 1}`), "team_id")
	requireRejected(t, decode(1<<20, "/?team_id=1", `{"fleet_id": 1}`), "team_id")
	require.NoError(t, decode(1<<20, "/?page=1", `{"fleet_id": 1}`))
	// Agent routes are not scanned, in the query string or in the body.
	requireRejected(t, decode(1<<20, "/api/fleet/orbit/config?team_id=1", `{"team_id": 1}`), "")
	requireRejected(t, decode(1<<20, "/api/latest/fleet/device/abc/ping", `{"team_id": 1}`), "")
	// A body without a size limit is not buffered or scanned.
	require.NoError(t, decode(-1, "/", `{"team_id": 1}`))
	// A body over the scan size is decoded without a scan, and still decodes. The route's limit is above the scan size.
	big := `{"team_id": 1, "x": "` + strings.Repeat("a", maxRemovedNamesScanSize) + `"}`
	require.NoError(t, decode(2*maxRemovedNamesScanSize, "/", big))
	requireRejected(t, decode(2*maxRemovedNamesScanSize, "/", `{"team_id": 1, "x": "`+strings.Repeat("a", maxRemovedNamesScanSize-100)+`"}`), "team_id")
	// Malformed JSON is still reported by the decoder, and an oversize body keeps its error.
	require.Error(t, decode(1<<20, "/", `{"team_id": `))
	_, ok := errors.AsType[platform_http.PayloadTooLargeError](decode(8, "/", `{"fleet_id": 1, "x": "0123456789"}`))
	require.True(t, ok)
}
