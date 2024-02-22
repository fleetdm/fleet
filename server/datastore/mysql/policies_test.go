package mysql

import (
	"context"
	"crypto/md5" //nolint:gosec // (only used for tests)
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicies(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"NewGlobalPolicyLegacy", testPoliciesNewGlobalPolicyLegacy},
		{"NewGlobalPolicyProprietary", testPoliciesNewGlobalPolicyProprietary},
		{"MembershipViewDeferred", func(t *testing.T, ds *Datastore) { testPoliciesMembershipView(true, t, ds) }},
		{"MembershipViewNotDeferred", func(t *testing.T, ds *Datastore) { testPoliciesMembershipView(false, t, ds) }},
		{"TeamPolicyLegacy", testTeamPolicyLegacy},
		{"TeamPolicyProprietary", testTeamPolicyProprietary},
		{"PolicyQueriesForHost", testPolicyQueriesForHost},
		{"PolicyQueriesForHostPlatforms", testPolicyQueriesForHostPlatforms},
		{"PoliciesByID", testPoliciesByID},
		{"TeamPolicyTransfer", testTeamPolicyTransfer},
		{"ApplyPolicySpec", testApplyPolicySpec},
		{"Save", testPoliciesSave},
		{"DelUser", testPoliciesDelUser},
		{"FlippingPoliciesForHost", testFlippingPoliciesForHost},
		{"PlatformUpdate", testPolicyPlatformUpdate},
		{"CleanupPolicyMembership", testPolicyCleanupPolicyMembership},
		{"DeleteAllPolicyMemberships", testDeleteAllPolicyMemberships},
		{"PolicyViolationDays", testPolicyViolationDays},
		{"IncreasePolicyAutomationIteration", testIncreasePolicyAutomationIteration},
		{"OutdatedAutomationBatch", testOutdatedAutomationBatch},
		{"TestUpdatePolicyFailureCountsForHosts", testUpdatePolicyFailureCountsForHosts},
		{"TestPolicyIDsByName", testPolicyByName},
		{"TestListGlobalPoliciesCanPaginate", testListGlobalPoliciesCanPaginate},
		{"TestListTeamPoliciesCanPaginate", testListTeamPoliciesCanPaginate},
		{"TestCountPolicies", testCountPolicies},
		{"TestUpdatePolicyHostCounts", testUpdatePolicyHostCounts},
		{"TestCachedPolicyCountDeletesOnPolicyChange", testCachedPolicyCountDeletesOnPolicyChange},
		{"TestPoliciesListOptions", testPoliciesListOptions},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testPoliciesNewGlobalPolicyLegacy(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	q, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query1",
		Description: "query1 desc",
		Query:       "select 1;",
		Saved:       true,
		Logging:     fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	p, err := ds.NewGlobalPolicy(context.Background(), &user1.ID, fleet.PolicyPayload{
		QueryID: &q.ID,
	})
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
		Logging:     fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	_, err = ds.NewGlobalPolicy(context.Background(), &user1.ID, fleet.PolicyPayload{
		QueryID: &q2.ID,
	})
	require.NoError(t, err)

	policies, err := ds.ListGlobalPolicies(context.Background(), fleet.ListOptions{})
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
	require.NoError(t, ds.DeleteQuery(context.Background(), nil, q.Name))

	_, err = ds.DeleteGlobalPolicies(context.Background(), []uint{policies[0].ID, policies[1].ID})
	require.NoError(t, err)

	policies, err = ds.ListGlobalPolicies(context.Background(), fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, policies, 0)
}

func testPoliciesNewGlobalPolicyProprietary(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	ctx := context.Background()
	p, err := ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:        "query1",
		Query:       "select 1;",
		Description: "query1 desc",
		Resolution:  "query1 resolution",
	})
	require.NoError(t, err)

	assert.Equal(t, "query1", p.Name)
	assert.Equal(t, "query1 desc", p.Description)
	assert.Equal(t, "select 1;", p.Query)
	require.NotNil(t, p.Resolution)
	assert.Equal(t, "query1 resolution", *p.Resolution)
	require.NotNil(t, p.AuthorID)
	assert.Equal(t, user1.ID, *p.AuthorID)

	_, err = ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:        "query2",
		Query:       "select 2;",
		Description: "query2 desc",
		Resolution:  "query2 resolution",
	})
	require.NoError(t, err)

	policies, err := ds.ListGlobalPolicies(ctx, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, policies, 2)
	assert.Equal(t, "query1", policies[0].Name)
	assert.Equal(t, "select 1;", policies[0].Query)
	assert.Equal(t, "query1 desc", policies[0].Description)
	require.NotNil(t, policies[0].Resolution)
	assert.Equal(t, "query1 resolution", *policies[0].Resolution)
	require.NotNil(t, policies[0].AuthorID)
	assert.Equal(t, user1.ID, *policies[0].AuthorID)
	assert.Equal(t, "query2", policies[1].Name)
	assert.Equal(t, "select 2;", policies[1].Query)
	assert.Equal(t, "query2 desc", policies[1].Description)
	require.NotNil(t, policies[1].Resolution)
	assert.Equal(t, "query2 resolution", *policies[1].Resolution)
	require.NotNil(t, policies[1].AuthorID)
	assert.Equal(t, user1.ID, *policies[1].AuthorID)

	// Can't create a global policy with an existing name.
	p3, err := ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:        "query1",
		Query:       "select 3;",
		Description: "query1 other description",
		Resolution:  "query1 other resolution",
	})
	require.Error(t, err)
	var isExist interface {
		IsExists() bool
	}
	require.True(t, errors.As(err, &isExist) && isExist.IsExists())
	require.Nil(t, p3)

	_, err = ds.DeleteGlobalPolicies(ctx, []uint{policies[0].ID, policies[1].ID})
	require.NoError(t, err)

	policies, err = ds.ListGlobalPolicies(ctx, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, policies, 0)

	// Now the name is available and we can create the global policy.
	p3, err = ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:        "query1",
		Query:       "select 3;",
		Description: "query1 other description",
		Resolution:  "query1 other resolution",
	})
	require.NoError(t, err)
	assert.Equal(t, "query1", p3.Name)
	assert.Equal(t, "select 3;", p3.Query)
	assert.Equal(t, "query1 other description", p3.Description)
	require.NotNil(t, p3.Resolution)
	assert.Equal(t, "query1 other resolution", *p3.Resolution)
	require.NotNil(t, p3.AuthorID)
	assert.Equal(t, user1.ID, *p3.AuthorID)
}

func testPoliciesListOptions(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	ctx := context.Background()

	_, err := ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:        "apple",
		Query:       "select 1;",
		Description: "query1 desc",
		Resolution:  "query1 resolution",
	})
	require.NoError(t, err)

	_, err = ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:        "banana",
		Query:       "select 1;",
		Description: "query2 desc",
		Resolution:  "query2 resolution",
	})
	require.NoError(t, err)

	_, err = ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:        "cherry",
		Query:       "select 1;",
		Description: "query3 desc",
		Resolution:  "query3 resolution",
	})
	require.NoError(t, err)

	_, err = ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:        "apple pie",
		Query:       "select 1;",
		Description: "query4 desc",
		Resolution:  "query4 resolution",
	})
	require.NoError(t, err)

	_, err = ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:        "rotten apple",
		Query:       "select 1;",
		Description: "query5 desc",
		Resolution:  "query5 resolution",
	})
	require.NoError(t, err)

	policies, err := ds.ListGlobalPolicies(ctx, fleet.ListOptions{MatchQuery: "apple", OrderKey: "name", OrderDirection: fleet.OrderAscending})
	require.NoError(t, err)
	require.Len(t, policies, 3)
	assert.Equal(t, "apple", policies[0].Name)
	assert.Equal(t, "apple pie", policies[1].Name)
	assert.Equal(t, "rotten apple", policies[2].Name)
}

func testPoliciesMembershipView(deferred bool, t *testing.T, ds *Datastore) {
	ctx := context.Background()

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	host1, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("1234"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.NoError(t, err)

	host2, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("5679"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		Hostname:        "bar.local",
	})
	require.NoError(t, err)

	q, err := ds.NewQuery(ctx, &fleet.Query{
		Name:        "query1",
		Description: "query1 desc",
		Query:       "select 1;",
		Saved:       true,
		Logging:     fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	p, err := ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		QueryID: &q.ID,
	})
	require.NoError(t, err)

	assert.Equal(t, "query1", p.Name)
	assert.Equal(t, "select 1;", p.Query)
	assert.Equal(t, "query1 desc", p.Description)
	require.NotNil(t, p.AuthorID)
	assert.Equal(t, user1.ID, *p.AuthorID)

	q2, err := ds.NewQuery(ctx, &fleet.Query{
		Name:        "query2",
		Description: "query2 desc",
		Query:       "select 42;",
		Saved:       true,
		Logging:     fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	p2, err := ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		QueryID: &q2.ID,
	})
	require.NoError(t, err)

	assert.Equal(t, "query2", p2.Name)
	assert.Equal(t, "select 42;", p2.Query)
	assert.Equal(t, "query2 desc", p2.Description)
	require.NotNil(t, p2.AuthorID)
	assert.Equal(t, user1.ID, *p2.AuthorID)

	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, host1, map[uint]*bool{p.ID: ptr.Bool(true)}, time.Now(), deferred))
	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, host1, map[uint]*bool{p.ID: ptr.Bool(true)}, time.Now(), deferred))

	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, host2, map[uint]*bool{p.ID: nil}, time.Now(), deferred))
	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, host2, map[uint]*bool{p.ID: ptr.Bool(false)}, time.Now(), deferred))
	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, host2, map[uint]*bool{p.ID: ptr.Bool(true)}, time.Now(), deferred))

	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, host2, map[uint]*bool{p2.ID: nil}, time.Now(), deferred))

	require.NoError(t, ds.UpdateHostPolicyCounts(ctx))

	policies, err := ds.ListGlobalPolicies(ctx, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, policies, 2)

	assert.Equal(t, p.ID, policies[0].ID)
	assert.Equal(t, uint(2), policies[0].PassingHostCount)
	assert.Equal(t, uint(0), policies[0].FailingHostCount)

	assert.Equal(t, p2.ID, policies[1].ID)
	assert.Equal(t, uint(0), policies[1].PassingHostCount)
	assert.Equal(t, uint(0), policies[1].FailingHostCount)

	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, host1, map[uint]*bool{p.ID: ptr.Bool(false)}, time.Now(), deferred))
	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, host2, map[uint]*bool{p2.ID: ptr.Bool(false)}, time.Now(), deferred))

	require.NoError(t, ds.UpdateHostPolicyCounts(ctx))

	policies, err = ds.ListGlobalPolicies(ctx, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, policies, 2)

	assert.Equal(t, p.ID, policies[0].ID)
	assert.Equal(t, uint(1), policies[0].PassingHostCount)
	assert.Equal(t, uint(1), policies[0].FailingHostCount)

	assert.Equal(t, p2.ID, policies[1].ID)
	assert.Equal(t, uint(0), policies[1].PassingHostCount)
	assert.Equal(t, uint(1), policies[1].FailingHostCount)

	policy, err := ds.Policy(ctx, policies[0].ID)
	require.NoError(t, err)
	assert.Equal(t, policies[0], policy)

	queries, err := ds.PolicyQueriesForHost(ctx, host1)
	require.NoError(t, err)
	require.Len(t, queries, 2)
	assert.Equal(t, q.Query, queries[fmt.Sprint(q.ID)])
	assert.Equal(t, q2.Query, queries[fmt.Sprint(q2.ID)])

	// create a couple teams and team-specific policies
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	t1pol, err := ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, fleet.PolicyPayload{
		Name:  "team1pol",
		Query: "SELECT 1",
	})
	require.NoError(t, err)
	t2pol, err := ds.NewTeamPolicy(ctx, team2.ID, &user1.ID, fleet.PolicyPayload{
		Name:  "team2pol",
		Query: "SELECT 2",
	})
	require.NoError(t, err)
	t2pol2, err := ds.NewTeamPolicy(ctx, team2.ID, &user1.ID, fleet.PolicyPayload{
		Name:  "team2pol2",
		Query: "SELECT 3",
	})
	require.NoError(t, err)

	// create hosts in each team
	host3, err := ds.EnrollHost(ctx, false, "3", "", "", "3", &team1.ID, 0)
	require.NoError(t, err)
	host4, err := ds.EnrollHost(ctx, false, "4", "", "", "4", &team2.ID, 0)
	require.NoError(t, err)
	host5, err := ds.EnrollHost(ctx, false, "5", "", "", "5", &team2.ID, 0)
	require.NoError(t, err)

	// create some policy results
	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, host3, map[uint]*bool{t1pol.ID: ptr.Bool(true), p.ID: ptr.Bool(true), p2.ID: ptr.Bool(false)}, time.Now(), deferred))
	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, host4, map[uint]*bool{t2pol.ID: ptr.Bool(false), t2pol2.ID: ptr.Bool(true), p.ID: ptr.Bool(false)}, time.Now(), deferred))
	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, host5, map[uint]*bool{t2pol.ID: ptr.Bool(true), t2pol2.ID: ptr.Bool(true), p2.ID: ptr.Bool(true)}, time.Now(), deferred))

	require.NoError(t, ds.UpdateHostPolicyCounts(ctx))

	t1Pols, t1Inherited, err := ds.ListTeamPolicies(ctx, team1.ID, fleet.ListOptions{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, t1Pols, 1)
	assert.Equal(t, uint(1), t1Pols[0].PassingHostCount)
	assert.Equal(t, uint(0), t1Pols[0].FailingHostCount)

	require.Len(t, t1Inherited, 2)
	require.Equal(t, p.ID, t1Inherited[0].ID)
	assert.Equal(t, uint(1), t1Inherited[0].PassingHostCount)
	assert.Equal(t, uint(0), t1Inherited[0].FailingHostCount)
	require.Equal(t, p2.ID, t1Inherited[1].ID)
	assert.Equal(t, uint(0), t1Inherited[1].PassingHostCount)
	assert.Equal(t, uint(1), t1Inherited[1].FailingHostCount)

	t2Pols, t2Inherited, err := ds.ListTeamPolicies(ctx, team2.ID, fleet.ListOptions{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, t2Pols, 2)
	require.Equal(t, t2pol.ID, t2Pols[0].ID)
	assert.Equal(t, uint(1), t2Pols[0].PassingHostCount)
	assert.Equal(t, uint(1), t2Pols[0].FailingHostCount)
	require.Equal(t, t2pol2.ID, t2Pols[1].ID)
	assert.Equal(t, uint(2), t2Pols[1].PassingHostCount)
	assert.Equal(t, uint(0), t2Pols[1].FailingHostCount)

	require.Len(t, t2Inherited, 2)
	require.Equal(t, p.ID, t2Inherited[0].ID)
	assert.Equal(t, uint(0), t2Inherited[0].PassingHostCount)
	assert.Equal(t, uint(1), t2Inherited[0].FailingHostCount)
	require.Equal(t, p2.ID, t2Inherited[1].ID)
	assert.Equal(t, uint(1), t2Inherited[1].PassingHostCount)
	assert.Equal(t, uint(0), t2Inherited[1].FailingHostCount)
}

func testTeamPolicyLegacy(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	q, err := ds.NewQuery(ctx, &fleet.Query{
		Name:        "query1",
		Description: "query1 desc",
		Query:       "select 1;",
		Saved:       true,
		Logging:     fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	q2, err := ds.NewQuery(ctx, &fleet.Query{
		Name:        "query2",
		Description: "query2 desc",
		Query:       "select 1;",
		Saved:       true,
		Logging:     fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	prevPolicies, err := ds.ListGlobalPolicies(ctx, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, prevPolicies, 0)

	_, err = ds.NewTeamPolicy(ctx, 99999999, &user1.ID, fleet.PolicyPayload{
		QueryID: &q.ID,
	})
	require.Error(t, err)

	p, err := ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, fleet.PolicyPayload{
		QueryID:    &q.ID,
		Resolution: "some resolution",
	})
	require.NoError(t, err)

	assert.Equal(t, "query1", p.Name)
	assert.Equal(t, "select 1;", p.Query)
	assert.Equal(t, "query1 desc", p.Description)
	require.NotNil(t, p.AuthorID)
	assert.Equal(t, user1.ID, *p.AuthorID)

	require.NotNil(t, p.Resolution)
	assert.Equal(t, "some resolution", *p.Resolution)

	gpol, err := ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:  "global_1",
		Query: "SELECT 1",
	})
	require.NoError(t, err)

	globalPolicies, err := ds.ListGlobalPolicies(ctx, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, globalPolicies, 1)

	p2, err := ds.NewTeamPolicy(ctx, team2.ID, &user1.ID, fleet.PolicyPayload{
		QueryID: &q2.ID,
	})
	require.NoError(t, err)

	assert.Equal(t, "query2", p2.Name)
	assert.Equal(t, "select 1;", p2.Query)
	assert.Equal(t, "query2 desc", p2.Description)
	require.NotNil(t, p2.AuthorID)
	assert.Equal(t, user1.ID, *p2.AuthorID)

	teamPolicies, inherited1, err := ds.ListTeamPolicies(ctx, team1.ID, fleet.ListOptions{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, teamPolicies, 1)
	assert.Equal(t, q.Name, teamPolicies[0].Name)
	assert.Equal(t, q.Query, teamPolicies[0].Query)
	assert.Equal(t, q.Description, teamPolicies[0].Description)
	require.NotNil(t, teamPolicies[0].AuthorID)
	require.Equal(t, user1.ID, *teamPolicies[0].AuthorID)

	require.Len(t, inherited1, 1)
	require.Equal(t, gpol, inherited1[0])

	team2Policies, inherited2, err := ds.ListTeamPolicies(ctx, team2.ID, fleet.ListOptions{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, team2Policies, 1)
	assert.Equal(t, q2.Name, team2Policies[0].Name)
	assert.Equal(t, q2.Query, team2Policies[0].Query)
	assert.Equal(t, q2.Description, team2Policies[0].Description)
	require.NotNil(t, team2Policies[0].AuthorID)
	require.Equal(t, user1.ID, *team2Policies[0].AuthorID)

	require.Len(t, inherited2, 1)
	require.Equal(t, gpol, inherited2[0])

	_, err = ds.DeleteTeamPolicies(ctx, team1.ID, []uint{teamPolicies[0].ID})
	require.NoError(t, err)

	teamPolicies, inherited1, err = ds.ListTeamPolicies(ctx, team1.ID, fleet.ListOptions{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, teamPolicies, 0)
	require.Len(t, inherited1, 1)
}

func testTeamPolicyProprietary(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	ctx := context.Background()
	gpol, err := ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:        "existing-query-global-1",
		Query:       "select 1;",
		Description: "query1 desc",
		Resolution:  "query1 resolution",
	})
	require.NoError(t, err)

	prevPolicies, err := ds.ListGlobalPolicies(ctx, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, prevPolicies, 1)

	// team does not exist
	_, err = ds.NewTeamPolicy(ctx, 99999999, &user1.ID, fleet.PolicyPayload{
		Name:        "query1",
		Query:       "select 1;",
		Description: "query1 desc",
		Resolution:  "query1 resolution",
	})
	require.Error(t, err)

	p, err := ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, fleet.PolicyPayload{
		Name:        "query1",
		Query:       "select 1;",
		Description: "query1 desc",
		Resolution:  "query1 resolution",
	})
	require.NoError(t, err)

	// Can't create a team policy with same team id and name.
	_, err = ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, fleet.PolicyPayload{
		Name:  "query1",
		Query: "select 1;",
	})
	require.Error(t, err)
	var isExist interface {
		IsExists() bool
	}
	require.True(t, errors.As(err, &isExist) && isExist.IsExists(), err)

	// Can't create a global policy with an existing global name.
	_, err = ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:  "existing-query-global-1",
		Query: "select 1;",
	})
	require.Error(t, err)
	require.True(t, errors.As(err, &isExist) && isExist.IsExists(), err)

	assert.Equal(t, "query1", p.Name)
	assert.Equal(t, "select 1;", p.Query)
	assert.Equal(t, "query1 desc", p.Description)
	require.NotNil(t, p.Resolution)
	assert.Equal(t, "query1 resolution", *p.Resolution)
	require.NotNil(t, p.AuthorID)
	assert.Equal(t, user1.ID, *p.AuthorID)

	globalPolicies, err := ds.ListGlobalPolicies(ctx, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, globalPolicies, len(prevPolicies))

	p2, err := ds.NewTeamPolicy(ctx, team2.ID, &user1.ID, fleet.PolicyPayload{
		Name:        "query2",
		Query:       "select 2;",
		Description: "query2 desc",
		Resolution:  "query2 resolution",
	})
	require.NoError(t, err)

	assert.Equal(t, "query2", p2.Name)
	assert.Equal(t, "select 2;", p2.Query)
	assert.Equal(t, "query2 desc", p2.Description)
	require.NotNil(t, p2.Resolution)
	assert.Equal(t, "query2 resolution", *p2.Resolution)
	require.NotNil(t, p2.AuthorID)
	assert.Equal(t, user1.ID, *p2.AuthorID)

	teamPolicies, inherited1, err := ds.ListTeamPolicies(ctx, team1.ID, fleet.ListOptions{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, teamPolicies, 1)
	assert.Equal(t, "query1", teamPolicies[0].Name)
	assert.Equal(t, "select 1;", teamPolicies[0].Query)
	assert.Equal(t, "query1 desc", teamPolicies[0].Description)
	require.NotNil(t, teamPolicies[0].Resolution)
	assert.Equal(t, "query1 resolution", *teamPolicies[0].Resolution)
	require.NotNil(t, teamPolicies[0].AuthorID)
	require.Equal(t, user1.ID, *teamPolicies[0].AuthorID)

	require.Len(t, inherited1, 1)
	require.Equal(t, gpol, inherited1[0])

	team2Policies, inherited2, err := ds.ListTeamPolicies(ctx, team2.ID, fleet.ListOptions{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, team2Policies, 1)
	assert.Equal(t, "query2", team2Policies[0].Name)
	assert.Equal(t, "select 2;", team2Policies[0].Query)
	assert.Equal(t, "query2 desc", team2Policies[0].Description)
	require.NotNil(t, team2Policies[0].Resolution)
	assert.Equal(t, "query2 resolution", *team2Policies[0].Resolution)
	require.NotNil(t, team2Policies[0].AuthorID)
	require.Equal(t, user1.ID, *team2Policies[0].AuthorID)

	require.Len(t, inherited2, 1)
	require.Equal(t, gpol, inherited2[0])

	// Can't create a policy with the same name on the same team.
	p3, err := ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, fleet.PolicyPayload{
		Name:        "query1",
		Query:       "select 2;",
		Description: "query2 other description",
		Resolution:  "query2 other resolution",
	})
	require.Error(t, err)
	require.Nil(t, p3)

	_, err = ds.DeleteTeamPolicies(ctx, team1.ID, []uint{teamPolicies[0].ID})
	require.NoError(t, err)

	teamPolicies, inherited1, err = ds.ListTeamPolicies(ctx, team1.ID, fleet.ListOptions{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, teamPolicies, 0)
	require.Len(t, inherited1, 1)
	require.Equal(t, gpol, inherited1[0])

	// Now the name is available and we can create the policy in the team.
	_, err = ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, fleet.PolicyPayload{
		Name:        "query1",
		Query:       "select 2;",
		Description: "query2 other description",
		Resolution:  "query2 other resolution",
	})
	require.NoError(t, err)

	teamPolicies, _, err = ds.ListTeamPolicies(ctx, team1.ID, fleet.ListOptions{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, teamPolicies, 1)
	assert.Equal(t, "query1", teamPolicies[0].Name)
	assert.Equal(t, "select 2;", teamPolicies[0].Query)
	assert.Equal(t, "query2 other description", teamPolicies[0].Description)
	require.NotNil(t, teamPolicies[0].Resolution)
	assert.Equal(t, "query2 other resolution", *teamPolicies[0].Resolution)
	require.NotNil(t, team2Policies[0].AuthorID)
	require.Equal(t, user1.ID, *team2Policies[0].AuthorID)
}

func newTestHostWithPlatform(t *testing.T, ds *Datastore, hostname, platform string, teamID *uint) *fleet.Host {
	nodeKey, err := server.GenerateRandomText(32)
	require.NoError(t, err)
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID:   ptr.String(uuid.NewString()),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         &nodeKey,
		UUID:            uuid.NewString(),
		Hostname:        hostname,
		Platform:        platform,
	})
	require.NoError(t, err)
	if teamID != nil {
		err := ds.AddHostsToTeam(context.Background(), teamID, []uint{host.ID})
		require.NoError(t, err)
		host, err = ds.Host(context.Background(), host.ID)
		require.NoError(t, err)
	}
	return host
}

func newTestPolicy(t *testing.T, ds *Datastore, user *fleet.User, name, platforms string, teamID *uint) *fleet.Policy {
	query := fmt.Sprintf("select %s;", name)
	if teamID == nil {
		gp, err := ds.NewGlobalPolicy(context.Background(), &user.ID, fleet.PolicyPayload{
			Name:     name,
			Query:    query,
			Platform: platforms,
		})
		require.NoError(t, err)
		return gp
	}
	tp, err := ds.NewTeamPolicy(context.Background(), *teamID, &user.ID, fleet.PolicyPayload{
		Name:     name,
		Query:    query,
		Platform: platforms,
	})
	require.NoError(t, err)
	return tp
}

type expectedPolicyResults struct {
	policyQueries map[string]string
	hostPolicies  []*fleet.HostPolicy
}

func expectedPolicyQueries(policies ...*fleet.Policy) expectedPolicyResults {
	queries := make(map[string]string)
	for _, policy := range policies {
		queries[strconv.Itoa(int(policy.ID))] = policy.Query
	}
	hostPolicies := make([]*fleet.HostPolicy, len(policies))
	for i := range policies {
		hostPolicies[i] = &fleet.HostPolicy{
			PolicyData: policies[i].PolicyData,
		}
	}
	return expectedPolicyResults{
		policyQueries: queries,
		hostPolicies:  hostPolicies,
	}
}

func testPolicyQueriesForHostPlatforms(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)

	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)

	// Global hosts:
	var global *uint
	host1GlobalUbuntu := newTestHostWithPlatform(t, ds, "host1_global_ubuntu", "ubuntu", global)
	host2GlobalDarwin := newTestHostWithPlatform(t, ds, "host2_global_darwin", "darwin", global)
	host3GlobalWindows := newTestHostWithPlatform(t, ds, "host3_global_windows", "windows", global)
	host4GlobalEmpty := newTestHostWithPlatform(t, ds, "host4_global_empty_platform", "", global)

	// team1 hosts:
	host1t1Rhel := newTestHostWithPlatform(t, ds, "host1_team1_ubuntu", "rhel", &team1.ID)
	host2t1Darwin := newTestHostWithPlatform(t, ds, "host2_team1_darwin", "darwin", &team1.ID)
	host3t1Windows := newTestHostWithPlatform(t, ds, "host3_team1_windows", "windows", &team1.ID)
	host4t1Empty := newTestHostWithPlatform(t, ds, "host4_team1_empty_platform", "", &team1.ID)

	// team2 hosts
	host1t2Debian := newTestHostWithPlatform(t, ds, "host1_team2_ubuntu", "debian", &team2.ID)
	host2t2Darwin := newTestHostWithPlatform(t, ds, "host2_team2_darwin", "darwin", &team2.ID)
	host3t2Windows := newTestHostWithPlatform(t, ds, "host3_team2_windows", "windows", &team2.ID)
	host4t2Empty := newTestHostWithPlatform(t, ds, "host4_team2_empty_platform", "", &team2.ID)

	// Global policies:
	policy1GlobalLinuxDarwin := newTestPolicy(t, ds, user1, "policy1_global_linux_darwin", "linux,darwin", global)
	policy2GlobalWindows := newTestPolicy(t, ds, user1, "policy2_global_windows", "windows", global)
	policy3GlobalAll := newTestPolicy(t, ds, user1, "policy3_global_all", "", global)

	// Team1 policies:
	policy1t1Darwin := newTestPolicy(t, ds, user1, "policy1_team1_darwin", "darwin", &team1.ID)
	policy2t1Windows := newTestPolicy(t, ds, user1, "policy2_team1_windows", "windows", &team1.ID)
	policy3t1All := newTestPolicy(t, ds, user1, "policy3_team1_all", "", &team1.ID)

	// Team2 policies:
	policy1t2LinuxDarwin := newTestPolicy(t, ds, user1, "policy1_team2_linux_darwin", "linux,darwin", &team2.ID)
	policy2t2Windows := newTestPolicy(t, ds, user1, "policy2_team2_windows", "windows", &team2.ID)
	policy3t2All1 := newTestPolicy(t, ds, user1, "policy3_team2_all1", "linux,darwin,windows", &team2.ID)
	policy4t2All2 := newTestPolicy(t, ds, user1, "policy4_team2_all2", "", &team2.ID)

	for _, tc := range []struct {
		host             *fleet.Host
		expectedPolicies expectedPolicyResults
	}{
		{
			host: host1GlobalUbuntu,
			expectedPolicies: expectedPolicyQueries(
				policy1GlobalLinuxDarwin,
				policy3GlobalAll,
			),
		},
		{
			host: host2GlobalDarwin,
			expectedPolicies: expectedPolicyQueries(
				policy1GlobalLinuxDarwin,
				policy3GlobalAll,
			),
		},
		{
			host: host3GlobalWindows,
			expectedPolicies: expectedPolicyQueries(
				policy2GlobalWindows,
				policy3GlobalAll,
			),
		},
		{
			host: host4GlobalEmpty,
			expectedPolicies: expectedPolicyQueries(
				policy3GlobalAll,
			),
		},
		{
			host: host1t1Rhel,
			expectedPolicies: expectedPolicyQueries(
				policy1GlobalLinuxDarwin,
				policy3GlobalAll,

				policy3t1All,
			),
		},
		{
			host: host2t1Darwin,
			expectedPolicies: expectedPolicyQueries(
				policy1GlobalLinuxDarwin,
				policy3GlobalAll,

				policy3t1All,
				policy1t1Darwin,
			),
		},
		{
			host: host3t1Windows,
			expectedPolicies: expectedPolicyQueries(
				policy2GlobalWindows,
				policy3GlobalAll,

				policy3t1All,
				policy2t1Windows,
			),
		},
		{
			host: host4t1Empty,
			expectedPolicies: expectedPolicyQueries(
				policy3GlobalAll,

				policy3t1All,
			),
		},
		{
			host: host1t2Debian,
			expectedPolicies: expectedPolicyQueries(
				policy1GlobalLinuxDarwin,
				policy3GlobalAll,

				policy1t2LinuxDarwin,
				policy3t2All1,
				policy4t2All2,
			),
		},
		{
			host: host2t2Darwin,
			expectedPolicies: expectedPolicyQueries(
				policy1GlobalLinuxDarwin,
				policy3GlobalAll,

				policy1t2LinuxDarwin,
				policy3t2All1,
				policy4t2All2,
			),
		},
		{
			host: host3t2Windows,
			expectedPolicies: expectedPolicyQueries(
				policy2GlobalWindows,
				policy3GlobalAll,

				policy2t2Windows,
				policy3t2All1,
				policy4t2All2,
			),
		},
		{
			host: host4t2Empty,
			expectedPolicies: expectedPolicyQueries(
				policy3GlobalAll,

				policy4t2All2,
			),
		},
	} {
		t.Run(tc.host.Hostname, func(t *testing.T) {
			// PolicyQueriesForHost is the endpoint used by osquery agents when they check in.
			queries, err := ds.PolicyQueriesForHost(context.Background(), tc.host)
			require.NoError(t, err)
			require.Equal(t, tc.expectedPolicies.policyQueries, queries)
			// ListPoliciesForHost is the endpoint used by fleet UI/API clients.
			hostPolicies, err := ds.ListPoliciesForHost(context.Background(), tc.host)
			require.NoError(t, err)
			sort.Slice(hostPolicies, func(i, j int) bool {
				return hostPolicies[i].ID < hostPolicies[j].ID
			})
			sort.Slice(tc.expectedPolicies.hostPolicies, func(i, j int) bool {
				return tc.expectedPolicies.hostPolicies[i].ID < tc.expectedPolicies.hostPolicies[j].ID
			})
			require.Equal(t, tc.expectedPolicies.hostPolicies, hostPolicies)
		})
	}
}

func testPolicyQueriesForHost(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	host1, err := ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID:   ptr.String("1234"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.NoError(t, err)

	require.NoError(t, ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host1.ID}))
	host1, err = ds.Host(context.Background(), host1.ID)
	require.NoError(t, err)

	host2, err := ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID:   ptr.String("5679"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		Hostname:        "bar.local",
	})
	require.NoError(t, err)

	q, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query1",
		Description: "query1 desc",
		Query:       "select 1;",
		Saved:       true,
		Logging:     fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	gp, err := ds.NewGlobalPolicy(context.Background(), &user1.ID, fleet.PolicyPayload{
		QueryID:    &q.ID,
		Resolution: "some gp resolution",
	})
	require.NoError(t, err)

	q2, err := ds.NewQuery(context.Background(), &fleet.Query{
		Name:        "query2",
		Description: "query2 desc",
		Query:       "select 42;",
		Saved:       true,
		Logging:     fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	tp, err := ds.NewTeamPolicy(context.Background(), team1.ID, &user1.ID, fleet.PolicyPayload{
		QueryID:    &q2.ID,
		Resolution: "some other gp resolution",
	})
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

	// Team policy ran with failing result.
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host1, map[uint]*bool{tp.ID: ptr.Bool(false), gp.ID: nil}, time.Now(), false))

	policies, err := ds.ListPoliciesForHost(context.Background(), host1)
	require.NoError(t, err)
	require.Len(t, policies, 2)

	checkGlobaPolicy := func(policy *fleet.HostPolicy) {
		assert.Equal(t, "query1", policy.Name)
		assert.Equal(t, "select 1;", policy.Query)
		assert.Equal(t, "query1 desc", policy.Description)
		require.NotNil(t, policy.AuthorID)
		assert.Equal(t, user1.ID, *policy.AuthorID)
		assert.Equal(t, "Alice", policy.AuthorName)
		assert.Equal(t, "alice@example.com", policy.AuthorEmail)
		assert.NotNil(t, policy.Resolution)
		assert.Equal(t, "some gp resolution", *policy.Resolution)
	}

	// Failing policy is listed first.
	assert.Equal(t, "fail", policies[0].Response)
	assert.Equal(t, "query2", policies[0].Name)
	assert.Equal(t, "select 42;", policies[0].Query)
	assert.Equal(t, "query2 desc", policies[0].Description)
	require.NotNil(t, policies[0].AuthorID)
	assert.Equal(t, user1.ID, *policies[0].AuthorID)
	assert.Equal(t, "Alice", policies[0].AuthorName)
	assert.Equal(t, "alice@example.com", policies[0].AuthorEmail)
	assert.NotNil(t, policies[0].Resolution)
	assert.Equal(t, "some other gp resolution", *policies[0].Resolution)

	checkGlobaPolicy(policies[1])
	assert.Equal(t, "", policies[1].Response)

	policies, err = ds.ListPoliciesForHost(context.Background(), host2)
	require.NoError(t, err)
	require.Len(t, policies, 1)

	checkGlobaPolicy(policies[0])
	assert.Equal(t, "", policies[0].Response)

	// Global policy ran with passing result.
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host2, map[uint]*bool{gp.ID: ptr.Bool(true)}, time.Now(), false))

	policies, err = ds.ListPoliciesForHost(context.Background(), host2)
	require.NoError(t, err)
	require.Len(t, policies, 1)

	checkGlobaPolicy(policies[0])

	assert.Equal(t, "pass", policies[0].Response)

	// Manually insert a global policy with null resolution.
	res, err := ds.writer(context.Background()).ExecContext(
		context.Background(),
		fmt.Sprintf(`INSERT INTO policies (name, query, description, checksum) VALUES (?, ?, ?, %s)`, policiesChecksumComputedColumn()),
		q.Name+"2", q.Query, q.Description+"2",
	)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host2, map[uint]*bool{uint(id): nil}, time.Now(), false))

	policies, err = ds.ListPoliciesForHost(context.Background(), host2)
	require.NoError(t, err)
	require.Len(t, policies, 2)

	// Global policy with null resolution is listed first, followed by passing policy.
	assert.Equal(t, "query1 desc2", policies[0].Description)
	assert.NotNil(t, policies[0].Resolution)
	assert.Empty(t, *policies[0].Resolution)

	assert.NotNil(t, policies[1].Resolution)
	assert.Equal(t, "some gp resolution", *policies[1].Resolution)
}

func testPoliciesByID(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	policy1 := newTestPolicy(t, ds, user1, "policy1", "darwin", nil)
	_ = newTestPolicy(t, ds, user1, "policy2", "darwin", nil)
	host1 := newTestHostWithPlatform(t, ds, "host1", "darwin", nil)

	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), host1, map[uint]*bool{policy1.ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, ds.UpdateHostPolicyCounts(context.Background()))

	policiesByID, err := ds.PoliciesByID(context.Background(), []uint{1, 2})
	require.NoError(t, err)
	assert.Equal(t, len(policiesByID), 2)
	assert.Equal(t, policiesByID[1].ID, policy1.ID)
	assert.Equal(t, policiesByID[1].Name, policy1.Name)
	assert.Equal(t, policiesByID[2].ID, uint(2))
	assert.Equal(t, policiesByID[2].Name, "policy2")
	assert.Equal(t, uint(1), policiesByID[1].PassingHostCount)

	_, err = ds.PoliciesByID(context.Background(), []uint{1, 2, 3})
	require.Error(t, err)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)
}

func testTeamPolicyTransfer(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team2"})
	require.NoError(t, err)

	host1, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("1234"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
	})
	require.NoError(t, err)
	host2, err := ds.EnrollHost(ctx, false, "2", "", "", "2", &team1.ID, 0)
	require.NoError(t, err)

	require.NoError(t, ds.AddHostsToTeam(ctx, &team1.ID, []uint{host1.ID}))
	host1, err = ds.Host(ctx, host1.ID)
	require.NoError(t, err)

	tq, err := ds.NewQuery(ctx, &fleet.Query{
		Name:        "query1",
		Description: "query1 desc",
		Query:       "select 1;",
		Saved:       true,
		Logging:     fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	team1Policy, err := ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, fleet.PolicyPayload{
		QueryID: &tq.ID,
	})
	require.NoError(t, err)

	gq, err := ds.NewQuery(ctx, &fleet.Query{
		Name:        "query2",
		Description: "query2 desc",
		Query:       "select 2;",
		Saved:       true,
		Logging:     fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	globalPolicy, err := ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		QueryID: &gq.ID,
	})
	require.NoError(t, err)

	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, host1, map[uint]*bool{team1Policy.ID: ptr.Bool(false), globalPolicy.ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, host1, map[uint]*bool{team1Policy.ID: ptr.Bool(true), globalPolicy.ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, host2, map[uint]*bool{team1Policy.ID: ptr.Bool(false), globalPolicy.ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, host2, map[uint]*bool{team1Policy.ID: ptr.Bool(true), globalPolicy.ID: ptr.Bool(true)}, time.Now(), false))

	require.NoError(t, ds.UpdateHostPolicyCounts(ctx))

	checkPassingCount := func(tm1, tm1Inherited, tm2Inherited, global uint) {
		t.Helper()
		require.NoError(t, ds.UpdateHostPolicyCounts(ctx))
		policies, inherited, err := ds.ListTeamPolicies(ctx, team1.ID, fleet.ListOptions{}, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, policies, 1)
		assert.Equal(t, tm1, policies[0].PassingHostCount)
		require.Len(t, inherited, 1)
		assert.Equal(t, tm1Inherited, inherited[0].PassingHostCount)

		policies, inherited, err = ds.ListTeamPolicies(ctx, team2.ID, fleet.ListOptions{}, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, policies, 0) // team 2 has no policies of its own
		require.Len(t, inherited, 1)
		assert.Equal(t, tm2Inherited, inherited[0].PassingHostCount)

		policies, err = ds.ListGlobalPolicies(ctx, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, policies, 1)
		assert.Equal(t, global, policies[0].PassingHostCount)
	}

	// both hosts belong to team1 and pass the team and the global policy
	checkPassingCount(2, 2, 0, 2)

	// team policies are removed when AddHostsToTeam is called
	require.NoError(t, ds.AddHostsToTeam(ctx, ptr.Uint(team2.ID), []uint{host1.ID}))
	// host2 passes tm1 and the global (so team1's inherited too), host1 passes the team2's inherited and the global
	checkPassingCount(1, 1, 1, 2)

	// all host policies are removed when a host is enrolled in the same team
	_, err = ds.EnrollHost(ctx, false, "2", "", "", "2", &team1.ID, 0)
	require.NoError(t, err)
	checkPassingCount(0, 0, 1, 1)

	// team policies are removed if the host is enrolled in a different team
	_, err = ds.EnrollHost(ctx, false, "2", "", "", "2", &team2.ID, 0)
	require.NoError(t, err)
	// both hosts are now in team2
	checkPassingCount(0, 0, 1, 1)

	// team policies are removed if the host is re-enrolled without a team
	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, host2, map[uint]*bool{team1Policy.ID: ptr.Bool(true), globalPolicy.ID: ptr.Bool(true)}, time.Now(), false))
	checkPassingCount(1, 0, 2, 2)

	// all host policies are removed when a host is re-enrolled
	_, err = ds.EnrollHost(ctx, false, "2", "", "", "2", nil, 0)
	require.NoError(t, err)
	checkPassingCount(0, 0, 1, 1)
}

func testApplyPolicySpec(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "User1", "user1@example.com", true)
	ctx := context.Background()
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	require.NoError(t, ds.ApplyPolicySpecs(ctx, user1.ID, []*fleet.PolicySpec{
		{
			Name:        "query1",
			Query:       "select 1;",
			Description: "query1 desc",
			Resolution:  "some resolution",
			Team:        "",
			Platform:    "",
		},
		{
			Name:        "query2",
			Query:       "select 2;",
			Description: "query2 desc",
			Resolution:  "some other resolution",
			Team:        "team1",
			Platform:    "darwin",
		},
		{
			Name:        "query3",
			Query:       "select 3;",
			Description: "query3 desc",
			Resolution:  "some other good resolution",
			Team:        "team1",
			Platform:    "windows,linux",
		},
	}))

	policies, err := ds.ListGlobalPolicies(ctx, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, policies, 1)
	assert.Equal(t, "query1", policies[0].Name)
	assert.Equal(t, "select 1;", policies[0].Query)
	assert.Equal(t, "query1 desc", policies[0].Description)
	require.NotNil(t, policies[0].AuthorID)
	assert.Equal(t, user1.ID, *policies[0].AuthorID)
	require.NotNil(t, policies[0].Resolution)
	assert.Equal(t, "some resolution", *policies[0].Resolution)
	assert.Equal(t, "", policies[0].Platform)

	teamPolicies, _, err := ds.ListTeamPolicies(ctx, team1.ID, fleet.ListOptions{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, teamPolicies, 2)
	assert.Equal(t, "query2", teamPolicies[0].Name)
	assert.Equal(t, "select 2;", teamPolicies[0].Query)
	assert.Equal(t, "query2 desc", teamPolicies[0].Description)
	require.NotNil(t, teamPolicies[0].AuthorID)
	assert.Equal(t, user1.ID, *teamPolicies[0].AuthorID)
	require.NotNil(t, teamPolicies[0].Resolution)
	assert.Equal(t, "some other resolution", *teamPolicies[0].Resolution)
	assert.Equal(t, "darwin", teamPolicies[0].Platform)

	assert.Equal(t, "query3", teamPolicies[1].Name)
	assert.Equal(t, "select 3;", teamPolicies[1].Query)
	assert.Equal(t, "query3 desc", teamPolicies[1].Description)
	require.NotNil(t, teamPolicies[1].AuthorID)
	assert.Equal(t, user1.ID, *teamPolicies[1].AuthorID)
	require.NotNil(t, teamPolicies[1].Resolution)
	assert.Equal(t, "some other good resolution", *teamPolicies[1].Resolution)
	assert.Equal(t, "windows,linux", teamPolicies[1].Platform)

	// Make sure apply is idempotent
	require.NoError(t, ds.ApplyPolicySpecs(ctx, user1.ID, []*fleet.PolicySpec{
		{
			Name:        "query1",
			Query:       "select 1;",
			Description: "query1 desc",
			Resolution:  "some resolution",
			Team:        "",
			Platform:    "",
		},
		{
			Name:        "query2",
			Query:       "select 2;",
			Description: "query2 desc",
			Resolution:  "some other resolution",
			Team:        "team1",
			Platform:    "darwin",
		},
		{
			Name:        "query3",
			Query:       "select 3;",
			Description: "query3 desc",
			Resolution:  "some other good resolution",
			Team:        "team1",
			Platform:    "windows,linux",
		},
	}))

	policies, err = ds.ListGlobalPolicies(ctx, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, policies, 1)
	teamPolicies, _, err = ds.ListTeamPolicies(ctx, team1.ID, fleet.ListOptions{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, teamPolicies, 2)

	// Test policy updating.
	require.NoError(t, ds.ApplyPolicySpecs(ctx, user1.ID, []*fleet.PolicySpec{
		{
			Name:        "query1",
			Query:       "select 1 from updated;",
			Description: "query1 desc updated",
			Resolution:  "some resolution updated",
			Team:        "", // No error, team did not change
			Platform:    "",
		},
		{
			Name:        "query2",
			Query:       "select 2 from updated;",
			Description: "query2 desc updated",
			Resolution:  "some other resolution updated",
			Team:        "team1", // No error, team did not change
			Platform:    "windows",
		},
	}))
	policies, err = ds.ListGlobalPolicies(ctx, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, policies, 1)

	assert.Equal(t, "query1", policies[0].Name)
	assert.Equal(t, "select 1 from updated;", policies[0].Query)
	assert.Equal(t, "query1 desc updated", policies[0].Description)
	require.NotNil(t, policies[0].AuthorID)
	assert.Equal(t, user1.ID, *policies[0].AuthorID)
	require.NotNil(t, policies[0].Resolution)
	assert.Equal(t, "some resolution updated", *policies[0].Resolution)
	assert.Equal(t, "", policies[0].Platform)

	teamPolicies, _, err = ds.ListTeamPolicies(ctx, team1.ID, fleet.ListOptions{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, teamPolicies, 2)

	assert.Equal(t, "query2", teamPolicies[0].Name)
	assert.Equal(t, "select 2 from updated;", teamPolicies[0].Query)
	assert.Equal(t, "query2 desc updated", teamPolicies[0].Description)
	require.NotNil(t, teamPolicies[0].AuthorID)
	assert.Equal(t, user1.ID, *teamPolicies[0].AuthorID)
	assert.Equal(t, team1.ID, *teamPolicies[0].TeamID)
	require.NotNil(t, teamPolicies[0].Resolution)
	assert.Equal(t, "some other resolution updated", *teamPolicies[0].Resolution)
	assert.Equal(t, "windows", teamPolicies[0].Platform)

	// Creating the same policy for a different team is allowed.
	require.NoError(
		t, ds.ApplyPolicySpecs(
			ctx, user1.ID, []*fleet.PolicySpec{
				{
					Name:        "query1",
					Query:       "select 1 from updated again;",
					Description: "query1 desc updated again",
					Resolution:  "some resolution updated again",
					Team:        "team1",
					Platform:    "",
				},
			}))
}

func testPoliciesSave(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "User1", "user1@example.com", true)
	ctx := context.Background()
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	err = ds.SavePolicy(ctx, &fleet.Policy{
		PolicyData: fleet.PolicyData{
			ID:    99999999,
			Name:  "non-existent query",
			Query: "select 1;",
		},
	}, false)
	require.Error(t, err)
	var nfe *notFoundError
	require.True(t, errors.As(err, &nfe))

	payload := fleet.PolicyPayload{
		Name:        "global query",
		Query:       "select 1;",
		Description: "global query desc",
		Resolution:  "global query resolution",
	}
	gp, err := ds.NewGlobalPolicy(ctx, &user1.ID, payload)
	require.NoError(t, err)
	require.Equal(t, gp.Name, payload.Name)
	require.Equal(t, gp.Query, payload.Query)
	require.Equal(t, gp.Description, payload.Description)
	require.Equal(t, *gp.Resolution, payload.Resolution)
	require.Equal(t, gp.Critical, payload.Critical)
	computeChecksum := func(policy fleet.Policy) string {
		h := md5.New() //nolint:gosec // (only used for tests)
		// Compute the same way as DB does.
		teamStr := ""
		if policy.TeamID != nil {
			teamStr = fmt.Sprint(*policy.TeamID)
		}
		cols := []string{teamStr, policy.Name}
		_, _ = fmt.Fprint(h, strings.Join(cols, "\x00"))
		checksum := h.Sum(nil)
		return hex.EncodeToString(checksum)
	}

	var globalChecksum []uint8
	err = ds.writer(context.Background()).Get(&globalChecksum, `SELECT checksum FROM policies WHERE id = ?`, gp.ID)
	require.NoError(t, err)
	assert.Equal(t, computeChecksum(*gp), hex.EncodeToString(globalChecksum))

	payload = fleet.PolicyPayload{
		Name:        "team1 query",
		Query:       "select 2;",
		Description: "team1 query desc",
		Resolution:  "team1 query resolution",
		Critical:    true,
	}
	tp1, err := ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, payload)
	require.NoError(t, err)
	require.Equal(t, tp1.Name, payload.Name)
	require.Equal(t, tp1.Query, payload.Query)
	require.Equal(t, tp1.Description, payload.Description)
	require.Equal(t, *tp1.Resolution, payload.Resolution)
	require.Equal(t, tp1.Critical, payload.Critical)
	var teamChecksum []uint8
	err = ds.writer(context.Background()).Get(&teamChecksum, `SELECT checksum FROM policies WHERE id = ?`, tp1.ID)
	require.NoError(t, err)
	assert.Equal(t, computeChecksum(*tp1), hex.EncodeToString(teamChecksum))

	// Change name only of a global query.
	gp2 := *gp
	gp2.Name = "global query updated"
	gp2.Critical = true
	err = ds.SavePolicy(ctx, &gp2, false)
	require.NoError(t, err)
	gp, err = ds.Policy(ctx, gp.ID)
	require.NoError(t, err)
	gp2.UpdateCreateTimestamps = gp.UpdateCreateTimestamps
	require.Equal(t, &gp2, gp)
	var globalChecksum2 []uint8
	err = ds.writer(context.Background()).Get(&globalChecksum2, `SELECT checksum FROM policies WHERE id = ?`, gp.ID)
	require.NoError(t, err)
	assert.NotEqual(t, globalChecksum, globalChecksum2, "Checksum should be different since policy name changed")
	assert.Equal(t, computeChecksum(*gp), hex.EncodeToString(globalChecksum2))

	// Change name, query, description and resolution of a team policy.
	tp2 := *tp1
	tp2.Name = "team1 query updated"
	tp2.Query = "select 12;"
	tp2.Description = "team1 query desc updated"
	tp2.Resolution = ptr.String("team1 query resolution updated")
	tp2.Critical = false
	err = ds.SavePolicy(ctx, &tp2, true)
	require.NoError(t, err)
	tp1, err = ds.Policy(ctx, tp1.ID)
	tp2.UpdateCreateTimestamps = tp1.UpdateCreateTimestamps
	require.NoError(t, err)
	require.Equal(t, tp1, &tp2)
	var teamChecksum2 []uint8
	err = ds.writer(context.Background()).Get(&teamChecksum2, `SELECT checksum FROM policies WHERE id = ?`, tp1.ID)
	require.NoError(t, err)
	assert.NotEqual(t, teamChecksum, teamChecksum2, "Checksum should be different since policy name changed")
	assert.Equal(t, computeChecksum(*tp1), hex.EncodeToString(teamChecksum2))

	loadMembershipStmt, args, err := sqlx.In(`SELECT policy_id, host_id FROM policy_membership WHERE policy_id = ?`, tp2.ID)
	require.NoError(t, err)

	type polHostIDs struct {
		PolicyID uint `db:"policy_id"`
		HostID   uint `db:"host_id"`
	}
	var rows []polHostIDs
	err = ds.writer(context.Background()).SelectContext(context.Background(), &rows, loadMembershipStmt, args...)
	require.NoError(t, err)
	require.Len(t, rows, 0)
}

func testCachedPolicyCountDeletesOnPolicyChange(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	teamHost, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("test-1"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("test-1"),
		UUID:            "test-1",
		Hostname:        "foo.local",
		Platform:        "windows",
	})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(ctx, &team1.ID, []uint{teamHost.ID}))

	globalHost, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("test-2"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("test-2"),
		UUID:            "test-2",
		Hostname:        "foo.local",
		Platform:        "windows",
	})
	require.NoError(t, err)

	globalPolicy, err := ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:        "global query",
		Query:       "select 1;",
		Description: "global query desc",
		Resolution:  "global query resolution",
	})
	require.NoError(t, err)

	teamPolicy, err := ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, fleet.PolicyPayload{
		Name:        "team query",
		Query:       "select 1;",
		Description: "team query desc",
		Resolution:  "team query resolution",
	})
	require.NoError(t, err)

	// teamHost and globalHost pass all policies
	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, teamHost, map[uint]*bool{globalPolicy.ID: ptr.Bool(true), globalPolicy.ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, teamHost, map[uint]*bool{teamPolicy.ID: ptr.Bool(true), teamPolicy.ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, globalHost, map[uint]*bool{globalPolicy.ID: ptr.Bool(true), globalPolicy.ID: ptr.Bool(true)}, time.Now(), false))

	err = ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)

	globalPolicy, err = ds.Policy(ctx, globalPolicy.ID)
	require.NoError(t, err)
	assert.Equal(t, uint(2), globalPolicy.PassingHostCount)
	teamPolicies, inheritedPolicies, err := ds.ListTeamPolicies(ctx, team1.ID, fleet.ListOptions{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, teamPolicies, 1)
	require.Len(t, inheritedPolicies, 1)
	assert.Equal(t, uint(1), teamPolicies[0].PassingHostCount)
	assert.Equal(t, uint(1), inheritedPolicies[0].PassingHostCount)

	// Update the global policy sql to trigger a cache invalidation
	err = ds.SavePolicy(ctx, globalPolicy, true)
	require.NoError(t, err)

	globalPolicy, err = ds.Policy(ctx, globalPolicy.ID)
	require.NoError(t, err)
	assert.Equal(t, uint(0), globalPolicy.PassingHostCount)
	teamPolicies, inheritedPolicies, err = ds.ListTeamPolicies(ctx, team1.ID, fleet.ListOptions{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, teamPolicies, 1)
	require.Len(t, inheritedPolicies, 1)
	assert.Equal(t, uint(1), teamPolicies[0].PassingHostCount)
	assert.Equal(t, uint(0), inheritedPolicies[0].PassingHostCount)

	// Update the team policy sql to trigger a cache invalidation
	err = ds.SavePolicy(ctx, teamPolicy, true)
	require.NoError(t, err)

	teamPolicies, inheritedPolicies, err = ds.ListTeamPolicies(ctx, team1.ID, fleet.ListOptions{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, teamPolicies, 1)
	require.Len(t, inheritedPolicies, 1)
	assert.Equal(t, uint(0), teamPolicies[0].PassingHostCount)
	assert.Equal(t, uint(0), inheritedPolicies[0].PassingHostCount)
}

func testPoliciesDelUser(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "User1", "user1@example.com", true)
	user2 := test.NewUser(t, ds, "User2", "user2@example.com", true)
	ctx := context.Background()
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	gp, err := ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:        "global query",
		Query:       "select 1;",
		Description: "global query desc",
		Resolution:  "global query resolution",
	})
	require.NoError(t, err)
	tp, err := ds.NewTeamPolicy(ctx, team1.ID, &user2.ID, fleet.PolicyPayload{
		Name:        "team1 query",
		Query:       "select 2;",
		Description: "team1 query desc",
		Resolution:  "team1 query resolution",
	})
	require.NoError(t, err)

	err = ds.DeleteUser(ctx, user1.ID)
	require.NoError(t, err)
	err = ds.DeleteUser(ctx, user2.ID)
	require.NoError(t, err)

	tp, err = ds.Policy(ctx, tp.ID)
	require.NoError(t, err)
	assert.Nil(t, tp.AuthorID)
	assert.Equal(t, "<deleted>", tp.AuthorName)
	assert.Empty(t, tp.AuthorEmail)

	gp, err = ds.Policy(ctx, gp.ID)
	require.NoError(t, err)
	assert.Nil(t, gp.AuthorID)
	assert.Equal(t, "<deleted>", gp.AuthorName)
	assert.Empty(t, gp.AuthorEmail)
}

func testPolicyByName(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "User1", "user1@example.com", true)
	ctx := context.Background()

	gp, err := ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:        "global query",
		Query:       "select 1;",
		Description: "global query desc",
		Resolution:  "global query resolution",
	})
	require.NoError(t, err)

	policy, err := ds.PolicyByName(ctx, "global query")
	require.NoError(t, err)
	assert.Equal(t, gp.ID, policy.ID)

	policy, err = ds.PolicyByName(ctx, "non-existent")
	require.Error(t, sql.ErrNoRows, err)
	assert.Nil(t, policy)
}

func testFlippingPoliciesForHost(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	ctx := context.Background()
	host1, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("test-1"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("test-1"),
		UUID:            "test-1",
		Hostname:        "foo.local",
		Platform:        "windows",
	})
	require.NoError(t, err)
	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	p1, err := ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:  "policy1",
		Query: "select 41;",
	})
	require.NoError(t, err)
	p2, err := ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, fleet.PolicyPayload{
		Name:  "policy2",
		Query: "select 42;",
	})
	require.NoError(t, err)
	// Create some unused policy.
	_, err = ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:  "policy3",
		Query: "select 43;",
	})
	require.NoError(t, err)
	pfailed, err := ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, fleet.PolicyPayload{
		Name:  "policy_failed",
		Query: "select * from unexistent_table;",
	})
	require.NoError(t, err)
	p4, err := ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, fleet.PolicyPayload{
		Name:  "policy_failed_to_run_then_pass",
		Query: "select 42;",
	})
	require.NoError(t, err)
	p5, err := ds.NewTeamPolicy(ctx, team1.ID, &user1.ID, fleet.PolicyPayload{
		Name:  "policy_failed_to_run_then_fail",
		Query: "select 42;",
	})
	require.NoError(t, err)

	// Unknown policies will be considered their first execution.
	newFailing, newPassing, err := ds.FlippingPoliciesForHost(ctx, host1.ID, map[uint]*bool{
		99997: nil, // considered as didn't run.
		99998: ptr.Bool(false),
		99999: ptr.Bool(true),
	})
	require.NoError(t, err)
	sort.Slice(newFailing, func(i, j int) bool {
		return newFailing[i] < newFailing[j]
	})
	require.Equal(t, []uint{99998}, newFailing)
	require.Empty(t, newPassing) // because this would be the first run.

	// Unknown host.
	newFailing, newPassing, err = ds.FlippingPoliciesForHost(ctx, 99999, map[uint]*bool{
		p1.ID: ptr.Bool(false),
	})
	require.NoError(t, err)
	require.Equal(t, []uint{p1.ID}, newFailing)
	require.Empty(t, newPassing)

	// Empty incoming results.
	newFailing, newPassing, err = ds.FlippingPoliciesForHost(ctx, host1.ID, map[uint]*bool{})
	require.NoError(t, err)
	require.Empty(t, newFailing)
	require.Empty(t, newPassing)

	// incoming policy 1 with first new failing result: => no
	// incoming policy 2 with first new passing result: => yes
	incoming := map[uint]*bool{
		p1.ID: ptr.Bool(false),
		p2.ID: ptr.Bool(true),
	}
	newFailing, newPassing, err = ds.FlippingPoliciesForHost(ctx, host1.ID, incoming)
	require.NoError(t, err)
	require.Equal(t, []uint{p1.ID}, newFailing)
	require.Empty(t, newPassing) // because this would be the first run.

	// Record the above executions.
	err = ds.RecordPolicyQueryExecutions(ctx, host1, incoming, time.Now(), false)
	require.NoError(t, err)

	// incoming policy 1 with passing result: no => yes
	// incoming policy 2 with failing result: yes => no
	incoming = map[uint]*bool{
		p1.ID: ptr.Bool(true),
		p2.ID: ptr.Bool(false),
	}
	newFailing, newPassing, err = ds.FlippingPoliciesForHost(ctx, host1.ID, incoming)
	require.NoError(t, err)
	require.Equal(t, []uint{p2.ID}, newFailing)
	require.Equal(t, []uint{p1.ID}, newPassing)

	// Record the above executions.
	err = ds.RecordPolicyQueryExecutions(ctx, host1, incoming, time.Now(), false)
	require.NoError(t, err)

	// incoming policy 1 with passing result: yes => yes
	// incoming policy 2 with failing result: no => no
	incoming = map[uint]*bool{
		p1.ID: ptr.Bool(true),
		p2.ID: ptr.Bool(false),
	}
	newFailing, newPassing, err = ds.FlippingPoliciesForHost(ctx, host1.ID, incoming)
	require.NoError(t, err)
	require.Empty(t, newFailing)
	require.Empty(t, newPassing)

	// Record the above executions.
	err = ds.RecordPolicyQueryExecutions(ctx, host1, incoming, time.Now(), false)
	require.NoError(t, err)

	// incoming policy 1 failed to execute: yes => no
	// incoming policy 2 failed to execute: no => no
	incoming = map[uint]*bool{
		p1.ID: ptr.Bool(false),
		p2.ID: ptr.Bool(false),
	}
	newFailing, newPassing, err = ds.FlippingPoliciesForHost(ctx, host1.ID, incoming)
	require.NoError(t, err)
	require.Equal(t, []uint{p1.ID}, newFailing)
	require.Empty(t, newPassing)

	// incoming pfailed failed to execute: ---
	incoming = map[uint]*bool{
		pfailed.ID: nil,
	}
	newFailing, newPassing, err = ds.FlippingPoliciesForHost(ctx, host1.ID, incoming)
	require.NoError(t, err)
	require.Empty(t, newFailing)
	require.Empty(t, newPassing)

	// Record the above executions.
	err = ds.RecordPolicyQueryExecutions(ctx, host1, incoming, time.Now(), false)
	require.NoError(t, err)

	// incoming pfailed again failed to execute: --- -> ---
	newFailing, newPassing, err = ds.FlippingPoliciesForHost(ctx, host1.ID, incoming)
	require.NoError(t, err)
	require.Empty(t, newFailing)
	require.Empty(t, newPassing)

	// incoming policy 4 failed to run: => ---
	// incoming policy 5 failed to run: => ---
	incoming = map[uint]*bool{
		p4.ID: nil,
		p5.ID: nil,
	}
	newFailing, newPassing, err = ds.FlippingPoliciesForHost(ctx, host1.ID, incoming)
	require.NoError(t, err)
	require.Empty(t, newFailing)
	require.Empty(t, newPassing)

	// Record the above executions.
	err = ds.RecordPolicyQueryExecutions(ctx, host1, incoming, time.Now(), false)
	require.NoError(t, err)

	// incoming policy 4 with first new failing result: --- => no
	// incoming policy 5 with first new passing result: --- => yes
	incoming = map[uint]*bool{
		p4.ID: ptr.Bool(false),
		p5.ID: ptr.Bool(true),
	}
	newFailing, newPassing, err = ds.FlippingPoliciesForHost(ctx, host1.ID, incoming)
	require.NoError(t, err)
	require.Equal(t, []uint{p4.ID}, newFailing)
	require.Empty(t, newPassing) // because this would be the first run.

	// incoming policy 4 now fails to execute: no => ---
	// incoming policy 5 now fails to execute: yes => ---
	incoming = map[uint]*bool{
		p4.ID: nil,
		p5.ID: nil,
	}
	newFailing, newPassing, err = ds.FlippingPoliciesForHost(ctx, host1.ID, incoming)
	require.NoError(t, err)
	require.Empty(t, newFailing)
	require.Empty(t, newPassing)
}

func testPolicyPlatformUpdate(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user := test.NewUser(t, ds, "Alice", "alice@example.com", true)
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	const hostWin, hostMac, hostDeb, hostLin = 0, 1, 2, 3
	platforms := []string{"windows", "darwin", "debian", "linux"}

	// create hosts with different platforms, for that team
	teamHosts := make([]*fleet.Host, len(platforms))
	for i, pl := range platforms {
		id := fmt.Sprintf("%s-%d", strings.ReplaceAll(t.Name(), "/", "_"), i)
		h, err := ds.NewHost(ctx, &fleet.Host{
			OsqueryHostID:   &id,
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         &id,
			UUID:            id,
			Hostname:        id,
			Platform:        pl,
			TeamID:          ptr.Uint(tm.ID),
		})
		require.NoError(t, err)
		teamHosts[i] = h
	}

	// create hosts with different platforms, without team
	globalHosts := make([]*fleet.Host, len(platforms))
	for i, pl := range platforms {
		id := fmt.Sprintf("g%s-%d", strings.ReplaceAll(t.Name(), "/", "_"), i)
		h, err := ds.NewHost(ctx, &fleet.Host{
			OsqueryHostID:   &id,
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         &id,
			UUID:            id,
			Hostname:        id,
			Platform:        pl,
		})
		require.NoError(t, err)
		globalHosts[i] = h
	}

	// new global policy for any platform
	_, err = ds.NewGlobalPolicy(ctx, ptr.Uint(user.ID), fleet.PolicyPayload{Name: "g1", Query: "select 1", Platform: ""})
	require.NoError(t, err)
	// new team policy for any platform
	_, err = ds.NewTeamPolicy(ctx, tm.ID, ptr.Uint(user.ID), fleet.PolicyPayload{Name: "t1", Query: "select 1", Platform: ""})
	require.NoError(t, err)

	// new global and team policies for Linux, via apply spec
	err = ds.ApplyPolicySpecs(ctx, user.ID, []*fleet.PolicySpec{
		{Name: "g2", Query: "select 2", Platform: "linux"},
		{Name: "t2", Query: "select 2", Team: tm.Name, Platform: "linux"},
	})
	require.NoError(t, err)

	// load the global policies
	gpols, err := ds.ListGlobalPolicies(ctx, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, gpols, 2)
	// load the team policies
	tpols, _, err := ds.ListTeamPolicies(ctx, tm.ID, fleet.ListOptions{}, fleet.ListOptions{})
	require.NoError(t, err)
	require.Len(t, tpols, 2)

	// index the policies by name for easier access in the rest of the test
	polsByName := make(map[string]*fleet.Policy, len(gpols)+len(tpols))
	for _, tpol := range tpols {
		polsByName[tpol.Name] = tpol
	}
	for _, gpol := range gpols {
		polsByName[gpol.Name] = gpol
	}

	// updating without change works fine
	err = ds.SavePolicy(ctx, polsByName["g1"], false)
	require.NoError(t, err)
	err = ds.SavePolicy(ctx, polsByName["t2"], false)
	require.NoError(t, err)
	// apply specs that result in an update (without change) works fine
	err = ds.ApplyPolicySpecs(ctx, user.ID, []*fleet.PolicySpec{
		{Name: polsByName["g2"].Name, Query: polsByName["g2"].Query, Platform: polsByName["g2"].Platform},
		{Name: polsByName["t1"].Name, Query: polsByName["t1"].Query, Team: tm.Name, Platform: polsByName["t1"].Platform},
	})
	require.NoError(t, err)

	pol, err := ds.Policy(ctx, polsByName["g2"].ID)
	require.NoError(t, err)
	require.Equal(t, polsByName["g2"], pol)
	pol, err = ds.Policy(ctx, polsByName["t1"].ID)
	require.NoError(t, err)
	require.Equal(t, polsByName["t1"], pol)

	// record some results for each policy
	for i, h := range teamHosts {
		res := map[uint]*bool{
			polsByName["t1"].ID: ptr.Bool(true),
		}
		if i == hostDeb || i == hostLin {
			// also record a result for linux policy
			res[polsByName["t2"].ID] = ptr.Bool(true)
		}
		err = ds.RecordPolicyQueryExecutions(ctx, h, res, time.Now(), false)
		require.NoError(t, err)
	}
	for i, h := range globalHosts {
		res := map[uint]*bool{
			polsByName["g1"].ID: ptr.Bool(true),
		}
		if i == hostDeb || i == hostLin {
			// also record a result for linux policy
			res[polsByName["g2"].ID] = ptr.Bool(true)
		}
		err = ds.RecordPolicyQueryExecutions(ctx, h, res, time.Now(), false)
		require.NoError(t, err)
	}

	wantHostsByPol := map[string][]uint{
		"g1": {globalHosts[hostWin].ID, globalHosts[hostMac].ID, globalHosts[hostDeb].ID, globalHosts[hostLin].ID},
		"g2": {globalHosts[hostDeb].ID, globalHosts[hostLin].ID},
		"t1": {teamHosts[hostWin].ID, teamHosts[hostMac].ID, teamHosts[hostDeb].ID, teamHosts[hostLin].ID},
		"t2": {teamHosts[hostDeb].ID, teamHosts[hostLin].ID},
	}
	assertPolicyMembership(t, ds, polsByName, wantHostsByPol)

	// update global policy g1 from any => linux
	g1 := polsByName["g1"]
	g1.Platform = "linux"
	polsByName["g1"] = g1
	err = ds.SavePolicy(ctx, g1, false)
	require.NoError(t, err)
	wantHostsByPol["g1"] = []uint{globalHosts[hostDeb].ID, globalHosts[hostLin].ID}
	assertPolicyMembership(t, ds, polsByName, wantHostsByPol)

	// update team policy t1 from any => windows, darwin
	t1 := polsByName["t1"]
	t1.Platform = "windows,darwin"
	polsByName["t1"] = t1
	err = ds.SavePolicy(ctx, t1, false)
	require.NoError(t, err)
	wantHostsByPol["t1"] = []uint{teamHosts[hostWin].ID, teamHosts[hostMac].ID}
	assertPolicyMembership(t, ds, polsByName, wantHostsByPol)

	// update g2 from linux => any, t2 from linux => debian, via ApplySpecs
	t2, g2 := polsByName["t2"], polsByName["g2"]
	g2.Platform = ""
	t2.Platform = "debian"
	polsByName["t2"], polsByName["g2"] = t2, g2
	err = ds.ApplyPolicySpecs(ctx, user.ID, []*fleet.PolicySpec{
		{Name: g2.Name, Query: g2.Query, Platform: g2.Platform},
		{Name: t2.Name, Query: t2.Query, Team: tm.Name, Platform: t2.Platform},
	})
	require.NoError(t, err)
	// nothing should've changed for g2 (platform changed to any, so nothing to cleanup),
	// while t2 should now only accept debian
	wantHostsByPol["t2"] = []uint{teamHosts[hostDeb].ID}
	assertPolicyMembership(t, ds, polsByName, wantHostsByPol)
}

func assertPolicyMembership(t *testing.T, ds *Datastore, polsByName map[string]*fleet.Policy, wantPolNameToHostIDs map[string][]uint) {
	policyIDs := make([]uint, 0, len(polsByName))
	for _, pol := range polsByName {
		policyIDs = append(policyIDs, pol.ID)
	}
	loadMembershipStmt, args, err := sqlx.In(`SELECT policy_id, host_id FROM policy_membership WHERE policy_id IN (?)`, policyIDs)
	require.NoError(t, err)

	type polHostIDs struct {
		PolicyID uint `db:"policy_id"`
		HostID   uint `db:"host_id"`
	}
	var rows []polHostIDs
	err = ds.writer(context.Background()).SelectContext(context.Background(), &rows, loadMembershipStmt, args...)
	require.NoError(t, err)

	// index the host IDs by policy ID
	hostIDsByPolID := make(map[uint][]uint, len(policyIDs))
	for _, row := range rows {
		hostIDsByPolID[row.PolicyID] = append(hostIDsByPolID[row.PolicyID], row.HostID)
	}

	// assert that they match the expected list of hosts by policy
	for polNm, hostIDs := range wantPolNameToHostIDs {
		pol, ok := polsByName[polNm]
		if !ok {
			require.Len(t, hostIDs, 0)
			continue
		}
		got := hostIDsByPolID[pol.ID]
		require.ElementsMatch(t, hostIDs, got)
	}
}

func testPolicyViolationDays(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	then := time.Now().Add(-48 * time.Hour)

	setStatsTimestampDB := func(updatedAt time.Time) error {
		_, err := ds.writer(ctx).ExecContext(ctx, `
			UPDATE aggregated_stats SET created_at = ?, updated_at = ? WHERE id = ? AND global_stats = ? AND type = ?
		`, then, updatedAt, 0, true, aggregatedStatsTypePolicyViolationsDays)
		return err
	}

	user := test.NewUser(t, ds, "Bob", "bob@example.com", true)

	hosts := make([]*fleet.Host, 3)
	for i, name := range []string{"h1", "h2", "h3"} {
		id := fmt.Sprintf("%s-%d", strings.ReplaceAll(t.Name(), "/", "_"), i)
		h, err := ds.NewHost(ctx, &fleet.Host{
			OsqueryHostID:   &id,
			DetailUpdatedAt: then,
			LabelUpdatedAt:  then,
			PolicyUpdatedAt: then,
			SeenTime:        then,
			NodeKey:         &id,
			UUID:            id,
			Hostname:        name,
		})
		require.NoError(t, err)
		hosts[i] = h
	}

	createPolStmt := fmt.Sprintf(
		`INSERT INTO policies (name, query, description, author_id, platforms, created_at, updated_at, checksum) VALUES (?, ?, '', ?, ?, ?, ?, %s)`,
		policiesChecksumComputedColumn(),
	)
	res, err := ds.writer(ctx).ExecContext(ctx, createPolStmt, "test_pol", "select 1", user.ID, "", then, then)
	require.NoError(t, err)
	id, _ := res.LastInsertId()
	pol, err := ds.Policy(ctx, uint(id))
	require.NoError(t, err)

	require.NoError(t, ds.InitializePolicyViolationDays(ctx)) // sets starting violation count to zero

	// initialize policy statuses: 1 failling, 2 passing
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), hosts[0], map[uint]*bool{pol.ID: ptr.Bool(false)}, then, false))
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), hosts[1], map[uint]*bool{pol.ID: ptr.Bool(true)}, then, false))
	require.NoError(t, ds.RecordPolicyQueryExecutions(context.Background(), hosts[2], map[uint]*bool{pol.ID: ptr.Bool(true)}, then, false))

	// setup db for test: starting counts zero, more than 24h since last updated, one outstanding violation
	require.NoError(t, setStatsTimestampDB(time.Now().Add(-25*time.Hour)))
	require.NoError(t, ds.IncrementPolicyViolationDays(ctx))
	actual, possible, err := amountPolicyViolationDaysDB(ctx, ds.reader(ctx))
	require.NoError(t, err)
	// actual should increment from 0 -> 1 (+1 outstanding violation)
	require.Equal(t, 1, actual)
	// possible should increment from 0 -> 3 (3 total hosts * 1 policy)
	require.Equal(t, 3, possible)
	// reset violation counts to zero for next test
	require.NoError(t, ds.InitializePolicyViolationDays(ctx))

	// setup for test: starting counts zero, less than 24h since last updated, one outstanding violation
	require.NoError(t, setStatsTimestampDB(time.Now().Add(-1*time.Hour)))
	require.NoError(t, ds.IncrementPolicyViolationDays(ctx))
	actual, possible, err = amountPolicyViolationDaysDB(ctx, ds.reader(ctx))
	require.NoError(t, err)
	// count should not increment from zero
	require.Equal(t, 0, actual)
	// possible should not increment from zero
	require.Equal(t, 0, possible)
	// leave counts at zero for next test

	// setup for test: starting count zero, more than 24h since last updated, add second outstanding violation
	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, hosts[1], map[uint]*bool{pol.ID: ptr.Bool(false)}, time.Now(), false))
	require.NoError(t, setStatsTimestampDB(time.Now().Add(-25*time.Hour)))
	require.NoError(t, ds.IncrementPolicyViolationDays(ctx))
	actual, possible, err = amountPolicyViolationDaysDB(ctx, ds.reader(ctx))
	require.NoError(t, err)
	// actual should increment from 0 -> 2 (+2 outstanding violations)
	require.Equal(t, 2, actual) // leave count at two for next test
	// possible should increment from 0 -> 3 (3 total hosts * 1 policy)
	require.Equal(t, 3, possible)
	// leave counts at 2 actual and 3 possible for next test

	// setup for test: starting counts at 2 actual and 3 possible, more than 24h since last updated, resolve one outstaning violation
	require.NoError(t, ds.RecordPolicyQueryExecutions(ctx, hosts[1], map[uint]*bool{pol.ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, setStatsTimestampDB(time.Now().Add(-25*time.Hour)))
	require.NoError(t, ds.IncrementPolicyViolationDays(ctx))
	actual, possible, err = amountPolicyViolationDaysDB(ctx, ds.reader(ctx))
	require.NoError(t, err)
	// actual should increment from 2 -> 3 (+1 outstanding violation)
	require.Equal(t, 3, actual)
	// possible should increment from 3 -> 6 (3 total hosts * 1 policy)
	require.Equal(t, 6, possible)
	// leave counts at 3 actual and 6 possible

	// attempt again immediately after last update, counts should not increment
	require.NoError(t, ds.IncrementPolicyViolationDays(ctx))
	actual, possible, err = amountPolicyViolationDaysDB(ctx, ds.reader(ctx))
	require.NoError(t, err)
	require.Equal(t, 3, actual)
	require.Equal(t, 6, possible)
}

func testPolicyCleanupPolicyMembership(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	user := test.NewUser(t, ds, "Bob", "bob@example.com", true)

	// create hosts with different platforms
	hostWin, hostMac, hostDeb, hostLin := 0, 1, 2, 3
	platforms := []string{"windows", "darwin", "debian", "linux"}
	hosts := make([]*fleet.Host, len(platforms))
	for i, pl := range platforms {
		id := fmt.Sprintf("%s-%d", strings.ReplaceAll(t.Name(), "/", "_"), i)
		h, err := ds.NewHost(ctx, &fleet.Host{
			OsqueryHostID:   &id,
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         &id,
			UUID:            id,
			Hostname:        id,
			Platform:        pl,
		})
		require.NoError(t, err)
		hosts[i] = h
	}

	// create some policies, using direct insert statements to control the timestamps
	createPolStmt := fmt.Sprintf(
		`INSERT INTO policies (name, query, description, author_id, platforms, created_at, updated_at, checksum)
                    VALUES (?, ?, '', ?, ?, ?, ?, %s)`, policiesChecksumComputedColumn(),
	)

	jan2020 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	feb2020 := time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC)
	mar2020 := time.Date(2020, 3, 1, 0, 0, 0, 0, time.UTC)
	apr2020 := time.Date(2020, 4, 1, 0, 0, 0, 0, time.UTC)
	may2020 := time.Date(2020, 5, 1, 0, 0, 0, 0, time.UTC)
	pols := make([]*fleet.Policy, 3)
	for i, dt := range []time.Time{jan2020, feb2020, mar2020} {
		res, err := ds.writer(ctx).ExecContext(ctx, createPolStmt, "p"+strconv.Itoa(i+1), "select 1", user.ID, "", dt, dt)
		require.NoError(t, err)
		id, _ := res.LastInsertId()
		pol, err := ds.Policy(ctx, uint(id))
		require.NoError(t, err)
		pols[i] = pol
	}
	// index the policies by name for easier access in the rest of the test
	polsByName := make(map[string]*fleet.Policy, len(pols))
	for _, pol := range pols {
		polsByName[pol.Name] = pol
	}

	wantHostsByPol := map[string][]uint{
		"p1": {},
		"p2": {},
		"p3": {},
	}
	// no recently updated policies
	err := ds.CleanupPolicyMembership(ctx, time.Now())
	require.NoError(t, err)
	assertPolicyMembership(t, ds, polsByName, wantHostsByPol)

	// record results for each policy, all hosts, even if invalid for the policy
	for _, h := range hosts {
		res := map[uint]*bool{
			polsByName["p1"].ID: ptr.Bool(true),
			polsByName["p2"].ID: ptr.Bool(true),
			polsByName["p3"].ID: ptr.Bool(true),
		}
		err = ds.RecordPolicyQueryExecutions(ctx, h, res, time.Now(), false)
		require.NoError(t, err)
	}

	// no recently updated policies, so no host gets cleaned up
	wantHostsByPol = map[string][]uint{
		"p1": {hosts[hostWin].ID, hosts[hostMac].ID, hosts[hostDeb].ID, hosts[hostLin].ID},
		"p2": {hosts[hostWin].ID, hosts[hostMac].ID, hosts[hostDeb].ID, hosts[hostLin].ID},
		"p3": {hosts[hostWin].ID, hosts[hostMac].ID, hosts[hostDeb].ID, hosts[hostLin].ID},
	}
	err = ds.CleanupPolicyMembership(ctx, time.Now())
	require.NoError(t, err)
	assertPolicyMembership(t, ds, polsByName, wantHostsByPol)

	// update policy p1, but do not change the platform (still any)
	pols[0].Description = "updated"
	updatePolicyWithTimestamp(t, ds, pols[0], feb2020)
	err = ds.CleanupPolicyMembership(ctx, time.Now())
	require.NoError(t, err)
	assertPolicyMembership(t, ds, polsByName, wantHostsByPol)

	// update policy p1 to "windows", but cleanup with a timestamp of apr2020, so
	// not "recently updated", no changes
	pols[0].Platform = "windows"
	updatePolicyWithTimestamp(t, ds, pols[0], mar2020)
	err = ds.CleanupPolicyMembership(ctx, apr2020)
	require.NoError(t, err)
	assertPolicyMembership(t, ds, polsByName, wantHostsByPol)

	// now cleanup with a timestamp of mar2020+1h, so "recently updated", only windows
	// hosts are kept
	err = ds.CleanupPolicyMembership(ctx, mar2020.Add(time.Hour))
	require.NoError(t, err)
	wantHostsByPol["p1"] = []uint{hosts[hostWin].ID}
	assertPolicyMembership(t, ds, polsByName, wantHostsByPol)

	// update policy p2 to "linux,darwin", but cleanup with a timestamp of just over 24h, so
	// not "recently updated", no changes
	pols[1].Platform = "linux,darwin"
	updatePolicyWithTimestamp(t, ds, pols[1], mar2020)
	err = ds.CleanupPolicyMembership(ctx, mar2020.Add(25*time.Hour))
	require.NoError(t, err)
	assertPolicyMembership(t, ds, polsByName, wantHostsByPol)

	// now cleanup with a timestamp of just under 24h, so it is "recently updated"
	err = ds.CleanupPolicyMembership(ctx, mar2020.Add(23*time.Hour))
	require.NoError(t, err)
	wantHostsByPol["p2"] = []uint{hosts[hostMac].ID, hosts[hostDeb].ID, hosts[hostLin].ID}
	assertPolicyMembership(t, ds, polsByName, wantHostsByPol)

	// update policy p2 to just "linux", p3 to "debian", both get cleaned up (using apr2020
	// because p3 was created with mar2020, so it will not be detected as updated if we use
	// that same timestamp for the update).
	pols[1].Platform = "linux"
	updatePolicyWithTimestamp(t, ds, pols[1], apr2020)
	pols[2].Platform = "debian"
	updatePolicyWithTimestamp(t, ds, pols[2], apr2020)
	err = ds.CleanupPolicyMembership(ctx, apr2020.Add(time.Hour))
	require.NoError(t, err)
	wantHostsByPol["p2"] = []uint{hosts[hostDeb].ID, hosts[hostLin].ID}
	wantHostsByPol["p3"] = []uint{hosts[hostDeb].ID}
	assertPolicyMembership(t, ds, polsByName, wantHostsByPol)

	// cleaning up again 1h later doesn't change anything
	err = ds.CleanupPolicyMembership(ctx, apr2020.Add(2*time.Hour))
	require.NoError(t, err)
	assertPolicyMembership(t, ds, polsByName, wantHostsByPol)

	// update policy p1 to allow any, doesn't clean up anything
	pols[0].Platform = ""
	updatePolicyWithTimestamp(t, ds, pols[0], may2020)
	err = ds.CleanupPolicyMembership(ctx, may2020.Add(time.Hour))
	require.NoError(t, err)
	assertPolicyMembership(t, ds, polsByName, wantHostsByPol)
}

func updatePolicyWithTimestamp(t *testing.T, ds *Datastore, p *fleet.Policy, ts time.Time) {
	sqlStmt := `
		UPDATE policies
			SET name = ?, query = ?, description = ?, resolution = ?, platforms = ?, updated_at = ?
			WHERE id = ?`
	_, err := ds.writer(context.Background()).ExecContext(
		context.Background(), sqlStmt, p.Name, p.Query, p.Description, p.Resolution, p.Platform, ts, p.ID,
	)
	require.NoError(t, err)
}

func testDeleteAllPolicyMemberships(t *testing.T, ds *Datastore) {
	user1 := test.NewUser(t, ds, "Alan", "alan@example.com", true)
	ctx := context.Background()
	globalPolicy, err := ds.NewGlobalPolicy(ctx, &user1.ID, fleet.PolicyPayload{
		Name:        "query1",
		Query:       "select 1;",
		Description: "query1 desc",
		Resolution:  "query1 resolution",
	})
	require.NoError(t, err)

	host, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   ptr.String("567898"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("4"),
		UUID:            "4",
		Hostname:        "bar.local",
	})
	require.NoError(t, err)

	err = ds.RecordPolicyQueryExecutions(
		ctx,
		host,
		map[uint]*bool{globalPolicy.ID: ptr.Bool(false)},
		time.Now(),
		false,
	)
	require.NoError(t, err)

	hostPolicies, err := ds.ListPoliciesForHost(ctx, host)
	require.NoError(t, err)
	require.Len(t, hostPolicies, 1)

	var count int
	err = ds.writer(ctx).Get(&count, "select COUNT(*) from policy_membership")
	require.NoError(t, err)
	require.Equal(t, 1, count)

	err = deleteAllPolicyMemberships(ctx, ds.writer(ctx), []uint{host.ID})
	require.NoError(t, err)

	err = ds.writer(ctx).Get(&count, "select COUNT(*) from policy_membership")
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func testIncreasePolicyAutomationIteration(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	pol1, err := ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: "policy1"})
	require.NoError(t, err)
	pol2, err := ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: "policy2"})
	require.NoError(t, err)
	require.NoError(t, ds.IncreasePolicyAutomationIteration(ctx, pol1.ID))
	require.NoError(t, ds.IncreasePolicyAutomationIteration(ctx, pol2.ID))
	require.NoError(t, ds.IncreasePolicyAutomationIteration(ctx, pol2.ID))
	require.NoError(t, ds.IncreasePolicyAutomationIteration(ctx, pol2.ID))
	type at []struct {
		PolicyID  uint `db:"policy_id"`
		Iteration int  `db:"iteration"`
	}
	var automations at
	err = ds.writer(ctx).Select(&automations, `SELECT policy_id, iteration FROM policy_automation_iterations;`)
	require.NoError(t, err)
	require.ElementsMatch(t, automations, at{
		{pol1.ID, 1},
		{pol2.ID, 3},
	})
}

func testOutdatedAutomationBatch(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	h1, err := ds.NewHost(ctx, &fleet.Host{OsqueryHostID: ptr.String("host1"), NodeKey: ptr.String("host1")})
	require.NoError(t, err)
	h2, err := ds.NewHost(ctx, &fleet.Host{OsqueryHostID: ptr.String("host2"), NodeKey: ptr.String("host2")})
	require.NoError(t, err)

	pol1, err := ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: "policy1"})
	require.NoError(t, err)
	pol2, err := ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: "policy2"})
	require.NoError(t, err)

	err = ds.RecordPolicyQueryExecutions(ctx, h1, map[uint]*bool{pol1.ID: ptr.Bool(false), pol2.ID: ptr.Bool(true)}, time.Now(), false)
	require.NoError(t, err)
	err = ds.RecordPolicyQueryExecutions(ctx, h2, map[uint]*bool{pol1.ID: ptr.Bool(false), pol2.ID: ptr.Bool(false)}, time.Now(), false)
	require.NoError(t, err)

	batch, err := ds.OutdatedAutomationBatch(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, batch, []fleet.PolicyFailure{})

	require.NoError(t, ds.IncreasePolicyAutomationIteration(ctx, pol1.ID))
	batch, err = ds.OutdatedAutomationBatch(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, batch, []fleet.PolicyFailure{
		{
			PolicyID: pol1.ID,
			Host: fleet.PolicySetHost{
				ID: h1.ID,
			},
		},
		{
			PolicyID: pol1.ID,
			Host: fleet.PolicySetHost{
				ID: h2.ID,
			},
		},
	})

	batch, err = ds.OutdatedAutomationBatch(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, batch, []fleet.PolicyFailure{})

	require.NoError(t, ds.IncreasePolicyAutomationIteration(ctx, pol1.ID))
	require.NoError(t, ds.IncreasePolicyAutomationIteration(ctx, pol2.ID))
	batch, err = ds.OutdatedAutomationBatch(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, batch, []fleet.PolicyFailure{
		{
			PolicyID: pol1.ID,
			Host: fleet.PolicySetHost{
				ID: h1.ID,
			},
		}, {
			PolicyID: pol1.ID,
			Host: fleet.PolicySetHost{
				ID: h2.ID,
			},
		}, {
			PolicyID: pol2.ID,
			Host: fleet.PolicySetHost{
				ID: h2.ID,
			},
		},
	})

	batch, err = ds.OutdatedAutomationBatch(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, batch, []fleet.PolicyFailure{})
}

func testUpdatePolicyFailureCountsForHosts(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// create 4 hosts
	var hosts []*fleet.Host
	for i := 0; i < 4; i++ {
		h, err := ds.NewHost(ctx, &fleet.Host{OsqueryHostID: ptr.String(fmt.Sprintf("host%d", i)), NodeKey: ptr.String(fmt.Sprintf("host%d", i))})
		require.NoError(t, err)
		hosts = append(hosts, h)
	}

	// create 2 policies
	var pols []*fleet.Policy
	for i := 0; i < 2; i++ {
		p, err := ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: fmt.Sprintf("policy%d", i)})
		require.NoError(t, err)
		pols = append(pols, p)
	}

	// create policy membership for hosts
	_, err := ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO policy_membership (policy_id, host_id, passes)
		VALUES
			(?, ?, 1),
			(?, ?, 1),
			(?, ?, 0),
			(?, ?, 0),
			(?, ?, 1),
			(?, ?, 0)
	`,
		pols[0].ID, hosts[0].ID,
		pols[0].ID, hosts[1].ID,
		pols[0].ID, hosts[2].ID,
		pols[1].ID, hosts[0].ID,
		pols[1].ID, hosts[1].ID,
		pols[1].ID, hosts[2].ID,
	)

	require.NoError(t, err)

	// update policy failure counts for hosts
	hostsUpdated, err := ds.UpdatePolicyFailureCountsForHosts(ctx, hosts)
	require.NoError(t, err)
	require.Len(t, hostsUpdated, 4)

	// host 0 should have 1 failing policy
	assert.Equal(t, 1, hostsUpdated[0].TotalIssuesCount)
	assert.Equal(t, 1, hostsUpdated[0].FailingPoliciesCount)

	// host 1 should have 0 failing policies
	assert.Equal(t, 0, hostsUpdated[1].TotalIssuesCount)
	assert.Equal(t, 0, hostsUpdated[1].FailingPoliciesCount)

	// host 2 should have 2 failing policies
	assert.Equal(t, 2, hostsUpdated[2].TotalIssuesCount)
	assert.Equal(t, 2, hostsUpdated[2].FailingPoliciesCount)

	// host 3 doesn't have any policy membership
	assert.Equal(t, 0, hostsUpdated[3].TotalIssuesCount)
	assert.Equal(t, 0, hostsUpdated[3].FailingPoliciesCount)

	// return empty list if no hosts are passed
	hostsUpdated, err = ds.UpdatePolicyFailureCountsForHosts(ctx, []*fleet.Host{})
	require.NoError(t, err)
	require.Len(t, hostsUpdated, 0)
}

func testListGlobalPoliciesCanPaginate(t *testing.T, ds *Datastore) {
	// create 30 policies
	for i := 0; i < 30; i++ {
		_, err := ds.NewGlobalPolicy(context.Background(), nil, fleet.PolicyPayload{Name: fmt.Sprintf("global policy %d", i)})
		require.NoError(t, err)
	}

	// create 30 team policies
	tm, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	for i := 0; i < 30; i++ {
		_, err := ds.NewTeamPolicy(context.Background(), tm.ID, nil, fleet.PolicyPayload{Name: fmt.Sprintf("team policy %d", i)})
		require.NoError(t, err)
	}

	// Page 0 contains 20 policies
	policies, err := ds.ListGlobalPolicies(context.Background(), fleet.ListOptions{
		Page:    0,
		PerPage: 20,
	})

	assert.Equal(t, "global policy 0", policies[0].Name)
	assert.Len(t, policies, 20)
	require.NoError(t, err)

	// Page 1 contains 10 policies
	policies, err = ds.ListGlobalPolicies(context.Background(), fleet.ListOptions{
		Page:    1,
		PerPage: 20,
	})

	assert.Equal(t, "global policy 20", policies[0].Name)
	assert.Len(t, policies, 10)
	require.NoError(t, err)

	// No list options returns all policies
	policies, err = ds.ListGlobalPolicies(context.Background(), fleet.ListOptions{})
	assert.Len(t, policies, 30)
	require.NoError(t, err)
}

func testListTeamPoliciesCanPaginate(t *testing.T, ds *Datastore) {
	tm, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	// create 30 team policies
	for i := 0; i < 30; i++ {
		_, err := ds.NewTeamPolicy(context.Background(), tm.ID, nil, fleet.PolicyPayload{Name: fmt.Sprintf("team policy %d", i)})
		require.NoError(t, err)
	}

	// create 30 global policies
	for i := 0; i < 30; i++ {
		_, err := ds.NewGlobalPolicy(context.Background(), nil, fleet.PolicyPayload{Name: fmt.Sprintf("global policy %d", i)})
		require.NoError(t, err)
	}

	// Page 0 contains 20 policies
	policies, _, err := ds.ListTeamPolicies(context.Background(), tm.ID, fleet.ListOptions{
		Page:    0,
		PerPage: 20,
	}, fleet.ListOptions{})

	assert.Equal(t, "team policy 0", policies[0].Name)
	assert.Len(t, policies, 20)
	require.NoError(t, err)

	// Page 1 contains 10 policies
	policies, _, err = ds.ListTeamPolicies(context.Background(), tm.ID, fleet.ListOptions{
		Page:    1,
		PerPage: 20,
	}, fleet.ListOptions{})

	assert.Equal(t, "team policy 20", policies[0].Name)
	assert.Len(t, policies, 10)
	require.NoError(t, err)

	// No list options returns all policies
	policies, _, err = ds.ListTeamPolicies(context.Background(), 1, fleet.ListOptions{}, fleet.ListOptions{})
	assert.Len(t, policies, 30)
	require.NoError(t, err)
}

func testCountPolicies(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	// no policies
	globalCount, err := ds.CountPolicies(ctx, nil, "")
	require.NoError(t, err)
	assert.Equal(t, 0, globalCount)

	teamCount, err := ds.CountPolicies(ctx, &tm.ID, "")
	require.NoError(t, err)
	assert.Equal(t, 0, teamCount)

	// 10 global policies
	for i := 0; i < 10; i++ {
		_, err := ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: fmt.Sprintf("global policy %d", i)})
		require.NoError(t, err)
	}

	globalCount, err = ds.CountPolicies(ctx, nil, "")
	require.NoError(t, err)
	assert.Equal(t, 10, globalCount)

	teamCount, err = ds.CountPolicies(ctx, &tm.ID, "")
	require.NoError(t, err)
	assert.Equal(t, 0, teamCount)

	// add 5 team policies
	for i := 0; i < 5; i++ {
		_, err := ds.NewTeamPolicy(ctx, tm.ID, nil, fleet.PolicyPayload{Name: fmt.Sprintf("team policy %d", i)})
		require.NoError(t, err)
	}

	teamCount, err = ds.CountPolicies(ctx, &tm.ID, "")
	require.NoError(t, err)
	assert.Equal(t, 5, teamCount)

	globalCount, err = ds.CountPolicies(ctx, nil, "")
	require.NoError(t, err)
	assert.Equal(t, 10, globalCount)
}

func testUpdatePolicyHostCounts(t *testing.T, ds *Datastore) {
	// new policy
	policy, err := ds.NewGlobalPolicy(context.Background(), nil, fleet.PolicyPayload{Name: "global policy 1"})
	require.NoError(t, err)

	team, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	// create 4 team hosts
	var teamHosts []*fleet.Host
	for i := 0; i < 4; i++ {
		h, err := ds.NewHost(context.Background(), &fleet.Host{OsqueryHostID: ptr.String(fmt.Sprintf("host%d", i)), NodeKey: ptr.String(fmt.Sprintf("host%d", i)), TeamID: &team.ID})
		require.NoError(t, err)
		teamHosts = append(teamHosts, h)
	}

	// create 4 global hosts
	var globalHosts []*fleet.Host
	for i := 4; i < 8; i++ {
		h, err := ds.NewHost(context.Background(), &fleet.Host{OsqueryHostID: ptr.String(fmt.Sprintf("host%d", i)), NodeKey: ptr.String(fmt.Sprintf("host%d", i)), TeamID: nil})
		require.NoError(t, err)
		globalHosts = append(globalHosts, h)
	}

	// add policy responses
	for _, h := range teamHosts {
		res := map[uint]*bool{
			policy.ID: ptr.Bool(true),
		}
		err = ds.RecordPolicyQueryExecutions(context.Background(), h, res, time.Now(), false)
		require.NoError(t, err)
	}

	for _, h := range globalHosts {
		res := map[uint]*bool{
			policy.ID: ptr.Bool(true),
		}
		err = ds.RecordPolicyQueryExecutions(context.Background(), h, res, time.Now(), false)
		require.NoError(t, err)
	}

	// check policy host counts before update
	policy, err = ds.Policy(context.Background(), policy.ID)
	require.NoError(t, err)
	require.Equal(t, uint(0), policy.FailingHostCount)
	require.Equal(t, uint(0), policy.PassingHostCount)
	assert.Nil(t, policy.HostCountUpdatedAt)

	// update policy host counts
	now := time.Now().Truncate(time.Second)
	later := now.Add(10 * time.Second)
	err = ds.UpdateHostPolicyCounts(context.Background())
	require.NoError(t, err)

	// check policy host counts
	policy, err = ds.Policy(context.Background(), policy.ID)
	require.NoError(t, err)
	require.Equal(t, uint(0), policy.FailingHostCount)
	require.Equal(t, uint(8), policy.PassingHostCount)
	require.NotNil(t, policy.HostCountUpdatedAt)
	assert.True(
		t, policy.HostCountUpdatedAt.Compare(now) >= 0, fmt.Sprintf("reference:%v HostCountUpdatedAt:%v", now, *policy.HostCountUpdatedAt),
	)
	assert.True(
		t, policy.HostCountUpdatedAt.Compare(later) < 0, fmt.Sprintf("later:%v HostCountUpdatedAt:%v", later, *policy.HostCountUpdatedAt),
	)
}
