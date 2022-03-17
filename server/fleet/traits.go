package fleet

import "time"

// CreateTimestamp contains common timestamp fields indicating create time
type CreateTimestamp struct {
	CreatedAt time.Time `json:"created_at" db:"created_at" csv:"created_at"`
}

// UpdateTimestamp contains a timestamp that is set whenever an entity is changed
type UpdateTimestamp struct {
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" csv:"updated_at"`
}

type UpdateCreateTimestamps struct {
	CreateTimestamp
	UpdateTimestamp
}
