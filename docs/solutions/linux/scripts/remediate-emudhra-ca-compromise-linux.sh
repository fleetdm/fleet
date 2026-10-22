#!/bin/bash
# remediate-emudhra-ca-compromise-linux.sh
# Remove trust in the roots behind the compromised eMudhra/emSign intermediates (Oct 2026).
#   1. Upgrade the distro ca-certificates package (apt, dnf, yum, zypper, apk).
#   2. If the system bundle still trusts a compromised root, distrust it the distro-native way.
#   3. Strip the same roots from fleetd's private CA bundle (/opt/orbit/certs.pem), which
#      osqueryd uses for its TLS connection to Fleet. Only after proving Fleet still verifies.
# Idempotent: safe to re-run. Built for Fleet "Run script" (runs as root). Needs bash and openssl.
# Save it as a Fleet script: it is larger than the 10,000-character ad hoc limit.
# Rollback: copy back the *.pre-emudhra backups, delete emudhra-*.pem from the p11-kit
# blocklist directory, then re-run update-ca-certificates or update-ca-trust.
# Exit: 0 = protected, 2 = still exposed or a step was skipped (read output), 1 = error.
# Source: https://emudhra.we-have-a-problem.com/

set -u
export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
export DEBIAN_FRONTEND=noninteractive

# SHA-1 of: emSign ECC Root CA - G3, AAA Certificate Services
BLOCKED_SHA1="3043fa4ff257dca0c380ee2e58ea78b23fe6bbc1 d1eb23a46d17d68fd92564c2f1f1601764d8e349"
ORBIT_BUNDLE="/opt/orbit/certs.pem"
ORBIT_ENV="/etc/default/orbit"
EXIT=0

read -r -d '' EMSIGN_ECC_G3_PEM <<'PEM' || true
-----BEGIN CERTIFICATE-----
MIICTjCCAdOgAwIBAgIKPPYHqWhwDtqLhDAKBggqhkjOPQQDAzBrMQswCQYDVQQG
EwJJTjETMBEGA1UECxMKZW1TaWduIFBLSTElMCMGA1UEChMcZU11ZGhyYSBUZWNo
bm9sb2dpZXMgTGltaXRlZDEgMB4GA1UEAxMXZW1TaWduIEVDQyBSb290IENBIC0g
RzMwHhcNMTgwMjE4MTgzMDAwWhcNNDMwMjE4MTgzMDAwWjBrMQswCQYDVQQGEwJJ
TjETMBEGA1UECxMKZW1TaWduIFBLSTElMCMGA1UEChMcZU11ZGhyYSBUZWNobm9s
b2dpZXMgTGltaXRlZDEgMB4GA1UEAxMXZW1TaWduIEVDQyBSb290IENBIC0gRzMw
djAQBgcqhkjOPQIBBgUrgQQAIgNiAAQjpQy4LRL1KPOxst3iAhKAnjlfSU2fySU0
WXTsuwYc58Byr+iuL+FBVIcUqEqy6HyC5ltqtdyzdc6LBtCGI79G1Y4PPwT01xyS
fvalY8L1X44uT6EYGQIrMgqCZH0Wk9GjQjBAMB0GA1UdDgQWBBR8XQKEE9TMipuB
zhccLikenEhjQjAOBgNVHQ8BAf8EBAMCAQYwDwYDVR0TAQH/BAUwAwEB/zAKBggq
hkjOPQQDAwNpADBmAjEAvvNhzwIQHWSVB7gYboiFBS+DCBeQyh+KTOgNG3qxrdWB
CUfvO6wIBHxcmbHtRwfSAjEAnbpV/KlK6O3t5nYBQnvI+GDZjVGLVTv7jHvrZQnD
+JbNR6iC8hZVdyR+EhCVBCyj
-----END CERTIFICATE-----
PEM
read -r -d '' AAA_CERT_SERVICES_PEM <<'PEM' || true
-----BEGIN CERTIFICATE-----
MIIEMjCCAxqgAwIBAgIBATANBgkqhkiG9w0BAQUFADB7MQswCQYDVQQGEwJHQjEb
MBkGA1UECAwSR3JlYXRlciBNYW5jaGVzdGVyMRAwDgYDVQQHDAdTYWxmb3JkMRow
GAYDVQQKDBFDb21vZG8gQ0EgTGltaXRlZDEhMB8GA1UEAwwYQUFBIENlcnRpZmlj
YXRlIFNlcnZpY2VzMB4XDTA0MDEwMTAwMDAwMFoXDTI4MTIzMTIzNTk1OVowezEL
MAkGA1UEBhMCR0IxGzAZBgNVBAgMEkdyZWF0ZXIgTWFuY2hlc3RlcjEQMA4GA1UE
BwwHU2FsZm9yZDEaMBgGA1UECgwRQ29tb2RvIENBIExpbWl0ZWQxITAfBgNVBAMM
GEFBQSBDZXJ0aWZpY2F0ZSBTZXJ2aWNlczCCASIwDQYJKoZIhvcNAQEBBQADggEP
ADCCAQoCggEBAL5AnfRu4ep2hxxNRUSOvkbIgwadwSr+GB+O5AL686tdUIoWMQua
BtDFcCLNSS1UY8y2bmhGC1Pqy0wkwLxyTurxFa70VJoSCsN6sjNg4tqJVfMiWPPe
3M/vg4aijJRPn2jymJBGhCfHdr/jzDUsi14HZGWCwEiwqJH5YZ92IFCokcdmtet4
YgNW8IoaE+oxox6gmf049vYnMlhvB/VruPsUK6+3qszWY19zjNoFmag4qMsXeDZR
rOme9Hg6jc8P2ULimAyrL58OAd7vn5lJ8S3frHRNG5i1R8XlKdH5kBjHYpy+g8cm
ez6KJcfA3Z3mNWgQIJ2P2N7Sw4ScDV7oL8kCAwEAAaOBwDCBvTAdBgNVHQ4EFgQU
oBEKIz6W8Qfs4q8p74Klf9AwpLQwDgYDVR0PAQH/BAQDAgEGMA8GA1UdEwEB/wQF
MAMBAf8wewYDVR0fBHQwcjA4oDagNIYyaHR0cDovL2NybC5jb21vZG9jYS5jb20v
QUFBQ2VydGlmaWNhdGVTZXJ2aWNlcy5jcmwwNqA0oDKGMGh0dHA6Ly9jcmwuY29t
b2RvLm5ldC9BQUFDZXJ0aWZpY2F0ZVNlcnZpY2VzLmNybDANBgkqhkiG9w0BAQUF
AAOCAQEACFb8AvCb6P+k+tZ7xkSAzk/ExfYAWMymtrwUSWgEdujm7l3sAg9g1o1Q
GE8mTgHj5rCl7r+8dFRBv/38ErjHT1r0iWAFf2C3BUrz9vHCv8S5dIa2LX1rzNLz
Rt0vxuBqw8M0Ayx9lt1awg6nCpnBBYurDC/zXDrPbDdVCYfeU0BsWO/8tqtlbgT2
G9w84FoVxp7Z8VlIMCFlA2zs6SFz7JsDoeA3raAVGI/6ugLOpyypEBMs1OUIJqsi
l2D4kF501KKaU73yqWjgom7C12yxow+ev+to51byrvLjKzg6CYG1a4XXvi3tPxq3
smPi9WIsgtRqAEFQ8TmDn5XpNpaYbg==
-----END CERTIFICATE-----
PEM

log()  { echo "$*"; }
warn() { echo "WARN: $*"; [ "$EXIT" -eq 0 ] && EXIT=2; }
die()  { echo "ERROR: $*" >&2; exit 1; }

# Print the SHA-1 (lowercase, no colons) of every blocked root found in a PEM bundle.
blocked_in_bundle() {
    local bundle="$1" work f fp
    [ -r "$bundle" ] || return 0
    work=$(mktemp -d) || return 0
    awk -v d="$work" '/-----BEGIN CERTIFICATE-----/{n++; f=sprintf("%s/%05d.pem", d, n)} f{print > f} /-----END CERTIFICATE-----/{close(f); f=""}' "$bundle"
    for f in "$work"/*.pem; do
        [ -e "$f" ] || continue
        fp=$(openssl x509 -in "$f" -noout -fingerprint -sha1 2>/dev/null | cut -d= -f2 | tr -d ':' | tr 'A-F' 'a-f')
        case " $BLOCKED_SHA1 " in *" $fp "*) [ -n "$fp" ] && printf '%s ' "$fp" ;; esac
    done
    rm -rf "$work"
}

# Copy bundle IN to OUT, dropping each blocked cert together with the comment lines above it.
filter_bundle() {
    local in="$1" out="$2" work f fp idx=0 skip=""
    work=$(mktemp -d) || return 1
    awk -v d="$work" '/-----BEGIN CERTIFICATE-----/{n++; f=sprintf("%s/%05d.pem", d, n)} f{print > f} /-----END CERTIFICATE-----/{close(f); f=""}' "$in"
    for f in "$work"/*.pem; do
        [ -e "$f" ] || continue
        idx=$((idx + 1))
        fp=$(openssl x509 -in "$f" -noout -fingerprint -sha1 2>/dev/null | cut -d= -f2 | tr -d ':' | tr 'A-F' 'a-f')
        case " $BLOCKED_SHA1 " in *" $fp "*) [ -n "$fp" ] && skip="$skip $idx" ;; esac
    done
    rm -rf "$work"
    awk -v skip="$skip " 'BEGIN{n=split(skip, a, " "); for (i = 1; i <= n; i++) drop[a[i]] = 1}
        { buf = buf $0 "\n" }
        /-----END CERTIFICATE-----/ { c++; if (!(c in drop)) printf "%s", buf; buf = "" }
        END { printf "%s", buf }' "$in" > "$out"
}

cert_count() { grep -c -- '-----BEGIN CERTIFICATE-----' "$1" 2>/dev/null || echo 0; }

find_system_bundle() {
    local b
    for b in /etc/ssl/certs/ca-certificates.crt /etc/pki/tls/certs/ca-bundle.crt \
             /var/lib/ca-certificates/ca-bundle.pem /etc/ssl/cert.pem; do
        [ -s "$b" ] && { echo "$b"; return; }
    done
}

upgrade_package() {
    if command -v apt-get >/dev/null 2>&1; then
        apt-get update -qq >/dev/null 2>&1 || warn "apt-get update failed (repo or network issue)"
        apt-get install --only-upgrade -y -qq ca-certificates >/dev/null 2>&1 || warn "apt-get upgrade of ca-certificates failed"
        log "PACKAGE: ca-certificates $(dpkg-query -W -f='${Version}' ca-certificates 2>/dev/null)"
    elif command -v dnf >/dev/null 2>&1; then
        dnf -y -q upgrade ca-certificates >/dev/null 2>&1 || warn "dnf upgrade of ca-certificates failed"
        log "PACKAGE: $(rpm -q ca-certificates)"
    elif command -v yum >/dev/null 2>&1; then
        yum -y -q update ca-certificates >/dev/null 2>&1 || warn "yum update of ca-certificates failed"
        log "PACKAGE: $(rpm -q ca-certificates)"
    elif command -v zypper >/dev/null 2>&1; then
        zypper -n -q update ca-certificates-mozilla ca-certificates >/dev/null 2>&1 || warn "zypper update failed"
        log "PACKAGE: $(rpm -q ca-certificates-mozilla 2>/dev/null || rpm -q ca-certificates)"
    elif command -v apk >/dev/null 2>&1; then
        apk add -q --upgrade ca-certificates >/dev/null 2>&1 || warn "apk upgrade of ca-certificates failed"
        log "PACKAGE: $(apk info -v ca-certificates 2>/dev/null | head -1)"
    else
        warn "no supported package manager found; update ca-certificates manually"
    fi
}

pem_for() {
    case "$1" in
        3043fa4ff257dca0c380ee2e58ea78b23fe6bbc1) printf '%s\n' "$EMSIGN_ECC_G3_PEM" ;;
        d1eb23a46d17d68fd92564c2f1f1601764d8e349) printf '%s\n' "$AAA_CERT_SERVICES_PEM" ;;
    esac
}

# Distro-native distrust: survives future ca-certificates regenerations.
distrust_roots() {
    local fp dir
    if [ -f /etc/ca-certificates.conf ] && command -v update-ca-certificates >/dev/null 2>&1; then
        # Debian, Ubuntu, Alpine: prefix the entry with "!" so update-ca-certificates skips it.
        [ -e /etc/ca-certificates.conf.pre-emudhra ] || cp -p /etc/ca-certificates.conf /etc/ca-certificates.conf.pre-emudhra
        sed -i -e 's|^mozilla/emSign_ECC_Root_CA_-_G3\.crt$|!&|' \
               -e 's|^mozilla/Comodo_AAA_Services_root\.crt$|!&|' /etc/ca-certificates.conf
        update-ca-certificates >/dev/null 2>&1 || warn "update-ca-certificates failed"
        log "DISTRUST: marked roots with ! in /etc/ca-certificates.conf (backup: /etc/ca-certificates.conf.pre-emudhra)"
    elif command -v update-ca-trust >/dev/null 2>&1; then
        # Fedora, RHEL, Rocky, Alma, Arch (p11-kit): drop the root into the blocklist directory.
        if [ -d /etc/ca-certificates/trust-source ]; then
            dir=/etc/ca-certificates/trust-source/blocklist      # Arch
        else
            dir=/etc/pki/ca-trust/source/blocklist
            [ -d "$dir" ] || [ ! -d /etc/pki/ca-trust/source/blacklist ] || dir=/etc/pki/ca-trust/source/blacklist
        fi
        mkdir -p "$dir"
        for fp in "$@"; do pem_for "$fp" > "$dir/emudhra-$fp.pem"; done
        update-ca-trust extract >/dev/null 2>&1 || warn "update-ca-trust extract failed"
        log "DISTRUST: wrote roots to $dir"
    elif [ -d /etc/pki/trust ] && command -v update-ca-certificates >/dev/null 2>&1; then
        # openSUSE / SLES (p11-kit).
        mkdir -p /etc/pki/trust/blacklist
        for fp in "$@"; do pem_for "$fp" > "/etc/pki/trust/blacklist/emudhra-$fp.pem"; done
        update-ca-certificates >/dev/null 2>&1 || warn "update-ca-certificates failed"
        log "DISTRUST: wrote roots to /etc/pki/trust/blacklist"
    else
        warn "no known distrust mechanism on this distro"
    fi
}

env_value() { sed -n "s/^$1=//p" "$ORBIT_ENV" 2>/dev/null | tail -1 | tr -d '"'"'"; }

fix_orbit_bundle() {
    local hits url pin tmp rc nocapath
    [ -f "$ORBIT_BUNDLE" ] || { log "FLEETD: $ORBIT_BUNDLE not present (fleetd not installed)"; return; }
    hits=$(blocked_in_bundle "$ORBIT_BUNDLE")
    [ -z "$hits" ] && { log "FLEETD: $ORBIT_BUNDLE already clean"; return; }

    tmp=$(mktemp) || die "mktemp failed"
    filter_bundle "$ORBIT_BUNDLE" "$tmp"
    if [ -n "$(blocked_in_bundle "$tmp")" ] || [ "$(cert_count "$tmp")" -lt 50 ]; then
        rm -f "$tmp"; warn "FLEETD: filtered bundle failed sanity check; not changed"; return
    fi

    url=$(env_value ORBIT_FLEET_URL)
    pin=$(env_value ORBIT_FLEET_CERTIFICATE)
    if [ -n "$pin" ]; then
        log "FLEETD: osquery pins $pin, so certs.pem is unused; cleaning it anyway"
    elif [ -n "$url" ]; then
        # Prove the Fleet server still verifies with the filtered bundle before swapping it in.
        # --capath points at an empty directory so curl cannot fall back to the system store.
        nocapath=$(mktemp -d)
        curl -sS -o /dev/null --max-time 20 --cacert "$tmp" --capath "$nocapath" "$url" 2>/dev/null; rc=$?
        rmdir "$nocapath"
        if [ "$rc" -ne 0 ]; then
            rm -f "$tmp"; warn "FLEETD: Fleet server did not verify with the filtered bundle (curl exit $rc); not changed"; return
        fi
        log "FLEETD: Fleet server verifies with the filtered bundle"
    else
        rm -f "$tmp"; warn "FLEETD: no ORBIT_FLEET_URL in $ORBIT_ENV; cannot prove safety; not changed"; return
    fi

    if { [ -e "$ORBIT_BUNDLE.pre-emudhra" ] || cp -p "$ORBIT_BUNDLE" "$ORBIT_BUNDLE.pre-emudhra"; } &&
       cp -p "$ORBIT_BUNDLE" "$ORBIT_BUNDLE.new" && cat "$tmp" > "$ORBIT_BUNDLE.new" && mv -f "$ORBIT_BUNDLE.new" "$ORBIT_BUNDLE"; then
        rm -f "$tmp"
    else
        rm -f "$tmp" "$ORBIT_BUNDLE.new"; warn "FLEETD: could not write $ORBIT_BUNDLE"; return
    fi
    log "FLEETD: removed $(echo "$hits" | wc -w | tr -d ' ') root(s) from $ORBIT_BUNDLE (backup: $ORBIT_BUNDLE.pre-emudhra). osquery reloads it on its next connection."
}

main() {
    local bundle hits
    [ "$(id -u)" -eq 0 ] || die "must run as root"
    if ! command -v openssl >/dev/null 2>&1; then
        command -v apk >/dev/null 2>&1 && apk add -q openssl >/dev/null 2>&1
        command -v openssl >/dev/null 2>&1 || die "openssl CLI is required to fingerprint certificates"
    fi

    log "----- system trust store -----"
    upgrade_package
    bundle=$(find_system_bundle)
    [ -n "$bundle" ] || die "no system CA bundle found"
    hits=$(blocked_in_bundle "$bundle")
    if [ -n "$hits" ]; then
        log "BUNDLE: $bundle still trusts: $hits (package fix not available yet)"
        # shellcheck disable=SC2086
        distrust_roots $hits
        hits=$(blocked_in_bundle "$bundle")
    fi
    if [ -z "$hits" ]; then log "BUNDLE: $bundle PROTECTED"; else warn "BUNDLE: $bundle STILL TRUSTS $hits"; fi

    log "----- fleetd osquery CA bundle -----"
    fix_orbit_bundle
    if [ -f "$ORBIT_BUNDLE" ]; then
        if [ -z "$(blocked_in_bundle "$ORBIT_BUNDLE")" ]; then log "VERIFY: $ORBIT_BUNDLE PROTECTED"
        else warn "VERIFY: $ORBIT_BUNDLE still trusts a compromised root"; fi
    fi

    log "----- next steps -----"
    log "Restart long-running services (web apps, agents, Go/Python daemons): they read the bundle at start."
    log "Node.js, pip-installed certifi and containers carry their own CA copy: update or rebuild them."
    exit "$EXIT"
}

[ "${BASH_SOURCE[0]}" = "$0" ] && main "$@"
