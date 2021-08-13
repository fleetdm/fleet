package fleet

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
	ListPacks(opt PackListOptions) ([]*Pack, error)

	// PackByName fetches pack if it exists, if the pack
	// exists the bool return value is true
	PackByName(name string, opts ...OptionalArg) (*Pack, bool, error)

	// ListPacksForHost lists the packs that a host should execute.
	ListPacksForHost(hid uint) (packs []*Pack, err error)

	// EnsureGlobalPack gets or inserts a pack with type global
	EnsureGlobalPack() (*Pack, error)

	// EnsureTeamPack gets or inserts a pack with type global
	EnsureTeamPack(teamID uint) (*Pack, error)
}

// PackService is the service interface for managing query packs.
type PackService interface {
	// ApplyPackSpecs applies a list of PackSpecs to the datastore,
	// creating and updating packs as necessary.
	ApplyPackSpecs(ctx context.Context, specs []*PackSpec) ([]*PackSpec, error)
	// GetPackSpecs returns all of the stored PackSpecs.
	GetPackSpecs(ctx context.Context) ([]*PackSpec, error)
	// GetPackSpec gets the spec for the pack with the given name.
	GetPackSpec(ctx context.Context, name string) (*PackSpec, error)

	// NewPack creates a new pack in the datastore.
	NewPack(ctx context.Context, p PackPayload) (pack *Pack, err error)

	// ModifyPack modifies an existing pack in the datastore.
	ModifyPack(ctx context.Context, id uint, p PackPayload) (pack *Pack, err error)

	// ListPacks lists all packs in the application.
	ListPacks(ctx context.Context, opt PackListOptions) (packs []*Pack, err error)

	// GetPack retrieves a pack by ID.
	GetPack(ctx context.Context, id uint) (pack *Pack, err error)

	// DeletePack deletes a pack record from the datastore.
	DeletePack(ctx context.Context, name string) (err error)

	// DeletePackByID is for backwards compatibility with the UI
	DeletePackByID(ctx context.Context, id uint) (err error)

	// ListPacksForHost lists the packs that a host should execute.
	ListPacksForHost(ctx context.Context, hid uint) (packs []*Pack, err error)
}

type PackListOptions struct {
	ListOptions

	// IncludeSystemPacks will include Global & Team Packs while listing packs
	IncludeSystemPacks bool
}

// Pack is the structure which represents an osquery query pack.
type Pack struct {
	UpdateCreateTimestamps
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Platform    string  `json:"platform,omitempty"`
	Disabled    bool    `json:"disabled,omitempty"`
	Type        *string `json:"type" db:"pack_type"`
	LabelIDs    []uint  `json:"label_ids"`
	HostIDs     []uint  `json:"host_ids"`
	TeamIDs     []uint  `json:"team_ids"`
}

// EditablePackType only returns true when the pack doesn't have a specific Type set, only nil & empty string Pack.Type
// is editable https://github.com/fleetdm/fleet/issues/1485
func (p *Pack) EditablePackType() bool {
	return p != nil && (p.Type == nil || (p.Type != nil && *p.Type == ""))
}

func (p Pack) AuthzType() string {
	return "pack"
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
	TeamIDs     *[]uint `json:"team_ids"`
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

// PackTarget targets a pack to a host, label, or team.
type PackTarget struct {
	ID     uint `db:"id"`
	PackID uint `db:"pack_id"`
	Target
}

type PackStats struct {
	PackID     uint                  `json:"pack_id,omitempty"`
	PackName   string                `json:"pack_name,omitempty"`
	QueryStats []ScheduledQueryStats `json:"query_stats"`
}
