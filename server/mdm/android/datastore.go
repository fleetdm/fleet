package android

import (
	"context"
)

type Datastore interface {
	CreateEnterprise(ctx context.Context) (uint, error)
	GetEnterpriseByID(ctx context.Context, ID uint) (*Enterprise, error)
	UpdateEnterprise(ctx context.Context, enterprise *Enterprise) error
	ListEnterprises(ctx context.Context) ([]*Enterprise, error)
}
