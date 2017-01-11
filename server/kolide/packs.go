package kolide

import (
	"golang.org/x/net/context"
)

// PackStore is the datastore interface for managing query packs.
type PackStore interface {
	// NewPack creates a new pack in the datastore.
	NewPack(pack *Pack) (*Pack, error)

	// SavePack updates an existing pack in the datastore.
	SavePack(pack *Pack) error

	// DeletePack deletes a pack record from the datastore.
	DeletePack(pid uint) error

	// Pack retrieves a pack from the datastore by ID.
	Pack(pid uint) (*Pack, error)

	// ListPacks lists all packs in the datastore.
	ListPacks(opt ListOptions) ([]*Pack, error)

	// AddLabelToPack adds an existing label to an existing pack, both by ID.
	AddLabelToPack(lid, pid uint) error

	// RemoveLabelFromPack removes an existing label from it's association with
	// an existing pack, both by ID.
	RemoveLabelFromPack(lid, pid uint) error

	// ListLabelsForPack lists all labels that are associated with a pack.
	ListLabelsForPack(pid uint) ([]*Label, error)

	// AddHostToPack adds an existing host to an existing pack, both by ID.
	AddHostToPack(hid uint, pid uint) error

	// RemoveHostFromPack removes an existing host from it's association with
	// an existing pack, both by ID.
	RemoveHostFromPack(hid uint, pid uint) error

	// ListHostsInPack lists all hosts that are associated with a pack, both
	// through labels and manual associations.
	ListHostsInPack(pid uint, opt ListOptions) ([]*Host, error)

	// ListExplicitHostsInPack lists hosts that have been manually associated
	// with a query pack.
	ListExplicitHostsInPack(pid uint, opt ListOptions) ([]*Host, error)
}

// PackService is the service interface for managing query packs.
type PackService interface {
	// ListPacks lists all packs in the application.
	ListPacks(ctx context.Context, opt ListOptions) (packs []*Pack, err error)

	// GetPack retrieves a pack by ID.
	GetPack(ctx context.Context, id uint) (pack *Pack, err error)

	// NewPack creates a new pack in the datastore.
	NewPack(ctx context.Context, p PackPayload) (pack *Pack, err error)

	// ModifyPack modifies an existing pack in the datastore.
	ModifyPack(ctx context.Context, id uint, p PackPayload) (pack *Pack, err error)

	// DeletePack deletes a pack record from the datastore.
	DeletePack(ctx context.Context, id uint) (err error)

	// AddLabelToPack adds an existing label to an existing pack, both by ID.
	AddLabelToPack(ctx context.Context, lid, pid uint) (err error)

	// RemoveLabelFromPack removes an existing label from it's association with
	// an existing pack, both by ID.
	RemoveLabelFromPack(ctx context.Context, lid, pid uint) (err error)

	// ListLabelsForPack lists all labels that are associated with a pack.
	ListLabelsForPack(ctx context.Context, pid uint) (labels []*Label, err error)

	// AddHostToPack adds an existing host to an existing pack, both by ID.
	AddHostToPack(ctx context.Context, hid, pid uint) (err error)

	// RemoveHostFromPack removes an existing host from it's association with
	// an existing pack, both by ID.
	RemoveHostFromPack(ctx context.Context, hid, pid uint) (err error)

	// ListPacksForHost lists the packs that a host should execute.
	ListPacksForHost(ctx context.Context, hid uint) (packs []*Pack, err error)

	// ListHostsInPack lists all hosts that are associated with a pack, both
	// through labels and manual associations.
	ListHostsInPack(ctx context.Context, pid uint, opt ListOptions) (hosts []*Host, err error)

	// ListExplicitHostsInPack lists hosts that have been manually associated
	// with a query pack.
	ListExplicitHostsInPack(ctx context.Context, pid uint, opt ListOptions) (hosts []*Host, err error)
}

// Pack is the structure which represents an osquery query pack.
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

// PackPayload is the struct which is used to create/update packs.
type PackPayload struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Platform    *string `json:"platform"`
	Disabled    *bool   `json:"disabled"`
	HostIDs     *[]uint `json:"host_ids"`
	LabelIDs    *[]uint `json:"label_ids"`
}

// PackTarget associates a pack with either a host or a label
type PackTarget struct {
	ID     uint
	PackID uint
	Target
}
