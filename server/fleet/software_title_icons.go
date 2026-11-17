package fleet

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"time"
)

var SoftwareTitleIconURLRegex = regexp.MustCompile(`fleet/software/titles/\d+/icon\?(?:team_id|fleet_id)=\d+`)

const SoftwareTitleIconSignedURLExpiry = 6 * time.Hour

type UploadSoftwareTitleIconPayload struct {
	TitleID   uint
	TeamID    uint
	Filename  string
	StorageID string
	IconFile  *TempFileReader
}

type SoftwareTitleIcon struct {
	// TeamID is the fleet (or no fleet if 0) the icon is associated with. It MUST have the JSON tag `fleet_id` so that the rego policies
	// can check fleet ownership (rego reads the struct through its JSON). This struct is never marshalled directly in API responses.
	TeamID          uint   `db:"team_id" json:"fleet_id"`
	SoftwareTitleID uint   `db:"software_title_id"`
	StorageID       string `db:"storage_id"`
	Filename        string `db:"filename"`
}

func (s *SoftwareTitleIcon) AuthzType() string {
	return "installable_entity"
}

func (s *SoftwareTitleIcon) IconUrl() string {
	return fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?fleet_id=%d", s.SoftwareTitleID, s.TeamID)
}

func (s *SoftwareTitleIcon) IconUrlWithDeviceToken(deviceToken string) string {
	return fmt.Sprintf("/api/latest/fleet/device/%s/software/titles/%d/icon", deviceToken, s.SoftwareTitleID)
}

type SoftwareTitleIconStore interface {
	Put(ctx context.Context, iconID string, content io.ReadSeeker) error
	Get(ctx context.Context, iconID string) (io.ReadCloser, int64, error)
	Exists(ctx context.Context, iconID string) (bool, error)
	Cleanup(ctx context.Context, usedIconIDs []string, removeCreatedBefore time.Time) (int, error)
	Sign(ctx context.Context, iconID string, expiresIn time.Duration) (string, error)
}

type DetailsForSoftwareIconActivity struct {
	SoftwareInstallerID *uint   `db:"software_installer_id"`
	InHouseAppID        *uint   `db:"in_house_app_id"`
	AdamID              *string `db:"adam_id"`
	VPPAppTeamID        *uint   `db:"vpp_app_team_id"`
	VPPTokenID          *uint   `db:"vpp_token_id"`
	// VersionName is the admin-provided label of the first-added App Store app
	// version on this title (`MIN(vpp_apps_teams.id)` for the title+team). It
	// reflects the first-added version by design — icon-edit activities fire
	// title-wide rather than per-version, so a single representative label is
	// surfaced. Empty for installer- or in-house-app-only titles.
	VersionName         string                     `db:"version_name"`
	VPPIconUrl          *string                    `db:"vpp_icon_url"`
	SoftwareTitle       string                     `db:"software_title"`
	SoftwareDisplayName string                     `db:"software_display_name"`
	Filename            *string                    `db:"filename"`
	TeamName            *string                    `db:"team_name"`
	TeamID              uint                       `db:"team_id"`
	SelfService         bool                       `db:"self_service"`
	SoftwareTitleID     uint                       `db:"software_title_id"`
	Platform            *InstallableDevicePlatform `json:"platform"`
	LabelsIncludeAny    []ActivitySoftwareLabel    `db:"-"`
	LabelsExcludeAny    []ActivitySoftwareLabel    `db:"-"`
	LabelsIncludeAll    []ActivitySoftwareLabel    `db:"-"`
}
