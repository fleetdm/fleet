package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// GetHostPet returns the pet associated with the given host. Returns a
// not-found error when the host has not adopted a pet yet.
func (ds *Datastore) GetHostPet(ctx context.Context, hostID uint) (*fleet.HostPet, error) {
	const stmt = `
		SELECT
			id, host_id, name, species,
			health, happiness, hunger, cleanliness,
			last_interacted_at, created_at, updated_at
		FROM host_pets
		WHERE host_id = ?`

	var pet fleet.HostPet
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &pet, stmt, hostID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("HostPet").WithID(hostID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get host pet")
	}
	return &pet, nil
}

// CreateHostPet adopts a new pet for the given host. Returns an already-exists
// error if the host already has a pet. Stats start at sensible defaults.
func (ds *Datastore) CreateHostPet(ctx context.Context, hostID uint, name, species string) (*fleet.HostPet, error) {
	const insertStmt = `
		INSERT INTO host_pets (host_id, name, species)
		VALUES (?, ?, ?)`

	res, err := ds.writer(ctx).ExecContext(ctx, insertStmt, hostID, name, species)
	if err != nil {
		if IsDuplicate(err) {
			return nil, ctxerr.Wrap(ctx, alreadyExists("HostPet", hostID))
		}
		return nil, ctxerr.Wrap(ctx, err, "create host pet")
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host pet last insert id")
	}

	pet, err := ds.GetHostPet(ctx, hostID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load newly-created host pet")
	}
	// Belt-and-braces: make sure the returned pet is the one we just wrote.
	if pet.ID != uint(id) { //nolint:gosec // safe narrowing from AUTO_INCREMENT id
		return nil, ctxerr.New(ctx, "created pet id mismatch")
	}
	return pet, nil
}

// SaveHostPet persists the pet's stats and last_interacted_at.
func (ds *Datastore) SaveHostPet(ctx context.Context, pet *fleet.HostPet) error {
	const stmt = `
		UPDATE host_pets
		SET
			name = ?,
			species = ?,
			health = ?,
			happiness = ?,
			hunger = ?,
			cleanliness = ?,
			last_interacted_at = ?
		WHERE id = ?`

	res, err := ds.writer(ctx).ExecContext(ctx, stmt,
		pet.Name,
		pet.Species,
		pet.Health,
		pet.Happiness,
		pet.Hunger,
		pet.Cleanliness,
		pet.LastInteractedAt,
		pet.ID,
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "save host pet")
	}
	n, err := res.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "save host pet rows affected")
	}
	if n == 0 {
		return ctxerr.Wrap(ctx, notFound("HostPet").WithID(pet.ID))
	}
	return nil
}
