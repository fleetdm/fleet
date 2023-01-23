package osquery

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"testing"
	"testing/quick"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/service"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/service/mock"
	"github.com/kolide/kit/testutil"
	"github.com/mixer/clock"
	"github.com/osquery/osquery-go/plugin/distributed"
	"github.com/osquery/osquery-go/plugin/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"
)

func makeTempDB(t *testing.T) (db *bbolt.DB, cleanup func()) {
	file, err := os.CreateTemp("", "kolide_launcher_test")
	if err != nil {
		t.Fatalf("creating temp file: %s", err.Error())
	}

	db, err = bbolt.Open(file.Name(), 0o600, nil)
	if err != nil {
		t.Fatalf("opening bolt DB: %s", err.Error())
	}

	return db, func() {
		db.Close()
		os.Remove(file.Name())
	}
}

func TestNewExtensionEmptyEnrollSecret(t *testing.T) {
	t.Parallel()

	e, err := NewExtension(&mock.KolideService{}, nil, ExtensionOpts{})
	assert.NotNil(t, err)
	assert.Nil(t, e)
}

func TestNewExtensionDatabaseError(t *testing.T) {
	t.Parallel()

	file, err := os.CreateTemp("", "kolide_launcher_test")
	if err != nil {
		t.Fatalf("creating temp file: %s", err.Error())
	}

	db, _ := makeTempDB(t)
	path := db.Path()
	db.Close()

	// Open read-only DB
	db, err = bbolt.Open(path, 0o600, &bbolt.Options{ReadOnly: true})
	if err != nil {
		t.Fatalf("opening bolt DB: %s", err.Error())
	}
	defer func() {
		db.Close()
		os.Remove(file.Name())
	}()

	e, err := NewExtension(&mock.KolideService{}, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	assert.NotNil(t, err)
	assert.Nil(t, e)
}

func TestGetHostIdentifier(t *testing.T) {
	t.Parallel()

	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(&mock.KolideService{}, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)

	ident, err := e.getHostIdentifier()
	require.Nil(t, err)
	assert.True(t, len(ident) > 10)
	oldIdent := ident

	ident, err = e.getHostIdentifier()
	require.Nil(t, err)
	assert.Equal(t, oldIdent, ident)

	db, cleanup = makeTempDB(t)
	defer cleanup()
	e, err = NewExtension(&mock.KolideService{}, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)

	ident, err = e.getHostIdentifier()
	require.Nil(t, err)
	// Should get different UUID with fresh DB
	assert.NotEqual(t, oldIdent, ident)
}

func TestGetHostIdentifierCorruptedData(t *testing.T) {
	t.Parallel()

	// Put bad data in the DB and ensure we can still generate a fresh UUID
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(&mock.KolideService{}, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)

	// Put garbage UUID in DB
	err = db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(configBucket))
		return b.Put([]byte(uuidKey), []byte("garbage_uuid"))
	})
	require.Nil(t, err)

	ident, err := e.getHostIdentifier()
	require.Nil(t, err)
	assert.True(t, len(ident) > 10)
	oldIdent := ident

	ident, err = e.getHostIdentifier()
	require.Nil(t, err)
	assert.Equal(t, oldIdent, ident)
}

func TestExtensionEnrollTransportError(t *testing.T) {
	t.Parallel()

	m := &mock.KolideService{
		RequestEnrollmentFunc: func(ctx context.Context, enrollSecret, hostIdentifier string, details service.EnrollmentDetails) (string, bool, error) {
			return "", false, errors.New("transport")
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)
	e.SetQuerier(mockClient{})

	key, invalid, err := e.Enroll(context.Background())
	assert.True(t, m.RequestEnrollmentFuncInvoked)
	assert.Equal(t, "", key)
	assert.True(t, invalid)
	assert.NotNil(t, err)
}

type mockClient struct{}

func (mockClient) Query(sql string) ([]map[string]string, error) {
	return []map[string]string{
		{
			"os_version":       "",
			"launcher_version": "",
			"os_build":         "",
			"platform":         "",
			"hostname":         "",
			"hardware_vendor":  "",
			"hardware_model":   "",
			"osquery_version":  "",
		},
	}, nil
}

func TestExtensionEnrollSecretInvalid(t *testing.T) {
	t.Parallel()

	m := &mock.KolideService{
		RequestEnrollmentFunc: func(ctx context.Context, enrollSecret, hostIdentifier string, details service.EnrollmentDetails) (string, bool, error) {
			return "", true, nil
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)
	e.SetQuerier(mockClient{})

	key, invalid, err := e.Enroll(context.Background())
	assert.True(t, m.RequestEnrollmentFuncInvoked)
	assert.Equal(t, "", key)
	assert.True(t, invalid)
	assert.NotNil(t, err)
}

func TestExtensionEnroll(t *testing.T) {
	t.Parallel()

	var gotEnrollSecret string
	expectedNodeKey := "node_key"
	m := &mock.KolideService{
		RequestEnrollmentFunc: func(ctx context.Context, enrollSecret, hostIdentifier string, details service.EnrollmentDetails) (string, bool, error) {
			gotEnrollSecret = enrollSecret
			return expectedNodeKey, false, nil
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	expectedEnrollSecret := "foo_secret"
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: expectedEnrollSecret})
	require.Nil(t, err)
	e.SetQuerier(mockClient{})

	key, invalid, err := e.Enroll(context.Background())
	require.Nil(t, err)
	assert.True(t, m.RequestEnrollmentFuncInvoked)
	assert.False(t, invalid)
	assert.Equal(t, expectedNodeKey, key)
	assert.Equal(t, expectedEnrollSecret, gotEnrollSecret)

	// Should not re-enroll with stored secret
	m.RequestEnrollmentFuncInvoked = false
	key, invalid, err = e.Enroll(context.Background())
	require.Nil(t, err)
	assert.False(t, m.RequestEnrollmentFuncInvoked) // Note False here.
	assert.False(t, invalid)
	assert.Equal(t, expectedNodeKey, key)
	assert.Equal(t, expectedEnrollSecret, gotEnrollSecret)

	e, err = NewExtension(m, db, ExtensionOpts{EnrollSecret: expectedEnrollSecret})
	require.Nil(t, err)
	e.SetQuerier(mockClient{})
	// Still should not re-enroll (because node key stored in DB)
	key, invalid, err = e.Enroll(context.Background())
	require.Nil(t, err)
	assert.False(t, m.RequestEnrollmentFuncInvoked) // Note False here.
	assert.False(t, invalid)
	assert.Equal(t, expectedNodeKey, key)
	assert.Equal(t, expectedEnrollSecret, gotEnrollSecret)

	// Re-enroll for new node key
	expectedNodeKey = "new_node_key"
	e.RequireReenroll(context.Background())
	assert.Empty(t, e.NodeKey)
	key, invalid, err = e.Enroll(context.Background())
	require.Nil(t, err)
	// Now enroll func should be called again
	assert.True(t, m.RequestEnrollmentFuncInvoked)
	assert.False(t, invalid)
	assert.Equal(t, expectedNodeKey, key)
	assert.Equal(t, expectedEnrollSecret, gotEnrollSecret)
}

func TestExtensionGenerateConfigsTransportError(t *testing.T) {
	t.Parallel()

	m := &mock.KolideService{
		RequestConfigFunc: func(ctx context.Context, nodeKey string) (string, bool, error) {
			return "", false, errors.New("transport")
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)

	configs, err := e.GenerateConfigs(context.Background())
	assert.True(t, m.RequestConfigFuncInvoked)
	assert.Nil(t, configs)
	// An error with the cache empty should be returned
	assert.NotNil(t, err)
}

func TestExtensionGenerateConfigsCaching(t *testing.T) {
	t.Parallel()

	configVal := `{"foo": "bar"}`
	m := &mock.KolideService{
		RequestConfigFunc: func(ctx context.Context, nodeKey string) (string, bool, error) {
			return configVal, false, nil
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)

	configs, err := e.GenerateConfigs(context.Background())
	assert.True(t, m.RequestConfigFuncInvoked)
	assert.Equal(t, map[string]string{"config": configVal}, configs)
	assert.Nil(t, err)

	// Now have requesting the config fail, and expect to get the same
	// config anyway (through the cache).
	m.RequestConfigFuncInvoked = false
	m.RequestConfigFunc = func(ctx context.Context, nodeKey string) (string, bool, error) {
		return "", false, errors.New("foobar")
	}
	configs, err = e.GenerateConfigs(context.Background())
	assert.True(t, m.RequestConfigFuncInvoked)
	assert.Equal(t, map[string]string{"config": configVal}, configs)
	// No error because config came from the cache.
	assert.Nil(t, err)
}

func TestExtensionGenerateConfigsEnrollmentInvalid(t *testing.T) {
	t.Parallel()

	expectedNodeKey := "good_node_key"
	var gotNodeKey string
	m := &mock.KolideService{
		RequestConfigFunc: func(ctx context.Context, nodeKey string) (string, bool, error) {
			gotNodeKey = nodeKey
			return "", true, nil
		},
		RequestEnrollmentFunc: func(ctx context.Context, enrollSecret, hostIdentifier string, details service.EnrollmentDetails) (string, bool, error) {
			return expectedNodeKey, false, nil
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)
	e.NodeKey = "bad_node_key"
	e.SetQuerier(mockClient{})

	configs, err := e.GenerateConfigs(context.Background())
	assert.True(t, m.RequestConfigFuncInvoked)
	assert.True(t, m.RequestEnrollmentFuncInvoked)
	assert.Nil(t, configs)
	assert.NotNil(t, err)
	assert.Equal(t, expectedNodeKey, gotNodeKey)
}

func TestExtensionGenerateConfigs(t *testing.T) {
	t.Parallel()

	configVal := `{"foo": "bar"}`
	m := &mock.KolideService{
		RequestConfigFunc: func(ctx context.Context, nodeKey string) (string, bool, error) {
			return configVal, false, nil
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)

	configs, err := e.GenerateConfigs(context.Background())
	assert.True(t, m.RequestConfigFuncInvoked)
	assert.Equal(t, map[string]string{"config": configVal}, configs)
	assert.Nil(t, err)
}

func TestExtensionWriteLogsTransportError(t *testing.T) {
	t.Parallel()

	m := &mock.KolideService{
		PublishLogsFunc: func(ctx context.Context, nodeKey string, logType logger.LogType, logs []string) (string, string, bool, error) {
			return "", "", false, errors.New("transport")
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)

	err = e.writeLogsWithReenroll(context.Background(), logger.LogTypeSnapshot, []string{"foobar"}, true)
	assert.True(t, m.PublishLogsFuncInvoked)
	assert.NotNil(t, err)
}

func TestExtensionWriteLogsEnrollmentInvalid(t *testing.T) {
	t.Parallel()

	expectedNodeKey := "good_node_key"
	var gotNodeKey string
	m := &mock.KolideService{
		PublishLogsFunc: func(ctx context.Context, nodeKey string, logType logger.LogType, logs []string) (string, string, bool, error) {
			gotNodeKey = nodeKey
			return "", "", true, nil
		},
		RequestEnrollmentFunc: func(ctx context.Context, enrollSecret, hostIdentifier string, details service.EnrollmentDetails) (string, bool, error) {
			return expectedNodeKey, false, nil
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)
	e.NodeKey = "bad_node_key"
	e.SetQuerier(mockClient{})

	err = e.writeLogsWithReenroll(context.Background(), logger.LogTypeString, []string{"foobar"}, true)
	assert.True(t, m.PublishLogsFuncInvoked)
	assert.True(t, m.RequestEnrollmentFuncInvoked)
	assert.NotNil(t, err)
	assert.Equal(t, expectedNodeKey, gotNodeKey)
}

func TestExtensionWriteLogs(t *testing.T) {
	t.Parallel()

	var gotNodeKey string
	var gotLogType logger.LogType
	var gotLogs []string
	m := &mock.KolideService{
		PublishLogsFunc: func(ctx context.Context, nodeKey string, logType logger.LogType, logs []string) (string, string, bool, error) {
			gotNodeKey = nodeKey
			gotLogType = logType
			gotLogs = logs
			return "", "", false, nil
		},
	}

	expectedNodeKey := "node_key"
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)
	e.NodeKey = expectedNodeKey

	err = e.writeLogsWithReenroll(context.Background(), logger.LogTypeStatus, []string{"foobar"}, true)
	assert.True(t, m.PublishLogsFuncInvoked)
	assert.Nil(t, err)
	assert.Equal(t, expectedNodeKey, gotNodeKey)
	assert.Equal(t, logger.LogTypeStatus, gotLogType)
	assert.Equal(t, []string{"foobar"}, gotLogs)
}

func TestKeyConversion(t *testing.T) {
	t.Parallel()

	expectedUintKeyVals := []uint64{1, 2, 64, 128, 200, 1000, 2000, 500003, 10000003, 200003005}
	byteKeys := make([][]byte, 0, len(expectedUintKeyVals))
	for _, k := range expectedUintKeyVals {
		byteKeys = append(byteKeys, byteKeyFromUint64(k))
	}

	// Assert correct sorted order of byte keys generated by key function
	require.True(t, sort.SliceIsSorted(byteKeys, func(i, j int) bool { return bytes.Compare(byteKeys[i], byteKeys[j]) <= 0 }))

	uintKeyVals := make([]uint64, 0, len(expectedUintKeyVals))
	for _, k := range byteKeys {
		uintKeyVals = append(uintKeyVals, uint64FromByteKey(k))
	}

	// Assert values are the same after roundtrip conversion
	require.Equal(t, expectedUintKeyVals, uintKeyVals)
}

func TestRandomKeyConversion(t *testing.T) {
	t.Parallel()

	// Check that roundtrips for random values result in the same key
	f := func(k uint64) bool {
		return k == uint64FromByteKey(byteKeyFromUint64(k))
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestByteKeyFromUint64(t *testing.T) {
	t.Parallel()

	// Assert correct sorted order of keys generated by key function
	keyVals := []uint64{1, 2, 64, 128, 200, 1000, 2000, 50000, 1000000, 2000000}
	keys := make([][]byte, 0, len(keyVals))
	for _, k := range keyVals {
		keys = append(keys, byteKeyFromUint64(k))
	}

	require.True(t, sort.SliceIsSorted(keyVals, func(i, j int) bool { return keyVals[i] < keyVals[j] }))
	assert.True(t, sort.SliceIsSorted(keys, func(i, j int) bool { return bytes.Compare(keys[i], keys[j]) <= 0 }))
}

func TestExtensionWriteBufferedLogsEmpty(t *testing.T) {
	t.Parallel()

	m := &mock.KolideService{
		PublishLogsFunc: func(ctx context.Context, nodeKey string, logType logger.LogType, logs []string) (string, string, bool, error) {
			t.Error("Publish logs function should not be called")
			return "", "", false, nil
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)

	// No buffered logs should result in success and no remote action being
	// taken.
	err = e.writeBufferedLogsForType(logger.LogTypeStatus)
	assert.Nil(t, err)
	assert.False(t, m.PublishLogsFuncInvoked)
}

func TestExtensionWriteBufferedLogs(t *testing.T) {
	t.Parallel()

	var gotStatusLogs, gotResultLogs []string
	m := &mock.KolideService{
		PublishLogsFunc: func(ctx context.Context, nodeKey string, logType logger.LogType, logs []string) (string, string, bool, error) {
			switch logType {
			case logger.LogTypeStatus:
				gotStatusLogs = logs
			case logger.LogTypeString:
				gotResultLogs = logs
			default:
				t.Error("Unknown log type")
			}
			return "", "", false, nil
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)

	e.LogString(context.Background(), logger.LogTypeStatus, "status foo")
	e.LogString(context.Background(), logger.LogTypeStatus, "status bar")

	e.LogString(context.Background(), logger.LogTypeString, "result foo")
	e.LogString(context.Background(), logger.LogTypeString, "result bar")

	err = e.writeBufferedLogsForType(logger.LogTypeStatus)
	assert.Nil(t, err)
	assert.True(t, m.PublishLogsFuncInvoked)
	assert.Equal(t, []string{"status foo", "status bar"}, gotStatusLogs)

	err = e.writeBufferedLogsForType(logger.LogTypeString)
	assert.Nil(t, err)
	assert.True(t, m.PublishLogsFuncInvoked)
	assert.Equal(t, []string{"result foo", "result bar"}, gotResultLogs)

	// No more logs should be written after logs flushed
	m.PublishLogsFuncInvoked = false
	gotStatusLogs = nil
	gotResultLogs = nil
	err = e.writeBufferedLogsForType(logger.LogTypeStatus)
	assert.Nil(t, err)
	err = e.writeBufferedLogsForType(logger.LogTypeString)
	assert.Nil(t, err)
	assert.False(t, m.PublishLogsFuncInvoked)
	assert.Nil(t, gotStatusLogs)
	assert.Nil(t, gotResultLogs)

	e.LogString(context.Background(), logger.LogTypeStatus, "status foo")

	err = e.writeBufferedLogsForType(logger.LogTypeStatus)
	assert.Nil(t, err)
	assert.True(t, m.PublishLogsFuncInvoked)
	assert.Equal(t, []string{"status foo"}, gotStatusLogs)
	assert.Nil(t, gotResultLogs)
}

func TestExtensionWriteBufferedLogsEnrollmentInvalid(t *testing.T) { //nolint:paralleltest
	// t.Parallel() commented out due to timeouts in github actions runner

	// Test for https://github.com/kolide/launcher/issues/219 in which a
	// call to writeBufferedLogsForType with an invalid node key causes a
	// deadlock.
	const expectedNodeKey = "good_node_key"
	var gotNodeKey string
	m := &mock.KolideService{
		PublishLogsFunc: func(ctx context.Context, nodeKey string, logType logger.LogType, logs []string) (string, string, bool, error) {
			gotNodeKey = nodeKey
			return "", "", nodeKey != expectedNodeKey, nil
		},
		RequestEnrollmentFunc: func(ctx context.Context, enrollSecret, hostIdentifier string, details service.EnrollmentDetails) (string, bool, error) {
			return expectedNodeKey, false, nil
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)
	e.SetQuerier(mockClient{})

	e.LogString(context.Background(), logger.LogTypeStatus, "status foo")
	e.LogString(context.Background(), logger.LogTypeStatus, "status bar")

	testutil.FatalAfterFunc(t, 2*time.Second, func() {
		err = e.writeBufferedLogsForType(logger.LogTypeStatus)
	})
	assert.Nil(t, err)
	assert.True(t, m.PublishLogsFuncInvoked)
	assert.True(t, m.RequestEnrollmentFuncInvoked)
	assert.Equal(t, expectedNodeKey, gotNodeKey)
}

func TestExtensionWriteBufferedLogsLimit(t *testing.T) {
	t.Parallel()

	var gotStatusLogs, gotResultLogs []string
	m := &mock.KolideService{
		PublishLogsFunc: func(ctx context.Context, nodeKey string, logType logger.LogType, logs []string) (string, string, bool, error) {
			switch logType {
			case logger.LogTypeStatus:
				gotStatusLogs = logs
			case logger.LogTypeString:
				gotResultLogs = logs
			default:
				t.Error("Unknown log type")
			}
			return "", "", false, nil
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{
		EnrollSecret:     "enroll_secret",
		MaxBytesPerBatch: 100,
	})
	require.Nil(t, err)

	expectedStatusLogs := []string{}
	expectedResultLogs := []string{}
	for i := 0; i < 20; i++ {
		status := fmt.Sprintf("status_%3d", i)
		expectedStatusLogs = append(expectedStatusLogs, status)
		e.LogString(context.Background(), logger.LogTypeStatus, status)

		result := fmt.Sprintf("result_%3d", i)
		expectedResultLogs = append(expectedResultLogs, result)
		e.LogString(context.Background(), logger.LogTypeString, result)
	}

	// Should write first 10 logs
	e.writeBufferedLogsForType(logger.LogTypeStatus)
	e.writeBufferedLogsForType(logger.LogTypeString)
	assert.True(t, m.PublishLogsFuncInvoked)
	assert.Equal(t, expectedStatusLogs[:10], gotStatusLogs)
	assert.Equal(t, expectedResultLogs[:10], gotResultLogs)

	// Should write last 10 logs
	m.PublishLogsFuncInvoked = false
	gotStatusLogs = nil
	gotResultLogs = nil
	e.writeBufferedLogsForType(logger.LogTypeStatus)
	e.writeBufferedLogsForType(logger.LogTypeString)
	assert.True(t, m.PublishLogsFuncInvoked)
	assert.Equal(t, expectedStatusLogs[10:], gotStatusLogs)
	assert.Equal(t, expectedResultLogs[10:], gotResultLogs)

	// No more logs to write
	m.PublishLogsFuncInvoked = false
	gotStatusLogs = nil
	gotResultLogs = nil
	e.writeBufferedLogsForType(logger.LogTypeStatus)
	e.writeBufferedLogsForType(logger.LogTypeString)
	assert.False(t, m.PublishLogsFuncInvoked)
	assert.Nil(t, gotStatusLogs)
	assert.Nil(t, gotResultLogs)
}

func TestExtensionWriteBufferedLogsDropsBigLog(t *testing.T) {
	t.Parallel()

	var gotStatusLogs, gotResultLogs []string
	m := &mock.KolideService{
		PublishLogsFunc: func(ctx context.Context, nodeKey string, logType logger.LogType, logs []string) (string, string, bool, error) {
			switch logType {
			case logger.LogTypeStatus:
				gotStatusLogs = logs
			case logger.LogTypeString:
				gotResultLogs = logs
			default:
				t.Error("Unknown log type")
			}
			return "", "", false, nil
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{
		EnrollSecret:     "enroll_secret",
		MaxBytesPerBatch: 15,
	})
	require.Nil(t, err)

	startLogCount, err := e.numberOfBufferedLogs(logger.LogTypeString)
	require.NoError(t, err)
	require.Equal(t, 0, startLogCount, "start with no buffered logs")

	expectedResultLogs := []string{"res1", "res2", "res3", "res4"}
	e.LogString(context.Background(), logger.LogTypeString, "this_result_is_tooooooo_big! oh noes")
	e.LogString(context.Background(), logger.LogTypeString, "res1")
	e.LogString(context.Background(), logger.LogTypeString, "res2")
	e.LogString(context.Background(), logger.LogTypeString, "this_result_is_tooooooo_big! wow")
	e.LogString(context.Background(), logger.LogTypeString, "this_result_is_tooooooo_big! scheiße")
	e.LogString(context.Background(), logger.LogTypeString, "res3")
	e.LogString(context.Background(), logger.LogTypeString, "res4")
	e.LogString(context.Background(), logger.LogTypeString, "this_result_is_tooooooo_big! darn")

	queuedLogCount, err := e.numberOfBufferedLogs(logger.LogTypeString)
	require.NoError(t, err)
	require.Equal(t, 8, queuedLogCount, "correct number of enqueued logs")

	// Should write first 3 logs
	e.writeBufferedLogsForType(logger.LogTypeString)
	assert.True(t, m.PublishLogsFuncInvoked)
	assert.Equal(t, expectedResultLogs[:3], gotResultLogs)

	// Should write last log
	m.PublishLogsFuncInvoked = false
	gotResultLogs = nil
	e.writeBufferedLogsForType(logger.LogTypeString)
	assert.True(t, m.PublishLogsFuncInvoked)
	assert.Equal(t, expectedResultLogs[3:], gotResultLogs)

	// No more logs to write
	m.PublishLogsFuncInvoked = false
	gotResultLogs = nil
	gotStatusLogs = nil
	e.writeBufferedLogsForType(logger.LogTypeString)
	assert.False(t, m.PublishLogsFuncInvoked)
	assert.Nil(t, gotResultLogs)
	assert.Nil(t, gotStatusLogs)

	finalLogCount, err := e.numberOfBufferedLogs(logger.LogTypeString)
	require.NoError(t, err)
	require.Equal(t, 0, finalLogCount, "no more queued logs")
}

func TestExtensionWriteLogsLoop(t *testing.T) { //nolint:paralleltest
	// t.Parallel() commented out due to timeouts in github actions runner

	var gotStatusLogs, gotResultLogs []string
	var funcInvokedStatus, funcInvokedResult bool
	done := make(chan struct{})
	m := &mock.KolideService{
		PublishLogsFunc: func(ctx context.Context, nodeKey string, logType logger.LogType, logs []string) (string, string, bool, error) {
			defer func() { done <- struct{}{} }()

			switch logType {
			case logger.LogTypeStatus:
				funcInvokedStatus = true
				gotStatusLogs = logs
			case logger.LogTypeString:
				funcInvokedResult = true
				gotResultLogs = logs
			default:
				t.Error("Unknown log type")
			}
			return "", "", false, nil
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	mockClock := clock.NewMockClock()
	expectedLoggingInterval := 10 * time.Second
	e, err := NewExtension(m, db, ExtensionOpts{
		EnrollSecret:     "enroll_secret",
		MaxBytesPerBatch: 200,
		Clock:            mockClock,
		LoggingInterval:  expectedLoggingInterval,
	})
	require.Nil(t, err)

	expectedStatusLogs := []string{}
	expectedResultLogs := []string{}
	for i := 0; i < 20; i++ {
		status := fmt.Sprintf("status_%013d", i)
		expectedStatusLogs = append(expectedStatusLogs, status)
		e.LogString(context.Background(), logger.LogTypeStatus, status)

		result := fmt.Sprintf("result_%013d", i)
		expectedResultLogs = append(expectedResultLogs, result)
		e.LogString(context.Background(), logger.LogTypeString, result)
	}

	// Should write first 10 logs
	e.Start()
	testutil.FatalAfterFunc(t, 1*time.Second, func() {
		// PublishLogsFunc runs twice for each run of the loop
		<-done
		<-done
	})
	assert.True(t, funcInvokedStatus)
	assert.True(t, funcInvokedResult)
	assert.Nil(t, err)
	assert.Equal(t, expectedStatusLogs[:10], gotStatusLogs)
	assert.Equal(t, expectedResultLogs[:10], gotResultLogs)

	funcInvokedStatus = false
	funcInvokedResult = false
	gotStatusLogs = nil
	gotResultLogs = nil

	// Should write last 10 logs
	mockClock.AddTime(expectedLoggingInterval + 1)
	testutil.FatalAfterFunc(t, 1*time.Second, func() {
		// PublishLogsFunc runs twice of each run of the loop
		<-done
		<-done
	})
	assert.True(t, funcInvokedStatus)
	assert.True(t, funcInvokedResult)
	assert.Nil(t, err)
	assert.Equal(t, expectedStatusLogs[10:], gotStatusLogs)
	assert.Equal(t, expectedResultLogs[10:], gotResultLogs)

	funcInvokedStatus = false
	funcInvokedResult = false
	gotStatusLogs = nil
	gotResultLogs = nil

	// No more logs to write
	mockClock.AddTime(expectedLoggingInterval + 1)
	// Block to ensure publish function could be called if the logic is
	// incorrect
	time.Sleep(1 * time.Millisecond)
	assert.False(t, funcInvokedStatus)
	assert.False(t, funcInvokedResult)
	assert.Nil(t, err)
	assert.Nil(t, gotStatusLogs)
	assert.Nil(t, gotResultLogs)

	testutil.FatalAfterFunc(t, 3*time.Second, func() {
		e.Shutdown()
	})
}

func TestExtensionPurgeBufferedLogs(t *testing.T) {
	t.Parallel()

	var gotStatusLogs, gotResultLogs []string
	m := &mock.KolideService{
		PublishLogsFunc: func(ctx context.Context, nodeKey string, logType logger.LogType, logs []string) (string, string, bool, error) {
			switch logType {
			case logger.LogTypeStatus:
				gotStatusLogs = logs
			case logger.LogTypeString:
				gotResultLogs = logs
			default:
				t.Error("Unknown log type")
			}
			// Mock as if sending logs errored
			return "", "", false, errors.New("server rejected logs")
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	max := 10
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret", MaxBufferedLogs: max})
	require.Nil(t, err)

	var expectedStatusLogs, expectedResultLogs []string
	for i := 0; i < 100; i++ {
		gotStatusLogs = nil
		gotResultLogs = nil
		statusLog := fmt.Sprintf("status %d", i)
		expectedStatusLogs = append(expectedStatusLogs, statusLog)
		e.LogString(context.Background(), logger.LogTypeStatus, statusLog)

		resultLog := fmt.Sprintf("result %d", i)
		expectedResultLogs = append(expectedResultLogs, resultLog)
		e.LogString(context.Background(), logger.LogTypeString, resultLog)

		e.writeAndPurgeLogs()

		if i < max {
			assert.Equal(t, expectedStatusLogs, gotStatusLogs)
			assert.Equal(t, expectedResultLogs, gotResultLogs)
		} else {
			assert.Equal(t, expectedStatusLogs[i-max:], gotStatusLogs)
			assert.Equal(t, expectedResultLogs[i-max:], gotResultLogs)
		}
	}
}

func TestExtensionGetQueriesTransportError(t *testing.T) {
	t.Parallel()

	m := &mock.KolideService{
		RequestQueriesFunc: func(ctx context.Context, nodeKey string) (*distributed.GetQueriesResult, bool, error) {
			return nil, false, errors.New("transport")
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)

	queries, err := e.GetQueries(context.Background())
	assert.True(t, m.RequestQueriesFuncInvoked)
	assert.NotNil(t, err)
	assert.Nil(t, queries)
}

func TestExtensionGetQueriesEnrollmentInvalid(t *testing.T) {
	t.Parallel()

	expectedNodeKey := "good_node_key"
	var gotNodeKey string
	m := &mock.KolideService{
		RequestQueriesFunc: func(ctx context.Context, nodeKey string) (*distributed.GetQueriesResult, bool, error) {
			gotNodeKey = nodeKey
			return nil, true, nil
		},
		RequestEnrollmentFunc: func(ctx context.Context, enrollSecret, hostIdentifier string, details service.EnrollmentDetails) (string, bool, error) {
			return expectedNodeKey, false, nil
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)
	e.NodeKey = "bad_node_key"
	e.SetQuerier(mockClient{})

	queries, err := e.GetQueries(context.Background())
	assert.True(t, m.RequestQueriesFuncInvoked)
	assert.True(t, m.RequestEnrollmentFuncInvoked)
	assert.NotNil(t, err)
	assert.Nil(t, queries)
	assert.Equal(t, expectedNodeKey, gotNodeKey)
}

func TestExtensionGetQueries(t *testing.T) {
	t.Parallel()

	expectedQueries := map[string]string{
		"time":    "select * from time",
		"version": "select version from osquery_info",
	}
	m := &mock.KolideService{
		RequestQueriesFunc: func(ctx context.Context, nodeKey string) (*distributed.GetQueriesResult, bool, error) {
			return &distributed.GetQueriesResult{
				Queries: expectedQueries,
			}, false, nil
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)

	queries, err := e.GetQueries(context.Background())
	assert.True(t, m.RequestQueriesFuncInvoked)
	require.Nil(t, err)
	assert.Equal(t, expectedQueries, queries.Queries)
}

func TestExtensionWriteResultsTransportError(t *testing.T) {
	t.Parallel()

	m := &mock.KolideService{
		PublishResultsFunc: func(ctx context.Context, nodeKey string, results []distributed.Result) (string, string, bool, error) {
			return "", "", false, errors.New("transport")
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)

	err = e.WriteResults(context.Background(), []distributed.Result{})
	assert.True(t, m.PublishResultsFuncInvoked)
	assert.NotNil(t, err)
}

func TestExtensionWriteResultsEnrollmentInvalid(t *testing.T) {
	t.Parallel()

	expectedNodeKey := "good_node_key"
	var gotNodeKey string
	m := &mock.KolideService{
		PublishResultsFunc: func(ctx context.Context, nodeKey string, results []distributed.Result) (string, string, bool, error) {
			gotNodeKey = nodeKey
			return "", "", true, nil
		},
		RequestEnrollmentFunc: func(ctx context.Context, enrollSecret, hostIdentifier string, details service.EnrollmentDetails) (string, bool, error) {
			return expectedNodeKey, false, nil
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)
	e.NodeKey = "bad_node_key"
	e.SetQuerier(mockClient{})

	err = e.WriteResults(context.Background(), []distributed.Result{})
	assert.True(t, m.PublishResultsFuncInvoked)
	assert.True(t, m.RequestEnrollmentFuncInvoked)
	assert.NotNil(t, err)
	assert.Equal(t, expectedNodeKey, gotNodeKey)
}

func TestExtensionWriteResults(t *testing.T) {
	t.Parallel()

	var gotResults []distributed.Result
	m := &mock.KolideService{
		PublishResultsFunc: func(ctx context.Context, nodeKey string, results []distributed.Result) (string, string, bool, error) {
			gotResults = results
			return "", "", false, nil
		},
	}
	db, cleanup := makeTempDB(t)
	defer cleanup()
	e, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.Nil(t, err)

	expectedResults := []distributed.Result{
		{
			QueryName: "foobar",
			Status:    0,
			Rows:      []map[string]string{{"foo": "bar"}},
		},
	}

	err = e.WriteResults(context.Background(), expectedResults)
	assert.True(t, m.PublishResultsFuncInvoked)
	assert.Nil(t, err)
	assert.Equal(t, expectedResults, gotResults)
}

func TestLauncherKeys(t *testing.T) {
	t.Parallel()

	m := &mock.KolideService{}

	db, cleanup := makeTempDB(t)
	defer cleanup()
	_, err := NewExtension(m, db, ExtensionOpts{EnrollSecret: "enroll_secret"})
	require.NoError(t, err)

	key, err := PrivateKeyFromDB(db)
	require.NoError(t, err)

	pubkeyPem, fingerprintStored, err := PublicKeyFromDB(db)
	require.NoError(t, err)

	fingerprint, err := rsaFingerprint(key)
	require.NoError(t, err)
	require.Equal(t, fingerprint, fingerprintStored)

	pubkey, err := KeyFromPem([]byte(pubkeyPem))
	require.NoError(t, err)

	require.Equal(t, &key.PublicKey, pubkey)
}
