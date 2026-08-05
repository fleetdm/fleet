// Package sigverify implements installer signature verification for the
// Fleet-maintained apps pipeline. It is shared by cmd/maintained-apps/verify
// (ingest-stage checks) and cmd/maintained-apps/validate (validator-stage
// checks on real macOS/Windows runners).
//
// The helpers shell out to platform tooling (osslsigncode, codesign, pkgutil,
// spctl, hdiutil); callers are responsible for only invoking the checks that
// exist in their environment.
package sigverify

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
)

// AuthenticodeResult is the outcome of an osslsigncode Authenticode
// verification. osslsigncode validates the signature and the embedded chain
// off-Windows; it does not check revocation against the Windows trust store —
// the validator's Get-AuthenticodeSignature remains the authoritative Windows
// check.
type AuthenticodeResult struct {
	// Available is false when osslsigncode is not installed.
	Available bool
	// Verified is true when osslsigncode's overall verdict was "Signature
	// verification: ok" for every signature in the file.
	Verified bool
	// NoSignature is true when the file carries no Authenticode signature.
	NoSignature bool
	// DigestMismatch is true when the message digest calculated from the file
	// does not match the digest inside the signature: the bytes are not what
	// the publisher signed, regardless of whether the certificate chain is
	// trusted on this host. This is the tamper signal an unverified chain is
	// not — osslsigncode verifies chains against the host's TLS CA bundle,
	// which lacks many Windows-only roots (e.g. Microsoft's Azure Trusted
	// Signing roots), so Verified=false with DigestMismatch=false is expected
	// for plenty of genuinely signed installers off-Windows.
	DigestMismatch bool
	// SubjectCNs are the leaf subject CommonNames of the signer(s).
	SubjectCNs []string
	// Detail carries a short failure description for reporting.
	Detail string
}

// VerifyAuthenticode verifies the Authenticode signature of an EXE or MSI
// installer using osslsigncode, which works on any OS. The result's
// Available field is false when osslsigncode is not installed.
func VerifyAuthenticode(ctx context.Context, installerPath string) *AuthenticodeResult {
	bin, err := exec.LookPath("osslsigncode")
	if err != nil {
		return &AuthenticodeResult{Available: false}
	}

	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	// osslsigncode exits non-zero for unsigned files and failed
	// verifications; both are results, not errors, so inspect the output.
	out, _ := exec.CommandContext(ctx, bin, "verify", installerPath).CombinedOutput()
	res := ParseOsslsigncodeOutput(string(out))
	res.Available = true
	return res
}

// signerSubjectPattern matches Subject lines within the "Signer's
// certificate:" block, e.g.:
//
//	Signer #0:
//		Subject: /C=US/ST=CA/O=Box, Inc./CN=Box, Inc.
var signerSubjectPattern = regexp.MustCompile(`(?m)^\s*Subject:\s*(.+)$`)

// ParseOsslsigncodeOutput parses `osslsigncode verify` output.
func ParseOsslsigncodeOutput(out string) *AuthenticodeResult {
	res := &AuthenticodeResult{}

	lower := strings.ToLower(out)
	if strings.Contains(lower, "no signature found") ||
		strings.Contains(lower, "file has no signature") {
		res.NoSignature = true
		res.Detail = "no Authenticode signature"
		return res
	}

	// The overall verdict must be matched as an exact line: a substring match
	// would be satisfied by the countersignature's "Timestamp Server Signature
	// verification: ok" line even when the verdict itself is "failed". Files
	// can carry multiple signatures, each with its own verdict line — all of
	// them must be ok.
	sawOK, sawFailed := false, false
	for line := range strings.Lines(out) {
		switch strings.TrimSpace(line) {
		case "Signature verification: ok":
			sawOK = true
		case "Signature verification: failed":
			sawFailed = true
		}
	}
	res.Verified = sawOK && !sawFailed
	if !res.Verified {
		// Prefer a failing verification line ("Timestamp Server Signature
		// verification: failed" sorts first when present, which is the more
		// specific cause) over e.g. a passing timestamp line.
		res.Detail = firstLineContaining(out, "Signature verification: failed")
		if res.Detail == "" {
			res.Detail = firstLineContaining(out, "Signature verification")
		}
		if res.Detail == "" {
			res.Detail = "osslsigncode did not report a successful verification"
		}
	}

	res.DigestMismatch = digestMismatch(out)

	// Only take Subject lines from the "Signer's certificate:" section: the
	// sections that follow (chain listing, countersignatures, timestamp and
	// CRL verification) repeat Subject lines for CA and TSA certificates,
	// which are not the signer.
	inSigner := false
	seen := make(map[string]struct{})
	for line := range strings.Lines(out) {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "Signer's certificate:"):
			inSigner = true
			continue
		case strings.HasPrefix(trimmed, "Message digest algorithm"),
			strings.HasPrefix(trimmed, "Number of certificates"),
			strings.HasPrefix(trimmed, "Authenticated attributes:"),
			strings.HasPrefix(trimmed, "Unauthenticated attributes:"),
			strings.HasPrefix(trimmed, "Countersignatures:"),
			strings.HasPrefix(trimmed, "Timestamp verified using:"),
			strings.HasPrefix(trimmed, "Certificate Revocation List"),
			strings.HasPrefix(trimmed, "CAfile:"):
			inSigner = false
			continue
		}
		if !inSigner {
			continue
		}
		m := signerSubjectPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if cn := SubjectCNFromDN(m[1]); cn != "" {
			if _, dup := seen[cn]; !dup {
				seen[cn] = struct{}{}
				res.SubjectCNs = append(res.SubjectCNs, cn)
			}
		}
	}

	return res
}

// digestLabelPairs are the "stored vs recomputed" line-label pairs
// osslsigncode prints, which vary by file format: PE files report "message
// digest" lines, MSI files report "DigitalSignature" (plus
// "MsiDigitalSignatureEx" when that stream is present).
var digestLabelPairs = [][2]string{
	{"Current message digest", "Calculated message digest"},
	{"Current DigitalSignature", "Calculated DigitalSignature"},
	{"Current MsiDigitalSignatureEx", "Calculated MsiDigitalSignatureEx"},
}

// digestMismatch reports whether any signature's stored digest differs from
// the digest osslsigncode recomputed from the file's bytes — the
// trust-store-independent tamper signal. Each label pair is compared
// positionally (the pairs are printed per signature, in order, and always
// adjacent in every known osslsigncode output format). osslsigncode also
// flags page-hash and digest mismatches with a "MISMATCH" marker in some
// output paths; treat either signal as a mismatch.
func digestMismatch(out string) bool {
	for _, pair := range digestLabelPairs {
		var current, calculated []string
		for line := range strings.Lines(out) {
			trimmed := strings.TrimSpace(line)
			if value, ok := lineValue(trimmed, pair[0]); ok {
				current = append(current, value)
			} else if value, ok := lineValue(trimmed, pair[1]); ok {
				calculated = append(calculated, value)
			}
		}
		for i, c := range current {
			if i >= len(calculated) {
				break
			}
			if !strings.EqualFold(c, calculated[i]) {
				return true
			}
		}
	}
	return strings.Contains(out, "MISMATCH")
}

// lineValue returns the value of a "key : value" line when the line starts
// with the given key.
func lineValue(line, key string) (string, bool) {
	rest, ok := strings.CutPrefix(line, key)
	if !ok {
		return "", false
	}
	rest = strings.TrimSpace(rest)
	value, ok := strings.CutPrefix(rest, ":")
	if !ok {
		return "", false
	}
	return strings.TrimSpace(value), true
}

// dnAttributePattern matches the start of a '/KEY=' attribute in an
// OpenSSL-style subject DN.
var dnAttributePattern = regexp.MustCompile(`/[A-Za-z][A-Za-z0-9.]*=`)

// SubjectCNFromDN extracts the CommonName from a subject DN in either of the
// formats osslsigncode has printed across versions:
//   - OpenSSL oneline: "/C=US/ST=California/O=Box, Inc./CN=Box, Inc."
//   - RFC 2253 (osslsigncode 2.x): "CN=Box\, Inc.,O=Box\, Inc.,C=US"
func SubjectCNFromDN(dn string) string {
	if idx := strings.LastIndex(dn, "/CN="); idx >= 0 {
		rest := dn[idx+len("/CN="):]
		// Cut at the start of the next attribute ("/KEY="), if any. A plain
		// '/' inside the CN value (rare, e.g. "OpenVPN Inc./US") does not
		// start a new attribute unless followed by KEY=.
		if next := dnAttributePattern.FindStringIndex(rest); next != nil {
			rest = rest[:next[0]]
		}
		return strings.TrimSpace(rest)
	}

	// RFC 2253: components separated by commas, with '\' escaping literal
	// commas (and other special characters) inside values.
	for _, component := range splitEscapedDN(dn, '\\') {
		component = strings.TrimSpace(component)
		if value, ok := strings.CutPrefix(component, "CN="); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// SubjectCNFromX500DN extracts the CommonName from a .NET
// X500DistinguishedName string as produced by Get-AuthenticodeSignature's
// SignerCertificate.Subject, e.g.:
//
//	CN="Box, Inc.", O="Box, Inc.", L=Redwood City, S=California, C=US
//
// Values containing special characters are wrapped in double quotes.
func SubjectCNFromX500DN(dn string) string {
	var components []string
	var current strings.Builder
	inQuotes := false
	for _, r := range dn {
		switch {
		case r == '"':
			inQuotes = !inQuotes
		case r == ',' && !inQuotes:
			components = append(components, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	components = append(components, current.String())

	for _, component := range components {
		component = strings.TrimSpace(component)
		if value, ok := strings.CutPrefix(component, "CN="); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// splitEscapedDN splits a DN on unescaped commas, unescaping backslash
// sequences as it goes.
func splitEscapedDN(dn string, escapeChar rune) []string {
	var components []string
	var current strings.Builder
	escaped := false
	for _, r := range dn {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == escapeChar:
			escaped = true
		case r == ',':
			components = append(components, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	return append(components, current.String())
}

// firstLineContaining returns the first line of out containing the given
// substring, for concise error details.
func firstLineContaining(out, contains string) string {
	for line := range strings.Lines(out) {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && strings.Contains(trimmed, contains) {
			return trimmed
		}
	}
	return ""
}
