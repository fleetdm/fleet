package windowsupdate

import (
	"github.com/go-ole/go-ole"
)

// IUpdateException represents info about the aspects of search results returned in the ISearchResult object that were incomplete. For more info, see Remarks.
// https://docs.microsoft.com/zh-cn/windows/win32/api/wuapi/nn-wuapi-iupdateexception
type IUpdateException struct {
	disp    *ole.IDispatch
	Context int32 // enum https://docs.microsoft.com/zh-cn/windows/win32/api/wuapi/ne-wuapi-updateexceptioncontext
	HResult int64
	Message string
}

func toIUpdateExceptions(updateExceptionsDisp *ole.IDispatch) ([]*IUpdateException, error) {
	// TODO
	return nil, nil
}
