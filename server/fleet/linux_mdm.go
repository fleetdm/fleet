package fleet

type LinuxDiskEncryptionSummary struct {
	Verified       uint `json:"verified"`
	ActionRequired uint `json:"action_required"`
	Failed         uint `json:"failed"`
}
