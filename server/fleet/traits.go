package fleet

import "time"

// Createable contains common timestamp fields indicating create time
type CreateTimestamp struct {
	CreatedAt time.Time `json:"created_at" db:"created_at" goqu:"skipupdate"`
}

// UpdateTimestamp contains a timestamp that is set whenever an entity is changed
type UpdateTimestamp struct {
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" goqu:"skipupdate"`
}

type UpdateCreateTimestamps struct {
	CreateTimestamp
	UpdateTimestamp
}
