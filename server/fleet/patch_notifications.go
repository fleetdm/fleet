package fleet

import (
	"encoding/json"
	"time"
)

const PatchNotificationKind = "patch"

type PatchNotificationApp struct {
	PolicyID            *uint `db:"policy_id"`
	SoftwareTitleID     uint  `db:"software_title_id"`
	SoftwareInstallerID *uint `db:"software_installer_id"`
}

// A patch notification whose deadline is close enough that Fleet has something to do with it now:
// send the reminder, or force the install.
type PatchNotificationDue struct {
	NotificationUUID string          `db:"notification_uuid"`
	HostID           uint            `db:"host_id"`
	Status           string          `db:"status"`
	Payload          json.RawMessage `db:"payload"`
	DisplayedAt      *time.Time      `db:"displayed_at"`
	CreatedAt        time.Time       `db:"created_at"`
	InstallAt        time.Time       `db:"install_at"`
}

// What a host has installed for one software title. A title can have more than one row when several
// copies are installed.
type HostSoftwareTitleVersion struct {
	HostID          uint   `db:"host_id"`
	SoftwareTitleID uint   `db:"title_id"`
	Version         string `db:"version"`
}

type PatchNotificationAppDetail struct {
	// NotificationUUID is only set when the apps of several notifications are listed at once.
	NotificationUUID string `db:"notification_uuid"`
	// InstallerVersion is the version the installer would put on the host, empty when the installer
	// is gone.
	InstallerVersion string `db:"installer_version"`

	PolicyID            *uint  `db:"policy_id"`
	SoftwareTitleID     uint   `db:"software_title_id"`
	SoftwareInstallerID *uint  `db:"software_installer_id"`
	Name                string `db:"name"`
	DisplayName         string `db:"display_name"`
	HasIcon             bool   `db:"has_icon"`
	InstallQueued       bool   `db:"install_queued"`
}
