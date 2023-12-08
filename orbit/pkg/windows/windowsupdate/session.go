// based on github.com/kolide/launcher/pkg/osquery/tables
package windowsupdate

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/orbit/pkg/windows/oleconv"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// IUpdateSession represents a session in which the caller can perform
// operations that involve updates.  For example, this interface
// represents sessions in which the caller performs a search,
// download, installation, or uninstallation operation.
// https://docs.microsoft.com/en-us/windows/win32/api/wuapi/nn-wuapi-iupdatesession
type IUpdateSession struct {
	disp                *ole.IDispatch
	ClientApplicationID string
	ReadOnly            bool
}

// NewUpdateSession creates a new Microsoft.Update.Session object
func NewUpdateSession() (*IUpdateSession, error) {
	unknown, err := oleutil.CreateObject("Microsoft.Update.Session")
	if err != nil {
		return nil, fmt.Errorf("create Microsoft.Update.Session: %w", err)
	}
	disp, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, fmt.Errorf("IID_IDispatch: %w", err)
	}
	return toIUpdateSession(disp)
}

func toIUpdateSession(updateSessionDisp *ole.IDispatch) (*IUpdateSession, error) {
	var err error

	iUpdateSession := &IUpdateSession{
		disp: updateSessionDisp,
	}

	if iUpdateSession.ClientApplicationID, err = oleconv.ToStringErr(oleutil.GetProperty(updateSessionDisp, "ClientApplicationID")); err != nil {
		return nil, fmt.Errorf("ClientApplicationID: %w", err)
	}

	if iUpdateSession.ReadOnly, err = oleconv.ToBoolErr(oleutil.GetProperty(updateSessionDisp, "ReadOnly")); err != nil {
		return nil, fmt.Errorf("ReadOnly: %w", err)
	}

	return iUpdateSession, nil
}

func (iUpdateSession *IUpdateSession) GetLocal() (uint32, error) {
	return oleconv.ToUint32Err(oleutil.GetProperty(iUpdateSession.disp, "UserLocale"))
}

func (iUpdateSession *IUpdateSession) SetLocal(locale uint32) error {
	if _, err := oleconv.ToUint32Err(oleutil.PutProperty(iUpdateSession.disp, "UserLocale", locale)); err != nil {
		return fmt.Errorf("putproperty userlocale: %w", err)
	}
	return nil
}

// CreateUpdateSearcher returns an IUpdateSearcher interface for this session.
// https://docs.microsoft.com/zh-cn/windows/win32/api/wuapi/nf-wuapi-iupdatesession-createupdatesearcher
func (iUpdateSession *IUpdateSession) CreateUpdateSearcher() (*IUpdateSearcher, error) {
	updateSearcherDisp, err := oleconv.ToIDispatchErr(oleutil.CallMethod(iUpdateSession.disp, "CreateUpdateSearcher"))
	if err != nil {
		return nil, err
	}

	return toIUpdateSearcher(updateSearcherDisp)
}
