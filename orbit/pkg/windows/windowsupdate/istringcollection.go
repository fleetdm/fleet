// based on github.com/kolide/launcher/pkg/osquery/tables
package windowsupdate

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/windows/oleconv"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// iStringCollectionToStringArrayErr takes a IDispatch to a
// stringcollection, and returns the array of strings
// https://docs.microsoft.com/en-us/windows/win32/api/wuapi/nn-wuapi-istringcollection
func iStringCollectionToStringArrayErr(disp *ole.IDispatch, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}

	if disp == nil {
		return nil, nil
	}

	count, err := oleconv.ToInt32Err(oleutil.GetProperty(disp, "Count"))
	if err != nil {
		return nil, err
	}

	stringCollection := make([]string, count)

	for i := 0; i < int(count); i++ {
		str, err := oleconv.ToStringErr(oleutil.GetProperty(disp, "Item", i))
		if err != nil {
			return nil, err
		}

		stringCollection[i] = str
	}
	return stringCollection, nil
}
