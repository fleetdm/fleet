package mysql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
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
		{"MembershipViewDeferred", func(t *testing.T, ds *Datastore) { testPoliciesMembershipView(true, t, ds) }},
		{"MembershipViewNotDeferred", func(t *testing.T, ds *Datastore) { testPoliciesMembershipView(false, t, ds) }},
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
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	q, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query1",
		Description: "query1 desc",
		Query:       "select 1;",
		Saved:       true,
	})
	require.NoError(t, err)
	p, err := ds.NewGlobalPolicy(context.Background(), user1.ID, q.ID, "", "", "", "")
	require.NoError(t, err)

	assert.Equal(t, "query1", p.Name)
	assert.Equal(t, "query1 desc", p.Description)
	assert.Equal(t, "select 1;", p.Query)
	require.NotNil(t, p.AuthorID)
	assert.Equal(t, user1.ID, *p.AuthorID)

	q2, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query2",
		Description: "query2 desc",
		Query:       "select 42;",
		Saved:       true,
	})
	require.NoError(t, err)
	_, err = ds.NewGlobalPolicy(context.Background(), user1.ID, q2.ID, "", "", "", "")
	require.NoError(t, err)

	policies, err := ds.ListGlobalPolicies(context.Background())
	require.NoError(t, err)
	require.Len(t, policies, 2)
	assert.Equal(t, q.Name, policies[0].Name)
	assert.Equal(t, q.Query, policies[0].Query)
	assert.Equal(t, q.Description, policies[0].Description)
	assert.Equal(t, q2.Name, policies[1].Name)
	assert.Equal(t, q2.Query, policies[1].Query)
	assert.Equal(t, q2.Description, policies[1].Description)
	require.NotNil(t, policies[1].AuthorID)
	assert.Equal(t, user1.ID, *policies[1].AuthorID)

	// The original query can be removed as the policy owns it's own query.
	require.NoError(t, ds.DeleteQuery(context.Background(), q.Name))

	_, err = ds.DeleteGlobalPolicies(context.Background(), []uint{policies[0].ID, policies[1].ID})
	require.NoError(t, err)

	policies, err = ds.ListGlobalPolicies(context.Background())
	require.NoError(t, err)
	require.Len(t, policies, 0)
}

func testPoliciesMembershipView(deferred bool, t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
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
	p, err := ds.NewGlobalPolicy(context.Background(), user1.ID, q.ID, "", "", "", "")
	require.NoError(t, err)

	assert.Equal(t, "query1", p.Name)
	assert.Equal(t, "select 1;", p.Query)
	assert.Equal(t, "query1 desc", p.Description)
	require.NotNil(t, p.AuthorID)
	assert.Equal(t, user1.ID, *p.AuthorID)

	q2, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query2",
		Description: "query2 desc",
		Query:       "select 42;",
		Saved:       true,
	})
	require.NoError(t, err)
	p2, err := ds.NewGlobalPolicy(context.Background(), user1.ID, q2.ID, "", "", "", "")
	require.NoError(t, err)

	assert.Equal(t, "query2", p2.Name)
	assert.Equal(t, "select 42;", p2.Query)
	assert.Equal(t, "query2 desc", p2.Description)
	require.NotNil(t, p2.AuthorID)
	assert.Equal(t, user1.ID, *p2.AuthorID)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host1, map[uint]*bool{p.ID: ptr.Bool(true)}, time.Now(), deferred))
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host1, map[uint]*bool{p.ID: ptr.Bool(true)}, time.Now(), deferred))

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host2, map[uint]*bool{p.ID: nil}, time.Now(), deferred))
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host2, map[uint]*bool{p.ID: ptr.Bool(false)}, time.Now(), deferred))
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host2, map[uint]*bool{p.ID: ptr.Bool(true)}, time.Now(), deferred))

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host2, map[uint]*bool{p2.ID: nil}, time.Now(), deferred))

	policies, err := ds.ListGlobalPolicies(context.Background())
	require.NoError(t, err)
	require.Len(t, policies, 2)

	assert.Equal(t, uint(2), policies[0].PassingHostCount)
	assert.Equal(t, uint(0), policies[0].FailingHostCount)

	assert.Equal(t, uint(0), policies[1].PassingHostCount)
	assert.Equal(t, uint(0), policies[1].FailingHostCount)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host1, map[uint]*bool{p.ID: ptr.Bool(false)}, time.Now(), deferred))
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host2, map[uint]*bool{p2.ID: ptr.Bool(false)}, time.Now(), deferred))

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
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
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

	_, err = ds.NewTeamPolicy(context.Background(), user1.ID, 99999999, q.ID, "", "", "", "")
	require.Error(t, err)

	p, err := ds.NewTeamPolicy(context.Background(), user1.ID, team1.ID, q.ID, "", "", "", "some resolution")
	require.NoError(t, err)

	assert.Equal(t, "query1", p.Name)
	assert.Equal(t, "select 1;", p.Query)
	assert.Equal(t, "query1 desc", p.Description)
	require.NotNil(t, p.AuthorID)
	assert.Equal(t, user1.ID, *p.AuthorID)

	require.NotNil(t, p.Resolution)
	assert.Equal(t, "some resolution", *p.Resolution)

	globalPolicies, err := ds.ListGlobalPolicies(context.Background())
	require.NoError(t, err)
	require.Len(t, globalPolicies, len(prevPolicies))

	p2, err := ds.NewTeamPolicy(context.Background(), user1.ID, team2.ID, q2.ID, "", "", "", "")
	require.NoError(t, err)

	assert.Equal(t, "query2", p2.Name)
	assert.Equal(t, "select 1;", p2.Query)
	assert.Equal(t, "query2 desc", p2.Description)
	require.NotNil(t, p2.AuthorID)
	assert.Equal(t, user1.ID, *p2.AuthorID)

	teamPolicies, err := ds.ListTeamPolicies(context.Background(), team1.ID)
	require.NoError(t, err)
	require.Len(t, teamPolicies, 1)
	assert.Equal(t, q.Name, teamPolicies[0].Name)
	assert.Equal(t, q.Query, teamPolicies[0].Query)
	assert.Equal(t, q.Description, teamPolicies[0].Description)
	require.NotNil(t, teamPolicies[0].AuthorID)
	require.Equal(t, user1.ID, *teamPolicies[0].AuthorID)

	team2Policies, err := ds.ListTeamPolicies(context.Background(), team2.ID)
	require.NoError(t, err)
	require.Len(t, team2Policies, 1)
	assert.Equal(t, q2.Name, team2Policies[0].Name)
	assert.Equal(t, q2.Query, team2Policies[0].Query)
	assert.Equal(t, q2.Description, team2Policies[0].Description)
	require.NotNil(t, team2Policies[0].AuthorID)
	require.Equal(t, user1.ID, *team2Policies[0].AuthorID)

	_, err = ds.DeleteTeamPolicies(context.Background(), team1.ID, []uint{teamPolicies[0].ID})
	require.NoError(t, err)

	teamPolicies, err = ds.ListGlobalPolicies(context.Background())
	require.NoError(t, err)
	require.Len(t, teamPolicies, 0)
}

func testPolicyQueriesForHost(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
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
	gp, err := ds.NewGlobalPolicy(context.Background(), user1.ID, q.ID, "", "", "", "some gp resolution")
	require.NoError(t, err)

	q2, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query2",
		Description: "query2 desc",
		Query:       "select 42;",
		Saved:       true,
	})
	require.NoError(t, err)
	tp, err := ds.NewTeamPolicy(context.Background(), user1.ID, team1.ID, q2.ID, "", "", "", "some other gp resolution")
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

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host1, map[uint]*bool{tp.ID: ptr.Bool(false), gp.ID: nil}, time.Now(), false))

	policies, err := ds.ListPoliciesForHost(context.Background(), host1.ID)
	require.NoError(t, err)
	require.Len(t, policies, 2)

	policies, err = ds.ListPoliciesForHost(context.Background(), host2.ID)
	require.NoError(t, err)
	require.Len(t, policies, 2)

	assert.Equal(t, "", policies[0].Response)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host2, map[uint]*bool{gp.ID: ptr.Bool(true)}, time.Now(), false))

	policies, err = ds.ListPoliciesForHost(context.Background(), host2.ID)
	require.NoError(t, err)
	require.Len(t, policies, 2)

	assert.Equal(t, "pass", policies[0].Response)

	// Manually insert a null resolution.
	res, err := ds.writer.ExecContext(context.Background(), `INSERT INTO policies (name, query, description) VALUES (?, ?, ?)`, q.Name+"2", q.Query, q.Description)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host2, map[uint]*bool{uint(id): nil}, time.Now(), false))

	policies, err = ds.ListPoliciesForHost(context.Background(), host2.ID)
	require.NoError(t, err)
	require.Len(t, policies, 3)

	assert.Equal(t, "query1 desc", policies[0].Description)
	assert.Equal(t, "some gp resolution", policies[0].Resolution)
}

func testTeamPolicyTransfer(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
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

	tq, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query1",
		Description: "query1 desc",
		Query:       "select 1;",
		Saved:       true,
	})
	require.NoError(t, err)
	teamPolicy, err := ds.NewTeamPolicy(context.Background(), user1.ID, team1.ID, tq.ID, "", "", "", "")
	require.NoError(t, err)

	gq, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query2",
		Description: "query2 desc",
		Query:       "select 2;",
		Saved:       true,
	})
	require.NoError(t, err)
	globalPolicy, err := ds.NewGlobalPolicy(context.Background(), user1.ID, gq.ID, "", "", "", "")
	require.NoError(t, err)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host1, map[uint]*bool{teamPolicy.ID: ptr.Bool(false), globalPolicy.ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host1, map[uint]*bool{teamPolicy.ID: ptr.Bool(true), globalPolicy.ID: ptr.Bool(true)}, time.Now(), false))

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
	user1 := test.NewUser(t, ds, "User1", "user1@example.com", true)
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	require.NoError(t, ds.ApplyPolicySpecs(context.Background(), user1.ID, []*fleet.PolicySpec{
		{
			Name:        "query1",
			Query:       "select 1;",
			Description: "query1 desc",
			Resolution:  "some resolution",
			Team:        "",
		},
		{
			Name:        "query2",
			Query:       "select 2;",
			Description: "query2 desc",
			Resolution:  "some other resolution",
			Team:        "team1",
		},
		{
			Name:        "query3",
			Query:       "select 3;",
			Description: "query3 desc",
			Resolution:  "some other good resolution",
			Team:        "team1",
		},
	}))

	policies, err := ds.ListGlobalPolicies(context.Background())
	require.NoError(t, err)
	require.Len(t, policies, 1)
	assert.Equal(t, "query1", policies[0].Name)
	assert.Equal(t, "select 1;", policies[0].Query)
	assert.Equal(t, "query1 desc", policies[0].Description)
	require.NotNil(t, policies[0].AuthorID)
	assert.Equal(t, user1.ID, *policies[0].AuthorID)
	require.NotNil(t, policies[0].Resolution)
	assert.Equal(t, "some resolution", *policies[0].Resolution)

	teamPolicies, err := ds.ListTeamPolicies(context.Background(), team1.ID)
	require.NoError(t, err)
	require.Len(t, teamPolicies, 2)
	assert.Equal(t, "query2", teamPolicies[0].Name)
	assert.Equal(t, "select 2;", teamPolicies[0].Query)
	assert.Equal(t, "query2 desc", teamPolicies[0].Description)
	require.NotNil(t, teamPolicies[0].AuthorID)
	assert.Equal(t, user1.ID, *teamPolicies[0].AuthorID)
	require.NotNil(t, teamPolicies[0].Resolution)
	assert.Equal(t, "some other resolution", *teamPolicies[0].Resolution)

	assert.Equal(t, "query3", teamPolicies[1].Name)
	assert.Equal(t, "select 3;", teamPolicies[1].Query)
	assert.Equal(t, "query3 desc", teamPolicies[1].Description)
	require.NotNil(t, teamPolicies[1].AuthorID)
	assert.Equal(t, user1.ID, *teamPolicies[1].AuthorID)
	require.NotNil(t, teamPolicies[1].Resolution)
	assert.Equal(t, "some other good resolution", *teamPolicies[1].Resolution)

	require.Error(t, ds.ApplyPolicySpecs(context.Background(), user1.ID, []*fleet.PolicySpec{
		{
			Name:  "",
			Query: "select 1;",
		},
	}))
	require.Error(t, ds.ApplyPolicySpecs(context.Background(), user1.ID, []*fleet.PolicySpec{
		{
			Name:  "query4",
			Query: "",
		},
	}))
	// Make sure apply is idempotent
	require.NoError(t, ds.ApplyPolicySpecs(context.Background(), user1.ID, []*fleet.PolicySpec{
		{
			Name:        "query1",
			Query:       "select 1;",
			Description: "query1 desc",
			Resolution:  "some resolution",
			Team:        "",
		},
		{
			Name:        "query2",
			Query:       "select 2;",
			Description: "query2 desc",
			Resolution:  "some other resolution",
			Team:        "team1",
		},
		{
			Name:        "query3",
			Query:       "select 3;",
			Description: "query3 desc",
			Resolution:  "some other good resolution",
			Team:        "team1",
		},
	}))

	policies, err = ds.ListGlobalPolicies(context.Background())
	require.NoError(t, err)
	require.Len(t, policies, 1)
	teamPolicies, err = ds.ListTeamPolicies(context.Background(), team1.ID)
	require.NoError(t, err)
	require.Len(t, teamPolicies, 2)

	// Test policy updating.
	require.NoError(t, ds.ApplyPolicySpecs(context.Background(), user1.ID, []*fleet.PolicySpec{
		{
			Name:        "query1",
			Query:       "select 1 from updated;",
			Description: "query1 desc updated",
			Resolution:  "some resolution updated",
			Team:        "",
		},
		{
			Name:        "query2",
			Query:       "select 2 from updated;",
			Description: "query2 desc updated",
			Resolution:  "some other resolution updated",
			Team:        "team1",
		},
	}))
	policies, err = ds.ListGlobalPolicies(context.Background())
	require.NoError(t, err)
	require.Len(t, policies, 1)

	assert.Equal(t, "query1", policies[0].Name)
	assert.Equal(t, "select 1 from updated;", policies[0].Query)
	assert.Equal(t, "query1 desc updated", policies[0].Description)
	require.NotNil(t, policies[0].AuthorID)
	assert.Equal(t, user1.ID, *policies[0].AuthorID)
	require.NotNil(t, policies[0].Resolution)
	assert.Equal(t, "some resolution updated", *policies[0].Resolution)

	teamPolicies, err = ds.ListTeamPolicies(context.Background(), team1.ID)
	require.NoError(t, err)
	require.Len(t, teamPolicies, 2)

	assert.Equal(t, "query2", teamPolicies[0].Name)
	assert.Equal(t, "select 2 from updated;", teamPolicies[0].Query)
	assert.Equal(t, "query2 desc updated", teamPolicies[0].Description)
	require.NotNil(t, teamPolicies[0].AuthorID)
	assert.Equal(t, user1.ID, *teamPolicies[0].AuthorID)
	require.NotNil(t, teamPolicies[0].Resolution)
	assert.Equal(t, "some other resolution updated", *teamPolicies[0].Resolution)
}
