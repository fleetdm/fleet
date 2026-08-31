package fleet

// PatchNotificationKind is the kind name a notify-before-patching notification
// is stored under, which is how the notifications bounded context knows to hand
// it to the patch kind.
const PatchNotificationKind = "patch"

// PatchNotificationApp is one app covered by a patch notification, as written
// when a policy install skips because the app was open.
type PatchNotificationApp struct {
	PolicyID            *uint `db:"policy_id"`
	SoftwareTitleID     uint  `db:"software_title_id"`
	SoftwareInstallerID *uint `db:"software_installer_id"`
}

// PatchNotificationAppDetail is a covered app with what it takes to draw it in
// the toast and to queue its install.
type PatchNotificationAppDetail struct {
	PolicyID            *uint  `db:"policy_id"`
	SoftwareTitleID     uint   `db:"software_title_id"`
	SoftwareInstallerID *uint  `db:"software_installer_id"`
	Name                string `db:"name"`
	DisplayName         string `db:"display_name"`
	HasIcon             bool   `db:"has_icon"`
}
