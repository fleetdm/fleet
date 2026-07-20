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
	// Verified is true when osslsigncode reported "Signature verification: ok".
	Verified bool
	// NoSignature is true when the file carries no Authenticode signature.
	NoSignature bool
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

	res.Verified = strings.Contains(out, "Signature verification: ok")
	if !res.Verified {
		res.Detail = firstLineContaining(out, "Signature verification")
		if res.Detail == "" {
			res.Detail = "osslsigncode did not report a successful verification"
		}
	}

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
