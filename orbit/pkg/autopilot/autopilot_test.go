package autopilot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// cachedProfile is the shape of PolicyJsonCache on a device that downloaded an assigned Autopilot profile.
const cachedProfile = `{"CloudAssignedTenantDomain":"contoso.onmicrosoft.com","DeploymentProfileName":"Autopilot",` +
	`"AadDeviceId":"b5b5715b-73e8-4c03-a6b5-89a121c39d3c","CloudAssignedTenantId":"5b1fc5b6-9502-4cf9-90cf-d0b656eaf7a4",` +
	`"ZtdRegistrationId":"EFDB13F9-44D6-4F99-A93F-08833FCCEF82","AutopilotMode":0,` +
	`"CloudAssignedAadServerData":"{\"ZeroTouchConfig\":{\"CloudAssignedTenantDomain\":\"contoso.onmicrosoft.com\",\"ForcedEnrollment\":1}}"}`

// emptyProfile is what a registered device without an assigned profile caches.
const emptyProfile = `{"AutopilotCreationDate":"2026-10-09T18:18:21Z","AutopilotMode":0,` +
	`"CloudAssignedAadServerData":"{\"ZeroTouchConfig\":{\"CloudAssignedTenantDomain\":\"\",\"ForcedEnrollment\":0}}"}`

func TestParseRegistrationID(t *testing.T) {
	for _, tc := range []struct {
		name       string
		policyJSON string
		want       string
		wantErr    bool
	}{
		{name: "assigned profile, normalized to lowercase", policyJSON: cachedProfile, want: "efdb13f9-44d6-4f99-a93f-08833fccef82"},
		{name: "no assigned profile", policyJSON: emptyProfile, want: ""},
		{name: "empty registration id", policyJSON: `{"ZtdRegistrationId":""}`, want: ""},
		{name: "registration id is not a GUID", policyJSON: `{"ZtdRegistrationId":"not-a-guid"}`, wantErr: true},
		{name: "registration id is not a string", policyJSON: `{"ZtdRegistrationId":42}`, wantErr: true},
		{name: "not JSON", policyJSON: `garbage`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseRegistrationID(tc.policyJSON)
			if tc.wantErr {
				require.Error(t, err)
				require.Empty(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
