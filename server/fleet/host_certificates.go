package fleet

import (
	"crypto/sha1" // nolint:gosec // used for compatibility with existing osquery certificates table schema
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type HostCertificateSource string

const (
	SystemHostCertificate HostCertificateSource = "system"
	UserHostCertificate   HostCertificateSource = "user"
)

// IsValid returns true if the current host certificate source value is
// accepted, otherwise false.
func (s HostCertificateSource) IsValid() bool {
	switch s {
	case SystemHostCertificate, UserHostCertificate:
		return true
	default:
		return false
	}
}

// HostCertificateOrigin identifies the ingestion path that recorded a
// host_certificates row. It scopes deletion semantics: each ingestion source
// only soft-deletes rows it owns, so an osquery sync omitting an MDM-only cert
// does not remove that cert, and vice versa.
//
// Internal-only: not exposed in the public API.
type HostCertificateOrigin string

const (
	HostCertificateOriginOsquery HostCertificateOrigin = "osquery"
	HostCertificateOriginMDM     HostCertificateOrigin = "mdm"
)

// HostCertificateScope identifies a single (source, username) certificate
// scope. It is used to tell UpdateHostCertificates which scopes the agent could
// authoritatively enumerate during a collection cycle, so reconciliation does
// not soft-delete certificates for a scope it could not observe.
//
// A user's Windows certificates are only visible to osquery while that user is
// logged in (their registry hive is loaded), so the Windows ingestion path
// passes the set of observed scopes and absent users' certificates are
// preserved. The macOS path reads every keychain from disk on every run, so it
// passes a nil slice, meaning "all scopes observed" and any absent certificate
// may be deleted.
type HostCertificateScope struct {
	Source   HostCertificateSource
	Username string
}

// HostCertificateRecord is the database model for a host certificate.
type HostCertificateRecord struct {
	ID     uint `json:"-" db:"id"`
	HostID uint `json:"-" db:"host_id"`

	// SHA1Sum is a SHA-1 hash of the DER encoded certificate.
	SHA1Sum []byte `json:"-" db:"sha1_sum"`

	// CreatedAt is the time the certificate was recorded by Fleet (i.e. certificate initially
	// reported to Fleet).
	CreatedAt time.Time `json:"-" db:"created_at"`
	// DeletedAt is the time the certificate was soft deleted by Fleet (i.e. previously reported to
	// Fleet certificate is subsequently not reported).
	DeletedAt *time.Time `json:"-" db:"deleted_at"`

	// The following fields are extracted from the certificate.
	NotValidAfter        time.Time `json:"-" db:"not_valid_after"`
	NotValidBefore       time.Time `json:"-" db:"not_valid_before"`
	CertificateAuthority bool      `json:"-" db:"certificate_authority"`
	CommonName           string    `json:"-" db:"common_name"`
	KeyAlgorithm         string    `json:"-" db:"key_algorithm"`
	KeyStrength          int       `json:"-" db:"key_strength"`
	KeyUsage             string    `json:"-" db:"key_usage"`
	Serial               string    `json:"-" db:"serial"`
	SigningAlgorithm     string    `json:"-" db:"signing_algorithm"`

	SubjectCountry            string `json:"-" db:"subject_country"`
	SubjectOrganization       string `json:"-" db:"subject_org"`
	SubjectOrganizationalUnit string `json:"-" db:"subject_org_unit"`
	SubjectCommonName         string `json:"-" db:"subject_common_name"`
	IssuerCountry             string `json:"-" db:"issuer_country"`
	IssuerOrganization        string `json:"-" db:"issuer_org"`
	IssuerOrganizationalUnit  string `json:"-" db:"issuer_org_unit"`
	IssuerCommonName          string `json:"-" db:"issuer_common_name"`

	Source   HostCertificateSource `json:"-" db:"source"`
	Username string                `json:"-" db:"username"` // username that owns the certificate, only if source == 'user'

	// Origin identifies the ingestion source (osquery vs mdm). Used internally to
	// scope deletion semantics; not exposed in the public API.
	Origin HostCertificateOrigin `json:"-" db:"origin"`
}

func NewHostCertificateRecord(
	hostID uint,
	cert *x509.Certificate,
) *HostCertificateRecord {
	hash := sha1.Sum(cert.Raw) // nolint:gosec

	return &HostCertificateRecord{
		HostID:               hostID,
		SHA1Sum:              hash[:], // nolint:gosec
		NotValidAfter:        cert.NotAfter,
		NotValidBefore:       cert.NotBefore,
		CertificateAuthority: cert.IsCA,
		// TODO: we need to define methodology for determining common name analogous to osquery,
		// which seems to preferentially use Subject.CommonName for this value:
		// https://github.com/osquery/osquery/blob/16bb01508eeca6d663b6d4f7e15034306be0fc3d/osquery/tables/system/posix/openssl_utils.cpp#L253
		CommonName:   cert.Subject.CommonName,
		KeyAlgorithm: cert.PublicKeyAlgorithm.String(),
		// TODO: we need to define methodology for determining key strength analogous to osquery,
		// which describes this value as "Key size used for RSA/DSA, or curve name":
		// https://github.com/osquery/osquery/blob/16bb01508eeca6d663b6d4f7e15034306be0fc3d/osquery/tables/system/posix/openssl_utils.cpp#L337
		KeyStrength: 0, // TODO: add key strength here
		// TODO: we need to define methodology for determining key usage analogous to osquery, which
		// describes this as "Certificate key usage and extended key usage":
		// https://github.com/osquery/osquery/blob/16bb01508eeca6d663b6d4f7e15034306be0fc3d/osquery/tables/system/posix/openssl_utils.cpp#L166
		KeyUsage:                  "",
		Serial:                    cert.SerialNumber.Text(16),
		SigningAlgorithm:          cert.SignatureAlgorithm.String(),
		SubjectCommonName:         cert.Subject.CommonName,
		SubjectCountry:            firstOrEmpty(cert.Subject.Country),                    // TODO: confirm methodology
		SubjectOrganization:       firstOrEmpty(cert.Subject.Organization),               // TODO: confirm methodology
		SubjectOrganizationalUnit: strings.Join(cert.Subject.OrganizationalUnit, "+OU="), // NOTE: concatenation approach matches what we've observed osquery to do when there are multiple OU values
		IssuerCommonName:          cert.Issuer.CommonName,
		IssuerCountry:             firstOrEmpty(cert.Issuer.Country),            // TODO: confirm methodology
		IssuerOrganization:        firstOrEmpty(cert.Issuer.Organization),       // TODO: confirm methodology
		IssuerOrganizationalUnit:  firstOrEmpty(cert.Issuer.OrganizationalUnit), // TODO: confirm methodology
		Source:                    SystemHostCertificate,                        // default to system host certificate, always 'system' for certs from MDM command for now
		Username:                  "",                                           // always empty since this is a system certificate
	}
}

// ToPayload fills a HostCertificatePayload with the fields of a
// HostCertificateRecord. The HostCertificatePayload is used in API responses.
func (r *HostCertificateRecord) ToPayload() *HostCertificatePayload {
	subject := &HostCertificateNameDetails{
		CommonName:         r.SubjectCommonName,
		Country:            r.SubjectCountry,
		Organization:       r.SubjectOrganization,
		OrganizationalUnit: r.SubjectOrganizationalUnit,
	}
	issuer := &HostCertificateNameDetails{
		CommonName:         r.IssuerCommonName,
		Country:            r.IssuerCountry,
		Organization:       r.IssuerOrganization,
		OrganizationalUnit: r.IssuerOrganizationalUnit,
	}
	return &HostCertificatePayload{
		ID:                   r.ID,
		NotValidAfter:        r.NotValidAfter,
		NotValidBefore:       r.NotValidBefore,
		CertificateAuthority: r.CertificateAuthority,
		CommonName:           r.CommonName,
		KeyAlgorithm:         r.KeyAlgorithm,
		KeyStrength:          r.KeyStrength,
		KeyUsage:             r.KeyUsage,
		Serial:               r.Serial,
		SigningAlgorithm:     r.SigningAlgorithm,
		Source:               r.Source,
		Username:             r.Username,
		Subject:              subject,
		Issuer:               issuer,
	}
}

// HostCertificatePayload is the JSON model for API endpoints that return host certificates.
type HostCertificatePayload struct {
	ID                   uint                  `json:"id"`
	NotValidAfter        time.Time             `json:"not_valid_after"`
	NotValidBefore       time.Time             `json:"not_valid_before"`
	CertificateAuthority bool                  `json:"certificate_authority"`
	CommonName           string                `json:"common_name"`
	KeyAlgorithm         string                `json:"key_algorithm"`
	KeyStrength          int                   `json:"key_strength"`
	KeyUsage             string                `json:"key_usage"`
	Serial               string                `json:"serial"`
	SigningAlgorithm     string                `json:"signing_algorithm"`
	Source               HostCertificateSource `json:"source"`
	Username             string                `json:"username"`

	Subject *HostCertificateNameDetails `json:"subject,omitempty"`
	Issuer  *HostCertificateNameDetails `json:"issuer,omitempty"`
}

type HostCertificateNameDetails struct {
	CommonName         string `json:"common_name"`
	Country            string `json:"country"`
	Organization       string `json:"organization"`
	OrganizationalUnit string `json:"organizational_unit"`
}

// MDMAppleCertificateListResponse is the plist model for a certificate list response.
// https://developer.apple.com/documentation/devicemanagement/certificatelistresponse
type MDMAppleCertificateListResponse struct {
	CertificateList  []MDMAppleCertificateListItem `plist:"CertificateList"`
	CommandUUID      string                        `plist:"CommandUUID"`
	EnrollmentID     string                        `plist:"EnrollmentID"`
	EnrollmentUserID string                        `plist:"EnrollmentUserID"`
	ErrorChain       []MDMAppleErrorChainItem      `plist:"ErrorChain"`
	NotOnConsole     bool                          `plist:"NotOnConsole"`
	Status           string                        `plist:"Status"`
	UDID             string                        `plist:"UDID"`
	UserID           string                        `plist:"UserID"`
	UserLongName     string                        `plist:"UserLongName"`
	UserShortName    string                        `plist:"UserShortName"`
}

// MDMAppleCertificateListItem is the plist model for a certificate.
// https://developer.apple.com/documentation/devicemanagement/certificatelistresponse/certificatelistitem
type MDMAppleCertificateListItem struct {
	CommonName string `plist:"CommonName"`
	// Data is the DER-encoded certificate.
	Data       []byte `plist:"Data"`
	IsIdentity bool   `plist:"IsIdentity"`
}

func (c *MDMAppleCertificateListItem) Parse(hostID uint) (*HostCertificateRecord, error) {
	cert, err := x509.ParseCertificate(c.Data)
	if err != nil {
		return nil, err
	}
	return NewHostCertificateRecord(hostID, cert), nil
}

// MdmAppleErrorChainItem is the plist model for an error chain item.
// https://developer.apple.com/documentation/devicemanagement/certificatelistresponse/errorchainitem
type MDMAppleErrorChainItem struct {
	ErrorCode            int    `plist:"ErrorCode"`
	ErrorDomain          string `plist:"ErrorDomain"`
	LocalizedDescription string `plist:"LocalizedDescription"`
	USEnglishDescription string `plist:"USEnglishDescription"`
}

// ExtractDetailsFromOsqueryDistinguishedName parses a distinguished name and returns the country,
// organization, and organizational unit. It assumes provided string follows the formatting used by
// osquery `certificates` table[1], which appears to follow the style used by openSSL for `-subj`
// values). Key-value pairs are assumed to be separated by forward slashes, for example:
// "/C=US/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=FleetDM".
//
// See https://osquery.io/schema/5.15.0/#certificates
func ExtractDetailsFromOsqueryDistinguishedName(hostPlatform string, dn string) (*HostCertificateNameDetails, error) {
	switch {
	case strings.EqualFold(hostPlatform, "windows"):
		return parseWindowsDN(dn)
	case strings.EqualFold(hostPlatform, "darwin"), strings.EqualFold(hostPlatform, "macos"):
		return parseDarwinDN(dn)
	default:
		// usage error for unsupported host platforms to alert callers and ensure that
		// platform-specific considerations are handled explicitly
		return nil, fmt.Errorf("host platform not supported for osquery distinguished name parsing: %s %s", hostPlatform, dn)
	}
}

// parseDarwinDN takes a distinguished name string and returns the country,
// organization, and organizational unit. It assumes provided string follows the formatting used by
// osquery `certificates` table[1] for macOS hosts, which appears to follow the style used by openSSL for `-subj`
// values). Key-value pairs are assumed to be separated by forward slashes, for example:
// "/C=US/O=Fleet Device Management Inc./OU=Fleet Device Management Inc./CN=FleetDM".
//
// See https://osquery.io/schema/5.15.0/#certificates
func parseDarwinDN(dn string) (*HostCertificateNameDetails, error) {
	dn = strings.TrimSpace(dn)
	dn = strings.Trim(dn, "/")

	dn = strings.ReplaceAll(dn, `\/`, `<<SLASH>>`) // Replace with our own "safe" sequence
	parts := strings.Split(dn, "/")

	if len(parts) == 1 {
		// Try to split into parts based on +
		parts = strings.Split(dn, "+")
	}

	ouParts := []string{}
	var details HostCertificateNameDetails
	for _, part := range parts {
		key, value, found := strings.Cut(part, "=")

		if !found {
			return nil, fmt.Errorf("invalid distinguished name, wrong key value pair format: %s", dn)
		}

		value = strings.ReplaceAll(strings.Trim(value, " "), `<<SLASH>>`, `/`) // Replace our "safe" sequence with forward slash

		applyDNAttribute(&details, &ouParts, key, value)
	}
	details.OrganizationalUnit = strings.Join(ouParts, "+OU=")

	return &details, nil
}

// applyDNAttribute assigns a single distinguished-name attribute (key/value
// pair) to the matching field of details. It is shared by the macOS and Windows
// distinguished-name parsers so the attribute → field mapping stays identical
// across platforms (only the tokenization differs). Organizational units are
// accumulated because osquery can report multiple OU values for one
// certificate; the caller joins them. Attributes Fleet does not display (state,
// locality, bare dotted-decimal OIDs, ...) are ignored.
func applyDNAttribute(details *HostCertificateNameDetails, ouParts *[]string, key, value string) {
	switch strings.ToUpper(strings.TrimSpace(key)) {
	case "C":
		details.Country = value
	case "O":
		details.Organization = value
	case "OU":
		// osquery is inconsistent in how it reports certs with multiple OUs; sometimes it
		// concatenates them all joined by `+OU=` separator within the same `/` delimited
		// string, other times it provides multiple `/` delimited strings that each contain
		// distinct OU values. For example, compare the following two lines:
		//   /OU=SomeValue/OU=fleet-a3d5d6f4c-819e-4159-9a42-0d6243a80ff8/CN=SomeName
		//   /OU=SomeValue+OU=fleet-a0c039413-d0c7-4b1f-9488-b93c865351ac/CN=SomeName
		//
		// To handle both cases, we collect all OU values and join them with `+OU=` (done by
		// the caller). We should probably reconsider our approaches for normalization of cert
		// data across the board.
		*ouParts = append(*ouParts, value)
	case "CN":
		details.CommonName = value
	}
}

// parseWindowsDN parses a distinguished name in the X.500 string form that
// osquery emits in the `subject2` / `issuer2` columns on Windows starting with
// osquery 5.23.1 (osquery/osquery#8963), for example:
//
//	CN=Example, O="Example, Inc.", OU=A + OU=B, C=US
//
// Relative distinguished names (RDNs) are comma-separated; a multi-valued RDN
// joins its attributes with `+`; a value containing a separator (`,`, `+`, `=`,
// ...) is wrapped in double quotes with any embedded quote doubled. Unlike the
// macOS form parsed by parseDarwinDN (a slash-delimited openSSL style with the
// attribute keys preserved), this form is comma-delimited and quoted, so it
// needs its own tokenizer. Malformed fragments are skipped rather than failing
// the whole certificate, since a single odd attribute should not block
// ingestion of the batch.
func parseWindowsDN(dn string) (*HostCertificateNameDetails, error) {
	var details HostCertificateNameDetails
	var ouParts []string
	for _, attr := range splitX500Attributes(dn) {
		key, value, found := strings.Cut(attr, "=")
		if !found {
			continue
		}
		applyDNAttribute(&details, &ouParts, key, unquoteX500Value(strings.TrimSpace(value)))
	}
	details.OrganizationalUnit = strings.Join(ouParts, "+OU=")

	return &details, nil
}

// splitX500Attributes splits an X.500 distinguished name into its individual
// `key=value` attributes, treating both `,` (RDN separator) and `+`
// (multi-valued RDN separator) as delimiters but ignoring any delimiter that
// appears inside a double-quoted value.
func splitX500Attributes(dn string) []string {
	var attrs []string
	var buf strings.Builder
	inQuotes := false
	for i := 0; i < len(dn); i++ {
		c := dn[i]
		switch {
		case c == '"':
			inQuotes = !inQuotes
			buf.WriteByte(c)
		case (c == ',' || c == '+') && !inQuotes:
			attrs = append(attrs, buf.String())
			buf.Reset()
		default:
			buf.WriteByte(c)
		}
	}
	if buf.Len() > 0 {
		attrs = append(attrs, buf.String())
	}
	return attrs
}

// unquoteX500Value removes the surrounding double quotes that CERT_X500_NAME_STR
// adds to a value containing special characters, and un-doubles any escaped
// quote inside it. A value without surrounding quotes is returned unchanged.
func unquoteX500Value(v string) string {
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		v = v[1 : len(v)-1]
		v = strings.ReplaceAll(v, `""`, `"`)
	}
	return v
}

// DecodeHexEscapes replaces literal \xHH escape sequences with the actual byte values.
// For example, the string `\xD0\x90` (8 ASCII characters) becomes the 2-byte UTF-8 sequence for the Cyrillic letter "А".
// Returns the original string unchanged if no escape sequences are found.
// Incomplete or invalid sequences (e.g. `\xZZ`, `\x` at end of string) are left as-is.
func DecodeHexEscapes(s string) string {
	if !strings.Contains(s, `\x`) {
		return s
	}

	var buf strings.Builder
	buf.Grow(len(s))
	i := 0
	for i < len(s) {
		if i+3 < len(s) && s[i] == '\\' && s[i+1] == 'x' {
			b, err := hex.DecodeString(s[i+2 : i+4])
			if err == nil {
				buf.Write(b)
				i += 4
				continue
			}
		}
		buf.WriteByte(s[i])
		i++
	}
	return buf.String()
}

// DecodeUnicodeEscapes replaces literal \uXXXX escape sequences (including UTF-16 surrogate pairs) with actual Unicode
// characters. For example, the string `\ud83d\udda8` (12 ASCII characters) becomes the 4-byte UTF-8 sequence for 🖨.
// Returns the original string unchanged if no escape sequences are found.
// Incomplete or invalid sequences (e.g. `\u00`, `\u` at end of string) are left as-is.
// Unpaired UTF-16 surrogates are also left as-is.
func DecodeUnicodeEscapes(s string) string {
	if !strings.Contains(s, `\u`) {
		return s
	}

	var buf strings.Builder
	buf.Grow(len(s))
	i := 0
	for i < len(s) {
		r, size := parseUnicodeEscape(s, i)
		if size > 0 {
			buf.WriteRune(r)
			i += size
		} else {
			buf.WriteByte(s[i])
			i++
		}
	}
	return buf.String()
}

// parseUnicodeEscape attempts to parse a \uXXXX sequence at position i in s. If the parsed value is a UTF-16 high
// surrogate, it looks for an adjacent low surrogate to form a complete code point. Returns the decoded rune and the
// number of bytes consumed, or (0, 0) if no valid \uXXXX escape was found at position i.
func parseUnicodeEscape(s string, i int) (rune, int) {
	hi, ok := parseHex4(s, i)
	if !ok {
		return 0, 0
	}
	// If it's a UTF-16 surrogate, handle pairing or leave as-is.
	if hi >= 0xD800 && hi <= 0xDBFF {
		lo, ok := parseHex4(s, i+6)
		if ok && lo >= 0xDC00 && lo <= 0xDFFF {
			return 0x10000 + (hi-0xD800)*0x400 + (lo - 0xDC00), 12
		}
		// Unpaired high surrogate — leave as-is.
		return 0, 0
	}
	if hi >= 0xDC00 && hi <= 0xDFFF {
		// Standalone low surrogate — leave as-is.
		return 0, 0
	}
	return hi, 6
}

// parseHex4 tries to parse a \uXXXX sequence at position i and returns the 16-bit code unit.
func parseHex4(s string, i int) (rune, bool) {
	if i+6 > len(s) || s[i] != '\\' || s[i+1] != 'u' {
		return 0, false
	}
	val, err := strconv.ParseUint(s[i+2:i+6], 16, 16)
	if err != nil {
		return 0, false
	}
	return rune(val), true
}

func firstOrEmpty(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}
