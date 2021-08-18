package service

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/logging"

	"github.com/WatchBeam/clock"
	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/v2/store/memstore"
)

func newTestService(ds fleet.Datastore, rs fleet.QueryResultStore, lq fleet.LiveQueryStore, opts ...TestServerOpts) fleet.Service {
	return newTestServiceWithConfig(ds, config.TestConfig(), rs, lq, opts...)
}

func newTestServiceWithConfig(ds fleet.Datastore, fleetConfig config.FleetConfig, rs fleet.QueryResultStore, lq fleet.LiveQueryStore, opts ...TestServerOpts) fleet.Service {
	mailer := &mockMailService{SendEmailFn: func(e fleet.Email) error { return nil }}
	license := fleet.LicenseInfo{Tier: "core"}
	writer, err := logging.NewFilesystemLogWriter(
		fleetConfig.Filesystem.StatusLogFile,
		kitlog.NewNopLogger(),
		fleetConfig.Filesystem.EnableLogRotation,
		fleetConfig.Filesystem.EnableLogCompression,
	)
	osqlogger := &logging.OsqueryLogger{Status: writer, Result: writer}
	logger := kitlog.NewNopLogger()
	if len(opts) > 0 && opts[0].Logger != nil {
		logger = opts[0].Logger
	}
	svc, err := NewService(ds, rs, logger, osqlogger, fleetConfig, mailer, clock.C, nil, lq, ds, license)
	if err != nil {
		panic(err)
	}
	return svc
}

func newTestBasicService(ds fleet.Datastore, rs fleet.QueryResultStore, lq fleet.LiveQueryStore, opts ...TestServerOpts) fleet.Service {
	mailer := &mockMailService{SendEmailFn: func(e fleet.Email) error { return nil }}
	license := fleet.LicenseInfo{Tier: fleet.TierBasic}
	testConfig := config.TestConfig()
	writer, err := logging.NewFilesystemLogWriter(
		testConfig.Filesystem.StatusLogFile,
		kitlog.NewNopLogger(),
		testConfig.Filesystem.EnableLogRotation,
		testConfig.Filesystem.EnableLogCompression,
	)
	osqlogger := &logging.OsqueryLogger{Status: writer, Result: writer}
	svc, err := NewService(ds, rs, kitlog.NewNopLogger(), osqlogger, testConfig, mailer, clock.C, nil, lq, ds, license)
	if err != nil {
		panic(err)
	}
	svc, err = eeservice.NewService(svc, ds, kitlog.NewNopLogger(), testConfig, mailer, clock.C, &license)
	if err != nil {
		panic(err)
	}
	return svc
}

func newTestServiceWithClock(ds fleet.Datastore, rs fleet.QueryResultStore, lq fleet.LiveQueryStore, c clock.Clock) fleet.Service {
	mailer := &mockMailService{SendEmailFn: func(e fleet.Email) error { return nil }}
	license := fleet.LicenseInfo{Tier: "core"}
	testConfig := config.TestConfig()
	writer, err := logging.NewFilesystemLogWriter(
		testConfig.Filesystem.StatusLogFile,
		kitlog.NewNopLogger(),
		testConfig.Filesystem.EnableLogRotation,
		testConfig.Filesystem.EnableLogCompression,
	)
	osqlogger := &logging.OsqueryLogger{Status: writer, Result: writer}
	svc, err := NewService(ds, rs, kitlog.NewNopLogger(), osqlogger, testConfig, mailer, c, nil, lq, ds, license)
	if err != nil {
		panic(err)
	}
	return svc
}

func createTestUsers(t *testing.T, ds fleet.Datastore) map[string]fleet.User {
	users := make(map[string]fleet.User)
	for _, u := range testUsers {
		role := fleet.RoleObserver
		if strings.Contains(u.Email, "admin") {
			role = fleet.RoleAdmin
		}
		user := &fleet.User{
			Name:       "Test Name " + u.Email,
			Email:      u.Email,
			GlobalRole: &role,
		}
		err := user.SetPassword(u.PlaintextPassword, 10, 10)
		require.Nil(t, err)
		user, err = ds.NewUser(user)
		require.Nil(t, err)
		users[user.Email] = *user
	}
	return users
}

var testUsers = map[string]struct {
	Email             string
	PlaintextPassword string
	GlobalRole        *string
}{
	"admin1": {
		PlaintextPassword: "foobarbaz1234!",
		Email:             "admin1@example.com",
		GlobalRole:        ptr.String(fleet.RoleAdmin),
	},
	"user1": {
		PlaintextPassword: "foobarbaz1234!",
		Email:             "user1@example.com",
		GlobalRole:        ptr.String(fleet.RoleMaintainer),
	},
	"user2": {
		PlaintextPassword: "bazfoo1234!",
		Email:             "user2@example.com",
		GlobalRole:        ptr.String(fleet.RoleObserver),
	},
}

type mockMailService struct {
	SendEmailFn func(e fleet.Email) error
	Invoked     bool
}

func (svc *mockMailService) SendEmail(e fleet.Email) error {
	svc.Invoked = true
	return svc.SendEmailFn(e)
}

type TestServerOpts struct {
	Tier   string
	Logger kitlog.Logger
}

func RunServerForTestsWithDS(t *testing.T, ds fleet.Datastore, opts ...TestServerOpts) (map[string]fleet.User, *httptest.Server) {
	newServiceFunc := newTestService
	if opts != nil && len(opts) > 0 {
		switch opts[0].Tier {
		case fleet.TierBasic:
			newServiceFunc = newTestBasicService
		}
	}
	svc := newServiceFunc(ds, nil, nil, opts...)
	users := createTestUsers(t, ds)
	logger := kitlog.NewLogfmtLogger(os.Stdout)
	if len(opts) > 0 && opts[0].Logger != nil {
		logger = opts[0].Logger
	}

	limitStore, _ := memstore.New(0)
	r := MakeHandler(svc, config.FleetConfig{}, logger, limitStore)
	server := httptest.NewServer(r)
	return users, server
}

func testKinesisPluginConfig() config.FleetConfig {
	c := config.TestConfig()
	c.Filesystem = config.FilesystemConfig{}
	c.Osquery.ResultLogPlugin = "kinesis"
	c.Osquery.StatusLogPlugin = "kinesis"
	c.Kinesis = config.KinesisConfig{
		Region:           "us-east-1",
		AccessKeyID:      "foo",
		SecretAccessKey:  "bar",
		StsAssumeRoleArn: "baz",
		StatusStream:     "test-status-stream",
		ResultStream:     "test-result-stream",
	}
	return c
}

func testFirehosePluginConfig() config.FleetConfig {
	c := config.TestConfig()
	c.Filesystem = config.FilesystemConfig{}
	c.Osquery.ResultLogPlugin = "firehose"
	c.Osquery.StatusLogPlugin = "firehose"
	c.Firehose = config.FirehoseConfig{
		Region:           "us-east-1",
		AccessKeyID:      "foo",
		SecretAccessKey:  "bar",
		StsAssumeRoleArn: "baz",
		StatusStream:     "test-status-firehose",
		ResultStream:     "test-result-firehose",
	}
	return c
}

func testLambdaPluginConfig() config.FleetConfig {
	c := config.TestConfig()
	c.Filesystem = config.FilesystemConfig{}
	c.Osquery.ResultLogPlugin = "lambda"
	c.Osquery.StatusLogPlugin = "lambda"
	c.Lambda = config.LambdaConfig{
		Region:           "us-east-1",
		AccessKeyID:      "foo",
		SecretAccessKey:  "bar",
		StsAssumeRoleArn: "baz",
		ResultFunction:   "result-func",
		StatusFunction:   "status-func",
	}
	return c
}

func testPubSubPluginConfig() config.FleetConfig {
	c := config.TestConfig()
	c.Filesystem = config.FilesystemConfig{}
	c.Osquery.ResultLogPlugin = "pubsub"
	c.Osquery.StatusLogPlugin = "pubsub"
	c.PubSub = config.PubSubConfig{
		Project:       "test",
		StatusTopic:   "status-topic",
		ResultTopic:   "result-topic",
		AddAttributes: false,
	}
	return c
}

func testStdoutPluginConfig() config.FleetConfig {
	c := config.TestConfig()
	c.Filesystem = config.FilesystemConfig{}
	c.Osquery.ResultLogPlugin = "stdout"
	c.Osquery.StatusLogPlugin = "stdout"
	return c
}
