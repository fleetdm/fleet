package fleet

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
