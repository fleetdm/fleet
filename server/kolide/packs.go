package kolide

import (
	"time"

	"golang.org/x/net/context"
)

type PackStore interface {
	// Pack methods
	NewPack(pack *Pack) error
	SavePack(pack *Pack) error
	DeletePack(pid uint) error
	Pack(pid uint) (*Pack, error)
	Packs() ([]*Pack, error)

	// Modifying the queries in packs
	AddQueryToPack(qid uint, pid uint) error
	GetQueriesInPack(pack *Pack) ([]*Query, error)
	RemoveQueryFromPack(query *Query, pack *Pack) error

	// Modifying the labels for packs
	AddLabelToPack(lid uint, pid uint) error
	GetLabelsForPack(pack *Pack) ([]*Label, error)
	RemoveLabelFromPack(label *Label, pack *Pack) error

	// Packs from the host's perspective
	ActivePacksForHost(hid uint) ([]*Pack, error)
}

type PackService interface {
	ListPacks(ctx context.Context) ([]*Pack, error)
	GetPack(ctx context.Context, id uint) (*Pack, error)
	NewPack(ctx context.Context, p PackPayload) (*Pack, error)
	ModifyPack(ctx context.Context, id uint, p PackPayload) (*Pack, error)
	DeletePack(ctx context.Context, id uint) error

	AddQueryToPack(ctx context.Context, qid, pid uint) error
	GetQueriesInPack(ctx context.Context, id uint) ([]*Query, error)
	RemoveQueryFromPack(ctx context.Context, qid, pid uint) error

	AddLabelToPack(ctx context.Context, lid, pid uint) error
	GetLabelsForPack(ctx context.Context, pid uint) ([]*Label, error)
	RemoveLabelFromPack(ctx context.Context, lid, pid uint) error
}

type Pack struct {
	ID        uint      `json:"id" gorm:"primary_key"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
	Name      string    `json:"name" gorm:"not null;unique_index:idx_pack_unique_name"`
	Platform  string    `json:"platform"`
}

type PackQuery struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	PackID    uint
	QueryID   uint
}

type TargetType int

const (
	TargetLabel TargetType = iota
	TargetHost
)

type PackTarget struct {
	ID       uint `gorm:"primary_key"`
	Type     TargetType
	PackID   uint
	TargetID uint
}
