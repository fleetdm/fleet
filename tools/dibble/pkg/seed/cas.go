package seed

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// CAOptions configures the CA seeder. Like the activity seeder it writes
// directly to MySQL (bypassing the service layer) so we don't have to spin
// up real SCEP / DigiCert / NDES / EST endpoints just to get past URL
// validation.
//
// Encrypted secret columns (api_token_encrypted, password_encrypted,
// challenge_encrypted, client_secret_encrypted) are intentionally left
// NULL. The rows show up in the UI list so reviewers can see the CA shape,
// but any actual request_certificate call will fail — which is the point.
// All user-facing names are prefixed with "*" and tagged with the run id
// so seeded rows are obvious and don't collide across runs.
type CAOptions struct {
	DSN   string
	Count int
}

// caFakePrefix marks every CA name dibble writes so it's obvious in the UI
// that the CA was planted by dibble and isn't expected to actually issue
// certificates. The "*" character isn't allowed by the service-layer name
// validator (^\w+$), which is fine — we bypass the service layer and the
// DB column is just a VARCHAR.
const caFakePrefix = "*"

// NDES is special: Fleet hardcodes the name to "NDES" and the (type, name)
// unique index allows only one row. We can't prefix it; we just insert it
// once per run and let duplicate-key errors become skips on re-runs.
const ndesName = "NDES"

// CAs writes a fresh batch of fake certificate authorities to MySQL. Each
// "batch" writes one row per non-NDES CA type (custom_scep_proxy,
// custom_est_proxy, digicert, hydrant, smallstep) plus a single NDES row
// at the start of the run. Count controls the number of batches.
func CAs(ctx context.Context, log Logger, opt CAOptions) Result {
	res := Result{Entity: "cas"}
	if opt.DSN == "" {
		res.Errors = append(res.Errors, errors.New("cas: empty DSN"))
		return res
	}
	if opt.Count <= 0 {
		opt.Count = 1
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

	runTag := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000)

	// NDES first — single insert, skipped if it already exists.
	switch err := insertCA(ctx, db, ndesRow()); {
	case err == nil:
		res.Created++
		log.Printf("ca: inserted ndes_scep_proxy/%s", ndesName)
	case isDuplicateCAError(err):
		res.Skipped++
		log.Printf("ca: skip ndes_scep_proxy/%s (already exists)", ndesName)
	default:
		res.Errors = append(res.Errors, fmt.Errorf("insert NDES: %w", err))
	}

	for b := 0; b < opt.Count; b++ {
		for _, build := range caTemplates {
			row := build(runTag, b+1)
			switch err := insertCA(ctx, db, row); {
			case err == nil:
				res.Created++
				log.Printf("ca: inserted %s/%s", row.Type, derefStr(row.Name))
			case isDuplicateCAError(err):
				res.Skipped++
				log.Printf("ca: skip %s/%s (already exists)", row.Type, derefStr(row.Name))
			default:
				res.Errors = append(res.Errors, fmt.Errorf("insert %s/%s: %w", row.Type, derefStr(row.Name), err))
			}
		}
	}
	log.Printf("cas: seeded %d batch(es) (tag=%s)", opt.Count, runTag)
	return res
}

// caRow mirrors the certificate_authorities table columns. Encrypted blob
// fields are left as zero-value []byte (NULL in SQL) so dibble doesn't need
// the server's private key.
type caRow struct {
	Type                          string
	Name                          *string
	URL                           *string
	APITokenEncrypted             []byte
	ProfileID                     *string
	CertificateCommonName         *string
	CertificateUserPrincipalNames []byte // JSON
	CertificateSeatID             *string
	AdminURL                      *string
	ChallengeURL                  *string
	Username                      *string
	PasswordEncrypted             []byte
	ChallengeEncrypted            []byte
	ClientID                      *string
	ClientSecretEncrypted         []byte
}

// caTemplates returns one row builder per non-NDES CA type. Each builder
// takes the run tag and a per-batch sequence number and returns a fully
// populated row with a unique (*-prefixed) name.
var caTemplates = []func(runTag string, seq int) caRow{
	func(runTag string, seq int) caRow {
		name := fmt.Sprintf("%s%s_scep_%d", caFakePrefix, runTag, seq)
		url := "https://fake.example.com/scep"
		return caRow{Type: "custom_scep_proxy", Name: &name, URL: &url}
	},
	func(runTag string, seq int) caRow {
		name := fmt.Sprintf("%s%s_est_%d", caFakePrefix, runTag, seq)
		url := "https://fake.example.com/.well-known/est"
		user := "*est-user"
		return caRow{Type: "custom_est_proxy", Name: &name, URL: &url, Username: &user}
	},
	func(runTag string, seq int) caRow {
		name := fmt.Sprintf("%s%s_digicert_%d", caFakePrefix, runTag, seq)
		url := "https://one.digicert.com"
		profileID := "00000000-0000-0000-0000-000000000000"
		cn := "*dibble-cn"
		seatID := "*dibble-seat"
		upns, _ := json.Marshal([]string{"*user@example.com"})
		return caRow{
			Type:                          "digicert",
			Name:                          &name,
			URL:                           &url,
			ProfileID:                     &profileID,
			CertificateCommonName:         &cn,
			CertificateUserPrincipalNames: upns,
			CertificateSeatID:             &seatID,
		}
	},
	func(runTag string, seq int) caRow {
		name := fmt.Sprintf("%s%s_hydrant_%d", caFakePrefix, runTag, seq)
		url := "https://fake.example.com/hydrant"
		clientID := "*dibble-client"
		return caRow{Type: "hydrant", Name: &name, URL: &url, ClientID: &clientID}
	},
	func(runTag string, seq int) caRow {
		name := fmt.Sprintf("%s%s_smallstep_%d", caFakePrefix, runTag, seq)
		url := "https://fake.example.com/scep"
		challengeURL := "https://fake.example.com/challenge"
		user := "*smallstep-user"
		return caRow{
			Type:         "smallstep",
			Name:         &name,
			URL:          &url,
			ChallengeURL: &challengeURL,
			Username:     &user,
		}
	},
}

func ndesRow() caRow {
	name := ndesName
	url := "https://fake.example.com/certsrv/mscep/mscep.dll"
	adminURL := "https://fake.example.com/certsrv/mscep_admin/"
	user := "*ndes-admin"
	return caRow{
		Type:     "ndes_scep_proxy",
		Name:     &name,
		URL:      &url,
		AdminURL: &adminURL,
		Username: &user,
	}
}

const insertCAStmt = `INSERT INTO certificate_authorities (
	type, name, url,
	api_token_encrypted, profile_id, certificate_common_name,
	certificate_user_principal_names, certificate_seat_id,
	admin_url, challenge_url, username,
	password_encrypted, challenge_encrypted,
	client_id, client_secret_encrypted
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

func insertCA(ctx context.Context, db *sql.DB, r caRow) error {
	_, err := db.ExecContext(ctx, insertCAStmt,
		r.Type, r.Name, r.URL,
		r.APITokenEncrypted, r.ProfileID, r.CertificateCommonName,
		r.CertificateUserPrincipalNames, r.CertificateSeatID,
		r.AdminURL, r.ChallengeURL, r.Username,
		r.PasswordEncrypted, r.ChallengeEncrypted,
		r.ClientID, r.ClientSecretEncrypted,
	)
	return err
}

// isDuplicateCAError reports whether err is a (type, name) unique-key
// collision, so the caller can count it as Skipped instead of Errored.
func isDuplicateCAError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "idx_ca_type_name") || strings.Contains(msg, "Duplicate entry")
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
