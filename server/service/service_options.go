package service

import (
	"context"
	"time"

	"github.com/kolide/kolide/server/kolide"
	"github.com/pkg/errors"
)

const expectedCheckinIntervalMultiplier = 2
const minimumExpectedCheckinInterval = 10 * time.Second

func (svc service) GetOptions(ctx context.Context) ([]kolide.Option, error) {
	opts, err := svc.ds.ListOptions()
	if err != nil {
		return nil, err
	}
	return opts, nil
}

func (svc service) ModifyOptions(ctx context.Context, req kolide.OptionRequest) ([]kolide.Option, error) {
	if err := svc.ds.SaveOptions(req.Options); err != nil {
		return nil, err
	}
	return req.Options, nil
}

func (svc service) ExpectedCheckinInterval(ctx context.Context) (time.Duration, error) {
	interval := uint(0)
	found := false

	osqueryIntervalOptionNames := []string{
		"distributed_interval",
		"logger_tls_period",
	}

	for _, option := range osqueryIntervalOptionNames {
		// for each option which is known to hold a TLS check-in interval, try to
		// fetch it
		opt, err := svc.ds.OptionByName(option)
		if err != nil {
			// if the option is not set, try the next known option
			if _, ok := err.(kolide.NotFoundError); ok {
				continue
			}
			// if some other error occured when getting the option, we want to return
			// that
			return 0, err
		}

		// try to cast the option as a uint. if this fails, the option has likely been set incorrectly
		var val uint
		switch v := opt.Value.Val.(type) {
		case int:
			val = uint(v)
		case uint:
			val = v
		case uint64:
			val = uint(v)
		case float64:
			val = uint(v)
		default:
			return 0, errors.New("Option is not a number: " + opt.Name)
		}

		// If an option has not been found yet, we want to save this interval.
		// If an option HAS been found already and this one is less, we want to
		// save that as our new minimum check-in interval.
		if !found || val < interval {
			found = true
			interval = val
		}
	}

	// if we never found any interval options set, the default distributed
	// interval is 60, so we use that
	if !found {
		interval = 60
	}

	// The interval is multiplied to ensure that we are being generous in
	// the calculation if the host is a bit slower than the interval to
	// check in. This prevents flapping of the online status.
	calculatedInterval := time.Duration(interval) * time.Second * expectedCheckinIntervalMultiplier

	// We use a minimum threshold here to ensure that online status does
	// not flap when the interval is set very low.
	if calculatedInterval < minimumExpectedCheckinInterval {
		calculatedInterval = minimumExpectedCheckinInterval
	}

	// return the lowest interval that we found
	return calculatedInterval, nil
}
