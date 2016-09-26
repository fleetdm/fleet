package datastore

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const bcryptCost = 6

func TestPasswordResetRequests(t *testing.T) {
	db := setup(t)
	defer teardown(t, db)

	testPasswordResetRequests(t, db)
}

func testPasswordResetRequests(t *testing.T, db kolide.Datastore) {
	createTestUsers(t, db)
	now := time.Now()
	tomorrow := now.Add(time.Hour * 24)
	var passwordResetTests = []struct {
		userID  uint
		expires time.Time
		token   string
	}{
		{userID: 1, expires: tomorrow, token: "abcd"},
	}

	for _, tt := range passwordResetTests {
		r := &kolide.PasswordResetRequest{
			UserID:    tt.userID,
			ExpiresAt: tt.expires,
			Token:     tt.token,
		}
		req, err := db.NewPasswordResetRequest(r)
		assert.Nil(t, err)
		assert.Equal(t, tt.userID, req.UserID)
	}
}

func TestEnrollHost(t *testing.T) {
	db := setup(t)
	defer teardown(t, db)

	testEnrollHost(t, db)

}

func TestAuthenticateHost(t *testing.T) {
	db := setup(t)
	defer teardown(t, db)

	testAuthenticateHost(t, db)

}

var enrollTests = []struct {
	uuid, hostname, ip, platform string
	nodeKeySize                  int
}{
	0: {uuid: "6D14C88F-8ECF-48D5-9197-777647BF6B26",
		hostname:    "web.kolide.co",
		ip:          "172.0.0.1",
		platform:    "linux",
		nodeKeySize: 12,
	},
	1: {uuid: "B998C0EB-38CE-43B1-A743-FBD7A5C9513B",
		hostname:    "mail.kolide.co",
		ip:          "172.0.0.2",
		platform:    "linux",
		nodeKeySize: 10,
	},
	2: {uuid: "008F0688-5311-4C59-86EE-00C2D6FC3EC2",
		hostname:    "home.kolide.co",
		ip:          "127.0.0.1",
		platform:    "darwin",
		nodeKeySize: 25,
	},
	3: {uuid: "uuid123",
		hostname:    "fakehostname",
		ip:          "192.168.1.1",
		platform:    "darwin",
		nodeKeySize: 1,
	},
}

func testEnrollHost(t *testing.T, db kolide.HostStore) {
	var hosts []*kolide.Host
	for _, tt := range enrollTests {
		h, err := db.EnrollHost(tt.uuid, tt.hostname, tt.ip, tt.platform, tt.nodeKeySize)
		assert.Nil(t, err)

		hosts = append(hosts, h)
		assert.Equal(t, tt.uuid, h.UUID)
		assert.Equal(t, tt.hostname, h.HostName)
		assert.Equal(t, tt.ip, h.IPAddress)
		assert.Equal(t, tt.platform, h.Platform)
		assert.NotEmpty(t, h.NodeKey)
	}

	for _, enrolled := range hosts {
		oldNodeKey := enrolled.NodeKey
		newhostname := fmt.Sprintf("changed.%s", enrolled.HostName)

		h, err := db.EnrollHost(enrolled.UUID, newhostname, enrolled.IPAddress, enrolled.Platform, 15)
		assert.Nil(t, err)
		assert.Equal(t, enrolled.UUID, h.UUID)
		assert.NotEmpty(t, h.NodeKey)
		assert.NotEqual(t, oldNodeKey, h.NodeKey)
	}

}

func testAuthenticateHost(t *testing.T, db kolide.HostStore) {
	for _, tt := range enrollTests {
		h, err := db.EnrollHost(tt.uuid, tt.hostname, tt.ip, tt.platform, tt.nodeKeySize)
		assert.Nil(t, err)

		returned, err := db.AuthenticateHost(h.NodeKey)
		assert.Nil(t, err)
		assert.Equal(t, h.NodeKey, returned.NodeKey)
	}

	_, err := db.AuthenticateHost("7B1A9DC9-B042-489F-8D5A-EEC2412C95AA")
	assert.NotNil(t, err)
}

// TestUser tests the UserStore interface
// this test uses the default testing backend
func TestCreateUser(t *testing.T) {
	db := setup(t)
	defer teardown(t, db)

	testCreateUser(t, db)
}

func TestSaveUser(t *testing.T) {
	db := setup(t)
	defer teardown(t, db)

	testSaveUser(t, db)
}

func testCreateUser(t *testing.T, db kolide.UserStore) {
	var createTests = []struct {
		username, password, email string
		isAdmin, passwordReset    bool
	}{
		{"marpaia", "foobar", "mike@kolide.co", true, false},
		{"jason", "foobar", "jason@kolide.co", true, false},
	}

	for _, tt := range createTests {
		u := &kolide.User{
			Username: tt.username,
			Password: []byte(tt.password),
			Admin:    tt.isAdmin,
			AdminForcedPasswordReset: tt.passwordReset,
			Email: tt.email,
		}
		user, err := db.NewUser(u)
		assert.Nil(t, err)

		verify, err := db.User(tt.username)
		assert.Nil(t, err)

		assert.Equal(t, user.ID, verify.ID)
		assert.Equal(t, tt.username, verify.Username)
		assert.Equal(t, tt.email, verify.Email)
		assert.Equal(t, tt.email, verify.Email)
	}
}

func TestUserByID(t *testing.T) {
	db := setup(t)
	defer teardown(t, db)

	testUserByID(t, db)
}

func testUserByID(t *testing.T, db kolide.UserStore) {
	users := createTestUsers(t, db)
	for _, tt := range users {
		returned, err := db.UserByID(tt.ID)
		assert.Nil(t, err)
		assert.Equal(t, tt.ID, returned.ID)
	}

	// test missing user
	_, err := db.UserByID(10000000000)
	assert.NotNil(t, err)
}

func createTestUsers(t *testing.T, db kolide.UserStore) []*kolide.User {
	var createTests = []struct {
		username, password, email string
		isAdmin, passwordReset    bool
	}{
		{"marpaia", "foobar", "mike@kolide.co", true, false},
		{"jason", "foobar", "jason@kolide.co", false, false},
	}

	var users []*kolide.User
	for _, tt := range createTests {
		u := &kolide.User{
			Username: tt.username,
			Password: []byte(tt.password),
			Admin:    tt.isAdmin,
			AdminForcedPasswordReset: tt.passwordReset,
			Email: tt.email,
		}

		user, err := db.NewUser(u)
		assert.Nil(t, err)

		users = append(users, user)
	}
	assert.NotEmpty(t, users)
	return users
}

func testSaveUser(t *testing.T, db kolide.UserStore) {
	users := createTestUsers(t, db)
	testAdminAttribute(t, db, users)
	testEmailAttribute(t, db, users)
	testPasswordAttribute(t, db, users)
}

func testPasswordAttribute(t *testing.T, db kolide.UserStore, users []*kolide.User) {
	for _, user := range users {
		user.Password = []byte(randomString(8))
		err := db.SaveUser(user)
		assert.Nil(t, err)

		verify, err := db.User(user.Username)
		assert.Nil(t, err)
		assert.Equal(t, user.Password, verify.Password)
	}
}

func testEmailAttribute(t *testing.T, db kolide.UserStore, users []*kolide.User) {
	for _, user := range users {
		user.Email = fmt.Sprintf("test.%s", user.Email)
		err := db.SaveUser(user)
		assert.Nil(t, err)

		verify, err := db.User(user.Username)
		assert.Nil(t, err)
		assert.Equal(t, user.Email, verify.Email)
	}
}

func testAdminAttribute(t *testing.T, db kolide.UserStore, users []*kolide.User) {
	for _, user := range users {
		user.Admin = false
		err := db.SaveUser(user)
		assert.Nil(t, err)

		verify, err := db.User(user.Username)
		assert.Nil(t, err)
		assert.Equal(t, user.Admin, verify.Admin)
	}
}

func TestLabelQueries(t *testing.T) {
	db := setup(t)
	defer teardown(t, db)

	testLabels(t, db)
}

func testLabels(t *testing.T, db kolide.Datastore) {
	hosts := []kolide.Host{}
	var host *kolide.Host
	var err error
	for i := 0; i < 10; i++ {
		host, err = db.EnrollHost(string(i), "foo", "", "", 10)
		assert.Nil(t, err, "enrollment should succeed")
		hosts = append(hosts, *host)
	}

	baseTime := time.Now()

	// No queries should be returned before labels or queries added
	queries, err := db.LabelQueriesForHost(host, baseTime)
	assert.Nil(t, err)
	assert.Empty(t, queries)

	// No labels should match
	labels, err := db.LabelsForHost(host)
	assert.Nil(t, err)
	assert.Empty(t, labels)

	labelQueries := []kolide.Query{
		kolide.Query{
			Name:     "query1",
			Query:    "query1",
			Platform: "darwin",
		},
		kolide.Query{
			Name:     "query2",
			Query:    "query2",
			Platform: "darwin",
		},
		kolide.Query{
			Name:     "query3",
			Query:    "query3",
			Platform: "darwin",
		},
		kolide.Query{
			Name:     "query4",
			Query:    "query4",
			Platform: "darwin",
		},
	}

	for _, query := range labelQueries {
		assert.Nil(t, db.NewQuery(&query))
	}

	// this one should not show up
	assert.NoError(t, db.NewQuery(&kolide.Query{
		Platform: "not_darwin",
		Query:    "query5",
	}))

	// No queries should be returned before labels added
	queries, err = db.LabelQueriesForHost(host, baseTime)
	assert.NoError(t, err)
	assert.Empty(t, queries)

	newLabels := []kolide.Label{
		// Note these are intentionally out of order
		kolide.Label{
			Name:    "label3",
			QueryID: 3,
		},
		kolide.Label{
			Name:    "label1",
			QueryID: 1,
		},
		kolide.Label{
			Name:    "label2",
			QueryID: 2,
		},
		kolide.Label{
			Name:    "label4",
			QueryID: 4,
		},
	}

	for _, label := range newLabels {
		assert.Nil(t, db.NewLabel(&label))
	}

	expectQueries := map[string]string{
		"1": "query3",
		"2": "query1",
		"3": "query2",
		"4": "query4",
	}

	host.Platform = "darwin"

	// Now queries should be returned
	queries, err = db.LabelQueriesForHost(host, baseTime)
	assert.NoError(t, err)
	assert.Equal(t, expectQueries, queries)

	// No labels should match with no results yet
	labels, err = db.LabelsForHost(host)
	assert.Nil(t, err)
	assert.Empty(t, labels)

	// Record a query execution
	err = db.RecordLabelQueryExecutions(host, map[string]bool{"1": true}, baseTime)
	assert.NoError(t, err)

	// Use a 10 minute interval, so the query we just added should show up
	queries, err = db.LabelQueriesForHost(host, time.Now().Add(-(10 * time.Minute)))
	assert.NoError(t, err)
	delete(expectQueries, "1")
	assert.Equal(t, expectQueries, queries)

	// Record an old query execution -- Shouldn't change the return
	err = db.RecordLabelQueryExecutions(host, map[string]bool{"2": true}, baseTime.Add(-1*time.Hour))
	assert.NoError(t, err)
	queries, err = db.LabelQueriesForHost(host, time.Now().Add(-(10 * time.Minute)))
	assert.NoError(t, err)
	assert.Equal(t, expectQueries, queries)

	// Record a newer execution for that query and another
	err = db.RecordLabelQueryExecutions(host, map[string]bool{"2": false, "3": true}, baseTime)
	assert.NoError(t, err)

	// Now these should no longer show up in the necessary to run queries
	delete(expectQueries, "2")
	delete(expectQueries, "3")
	queries, err = db.LabelQueriesForHost(host, time.Now().Add(-(10 * time.Minute)))
	assert.NoError(t, err)
	assert.Equal(t, expectQueries, queries)

	// Now the two matching labels should be returned
	labels, err = db.LabelsForHost(host)
	assert.Nil(t, err)
	if assert.Len(t, labels, 2) {
		assert.Equal(t, "label3", labels[0].Name)
		assert.Equal(t, "label2", labels[1].Name)
	}

	// A host that hasn't executed any label queries should still be asked
	// to execute those queries
	hosts[0].Platform = "darwin"
	queries, err = db.LabelQueriesForHost(host, time.Now())
	assert.Nil(t, err)
	assert.Len(t, queries, 4)

	// There should still be no labels returned for a host that never
	// executed any label queries
	labels, err = db.LabelsForHost(&hosts[0])
	assert.Nil(t, err)
	assert.Empty(t, labels)
}

// setup creates a datastore for testing
func setup(t *testing.T) kolide.Datastore {
	db, err := gorm.Open("sqlite3", ":memory:")
	require.Nil(t, err)

	ds := gormDB{DB: db, Driver: "sqlite3"}
	err = ds.Migrate()
	assert.Nil(t, err)
	// Log using t.Log so that output only shows up if the test fails
	//db.SetLogger(&testLogger{t: t})
	//db.LogMode(true)
	return ds
}

func teardown(t *testing.T, ds kolide.Datastore) {
	err := ds.Drop()
	assert.Nil(t, err)
}

type testLogger struct {
	t *testing.T
}

func (t *testLogger) Print(v ...interface{}) {
	t.t.Log(v...)
}

func (t *testLogger) Write(p []byte) (n int, err error) {
	t.t.Log(string(p))
	return len(p), nil
}

func randomString(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

func testSaveQuery(t *testing.T, ds kolide.Datastore) {
	query := kolide.Query{
		Name:  "foo",
		Query: "bar",
	}
	err := ds.SaveQuery(&query)
	assert.Nil(t, err)
	assert.NotEqual(t, 0, query.ID)

	query.Query = "baz"
	err = ds.SaveQuery(&query)
	assert.Nil(t, err)

	queryVerify, err := ds.Query(query.ID)
	assert.Nil(t, err)
	assert.Equal(t, "baz", queryVerify.Query)
}

func testDeleteQuery(t *testing.T, ds kolide.Datastore) {
	query := kolide.Query{
		Name:  "foo",
		Query: "bar",
	}
	err := ds.SaveQuery(&query)
	assert.Nil(t, err)
	assert.NotEqual(t, query.ID, 0)

	err = ds.DeleteQuery(&query)
	assert.Nil(t, err)

	assert.NotEqual(t, query.ID, 0)
	_, err = ds.Query(query.ID)
	assert.NotNil(t, err)
}

func testDeletePack(t *testing.T, ds kolide.Datastore) {
	pack := &kolide.Pack{
		Name: "foo",
	}
	err := ds.NewPack(pack)
	assert.Nil(t, err)
	assert.NotEqual(t, pack.ID, 0)

	pack, err = ds.Pack(pack.ID)
	assert.Nil(t, err)

	err = ds.DeletePack(pack)
	assert.Nil(t, err)

	assert.NotEqual(t, pack.ID, 0)
	pack, err = ds.Pack(pack.ID)
	assert.NotNil(t, err)
}

func testAddAndRemoveQueryFromPack(t *testing.T, ds kolide.Datastore) {
	pack := &kolide.Pack{
		Name: "foo",
	}
	err := ds.NewPack(pack)
	assert.Nil(t, err)

	q1 := &kolide.Query{
		Name:  "bar",
		Query: "bar",
	}
	err = ds.NewQuery(q1)
	assert.Nil(t, err)
	err = ds.AddQueryToPack(q1, pack)
	assert.Nil(t, err)

	q2 := &kolide.Query{
		Name:  "baz",
		Query: "baz",
	}
	err = ds.NewQuery(q2)
	assert.Nil(t, err)
	err = ds.AddQueryToPack(q2, pack)
	assert.Nil(t, err)

	queries, err := ds.GetQueriesInPack(pack)
	assert.Nil(t, err)
	assert.Len(t, queries, 2)

	err = ds.RemoveQueryFromPack(q1, pack)
	assert.Nil(t, err)

	queries, err = ds.GetQueriesInPack(pack)
	assert.Nil(t, err)
	assert.Len(t, queries, 1)
}
