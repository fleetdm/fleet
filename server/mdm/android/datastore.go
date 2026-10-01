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
		"android_zero_touch_tokens",
	}
}

type Datastore interface {
	CreateEnterprise(ctx context.Context, userID uint) (uint, error)
	GetEnterpriseByID(ctx context.Context, id uint) (*EnterpriseDetails, error)
	GetEnterpriseBySignupToken(ctx context.Context, signupToken string) (*EnterpriseDetails, error)
	GetEnterprise(ctx context.Context) (*Enterprise, error)
	UpdateEnterprise(ctx context.Context, enterprise *EnterpriseDetails) error
	DeleteAllEnterprises(ctx context.Context) error
	DeleteOtherEnterprises(ctx context.Context, id uint) error

	CreateDeviceTx(ctx context.Context, tx sqlx.ExtContext, device *Device) (*Device, error)
	UpdateDeviceTx(ctx context.Context, tx sqlx.ExtContext, device *Device) error

	// GetZeroTouchEnrollmentToken returns the zero-touch enrollment token for the given team.
	// Pass nil for the "Unassigned" (no team) token. Returns a not-found error if none exists.
	GetZeroTouchEnrollmentToken(ctx context.Context, teamID *uint) (*ZeroTouchToken, error)
	// CreateZeroTouchEnrollmentToken inserts a new zero-touch enrollment token.
	CreateZeroTouchEnrollmentToken(ctx context.Context, token *ZeroTouchToken) (*ZeroTouchToken, error)
	// DeleteZeroTouchEnrollmentTokens deletes all zero-touch enrollment tokens.
	DeleteZeroTouchEnrollmentTokens(ctx context.Context) error
}
