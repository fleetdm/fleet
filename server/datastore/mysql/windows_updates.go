package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (ds *Datastore) InsertWindowsUpdates(ctx context.Context, hostID uint, KBIDs map[uint]fleet.WindowsUpdate) error {
	panic("not implemented")
}
