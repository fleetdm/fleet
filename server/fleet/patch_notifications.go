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

// A patch notification whose deadline is close enough that the countdown pass has something to do
// with it: send the reminder, or force the install.
type PatchNotificationDue struct {
	NotificationUUID string          `db:"notification_uuid"`
	HostID           uint            `db:"host_id"`
	Status           string          `db:"status"`
	Payload          json.RawMessage `db:"payload"`
	DisplayedAt      *time.Time      `db:"displayed_at"`
	CreatedAt        time.Time       `db:"created_at"`
	InstallAt        time.Time       `db:"install_at"`
}

type PatchNotificationAppDetail struct {
	PolicyID            *uint  `db:"policy_id"`
	SoftwareTitleID     uint   `db:"software_title_id"`
	SoftwareInstallerID *uint  `db:"software_installer_id"`
	Name                string `db:"name"`
	DisplayName         string `db:"display_name"`
	HasIcon             bool   `db:"has_icon"`
	InstallQueued       bool   `db:"install_queued"`
}
