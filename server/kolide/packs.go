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
	AddLabelToPack(lid, pid uint) error
	RemoveLabelFromPack(lid, pid uint) error
	ListLabelsForPack(pid uint) ([]*Label, error)

	// Modifying the hosts for packs
	AddHostToPack(hid uint, pid uint) error
	RemoveHostFromPack(hid uint, pid uint) error
	ListHostsInPack(pid uint, opt ListOptions) ([]*Host, error)
}

type PackService interface {
	// Pack methods
	ListPacks(ctx context.Context, opt ListOptions) (packs []*Pack, err error)
	GetPack(ctx context.Context, id uint) (pack *Pack, err error)
	NewPack(ctx context.Context, p PackPayload) (pack *Pack, err error)
	ModifyPack(ctx context.Context, id uint, p PackPayload) (pack *Pack, err error)
	DeletePack(ctx context.Context, id uint) (err error)

	// Modifying the labels for packs
	AddLabelToPack(ctx context.Context, lid, pid uint) (err error)
	RemoveLabelFromPack(ctx context.Context, lid, pid uint) (err error)
	ListLabelsForPack(ctx context.Context, pid uint) (labels []*Label, err error)

	// Modifying the hosts for packs
	AddHostToPack(ctx context.Context, hid, pid uint) (err error)
	RemoveHostFromPack(ctx context.Context, hid, pid uint) (err error)
	ListPacksForHost(ctx context.Context, hid uint) (packs []*Pack, err error)
	ListHostsInPack(ctx context.Context, pid uint, opt ListOptions) (hosts []*Host, err error)
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
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Platform    *string `json:"platform"`
	Disabled    *bool   `json:"disabled"`
	HostIDs     *[]uint `json:"host_ids"`
	LabelIDs    *[]uint `json:"label_ids"`
}

type PackTarget struct {
	ID     uint
	PackID uint
	Target
}
