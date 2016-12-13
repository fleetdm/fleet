package kolide

import (
	"golang.org/x/net/context"
)

type PackStore interface {
	// Pack methods
	NewPack(pack *Pack) (*Pack, error)
	SavePack(pack *Pack) error
	DeletePack(pid uint) error
	Pack(pid uint) (*Pack, error)
	ListPacks(opt ListOptions) ([]*Pack, error)

	// Modifying the labels for packs
	AddLabelToPack(lid uint, pid uint) error
	ListLabelsForPack(pid uint) ([]*Label, error)
	RemoveLabelFromPack(label *Label, pack *Pack) error

	ListHostsInPack(pid uint, opt ListOptions) ([]*Host, error)
}

type PackService interface {
	ListPacks(ctx context.Context, opt ListOptions) ([]*Pack, error)
	GetPack(ctx context.Context, id uint) (*Pack, error)
	NewPack(ctx context.Context, p PackPayload) (*Pack, error)
	ModifyPack(ctx context.Context, id uint, p PackPayload) (*Pack, error)
	DeletePack(ctx context.Context, id uint) error

	AddLabelToPack(ctx context.Context, lid, pid uint) error
	ListLabelsForPack(ctx context.Context, pid uint) ([]*Label, error)
	RemoveLabelFromPack(ctx context.Context, lid, pid uint) error

	ListPacksForHost(ctx context.Context, hid uint) ([]*Pack, error)
	ListHostsInPack(ctx context.Context, pid uint, opt ListOptions) ([]*Host, error)
}

type Pack struct {
	UpdateCreateTimestamps
	DeleteFields
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Platform    string `json:"platform"`
	CreatedBy   uint   `json:"created_by" db:"created_by"`
	Disabled    bool   `json:"disabled"`
}

type PackPayload struct {
	Name        *string
	Description *string
	Platform    *string
	Disabled    *bool
}

type PackTarget struct {
	ID     uint
	PackID uint
	Target
}
