package fleet

const PatchNotificationKind = "patch"

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
