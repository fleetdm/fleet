//go:build unix

package apple_accounts

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixtureSchema is a trimmed copy of the Core Data schema accountsd uses for
// Accounts4.sqlite, with the account types a signed-in Mac typically has.
// Several of Apple's helper account types repeat the Apple Account's username.
const fixtureSchema = `
CREATE TABLE ZACCOUNTTYPE (Z_PK INTEGER PRIMARY KEY, Z_ENT INTEGER, Z_OPT INTEGER,
  ZACCOUNTTYPEDESCRIPTION VARCHAR, ZIDENTIFIER VARCHAR);
CREATE TABLE ZACCOUNT (Z_PK INTEGER PRIMARY KEY, Z_ENT INTEGER, Z_OPT INTEGER,
  ZACTIVE INTEGER, ZAUTHENTICATED INTEGER, ZACCOUNTTYPE INTEGER, ZPARENTACCOUNT INTEGER,
  ZACCOUNTDESCRIPTION VARCHAR, ZIDENTIFIER VARCHAR, ZUSERNAME VARCHAR);
INSERT INTO ZACCOUNTTYPE (Z_PK, ZACCOUNTTYPEDESCRIPTION, ZIDENTIFIER) VALUES
  (1, 'iCloud', 'com.apple.account.AppleAccount'),
  (2, 'Apple ID Authentication', 'com.apple.account.AppleIDAuthentication'),
  (3, 'iMessage', 'com.apple.account.IdentityServices'),
  (4, 'Google', 'com.apple.account.Google');`

const oneAppleAccount = `
INSERT INTO ZACCOUNT (Z_PK, ZACTIVE, ZAUTHENTICATED, ZACCOUNTTYPE, ZUSERNAME) VALUES
  (1, 1, 1, 1, 'someone@icloud.com');`

// The fixture databases are owned by the user running the tests, which is
// the owner readAppleAccounts expects to find.
var (
	me    = strconv.Itoa(os.Getuid())
	myGID = strconv.Itoa(os.Getgid())
)

// requireSQLite skips tests that need sqlite3 on machines without it, but not
// in CI, where skipping would silently drop the hardening tests.
func requireSQLite(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(sqlite3Path); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatalf("%s is required in CI: %v", sqlite3Path, err)
		}
		t.Skipf("%s not available: %v", sqlite3Path, err)
	}
}

func execSQLite(t *testing.T, path string, commands ...string) {
	t.Helper()
	args := append([]string{"-init", "/dev/null", "-batch", "-bail", path}, commands...)
	out, err := exec.Command(sqlite3Path, args...).CombinedOutput()
	require.NoError(t, err, "sqlite3 %s: %s", path, out)
}

// writeAccountsDB creates an accounts database under home. committed is
// checkpointed into the database file. walOnly is left in the -wal file, as
// it is while accountsd has the database open.
func writeAccountsDB(t *testing.T, home, committed, walOnly string) string {
	t.Helper()
	dir := filepath.Join(home, "Library", "Accounts")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	path := filepath.Join(dir, "Accounts4.sqlite")
	execSQLite(t, path, "PRAGMA journal_mode=WAL;", fixtureSchema, committed)
	if walOnly != "" {
		execSQLite(t, path, ".dbconfig no_ckpt_on_close on", walOnly)
		_, err := os.Stat(path + "-wal")
		require.NoError(t, err, "fixture has no -wal file, so the test would not exercise it")
	}
	return path
}

// writeRawDB creates an accounts database under home from arbitrary SQL.
func writeRawDB(t *testing.T, home, sql string) {
	t.Helper()
	dir := filepath.Join(home, "Library", "Accounts")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	execSQLite(t, filepath.Join(dir, "Accounts4.sqlite"), sql)
}

func resetUsersCache() {
	usersCache.Lock()
	usersCache.expires = time.Time{}
	usersCache.Unlock()
}

func setUsers(t *testing.T, users ...localUser) {
	t.Helper()
	resetUsersCache()
	listUsers = func(context.Context) ([]localUser, error) { return users, nil }
	t.Cleanup(func() {
		listUsers = localUsers
		resetUsersCache()
	})
}

// dirContents maps each file in dir to its contents.
func dirContents(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	contents := map[string]string{}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		require.NoError(t, err)
		contents[e.Name()] = string(data)
	}
	return contents
}

// withTimeout returns fn's error, failing the test if fn blocks instead of
// hanging the test run.
func withTimeout(t *testing.T, fn func() error) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		return err
	case <-time.After(15 * time.Second):
		t.Fatal("timed out")
		return nil
	}
}

func TestGenerate(t *testing.T) {
	requireSQLite(t)
	root := t.TempDir()

	alice := filepath.Join(root, "alice")
	alicePath := writeAccountsDB(t, alice, `
INSERT INTO ZACCOUNT (Z_PK, ZACTIVE, ZAUTHENTICATED, ZACCOUNTTYPE, ZUSERNAME) VALUES
  (1, 1, 1, 1, 'alice@icloud.com'),
  (2, 1, 1, 2, 'alice@icloud.com'),
  (3, 1, 1, 3, 'alice@icloud.com'),
  (4, 1, 1, 4, 'alice@gmail.com');`, "")

	// bob signed in while accountsd had the database open, so the account
	// exists only in the -wal file.
	bob := filepath.Join(root, "bob")
	bobPath := writeAccountsDB(t, bob, "", `
INSERT INTO ZACCOUNT (Z_PK, ZACTIVE, ZAUTHENTICATED, ZACCOUNTTYPE, ZUSERNAME) VALUES
  (1, 1, 0, 1, 'bob@example.com');`)
	bobBefore := dirContents(t, filepath.Dir(bobPath))

	// carol has never used anything backed by the Accounts framework.
	carol := filepath.Join(root, "carol")
	require.NoError(t, os.MkdirAll(carol, 0o700))

	setUsers(t,
		localUser{UID: me, GID: myGID, Username: "alice", HomeDir: alice},
		localUser{UID: me, GID: myGID, Username: "bob", HomeDir: bob},
		localUser{UID: me, GID: myGID, Username: "carol", HomeDir: carol},
	)

	rows, err := Generate(t.Context(), table.QueryContext{})
	require.NoError(t, err)
	assert.Equal(t, []map[string]string{
		{"uid": me, "username": "alice", "apple_account": "alice@icloud.com", "active": "1", "authenticated": "1", "path": alicePath},
		{"uid": me, "username": "bob", "apple_account": "bob@example.com", "active": "1", "authenticated": "0", "path": bobPath},
	}, rows)

	assert.Equal(t, bobBefore, dirContents(t, filepath.Dir(bobPath)), "querying changed the live database directory")
}

func TestGenerateColumnsMatchSchema(t *testing.T) {
	requireSQLite(t)
	home := t.TempDir()
	writeAccountsDB(t, home, oneAppleAccount, "")
	setUsers(t, localUser{UID: me, GID: myGID, Username: "someone", HomeDir: home})

	rows, err := Generate(t.Context(), table.QueryContext{})
	require.NoError(t, err)
	require.Len(t, rows, 1)

	columns := Columns()
	assert.Len(t, rows[0], len(columns))
	for _, c := range columns {
		assert.Contains(t, rows[0], c.Name)
	}
}

// A database fleetd cannot read, most often because it lacks Full Disk
// Access, must fail the query rather than look like a user who is signed out.
func TestGenerateUnreadableDatabase(t *testing.T) {
	requireSQLite(t)
	if os.Geteuid() == 0 {
		t.Skip("root can read the database regardless of its permissions")
	}
	root := t.TempDir()
	alice := filepath.Join(root, "alice")
	writeAccountsDB(t, alice, oneAppleAccount, "")
	bob := filepath.Join(root, "bob")
	require.NoError(t, os.Chmod(writeAccountsDB(t, bob, oneAppleAccount, ""), 0))

	setUsers(t,
		localUser{UID: me, GID: myGID, Username: "alice", HomeDir: alice},
		localUser{UID: me, GID: myGID, Username: "bob", HomeDir: bob},
	)

	rows, err := Generate(t.Context(), table.QueryContext{})
	require.ErrorIs(t, err, fs.ErrPermission)
	require.ErrorContains(t, err, "Full Disk Access")
	assert.Nil(t, rows)

	// A query that only asks about alice doesn't need bob's database.
	rows, err = Generate(t.Context(), table.QueryContext{Constraints: map[string]table.ConstraintList{
		"username": {Constraints: []table.Constraint{{Operator: table.OperatorEquals, Expression: "alice"}}},
	}})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "alice", rows[0]["username"])
}

// Failures that have nothing to do with the user's database must not look
// like a user without one, even though they also wrap fs.ErrNotExist.
func TestGenerateEnvironmentFailures(t *testing.T) {
	requireSQLite(t)
	home := t.TempDir()
	writeAccountsDB(t, home, oneAppleAccount, "")
	setUsers(t, localUser{UID: me, GID: myGID, Username: "someone", HomeDir: home})

	t.Run("missing sqlite3", func(t *testing.T) {
		orig := sqlite3Path
		sqlite3Path = filepath.Join(t.TempDir(), "sqlite3")
		t.Cleanup(func() { sqlite3Path = orig })

		rows, err := Generate(t.Context(), table.QueryContext{})
		require.ErrorIs(t, err, fs.ErrNotExist)
		assert.Nil(t, rows)
	})

	t.Run("missing temporary directory", func(t *testing.T) {
		t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing"))

		rows, err := Generate(t.Context(), table.QueryContext{})
		require.ErrorIs(t, err, fs.ErrNotExist)
		assert.Nil(t, rows)
	})
}

func TestCachedUsers(t *testing.T) {
	setUsers(t)
	release := make(chan struct{})
	var calls atomic.Int32
	listUsers = func(ctx context.Context) ([]localUser, error) {
		calls.Add(1)
		<-release
		// A canceled caller must not cut the shared lookup short.
		assert.NoError(t, ctx.Err())
		return []localUser{{UID: me, Username: "someone"}}, nil
	}

	// A canceled caller leaves at once, while the lookup it started carries on.
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := cachedUsers(canceled)
	require.ErrorIs(t, err, context.Canceled)
	require.Eventually(t, func() bool { return calls.Load() == 1 }, time.Second, time.Millisecond)

	// Concurrent callers share that lookup.
	results := make([][]localUser, 8)
	errs := make([]error, len(results))
	var wg sync.WaitGroup
	for i := range results {
		wg.Go(func() { results[i], errs[i] = cachedUsers(t.Context()) })
	}
	time.Sleep(50 * time.Millisecond) // let the callers join the lookup
	close(release)
	wg.Wait()
	for i := range results {
		require.NoError(t, errs[i])
		assert.Len(t, results[i], 1)
	}
	// Later callers get the cached result.
	_, err = cachedUsers(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int32(1), calls.Load())

	// A failed lookup fails the query and isn't cached.
	resetUsersCache()
	listUsers = func(context.Context) ([]localUser, error) {
		calls.Add(1)
		return nil, errors.New("dscl failed")
	}
	_, err = cachedUsers(t.Context())
	require.ErrorContains(t, err, "dscl failed")
	_, err = Generate(t.Context(), table.QueryContext{})
	require.ErrorContains(t, err, "dscl failed")
	assert.Equal(t, int32(3), calls.Load())
}

// A user lookup that never returns (a stat blocked on a stalled network home,
// say) times out instead of hanging every query, and isn't started again
// while it's stuck.
func TestCachedUsersStalledLookup(t *testing.T) {
	setUsers(t)
	origTimeout := readTimeout
	t.Cleanup(func() { readTimeout = origTimeout })
	readTimeout = 50 * time.Millisecond

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	var calls atomic.Int32
	listUsers = func(context.Context) ([]localUser, error) {
		calls.Add(1)
		<-release
		return nil, nil
	}

	for range 3 {
		_, err := Generate(t.Context(), table.QueryContext{})
		require.ErrorContains(t, err, "listing users: timed out")
	}
	assert.Equal(t, int32(1), calls.Load())
}

// A read that never returns (an open blocked on a stalled network mount, say)
// times out instead of hanging the query, and later queries for the same user
// wait on that read rather than piling up more of them.
func TestReadUserTimeout(t *testing.T) {
	origRead, origTimeout := readAccounts, readTimeout
	t.Cleanup(func() { readAccounts, readTimeout = origRead, origTimeout })
	readTimeout = 50 * time.Millisecond

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	var calls atomic.Int32
	readAccounts = func(context.Context, localUser) ([]appleAccount, error) {
		calls.Add(1)
		<-release
		return nil, nil
	}

	u := localUser{UID: me, GID: myGID, Username: "stuck", HomeDir: "/stuck"}
	for range 3 {
		_, err := readUser(t.Context(), u)
		require.ErrorContains(t, err, "timed out")
	}
	assert.Equal(t, int32(1), calls.Load())
}

func TestReadUserSharesConcurrentReads(t *testing.T) {
	origRead := readAccounts
	t.Cleanup(func() { readAccounts = origRead })

	release := make(chan struct{})
	var calls atomic.Int32
	readAccounts = func(context.Context, localUser) ([]appleAccount, error) {
		calls.Add(1)
		<-release
		return []appleAccount{{AppleAccount: "someone@icloud.com"}}, nil
	}

	u := localUser{UID: me, GID: myGID, Username: "someone", HomeDir: "/shared"}
	results := make([][]appleAccount, 4)
	errs := make([]error, len(results))
	var started, wg sync.WaitGroup
	started.Add(len(results))
	for i := range results {
		wg.Go(func() {
			started.Done()
			results[i], errs[i] = readUser(t.Context(), u)
		})
	}
	started.Wait()
	require.Eventually(t, func() bool { return calls.Load() == 1 }, time.Second, time.Millisecond)
	time.Sleep(50 * time.Millisecond) // let the other callers join the read
	close(release)
	wg.Wait()
	for i := range results {
		require.NoError(t, errs[i])
		assert.Len(t, results[i], 1)
	}
	assert.Equal(t, int32(1), calls.Load())
}

func TestScanHomes(t *testing.T) {
	dir := t.TempDir()
	elsewhere := t.TempDir()
	for _, name := range []string{"alice", ".hidden", "Shared"} {
		require.NoError(t, os.Mkdir(filepath.Join(dir, name), 0o700))
	}
	require.NoError(t, os.Symlink(elsewhere, filepath.Join(dir, "linked")))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file"), nil, 0o600))

	users, err := scanHomes(dir)
	require.NoError(t, err)
	assert.Equal(t, []localUser{
		{Username: "alice", HomeDir: filepath.Join(dir, "alice")},
		{Username: "linked", HomeDir: filepath.Join(dir, "linked")},
	}, users)

	users, err = scanHomes(filepath.Join(dir, "missing"))
	require.NoError(t, err)
	assert.Empty(t, users)

	if os.Geteuid() != 0 {
		unreadable := filepath.Join(dir, "alice")
		require.NoError(t, os.Chmod(unreadable, 0))
		t.Cleanup(func() { os.Chmod(unreadable, 0o700) }) //nolint:errcheck,gosec // best effort, so TempDir can clean up
		_, err = scanHomes(unreadable)
		require.ErrorIs(t, err, fs.ErrPermission)
	}
}

func TestResolveHomes(t *testing.T) {
	root := t.TempDir()
	alice := filepath.Join(root, "alice")
	bob := filepath.Join(root, "bob")
	require.NoError(t, os.Mkdir(alice, 0o700))
	require.NoError(t, os.Mkdir(bob, 0o700))
	require.NoError(t, os.Symlink(alice, filepath.Join(root, "alias")))
	require.NoError(t, os.WriteFile(filepath.Join(root, "file"), nil, 0o600))

	users, err := resolveHomes([]localUser{
		{UID: "501", GID: "20", Username: "alice", HomeDir: alice},
		{Username: "bob", HomeDir: bob},
		{Username: "alias", HomeDir: filepath.Join(root, "alias")},      // alice's home again
		{Username: "gone", HomeDir: filepath.Join(root, "gone")},        // no home
		{Username: "file", HomeDir: filepath.Join(root, "file")},        // not a directory
		{Username: "nested", HomeDir: filepath.Join(root, "file", "x")}, // ENOTDIR
	})
	require.NoError(t, err)
	assert.Equal(t, []localUser{
		{UID: "501", GID: "20", Username: "alice", HomeDir: alice},
		{UID: me, GID: myGID, Username: "bob", HomeDir: bob},
	}, users)

	// A home that exists but can't be checked fails the lookup rather than
	// dropping the user.
	if os.Geteuid() != 0 {
		locked := filepath.Join(root, "locked")
		require.NoError(t, os.Mkdir(locked, 0o700))
		require.NoError(t, os.Chmod(locked, 0))
		t.Cleanup(func() { os.Chmod(locked, 0o700) }) //nolint:errcheck,gosec // best effort, so TempDir can clean up
		_, err = resolveHomes([]localUser{{Username: "carol", HomeDir: filepath.Join(locked, "carol")}})
		require.ErrorIs(t, err, fs.ErrPermission)
	}
}

func TestMatchesConstraints(t *testing.T) {
	alice := localUser{UID: "501", Username: "alice"}
	eq := func(column, value string) table.QueryContext {
		return table.QueryContext{Constraints: map[string]table.ConstraintList{
			column: {Constraints: []table.Constraint{{Operator: table.OperatorEquals, Expression: value}}},
		}}
	}
	assert.True(t, matchesConstraints(alice, table.QueryContext{}))
	assert.True(t, matchesConstraints(alice, eq("uid", "501")))
	assert.False(t, matchesConstraints(alice, eq("uid", "502")))
	assert.True(t, matchesConstraints(alice, eq("username", "alice")))
	assert.True(t, matchesConstraints(alice, eq("username", "Alice")))
	assert.False(t, matchesConstraints(alice, eq("username", "bob")))
	// Anything else is left to osquery, which re-applies every constraint.
	assert.True(t, matchesConstraints(alice, eq("uid", "not a number")))
	assert.True(t, matchesConstraints(alice, table.QueryContext{Constraints: map[string]table.ConstraintList{
		"uid": {Constraints: []table.Constraint{{Operator: table.OperatorGreaterThan, Expression: "600"}}},
	}}))
}

func TestReadAppleAccountsNoDatabase(t *testing.T) {
	_, err := readAppleAccounts(t.Context(), localUser{UID: me, GID: myGID, HomeDir: t.TempDir()})
	assert.ErrorIs(t, err, errNoDatabase)
}

func TestReadAppleAccountsNoAppleAccount(t *testing.T) {
	requireSQLite(t)
	home := t.TempDir()
	writeAccountsDB(t, home, `
INSERT INTO ZACCOUNT (Z_PK, ZACTIVE, ZAUTHENTICATED, ZACCOUNTTYPE, ZUSERNAME) VALUES
  (1, 1, 1, 4, 'someone@gmail.com');`, "")

	accounts, err := readAppleAccounts(t.Context(), localUser{UID: me, GID: myGID, HomeDir: home})
	require.NoError(t, err)
	assert.Empty(t, accounts)
}

// Apple's sqlite3 leaves an empty -wal file after closing a WAL-mode database
// and can't open one read-only without it; other copies of the database (and
// other sqlite3 builds) may have none.
func TestReadAppleAccountsWithoutWALFile(t *testing.T) {
	requireSQLite(t)
	home := t.TempDir()
	path := writeAccountsDB(t, home, oneAppleAccount, "")
	require.NoError(t, os.RemoveAll(path+"-wal"))
	require.NoError(t, os.RemoveAll(path+"-shm"))

	accounts, err := readAppleAccounts(t.Context(), localUser{UID: me, GID: myGID, HomeDir: home})
	require.NoError(t, err)
	assert.Equal(t, []appleAccount{{AppleAccount: "someone@icloud.com", Active: "1", Authenticated: "1"}}, accounts)
}

func TestReadAppleAccountsRejectsLinkOutsideHome(t *testing.T) {
	requireSQLite(t)
	root := t.TempDir()
	victimPath := writeAccountsDB(t, filepath.Join(root, "victim"), oneAppleAccount, "")

	home := filepath.Join(root, "attacker")
	dir := filepath.Join(home, "Library", "Accounts")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, os.Symlink(victimPath, filepath.Join(dir, "Accounts4.sqlite")))

	accounts, err := readAppleAccounts(t.Context(), localUser{UID: me, GID: myGID, HomeDir: home})
	require.ErrorContains(t, err, "outside the home directory")
	assert.Empty(t, accounts)
}

// resolveInHome only checks the database's path, so the -wal file is guarded
// by refusing to follow a symlink there at all.
func TestReadAppleAccountsRejectsWALLink(t *testing.T) {
	requireSQLite(t)
	root := t.TempDir()
	victimPath := writeAccountsDB(t, filepath.Join(root, "victim"), "", oneAppleAccount)

	home := filepath.Join(root, "attacker")
	path := writeAccountsDB(t, home, "", "")
	require.NoError(t, os.RemoveAll(path+"-wal")) // Apple's sqlite3 leaves an empty one
	require.NoError(t, os.Symlink(victimPath+"-wal", path+"-wal"))

	_, err := readAppleAccounts(t.Context(), localUser{UID: me, GID: myGID, HomeDir: home})
	require.ErrorIs(t, err, syscall.ELOOP)
}

// The file read must belong to the user it is reported for. This is what
// catches a path swapped for a link to another user's database after
// resolveInHome checked it.
func TestReadAppleAccountsRejectsOtherOwner(t *testing.T) {
	requireSQLite(t)
	home := t.TempDir()
	writeAccountsDB(t, home, oneAppleAccount, "")

	other := strconv.Itoa(os.Getuid() + 1)
	_, err := readAppleAccounts(t.Context(), localUser{UID: other, GID: myGID, HomeDir: home})
	require.ErrorContains(t, err, "is not owned by uid "+other)
}

// A FIFO would block a plain open() forever, waiting for a writer.
func TestReadAppleAccountsRejectsFIFO(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "Library", "Accounts")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, syscall.Mkfifo(filepath.Join(dir, "Accounts4.sqlite"), 0o600))

	err := withTimeout(t, func() error {
		_, err := readAppleAccounts(t.Context(), localUser{UID: me, GID: myGID, HomeDir: home})
		return err
	})
	require.ErrorContains(t, err, "is not a regular file")
}

func TestReadAppleAccountsRejectsOversizedFile(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "Library", "Accounts")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	path := filepath.Join(dir, "Accounts4.sqlite")
	require.NoError(t, os.WriteFile(path, nil, 0o600))
	require.NoError(t, os.Truncate(path, maxFileBytes+1)) // sparse, so cheap to create

	_, err := readAppleAccounts(t.Context(), localUser{UID: me, GID: myGID, HomeDir: home})
	require.ErrorContains(t, err, "is larger than")
}

// The database belongs to the user, so anything that would make the account
// query run SQL of the user's choosing is refused before that query runs.
func TestReadAppleAccountsRejectsNonOrdinaryTables(t *testing.T) {
	requireSQLite(t)
	const accountType = `CREATE TABLE ZACCOUNTTYPE (Z_PK INTEGER PRIMARY KEY, ZIDENTIFIER VARCHAR);
INSERT INTO ZACCOUNTTYPE VALUES (1, 'com.apple.account.AppleAccount');`
	const endless = `WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x + 1 FROM n)`

	for name, sql := range map[string]string{
		"missing tables": `CREATE TABLE ZSOMETHINGELSE (Z_PK INTEGER);`,
		"endless view": accountType + `CREATE VIEW ZACCOUNT AS ` + endless + `
  SELECT x AS Z_PK, 1 AS ZACTIVE, 1 AS ZAUTHENTICATED, 1 AS ZACCOUNTTYPE, 'x' AS ZUSERNAME FROM n;`,
		"view calling writefile": accountType + `CREATE VIEW ZACCOUNT AS
  SELECT 1 AS Z_PK, 1 AS ZACTIVE, 1 AS ZAUTHENTICATED, 1 AS ZACCOUNTTYPE, writefile('/dev/null', 'x') AS ZUSERNAME;`,
		"virtual table backed by an endless view": accountType + `CREATE VIEW spin AS ` + endless + `
  SELECT x AS rowid, 1 AS ZACCOUNTTYPE, 'x' AS ZUSERNAME FROM n;
CREATE VIRTUAL TABLE ZACCOUNT USING fts5(ZACCOUNTTYPE, ZUSERNAME, content='spin');`,
		"virtual generated column": accountType + `CREATE TABLE ZACCOUNT (Z_PK INTEGER PRIMARY KEY, ZACTIVE INTEGER,
  ZAUTHENTICATED INTEGER, ZACCOUNTTYPE INTEGER, ZUSERNAME AS (hex(zeroblob(100000000))));
INSERT INTO ZACCOUNT (Z_PK, ZACCOUNTTYPE) VALUES (1, 1);`,
	} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			writeRawDB(t, home, sql)

			_, err := readAppleAccounts(t.Context(), localUser{UID: me, GID: myGID, HomeDir: home})
			require.Error(t, err)
			// Some sqlite3 builds refuse writefile() in a view while the
			// schema check reads the view's columns, before Go sees the result.
			if !strings.Contains(err.Error(), "unsafe use of writefile()") {
				require.ErrorContains(t, err, "not ordinary tables")
			}
		})
	}
}

// A crafted table can't make fleetd buffer more than maxStdoutBytes of
// sqlite3 output.
func TestReadAppleAccountsOutputLimit(t *testing.T) {
	requireSQLite(t)
	home := t.TempDir()
	value := `hex(zeroblob(` + strconv.Itoa(maxStdoutBytes/4) + `))`
	writeAccountsDB(t, home, `
INSERT INTO ZACCOUNT (Z_PK, ZACTIVE, ZAUTHENTICATED, ZACCOUNTTYPE, ZUSERNAME) VALUES
  (1, 1, 1, 1, `+value+`), (2, 1, 1, 1, `+value+`), (3, 1, 1, 1, `+value+`);`, "")

	_, err := readAppleAccounts(t.Context(), localUser{UID: me, GID: myGID, HomeDir: home})
	require.ErrorContains(t, err, "output exceeded")
}

// All-NULL rows are a few bytes each, so the output cap alone would let a
// crafted table hand osquery hundreds of thousands of rows.
func TestReadAppleAccountsTooManyRows(t *testing.T) {
	requireSQLite(t)
	home := t.TempDir()
	writeAccountsDB(t, home, `
WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x + 1 FROM n LIMIT `+strconv.Itoa(maxAccounts+1)+`)
INSERT INTO ZACCOUNT (Z_PK, ZACCOUNTTYPE) SELECT x, 1 FROM n;`, "")

	_, err := readAppleAccounts(t.Context(), localUser{UID: me, GID: myGID, HomeDir: home})
	require.ErrorContains(t, err, "more than 100 Apple Accounts")
}

// When fleetd runs as root, sqlite3 must run as the database's owner, with
// root's supplementary groups dropped, and still be able to read the copy.
func TestReadAppleAccountsDropsPrivileges(t *testing.T) {
	requireSQLite(t)
	if os.Geteuid() != 0 {
		t.Skip("needs root to switch users")
	}
	const nobody = 65534
	owner := localUser{UID: strconv.Itoa(nobody), GID: strconv.Itoa(nobody), HomeDir: t.TempDir()}
	path := writeAccountsDB(t, owner.HomeDir, oneAppleAccount, "")
	for _, p := range []string{path, path + "-wal"} {
		if err := os.Chown(p, nobody, nobody); err != nil && !errors.Is(err, fs.ErrNotExist) {
			require.NoError(t, err)
		}
	}

	accounts, err := readAppleAccounts(t.Context(), owner)
	require.NoError(t, err)
	require.Len(t, accounts, 1)

	// t.TempDir is root-only, so the user sqlite3 runs as couldn't reach a
	// script there.
	scriptDir, err := os.MkdirTemp("", "apple_accounts_test-")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(scriptDir) })
	require.NoError(t, os.Chmod(scriptDir, 0o755)) //nolint:gosec // test script directory
	ids := filepath.Join(scriptDir, "sqlite3")
	require.NoError(t, os.WriteFile(ids, []byte("#!/bin/sh\necho \"ids $(id -u) $(id -g) $(id -G)\" >&2\nexit 1\n"), 0o755)) //nolint:gosec // test script
	orig := sqlite3Path
	sqlite3Path = ids
	t.Cleanup(func() { sqlite3Path = orig })

	_, err = readAppleAccounts(t.Context(), owner)
	require.ErrorContains(t, err, "ids 65534 65534 65534")
}

func TestLimitedBuffer(t *testing.T) {
	b := &limitedBuffer{max: 5}
	n, err := b.Write([]byte("abc"))
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	n, err = b.Write([]byte("defg"))
	require.ErrorIs(t, err, errOutputLimit)
	assert.Equal(t, 2, n)
	assert.True(t, b.exceeded)
	assert.Equal(t, "abcde", b.String())

	// io.Copy is how os/exec fills it; that must go through Write too.
	b = &limitedBuffer{max: 5}
	_, err = io.Copy(b, strings.NewReader("abcdefgh"))
	require.ErrorIs(t, err, errOutputLimit)
	assert.Equal(t, "abcde", b.String())
}

func TestParseAccounts(t *testing.T) {
	assert.Nil(t, parseAccounts(nil))
	assert.Equal(t, []appleAccount{
		{AppleAccount: "odd,name@icloud.com", Active: "1", Authenticated: ""},
		{AppleAccount: "plain@icloud.com", Active: "0", Authenticated: "1"},
	}, parseAccounts([][]string{
		{"authenticated", "apple_account", "active"},
		{"", "odd,name@icloud.com", "1"},
		{"1", "plain@icloud.com", "0"},
	}))
}
