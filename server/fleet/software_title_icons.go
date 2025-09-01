package fleet

import (
	"context"
	"io"
	"time"
)

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

type SoftwareTitleIconStore interface {
	Put(ctx context.Context, iconID string, content io.ReadSeeker) error
	Get(ctx context.Context, iconID string) (io.ReadCloser, int64, error)
	Exists(ctx context.Context, iconID string) (bool, error)
	Cleanup(ctx context.Context, usedIconIDs []string, removeCreatedBefore time.Time) (int, error)
	Sign(ctx context.Context, iconID string) (string, error)
}

type SoftwareTitleIconActivity struct {
	SoftwareInstallerID *uint                   `db:"software_installer_id"`
	AdamID              *string                 `db:"adam_id"`
	SoftwareTitle       string                  `db:"software_title"`
	Filename            string                  `db:"filename"`
	TeamName            string                  `db:"team_name"`
	TeamID              uint                    `db:"team_id"`
	SelfService         bool                    `db:"self_service"`
	SoftwareTitleID     uint                    `db:"software_title_id"`
	LabelsIncludeAny    []ActivitySoftwareLabel `db:"-"`
	LabelsExcludeAny    []ActivitySoftwareLabel `db:"-"`
}
