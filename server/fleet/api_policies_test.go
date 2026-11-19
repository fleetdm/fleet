package fleet

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPolicyResponsesOmitAutomationBooleans(t *testing.T) {
	policy := &Policy{PolicyData: PolicyData{ID: 1, Name: "p", CalendarEventsEnabled: true}}

	for name, resp := range map[string]any{
		"get":    GetPolicyByIDResponse{Policy: policy},
		"create": CreatePolicyResponse{Policy: policy},
		"modify": ModifyPolicyResponse{Policy: policy},
		"list":   ListPoliciesResponse{Policies: []*Policy{policy}},
	} {
		t.Run(name, func(t *testing.T) {
			b, err := json.Marshal(resp)
			require.NoError(t, err)
			var body map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(b, &body))
			raw := body["policy"]
			if name == "list" {
				var list []json.RawMessage
				require.NoError(t, json.Unmarshal(body["policies"], &list))
				require.Len(t, list, 1)
				raw = list[0]
			}
			var got map[string]any
			require.NoError(t, json.Unmarshal(raw, &got))
			require.Equal(t, "p", got["name"])
			require.Equal(t, string(PolicyAutomationCalendar), got["automation_type"])
			require.NotContains(t, got, "calendar_events_enabled")
			require.NotContains(t, got, "conditional_access_enabled")
		})
	}

	// Host policy lists and the failing policies webhook marshal Policy itself, which keeps them.
	b, err := json.Marshal(policy)
	require.NoError(t, err)
	require.Contains(t, string(b), `"calendar_events_enabled":true`)
	require.Contains(t, string(b), `"conditional_access_enabled":false`)

	b, err = json.Marshal(ListPoliciesResponse{})
	require.NoError(t, err)
	require.JSONEq(t, `{"policies": null}`, string(b))
}
