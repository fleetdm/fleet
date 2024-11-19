package e2e

import (
	"context"
	"fmt"
	"io"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/test"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/test/enrollment"
	"github.com/groob/plist"
)

// device is a wrapper around our enrollment for ease of use.
type device struct {
	*enrollment.Enrollment
}

func newDeviceFromCheckins(doer Doer, serverURL, authPath, tokUpdPath string) (*device, error) {
	e, err := enrollment.NewFromCheckins(doer, serverURL, "", authPath, tokUpdPath)
	if err != nil {
		return nil, err
	}
	return &device{Enrollment: e}, nil
}

func newCommand(uuid, requestType string) *mdm.Command {
	if uuid == "" && requestType == "" {
		return nil
	}
	return &mdm.Command{
		CommandUUID: uuid,
		Command: struct{ RequestType string }{
			RequestType: requestType,
		},
	}
}

func (d *device) NewCommandReport(uuid, status string, errors []mdm.ErrorChain) *mdm.CommandResults {
	return &mdm.CommandResults{
		Enrollment:  *d.GetEnrollment(),
		CommandUUID: uuid,
		Status:      status,
		ErrorChain:  errors,
	}
}

const Limit1MiB = 1024 * 1024

func (d *device) CMDDoReportAndFetch(ctx context.Context, report *mdm.CommandResults) (*mdm.Command, error) {
	reportReader, err := test.PlistReader(report)
	if err != nil {
		return nil, err
	}

	resp, err := d.DoReportAndFetch(ctx, reportReader)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, Limit1MiB))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, enrollment.NewHTTPError(resp, body)
	}

	var cmd *mdm.Command

	if len(body) > 0 {
		cmd = new(mdm.Command)
		if err = plist.Unmarshal(body, cmd); err != nil {
			return nil, fmt.Errorf("decoding command body: %w", err)
		}
	}

	return cmd, nil
}
