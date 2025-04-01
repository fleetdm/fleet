package scim

import (
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/stretchr/testify/assert"
)

func TestSCIM(t *testing.T) {
	s := SetUpSuite(t, "integrationtest.SCIM")

	cases := []struct {
		name string
		fn   func(t *testing.T, s *Suite)
	}{
		{"Auth", testAuth},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, s.DS)
			c.fn(t, s)
		})
	}
}

func testAuth(t *testing.T, s *Suite) {
	t.Cleanup(func() {
		s.Token = s.GetTestAdminToken(t)
	})
	// s.Token = s.GetTestToken(t, s.Users[service.TestObserverUser].Email, test.GoodPassword)
	s.Token = "bozo"
	var resp map[string]interface{}
	s.DoJSON(t, "GET", "/api/v1/fleet/scim", nil, http.StatusUnauthorized, &resp)
	assert.NotNil(t, resp["detail"])
	assert.EqualValues(t, resp["schemas"], []interface{}{"urn:ietf:params:scim:api:messages:2.0:Error"})
}
