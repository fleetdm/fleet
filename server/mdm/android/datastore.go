package android

import (
	"context"
)

type Datastore interface {
	CreateEnterprise(ctx context.Context) (uint, error)
	GetEnterpriseByID(ctx context.Context, ID uint) (*Enterprise, error)
	GetEnterprise(ctx context.Context) (*Enterprise, error)
	UpdateEnterprise(ctx context.Context, enterprise *Enterprise) error
	DeleteEnterprises(ctx context.Context) error
	DeleteOtherEnterprises(ctx context.Context, ID uint) error
}
