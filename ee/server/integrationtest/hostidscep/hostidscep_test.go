package hostidscep

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
)

func TestHostIdentitySCEP(t *testing.T) {
	s := SetUpSuite(t, "integrationtest.HostIdentitySCEP.")

	cases := []struct {
		name string
		fn   func(t *testing.T, s *Suite)
	}{
		{"GetCert", testGetCert},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, s.BaseSuite.DS, []string{
				"host_identity_scep_serials", "host_identity_scep_certificates",
			}...)
			c.fn(t, s)
		})
	}
}

func testGetCert(t *testing.T, s *Suite) {
	// TODO
}
