#!/bin/bash
# remediate-emudhra-ca-compromise-macos.sh
# Remove trust in the roots behind the compromised eMudhra/emSign intermediates (Oct 2026).
# Apple's trustd block (Safari, system TLS) landed on 2026-10-19. This script fixes the
# trust stores that block does NOT cover:
#   1. /etc/ssl/cert.pem  - used by Apple's curl, git, python3, ruby, perl, openssl.
#   2. /opt/orbit/certs.pem - fleetd's bundle that osqueryd uses for TLS to Fleet
#      (changed only after proving the Fleet server still verifies).
#   3. Homebrew ca-certificates and certifi formulae (upgraded as the Homebrew owner).
# Idempotent: safe to re-run. Built for Fleet "Run script" (runs as root).
# Save it as a Fleet script: it is larger than the 10,000-character ad hoc limit.
# Rollback: copy a *.pre-emudhra backup over the file it was taken from.
# Exit: 0 = protected, 2 = still exposed or a step was skipped (read output), 1 = error.
# Source: https://emudhra.we-have-a-problem.com/

set -u
export PATH="/usr/bin:/bin:/usr/sbin:/sbin"

# SHA-1 of: emSign ECC Root CA - G3, AAA Certificate Services
BLOCKED_SHA1="3043fa4ff257dca0c380ee2e58ea78b23fe6bbc1 d1eb23a46d17d68fd92564c2f1f1601764d8e349"
SYSTEM_PEM="/etc/ssl/cert.pem"
ORBIT_BUNDLE="/opt/orbit/certs.pem"
ORBIT_PLIST="/Library/LaunchDaemons/com.fleetdm.orbit.plist"
OPENSSL="/usr/bin/openssl"
EXIT=0

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
        fp=$("$OPENSSL" x509 -in "$f" -noout -fingerprint -sha1 2>/dev/null | cut -d= -f2 | tr -d ':' | tr 'A-F' 'a-f')
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
        fp=$("$OPENSSL" x509 -in "$f" -noout -fingerprint -sha1 2>/dev/null | cut -d= -f2 | tr -d ':' | tr 'A-F' 'a-f')
        case " $BLOCKED_SHA1 " in *" $fp "*) [ -n "$fp" ] && skip="$skip $idx" ;; esac
    done
    rm -rf "$work"
    awk -v skip="$skip " 'BEGIN{n=split(skip, a, " "); for (i = 1; i <= n; i++) drop[a[i]] = 1}
        { buf = buf $0 "\n" }
        /-----END CERTIFICATE-----/ { c++; if (!(c in drop)) printf "%s", buf; buf = "" }
        END { printf "%s", buf }' "$in" > "$out"
}

# Key identifiers as uppercase hex without colons (LibreSSL and OpenSSL 3 text output).
ski_of() { "$OPENSSL" x509 -in "$1" -noout -text 2>/dev/null | awk '/X509v3 Subject Key Identifier/{getline; gsub(/[ :]/, ""); print toupper($0); exit}'; }
aki_of() { "$OPENSSL" x509 -in "$1" -noout -text 2>/dev/null | awk '/X509v3 Authority Key Identifier/{getline; sub(/.*keyid:/, ""); gsub(/[ :]/, ""); print toupper($0); exit}'; }

# 0 if the TLS chain served at HOST:PORT is issued by, or contains, a certificate in BUNDLE.
# Apple's curl and LibreSSL always fall back to the system trust store, so they cannot
# prove that a bundle works on its own. This compares key identifiers instead.
chain_anchored_in() {
    local bundle="$1" host="$2" port="$3" work f k skis="" rc=1
    work=$(mktemp -d) || return 1
    awk -v d="$work" '/-----BEGIN CERTIFICATE-----/{n++; f=sprintf("%s/b%05d.pem", d, n)} f{print > f} /-----END CERTIFICATE-----/{close(f); f=""}' "$bundle"
    for f in "$work"/b*.pem; do [ -e "$f" ] && skis="$skis $(ski_of "$f")"; done
    /usr/bin/perl -e 'alarm 20; exec @ARGV' "$OPENSSL" s_client -connect "$host:$port" -servername "$host" -showcerts </dev/null 2>/dev/null |
        awk -v d="$work" '/-----BEGIN CERTIFICATE-----/{n++; f=sprintf("%s/s%02d.pem", d, n)} f{print > f} /-----END CERTIFICATE-----/{close(f); f=""}'
    for f in "$work"/s*.pem; do
        [ -e "$f" ] || continue
        for k in $(aki_of "$f") $(ski_of "$f"); do
            case " $skis " in *" $k "*) rc=0 ;; esac
        done
    done
    rm -rf "$work"
    return $rc
}

# grep -c prints 0 and exits 1 on no match, so '|| echo 0' would print a second 0 and
# break the integer test. Capture once and fall back to 0 so empty bundles fail closed.
cert_count() { local n; n=$(grep -c -- '-----BEGIN CERTIFICATE-----' "$1" 2>/dev/null); echo "${n:-0}"; }

# Swap TARGET for the filtered TMP atomically, keeping owner/mode and a one-time backup.
swap_in() {
    local target="$1" tmp="$2"
    [ -e "$target.pre-emudhra" ] || cp -p "$target" "$target.pre-emudhra" || return 1
    cp -p "$target" "$target.new" && cat "$tmp" > "$target.new" && mv -f "$target.new" "$target"
}

fix_system_pem() {
    local hits tmp
    hits=$(blocked_in_bundle "$SYSTEM_PEM")
    [ -z "$hits" ] && { log "SYSTEM: $SYSTEM_PEM already clean"; return; }
    tmp=$(mktemp) || die "mktemp failed"
    filter_bundle "$SYSTEM_PEM" "$tmp"
    if [ -n "$(blocked_in_bundle "$tmp")" ] || [ "$(cert_count "$tmp")" -lt 50 ]; then
        rm -f "$tmp"; warn "SYSTEM: filtered bundle failed sanity check; not changed"; return
    fi
    if swap_in "$SYSTEM_PEM" "$tmp"; then
        log "SYSTEM: removed $(echo $hits | wc -w | tr -d ' ') root(s) from $SYSTEM_PEM (backup: $SYSTEM_PEM.pre-emudhra)"
        log "SYSTEM: note - a macOS update may restore this file; re-run this script after OS updates"
    else
        warn "SYSTEM: could not write $SYSTEM_PEM"
    fi
    rm -f "$tmp"
}

plist_env() { /usr/libexec/PlistBuddy -c "Print :EnvironmentVariables:$1" "$ORBIT_PLIST" 2>/dev/null; }

fleet_url() {
    local u
    u=$(plist_env ORBIT_FLEET_URL)
    [ -z "$u" ] && [ -r /opt/orbit/fleet_url.txt ] && u=$(tr -d '[:space:]' < /opt/orbit/fleet_url.txt)
    echo "$u"
}

# Only the plist decides: a leftover /opt/orbit/fleet.pem is ignored by osquery when
# ORBIT_FLEET_CERTIFICATE is not set (for example after fleetd-base.pkg is reinstalled).
fleet_pin() { plist_env ORBIT_FLEET_CERTIFICATE; }

fix_orbit_bundle() {
    local hits url pin tmp hostport host port
    [ -f "$ORBIT_BUNDLE" ] || { log "FLEETD: $ORBIT_BUNDLE not present (fleetd not installed)"; return; }
    hits=$(blocked_in_bundle "$ORBIT_BUNDLE")
    [ -z "$hits" ] && { log "FLEETD: $ORBIT_BUNDLE already clean"; return; }
    tmp=$(mktemp) || die "mktemp failed"
    filter_bundle "$ORBIT_BUNDLE" "$tmp"
    if [ -n "$(blocked_in_bundle "$tmp")" ] || [ "$(cert_count "$tmp")" -lt 50 ]; then
        rm -f "$tmp"; warn "FLEETD: filtered bundle failed sanity check; not changed"; return
    fi
    url=$(fleet_url); pin=$(fleet_pin)
    if [ -n "$pin" ]; then
        log "FLEETD: osquery pins $pin, so certs.pem is unused; cleaning it anyway"
    elif [ -n "$url" ]; then
        # Prove the Fleet server's chain still anchors in the filtered bundle before swapping it in.
        hostport=${url#*://}; hostport=${hostport%%/*}
        host=${hostport%%:*}; port=443
        [ "$host" != "$hostport" ] && port=${hostport##*:}
        if ! chain_anchored_in "$tmp" "$host" "$port"; then
            rm -f "$tmp"; warn "FLEETD: Fleet server chain at $host:$port does not anchor in the filtered bundle; not changed"; return
        fi
        log "FLEETD: Fleet server chain at $host:$port anchors in the filtered bundle"
    else
        rm -f "$tmp"; warn "FLEETD: Fleet URL not found; cannot prove safety; not changed"; return
    fi
    if swap_in "$ORBIT_BUNDLE" "$tmp"; then
        log "FLEETD: removed $(echo $hits | wc -w | tr -d ' ') root(s) from $ORBIT_BUNDLE (backup: $ORBIT_BUNDLE.pre-emudhra). osquery reloads it on its next connection."
    else
        warn "FLEETD: could not write $ORBIT_BUNDLE"
    fi
    rm -f "$tmp"
}

fix_homebrew() {
    local brew owner f
    for brew in /opt/homebrew/bin/brew /usr/local/bin/brew; do [ -x "$brew" ] && break; brew=""; done
    [ -z "$brew" ] && { log "HOMEBREW: not installed"; return; }
    # /usr/local is root-owned on Intel Macs; the Homebrew repository is owned by the brew user.
    owner=$(dirname "$(dirname "$brew")")
    [ -d "$owner/Homebrew" ] && owner="$owner/Homebrew"
    owner=$(stat -f %Su "$owner")
    run_brew() {
        # sudo resets the environment, so pass Homebrew settings through env explicitly.
        if [ "$(id -un)" = "$owner" ]; then (cd /tmp && env HOMEBREW_NO_INSTALLED_DEPENDENTS_CHECK=1 "$brew" "$@")
        else (cd /tmp && sudo -u "$owner" -H env HOMEBREW_NO_INSTALLED_DEPENDENTS_CHECK=1 "$brew" "$@"); fi
    }
    for f in ca-certificates certifi; do
        if run_brew list --versions "$f" >/dev/null 2>&1; then
            run_brew upgrade "$f" >/dev/null 2>&1 || true
            log "HOMEBREW: $(run_brew list --versions "$f")"
        fi
    done
    for f in node go; do
        run_brew list --versions "$f" >/dev/null 2>&1 && \
            log "HOMEBREW: $(run_brew list --versions "$f") - fixed: node 22.23.4/24.22.1/26.12.1, go 1.27.3/1.26.10 (not auto-upgraded)"
    done
}

main() {
    [ -w "$SYSTEM_PEM" ] || die "must run as root (cannot write $SYSTEM_PEM)"
    [ -x "$OPENSSL" ] || die "$OPENSSL not found"

    log "----- Apple CLI trust bundle -----"
    fix_system_pem
    log "----- fleetd osquery CA bundle -----"
    fix_orbit_bundle
    log "----- Homebrew -----"
    fix_homebrew

    log "----- verification -----"
    [ -z "$(blocked_in_bundle "$SYSTEM_PEM")" ] && log "VERIFY: $SYSTEM_PEM PROTECTED" || warn "VERIFY: $SYSTEM_PEM still trusts a compromised root"
    if [ -f "$ORBIT_BUNDLE" ]; then
        [ -z "$(blocked_in_bundle "$ORBIT_BUNDLE")" ] && log "VERIFY: $ORBIT_BUNDLE PROTECTED" || warn "VERIFY: $ORBIT_BUNDLE still trusts a compromised root"
    fi
    log "Restart long-running CLI daemons. Update Node.js and pip-installed certifi in virtualenvs separately."
    exit "$EXIT"
}

[ "${BASH_SOURCE[0]}" = "$0" ] && main "$@"
