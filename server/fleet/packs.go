package fleet

import (
	"errors"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/ptr"
)

// MaxScheduledQueryInterval is the maximum interval value (in seconds) allowed by osquery
const MaxScheduledQueryInterval = 604800

type PackListOptions struct {
	ListOptions

	// IncludeSystemPacks will include Global & Team Packs while listing packs
	IncludeSystemPacks bool
}

// Pack is the structure which represents an osquery query pack.
//
// NOTE: A "team pack" is a special type of pack with Pack.Type="team-$TEAM_ID".
// Such team packs hold the scheduled queries for a team. This is different from a
// pack that has a team as target (Pack.Teams and Pack.TeamIDs fields).
type Pack struct {
	UpdateCreateTimestamps
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Platform    string `json:"platform,omitempty"`
	Disabled    bool   `json:"disabled"`
	// Type indicates the type of the pack:
	//	- "global" is the type of the global pack.
	//	- "team-$ID" is the type for team packs.
	//	- nil is the type for a user created pack.
	Type     *string  `json:"type" db:"pack_type"`
	Labels   []Target `json:"labels"`
	LabelIDs []uint   `json:"label_ids"`
	Hosts    []Target `json:"hosts"`
	HostIDs  []uint   `json:"host_ids"`
	Teams    []Target `json:"teams"`
	// TeamIDs holds the ID of the teams this pack should target.
	TeamIDs []uint `json:"team_ids"`

	/////////////////////////////////////////////////////////////////
	// WARNING: If you add to this struct make sure it's taken into
	// account in the Clone implementation of Pack!
	/////////////////////////////////////////////////////////////////
}

// Clone implements cloner for Pack.
func (p *Pack) Clone() (Cloner, error) {
	return p.Copy(), nil
}

// Copy returns a deep copy of the Pack.
func (p *Pack) Copy() *Pack {
	if p == nil {
		return nil
	}

	clone := *p
	if p.Type != nil {
		clone.Type = ptr.String(*p.Type)
	}
	if p.Labels != nil {
		clone.Labels = make([]Target, len(p.Labels))
		copy(clone.Labels, p.Labels)
	}
	if p.LabelIDs != nil {
		clone.LabelIDs = make([]uint, len(p.LabelIDs))
		copy(clone.LabelIDs, p.LabelIDs)
	}
	if p.Hosts != nil {
		clone.Hosts = make([]Target, len(p.Hosts))
		copy(clone.Hosts, p.Hosts)
	}
	if p.HostIDs != nil {
		clone.HostIDs = make([]uint, len(p.HostIDs))
		copy(clone.HostIDs, p.HostIDs)
	}
	if p.Teams != nil {
		clone.Teams = make([]Target, len(p.Teams))
		copy(clone.Teams, p.Teams)
	}
	if p.TeamIDs != nil {
		clone.TeamIDs = make([]uint, len(p.TeamIDs))
		copy(clone.TeamIDs, p.TeamIDs)
	}
	return &clone
}

// isTeamPack returns true if the pack is a pack specifically made for a team.
func (p *Pack) isTeamPack() bool {
	return p.Type != nil && strings.HasPrefix(*p.Type, "team-")
}

// isGlobalPack returns true if the pack is the global pack.
func (p *Pack) isGlobalPack() bool {
	return p.Type != nil && *p.Type == "global"
}

// TeamPack returns the team ID for a team's pack.
// Returns (nil, nil) if the pack is not a team pack.
func (p *Pack) teamPack() (*uint, error) {
	if !p.isTeamPack() {
		return nil, nil
	}
	t := strings.TrimPrefix(*p.Type, "team-")
	teamID, err := strconv.ParseUint(t, 10, 32)
	if err != nil {
		return nil, err
	}
	return ptr.Uint(uint(teamID)), nil
}

// ExtraAuthz implements authz.ExtraAuthzer.
func (p *Pack) ExtraAuthz() (map[string]interface{}, error) {
	packTeamID, err := p.teamPack()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"pack_team_id":   packTeamID,
		"is_global_pack": p.isGlobalPack(),
	}, nil
}

// Verify verifies the pack's fields are valid.
func (p *Pack) Verify() error {
	if emptyString(p.Name) {
		return errPackEmptyName
	}
	return nil
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

var (
	errPackEmptyName       = errors.New("pack name cannot be empty")
	errPackInvalidInterval = errors.New("pack scheduled query interval must be an integer greater than 1 and less than 604800")
)

// Verify verifies the pack's payload fields are valid.
func (p *PackPayload) Verify() error {
	if p.Name != nil {
		if emptyString(*p.Name) {
			return errPackEmptyName
		}
	}
	return nil
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

// Verify verifies the pack's spec fields are valid.
func (p *PackSpec) Verify() error {
	if emptyString(p.Name) {
		return errPackEmptyName
	}
	for _, sq := range p.Queries {
		if sq.Interval < 1 || sq.Interval > MaxScheduledQueryInterval {
			return errPackInvalidInterval
		}
	}
	return nil
}

type PackSpecTargets struct {
	Labels []string `json:"labels"`
	Teams  []string `json:"teams"`
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

type PackStats struct {
	PackID   uint   `json:"pack_id"`
	PackName string `json:"pack_name"`
	// Type indicates the type of the pack:
	//	- "global" is the type of the global pack.
	//	- "team-$ID" is returned for team packs.
	//	- "pack" means it is a user created pack.
	Type       string                `json:"type"`
	QueryStats []ScheduledQueryStats `json:"query_stats"`
}
