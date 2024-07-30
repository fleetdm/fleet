package oval_parsed

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestEvalKernel(t *testing.T) {
	r := NewUbuntuResult()
	r.AddDefinition(Definition{
		Criteria: &Criteria{
			Operator:   And,
			Criteriums: []int{100, 200},
		},
		Vulnerabilities: []string{"CVE-2019-1234"},
	})
	r.AddUnameTest(100, &UnixUnameTest{
		States: []ObjectStateString{
			NewObjectStateString("less than", "0:5.15.0-1004"),
		},
	})
	r.AddUnameTest(200, &UnixUnameTest{
		States: []ObjectStateString{
			NewObjectStateString("pattern match", `5.15.0-\d+(-generic|-generic-64k|-generic-lpae|-lowlatency|-lowlatency-64k)`),
		},
	})

	software := []fleet.Software{
		{ID: 1, Name: "linux-image-5.15.0-1003-generic"},
		{ID: 2, Name: "linux-image-5.15.0-1004-generic"},
		{ID: 3, Name: "linux-image-5.15.0-1005-generic"},
		{ID: 4, Name: "linux-image-5.15.0-1003-lowlatency"},
		{ID: 5, Name: "linux-image-5.15.0-1004-foo"},
		{ID: 6, Name: "linux-image-4.0.0-10-generic"},
		{ID: 7, Name: "linux-image-6.0.0-10-generic"},
	}

	vuln, err := r.EvalKernel(software)
	require.NoError(t, err)
	require.ElementsMatch(t, vuln, []fleet.SoftwareVulnerability{
		{SoftwareID: 1, CVE: "CVE-2019-1234"},
		{SoftwareID: 4, CVE: "CVE-2019-1234"},
	})
}
