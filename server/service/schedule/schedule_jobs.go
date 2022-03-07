package schedule

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
)

func DoAsyncTestJob(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, config config.FleetConfig, instanceID string) (interface{}, error) {
	rand.Seed(time.Now().UnixNano())
	id := rand.Intn(40)
	host, err := ds.Host(ctx, uint(id), true)
	if err != nil {
		return nil, err
	}
	// if id == 20 {
	// 	return nil, errors.New("this is just a test error!")
	// }
	stats := make(map[string]string)
	stats[fmt.Sprint("host_", host.ID)] = host.Hostname

	return stats, nil
}
