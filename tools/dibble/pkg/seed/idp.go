package seed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// IDPOptions configures the IDP seeder. DSN points at a Fleet MySQL instance;
// like vulns and activities, this seeder writes directly to MySQL because the
// IDP-linkage tables (mdm_idp_accounts, host_mdm_idp_accounts, scim_users,
// host_scim_user) are normally populated by the MDM enrollment and SCIM sync
// flows rather than any public API.
type IDPOptions struct {
	DSN       string
	UserCount int // how many seeded users get an mdm_idp_accounts row
	HostCount int // how many hosts get a host_mdm_idp_accounts assignment
}

// idpUser is the subset of the GET /users response we care about.
type idpUser struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// idpHost is the subset of the GET /hosts response we care about.
type idpHost struct {
	ID       uint   `json:"id"`
	UUID     string `json:"uuid"`
	Hostname string `json:"hostname"`
}

// IDP seeds IDP linkage for the most-recently-created Fleet users so they
// surface on host detail pages. Users and hosts are fetched via the Fleet
// API (users most-recent first, so dibble-created users come up before the
// bootstrap admin); the IDP tables are written directly via the supplied
// MySQL DSN.
//
// Two layers are written per user:
//   - mdm_idp_accounts (UUID-keyed) + host_mdm_idp_accounts (host UUID →
//     account UUID). Used by the MDM enrollment flow.
//   - scim_users (numeric id) + host_scim_user (numeric host id → scim user
//     id). This is what the host details "User" card reads: GetEndUsers in
//     server/fleet/hosts.go prefers SCIM and only falls back to host_emails
//     for the username field, never populating IdpFullName from the legacy
//     mdm_idp_accounts table.
//
// Assignment is round-robin: host[i] gets seeded_user[i % len(users)] so the
// host count can exceed the user count and the extras still pick up real
// IDP records. Idempotent: existing rows matched by their unique key (email
// for mdm_idp_accounts, user_name for scim_users, host_uuid/host_id for the
// linkage tables) are reused rather than re-inserted; linkage inserts use
// INSERT IGNORE.
func IDP(ctx context.Context, c Client, log Logger, opt IDPOptions) Result {
	res := Result{Entity: "idp"}
	if opt.UserCount <= 0 {
		opt.UserCount = 3
	}
	if opt.HostCount < 0 {
		opt.HostCount = 0
	}

	users, err := fetchUsersForIDP(c, opt.UserCount)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("list users: %w", err))
		return res
	}
	if len(users) == 0 {
		res.Errors = append(res.Errors, errors.New("no users found — run `dibble users` first"))
		return res
	}

	hosts, err := fetchHostsForIDP(c, opt.HostCount)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("list hosts: %w", err))
		return res
	}

	dsn, err := mysqlDSN(opt.DSN, false)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("parse DSN: %w", err))
		return res
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("open mysql: %w", err))
		return res
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		res.Errors = append(res.Errors, fmt.Errorf("mysql ping: %w", err))
		return res
	}

	// 1. Upsert mdm_idp_accounts + scim_users for each user. The two writes
	// are paired: if either fails for a user, neither identity is retained
	// for host assignment so the round-robin in step 2 stays aligned and
	// host i's mdm_idp_account always points at the same identity as host
	// i's scim_user.
	seeded := make([]seededIDPUser, 0, len(users))
	for _, u := range users {
		accUUID, accCreated, err := upsertIDPAccount(ctx, db, u)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("upsert idp account for %s: %w", u.Email, err))
			continue
		}
		scimID, scimCreated, err := upsertSCIMUser(ctx, db, u)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("upsert scim user for %s: %w", u.Email, err))
			continue
		}
		seeded = append(seeded, seededIDPUser{accountUUID: accUUID, scimUserID: scimID})

		if accCreated {
			res.Created++
			log.Printf("idp account %s <%s>", u.Name, u.Email)
		} else {
			res.Skipped++
		}
		if scimCreated {
			res.Created++
			log.Printf("scim user %s <%s>", u.Name, u.Email)
		} else {
			res.Skipped++
		}
	}

	if len(seeded) == 0 {
		res.Errors = append(res.Errors, errors.New("no idp or scim users created or found"))
		return res
	}

	// 2. Assign hosts (round-robin) to both linkage tables using the paired
	// identities from step 1.
	if len(hosts) == 0 && opt.HostCount > 0 {
		log.Printf("idp: no hosts found — run osquery-perf to enroll some first")
	}
	for i, h := range hosts {
		pair := seeded[i%len(seeded)]

		assigned, err := assignHostToIDPAccount(ctx, db, h.UUID, pair.accountUUID)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("assign host %s to idp account: %w", h.UUID, err))
		} else if assigned {
			res.Created++
			log.Printf("idp host %s (%s) → account %s", h.Hostname, h.UUID, pair.accountUUID)
		} else {
			res.Skipped++
		}

		assigned, err = assignHostToSCIMUser(ctx, db, h.ID, pair.scimUserID)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("assign host %d to scim user: %w", h.ID, err))
			continue
		}
		if assigned {
			res.Created++
			log.Printf("scim host %s (id=%d) → scim_user_id=%d", h.Hostname, h.ID, pair.scimUserID)
		} else {
			res.Skipped++
		}
	}

	return res
}

// seededIDPUser holds the IDs both IDP tables produced for a single user, so
// host assignments to mdm_idp_accounts and scim_users stay aligned.
type seededIDPUser struct {
	accountUUID string
	scimUserID  uint
}

// fetchUsersForIDP pulls up to `limit` users from the Fleet API, sorted
// most-recently-created first so dibble-seeded users surface ahead of the
// bootstrap admin.
func fetchUsersForIDP(c Client, limit int) ([]idpUser, error) {
	var resp struct {
		Users []idpUser `json:"users"`
	}
	path := fmt.Sprintf("/api/latest/fleet/users?order_key=created_at&order_direction=desc&per_page=%d", limit)
	if err := c.Get(path, &resp); err != nil {
		return nil, err
	}
	return resp.Users, nil
}

// fetchHostsForIDP pulls up to `limit` hosts from the Fleet API. Returns an
// empty slice (not an error) when limit == 0 so callers can disable host
// assignment without a separate code path.
func fetchHostsForIDP(c Client, limit int) ([]idpHost, error) {
	if limit == 0 {
		return nil, nil
	}
	var resp struct {
		Hosts []idpHost `json:"hosts"`
	}
	path := fmt.Sprintf("/api/latest/fleet/hosts?per_page=%d", limit)
	if err := c.Get(path, &resp); err != nil {
		return nil, err
	}
	return resp.Hosts, nil
}

// upsertIDPAccount returns the account UUID for the given user's email. If a
// row already exists (matched by the unique email), its UUID is reused and
// created=false. Otherwise a new UUID is generated, inserted, and returned
// with created=true.
func upsertIDPAccount(ctx context.Context, db *sql.DB, u idpUser) (accountUUID string, created bool, err error) {
	if err := db.QueryRowContext(ctx,
		`SELECT uuid FROM mdm_idp_accounts WHERE email = ?`, u.Email,
	).Scan(&accountUUID); err == nil {
		return accountUUID, false, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return "", false, err
	}

	accountUUID = uuid.NewString()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO mdm_idp_accounts (uuid, username, fullname, email) VALUES (?, ?, ?, ?)`,
		accountUUID, u.Email, u.Name, u.Email,
	); err != nil {
		return "", false, err
	}
	return accountUUID, true, nil
}

// assignHostToIDPAccount inserts a host_mdm_idp_accounts row, returning
// assigned=true when the row was created and false when an existing
// assignment for this host was preserved (the table has a UNIQUE on
// host_uuid).
func assignHostToIDPAccount(ctx context.Context, db *sql.DB, hostUUID, accountUUID string) (assigned bool, err error) {
	r, err := db.ExecContext(ctx,
		`INSERT IGNORE INTO host_mdm_idp_accounts (host_uuid, account_uuid) VALUES (?, ?)`,
		hostUUID, accountUUID,
	)
	if err != nil {
		return false, err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// upsertSCIMUser returns the scim_users.id for the given user, matched by
// user_name (UNIQUE). user_name is set to the user's email so the seeded row
// lines up with what the SCIM sync would produce. given_name / family_name
// come from splitting the user's full name on the first space — what
// ScimUser.DisplayName() concatenates back together for the "Full name (IdP)"
// field on the host details card.
func upsertSCIMUser(ctx context.Context, db *sql.DB, u idpUser) (id uint, created bool, err error) {
	if err := db.QueryRowContext(ctx,
		`SELECT id FROM scim_users WHERE user_name = ?`, u.Email,
	).Scan(&id); err == nil {
		return id, false, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, false, err
	}

	given, family := splitFullName(u.Name)
	r, err := db.ExecContext(ctx,
		`INSERT INTO scim_users (user_name, given_name, family_name, active) VALUES (?, ?, ?, 1)`,
		u.Email, given, family,
	)
	if err != nil {
		return 0, false, err
	}
	lastID, err := r.LastInsertId()
	if err != nil {
		return 0, false, err
	}
	return uint(lastID), true, nil
}

// assignHostToSCIMUser inserts a host_scim_user row, returning assigned=true
// when the row was created. host_scim_user.host_id is the PRIMARY KEY, so an
// existing mapping for this host is preserved (INSERT IGNORE).
func assignHostToSCIMUser(ctx context.Context, db *sql.DB, hostID, scimUserID uint) (assigned bool, err error) {
	r, err := db.ExecContext(ctx,
		`INSERT IGNORE INTO host_scim_user (host_id, scim_user_id) VALUES (?, ?)`,
		hostID, scimUserID,
	)
	if err != nil {
		return false, err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// splitFullName breaks a single-string name into SCIM's given/family parts.
// Single-word names go entirely into given_name so DisplayName() still
// renders them.
func splitFullName(full string) (given, family string) {
	full = strings.TrimSpace(full)
	if i := strings.IndexByte(full, ' '); i > 0 {
		return full[:i], strings.TrimSpace(full[i+1:])
	}
	return full, ""
}
