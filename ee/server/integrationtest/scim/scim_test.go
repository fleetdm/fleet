package scim

import (
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
)

func TestSCIM(t *testing.T) {
	s := SetUpSuite(t, "integrationtest.SCIM")

	cases := []struct {
		name string
		fn   func(t *testing.T, s *Suite)
	}{
		{"Auth", testAuth},
		{"BaseEndpoints", testBaseEndpoints},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, s.DS, tablesToTruncate...)
			c.fn(t, s)
		})
	}
}

var tablesToTruncate = []string{"host_scim_user", "scim_users", "scim_groups"}

func testAuth(t *testing.T, s *Suite) {
	t.Cleanup(func() {
		s.Token = s.GetTestAdminToken(t)
	})

	// Unauthenticated
	s.Token = "bozo"
	var resp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Schemas"), nil, http.StatusUnauthorized, &resp)
	assert.Contains(t, resp["detail"], "Authentication")
	assert.EqualValues(t, resp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})

	// Unauthorized
	resp = nil
	s.Token = s.GetTestToken(t, service.TestObserverUserEmail, test.GoodPassword)
	s.DoJSON(t, "GET", scimPath("/Schemas"), nil, http.StatusForbidden, &resp)
	assert.Contains(t, resp["detail"], "forbidden")
	assert.EqualValues(t, resp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})

	// Authorized
	resp = nil
	s.Token = s.GetTestToken(t, service.TestMaintainerUserEmail, test.GoodPassword)
	s.DoJSON(t, "GET", scimPath("/Schemas"), nil, http.StatusOK, &resp)
	assert.EqualValues(t, resp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"})
}

func testBaseEndpoints(t *testing.T, s *Suite) {
	var resp map[string]interface{}
	s.DoJSON(t, "GET", scimPath("/Schemas"), nil, http.StatusOK, &resp)
}

// func responseAsJSON(t *testing.T, resp map[string]interface{}) {
// 	formattedResp, err := json.MarshalIndent(resp, "", "  ")
// 	require.NoError(t, err)
// 	t.Logf("formatted resp: %s", string(formattedResp))
// }

func scimPath(suffix string) string {
	paths := []string{"/api/v1/fleet/scim", "/api/latest/fleet/scim"}
	prefix := paths[time.Now().UnixNano()%int64(len(paths))]
	return prefix + suffix
}
