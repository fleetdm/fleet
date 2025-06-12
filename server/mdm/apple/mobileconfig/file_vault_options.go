package mobileconfig

import "github.com/micromdm/plist"

type FDEFileVaultOptionsProfileContent struct {
	PayloadContent []FDEFileVaultOptionsPayload `plist:"PayloadContent"`
}
type FDEFileVaultOptionsPayload struct {
	PayloadType           string `plist:"PayloadType"`
	DestroyFVKeyOnStandby *bool  `plist:"DestroyFVKeyOnStandby"`
}

// ContainsFDEFileVaultOptionsPayload returns true if the payload contains any FileVault options.
// https://developer.apple.com/documentation/devicemanagement/fdefilevaultoptions
// Fleet users are not allowed to upload such payloads because Fleet fully manages disk encryption (FileVault).
func ContainsFDEFileVaultOptionsPayload(contents []byte) (bool, error) {
	if len(contents) == 0 {
		return false, nil
	}
	var prof FDEFileVaultOptionsProfileContent
	err := plist.Unmarshal(contents, &prof)
	if err != nil {
		return false, err
	}
	for _, p := range prof.PayloadContent {
		if p.PayloadType == FleetCustomSettingsPayloadType && p.DestroyFVKeyOnStandby != nil {
			return true, nil
		}
	}
	return false, nil
}
