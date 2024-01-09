package goose

import (
	"database/sql"
	"fmt"
	"time"
)

// Create writes a new blank migration file.
func Create(db *sql.DB, dir, name, migrationType string) error {
	paths, err := CreateMigration(name, migrationType, dir, time.Now())
	if err != nil {
		return err
	}
	fmt.Printf("Created %s migration files at %v\n", migrationType, paths)

	return nil
}
