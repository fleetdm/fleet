package mobileconfig

import "github.com/micromdm/plist"

type FDEFileVaultOptionsProfileContent struct {
	PayloadContent []FDEFileVaultOptionsPayload `plist:"PayloadContent"`
}
type FDEFileVaultOptionsPayload struct {
	PayloadType           string `plist:"PayloadType"`
	DestroyFVKeyOnStandby *bool  `plist:"DestroyFVKeyOnStandby"`
	DontAllowFDEDisable   *bool  `plist:"dontAllowFDEDisable"`
	DontAllowFDEEnable    *bool  `plist:"dontAllowFDEEnable"`
}

func ContainsFDEVileVaultOptionsPayload(contents []byte) (bool, error) {
	if len(contents) == 0 {
		return false, nil
	}
	var prof FDEFileVaultOptionsProfileContent
	err := plist.Unmarshal(contents, &prof)
	if err != nil {
		return false, err
	}
	for _, p := range prof.PayloadContent {
		if p.PayloadType == FleetCustomSettingsPayloadType && (p.DontAllowFDEDisable != nil || p.DontAllowFDEEnable != nil || p.DestroyFVKeyOnStandby != nil) {
			return true, nil
		}
	}
	return false, nil
}
