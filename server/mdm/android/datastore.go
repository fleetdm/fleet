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
	DeleteAllEnterprises(ctx context.Context) error
	DeleteOtherEnterprises(ctx context.Context, ID uint) error

	CreateDeviceTx(ctx context.Context, tx sqlx.ExtContext, device *Device) (*Device, error)
	UpdateDeviceTx(ctx context.Context, tx sqlx.ExtContext, device *Device) error
}
