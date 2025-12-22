package mock

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	svcmock "github.com/fleetdm/fleet/v4/server/mock/service"
)

//go:generate go run ./mockimpl/impl.go -o service/service_mock.go "s *Service" "fleet.Service"

var _ fleet.Service = new(svcmock.Service)
