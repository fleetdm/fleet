package allmulti

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/log/ctxlog"
)

func (ms *MultiAllStorage) RetrieveMigrationCheckins(ctx context.Context, c chan<- interface{}) error {
	ctxlog.Logger(ctx, ms.logger).Info(
		"msg", "only using first store for migration",
	)
	return ms.stores[0].RetrieveMigrationCheckins(ctx, c)
}
