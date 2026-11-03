package mdm

import (
	"context"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

type fakeIdPAccounts map[string]string

func (f fakeIdPAccounts) GetMDMIdPAccountByUUID(_ context.Context, uuid string) (*fleet.MDMIdPAccount, error) {
	email, ok := f[uuid]
	if !ok {
		return nil, &notFoundError{}
	}
	return &fleet.MDMIdPAccount{UUID: uuid, Email: email}, nil
}

type notFoundError struct{}

func (e *notFoundError) Error() string    { return "not found" }
func (e *notFoundError) IsNotFound() bool { return true }

func TestLogHostIdPAccountLinkChange(t *testing.T) {
	accounts := fakeIdPAccounts{"a1": "alice@example.com", "b1": "bob@example.com"}

	cases := []struct {
		name              string
		previous, current string
		want              fleet.ActivityDetails
	}{
		{"unchanged", "a1", "a1", nil},
		{"never linked", "", "", nil},
		{"first link", "", "a1", fleet.ActivityTypeBoundHostToIdPAccount{HostUUID: "h1", IdPEmail: "alice@example.com"}},
		{"replaced", "a1", "b1", fleet.ActivityTypeBoundHostToIdPAccount{HostUUID: "h1", IdPEmail: "bob@example.com", ReplacedIdPEmail: "alice@example.com"}},
		{"removed", "a1", "", fleet.ActivityTypeUnboundHostFromIdPAccount{HostUUID: "h1", IdPEmail: "alice@example.com"}},
		{"unknown account", "a1", "missing", nil},
		{"removed unknown account", "missing", "", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got []fleet.ActivityDetails
			newActivity := func(_ context.Context, user *fleet.User, act fleet.ActivityDetails) error {
				require.Nil(t, user)
				got = append(got, act)
				return nil
			}
			LogHostIdPAccountLinkChange(t.Context(), accounts, newActivity, slog.New(slog.DiscardHandler), "h1", c.previous, c.current)
			if c.want == nil {
				require.Empty(t, got)
				return
			}
			require.Equal(t, []fleet.ActivityDetails{c.want}, got)
		})
	}
}
