package allmulti

import (
	"context"

	"github.com/micromdm/nanolib/log/ctxlog"
)

func (ms *MultiAllStorage) RetrieveMigrationCheckins(ctx context.Context, c chan<- interface{}) error {
	ctxlog.Logger(ctx, ms.logger).Info(
		"msg", "only using first store for migration",
	)
	return ms.stores[0].RetrieveMigrationCheckins(ctx, c)
}
