package schedule

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	kitlog "github.com/go-kit/kit/log"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/log"

	"github.com/stretchr/testify/require"
)

func TestCronVulnerabilitiesCreatesDatabasesPath(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			HostSettings: fleet.HostSettings{EnableSoftwareInventory: true},
		}, nil
	}
	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		return true, nil
	}
	ds.UnlockFunc = func(ctx context.Context, name string, owner string) error {
		return nil
	}

	// because the path should be created before the bulk of vuln processing begins, we can use a call to any ds methods
	// below to signal that we are ready to make our test assertion without waiting for processing to finish
	dsSoftwareFnCalled := make(chan bool)
	ds.AllSoftwareWithoutCPEIteratorFunc = func(ctx context.Context) (fleet.SoftwareIterator, error) {
		dsSoftwareFnCalled <- true
		return nil, fmt.Errorf("forced error for test purposes")
	}
	ds.CalculateHostsPerSoftwareFunc = func(ctx context.Context, time time.Time) error {
		dsSoftwareFnCalled <- true
		return fmt.Errorf("forced error for test purposes")
	}

	vulnPath := path.Join(t.TempDir(), "something")
	require.NoDirExists(t, vulnPath)

	fleetConfig := config.FleetConfig{
		Vulnerabilities: config.VulnerabilitiesConfig{
			DatabasesPath:         vulnPath,
			Periodicity:           1 * time.Second,
			CurrentInstanceChecks: "auto",
			DisableDataSync:       true,
		},
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	logger := log.NewNopLogger()
	vulnerabilities, err := New(ctx, "vulnerabilities", "test_instance", fleetConfig.Vulnerabilities.Periodicity, ds, logger)
	require.NoError(t, err)

	vulnerabilities.SetPreflightCheck(func() bool { return fleetConfig.Vulnerabilities.CurrentInstanceChecks == "auto" })
	vulnerabilities.AddJob("cron_vulnerabilities", func(ctx context.Context) (interface{}, error) {
		return DoVulnProcessing(ctx, ds, logger, fleetConfig)
	}, func(interface{}, error) {})

	failCheck := time.After(5 * time.Second)

TEST:
	for {
		select {
		case <-dsSoftwareFnCalled:
			require.DirExists(t, vulnPath)
			break TEST

		case <-failCheck:
			require.DirExists(t, vulnPath)
			break TEST
		}
	}
}

// TODO: fix races
func TestCronVulnerabilitiesAcceptsExistingDbPath(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			HostSettings: fleet.HostSettings{EnableSoftwareInventory: true},
		}, nil
	}
	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		return true, nil
	}
	ds.UnlockFunc = func(ctx context.Context, name string, owner string) error {
		return nil
	}

	// because the path should be created before the bulk of vuln processing begins, we can use a call to any ds methods
	// below to signal that we are ready to make our test assertion without waiting for processing to finish
	dsSoftwareFnCalled := make(chan bool)
	ds.AllSoftwareWithoutCPEIteratorFunc = func(ctx context.Context) (fleet.SoftwareIterator, error) {
		dsSoftwareFnCalled <- true
		return nil, fmt.Errorf("forced error for test purposes")
	}
	ds.CalculateHostsPerSoftwareFunc = func(ctx context.Context, time time.Time) error {
		dsSoftwareFnCalled <- true
		return fmt.Errorf("forced error for test purposes")
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	buf := new(bytes.Buffer)
	logger := kitlog.NewJSONLogger(buf)
	logger = level.NewFilter(logger, level.AllowDebug())
	dbPath := t.TempDir()
	fleetConfig := config.FleetConfig{
		Vulnerabilities: config.VulnerabilitiesConfig{
			DatabasesPath:         dbPath,
			Periodicity:           1 * time.Second,
			CurrentInstanceChecks: "auto",
			DisableDataSync:       true,
		},
	}

	vulnerabilities, err := New(ctx, "vulnerabilities", "test_instance", fleetConfig.Vulnerabilities.Periodicity, ds, logger)
	require.NoError(t, err)

	vulnerabilities.SetPreflightCheck(func() bool { return fleetConfig.Vulnerabilities.CurrentInstanceChecks == "auto" })
	vulnerabilities.SetConfigCheck(func(time.Time, time.Duration) (*time.Duration, error) {
		return &fleetConfig.Vulnerabilities.Periodicity, nil
	})
	vulnerabilities.AddJob("cron_vulnerabilities", func(ctx context.Context) (interface{}, error) {
		return DoVulnProcessing(ctx, ds, logger, fleetConfig)
	}, func(interface{}, error) {})

	failCheck := time.After(5 * time.Second)

TEST:
	for {
		select {
		case <-dsSoftwareFnCalled:
			require.Contains(t, buf.String(), "checking for recent vulnerabilities")
			require.Contains(t, buf.String(), fmt.Sprintf(`"vuln-path":"%s"`, dbPath))
			break TEST

		case <-failCheck:
			require.Contains(t, buf.String(), "checking for recent vulnerabilities")
			require.Contains(t, buf.String(), fmt.Sprintf(`"vuln-path":"%s"`, dbPath))
			break TEST
		}
	}
}

// TODO: fix races
func TestCronVulnerabilitiesQuitsIfErrorVulnPath(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			HostSettings: fleet.HostSettings{EnableSoftwareInventory: true},
		}, nil
	}
	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		return true, nil
	}
	ds.UnlockFunc = func(ctx context.Context, name string, owner string) error {
		return nil
	}

	// because the logic we care about should be created before the bulk of vuln processing begins,
	// we can use a call to any ds methods below to signal a failed test
	dsSoftwareFnCalled := make(chan bool)
	ds.AllSoftwareWithoutCPEIteratorFunc = func(ctx context.Context) (fleet.SoftwareIterator, error) {
		dsSoftwareFnCalled <- true
		return nil, fmt.Errorf("forced error for test purposes")
	}
	ds.CalculateHostsPerSoftwareFunc = func(ctx context.Context, time time.Time) error {
		dsSoftwareFnCalled <- true
		return fmt.Errorf("forced error for test purposes")
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	buf := new(bytes.Buffer)
	logger := kitlog.NewJSONLogger(buf)
	logger = level.NewFilter(logger, level.AllowDebug())

	fileVulnPath := path.Join(t.TempDir(), "somefile")
	_, err := os.Create(fileVulnPath)
	require.NoError(t, err)

	fleetConfig := config.FleetConfig{
		Vulnerabilities: config.VulnerabilitiesConfig{
			DatabasesPath:         fileVulnPath,
			Periodicity:           1 * time.Second,
			CurrentInstanceChecks: "auto",
			DisableDataSync:       true, // TODO: do we need for this test?
		},
	}

	vulnerabilities, err := New(ctx, "vulnerabilities", "test_instance", fleetConfig.Vulnerabilities.Periodicity, ds, logger)
	require.NoError(t, err)

	vulnerabilities.SetPreflightCheck(func() bool { return fleetConfig.Vulnerabilities.CurrentInstanceChecks == "auto" })
	vulnerabilities.SetConfigCheck(func(time.Time, time.Duration) (*time.Duration, error) {
		return &fleetConfig.Vulnerabilities.Periodicity, nil
	})
	vulnerabilities.AddJob("cron_vulnerabilities", func(ctx context.Context) (interface{}, error) {
		return DoVulnProcessing(ctx, ds, logger, fleetConfig)
	}, func(interface{}, error) {})

	failCheck := time.After(5 * time.Second)

TEST:
	for {
		select {
		case <-dsSoftwareFnCalled:
			t.FailNow() // TODO: review this test with Tomas
		case <-failCheck:
			require.Contains(t, buf.String(), `"databases-path":"creation failed, returning"`)
			break TEST
		}
	}
}

// TODO: fix races
func TestCronVulnerabilitiesSkipCreationIfStatic(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			HostSettings: fleet.HostSettings{EnableSoftwareInventory: true},
		}, nil
	}
	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		return true, nil
	}
	ds.UnlockFunc = func(ctx context.Context, name string, owner string) error {
		return nil
	}

	// because the logic we care about should be created before the bulk of vuln processing begins,
	// we can use a call to any ds methods below to signal a failed test
	dsSoftwareFnCalled := make(chan bool)
	ds.AllSoftwareWithoutCPEIteratorFunc = func(ctx context.Context) (fleet.SoftwareIterator, error) {
		dsSoftwareFnCalled <- true
		return nil, fmt.Errorf("forced error for test purposes")
	}
	ds.CalculateHostsPerSoftwareFunc = func(ctx context.Context, time time.Time) error {
		dsSoftwareFnCalled <- true
		return fmt.Errorf("forced error for test purposes")
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	buf := new(bytes.Buffer)
	logger := kitlog.NewJSONLogger(buf)
	logger = level.NewFilter(logger, level.AllowDebug())

	vulnPath := path.Join(t.TempDir(), "something")
	require.NoDirExists(t, vulnPath)

	fleetConfig := config.FleetConfig{
		Vulnerabilities: config.VulnerabilitiesConfig{
			DatabasesPath:         vulnPath,
			Periodicity:           1 * time.Second,
			CurrentInstanceChecks: "1",
			DisableDataSync:       true, // TODO: do we need for this test?

		},
	}

	vulnerabilities, err := New(ctx, "vulnerabilities", "test_instance", fleetConfig.Vulnerabilities.Periodicity, ds, logger)
	require.NoError(t, err)

	vulnerabilities.SetPreflightCheck(func() bool { return fleetConfig.Vulnerabilities.CurrentInstanceChecks == "auto" })
	vulnerabilities.SetConfigCheck(func(time.Time, time.Duration) (*time.Duration, error) {
		return &fleetConfig.Vulnerabilities.Periodicity, nil
	})
	vulnerabilities.AddJob("cron_vulnerabilities", func(ctx context.Context) (interface{}, error) {
		return DoVulnProcessing(ctx, ds, logger, fleetConfig)
	}, func(interface{}, error) {})

	failCheck := time.After(5 * time.Second)

TEST:
	for {
		select {
		case <-dsSoftwareFnCalled:
			t.FailNow() // TODO: review this test with Tomas
		case <-failCheck:
			require.NoDirExists(t, vulnPath)
			break TEST
		}
	}
}
