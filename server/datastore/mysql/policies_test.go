package mysql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicies(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"NewGlobalPolicy", testPoliciesNewGlobalPolicy},
		{"MembershipView", testPoliciesMembershipView},
		{"TeamPolicy", testTeamPolicy},
		{"PolicyQueriesForHost", testPolicyQueriesForHost},
		{"TeamPolicyTransfer", testTeamPolicyTransfer},
		{"ApplyPolicySpec", testApplyPolicySpec},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testPoliciesNewGlobalPolicy(t *testing.T, ds *Datastore) {
	q, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query1",
		Description: "query1 desc",
		Query:       "select 1;",
		Saved:       true,
	})
	require.NoError(t, err)
	p, err := ds.NewGlobalPolicy(context.Background(), q.ID, "")
	require.NoError(t, err)

	assert.Equal(t, "query1", p.QueryName)

	q2, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query2",
		Description: "query2 desc",
		Query:       "select 42;",
		Saved:       true,
	})
	require.NoError(t, err)
	_, err = ds.NewGlobalPolicy(context.Background(), q2.ID, "")
	require.NoError(t, err)

	policies, err := ds.ListGlobalPolicies(context.Background())
	require.NoError(t, err)
	require.Len(t, policies, 2)
	assert.Equal(t, q.ID, policies[0].QueryID)
	assert.Equal(t, q2.ID, policies[1].QueryID)

	// Cannot delete a query if it's in a policy
	require.Error(t, ds.DeleteQuery(context.Background(), q.Name))

	_, err = ds.DeleteGlobalPolicies(context.Background(), []uint{policies[0].ID, policies[1].ID})
	require.NoError(t, err)

	policies, err = ds.ListGlobalPolicies(context.Background())
	require.NoError(t, err)
	require.Len(t, policies, 0)

	// But you can delete the query if the policy is gone
	require.NoError(t, ds.DeleteQuery(context.Background(), q.Name))
}

func testPoliciesMembershipView(t *testing.T, ds *Datastore) {
	host1, err := ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID:   "1234",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.NoError(t, err)

	host2, err := ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID:   "5679",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "2",
		UUID:            "2",
		Hostname:        "bar.local",
	})
	require.NoError(t, err)

	q, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query1",
		Description: "query1 desc",
		Query:       "select 1;",
		Saved:       true,
	})
	require.NoError(t, err)
	p, err := ds.NewGlobalPolicy(context.Background(), q.ID, "")
	require.NoError(t, err)

	q2, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query2",
		Description: "query2 desc",
		Query:       "select 42;",
		Saved:       true,
	})
	require.NoError(t, err)
	p2, err := ds.NewGlobalPolicy(context.Background(), q2.ID, "")
	require.NoError(t, err)

	assert.Equal(t, "query1", p.QueryName)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host1, map[uint]*bool{p.ID: ptr.Bool(true)}, time.Now()))
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host1, map[uint]*bool{p.ID: ptr.Bool(true)}, time.Now()))

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host2, map[uint]*bool{p.ID: nil}, time.Now()))
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host2, map[uint]*bool{p.ID: ptr.Bool(false)}, time.Now()))
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host2, map[uint]*bool{p.ID: ptr.Bool(true)}, time.Now()))

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host2, map[uint]*bool{p2.ID: nil}, time.Now()))

	policies, err := ds.ListGlobalPolicies(context.Background())
	require.NoError(t, err)
	require.Len(t, policies, 2)

	assert.Equal(t, uint(2), policies[0].PassingHostCount)
	assert.Equal(t, uint(0), policies[0].FailingHostCount)

	assert.Equal(t, uint(0), policies[1].PassingHostCount)
	assert.Equal(t, uint(0), policies[1].FailingHostCount)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host1, map[uint]*bool{p.ID: ptr.Bool(false)}, time.Now()))
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host2, map[uint]*bool{p2.ID: ptr.Bool(false)}, time.Now()))

	policies, err = ds.ListGlobalPolicies(context.Background())
	require.NoError(t, err)
	require.Len(t, policies, 2)

	assert.Equal(t, uint(1), policies[0].PassingHostCount)
	assert.Equal(t, uint(1), policies[0].FailingHostCount)

	assert.Equal(t, uint(0), policies[1].PassingHostCount)
	assert.Equal(t, uint(1), policies[1].FailingHostCount)

	policy, err := ds.Policy(context.Background(), policies[0].ID)
	require.NoError(t, err)
	assert.Equal(t, policies[0], policy)

	queries, err := ds.PolicyQueriesForHost(context.Background(), host1)
	require.NoError(t, err)
	require.Len(t, queries, 2)
	assert.Equal(t, q.Query, queries[fmt.Sprint(q.ID)])
	assert.Equal(t, q2.Query, queries[fmt.Sprint(q2.ID)])
}

func testTeamPolicy(t *testing.T, ds *Datastore) {
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	q, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query1",
		Description: "query1 desc",
		Query:       "select 1;",
		Saved:       true,
	})
	require.NoError(t, err)

	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	q2, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query2",
		Description: "query2 desc",
		Query:       "select 1;",
		Saved:       true,
	})
	require.NoError(t, err)

	prevPolicies, err := ds.ListGlobalPolicies(context.Background())
	require.NoError(t, err)

	_, err = ds.NewTeamPolicy(context.Background(), 99999999, q.ID, "")
	require.Error(t, err)

	p, err := ds.NewTeamPolicy(context.Background(), team1.ID, q.ID, "some resolution")
	require.NoError(t, err)

	assert.Equal(t, "query1", p.QueryName)
	require.NotNil(t, p.Resolution)
	assert.Equal(t, "some resolution", *p.Resolution)

	globalPolicies, err := ds.ListGlobalPolicies(context.Background())
	require.NoError(t, err)
	require.Len(t, globalPolicies, len(prevPolicies))

	_, err = ds.NewTeamPolicy(context.Background(), team2.ID, q2.ID, "")
	require.NoError(t, err)

	teamPolicies, err := ds.ListTeamPolicies(context.Background(), team1.ID)
	require.NoError(t, err)
	require.Len(t, teamPolicies, 1)
	assert.Equal(t, q.ID, teamPolicies[0].QueryID)

	team2Policies, err := ds.ListTeamPolicies(context.Background(), team2.ID)
	require.NoError(t, err)
	require.Len(t, team2Policies, 1)
	assert.Equal(t, q2.ID, team2Policies[0].QueryID)

	_, err = ds.DeleteTeamPolicies(context.Background(), team1.ID, []uint{teamPolicies[0].ID})
	require.NoError(t, err)

	teamPolicies, err = ds.ListGlobalPolicies(context.Background())
	require.NoError(t, err)
	require.Len(t, teamPolicies, 0)
}

func testPolicyQueriesForHost(t *testing.T, ds *Datastore) {
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	host1, err := ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID:   "1234",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.NoError(t, err)

	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host1.ID}))
	host1, err = ds.Host(context.Background(), host1.ID)
	require.NoError(t, err)

	host2, err := ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID:   "5679",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "2",
		UUID:            "2",
		Hostname:        "bar.local",
	})
	require.NoError(t, err)

	q, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query1",
		Description: "query1 desc",
		Query:       "select 1;",
		Saved:       true,
	})
	require.NoError(t, err)
	gp, err := ds.NewGlobalPolicy(context.Background(), q.ID, "")
	require.NoError(t, err)

	q2, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query2",
		Description: "query2 desc",
		Query:       "select 42;",
		Saved:       true,
	})
	require.NoError(t, err)
	tp, err := ds.NewTeamPolicy(context.Background(), team1.ID, q2.ID, "")
	require.NoError(t, err)

	queries, err := ds.PolicyQueriesForHost(context.Background(), host1)
	require.NoError(t, err)
	require.Len(t, queries, 2)
	assert.Equal(t, q.Query, queries[fmt.Sprint(q.ID)])
	assert.Equal(t, q2.Query, queries[fmt.Sprint(q2.ID)])

	queries, err = ds.PolicyQueriesForHost(context.Background(), host2)
	require.NoError(t, err)
	require.Len(t, queries, 1)
	assert.Equal(t, q.Query, queries[fmt.Sprint(q.ID)])

	require.NoError(t, ds.RecordPolicyQueryExecutions(
		context.Background(), host1, map[uint]*bool{tp.ID: ptr.Bool(false), gp.ID: nil}, time.Now()))

	policies, err := ds.ListPoliciesForHost(context.Background(), host1.ID)
	require.NoError(t, err)
	require.Len(t, policies, 2)

	policies, err = ds.ListPoliciesForHost(context.Background(), host2.ID)
	require.NoError(t, err)
	require.Len(t, policies, 0)

	require.NoError(t, ds.RecordPolicyQueryExecutions(
		context.Background(), host2, map[uint]*bool{gp.ID: nil}, time.Now()))

	policies, err = ds.ListPoliciesForHost(context.Background(), host2.ID)
	require.NoError(t, err)
	require.Len(t, policies, 1)
}

func testTeamPolicyTransfer(t *testing.T, ds *Datastore) {
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: t.Name() + "team2"})
	require.NoError(t, err)

	host1, err := ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID:   "1234",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.NoError(t, err)

	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host1.ID}))
	host1, err = ds.Host(context.Background(), host1.ID)
	require.NoError(t, err)

	q, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query1",
		Description: "query1 desc",
		Query:       "select 1;",
		Saved:       true,
	})
	require.NoError(t, err)
	teamPolicy, err := ds.NewTeamPolicy(context.Background(), team1.ID, q.ID, "")
	require.NoError(t, err)

	globalPolicy, err := ds.NewGlobalPolicy(context.Background(), q.ID, "")
	require.NoError(t, err)

	require.NoError(t, ds.RecordPolicyQueryExecutions(
		context.Background(), host1, map[uint]*bool{teamPolicy.ID: ptr.Bool(false), globalPolicy.ID: ptr.Bool(true)}, time.Now()))
	require.NoError(t, ds.RecordPolicyQueryExecutions(
		context.Background(), host1, map[uint]*bool{teamPolicy.ID: ptr.Bool(true), globalPolicy.ID: ptr.Bool(true)}, time.Now()))

	checkPassingCount := func(expectedCount uint) {
		policies, err := ds.ListTeamPolicies(context.Background(), team1.ID)
		require.NoError(t, err)
		require.Len(t, policies, 1)

		assert.Equal(t, expectedCount, policies[0].PassingHostCount)

		policies, err = ds.ListGlobalPolicies(context.Background())
		require.NoError(t, err)
		require.Len(t, policies, 1)
		assert.Equal(t, uint(1), policies[0].PassingHostCount)

		policies, err = ds.ListTeamPolicies(context.Background(), team2.ID)
		require.NoError(t, err)
		require.Len(t, policies, 0)
	}

	checkPassingCount(1)

	require.NoError(t, ds.AddHostsToTeam(context.Background(), ptr.Uint(team2.ID), []uint{host1.ID}))

	checkPassingCount(0)
}

func testApplyPolicySpec(t *testing.T, ds *Datastore) {
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	q, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query1",
		Description: "query1 desc",
		Query:       "select 1;",
		Saved:       true,
	})
	require.NoError(t, err)

	q2, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query2",
		Description: "query2 desc",
		Query:       "select 1;",
		Saved:       true,
	})
	require.NoError(t, err)

	require.NoError(t, ds.ApplyPolicySpecs(context.Background(), []*fleet.PolicySpec{
		{
			QueryName:  "query1",
			Resolution: "some resolution",
		},
		{
			QueryName:  "query2",
			Resolution: "some other resolution",
			Team:       "team1",
		},
		{
			QueryName: "query1",
			Team:      "team1",
		},
	}))

	policies, err := ds.ListGlobalPolicies(context.Background())
	require.NoError(t, err)
	require.Len(t, policies, 1)
	assert.Equal(t, q.ID, policies[0].QueryID)
	require.NotNil(t, policies[0].Resolution)
	assert.Equal(t, "some resolution", *policies[0].Resolution)

	teamPolicies, err := ds.ListTeamPolicies(context.Background(), team1.ID)
	require.NoError(t, err)
	require.Len(t, teamPolicies, 2)
	assert.Equal(t, q2.ID, teamPolicies[0].QueryID)
	require.NotNil(t, teamPolicies[0].Resolution)
	assert.Equal(t, "some other resolution", *teamPolicies[0].Resolution)

	assert.Equal(t, q.ID, teamPolicies[1].QueryID)
	require.NotNil(t, teamPolicies[1].Resolution)
	assert.Equal(t, "", *teamPolicies[1].Resolution)

	require.Error(t, ds.ApplyPolicySpecs(context.Background(), []*fleet.PolicySpec{
		{
			QueryName: "query13",
		},
	}))

	require.Error(t, ds.ApplyPolicySpecs(context.Background(), []*fleet.PolicySpec{
		{
			Team: "team1",
		},
	}))

	require.Error(t, ds.ApplyPolicySpecs(context.Background(), []*fleet.PolicySpec{
		{
			QueryName: "query123",
			Team:      "team1",
		},
	}))

	// Make sure apply is idempotent
	require.NoError(t, ds.ApplyPolicySpecs(context.Background(), []*fleet.PolicySpec{
		{
			QueryName:  "query1",
			Resolution: "some resolution",
		},
		{
			QueryName:  "query2",
			Resolution: "some other resolution",
			Team:       "team1",
		},
		{
			QueryName: "query1",
			Team:      "team1",
		},
	}))

	policies, err = ds.ListGlobalPolicies(context.Background())
	require.NoError(t, err)
	require.Len(t, policies, 1)
}
