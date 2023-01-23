package windowsupdate

import (
	"github.com/go-ole/go-ole"
)

// IImageInformation contains information about a localized image that is associated with an update or a category.
// https://docs.microsoft.com/zh-cn/windows/win32/api/wuapi/nn-wuapi-iimageinformation
type IImageInformation struct {
	disp    *ole.IDispatch
	AltText string
	Height  int64
	Source  string
	Width   int64
}

func toIImageInformation(imageInformationDisp *ole.IDispatch) (*IImageInformation, error) {
	// TODO
	return nil, nil
}
