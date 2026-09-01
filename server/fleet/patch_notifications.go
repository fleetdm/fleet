package fleet

import "time"

const PatchNotificationKind = "patch"

// PatchNotification is the patch half of an end user notification: when its
// apps are due to be installed, and how far it has got.
type PatchNotification struct {
	NotificationUUID string     `db:"notification_uuid"`
	InstallAt        *time.Time `db:"install_at"`
	ReminderSentAt   *time.Time `db:"reminder_sent_at"`
	InstallsQueuedAt *time.Time `db:"installs_queued_at"`
}

type PatchNotificationApp struct {
	PolicyID            *uint `db:"policy_id"`
	SoftwareTitleID     uint  `db:"software_title_id"`
	SoftwareInstallerID *uint `db:"software_installer_id"`
}

type PatchNotificationAppDetail struct {
	PolicyID            *uint  `db:"policy_id"`
	SoftwareTitleID     uint   `db:"software_title_id"`
	SoftwareInstallerID *uint  `db:"software_installer_id"`
	Name                string `db:"name"`
	DisplayName         string `db:"display_name"`
	HasIcon             bool   `db:"has_icon"`
}
