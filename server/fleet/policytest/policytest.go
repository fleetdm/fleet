package policytest

import (
	"fmt"
	"sort"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func RunFailing1000hosts(t *testing.T, r fleet.FailingPolicySet) {
	hosts := make([]fleet.PolicySetHost, 1000)
	for i := range hosts {
		hosts[i] = fleet.PolicySetHost{
			ID:       uint(i + 1), //nolint:gosec // dismiss G115
			Hostname: fmt.Sprintf("test.hostname.%d", i+1),
		}
	}

	policyID1k := uint(9999)
	for _, h := range hosts {
		err := r.AddHost(policyID1k, h)
		require.NoError(t, err)
	}

	fetchedHosts, err := r.ListHosts(policyID1k)
	require.NoError(t, err)
	require.Len(t, fetchedHosts, len(hosts))

	sort.Slice(fetchedHosts, func(i, j int) bool {
		return fetchedHosts[i].ID < fetchedHosts[j].ID
	})
	require.Equal(t, hosts, fetchedHosts)
	err = r.RemoveHosts(policyID1k, hosts)
	require.NoError(t, err)
	fetchedHosts, err = r.ListHosts(policyID1k)
	require.NoError(t, err)
	require.Empty(t, fetchedHosts)
}

func RunFailingBasic(t *testing.T, r fleet.FailingPolicySet) {
	policyID1 := uint(1)

	// Test listing policy sets with no sets.
	policyIDs, err := r.ListSets()
	require.NoError(t, err)
	require.Empty(t, policyIDs)

	// Test listing if the policy set doesn't exist.
	hostIDs, err := r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Empty(t, hostIDs)

	// Test removing hosts if the set doesn't exist.
	hostx := fleet.PolicySetHost{
		ID:       uint(999),
		Hostname: "hostx.example",
	}
	err = r.RemoveHosts(policyID1, []fleet.PolicySetHost{hostx})
	require.NoError(t, err)

	// Remove no hosts.
	err = r.RemoveHosts(policyID1, []fleet.PolicySetHost{})
	require.NoError(t, err)

	// Test adding a new host to a policy set.
	host2 := fleet.PolicySetHost{
		ID:       uint(2),
		Hostname: "host2.example",
	}
	err = r.AddHost(policyID1, host2)
	require.NoError(t, err)

	// Test listing the created policy set.
	policyIDs, err = r.ListSets()
	require.NoError(t, err)
	require.Len(t, policyIDs, 1)
	require.Equal(t, policyID1, policyIDs[0])

	// Test listing the policy set.
	hostIDs, err = r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Len(t, hostIDs, 1)
	require.Equal(t, host2, hostIDs[0])

	// Test adding a second host to the policy set.
	host3 := fleet.PolicySetHost{
		ID:       uint(3),
		Hostname: "host3.example",
	}
	err = r.AddHost(policyID1, host3)
	require.NoError(t, err)

	// Test adding a shared host on a different policy set.
	policyID2 := uint(2)
	err = r.AddHost(policyID2, host2)
	require.NoError(t, err)

	// Test listing the newly created policy set.
	policyIDs, err = r.ListSets()
	require.NoError(t, err)
	require.Len(t, policyIDs, 2)
	sort.Slice(policyIDs, func(i, j int) bool {
		return policyIDs[i] < policyIDs[j]
	})
	require.Equal(t, policyID1, policyIDs[0])
	require.Equal(t, policyID2, policyIDs[1])

	// Test listing of first and second policy set.
	hostIDs, err = r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Len(t, hostIDs, 2)
	sort.Slice(hostIDs, func(i, j int) bool {
		return hostIDs[i].ID < hostIDs[j].ID
	})
	require.Equal(t, host2, hostIDs[0])
	require.Equal(t, host3, hostIDs[1])
	hostIDs, err = r.ListHosts(policyID2)
	require.NoError(t, err)
	require.Len(t, hostIDs, 1)
	require.Equal(t, host2, hostIDs[0])

	// Test removing all hosts from the first policy set.
	err = r.RemoveHosts(policyID1, []fleet.PolicySetHost{host2, host3})
	require.NoError(t, err)

	// Test listing of first and second policy set (after some removal).
	hostIDs, err = r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Empty(t, hostIDs)
	hostIDs, err = r.ListHosts(policyID2)
	require.NoError(t, err)
	require.Len(t, hostIDs, 1)
	require.Equal(t, host2, hostIDs[0])

	// Test removal of a host on the second policy set.
	err = r.RemoveHosts(policyID2, []fleet.PolicySetHost{host2})
	require.NoError(t, err)

	// Test listing of the second policy set.
	hostIDs, err = r.ListHosts(policyID2)
	require.NoError(t, err)
	require.Empty(t, hostIDs)

	// Add another host to the first policy set (after removal of the other two).
	host4 := fleet.PolicySetHost{
		ID:       uint(4),
		Hostname: "host4.example",
	}
	err = r.AddHost(policyID1, host4)
	require.NoError(t, err)

	// Test listing of the first policy set.
	hostIDs, err = r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Len(t, hostIDs, 1)
	require.Equal(t, host4, hostIDs[0])

	// Test removal of the first policy set.
	err = r.RemoveSet(policyID1)
	require.NoError(t, err)

	// Test listing of a removed policy set.
	hostIDs, err = r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Empty(t, hostIDs)

	// Test a second removal of the first policy set.
	err = r.RemoveSet(policyID1)
	require.NoError(t, err)

	// Test removing hosts if the set doesn't exist anymore.
	err = r.RemoveHosts(policyID1, []fleet.PolicySetHost{host4})
	require.NoError(t, err)

	// Test removal of an unexisting policy set.
	policyIDX := uint(999)
	err = r.RemoveSet(policyIDX)
	require.NoError(t, err)

	// Test listing policy still returns the policyID2 set.
	policyIDs, err = r.ListSets()
	require.NoError(t, err)
	require.Len(t, policyIDs, 1)
	require.Equal(t, policyID2, policyIDs[0])

	// Now remove the remaining set.
	err = r.RemoveSet(policyID2)
	require.NoError(t, err)

	// And now it should be empty.
	policyIDs, err = r.ListSets()
	require.NoError(t, err)
	require.Empty(t, policyIDs)
}
