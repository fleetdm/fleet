package fleet

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"time"
)

var SoftwareTitleIconURLRegex = regexp.MustCompile(`fleet/software/titles/\d+/icon\?team_id=\d+`)

type UploadSoftwareTitleIconPayload struct {
	TitleID   uint
	TeamID    uint
	Filename  string
	StorageID string
	IconFile  *TempFileReader
}

type SoftwareTitleIcon struct {
	TeamID          uint   `db:"team_id"`
	SoftwareTitleID uint   `db:"software_title_id"`
	StorageID       string `db:"storage_id"`
	Filename        string `db:"filename"`
}

func (s *SoftwareTitleIcon) AuthzType() string {
	return "installable_entity"
}

func (s *SoftwareTitleIcon) IconUrl() string {
	return fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?team_id=%d", s.SoftwareTitleID, s.TeamID)
}

func (s *SoftwareTitleIcon) IconUrlWithDeviceToken(deviceToken string) string {
	return fmt.Sprintf("/api/latest/fleet/device/%s/software/titles/%d/icon", deviceToken, s.SoftwareTitleID)
}

type SoftwareTitleIconStore interface {
	Put(ctx context.Context, iconID string, content io.ReadSeeker) error
	Get(ctx context.Context, iconID string) (io.ReadCloser, int64, error)
	Exists(ctx context.Context, iconID string) (bool, error)
	Cleanup(ctx context.Context, usedIconIDs []string, removeCreatedBefore time.Time) (int, error)
	Sign(ctx context.Context, iconID string) (string, error)
}

type DetailsForSoftwareIconActivity struct {
	SoftwareInstallerID *uint                   `db:"software_installer_id"`
	InHouseAppID        *uint                   `db:"in_house_app_id"`
	AdamID              *string                 `db:"adam_id"`
	VPPAppTeamID        *uint                   `db:"vpp_app_team_id"`
	VPPIconUrl          *string                 `db:"vpp_icon_url"`
	SoftwareTitle       string                  `db:"software_title"`
	Filename            *string                 `db:"filename"`
	TeamName            *string                 `db:"team_name"`
	TeamID              uint                    `db:"team_id"`
	SelfService         bool                    `db:"self_service"`
	SoftwareTitleID     uint                    `db:"software_title_id"`
	Platform            *InstallableDevicePlatform    `json:"platform"`
	LabelsIncludeAny    []ActivitySoftwareLabel `db:"-"`
	LabelsExcludeAny    []ActivitySoftwareLabel `db:"-"`
}
