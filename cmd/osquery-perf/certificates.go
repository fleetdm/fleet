package main

import (
	"crypto/sha1" //nolint:gosec
	"encoding/hex"
	"fmt"
	"maps"
	// osquery-perf shares one global math/rand RNG seeded from the --seed flag so load-test runs are reproducible
	"math/rand" //nolint:depguard
	"strings"
	"time"

	"github.com/google/uuid"
)

// simulatedCert is a platform-neutral description of a certificate that osquery-perf reports for the `certificates`
// detail query.
type simulatedCert struct {
	ca                 bool
	commonName         string
	subjectCommonName  string
	subjectOrg         string
	subjectOrgUnit     string
	subjectCountry     string
	issuerCommonName   string
	issuerOrg          string
	issuerCountry      string
	keyAlgorithm       string
	keyStrength        string
	keyUsage           string
	signingAlgorithm   string
	serial             string
	notValidAfterUnix  string
	notValidBeforeUnix string
	// user reports whether the certificate lives in a user's store (true) or in the machine/system store (false).
	user     bool
	username string
}

// sha1Hex returns the hex-encoded SHA1 osquery would report for this cert. It is derived from the serial so that shared
// certs (fixed serial) dedupe to a single host_certificates row across all hosts, while per-host certs (uuid serial)
// stay unique per host.
func (c simulatedCert) sha1Hex() string {
	sum := sha1.Sum([]byte(c.serial)) //nolint: gosec
	return hex.EncodeToString(sum[:])
}

// sharedCerts are reported by every simulated host (common root and intermediate CAs).
var sharedCerts = []simulatedCert{
	{
		ca: true, commonName: "Fleet Root CA",
		subjectCommonName: "Fleet Root CA", subjectOrg: "Fleet Device Management Inc.", subjectCountry: "US",
		issuerCommonName: "Fleet Root CA", issuerOrg: "Fleet Device Management Inc.", issuerCountry: "US",
		keyAlgorithm: "rsaEncryption", keyStrength: "4096", keyUsage: "Certificate Signing, CRL Signing",
		signingAlgorithm: "sha256WithRSAEncryption", serial: "osquery-perf-shared-fleet-root-ca",
		notValidBeforeUnix: "1577836800", notValidAfterUnix: "1893456000", // 2020-01-01 .. 2030-01-01
	},
	{
		ca: true, commonName: "Fleet Intermediate CA",
		subjectCommonName: "Fleet Intermediate CA", subjectOrg: "Fleet Device Management Inc.", subjectOrgUnit: "Issuing", subjectCountry: "US",
		issuerCommonName: "Fleet Root CA", issuerOrg: "Fleet Device Management Inc.", issuerCountry: "US",
		keyAlgorithm: "rsaEncryption", keyStrength: "2048", keyUsage: "Certificate Signing, CRL Signing",
		signingAlgorithm: "sha256WithRSAEncryption", serial: "osquery-perf-shared-fleet-intermediate-ca",
		notValidBeforeUnix: "1577836800", notValidAfterUnix: "1893456000",
	},
	{
		ca: true, commonName: "DigiCert Global Root CA",
		subjectCommonName: "DigiCert Global Root CA", subjectOrg: "DigiCert Inc", subjectCountry: "US",
		issuerCommonName: "DigiCert Global Root CA", issuerOrg: "DigiCert Inc", issuerCountry: "US",
		keyAlgorithm: "rsaEncryption", keyStrength: "2048", keyUsage: "Certificate Signing, CRL Signing",
		signingAlgorithm: "sha256WithRSAEncryption", serial: "osquery-perf-shared-digicert-global-root-ca",
		notValidBeforeUnix: "1577836800", notValidAfterUnix: "1893456000",
	},
	{
		ca: true, commonName: "Microsoft Root Certificate Authority 2011",
		subjectCommonName: "Microsoft Root Certificate Authority 2011", subjectOrg: "Microsoft Corporation", subjectCountry: "US",
		issuerCommonName: "Microsoft Root Certificate Authority 2011", issuerOrg: "Microsoft Corporation", issuerCountry: "US",
		keyAlgorithm: "rsaEncryption", keyStrength: "4096", keyUsage: "Certificate Signing, CRL Signing",
		signingAlgorithm: "sha256WithRSAEncryption", serial: "osquery-perf-shared-microsoft-root-ca-2011",
		notValidBeforeUnix: "1577836800", notValidAfterUnix: "1893456000",
	},
	{
		ca: true, commonName: "GlobalSign Root CA",
		subjectCommonName: "GlobalSign Root CA", subjectOrg: "GlobalSign nv-sa", subjectOrgUnit: "Root CA", subjectCountry: "BE",
		issuerCommonName: "GlobalSign Root CA", issuerOrg: "GlobalSign nv-sa", issuerCountry: "BE",
		keyAlgorithm: "rsaEncryption", keyStrength: "2048", keyUsage: "Certificate Signing, CRL Signing",
		signingAlgorithm: "sha256WithRSAEncryption", serial: "osquery-perf-shared-globalsign-root-ca",
		notValidBeforeUnix: "1577836800", notValidAfterUnix: "1893456000",
	},
	{
		ca: true, commonName: "USERTrust RSA Certification Authority",
		subjectCommonName: "USERTrust RSA Certification Authority", subjectOrg: "The USERTRUST Network", subjectCountry: "US",
		issuerCommonName: "USERTrust RSA Certification Authority", issuerOrg: "The USERTRUST Network", issuerCountry: "US",
		keyAlgorithm: "rsaEncryption", keyStrength: "4096", keyUsage: "Certificate Signing, CRL Signing",
		signingAlgorithm: "sha384WithRSAEncryption", serial: "osquery-perf-shared-usertrust-rsa-ca",
		notValidBeforeUnix: "1577836800", notValidAfterUnix: "1893456000",
	},
	{
		ca: true, commonName: "ISRG Root X1",
		subjectCommonName: "ISRG Root X1", subjectOrg: "Internet Security Research Group", subjectCountry: "US",
		issuerCommonName: "ISRG Root X1", issuerOrg: "Internet Security Research Group", issuerCountry: "US",
		keyAlgorithm: "rsaEncryption", keyStrength: "4096", keyUsage: "Certificate Signing, CRL Signing",
		signingAlgorithm: "sha256WithRSAEncryption", serial: "osquery-perf-shared-isrg-root-x1",
		notValidBeforeUnix: "1577836800", notValidAfterUnix: "1893456000",
	},
	{
		ca: true, commonName: "Amazon Root CA 1",
		subjectCommonName: "Amazon Root CA 1", subjectOrg: "Amazon", subjectCountry: "US",
		issuerCommonName: "Amazon Root CA 1", issuerOrg: "Amazon", issuerCountry: "US",
		keyAlgorithm: "rsaEncryption", keyStrength: "2048", keyUsage: "Certificate Signing, CRL Signing",
		signingAlgorithm: "sha256WithRSAEncryption", serial: "osquery-perf-shared-amazon-root-ca-1",
		notValidBeforeUnix: "1577836800", notValidAfterUnix: "1893456000",
	},
	{
		ca: true, commonName: "Baltimore CyberTrust Root",
		subjectCommonName: "Baltimore CyberTrust Root", subjectOrg: "Baltimore", subjectOrgUnit: "CyberTrust", subjectCountry: "IE",
		issuerCommonName: "Baltimore CyberTrust Root", issuerOrg: "Baltimore", issuerCountry: "IE",
		keyAlgorithm: "rsaEncryption", keyStrength: "2048", keyUsage: "Certificate Signing, CRL Signing",
		signingAlgorithm: "sha256WithRSAEncryption", serial: "osquery-perf-shared-baltimore-cybertrust-root",
		notValidBeforeUnix: "1577836800", notValidAfterUnix: "1893456000",
	},
	{
		ca: true, commonName: "Entrust Root Certification Authority - G2",
		subjectCommonName: "Entrust Root Certification Authority - G2", subjectOrg: "Entrust, Inc.", subjectOrgUnit: "See www.entrust.net/legal-terms", subjectCountry: "US",
		issuerCommonName: "Entrust Root Certification Authority - G2", issuerOrg: "Entrust, Inc.", issuerCountry: "US",
		keyAlgorithm: "rsaEncryption", keyStrength: "2048", keyUsage: "Certificate Signing, CRL Signing",
		signingAlgorithm: "sha256WithRSAEncryption", serial: "osquery-perf-shared-entrust-root-ca-g2",
		notValidBeforeUnix: "1577836800", notValidAfterUnix: "1893456000",
	},
}

const certDay = 24 * time.Hour

// certChurnPercent is the percent chance that some of a host's per-host certificates rotate (new serial and SHA1),
// simulating certificate renewal/reinstall. It is rolled each time the host answers the certificates detail query,
// i.e. on the periodic detail refresh (osquery.detail_update_interval, 1h by default) or a forced refetch. Shared
// certs never churn.
const certChurnPercent = 5

// generateCertSpecs returns the certs this host reports: the constant shared certs plus this host's per-host certs.
// Per-host certs are generated once and cached so they're stable across detail-query refreshes, then occasionally
// churned to simulate certificate rotation/installs. Shared certs are never churned.
func (a *agent) generateCertSpecs() []simulatedCert {
	a.certificatesMutex.Lock()
	defer a.certificatesMutex.Unlock()

	switch {
	case a.hostCertSpecs == nil:
		a.hostCertSpecs = a.newPerHostCertSpecs()
	case rand.Intn(100) < certChurnPercent:
		a.churnPerHostCertSpecs()
	}

	specs := make([]simulatedCert, 0, len(sharedCerts)+len(a.hostCertSpecs))
	specs = append(specs, sharedCerts...)
	specs = append(specs, a.hostCertSpecs...)
	return specs
}

// newPerHostCertSpecs generates 0-10 certificates unique to this host
func (a *agent) newPerHostCertSpecs() []simulatedCert {
	count := rand.Intn(11) // 0..10
	users := a.hostUsers()
	specs := make([]simulatedCert, 0, count+1)
	for i := range count {
		specs = append(specs, a.newPerHostCertSpec(i, users))
	}
	// Model a device certificate present in both the machine store and a user's store (same SHA1, two scopes),
	// exercising the server's cross-scope handling (one host_certificates row, two host_certificate_sources rows).
	// churnPerHostCertSpecs may later rotate one of the pair and break the cross-scope match; not worth guarding against
	// for a load-test simulator (the path is exercised on initial ingestion and this is a low-volume, per-host scenario).
	if count > 0 && len(users) > 0 {
		dup := specs[0]
		dup.user = !specs[0].user
		if dup.user {
			dup.username = users[rand.Intn(len(users))]["username"]
		} else {
			dup.username = ""
		}
		specs = append(specs, dup)
	}
	return specs
}

func (a *agent) newPerHostCertSpec(i int, users []map[string]string) simulatedCert {
	user := rand.Intn(2) == 0 && len(users) > 0
	username := ""
	if user {
		username = users[rand.Intn(len(users))]["username"]
	}
	return simulatedCert{
		commonName:        uuid.NewString(),
		subjectCommonName: fmt.Sprintf("Subject %d Common Name", i),
		subjectOrg:        fmt.Sprintf("Subject %d Inc.", i),
		subjectOrgUnit:    fmt.Sprintf("Subject %d Org Unit", i),
		subjectCountry:    "US",
		issuerCommonName:  fmt.Sprintf("Issuer %d Common Name", i),
		issuerOrg:         fmt.Sprintf("Issuer %d Inc.", i),
		issuerCountry:     "US",
		keyAlgorithm:      "rsaEncryption",
		keyStrength:       "2048",
		keyUsage:          "Data Encipherment, Key Encipherment, Digital Signature",
		signingAlgorithm:  "sha256WithRSAEncryption",
		serial:            uuid.NewString(),
		// generate so that it may be expired (notAfter in [-1d, +99d])
		notValidAfterUnix: fmt.Sprint(time.Now().Add(-1 * certDay).Add(time.Duration(rand.Intn(100)) * certDay).Unix()),
		// notBefore is always in the past (1-10 days)
		notValidBeforeUnix: fmt.Sprint(time.Now().Add(-time.Duration(rand.Intn(10)+1) * certDay).Unix()),
		user:               user,
		username:           username,
	}
}

// churnPerHostCertSpecs rotates 1..N of this host's per-host certs by assigning new serials (and thus new SHA1s),
// simulating certificate renewal/reinstall.
func (a *agent) churnPerHostCertSpecs() {
	if len(a.hostCertSpecs) == 0 {
		return
	}
	n := rand.Intn(min(10, len(a.hostCertSpecs))) + 1
	for range n {
		idx := rand.Intn(len(a.hostCertSpecs))
		a.hostCertSpecs[idx].serial = uuid.NewString()
		a.hostCertSpecs[idx].commonName = uuid.NewString()
		a.hostCertSpecs[idx].notValidAfterUnix = fmt.Sprint(time.Now().Add(-1 * certDay).Add(time.Duration(rand.Intn(100)) * certDay).Unix())
	}
}

func boolStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// darwinDN renders a slash-delimited distinguished name (e.g. /C=US/O=Org/OU=Unit/CN=Name) as osquery returns on macOS.
// Empty fields are omitted.
func darwinDN(country, org, orgUnit, commonName string) string {
	var b strings.Builder
	if country != "" {
		b.WriteString("/C=" + escapeDarwinDNValue(country))
	}
	if org != "" {
		b.WriteString("/O=" + escapeDarwinDNValue(org))
	}
	if orgUnit != "" {
		b.WriteString("/OU=" + escapeDarwinDNValue(orgUnit))
	}
	if commonName != "" {
		b.WriteString("/CN=" + escapeDarwinDNValue(commonName))
	}
	return b.String()
}

// escapeDarwinDNValue backslash-escapes slashes inside an attribute value, as osquery does on macOS
func escapeDarwinDNValue(v string) string {
	return strings.ReplaceAll(v, "/", `\/`)
}

// windowsDN renders an X.500 (RFC 1779) distinguished name (e.g. "CN=Name, O=Org, OU=Unit, C=US") as osquery returns in
// subject2/issuer2 on Windows starting with osquery 5.23.1.
func windowsDN(country, org, orgUnit, commonName string) string {
	var parts []string
	if commonName != "" {
		parts = append(parts, "CN="+quoteX500Value(commonName))
	}
	if org != "" {
		parts = append(parts, "O="+quoteX500Value(org))
	}
	if orgUnit != "" {
		parts = append(parts, "OU="+quoteX500Value(orgUnit))
	}
	if country != "" {
		parts = append(parts, "C="+quoteX500Value(country))
	}
	return strings.Join(parts, ", ")
}

// quoteX500Value double-quotes an attribute value that contains a comma (doubling any embedded quotes), as
// CERT_X500_NAME_STR does, e.g. O="Entrust, Inc.".
func quoteX500Value(v string) string {
	if !strings.ContainsAny(v, `,"`) {
		return v
	}
	return `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
}

// windowsUserSID returns a stable per-(host, user) security identifier so a user's certs classify as User scope and stay
// consistent across detail-query refreshes.
func (a *agent) windowsUserSID(username string) string {
	var h uint32 = 2166136261
	for i := 0; i < len(username); i++ {
		h = (h ^ uint32(username[i])) * 16777619
	}
	rid := 1000 + int(h%5000)
	return fmt.Sprintf("S-1-5-21-%d-%d-%d-%d", 1000000000+a.agentIndex, 2000000000, 3000000000, rid)
}

func (a *agent) certificatesDarwin() []map[string]string {
	specs := a.generateCertSpecs()
	rows := make([]map[string]string, 0, len(specs))
	for _, c := range specs {
		rows = append(rows, c.darwinRow())
	}
	return rows
}

func (c simulatedCert) darwinRow() map[string]string {
	source := "system"
	path := "/Library/Keychains/System.keychain"
	if c.user {
		source = "user"
		path = fmt.Sprintf("/Users/%s/Library/Keychains/login.keychain-db", c.username)
	}
	return map[string]string{
		"ca":                boolStr(c.ca),
		"common_name":       c.commonName,
		"subject":           darwinDN(c.subjectCountry, c.subjectOrg, c.subjectOrgUnit, c.subjectCommonName),
		"issuer":            darwinDN(c.issuerCountry, c.issuerOrg, "", c.issuerCommonName),
		"key_algorithm":     c.keyAlgorithm,
		"key_strength":      c.keyStrength,
		"key_usage":         c.keyUsage,
		"signing_algorithm": c.signingAlgorithm,
		"not_valid_after":   c.notValidAfterUnix,
		"not_valid_before":  c.notValidBeforeUnix,
		"serial":            c.serial,
		"sha1":              c.sha1Hex(),
		"source":            source,
		"path":              path,
	}
}

func (a *agent) certificatesWindows() []map[string]string {
	specs := a.generateCertSpecs()
	// User certs are enumerated from more than one hive, so allocate room for ~2
	// rows per spec.
	rows := make([]map[string]string, 0, len(specs)*2)
	for _, c := range specs {
		rows = append(rows, a.windowsRows(c)...)
	}
	return rows
}

// windowsRows renders the osquery `certificates` rows for a cert on Windows. Machine-scoped certs produce one row.
// User-scoped certs produce the redundant rows osquery returns from the user's Personal hive and its companion _Classes
// hive (the Fleet server dedupes them by SHA1 + scope + username).
func (a *agent) windowsRows(c simulatedCert) []map[string]string {
	base := map[string]string{
		"ca":          boolStr(c.ca),
		"common_name": c.commonName,
		// subject2/issuer2 are the X.500 distinguished name columns Fleet's Windows certificates query selects,
		// populated on Windows starting with osquery 5.23.1.
		"subject2":          windowsDN(c.subjectCountry, c.subjectOrg, c.subjectOrgUnit, c.subjectCommonName),
		"issuer2":           windowsDN(c.issuerCountry, c.issuerOrg, "", c.issuerCommonName),
		"key_algorithm":     c.keyAlgorithm,
		"key_strength":      c.keyStrength,
		"key_usage":         c.keyUsage,
		"signing_algorithm": c.signingAlgorithm,
		"not_valid_after":   c.notValidAfterUnix,
		"not_valid_before":  c.notValidBeforeUnix,
		"serial":            c.serial,
		"sha1":              c.sha1Hex(),
	}

	if !c.user {
		row := maps.Clone(base)
		row["sid"] = ""
		row["username"] = ""
		row["store_location"] = "LocalMachine"
		row["path"] = "LocalMachine\\Personal"
		return []map[string]string{row}
	}

	sid := a.windowsUserSID(c.username)
	personal := maps.Clone(base)
	personal["sid"] = sid
	personal["username"] = c.username
	personal["store_location"] = "Users"
	personal["path"] = fmt.Sprintf("Users\\%s\\Personal", sid)

	classes := maps.Clone(personal)
	classes["path"] = fmt.Sprintf("Users\\%s_Classes\\Personal", sid)

	return []map[string]string{personal, classes}
}
