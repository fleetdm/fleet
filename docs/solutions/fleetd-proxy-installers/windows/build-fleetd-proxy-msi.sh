#!/bin/bash
# ============================================================================
# build-fleetd-proxy-msi.sh: build a fleetd MSI whose proxy settings are passed as
# INSTALL-TIME properties, so one MSI can be deployed behind a forward proxy (Squid etc.)
# by any tool (Intune / SCCM / PDQ / GPO / msiexec).
#
# Two ways to run it (both need amd64; on an Apple-silicon Mac, Wine segfaults under emulation):
#   - Sandboxed (recommended): ./build-in-container.sh runs this script inside Fleet's own
#     fleetdm/wix image with `fleetctl package --native-tooling`. Nothing is installed on the
#     host, and no Docker socket or --privileged is needed. Works with Docker or Podman.
#   - Directly on amd64 Linux with Docker, as root or with sudo (it apt-installs what it needs).
#
# It builds a stock `fleetctl package --type msi`, then edits it with msitools SQL:
#   - adds public properties ORBIT_PROXY_URL / OSQUERY_PROXY / NO_PROXY (+ SecureCustomProperties)
#   - appends `-- --proxy_hostname="[OSQUERY_PROXY]"` to the Orbit service arguments;
#     osquery ignores HTTP(S)_PROXY and only honours its own --proxy_hostname
#   - writes HTTP_PROXY / HTTPS_PROXY / NO_PROXY into the service-specific environment
#     HKLM\SYSTEM\CurrentControlSet\Services\Fleet osquery\Environment (REG_MULTI_SZ), which
#     Orbit's Go HTTP clients read from the first service start. If the stock MSI already
#     writes that value (fleetctl after v4.92.0, fleetdm/fleet#52475), the proxy entries
#     are merged into it instead of racing it.
#   - optionally Authenticode-signs the result (osslsigncode) when SIGN_PFX is set
# FLEET_URL / FLEET_SECRET stay overridable at install time (stock behaviour).
#
# Knobs (env vars):
#   FLEET_VER         fleetctl release that builds the stock MSI, e.g. v4.92.0  (default v4.91.1)
#   ORBIT_CHANNEL     Orbit version baked in and followed for auto-updates: stable | edge | 1.60.0
#   OSQUERYD_CHANNEL  osqueryd version, same idea: stable | 5.23.1                (default stable)
#   ARCH              amd64 | arm64 (Windows on ARM)                               (default amd64)
#   EXTRA_PACKAGE_ARGS  anything else for `fleetctl package`, e.g. "--fleet-desktop"
#   OUT_NAME          output file name                              (default fleetd-proxy.msi)
#   WORK              build directory                               (default ~/fleetd-proxy-build)
#   SIGN_PFX          path to a .pfx/.p12 code-signing cert (optional; unsigned if unset)
#   SIGN_PFX_PASS_FILE  file holding the PFX password (kept off the command line)
#   TIMESTAMP_URL     RFC 3161 timestamp server                  (default http://timestamp.digicert.com)
#   OUT_DIR           where the final MSI is copied                  (default $WORK)
#   NATIVE_TOOLING=1  run WiX from PATH instead of via Docker (set by build-in-container.sh)
#
# Deploy:
#   msiexec /i fleetd-proxy.msi /qn FLEET_URL=https://fleet.example.com FLEET_SECRET=<enroll-secret> ^
#     ORBIT_PROXY_URL=http://proxy.example.internal:3128 OSQUERY_PROXY=proxy.example.internal:3128 ^
#     NO_PROXY=localhost,127.0.0.1
# ============================================================================
set -uo pipefail
export DEBIAN_FRONTEND=noninteractive
FLEET_VER="${FLEET_VER:-v4.91.1}"
ORBIT_CHANNEL="${ORBIT_CHANNEL:-stable}"
OSQUERYD_CHANNEL="${OSQUERYD_CHANNEL:-stable}"
ARCH="${ARCH:-amd64}"
EXTRA_PACKAGE_ARGS="${EXTRA_PACKAGE_ARGS:-}"
OUT_NAME="${OUT_NAME:-fleetd-proxy.msi}"
WORK="${WORK:-$HOME/fleetd-proxy-build}"
SIGN_PFX="${SIGN_PFX:-}"
SIGN_PFX_PASS_FILE="${SIGN_PFX_PASS_FILE:-}"
TIMESTAMP_URL="${TIMESTAMP_URL:-http://timestamp.digicert.com}"
OUT_DIR="${OUT_DIR:-$WORK}"
NATIVE_TOOLING="${NATIVE_TOOLING:-}"
die() { echo "ERROR: $*" >&2; exit 1; }
SUDO=""; [ "$(id -u)" -eq 0 ] || SUDO="sudo"
mkdir -p "$WORK"; cd "$WORK" || die "cannot cd to $WORK"

echo "== deps =="
need=""
command -v msibuild >/dev/null || need="$need msitools"
command -v python3  >/dev/null || need="$need python3"
command -v curl     >/dev/null || need="$need curl ca-certificates"
[ -z "$SIGN_PFX" ] || command -v osslsigncode >/dev/null || need="$need osslsigncode"
if [ -n "$NATIVE_TOOLING" ]; then
  for t in heat candle light; do command -v $t >/dev/null || die "NATIVE_TOOLING set but WiX '$t' not on PATH (use build-in-container.sh)"; done
  PKG_MODE=(--native-tooling)
else
  command -v docker >/dev/null || need="$need docker.io"
  PKG_MODE=()
fi
[ -z "$need" ] || { $SUDO apt-get update -qq && $SUDO apt-get install -y -qq --no-install-recommends $need; } || die "apt install failed:$need"
[ -n "$NATIVE_TOOLING" ] || docker info >/dev/null 2>&1 || die "docker daemon not reachable (running? in the docker group?)"

echo "== fleetctl $FLEET_VER =="
# Always use a per-version fleetctl, never whatever is on PATH: the MSI template comes from fleetctl.
FC="$WORK/fleetctl-$FLEET_VER/fleetctl"
if [ ! -x "$FC" ]; then
  mkdir -p "$WORK/fleetctl-$FLEET_VER"
  curl -fsSL -o "$WORK/fleetctl-$FLEET_VER.tgz" \
    "https://github.com/fleetdm/fleet/releases/download/fleet-${FLEET_VER}/fleetctl_${FLEET_VER}_linux_amd64.tar.gz" \
    || die "no fleetctl release fleet-$FLEET_VER"
  tar -xzf "$WORK/fleetctl-$FLEET_VER.tgz" -C "$WORK/fleetctl-$FLEET_VER" --strip-components=1
fi
"$FC" --version | head -1

echo "== stock MSI (placeholder URL/secret, overridden at install) =="
STOCKDIR="$WORK/stock-$FLEET_VER-$ARCH-orbit_$ORBIT_CHANNEL-osq_$OSQUERYD_CHANNEL"
STOCK="$(find "$STOCKDIR" -maxdepth 2 -name '*.msi' 2>/dev/null | head -1)"   # native tooling writes build/*.msi
if [ -z "$STOCK" ]; then
  mkdir -p "$STOCKDIR"
  # shellcheck disable=SC2086
  ( cd "$STOCKDIR" && "$FC" package --type msi "${PKG_MODE[@]}" --arch "$ARCH" \
      --fleet-url "https://placeholder.fleet.example.com" \
      --enroll-secret "PLACEHOLDERENROLLSECRET0123456789" \
      --orbit-channel "$ORBIT_CHANNEL" --osqueryd-channel "$OSQUERYD_CHANNEL" $EXTRA_PACKAGE_ARGS ) \
    || die "fleetctl package failed"
  STOCK="$(find "$STOCKDIR" -maxdepth 2 -name '*.msi' | head -1)"
fi
[ -f "$STOCK" ] || die "stock MSI not found in $STOCKDIR"
echo "stock: $STOCK"

echo "== inject proxy support =="
UNSIGNED="$WORK/${OUT_NAME%.msi}.unsigned.msi"; cp -f "$STOCK" "$UNSIGNED"
OUT="$UNSIGNED" python3 - <<'PY' || die "proxy injection failed verification"
import subprocess, os, sys
OUT=os.environ['OUT']
def q(query):
    r=subprocess.run(['msibuild',OUT,'-q',query],capture_output=True,text=True)
    print(('OK   ' if r.returncode==0 else 'note ')+query[:80])
def rows(t):
    lines=subprocess.run(['msiinfo','export',OUT,t],capture_output=True,text=True).stdout.split('\n')
    return [l.split('\t') for l in lines[3:] if l.strip()]
svc=[r for r in rows('ServiceInstall') if len(r)>1 and r[1]=='Fleet osquery']
if not svc: sys.exit('no ServiceInstall row named "Fleet osquery"')
key=svc[0][0]; args=svc[0][10] if len(svc[0])>10 else ''
if '--proxy_hostname' not in args:
    args=(args.rstrip()+' -- --proxy_hostname="[OSQUERY_PROXY]"').lstrip()
    q("UPDATE `ServiceInstall` SET `Arguments`='%s' WHERE `ServiceInstall`='%s'"%(args.replace("'","''"),key))
q("INSERT INTO `Property` (`Property`,`Value`) VALUES ('NO_PROXY','localhost,127.0.0.1')")  # empty-valued props are omitted; passed at install
PROXY_ENV='HTTP_PROXY=[ORBIT_PROXY_URL][~]HTTPS_PROXY=[ORBIT_PROXY_URL][~]NO_PROXY=[NO_PROXY]'
reg=rows('Registry')
stock=[r for r in reg if len(r)>=6 and r[2].lower()=='system\\currentcontrolset\\services\\fleet osquery' and r[3].lower()=='environment']
if stock:   # fleetctl > v4.92.0 writes its own Environment value: merge, don't race it
    core=stock[0][4]
    core=core[3:] if core.startswith('[~]') else core
    core=core[:-3] if core.endswith('[~]') else core
    merged='[~]%s[~]%s[~]'%(core,PROXY_ENV)  # [~] at both ends = replace with the full merged list
    q("UPDATE `Registry` SET `Value`='%s' WHERE `Registry`='%s'"%(merged.replace("'","''"),stock[0][0]))
else:
    comp=[r[0] for r in rows('Component') if r and r[0]=='C_ORBITBIN'] or [svc[0][11]]  # else the service's component
    q("CREATE TABLE `Registry` (`Registry` CHAR(72) NOT NULL, `Root` SHORT NOT NULL, `Key` CHAR(255) NOT NULL LOCALIZABLE, `Name` CHAR(255) LOCALIZABLE, `Value` CHAR(0) LOCALIZABLE, `Component_` CHAR(72) NOT NULL PRIMARY KEY `Registry`)")
    q("INSERT INTO `Registry` (`Registry`,`Root`,`Key`,`Name`,`Value`,`Component_`) VALUES ('RegOrbitProxyEnv',2,'SYSTEM\\CurrentControlSet\\Services\\Fleet osquery','Environment','%s','%s')"%(PROXY_ENV,comp[0]))
q("INSERT INTO `InstallExecuteSequence` (`Action`,`Sequence`) VALUES ('WriteRegistryValues',5000)")
scp=[r for r in rows('Property') if r[0]=='SecureCustomProperties']
want='ORBIT_PROXY_URL;OSQUERY_PROXY;NO_PROXY'
if scp:
    merged=';'.join(dict.fromkeys(scp[0][1].split(';')+want.split(';')))
    q("UPDATE `Property` SET `Value`='%s' WHERE `Property`='SecureCustomProperties'"%merged)
else:
    q("INSERT INTO `Property` (`Property`,`Value`) VALUES ('SecureCustomProperties','%s')"%want)

print('\n== verify ==')
ok=True
a=[r for r in rows('ServiceInstall') if r[1]=='Fleet osquery'][0]
a=a[10] if len(a)>10 else ''
print('service args end:', a[-60:]); ok&='--proxy_hostname="[OSQUERY_PROXY]"' in a
env=[r[4] for r in rows('Registry') if r[3].lower()=='environment' and 'fleet osquery' in r[2].lower()]
print('Environment rows:', len(env)); ok&=len(env)==1 and 'HTTP_PROXY=[ORBIT_PROXY_URL]' in env[0]
if env: print('Environment value:', env[0][:200])
seq=[r[0] for r in rows('InstallExecuteSequence')]; ok&='WriteRegistryValues' in seq
props={r[0]:(r[1] if len(r)>1 else '') for r in rows('Property')}
for p in ('FLEET_URL','FLEET_SECRET','NO_PROXY','SecureCustomProperties'): print('%-22s %s'%(p, props.get(p,'<missing>')))
ok&=all(p in props for p in ('FLEET_URL','FLEET_SECRET','NO_PROXY'))
print('VERIFY', 'PASS' if ok else 'FAIL'); sys.exit(0 if ok else 1)
PY

mkdir -p "$OUT_DIR"; OUT="$OUT_DIR/$OUT_NAME"
if [ -n "$SIGN_PFX" ]; then
  echo "== sign (osslsigncode, sha256, timestamp $TIMESTAMP_URL) =="
  [ -f "$SIGN_PFX" ] || die "SIGN_PFX not found: $SIGN_PFX"
  passarg=(); [ -n "$SIGN_PFX_PASS_FILE" ] && passarg=(-readpass "$SIGN_PFX_PASS_FILE")
  rm -f "$OUT"
  osslsigncode sign -pkcs12 "$SIGN_PFX" "${passarg[@]}" -h sha256 \
    -n "Fleet osquery" -i "https://fleetdm.com" -ts "$TIMESTAMP_URL" \
    -in "$UNSIGNED" -out "$OUT" || die "signing failed"
  osslsigncode verify -in "$OUT" ${VERIFY_CAFILE:+-CAfile "$VERIFY_CAFILE"} 2>&1 | grep -E "Signer|Subject:|Issuer:|Message digest|Signature verification|Timestamp" | head -12
else
  cp -f "$UNSIGNED" "$OUT"; echo "== unsigned (set SIGN_PFX to sign) =="
fi

[ -z "${HOST_UID:-}" ] || chown "$HOST_UID:${HOST_GID:-$HOST_UID}" "$OUT"   # container runs: hand the file back to the caller
echo "== result =="
ls -lh "$OUT"; sha256sum "$OUT"
