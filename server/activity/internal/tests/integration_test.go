package tests

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	s := setupIntegrationTest(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, s *integrationTestSuite)
	}{
		{"ListActivities", testListActivities},
		{"ListActivitiesPagination", testListActivitiesPagination},
		{"ListActivitiesCursorPagination", testListActivitiesCursorPagination},
		{"ListActivitiesFilters", testListActivitiesFilters},
		{"ListActivitiesUserEnrichment", testListActivitiesUserEnrichment},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer s.truncateTables(t)
			c.fn(t, s)
		})
	}
}

func testListActivities(t *testing.T, s *integrationTestSuite) {
	userID := s.insertUser(t, "admin", "admin@example.com")

	// Insert activities
	s.InsertActivity(t, ptr.Uint(userID), "applied_spec_pack", map[string]any{})
	s.InsertActivity(t, ptr.Uint(userID), "deleted_pack", map[string]any{})
	s.InsertActivity(t, ptr.Uint(userID), "edited_pack", map[string]any{})

	result, statusCode := s.getActivities(t, "per_page=100")

	assert.Equal(t, http.StatusOK, statusCode)
	assert.Len(t, result.Activities, 3)
	assert.NotNil(t, result.Meta)

	// Verify order (newest first by default)
	assert.Equal(t, "edited_pack", result.Activities[0].Type)
	assert.Equal(t, "deleted_pack", result.Activities[1].Type)
	assert.Equal(t, "applied_spec_pack", result.Activities[2].Type)
}

func testListActivitiesPagination(t *testing.T, s *integrationTestSuite) {
	userID := s.insertUser(t, "admin", "admin@example.com")

	// Insert 5 activities
	for i := range 5 {
		s.InsertActivity(t, ptr.Uint(userID), "test_activity", map[string]any{"index": i})
	}

	// First page
	result, _ := s.getActivities(t, "per_page=2&order_key=id&order_direction=asc")
	assert.Len(t, result.Activities, 2)
	assert.True(t, result.Meta.HasNextResults)
	assert.False(t, result.Meta.HasPreviousResults)

	// Second page
	result, _ = s.getActivities(t, "per_page=2&page=1&order_key=id&order_direction=asc")
	assert.Len(t, result.Activities, 2)
	assert.True(t, result.Meta.HasNextResults)
	assert.True(t, result.Meta.HasPreviousResults)

	// Last page
	result, _ = s.getActivities(t, "per_page=2&page=2&order_key=id&order_direction=asc")
	assert.Len(t, result.Activities, 1)
	assert.False(t, result.Meta.HasNextResults)
	assert.True(t, result.Meta.HasPreviousResults)
}

func testListActivitiesCursorPagination(t *testing.T, s *integrationTestSuite) {
	userID := s.insertUser(t, "admin", "admin@example.com")

	// Insert 3 activities
	s.InsertActivity(t, ptr.Uint(userID), "applied_spec_pack", map[string]any{})
	s.InsertActivity(t, ptr.Uint(userID), "deleted_pack", map[string]any{})
	s.InsertActivity(t, ptr.Uint(userID), "edited_pack", map[string]any{})

	// Test cursor-based pagination with after=0
	// Meta should be nil for cursor-based pagination (doesn't return metadata)
	result, statusCode := s.getActivities(t, "per_page=1&order_key=id&after=0")
	assert.Equal(t, http.StatusOK, statusCode)
	assert.Len(t, result.Activities, 1)
	assert.Nil(t, result.Meta)
	assert.Equal(t, "applied_spec_pack", result.Activities[0].Type)

	// Test cursor pagination to get the next activity
	firstID := result.Activities[0].ID
	result, _ = s.getActivities(t, "per_page=1&order_key=id&after="+strconv.FormatUint(uint64(firstID), 10))
	assert.Len(t, result.Activities, 1)
	assert.Nil(t, result.Meta)
	assert.Equal(t, "deleted_pack", result.Activities[0].Type)

	// Test descending order with cursor
	result, _ = s.getActivities(t, "per_page=1&order_key=id&order_direction=desc&after=999999")
	assert.Len(t, result.Activities, 1)
	assert.Nil(t, result.Meta)
	// Descending order, so the newest (edited_pack) should be first
	assert.Equal(t, "edited_pack", result.Activities[0].Type)
}

func testListActivitiesFilters(t *testing.T, s *integrationTestSuite) {
	johnUserID := s.insertUser(t, "john_doe", "john@example.com")
	janeUserID := s.insertUser(t, "jane_smith", "jane@example.com")
	now := time.Now().UTC().Truncate(time.Second)

	// Insert activities with different types, times, and users
	s.InsertActivityWithTime(t, ptr.Uint(johnUserID), "type_a", map[string]any{}, now.Add(-48*time.Hour))
	s.InsertActivityWithTime(t, ptr.Uint(johnUserID), "type_a", map[string]any{}, now.Add(-24*time.Hour))
	s.InsertActivityWithTime(t, ptr.Uint(johnUserID), "type_b", map[string]any{}, now)
	s.InsertActivityWithTime(t, ptr.Uint(janeUserID), "type_a", map[string]any{}, now) // Jane's activity

	// Filter by type
	result, _ := s.getActivities(t, "per_page=100&activity_type=type_a")
	assert.Len(t, result.Activities, 3) // 2 from john + 1 from jane
	for _, a := range result.Activities {
		assert.Equal(t, "type_a", a.Type)
	}

	// Filter by date range
	startDate := now.Add(-36 * time.Hour).Format(time.RFC3339)
	result, _ = s.getActivities(t, "per_page=100&start_created_at="+startDate)
	assert.Len(t, result.Activities, 3) // -24h, now (john), now (jane)

	// Filter by user search query - should only return john's activities
	result, _ = s.getActivities(t, "per_page=100&query=john")
	assert.Len(t, result.Activities, 3) // Only john's 3 activities, not jane's
	for _, a := range result.Activities {
		require.NotNil(t, a.ActorID)
		assert.Equal(t, johnUserID, *a.ActorID)
	}

	// Filter by user search query - should only return jane's activities
	result, _ = s.getActivities(t, "per_page=100&query=jane")
	assert.Len(t, result.Activities, 1) // Only jane's 1 activity
	require.NotNil(t, result.Activities[0].ActorID)
	assert.Equal(t, janeUserID, *result.Activities[0].ActorID)
}

func testListActivitiesUserEnrichment(t *testing.T, s *integrationTestSuite) {
	userID := s.insertUser(t, "John Doe", "john@example.com")

	s.InsertActivity(t, ptr.Uint(userID), "test_activity", map[string]any{})

	result, _ := s.getActivities(t, "per_page=100")
	require.Len(t, result.Activities, 1)

	// Verify user enrichment from mock user provider
	a := result.Activities[0]
	assert.NotNil(t, a.ActorID)
	assert.Equal(t, userID, *a.ActorID)
	assert.NotNil(t, a.ActorFullName)
	assert.Equal(t, "John Doe", *a.ActorFullName)
	assert.NotNil(t, a.ActorEmail)
	assert.Equal(t, "john@example.com", *a.ActorEmail)
}
