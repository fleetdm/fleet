package android

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// MySQLTables are the tables that are present in Android's schema.sql
// This is an optimization/encapsulation exercise -- Android Datastore is only unit tested with the tables it uses.
func MySQLTables() []string {
	return []string{
		"android_enterprises",
		"android_devices",
		"hosts",
		"host_display_names",
		"host_mdm",
		"users", // needed as FK for labels
		"labels",
		"label_membership",
		"mobile_device_management_solutions",
	}
}

type Datastore interface {
	CreateEnterprise(ctx context.Context, userID uint) (uint, error)
	GetEnterpriseByID(ctx context.Context, ID uint) (*EnterpriseDetails, error)
	GetEnterpriseBySignupToken(ctx context.Context, signupToken string) (*EnterpriseDetails, error)
	GetEnterprise(ctx context.Context) (*Enterprise, error)
	UpdateEnterprise(ctx context.Context, enterprise *EnterpriseDetails) error
	DeleteAllEnterprises(ctx context.Context) error
	DeleteOtherEnterprises(ctx context.Context, ID uint) error

	NewAndroidHost(ctx context.Context, serverURL string, host *Host) (*Host, error)
	AndroidHostLite(ctx context.Context, enterpriseSpecificID string) (*Host, error)
	UpdateAndroidHost(ctx context.Context, serverURL string, host *Host, fromEnroll bool) error
	BulkSetAndroidHostsUnenrolled(ctx context.Context) error

	CreateDeviceTx(ctx context.Context, tx sqlx.ExtContext, device *Device) (*Device, error)
	UpdateDeviceTx(ctx context.Context, tx sqlx.ExtContext, device *Device) error
}
