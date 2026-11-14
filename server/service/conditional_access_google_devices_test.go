package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_google"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const googleCADeviceTestCustomer = "C02cajzvg"

// googleCADeviceTest is a connected integration with a company-owned iPhone. Its
// datastore holds the host's Google registration and, like MySQL, fails writes on a
// canceled context. onStored runs once a registration is stored.
type googleCADeviceTest struct {
	*googleCADisconnectTest
	host     *fleet.Host
	deviceID string
	states   map[string]fleet.GoogleConditionalAccessDeviceUserState
	onStored func()
}

func newGoogleCADeviceTest(t *testing.T) *googleCADeviceTest {
	g := &googleCADeviceTest{
		googleCADisconnectTest: newGoogleCADisconnectTest(t, googleCADeviceTestCustomer),
		host:                   &fleet.Host{ID: 1, UUID: "host-uuid", Hostname: "iphone", Platform: "ios"},
		states:                 map[string]fleet.GoogleConditionalAccessDeviceUserState{},
	}
	store := func(ctx context.Context, deviceID string, names []string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		g.deviceID = deviceID
		for _, name := range names {
			g.states[name] = fleet.GoogleConditionalAccessDeviceUserPending
		}
		if g.onStored != nil {
			g.onStored()
		}
		return nil
	}

	g.ds.HostLiteFunc = func(context.Context, uint) (*fleet.Host, error) { return g.host, nil }
	g.ds.HostByUUIDFunc = func(context.Context, string) (*fleet.Host, error) { return g.host, nil }
	g.ds.GetHostMDMFunc = func(context.Context, uint) (*fleet.HostMDM, error) {
		return &fleet.HostMDM{Enrolled: true, InstalledFromDep: true}, nil
	}
	g.ds.ConsumeGoogleConditionalAccessRegistrationTokenFunc = func(ctx context.Context, _ string, _ []byte) (string, error) {
		return g.host.UUID, ctx.Err()
	}
	g.ds.RecordGoogleConditionalAccessRegistrationFunc = func(ctx context.Context, _, _ string, names []string) error {
		return store(ctx, fleet.GoogleDeviceIDFromDeviceUserName(names[0]), names)
	}
	g.ds.SetHostGoogleDeviceIDFunc = func(ctx context.Context, _, _, googleDeviceID string, names []string) (bool, error) {
		if err := store(ctx, googleDeviceID, names); err != nil {
			return false, err
		}
		return true, nil
	}
	g.ds.GetHostGoogleDeviceIDFunc = func(context.Context, string, string) (string, error) { return g.deviceID, nil }
	g.ds.UpdateGoogleConditionalAccessDeviceUserStateFunc = func(
		ctx context.Context, name string, _, toState fleet.GoogleConditionalAccessDeviceUserState, _ *string,
	) (bool, error) {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		g.states[name] = toState
		return true, nil
	}
	return g
}

func TestGoogleConditionalAccessDeviceRegistration(t *testing.T) {
	const name = "devices/device-a/deviceUsers/1"
	setActivity := []string{fleet.ActivityTypeSetHostGoogleDeviceID{}.ActivityName()}
	pending := map[string]fleet.GoogleConditionalAccessDeviceUserState{name: fleet.GoogleConditionalAccessDeviceUserPending}
	managed := map[string]fleet.GoogleConditionalAccessDeviceUserState{name: fleet.GoogleConditionalAccessDeviceUserManaged}
	signIn := func(t *testing.T, g *googleCADeviceTest) fleet.GoogleConditionalAccessRegisterResult {
		result, err := g.svc.RegisterGoogleConditionalAccessDevice(g.ctx, "secret-"+googleCADeviceTestCustomer,
			fleet.GoogleConditionalAccessRegistration{
				Token: "token", Names: []string{name}, Customer: googleCADeviceTestCustomer, Email: "user@example.com",
			})
		require.NoError(t, err)
		return result
	}

	t.Run("a sign-in is logged and marked when fleetdm.com disconnects", func(t *testing.T) {
		g := newGoogleCADeviceTest(t)
		g.onStored = g.relay.disconnect
		require.Equal(t, fleet.GoogleConditionalAccessRegisterDone, signIn(t, g))
		require.Equal(t, setActivity, g.activities)
		require.Equal(t, managed, g.states)
	})

	t.Run("a sign-in that can't be logged is still done and marked", func(t *testing.T) {
		g := newGoogleCADeviceTest(t)
		g.activityErr = errors.New("activity failed")
		require.Equal(t, fleet.GoogleConditionalAccessRegisterDone, signIn(t, g))
		require.Equal(t, managed, g.states)
	})

	t.Run("a sign-in is done and logged when the key can't be read", func(t *testing.T) {
		g := newGoogleCADeviceTest(t)
		g.ds.GetAllMDMConfigAssetsByNameFunc = func(context.Context, []fleet.MDMAssetName, sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return nil, errors.New("decrypt failed")
		}
		require.Equal(t, fleet.GoogleConditionalAccessRegisterDone, signIn(t, g))
		require.Equal(t, setActivity, g.activities)
		require.Equal(t, pending, g.states)
	})

	t.Run("setting the ID is logged and marked when the admin disconnects", func(t *testing.T) {
		g := newGoogleCADeviceTest(t)
		g.relay.setDev(func(d *googleDeviceMock) { d.deviceUsers = map[string][]string{"device-a": {name}} })
		g.onStored = g.relay.disconnect
		require.NoError(t, g.svc.SetHostGoogleDeviceID(g.ctx, g.host.ID, "device-a"))
		require.Equal(t, setActivity, g.activities)
		require.Equal(t, managed, g.states)
	})

	// The datastore here binds whatever it is given, as MySQL does when the stored ID is
	// deleted between the service's read and its write.
	t.Run("a different ID is refused before anything is bound", func(t *testing.T) {
		g := newGoogleCADeviceTest(t)
		g.deviceID = "device-a"
		g.relay.setDev(func(d *googleDeviceMock) {
			d.deviceUsers = map[string][]string{"device-b": {"devices/device-b/deviceUsers/1"}}
		})
		err := g.svc.SetHostGoogleDeviceID(g.ctx, g.host.ID, "device-b")
		var conflict *fleet.HostGoogleDeviceIDConflictError
		require.ErrorAs(t, err, &conflict)
		require.Equal(t, "device-a", g.deviceID)
		require.Empty(t, g.states)
		require.Empty(t, g.activities)
	})
}

func TestMarkGoogleDeviceUserKeepsGoogleError(t *testing.T) {
	const name = "devices/device-a/deviceUsers/1"
	googleErr := errors.New("Google returned HTTP 500")
	recordErr := errors.New("connection lost")

	ds := new(mock.Store)
	ds.UpdateGoogleConditionalAccessDeviceUserStateFunc = func(
		context.Context, string, fleet.GoogleConditionalAccessDeviceUserState, fleet.GoogleConditionalAccessDeviceUserState, *string,
	) (bool, error) {
		return false, recordErr
	}
	client := &mockedGoogleConditionalAccessClient{}
	client.setDev(func(d *googleDeviceMock) { d.markErrs = map[string]error{name: googleErr} })
	ci, err := client.cloudIdentity(t.Context(), conditional_access_google.Credentials{})
	require.NoError(t, err)

	_, err = markGoogleDeviceUser(t.Context(), ds, ci, name, "host-uuid")
	require.ErrorIs(t, err, recordErr)
	require.ErrorIs(t, err, googleErr)
}

func TestSameGoogleCustomer(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"C02cajzvg", "C02cajzvg", true},
		{"customers/C02cajzvg", "C02cajzvg", true},
		{"customers/02cajzvg", "C02cajzvg", true},
		{"02cajzvg", "C02cajzvg", true},
		{" customers/02cajzvg\n", "C02cajzvg", true},
		{"C03other", "C02cajzvg", false},
		{"customers/03other", "C02cajzvg", false},
		{"", "C02cajzvg", false},
		{"customers/", "C02cajzvg", false},
		{"", "", false},
		{"customers/C", "C", false},
	} {
		require.Equal(t, c.want, sameGoogleCustomer(c.a, c.b), "%q and %q", c.a, c.b)
	}
}
