package seed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// IDPOptions configures the IDP seeder. DSN points at a Fleet MySQL instance;
// like vulns and activities, this seeder writes directly to MySQL because
// mdm_idp_accounts and host_mdm_idp_accounts are populated by the MDM
// enrollment flow rather than any public API.
type IDPOptions struct {
	DSN        string
	UserCount  int // how many seeded users get an mdm_idp_accounts row
	HostCount  int // how many hosts get a host_mdm_idp_accounts assignment
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

// IDP creates mdm_idp_accounts rows for the most-recently-created seeded
// users and assigns them to existing hosts. Users are fetched via the Fleet
// API (most-recent first, so dibble-created users come up before the bootstrap
// admin), hosts are fetched via the Fleet API, and the IDP tables are written
// directly via the supplied MySQL DSN.
//
// Assignment is round-robin: host[i] gets account[i % len(accounts)] so the
// host count can exceed the account count and the extras still pick up a
// real IDP record.
//
// Idempotent: an account that already exists (matched by email) is reused
// rather than re-inserted; host assignments use INSERT IGNORE.
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

	// 1. Upsert mdm_idp_accounts for each user, collecting the resulting UUIDs.
	accountUUIDs := make([]string, 0, len(users))
	for _, u := range users {
		accUUID, created, err := upsertIDPAccount(ctx, db, u)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("upsert idp account for %s: %w", u.Email, err))
			continue
		}
		accountUUIDs = append(accountUUIDs, accUUID)
		if created {
			res.Created++
			log.Printf("idp account %s <%s>", u.Name, u.Email)
		} else {
			res.Skipped++
		}
	}

	if len(accountUUIDs) == 0 {
		res.Errors = append(res.Errors, errors.New("no idp accounts created or found"))
		return res
	}

	// 2. Assign hosts (round-robin) to the accounts.
	if len(hosts) == 0 && opt.HostCount > 0 {
		log.Printf("idp: no hosts found — run osquery-perf to enroll some first")
	}
	for i, h := range hosts {
		accUUID := accountUUIDs[i%len(accountUUIDs)]
		assigned, err := assignHostToIDPAccount(ctx, db, h.UUID, accUUID)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Errorf("assign host %s: %w", h.UUID, err))
			continue
		}
		if assigned {
			res.Created++
			log.Printf("idp host %s (%s) → %s", h.Hostname, h.UUID, accUUID)
		} else {
			res.Skipped++
		}
	}

	return res
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
