package fleet

type MDMLinuxDiskEncryptionSummary struct {
	Verified       uint `json:"verified"`
	ActionRequired uint `json:"action_required"`
	Failed         uint `json:"failed"`
}
