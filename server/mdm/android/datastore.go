package android

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type Datastore interface {
	CreateEnterprise(ctx context.Context) (uint, error)
	GetEnterpriseByID(ctx context.Context, ID uint) (*EnterpriseDetails, error)
	GetEnterprise(ctx context.Context) (*Enterprise, error)
	UpdateEnterprise(ctx context.Context, enterprise *EnterpriseDetails) error
	DeleteEnterprises(ctx context.Context) error
	DeleteOtherEnterprises(ctx context.Context, ID uint) error

	CreateDeviceTx(ctx context.Context, device *Device, tx sqlx.ExtContext) (*Device, error)
	GetDeviceByDeviceID(ctx context.Context, deviceID string) (*Device, error)
}
