// based on github.com/kolide/launcher/pkg/osquery/tables
package windowsupdate

import (
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/windows/oleconv"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// IUpdateHistoryEntry represents the recorded history of an update.
// https://docs.microsoft.com/en-us/windows/win32/api/wuapi/nn-wuapi-iupdatehistoryentry
type IUpdateHistoryEntry struct {
	disp                *ole.IDispatch
	ClientApplicationID string
	Date                *time.Time
	Description         string
	HResult             int32
	Operation           int32 // enum https://docs.microsoft.com/en-us/windows/win32/api/wuapi/ne-wuapi-updateoperation
	ResultCode          int32 // enum https://docs.microsoft.com/en-us/windows/win32/api/wuapi/ne-wuapi-operationresultcode
	ServerSelection     int32 // enum
	ServiceID           string
	SupportUrl          string
	Title               string
	UninstallationNotes string
	UninstallationSteps []string
	UnmappedResultCode  int32
	UpdateIdentity      *IUpdateIdentity
}

func toIUpdateHistoryEntries(updateHistoryEntriesDisp *ole.IDispatch) ([]*IUpdateHistoryEntry, error) {
	count, err := oleconv.ToInt32Err(oleutil.GetProperty(updateHistoryEntriesDisp, "Count"))
	if err != nil {
		return nil, fmt.Errorf("Count: %w", err)
	}

	updateHistoryEntries := make([]*IUpdateHistoryEntry, count)
	for i := 0; i < int(count); i++ {
		updateHistoryEntryDisp, err := oleconv.ToIDispatchErr(oleutil.GetProperty(updateHistoryEntriesDisp, "Item", i))
		if err != nil {
			return nil, fmt.Errorf("item %d: %w", i, err)
		}

		updateHistoryEntry, err := toIUpdateHistoryEntry(updateHistoryEntryDisp)
		if err != nil {
			return nil, fmt.Errorf("toIUpdateHistoryEntry: %w", err)
		}

		updateHistoryEntries[i] = updateHistoryEntry
	}
	return updateHistoryEntries, nil
}

func toIUpdateHistoryEntry(updateHistoryEntryDisp *ole.IDispatch) (*IUpdateHistoryEntry, error) {
	var err error
	iUpdateHistoryEntry := &IUpdateHistoryEntry{
		disp: updateHistoryEntryDisp,
	}

	if iUpdateHistoryEntry.ClientApplicationID, err = oleconv.ToStringErr(oleutil.GetProperty(updateHistoryEntryDisp, "ClientApplicationID")); err != nil {
		return nil, fmt.Errorf("ClientApplicationID: %w", err)
	}

	if iUpdateHistoryEntry.Date, err = oleconv.ToTimeErr(oleutil.GetProperty(updateHistoryEntryDisp, "Date")); err != nil {
		return nil, fmt.Errorf("Date: %w", err)
	}

	if iUpdateHistoryEntry.Description, err = oleconv.ToStringErr(oleutil.GetProperty(updateHistoryEntryDisp, "Description")); err != nil {
		return nil, fmt.Errorf("Description: %w", err)
	}

	if iUpdateHistoryEntry.HResult, err = oleconv.ToInt32Err(oleutil.GetProperty(updateHistoryEntryDisp, "HResult")); err != nil {
		return nil, fmt.Errorf("HResult: %w", err)
	}

	if iUpdateHistoryEntry.Operation, err = oleconv.ToInt32Err(oleutil.GetProperty(updateHistoryEntryDisp, "Operation")); err != nil {
		return nil, fmt.Errorf("Operation: %w", err)
	}

	if iUpdateHistoryEntry.ResultCode, err = oleconv.ToInt32Err(oleutil.GetProperty(updateHistoryEntryDisp, "ResultCode")); err != nil {
		return nil, fmt.Errorf("ResultCode: %w", err)
	}

	if iUpdateHistoryEntry.ServerSelection, err = oleconv.ToInt32Err(oleutil.GetProperty(updateHistoryEntryDisp, "ServerSelection")); err != nil {
		return nil, fmt.Errorf("ServerSelection: %w", err)
	}

	if iUpdateHistoryEntry.ServiceID, err = oleconv.ToStringErr(oleutil.GetProperty(updateHistoryEntryDisp, "ServiceID")); err != nil {
		return nil, fmt.Errorf("ServiceID: %w", err)
	}

	if iUpdateHistoryEntry.SupportUrl, err = oleconv.ToStringErr(oleutil.GetProperty(updateHistoryEntryDisp, "SupportUrl")); err != nil {
		return nil, fmt.Errorf("SupportUrl: %w", err)
	}

	if iUpdateHistoryEntry.Title, err = oleconv.ToStringErr(oleutil.GetProperty(updateHistoryEntryDisp, "Title")); err != nil {
		return nil, fmt.Errorf("Title: %w", err)
	}

	if iUpdateHistoryEntry.UninstallationNotes, err = oleconv.ToStringErr(oleutil.GetProperty(updateHistoryEntryDisp, "UninstallationNotes")); err != nil {
		return nil, fmt.Errorf("UninstallationNotes: %w", err)
	}

	if iUpdateHistoryEntry.UninstallationSteps, err = oleconv.ToStringSliceErr(oleutil.GetProperty(updateHistoryEntryDisp, "UninstallationSteps")); err != nil {
		return nil, fmt.Errorf("UninstallationSteps: %w", err)
	}

	if iUpdateHistoryEntry.UnmappedResultCode, err = oleconv.ToInt32Err(oleutil.GetProperty(updateHistoryEntryDisp, "UnmappedResultCode")); err != nil {
		return nil, fmt.Errorf("UnmappedResultCode: %w", err)
	}

	updateIdentityDisp, err := oleconv.ToIDispatchErr(oleutil.GetProperty(updateHistoryEntryDisp, "UpdateIdentity"))
	if err != nil {
		return nil, fmt.Errorf("UpdateIdentity: %w", err)
	}
	if updateIdentityDisp != nil {
		if iUpdateHistoryEntry.UpdateIdentity, err = toIUpdateIdentity(updateIdentityDisp); err != nil {
			return nil, fmt.Errorf("toIUpdateIdentity: %w", err)
		}
	}

	return iUpdateHistoryEntry, nil
}
