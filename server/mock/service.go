package mock

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	kvmock "github.com/fleetdm/fleet/v4/server/mock/redis"
	svcmock "github.com/fleetdm/fleet/v4/server/mock/service"
)

//go:generate go run ./mockimpl/impl.go -o service/service_mock.go "s *Service" "fleet.Service"
//go:generate go run ./mockimpl/impl.go -o redis/key_value_store.go "kv *KeyValueStore" "fleet.KeyValueStore"

var _ fleet.Service = new(svcmock.Service)

type KVStore struct {
	kvmock.KeyValueStore
}
