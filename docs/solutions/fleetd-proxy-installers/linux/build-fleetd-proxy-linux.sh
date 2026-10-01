#!/bin/bash
# ============================================================================
# build-fleetd-proxy-linux.sh: build fleetd .deb and .rpm packages that take the Fleet URL,
# enroll secret and proxy at INSTALL time, for headless Linux hosts behind a forward proxy.
#
# Recommended: run it through ./build-linux-in-container.sh (Fedora container, nothing
# installed on the host). It also runs directly on Fedora/RHEL with dnf (needs root for dnf).
#
# It builds stock packages with `fleetctl package --type deb|rpm`, then changes ONLY their
# install scripts; the payload (binaries, unit file, /etc/default/orbit) is byte-for-byte stock.
# The added post-install block reads these from the installer's environment, falling back to
# KEY=value lines in /etc/default/fleetd-proxy (for tools that can drop a file but not set env):
#   FLEET_URL        e.g. https://fleet.example.com (optional if baked in at build time)
#   FLEET_SECRET     enroll secret (team secret picks the team)
#   ORBIT_PROXY_URL  e.g. http://proxy.example.internal:3128
#   OSQUERY_PROXY    host:port for osquery; optional, derived from ORBIT_PROXY_URL when unset
#   NO_PROXY         optional; defaults to localhost,127.0.0.1 when a proxy is set
# and writes them to /etc/default/orbit-proxy plus a systemd drop-in
# (/etc/systemd/system/orbit.service.d/10-fleetd-proxy.conf). Package-owned files are never
# edited, so package upgrades keep the settings. The osquery proxy is passed as
# `-- --proxy_hostname=...` on Orbit's command line, which overrides osquery.flags (that file
# can be rewritten by Fleet's agent options).
#
# Install example:
#   sudo env FLEET_SECRET=<team-secret> ORBIT_PROXY_URL=http://proxy.example.internal:3128 \
#     apt-get install -y ./fleetd-proxy_amd64.deb        # or: dnf install -y ./fleetd-proxy.x86_64.rpm
#
# Knobs (env vars):
#   FLEET_VER          fleetctl release                                  (default v4.91.1)
#   FLEET_URL          bake a default Fleet URL into the packages (optional; install-time value wins)
#   ORBIT_CHANNEL, OSQUERYD_CHANNEL   stable | edge | a version such as 1.60.0 / 5.23.1  (default stable)
#   ARCH               amd64 | arm64                                     (default amd64)
#   TYPES              "deb rpm" | deb | rpm                             (default "deb rpm")
#   OUT_DIR            output directory                                  (default ./fleetd-proxy-linux-out)
#   WORK               scratch directory                                 (default /tmp/fleetd-proxy-linux)
# ============================================================================
set -uo pipefail
FLEET_VER="${FLEET_VER:-v4.91.1}"
FLEET_URL_DEFAULT="${FLEET_URL:-}"
ORBIT_CHANNEL="${ORBIT_CHANNEL:-stable}"
OSQUERYD_CHANNEL="${OSQUERYD_CHANNEL:-stable}"
ARCH="${ARCH:-amd64}"
TYPES="${TYPES:-deb rpm}"
OUT_DIR="${OUT_DIR:-$PWD/fleetd-proxy-linux-out}"
WORK="${WORK:-/tmp/fleetd-proxy-linux}"
MARK="fleetd-proxy install-time configuration"
die() { echo "ERROR: $*" >&2; exit 1; }
mkdir -p "$WORK" "$OUT_DIR"; OUT_DIR="$(cd "$OUT_DIR" && pwd)"; cd "$WORK" || die "cannot cd to $WORK"

echo "== deps =="
need=""
for t in curl:curl dpkg-deb:dpkg rpm:rpm rpmbuild:rpm-build rpmrebuild:rpmrebuild cpio:cpio; do
  command -v "${t%%:*}" >/dev/null || need="$need ${t#*:}"
done
if [ -n "$need" ]; then
  command -v dnf >/dev/null || die "missing:$need (run via build-linux-in-container.sh, or on Fedora/RHEL with dnf)"
  dnf -y -q install $need >/dev/null || die "dnf install failed:$need"
fi

echo "== fleetctl $FLEET_VER =="
FC="$WORK/fleetctl-$FLEET_VER/fleetctl"
if [ ! -x "$FC" ]; then
  mkdir -p "$WORK/fleetctl-$FLEET_VER"
  curl -fsSL "https://github.com/fleetdm/fleet/releases/download/fleet-${FLEET_VER}/fleetctl_${FLEET_VER}_linux_amd64.tar.gz" \
    | tar -xz -C "$WORK/fleetctl-$FLEET_VER" --strip-components=1 || die "no fleetctl release fleet-$FLEET_VER"
fi
"$FC" --version | head -1

# ---- the block added to the stock post-install script (POSIX sh; runs before the stock restart) ----
cat > "$WORK/postinst-block.sh" <<'BLOCK'
# >>> fleetd-proxy install-time configuration (added to the stock fleetd package) >>>
# Reads FLEET_URL, FLEET_SECRET, ORBIT_PROXY_URL, OSQUERY_PROXY, NO_PROXY from the installer's
# environment, then from KEY=value lines in /etc/default/fleetd-proxy. With none supplied
# (for example on upgrades), the existing settings are left as they are.
fp_conf=/etc/default/fleetd-proxy
fp_env=/etc/default/orbit-proxy
fp_dropin=/etc/systemd/system/orbit.service.d/10-fleetd-proxy.conf
fp_get() {
  fp_v=$(printenv "$1" 2>/dev/null || true)
  if [ -z "$fp_v" ] && [ -r "$fp_conf" ]; then
    fp_v=$(sed -n "s/^[[:space:]]*$1=//p" "$fp_conf" | tail -n 1 | sed -e 's/^["'\'']//' -e 's/["'\'']$//')
  fi
  printf '%s' "$fp_v"
}
fp_url=$(fp_get FLEET_URL); fp_secret=$(fp_get FLEET_SECRET)
fp_proxy=$(fp_get ORBIT_PROXY_URL); fp_osq=$(fp_get OSQUERY_PROXY)
if [ -n "$fp_url$fp_secret$fp_proxy$fp_osq" ]; then
  if [ -n "$fp_proxy" ]; then
    case "$fp_proxy" in *://*) ;; *) fp_proxy="http://$fp_proxy" ;; esac
    fp_noproxy=$(fp_get NO_PROXY)
    if [ -z "$fp_osq" ]; then
      fp_osq=${fp_proxy#*://}; fp_osq=${fp_osq%%/*}
      case "$fp_osq" in
        *:*) ;;
        *) case "$fp_proxy" in https://*) fp_osq="$fp_osq:443" ;; *) fp_osq="$fp_osq:80" ;; esac ;;
      esac
    fi
  fi
  fp_osq=${fp_osq#*://}; fp_osq=${fp_osq%%/*}
  # Anything not supplied this time keeps its previous value.
  if [ -r "$fp_env" ]; then
    if [ -z "$fp_url" ]; then fp_url=$(sed -n 's/^ORBIT_FLEET_URL=//p' "$fp_env" | tail -n 1); fi
    if [ -z "$fp_secret" ]; then fp_secret=$(sed -n 's/^ORBIT_ENROLL_SECRET=//p' "$fp_env" | tail -n 1); fi
    if [ -z "$fp_proxy" ]; then
      fp_proxy=$(sed -n 's/^HTTPS_PROXY=//p' "$fp_env" | tail -n 1)
      fp_noproxy=$(sed -n 's/^NO_PROXY=//p' "$fp_env" | tail -n 1)
    fi
  fi
  if [ -z "$fp_osq" ] && [ -r "$fp_dropin" ]; then
    fp_osq=$(sed -n 's/.*--proxy_hostname=\([^ ]*\).*/\1/p' "$fp_dropin" | tail -n 1)
  fi
  fp_exec=$(sed -n 's/^ExecStart=//p' /usr/lib/systemd/system/orbit.service 2>/dev/null | tail -n 1)
  if [ -z "$fp_exec" ]; then fp_exec=/opt/orbit/bin/orbit/orbit; fi
  fp_old_umask=$(umask); umask 077
  {
    echo "# Written by the fleetd-proxy package install; overrides /etc/default/orbit."
    if [ -n "$fp_url" ]; then echo "ORBIT_FLEET_URL=$fp_url"; fi
    if [ -n "$fp_secret" ]; then echo "ORBIT_ENROLL_SECRET=$fp_secret"; fi
    if [ -n "$fp_proxy" ]; then
      echo "HTTP_PROXY=$fp_proxy"
      echo "HTTPS_PROXY=$fp_proxy"
      echo "NO_PROXY=${fp_noproxy:-localhost,127.0.0.1}"
    fi
  } > "$fp_env.tmp" && mv -f "$fp_env.tmp" "$fp_env"
  umask "$fp_old_umask"
  mkdir -p "${fp_dropin%/*}"
  {
    echo "# Written by the fleetd-proxy package install."
    echo "[Service]"
    echo "EnvironmentFile=$fp_env"
    if [ -n "$fp_osq" ]; then
      echo "ExecStart="
      echo "ExecStart=$fp_exec -- --proxy_hostname=$fp_osq"
    fi
  } > "$fp_dropin"
  fp_secret_state=unchanged; if [ -n "$(printenv FLEET_SECRET 2>/dev/null)$(sed -n 's/^[[:space:]]*FLEET_SECRET=//p' "$fp_conf" 2>/dev/null)" ]; then fp_secret_state=set; fi
  echo "fleetd-proxy: Fleet URL ${fp_url:-(package default)}, enroll secret ${fp_secret_state}, proxy ${fp_proxy:-none}, osquery proxy ${fp_osq:-none}"
fi
# <<< fleetd-proxy install-time configuration <<<
BLOCK
cat > "$WORK/postrm-block.sh" <<'BLOCK'
# >>> fleetd-proxy install-time configuration: cleanup on uninstall >>>
if [ "${1:-}" = 0 ] || [ "${1:-}" = remove ] || [ "${1:-}" = purge ]; then
  rm -f /etc/default/orbit-proxy /etc/systemd/system/orbit.service.d/10-fleetd-proxy.conf
  rmdir /etc/systemd/system/orbit.service.d 2>/dev/null || true
  if command -v systemctl >/dev/null 2>&1; then systemctl daemon-reload >/dev/null 2>&1 || true; fi
fi
# <<< fleetd-proxy cleanup <<<
BLOCK
sh -n "$WORK/postinst-block.sh" && sh -n "$WORK/postrm-block.sh" || die "script block syntax"

stock_pkg() {  # $1 = deb|rpm -> prints path of the stock package
  local d="$WORK/stock-$1-$FLEET_VER-$ARCH-orbit_$ORBIT_CHANNEL-osq_$OSQUERYD_CHANNEL" f
  f="$(find "$d" -maxdepth 1 -name "*.$1" 2>/dev/null | head -1)"
  if [ -z "$f" ]; then
    mkdir -p "$d"
    ( cd "$d" && "$FC" package --type "$1" --arch "$ARCH" \
        --fleet-url "${FLEET_URL_DEFAULT:-https://placeholder.fleet.example.com}" \
        --enroll-secret "PLACEHOLDERENROLLSECRET0123456789" \
        --orbit-channel "$ORBIT_CHANNEL" --osqueryd-channel "$OSQUERYD_CHANNEL" >&2 ) || die "fleetctl package --type $1 failed"
    f="$(find "$d" -maxdepth 1 -name "*.$1" | head -1)"
  fi
  [ -f "$f" ] || die "stock .$1 not found"
  echo "$f"
}

ok=1
for type in $TYPES; do
  echo "== $type =="
  stock="$(stock_pkg "$type")"; echo "stock: $stock"
  case "$type" in
  deb)
    rm -rf "$WORK/deb"; dpkg-deb -R "$stock" "$WORK/deb" || die "dpkg-deb -R"
    for s in postinst postrm; do
      f="$WORK/deb/DEBIAN/$s"; [ -f "$f" ] || die "stock deb has no $s"
      if [ "$s" = postinst ]; then
        { head -n 1 "$f"; cat "$WORK/postinst-block.sh"; tail -n +2 "$f"; } > "$f.new"   # after the shebang
      else
        { cat "$f"; echo; cat "$WORK/postrm-block.sh"; } > "$f.new"
      fi
      mv "$f.new" "$f"; chmod 0755 "$f"
    done
    out="$OUT_DIR/$(basename "$stock" .deb | sed 's/^fleet-osquery/fleetd-proxy/').deb"
    dpkg-deb --root-owner-group -b "$WORK/deb" "$out" >/dev/null || die "dpkg-deb -b"
    echo "-- verify"
    debls() { dpkg-deb -c "$1" | awk '$6 != "./" { m = $1; if (m ~ /^l/) m = "l"; print m, $2, $3, $6, $7, $8 }' | sort; }  # symlink modes are meaningless
    if diff <(debls "$stock") <(debls "$out") >/dev/null; then
      echo "payload: identical to stock"; else echo "payload: DIFFERS"; ok=0; fi
    dpkg-deb -I "$out" postinst | grep -q "$MARK" && echo "postinst: block present" || { echo "postinst: block MISSING"; ok=0; }
    dpkg-deb -I "$out" postrm | grep -q "$MARK" && echo "postrm: cleanup present" || { echo "postrm: cleanup MISSING"; ok=0; }
    ;;
  rpm)
    # Insert the block right after the %post header, append cleanup to %postun.
    # (rpmrebuild escapes % in the filtered sections itself; don't pre-escape.)
    cp "$WORK/postinst-block.sh" "$WORK/postinst-block.rpm"; cp "$WORK/postrm-block.sh" "$WORK/postrm-block.rpm"
    cat > "$WORK/post-filter.sh" <<EOF
#!/bin/sh
awk '{ print } /^%post/ && !done { while ((getline l < "$WORK/postinst-block.rpm") > 0) print l; done = 1 }'
EOF
    cat > "$WORK/postun-filter.sh" <<EOF
#!/bin/sh
cat; echo; cat "$WORK/postrm-block.rpm"
EOF
    chmod +x "$WORK/post-filter.sh" "$WORK/postun-filter.sh"
    rm -rf "$WORK/rpmout"; mkdir -p "$WORK/rpmout"
    rpmrebuild --package --notest-install --directory="$WORK/rpmout" \
      --change-spec-post="$WORK/post-filter.sh" --change-spec-postun="$WORK/postun-filter.sh" \
      "$stock" >/dev/null || die "rpmrebuild failed"
    built="$(find "$WORK/rpmout" -name '*.rpm' | head -1)"; [ -f "$built" ] || die "rebuilt rpm not found"
    out="$OUT_DIR/$(basename "$stock" | sed 's/^fleet-osquery/fleetd-proxy/')"
    cp -f "$built" "$out"
    echo "-- verify"
    rpmls() { rpm -qp --qf '[%{FILEMODES:perms} %{FILEUSERNAME} %{FILEGROUPNAME} %{FILEDIGESTS} %{FILENAMES} %{FILELINKTOS}\n]' "$1" 2>/dev/null \
                | sed 's/^l[-rwx]\{9\} /l /' | sort; }  # symlink modes are meaningless
    if diff <(rpmls "$stock") <(rpmls "$out") >/dev/null; then
      echo "payload: identical to stock"; else echo "payload: DIFFERS"; ok=0; fi
    rpm -qp --scripts "$out" > "$WORK/scripts.txt" 2>/dev/null
    grep -q ">>> $MARK (added" "$WORK/scripts.txt" && grep -q "cleanup on uninstall" "$WORK/scripts.txt" \
      && echo "scripts: block + cleanup present" || { echo "scripts: block or cleanup MISSING"; ok=0; }
    grep -q 'posttrans' "$WORK/scripts.txt" && echo "posttrans: kept" || { echo "posttrans: MISSING"; ok=0; }
    grep -qF 'fp_osq=${fp_osq%%/*}' "$WORK/scripts.txt" && grep -qF 'mkdir -p "${fp_dropin%/*}"' "$WORK/scripts.txt" \
      && echo "scripts: % survived rpmbuild intact" || { echo "scripts: % MANGLED by rpmbuild"; ok=0; }
    ;;
  *) die "unknown type $type" ;;
  esac
  [ -z "${HOST_UID:-}" ] || chown "$HOST_UID:${HOST_GID:-$HOST_UID}" "$out"
  ls -l "$out"; sha256sum "$out"
done
[ "$ok" = 1 ] && echo "VERIFY PASS" || { echo "VERIFY FAIL"; exit 1; }
