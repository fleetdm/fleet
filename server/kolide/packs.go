package kolide

import (
	"context"
)

// PackStore is the datastore interface for managing query packs.
type PackStore interface {
	// ApplyPackSpecs applies a list of PackSpecs to the datastore,
	// creating and updating packs as necessary.
	ApplyPackSpecs(specs []*PackSpec) error
	// GetPackSpecs returns all of the stored PackSpecs.
	GetPackSpecs() ([]*PackSpec, error)
	// GetPackSpec returns the spec for the named pack.
	GetPackSpec(name string) (*PackSpec, error)

	// NewPack creates a new pack in the datastore.
	NewPack(pack *Pack, opts ...OptionalArg) (*Pack, error)

	// SavePack updates an existing pack in the datastore.
	SavePack(pack *Pack) error

	// DeletePack deletes a pack record from the datastore.
	DeletePack(name string) error

	// Pack retrieves a pack from the datastore by ID.
	Pack(pid uint) (*Pack, error)

	// ListPacks lists all packs in the datastore.
	ListPacks(opt ListOptions) ([]*Pack, error)

	// PackByName fetches pack if it exists, if the pack
	// exists the bool return value is true
	PackByName(name string, opts ...OptionalArg) (*Pack, bool, error)

	// AddLabelToPack adds an existing label to an existing pack, both by ID.
	AddLabelToPack(lid, pid uint, opts ...OptionalArg) error

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

	// ListPacksForHost lists the packs that a host should execute.
	ListPacksForHost(hid uint) (packs []*Pack, err error)

	// ListHostsInPack lists the IDs of all hosts that are associated with a pack
	// through labels.
	ListHostsInPack(pid uint, opt ListOptions) ([]uint, error)

	// ListExplicitHostsInPack lists the IDs of hosts that have been manually
	// associated with a query pack.
	ListExplicitHostsInPack(pid uint, opt ListOptions) ([]uint, error)
}

// PackService is the service interface for managing query packs.
type PackService interface {
	// ApplyPackSpecs applies a list of PackSpecs to the datastore,
	// creating and updating packs as necessary.
	ApplyPackSpecs(ctx context.Context, specs []*PackSpec) error
	// GetPackSpecs returns all of the stored PackSpecs.
	GetPackSpecs(ctx context.Context) ([]*PackSpec, error)
	// GetPackSpec gets the spec for the pack with the given name.
	GetPackSpec(ctx context.Context, name string) (*PackSpec, error)

	// NewPack creates a new pack in the datastore.
	NewPack(ctx context.Context, p PackPayload) (pack *Pack, err error)

	// ModifyPack modifies an existing pack in the datastore.
	ModifyPack(ctx context.Context, id uint, p PackPayload) (pack *Pack, err error)

	// ListPacks lists all packs in the application.
	ListPacks(ctx context.Context, opt ListOptions) (packs []*Pack, err error)

	// GetPack retrieves a pack by ID.
	GetPack(ctx context.Context, id uint) (pack *Pack, err error)

	// DeletePack deletes a pack record from the datastore.
	DeletePack(ctx context.Context, name string) (err error)

	// DeletePackByID is for backwards compatibility with the UI
	DeletePackByID(ctx context.Context, id uint) (err error)

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

	// ListHostsInPack lists the IDs of all hosts that are associated with a pack,
	// both through labels and manual associations.
	ListHostsInPack(ctx context.Context, pid uint, opt ListOptions) (hosts []uint, err error)

	// ListExplicitHostsInPack lists the IDs of hosts that have been manually associated
	// with a query pack.
	ListExplicitHostsInPack(ctx context.Context, pid uint, opt ListOptions) (hosts []uint, err error)
}

// Pack is the structure which represents an osquery query pack.
type Pack struct {
	UpdateCreateTimestamps
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Platform    string `json:"platform,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
}

const (
	PackKind = "pack"
)

// PackPayload is the struct which is used to create/update packs.
type PackPayload struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Platform    *string `json:"platform"`
	Disabled    *bool   `json:"disabled"`
	HostIDs     *[]uint `json:"host_ids"`
	LabelIDs    *[]uint `json:"label_ids"`
}

type PackSpec struct {
	ID          uint            `json:"id,omitempty"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Platform    string          `json:"platform,omitempty"`
	Disabled    bool            `json:"disabled"`
	Targets     PackSpecTargets `json:"targets,omitempty"`
	Queries     []PackSpecQuery `json:"queries,omitempty"`
}

type PackSpecTargets struct {
	Labels []string `json:"labels"`
}

type PackSpecQuery struct {
	QueryName   string  `json:"query" db:"query_name"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Interval    uint    `json:"interval"`
	Snapshot    *bool   `json:"snapshot,omitempty"`
	Removed     *bool   `json:"removed,omitempty"`
	Shard       *uint   `json:"shard,omitempty"`
	Platform    *string `json:"platform,omitempty"`
	Version     *string `json:"version,omitempty"`
	Denylist    *bool   `json:"denylist,omitempty"`
}

// PackTarget associates a pack with either a host or a label
type PackTarget struct {
	ID     uint
	PackID uint
	Target
}
