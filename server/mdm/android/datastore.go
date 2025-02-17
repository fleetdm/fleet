package android

import (
	"context"
)

type Datastore interface {
	CreateEnterprise(ctx context.Context) (uint, error)
	GetEnterpriseByID(ctx context.Context, ID uint) (*EnterpriseDetails, error)
	GetEnterprise(ctx context.Context) (*Enterprise, error)
	UpdateEnterprise(ctx context.Context, enterprise *EnterpriseDetails) error
	DeleteEnterprises(ctx context.Context) error
	DeleteOtherEnterprises(ctx context.Context, ID uint) error
}
