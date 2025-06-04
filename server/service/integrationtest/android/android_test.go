package android

import (
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/stretchr/testify/assert"
)

func TestAndroid(t *testing.T) {
	s := SetUpSuite(t, "integrationtest.Android")

	cases := []struct {
		name string
		fn   func(t *testing.T, s *Suite)
	}{
		{"HappyPath", testHappyPath},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, s.DS)
			c.fn(t, s)
		})
	}
}

func testHappyPath(t *testing.T, s *Suite) {
	signupDetails := expectSignupDetails(t, s)
	var signupURL android.EnterpriseSignupResponse
	s.DoJSON(t, "GET", "/api/v1/fleet/android_enterprise/signup_url", nil, http.StatusOK, &signupURL)
	assert.Equal(t, signupURL.Url, signupDetails.Url)
}

func expectSignupDetails(t *testing.T, s *Suite) *android.SignupDetails {
	signupDetails := &android.SignupDetails{
		Url:  "URL",
		Name: "Name",
	}
	s.AndroidProxy.SignupURLsCreateFunc = func(callbackURL string) (*android.SignupDetails, error) {
		// We will need to extract the security token from the callbackURL for further testing
		assert.Contains(t, callbackURL, "/api/v1/fleet/android_enterprise/connect/")
		return signupDetails, nil
	}
	return signupDetails
}
