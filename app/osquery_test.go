package app

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEnrollHost(t *testing.T) {
	db := openTestDB(t)

	expect := Host{
		UUID:      "uuid123",
		HostName:  "fakehostname",
		IPAddress: "192.168.1.1",
		Platform:  "Mac OSX",
	}

	host, err := EnrollHost(db, expect.UUID, expect.HostName, expect.IPAddress, expect.Platform)
	if err != nil {
		t.Fatal(err.Error())
	}

	if host.UUID != expect.UUID {
		t.Errorf("UUID not as expected: %s != %s", host.UUID, expect.UUID)
	}

	if host.HostName != expect.HostName {
		t.Errorf("HostName not as expected: %s != %s", host.HostName, expect.HostName)
	}

	if host.IPAddress != expect.IPAddress {
		t.Errorf("IPAddress not as expected: %s != %s", host.IPAddress, expect.IPAddress)
	}

	if host.Platform != expect.Platform {
		t.Errorf("Platform not as expected: %s != %s", host.Platform, expect.Platform)
	}

	if host.NodeKey == "" {
		t.Error("Node key was not set")
	}

}

func TestReEnrollHost(t *testing.T) {
	db := openTestDB(t)

	expect := Host{
		UUID:      "uuid123",
		HostName:  "fakehostname",
		IPAddress: "192.168.1.1",
		Platform:  "Mac OSX",
	}

	host, err := EnrollHost(db, expect.UUID, expect.HostName, expect.IPAddress, expect.Platform)
	if err != nil {
		t.Fatal(err.Error())
	}

	// Save the node key to check that it changed
	oldNodeKey := host.NodeKey

	expect.HostName = "newhostname"

	host, err = EnrollHost(db, expect.UUID, expect.HostName, "", "")
	if err != nil {
		t.Fatal(err.Error())
	}

	if host.UUID != expect.UUID {
		t.Errorf("UUID not as expected: %s != %s", host.UUID, expect.UUID)
	}

	if host.HostName != expect.HostName {
		t.Errorf("HostName not as expected: %s != %s", host.HostName, expect.HostName)
	}

	if host.IPAddress != expect.IPAddress {
		t.Errorf("IPAddress not as expected: %s != %s", host.IPAddress, expect.IPAddress)
	}

	if host.Platform != expect.Platform {
		t.Errorf("Platform not as expected: %s != %s", host.Platform, expect.Platform)
	}

	if host.NodeKey == "" {
		t.Error("Node key was not set")
	}

	if host.NodeKey == oldNodeKey {
		t.Error("Node key should have changed")
	}

}

func TestOsqueryLogWriterStatus(t *testing.T) {
	buf := bytes.Buffer{}
	logWriter := OsqueryLogWriter{Writer: &buf}
	log := OsqueryStatusLog{
		Severity: "bad",
		Filename: "nope.cpp",
		Line:     "42",
		Message:  "bad stuff happened",
		Version:  "1.8.0",
		Decorations: map[string]string{
			"foo": "bar",
		},
	}

	assert.NoError(t, logWriter.HandleStatusLog(log, "foo"))

	jsonStr, err := json.Marshal(&log)
	assert.NoError(t, err)
	assert.Equal(t, string(jsonStr)+"\n", buf.String())

}

func TestOsqueryLogWriterResult(t *testing.T) {
	buf := bytes.Buffer{}
	logWriter := OsqueryLogWriter{Writer: &buf}
	log := OsqueryResultLog{
		Name:           "query",
		HostIdentifier: "somehost",
		UnixTime:       "the time",
		CalendarTime:   "other time",
		Columns: map[string]string{
			"foo": "bar",
			"baz": "bang",
		},
		Action: "none",
	}

	assert.NoError(t, logWriter.HandleResultLog(log, "foo"))

	jsonStr, err := json.Marshal(&log)
	assert.NoError(t, err)
	assert.Equal(t, string(jsonStr)+"\n", buf.String())

}

func TestOsqueryHandlerHandleStatusLogs(t *testing.T) {
	writer := new(mockOsqueryStatusLogWriter)
	handler := OsqueryHandler{StatusHandler: writer}

	data := json.RawMessage("{")
	assert.Error(t,
		handler.handleStatusLogs(nil, &data, "foo"),
		"should error with bad json",
	)

	expect := []OsqueryStatusLog{
		OsqueryStatusLog{
			Severity: "bad",
			Filename: "nope.cpp",
			Line:     "42",
			Message:  "bad stuff happened",
			Version:  "1.8.0",
			Decorations: map[string]string{
				"foo": "bar",
			},
		},
		OsqueryStatusLog{
			Severity: "worse",
			Filename: "uhoh.cpp",
			Line:     "42",
			Message:  "bad stuff happened",
			Version:  "1.8.0",
			Decorations: map[string]string{
				"foo": "bar",
				"baz": "bang",
			},
		},
	}

	jsonVal, err := json.Marshal(&expect)
	assert.NoError(t, err)
	data = json.RawMessage(jsonVal)
	assert.NoError(t, handler.handleStatusLogs(nil, &data, "foo"))

	assert.Equal(t, expect, writer.Logs)
}

func TestOsqueryHandlerHandleResultLogs(t *testing.T) {
	writer := new(mockOsqueryResultLogWriter)
	handler := OsqueryHandler{ResultHandler: writer}

	data := json.RawMessage("{")
	assert.Error(t,
		handler.handleResultLogs(nil, &data, "foo"),
		"should error with bad json",
	)

	expect := []OsqueryResultLog{
		OsqueryResultLog{
			Name:           "query",
			HostIdentifier: "somehost",
			UnixTime:       "the time",
			CalendarTime:   "other time",
			Columns: map[string]string{
				"foo": "bar",
				"baz": "bang",
			},
			Action: "none",
		},
		OsqueryResultLog{
			Name:           "other query",
			HostIdentifier: "somehost",
			UnixTime:       "the time",
			CalendarTime:   "other time",
			Columns: map[string]string{
				"jim": "jam",
				"foo": "bar",
				"baz": "bang",
			},
			Action: "none",
		},
	}

	jsonVal, err := json.Marshal(&expect)
	assert.NoError(t, err)

	data = json.RawMessage(jsonVal)
	assert.NoError(t, handler.handleResultLogs(nil, &data, "foo"))

	assert.Equal(t, expect, writer.Logs)
}

func TestOsqueryAuthenticateRequest(t *testing.T) {
	db := openTestDB(t)

	assert.Error(t, authenticateRequest(db, "foo"), "bad node key should fail")

	host, err := EnrollHost(db, "fake_uuid", "fake_hostname", "", "")

	assert.NoError(t, err, "enroll should succeed")
	assert.NoError(t, authenticateRequest(db, host.NodeKey), "enrolled host should pass auth")

	// re-enroll so the old node key is no longer valid
	oldNodeKey := host.NodeKey
	host, err = EnrollHost(db, "fake_uuid", "fake_hostname", "", "")

	assert.NoError(t, err, "re-enroll should succeed")
	// auth should fail now
	assert.Error(t, authenticateRequest(db, oldNodeKey), "auth should succeed")
}

func TestOsqueryUpdateLastSeen(t *testing.T) {
	db := openTestDB(t)

	host, err := EnrollHost(db, "fake_uuid", "fake_hostname", "", "")
	assert.NoError(t, err)

	// Set update time to 0, then call update and make sure it is updated
	assert.NoError(t, db.Exec("UPDATE hosts SET updated_at=? WHERE node_key=?", time.Time{}, host.NodeKey).Error)

	// Clear and reload host
	host = &Host{NodeKey: host.NodeKey}
	assert.NoError(t, db.Where(&host).First(&host).Error)
	assert.True(t, host.UpdatedAt.IsZero())

	assert.NoError(t, updateLastSeen(db, host))
	assert.False(t, host.UpdatedAt.IsZero())

	// Make sure the update propagated to the DB
	host = &Host{NodeKey: host.NodeKey}
	assert.NoError(t, db.Where(&host).First(&host).Error)
	assert.False(t, host.UpdatedAt.IsZero())
}
