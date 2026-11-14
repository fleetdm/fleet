package endpointer

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type removedKeysInner struct {
	RequireAll bool `json:"macos_require_all_software" removedkey:"require_all_software_macos"`
	Scoped     bool `json:"new_scoped" removedkey:"old_scoped" renamescope:"setup"`
	Ignored    bool `json:"-" removedkey:"never_registered"`
}

type removedKeysRequest struct {
	NewID     uint              `json:"new_id" removedkey:"old_id,oldid"`
	FleetName string            `json:"fleet_name" removedkey:"team_name"`
	Setup     *removedKeysInner `json:"setup"`
	Settings  []removedKeysInner
	Name      string `json:"name" renameto:"display_name"`
}

func TestExtractRemovedKeyRules(t *testing.T) {
	require.Nil(t, ExtractRemovedKeyRules(nil))
	require.Nil(t, ExtractRemovedKeyRules("not a struct"))

	rules := ExtractRemovedKeyRules(&removedKeysRequest{})
	require.ElementsMatch(t, []RemovedKeyRule{
		{Key: "old_id"},
		{Key: "oldid"},
		{Key: "team_name"},
		{Key: "require_all_software_macos"},
		{Key: "old_scoped", Scope: []string{"setup"}},
	}, rules, "found through pointers and slices, deduplicated, json:\"-\" skipped")

	// renameto rules are unaffected by removedkey tags.
	require.Equal(t, []AliasRule{{OldKey: "name", NewKey: "display_name"}}, ExtractAliasRules(removedKeysRequest{}))
}

func TestCheckRemovedKeys(t *testing.T) {
	rules := ExtractRemovedKeyRules(removedKeysRequest{})

	for _, tc := range []struct {
		name        string
		body        string
		wantKey     string
		wantInvalid bool
	}{
		{name: "top-level key", body: `{"old_id": 1}`, wantKey: "old_id"},
		{name: "nested key", body: `{"setup": {"require_all_software_macos": true}}`, wantKey: "require_all_software_macos"},
		{name: "inside an array", body: `{"x": [{"oldid": null}]}`, wantKey: "oldid"},
		{name: "old and new together", body: `{"new_id": 1, "old_id": 1}`, wantKey: "old_id"},
		{name: "scoped key in scope", body: `{"setup": {"old_scoped": true}}`, wantKey: "old_scoped"},
		{name: "different case", body: `{"Old_ID": 1}`, wantKey: "Old_ID"},
		{name: "scoped key in a differently cased scope", body: `{"SETUP": {"OLD_SCOPED": true}}`, wantKey: "OLD_SCOPED"},
		{name: "old name as a string value", body: `{"name": "old_id", "list": ["require_all_software_macos"]}`},
		{name: "scoped key out of scope", body: `{"other": {"old_scoped": true}}`},
		{name: "new names only", body: `{"new_id": 1, "setup": {"macos_require_all_software": true}}`},
		{name: "malformed JSON", body: `{"new_id": `, wantInvalid: true},
		{name: "invalid UTF-8 before a removed key", body: "{\"name\": \"\xff\", \"old_id\": 1}", wantInvalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckRemovedKeys([]byte(tc.body), rules)
			if tc.wantInvalid {
				_, removed := errors.AsType[*RemovedKeyError](err)
				require.Error(t, err)
				require.False(t, removed, "got %v", err)
				return
			}
			if tc.wantKey == "" {
				require.NoError(t, err)
				return
			}
			rke, ok := err.(*RemovedKeyError)
			require.True(t, ok, "got %v", err)
			require.Equal(t, tc.wantKey, rke.Key)
			require.Equal(t, `The "`+tc.wantKey+`" parameter is not available anymore.`, rke.Error())
		})
	}

	// The message comes from SetRemovedKeyMessage when it's set.
	prev := removedKeyMessage.Load()
	t.Cleanup(func() { removedKeyMessage.Store(prev) })
	SetRemovedKeyMessage(func(key string) string { return "gone: " + key })
	require.EqualError(t, CheckRemovedKeys([]byte(`{"team_name": "x"}`), rules), "gone: team_name")

	require.NoError(t, CheckRemovedKeys([]byte(`{"old_id": 1}`), nil))
}

func TestMakeDecoderRejectsRemovedKeys(t *testing.T) {
	decode := MakeDecoder(removedKeysRequest{}, defaultJSONUnmarshal, nil, nil, nil, nil, -1)

	r := httptest.NewRequest("PATCH", "/", strings.NewReader(`{"display_name": "a", "setup": {"require_all_software_macos": false}}`))
	_, err := decode(t.Context(), r)
	require.Error(t, err)

	recorder := httptest.NewRecorder()
	EncodeError(t.Context(), err, recorder, nil)
	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
	var body JsonError
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "Validation Failed", body.Message)
	require.Equal(t, []map[string]string{{
		"name":   "require_all_software_macos",
		"reason": `The "require_all_software_macos" parameter is not available anymore.`,
	}}, body.Errors)

	// New names still decode, and renameto aliases keep working next to removed keys.
	r = httptest.NewRequest("PATCH", "/", strings.NewReader(`{"display_name": "a", "new_id": 2, "setup": {"macos_require_all_software": true}}`))
	got, err := decode(t.Context(), r)
	require.NoError(t, err)
	req := got.(*removedKeysRequest)
	require.Equal(t, "a", req.Name)
	require.EqualValues(t, 2, req.NewID)
	require.True(t, req.Setup.RequireAll)
}
