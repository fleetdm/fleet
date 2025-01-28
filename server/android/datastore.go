package android

import (
	"context"
)

type Datastore interface {
	// MigrateTables creates and migrates the table schemas
	MigrateTables(ctx context.Context) error
	// MigrationStatus returns nil if migrations are complete, and an error if migrations need to be run.
	MigrationStatus(ctx context.Context) (*MigrationStatus, error)

	CreateEnterprise(ctx context.Context) (uint, error)
	GetEnterpriseByID(ctx context.Context, ID uint) (*Enterprise, error)
	UpdateEnterprise(ctx context.Context, enterprise *Enterprise) error
}

type MigrationStatus struct {
	// StatusCode holds the code for the migration status.
	//
	// If StatusCode is NoMigrationsCompleted or AllMigrationsCompleted
	// then all other fields are empty.
	//
	// If StatusCode is SomeMigrationsCompleted, then missing migrations
	// are available in MissingTable and MissingData.
	//
	// If StatusCode is UnknownMigrations, then unknown migrations
	// are available in UnknownTable and UnknownData.
	StatusCode MigrationStatusCode `json:"status_code"`
	// MissingTable holds the missing table migrations.
	MissingTable []int64 `json:"missing_table"`
	// UnknownTable holds unknown applied table migrations.
	UnknownTable []int64 `json:"unknown_table"`
}

type MigrationStatusCode int

const (
	// NoMigrationsCompleted indicates the database has no migrations installed.
	NoMigrationsCompleted MigrationStatusCode = iota
	// SomeMigrationsCompleted indicates some (not all) migrations are missing.
	SomeMigrationsCompleted
	// AllMigrationsCompleted means all migrations have been installed successfully.
	AllMigrationsCompleted
	// UnknownMigrations means some unidentified migrations were detected on the database.
	UnknownMigrations
)
