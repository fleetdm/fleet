package kolide

import "time"

// Createable contains common timestamp fields indicating create time
type CreateTimestamp struct {
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Deleteable is used to indicate a record is deleted.  We don't actually
// delete record in the database. We mark it deleted, records with Deleted
// set to true will not normally be included in results
type DeleteFields struct {
	DeletedAt *time.Time `json:"deleted_at" db:"deleted_at"`
	Deleted   bool       `json:"deleted"`
}

// MarkDeleted indicates a record is deleted. It won't actually be removed from
// the database, but won't be returned in result sets.
func (d *DeleteFields) MarkDeleted(deleted time.Time) {
	d.DeletedAt = &deleted
	d.Deleted = true
}

// UpdateTimestamp contains a timestamp that is set whenever an entity is changed
type UpdateTimestamp struct {
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type UpdateCreateTimestamps struct {
	CreateTimestamp
	UpdateTimestamp
}
