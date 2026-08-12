// Command decrypt-value decrypts a value that Fleet encrypted for MDM, such as
// a disk encryption key escrowed by a host (FileVault or LUKS) or an entry in
// the mdm_config_assets table.
//
// Fleet uses two schemes for these values, and the tool detects which one
// applies from the value itself:
//
//   - CMS/PKCS#7: the value is encrypted to the Apple MDM CA certificate, which
//     the tool reads from the mdm_config_assets table (including certificates
//     from previous CA rollovers, so keys escrowed before a rollover still
//     decrypt).
//   - AES-256-GCM: the value is encrypted with the Fleet server private key
//     alone, no database needed. Used for LUKS passphrases and salts, and for
//     the mdm_config_assets values themselves.
//
// So all it needs is the database connection, the Fleet server private key, and
// the value; the certificate and key material come out of the database.
//
// The value may be base64 (as stored in host_disk_encryption_keys) or hex, with
// or without a 0x prefix (as MySQL clients print binary columns such as
// mdm_config_assets.value). Whitespace and line breaks are ignored. Some inputs
// are valid in more than one encoding, so the tool tries each and reports which
// one decrypted.
//
// Windows (BitLocker) keys are encrypted to the WSTEP identity certificate,
// which lives in the server configuration rather than the database. Use
// ./tools/mdm/decrypt-disk-encryption-key with the WSTEP cert and key files for
// those.
//
// Example usage (running from the root of this repository):
//
//	go run ./tools/mdm/decrypt-value -db fleet:pass@db.example.com:3306/fleet \
//	  -key $FLEET_SERVER_PRIVATE_KEY -value encrypted-value
package main

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/smallstep/pkcs7"
)

const defaultDB = "fleet:insecure@localhost:3306/fleet"

// aesGCMOverhead is the smallest an AES-GCM value can be: a 12 byte nonce plus
// a 16 byte tag. Anything shorter can't be one, and must not be handed to
// mdm.DecodeAndDecrypt, which slices off the nonce without a length check.
const aesGCMOverhead = 12 + 16

func main() {
	var (
		flagDB    = flag.String("db", defaultDB, "MySQL connection to the Fleet database, as user:password@host:port/database.")
		flagKey   = flag.String("key", os.Getenv("FLEET_SERVER_PRIVATE_KEY"), "The Fleet server private key (defaults to $FLEET_SERVER_PRIVATE_KEY).")
		flagValue = flag.String("value", "", "The encrypted value to decrypt, base64 or hex encoded (required).")
	)
	flag.Parse()

	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	}))

	decrypted, err := run(ctx, logger, *flagDB, *flagKey, *flagValue)
	if err != nil {
		logger.ErrorContext(ctx, "decrypting value", "err", err)
		os.Exit(1)
	}
	fmt.Printf("Decrypted value: %s\n", decrypted)
}

func run(ctx context.Context, logger *slog.Logger, db, privateKey, value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		flag.Usage()
		return "", errors.New("-value is required")
	}
	if privateKey == "" {
		return "", errors.New("-key is required (the Fleet server private key)")
	}
	// AES-256 requires a 32 byte (256 bit) key, but some infra setups generate
	// keys that are longer than 32 bytes, so truncate like the server does (see
	// cmd/fleet/serve.go).
	keyBytes := len(privateKey)
	if len(privateKey) > 32 {
		privateKey = privateKey[:32]
	}

	candidates := decodeCandidates(value)
	if len(candidates) == 0 {
		return "", errors.New("-value is neither valid base64 nor valid hex, make sure the whole value was copied")
	}

	// The Apple MDM CA is only loaded if some candidate is actually a CMS value,
	// so values encrypted with the server private key need no database at all.
	loadCerts := certLoader(ctx, db, privateKey)

	var failures []string
	for _, c := range candidates {
		// A value encrypted to a certificate parses as PKCS#7; one encrypted
		// with the server private key doesn't. The parse result picks the
		// scheme, and AES-GCM authentication rules out a wrong guess.
		b64 := base64.StdEncoding.EncodeToString(c.raw)

		if _, err := pkcs7.Parse(c.raw); err == nil {
			certs, key, err := loadCerts()
			if err != nil {
				return "", err
			}
			decrypted, err := mdm.DecryptBase64CMSWithCerts(b64, key, certs)
			if err != nil {
				failures = append(failures, fmt.Sprintf("%s/CMS: %v", c.encoding, err))
				continue
			}
			logger.InfoContext(ctx, "decrypted value", "encoding", c.encoding, "scheme", "CMS",
				"value_bytes", len(c.raw), "private_key_bytes", keyBytes)
			return string(decrypted), nil
		}

		if len(c.raw) < aesGCMOverhead {
			failures = append(failures, fmt.Sprintf("%s: %d bytes is too short to be an encrypted value", c.encoding, len(c.raw)))
			continue
		}
		decrypted, err := mdm.DecodeAndDecrypt(b64, privateKey)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s/AES-GCM: %v", c.encoding, err))
			continue
		}
		logger.InfoContext(ctx, "decrypted value", "encoding", c.encoding, "scheme", "AES-GCM",
			"value_bytes", len(c.raw), "private_key_bytes", keyBytes)
		return decrypted, nil
	}

	return "", fmt.Errorf("could not decrypt the value (%s)."+
		" Check that -key is the FLEET_SERVER_PRIVATE_KEY of the environment that escrowed this value,"+
		" and that the value was copied in full."+
		" A Windows (BitLocker) key needs the WSTEP certificate instead, see ./tools/mdm/decrypt-disk-encryption-key",
		strings.Join(failures, "; "))
}

// candidate is one interpretation of the -value flag: the bytes it decodes to
// under a given encoding.
type candidate struct {
	encoding string
	raw      []byte
}

// decodeCandidates returns every way the value decodes, most likely first. A
// value can be valid base64 and valid hex at the same time (e.g. "deadbeef"),
// in which case both interpretations are returned and the caller decides by
// trying to decrypt each.
func decodeCandidates(value string) []candidate {
	// Strip whitespace so values pasted across several lines work, along with
	// the 0x prefix that MySQL clients put on binary column values.
	value = strings.Join(strings.Fields(value), "")
	unprefixed := strings.TrimPrefix(strings.TrimPrefix(value, "0x"), "0X")

	var candidates []candidate
	if raw, err := hex.DecodeString(unprefixed); err == nil && len(raw) > 0 {
		candidates = append(candidates, candidate{encoding: "hex", raw: raw})
	}
	// Padded first, since that's how Fleet stores base64 values.
	for _, base64Enc := range []struct {
		name string
		enc  *base64.Encoding
	}{
		{"base64", base64.StdEncoding},
		{"base64 (unpadded)", base64.RawStdEncoding},
	} {
		if raw, err := base64Enc.enc.DecodeString(value); err == nil && len(raw) > 0 {
			candidates = append(candidates, candidate{encoding: base64Enc.name, raw: raw})
			break
		}
	}
	return candidates
}

// certLoader returns a function that loads the Apple MDM CA certificates and
// private key from the database on first call, and caches the outcome.
func certLoader(ctx context.Context, db, privateKey string) func() ([]*x509.Certificate, crypto.PrivateKey, error) {
	var (
		loaded bool
		certs  []*x509.Certificate
		key    crypto.PrivateKey
		err    error
	)
	return func() ([]*x509.Certificate, crypto.PrivateKey, error) {
		if loaded {
			return certs, key, err
		}
		loaded = true

		var mysqlCfg config.MysqlConfig
		if mysqlCfg, err = parseDB(db); err != nil {
			return nil, nil, err
		}
		var ds *mysql.Datastore
		ds, err = mysql.New(
			mysqlCfg,
			clock.C,
			mysql.LimitAttempts(1),
			mysql.WithFleetConfig(&config.FleetConfig{
				Server: config.ServerConfig{PrivateKey: privateKey},
			}),
		)
		if err != nil {
			err = fmt.Errorf("connecting to the database: %w", err)
			return nil, nil, err
		}
		defer ds.Close()

		// Include previously-rolled-over CA certs so keys escrowed against an
		// earlier CA still decrypt with the (unchanged) private key.
		certs, key, err = assets.CACertsAndKeyForDecryption(ctx, ds)
		if err != nil {
			err = fmt.Errorf("loading the Apple MDM CA certificate and key from the database: %w", err)
			return nil, nil, err
		}
		return certs, key, nil
	}
}

// parseDB turns a user:password@host:port/database connection string into a
// MySQL config. Full driver DSNs (with an explicit tcp(...)/unix(...) address
// and parameters) are accepted too.
func parseDB(db string) (config.MysqlConfig, error) {
	cfg, err := mysqldriver.ParseDSN(normalizeDSN(db))
	if err != nil {
		return config.MysqlConfig{}, fmt.Errorf("parsing -db %q, expected user:password@host:port/database: %w", db, err)
	}
	if cfg.DBName == "" {
		return config.MysqlConfig{}, fmt.Errorf("parsing -db %q: missing database name", db)
	}

	tlsConfig := cfg.TLSConfig
	if tlsConfig == "" {
		tlsConfig = "skip-verify"
	}
	return config.MysqlConfig{
		Username:  cfg.User,
		Password:  cfg.Passwd,
		Address:   cfg.Addr,
		Database:  cfg.DBName,
		TLSConfig: tlsConfig,
	}, nil
}

// normalizeDSN wraps a bare host:port address in the tcp(...) form the MySQL
// driver expects, so that both user:pass@localhost:3306/fleet and
// user:pass@tcp(localhost:3306)/fleet work.
func normalizeDSN(dsn string) string {
	at := strings.LastIndex(dsn, "@")
	if at < 0 {
		return dsn
	}
	credentials, rest := dsn[:at+1], dsn[at+1:]

	address, dbAndParams := rest, ""
	if slash := strings.Index(rest, "/"); slash >= 0 {
		address, dbAndParams = rest[:slash], rest[slash:]
	}
	if address == "" || strings.HasSuffix(address, ")") {
		return dsn
	}
	return credentials + "tcp(" + address + ")" + dbAndParams
}
