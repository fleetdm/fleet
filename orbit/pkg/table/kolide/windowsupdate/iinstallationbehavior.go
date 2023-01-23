package windowsupdate

import (
	"github.com/go-ole/go-ole"
)

// IInstallationBehavior represents the installation and uninstallation options of an update.
// https://docs.microsoft.com/zh-cn/windows/win32/api/wuapi/nn-wuapi-iinstallationbehavior
type IInstallationBehavior struct {
	disp                        *ole.IDispatch
	CanRequestUserInput         bool
	Impact                      int32 // enum https://docs.microsoft.com/zh-cn/windows/win32/api/wuapi/ne-wuapi-installationimpact
	RebootBehavior              int32 // enum https://docs.microsoft.com/zh-cn/windows/win32/api/wuapi/ne-wuapi-installationrebootbehavior
	RequiresNetworkConnectivity bool
}

func toIInstallationBehavior(installationBehaviorDisp *ole.IDispatch) (*IInstallationBehavior, error) {
	// TODO
	return nil, nil
}
