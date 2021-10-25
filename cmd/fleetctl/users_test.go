package main

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
)

func TestUserDelete(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
		return &fleet.User{
			ID:    42,
			Name:  "test1",
			Email: "user1@test.com",
		}, nil
	}

	deletedUser := uint(0)

	ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
		deletedUser = id
		return nil
	}

	assert.Equal(t, "", runAppForTest(t, []string{"user", "delete", "--email", "user1@test.com"}))
	assert.Equal(t, uint(42), deletedUser)
}
