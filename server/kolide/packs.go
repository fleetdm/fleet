package kolide

import (
	"time"

	"golang.org/x/net/context"
)

type PackStore interface {
	// Pack methods
	NewPack(pack *Pack) error
	SavePack(pack *Pack) error
	DeletePack(pack *Pack) error
	Pack(id uint) (*Pack, error)
	Packs() ([]*Pack, error)

	// Modifying the queries in packs
	AddQueryToPack(query *Query, pack *Pack) error
	GetQueriesInPack(pack *Pack) ([]*Query, error)
	RemoveQueryFromPack(query *Query, pack *Pack) error
}

type PackService interface {
	GetAllPacks(ctx context.Context) ([]*Pack, error)
	GetPack(ctx context.Context, id uint) (*Pack, error)
	NewPack(ctx context.Context, p PackPayload) (*Pack, error)
	ModifyPack(ctx context.Context, id uint, p PackPayload) (*Pack, error)
	DeletePack(ctx context.Context, id uint) error

	AddQueryToPack(ctx context.Context, qid, pid uint) error
	GetQueriesInPack(ctx context.Context, id uint) ([]*Query, error)
	RemoveQueryFromPack(ctx context.Context, qid, pid uint) error
}

type Pack struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string `gorm:"not null;unique_index:idx_pack_unique_name"`
	Platform  string
}

type PackQuery struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	PackID    uint
	QueryID   uint
}

type PackTarget struct {
	ID       uint `gorm:"primary_key"`
	PackID   uint
	TargetID uint
}
