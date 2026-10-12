//go:build unix

// Package apple_accounts implements the apple_accounts table, which reports the
// Apple Account (Apple ID / iCloud) each local user is signed in to on macOS.
package apple_accounts

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/singleflight"
)

// accountsDB is where accountsd keeps a user's Internet Accounts, relative to
// the home directory. Since macOS 27, ~/Library/Preferences/MobileMeAccounts.plist
// is an empty dictionary, so this Core Data store is the only place the
// signed-in Apple Account can still be read from.
const accountsDB = "Library/Accounts/Accounts4.sqlite"

// schemaCheckQuery returns 1 only if ZACCOUNT and ZACCOUNTTYPE are ordinary
// tables without virtual generated columns. The database is user-controlled,
// and a view, a virtual table, or a generated column would make
// appleAccountQuery run a query of the user's choosing. This reads only schema
// metadata, so it can't be subverted the same way. It runs as its own sqlite3
// process because SQLite still scans a virtual table when a guard in the same
// statement is false.
const schemaCheckQuery = `SELECT (SELECT count(*) FROM sqlite_schema WHERE type = 'table'
    AND name IN ('ZACCOUNT', 'ZACCOUNTTYPE') AND sql LIKE 'CREATE TABLE %') = 2
  AND NOT EXISTS (SELECT 1 FROM pragma_table_xinfo('ZACCOUNT') WHERE hidden <> 0)
  AND NOT EXISTS (SELECT 1 FROM pragma_table_xinfo('ZACCOUNTTYPE') WHERE hidden <> 0) AS ordinary;`

// appleAccountQuery selects the Apple Account rows. Other account types live
// in the same table: Google, Exchange, and Apple's own helper accounts
// (com.apple.account.AppleIDAuthentication, IdentityServices, ...), which
// repeat the Apple Account's username and would produce duplicate rows. It has
// no ORDER BY so that sqlite3 never sorts (or spills to temp files); the plan
// is a scan of ZACCOUNT in rowid order with a primary key lookup. The LIMIT
// lets Generate detect a crafted table with more rows than maxAccounts.
var appleAccountQuery = fmt.Sprintf(`SELECT a.ZUSERNAME AS apple_account, a.ZACTIVE AS active, a.ZAUTHENTICATED AS authenticated
FROM ZACCOUNT a JOIN ZACCOUNTTYPE t ON a.ZACCOUNTTYPE = t.Z_PK
WHERE t.ZIDENTIFIER = 'com.apple.account.AppleAccount'
LIMIT %d;`, maxAccounts+1)

const (
	queryTimeout     = 5 * time.Second
	snapshotAttempts = 3

	// A user normally has one Apple Account. The limits below bound what a
	// crafted database can make fleetd copy, buffer, and hand to osquery:
	// all-NULL rows are a few bytes each, so the output cap alone would still
	// allow hundreds of thousands of them.
	maxAccounts    = 100
	maxFileBytes   = 64 << 20
	maxStdoutBytes = 64 << 10
	maxStderrBytes = 4 << 10

	// osquery calls Generate once per joined row, back to back, for a query
	// that joins on this table. Reusing the user list that briefly lets those
	// calls share one directory lookup.
	usersCacheTTL = 5 * time.Second
)

// errNoDatabase means the user has no accounts database. It is kept distinct
// from fs.ErrNotExist so that a missing sqlite3 or temporary directory fails
// the query instead of looking like a user who isn't signed in.
var errNoDatabase = errors.New("no accounts database")

var (
	// Shelling out to the sqlite3 that ships with macOS avoids cgo and a
	// bundled SQLite. These are swapped out in tests.
	sqlite3Path  = "/usr/bin/sqlite3"
	listUsers    = localUsers
	readAccounts = readAppleAccounts

	// readTimeout bounds one user's read, including file operations that
	// can't be canceled, such as an open that blocks on a stalled network
	// mount or on downloading a cloud file. Swapped out in tests.
	readTimeout = 15 * time.Second

	// reads lets concurrent queries share one read per user, and keeps a read
	// stuck past readTimeout from being started again while it's still stuck.
	reads singleflight.Group

	usersCache struct {
		sync.Mutex
		users   []localUser
		expires time.Time
	}
)

// localUser is an account whose home directory may hold an accounts database.
type localUser struct {
	UID      string
	GID      string
	Username string
	HomeDir  string
}

// appleAccount is one Apple Account row from a user's accounts database.
type appleAccount struct {
	AppleAccount  string
	Active        string
	Authenticated string
}

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.BigIntColumn("uid"),
		table.TextColumn("username"),
		table.TextColumn("apple_account"),
		table.IntegerColumn("active"),
		table.IntegerColumn("authenticated"),
		table.TextColumn("path"),
	}
}

// Generate is called to return the results for the table at query time.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	users, err := cachedUsers(ctx)
	if err != nil {
		return nil, err
	}
	results := []map[string]string{}
	for _, u := range users {
		if !matchesConstraints(u, queryContext) {
			continue
		}
		path := filepath.Join(u.HomeDir, accountsDB)
		accounts, err := readUser(ctx, u)
		if errors.Is(err, errNoDatabase) {
			// The user has never used anything backed by the Accounts
			// framework, so there is nobody signed in.
			continue
		}
		if err != nil {
			// Failing the query, rather than skipping the user, keeps a
			// database fleetd cannot read from looking like a user who is not
			// signed in, which a policy would count as a pass.
			if errors.Is(err, fs.ErrPermission) {
				return nil, fmt.Errorf("reading %s: %w (fleetd needs Full Disk Access)", path, err)
			}
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		log.Debug().Str("path", path).Int("accounts", len(accounts)).Msg("apple_accounts: read accounts database")
		for _, a := range accounts {
			results = append(results, map[string]string{
				"uid":           u.UID,
				"username":      u.Username,
				"apple_account": a.AppleAccount,
				"active":        a.Active,
				"authenticated": a.Authenticated,
				"path":          path,
			})
		}
	}
	return results, nil
}

// cachedUsers returns listUsers' result, reusing it for usersCacheTTL.
// Concurrent callers wait for a single lookup, which is detached from the
// caller's cancellation so one canceled query can't fail it for the others.
func cachedUsers(ctx context.Context) ([]localUser, error) {
	usersCache.Lock()
	defer usersCache.Unlock()
	if time.Now().After(usersCache.expires) {
		users, err := listUsers(context.WithoutCancel(ctx))
		if err != nil {
			return nil, fmt.Errorf("listing users: %w", err)
		}
		usersCache.users = users
		usersCache.expires = time.Now().Add(usersCacheTTL)
	}
	return usersCache.users, nil
}

// matchesConstraints reports whether u can satisfy the query's equality
// constraints on uid and username. osquery calls Generate once per joined row
// (for example `users JOIN apple_accounts USING (uid)`) with that row's uid,
// and re-applies every constraint to the rows returned, so this only saves
// reading databases whose rows osquery would discard. Usernames are compared
// case-insensitively because osquery passes `= 'x' COLLATE NOCASE` as a plain
// equality.
func matchesConstraints(u localUser, queryContext table.QueryContext) bool {
	for _, c := range queryContext.Constraints["uid"].Constraints {
		if c.Operator != table.OperatorEquals {
			continue
		}
		if want, err := strconv.ParseInt(c.Expression, 10, 64); err == nil && strconv.FormatInt(want, 10) != u.UID {
			return false
		}
	}
	for _, c := range queryContext.Constraints["username"].Constraints {
		if c.Operator == table.OperatorEquals && !strings.EqualFold(c.Expression, u.Username) {
			return false
		}
	}
	return true
}

// readUser runs readAccounts for u under readTimeout.
func readUser(ctx context.Context, u localUser) ([]appleAccount, error) {
	ch := reads.DoChan(u.UID+"\x00"+u.HomeDir, func() (any, error) {
		return readAccounts(context.WithoutCancel(ctx), u)
	})
	timer := time.NewTimer(readTimeout)
	defer timer.Stop()
	select {
	case r := <-ch:
		accounts, _ := r.Val.([]appleAccount)
		return accounts, r.Err
	case <-timer.C:
		return nil, fmt.Errorf("timed out after %s", readTimeout)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// readAppleAccounts returns the Apple Accounts recorded in the accounts
// database under u's home directory, or errNoDatabase if there is none.
func readAppleAccounts(ctx context.Context, u localUser) ([]appleAccount, error) {
	path, err := resolveInHome(u.HomeDir, filepath.Join(u.HomeDir, accountsDB))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, errNoDatabase
	}
	if err != nil {
		return nil, err
	}
	uid, err := strconv.ParseUint(u.UID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid uid %q: %w", u.UID, err)
	}
	gid, err := strconv.ParseUint(u.GID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid gid %q: %w", u.GID, err)
	}
	owner := &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}

	snapshot, cleanup, err := snapshotDB(path, owner)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	records, err := runSQLite(ctx, snapshot, schemaCheckQuery, owner)
	if err != nil {
		return nil, err
	}
	if !slices.EqualFunc(records, [][]string{{"ordinary"}, {"1"}}, slices.Equal) {
		return nil, errors.New("ZACCOUNT and ZACCOUNTTYPE are not ordinary tables")
	}

	records, err = runSQLite(ctx, snapshot, appleAccountQuery, owner)
	if err != nil {
		return nil, err
	}
	accounts := parseAccounts(records)
	if len(accounts) > maxAccounts {
		return nil, fmt.Errorf("more than %d Apple Accounts", maxAccounts)
	}
	return accounts, nil
}

// resolveInHome resolves symlinks in path and checks the result is still
// inside home. fleetd runs as root, so without this a user could link their
// database to another user's and be reported with that user's Apple Account.
func resolveInHome(home, path string) (string, error) {
	realHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		return "", err
	}
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if rel, err := filepath.Rel(realHome, realPath); err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("%s resolves to %s, outside the home directory", path, realPath)
	}
	return realPath, nil
}

// snapshotDB copies the database and its write-ahead log into a new private
// directory and returns the copy's path.
//
// accountsd keeps the database open in WAL mode, so a recent change such as a
// sign-in can sit in the -wal file until the next checkpoint. Querying a copy
// sees that change without taking locks on the live database or creating
// files next to it. The database and log are copied separately, so the copy
// is retried if either changed while it was being made.
//
// The copy is then handed to owner, whom sqlite3 runs as: the inner directory
// and its files become theirs, while the outer directory stays root's and only
// lets them traverse it, so they can't rename or replace the inner one.
func snapshotDB(path string, owner *syscall.Credential) (string, func(), error) {
	outer, err := os.MkdirTemp("", "apple_accounts-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(outer) }
	fail := func(err error) (string, func(), error) {
		cleanup()
		return "", nil, err
	}
	inner := filepath.Join(outer, "db")
	if err := os.Mkdir(inner, 0o700); err != nil {
		return fail(err)
	}
	dst := filepath.Join(inner, filepath.Base(path))

	for attempt := 1; ; attempt++ {
		before := fileStates(path, path+"-wal")
		copied, err := copyOwnedFile(path, dst, owner.Uid)
		if err != nil {
			return fail(err)
		}
		if !copied {
			return fail(errNoDatabase)
		}
		copied, err = copyOwnedFile(path+"-wal", dst+"-wal", owner.Uid)
		if err != nil {
			return fail(err)
		}
		if !copied {
			// Apple's sqlite3 can't open a WAL-mode database read-only without
			// a -wal file next to it. An empty one means the same as none.
			if err := os.WriteFile(dst+"-wal", nil, 0o600); err != nil {
				return fail(err)
			}
		}
		if fileStates(path, path+"-wal") == before {
			break
		}
		if attempt == snapshotAttempts {
			return fail(fmt.Errorf("%s kept changing while it was being copied", path))
		}
	}

	for _, p := range []string{dst, dst + "-wal", inner} {
		if err := os.Lchown(p, int(owner.Uid), int(owner.Gid)); err != nil {
			return fail(err)
		}
	}
	if err := os.Chmod(outer, 0o711); err != nil { //nolint:gosec // traverse-only, so sqlite3 can reach the inner directory
		return fail(err)
	}
	return dst, cleanup, nil
}

// fileStates summarizes the identity, size, and modification time of each
// path, for detecting a write between two points in time. A missing file is
// recorded as such, so a -wal file appearing or disappearing also counts.
func fileStates(paths ...string) string {
	var b strings.Builder
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			b.WriteString("-;")
			continue
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			b.WriteString("-;")
			continue
		}
		fmt.Fprintf(&b, "%d/%d/%d/%d;", st.Dev, st.Ino, info.Size(), info.ModTime().UnixNano())
	}
	return b.String()
}

// copyOwnedFile copies src to dst if src is a regular file owned by uid and
// no larger than maxFileBytes. It returns false, and no error, if src doesn't
// exist.
//
// The checks run on the opened file rather than the path, so a path component
// swapped for a symlink after resolveInHome (or a -wal symlink, which it never
// sees) leads to a file the user doesn't own and is rejected. noFollowFlags
// refuses to follow symlinks while opening, and O_NONBLOCK keeps a FIFO from
// blocking the open.
func copyOwnedFile(src, dst string, uid uint32) (bool, error) {
	in, err := os.OpenFile(src, os.O_RDONLY|noFollowFlags|syscall.O_NONBLOCK, 0)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("%s is not a regular file", src)
	}
	if st, ok := info.Sys().(*syscall.Stat_t); !ok || st.Uid != uid {
		return false, fmt.Errorf("%s is not owned by uid %d", src, uid)
	}
	if info.Size() > maxFileBytes {
		return false, fmt.Errorf("%s is larger than %d bytes", src, maxFileBytes)
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return false, err
	}
	n, err := io.Copy(out, io.LimitReader(in, maxFileBytes+1))
	if err == nil && n > maxFileBytes {
		err = fmt.Errorf("%s is larger than %d bytes", src, maxFileBytes)
	}
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	return err == nil, err
}

// runSQLite runs query against the database at path and returns sqlite3's CSV
// output as records, starting with the header row when there are any rows.
// When fleetd runs as root, sqlite3 runs as owner, so a flaw in SQLite's
// handling of a crafted file gains the user nothing they don't already have.
func runSQLite(ctx context.Context, path, query string, owner *syscall.Credential) ([][]string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// -safe and -readonly keep even a query the user controls from writing
	// files, attaching databases, or loading extensions. -init /dev/null skips
	// ~/.sqliterc, which could change the output mode.
	cmd := exec.CommandContext(ctx, sqlite3Path,
		"-init", "/dev/null", "-safe", "-readonly", "-batch", "-bail", "-csv", "-header",
		path, query)
	if os.Geteuid() == 0 {
		cmd.SysProcAttr = &syscall.SysProcAttr{Credential: owner}
	}
	stdout := &limitedBuffer{max: maxStdoutBytes}
	stderr := &limitedBuffer{max: maxStderrBytes}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	switch {
	case stdout.exceeded:
		return nil, fmt.Errorf("sqlite3 output exceeded %d bytes", maxStdoutBytes)
	case ctx.Err() != nil:
		return nil, fmt.Errorf("sqlite3: %w", ctx.Err())
	case err != nil:
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("sqlite3: %w: %s", err, msg)
		}
		return nil, fmt.Errorf("sqlite3: %w", err)
	}

	records, err := csv.NewReader(bytes.NewReader(stdout.Bytes())).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing sqlite3 output: %w", err)
	}
	return records, nil
}

var errOutputLimit = errors.New("output limit exceeded")

// limitedBuffer keeps the first max bytes written to it. A write past that
// fails, which closes the pipe and stops sqlite3 at its next write instead of
// letting it stream output until the timeout. The bytes.Buffer is a named
// field, not embedded, so io.Copy can't bypass Write through its ReadFrom.
type limitedBuffer struct {
	buf      bytes.Buffer
	max      int
	exceeded bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if room := b.max - b.buf.Len(); len(p) > room {
		b.exceeded = true
		b.buf.Write(p[:room])
		return room, errOutputLimit
	}
	return b.buf.Write(p)
}

func (b *limitedBuffer) Bytes() []byte  { return b.buf.Bytes() }
func (b *limitedBuffer) String() string { return b.buf.String() }

// parseAccounts maps appleAccountQuery's records, which start with a header
// row unless the query matched nothing, to accounts.
func parseAccounts(records [][]string) []appleAccount {
	if len(records) == 0 {
		return nil
	}

	column := map[string]int{}
	for i, name := range records[0] {
		column[name] = i
	}
	field := func(record []string, name string) string {
		if i, ok := column[name]; ok && i < len(record) {
			return record[i]
		}
		return ""
	}

	accounts := make([]appleAccount, 0, len(records)-1)
	for _, record := range records[1:] {
		accounts = append(accounts, appleAccount{
			AppleAccount:  field(record, "apple_account"),
			Active:        field(record, "active"),
			Authenticated: field(record, "authenticated"),
		})
	}
	return accounts
}
