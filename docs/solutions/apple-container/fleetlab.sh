#!/usr/bin/env bash
# fleetlab.sh - run a Fleet lab on a Mac with Apple container (no Docker).
#
#   ./fleetlab.sh setup    One-time: checks, Rosetta, Apple container, DNS, .env, certs
#   ./fleetlab.sh up       Deploy Fleet (first time) and wait until it's ready
#   ./fleetlab.sh start    Start an existing deployment
#   ./fleetlab.sh stop     Stop everything (data kept)
#   ./fleetlab.sh status   Show containers and whether Fleet is answering
#   ./fleetlab.sh logs     Follow Fleet's logs (logs mysql / logs redis also work)
#   ./fleetlab.sh update   Pull the newest Fleet image and recreate the Fleet container
#   ./fleetlab.sh certs    Regenerate the TLS certificate (new IP address, or renewal)
#   ./fleetlab.sh down     Remove containers (data kept)
#   ./fleetlab.sh reset    Remove containers AND all data (asks first)

set -euo pipefail
cd "$(dirname "$0")"

DNS_DOMAIN="${DNS_DOMAIN:-test}"
NET="fleet-net"
VOLUMES=(fleet-mysql fleet-redis fleet-data fleet-logs fleet-vulndb)
CONTAINERS=(fleet fleet-redis fleet-mysql)
ENV_URL="https://raw.githubusercontent.com/fleetdm/fleet/refs/heads/main/docs/solutions/docker-compose/env.example"
CONFIG_TOML="$HOME/.config/container/config.toml"

say()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m ok\033[0m %s\n' "$*"; }
die()  { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

exists()   { container ls -a 2>/dev/null | awk 'NR>1 {print $1}' | grep -qx "$1"; }
running()  { container ls    2>/dev/null | awk 'NR>1 {print $1}' | grep -qx "$1"; }
volume_exists() { container volume ls 2>/dev/null | awk 'NR>1 {print $1}' | grep -qx "$1"; }
lan_ip()   { ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null || echo "<your-mac-ip>"; }
# `container ls` only succeeds when Apple container's services are running.
ensure_system() { container ls >/dev/null 2>&1 || container system start; }

load_env() {
  [[ -f .env ]] || die ".env not found. Run: ./fleetlab.sh setup"
  set -a; source ./.env; set +a
  FLEET_PORT="${FLEET_SERVER_PORT:-1337}"
  VULNDB_PATH="${FLEET_VULNERABILITIES_DATABASES_PATH:-/vulndb}"
  FLEET_IMAGE="${FLEET_IMAGE:-fleetdm/fleet:latest}"
  if [[ "${FLEET_SERVER_TLS:-true}" == "true" ]]; then SCHEME="https"; else SCHEME="http"; fi
}

warn_no_tls() {
  if [[ "$SCHEME" == "http" ]]; then
    say "Note: FLEET_SERVER_TLS is off, so Fleet serves plain HTTP. fleetctl and host enrollment need HTTPS."
  fi
}

# Set KEY=VALUE in .env, replacing an existing line or appending a new one.
set_env() {
  if grep -q "^$1=" .env; then sed -i '' "s|^$1=.*|$1=$2|" .env; else echo "$1=$2" >> .env; fi
}

# Create a self-signed certificate whose Subject Alternative Names (SANs) cover
# every address hosts might use to reach Fleet. TLS clients ignore the CN.
make_cert() {
  local host ip san cnf
  host="$(hostname)"
  ip="$(ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null || true)"
  san="DNS:${host},DNS:localhost,DNS:fleet.${DNS_DOMAIN},IP:127.0.0.1"
  if [[ -n "$ip" ]]; then san="${san},IP:${ip}"; fi
  mkdir -p certs
  cnf="$(mktemp)"
  cat > "$cnf" << EOF
[req]
distinguished_name = dn
x509_extensions    = v3
prompt             = no
[dn]
CN = ${host}
[v3]
subjectAltName   = ${san}
basicConstraints = critical, CA:TRUE
keyUsage         = critical, digitalSignature, keyEncipherment, keyCertSign
extendedKeyUsage = serverAuth
EOF
  openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout certs/fleet.key -out certs/fleet.crt -config "$cnf" 2>/dev/null
  rm -f "$cnf"
}

show_cert() {
  openssl x509 -in certs/fleet.crt -noout -text | grep -A1 "Subject Alternative Name" | tail -1 | sed 's/^ */    Valid for: /'
  openssl x509 -in certs/fleet.crt -noout -enddate | sed 's/^notAfter=/    Expires:   /'
}

# ---------------------------------------------------------------- setup ----
cmd_setup() {
  say "Checking this Mac"
  [[ "$(uname -m)" == "arm64" ]] || die "Apple silicon is required."
  local major; major="$(sw_vers -productVersion | cut -d. -f1)"
  (( major >= 26 )) || die "macOS 26 or later is required (found $(sw_vers -productVersion))."
  ok "Apple silicon, macOS $(sw_vers -productVersion)"

  if arch -x86_64 /usr/bin/true 2>/dev/null; then ok "Rosetta installed"
  else say "Installing Rosetta"; softwareupdate --install-rosetta --agree-to-license; fi

  if command -v container >/dev/null; then ok "Apple container installed"
  else
    say "Apple container is not installed"
    local url; url="$(curl -fsSL https://api.github.com/repos/apple/container/releases/latest \
      | grep -o '"browser_download_url": *"[^"]*signed\.pkg"' | head -1 | cut -d'"' -f4)"
    [[ -n "$url" ]] || die "Couldn't find the installer. Download it from https://github.com/apple/container/releases"
    read -r -p "Download and install $(basename "$url")? [y/N] " a
    [[ "$a" =~ ^[Yy]$ ]] || die "Install Apple container, then re-run setup."
    local tmp; tmp="$(mktemp -d)"
    trap 'rm -rf "$tmp"' EXIT
    curl -fL "$url" -o "$tmp/container-installer.pkg"
    sudo installer -pkg "$tmp/container-installer.pkg" -target /
    rm -rf "$tmp"; trap - EXIT
  fi
  ensure_system

  say "Configuring local DNS (.$DNS_DOMAIN)"
  if container system dns list 2>/dev/null | grep -qw "$DNS_DOMAIN"; then ok "DNS domain exists"
  else sudo container system dns create "$DNS_DOMAIN"; fi

  mkdir -p "$(dirname "$CONFIG_TOML")"; touch "$CONFIG_TOML"
  if grep -q '^\[dns\]' "$CONFIG_TOML"; then
    grep -q "domain *= *\"$DNS_DOMAIN\"" "$CONFIG_TOML" && ok "config.toml already sets domain" \
      || die "$CONFIG_TOML has a [dns] section without domain = \"$DNS_DOMAIN\". Edit it, then re-run."
  else
    printf '\n[dns]\ndomain = "%s"\n' "$DNS_DOMAIN" >> "$CONFIG_TOML"
    say "Restarting Apple container to apply DNS settings"
    container system stop; container system start
  fi

  say "Preparing .env"
  if [[ -f .env ]]; then ok ".env already exists (left unchanged)"
  else
    curl -fsSL "$ENV_URL" -o .env
    set_env MYSQL_ROOT_PASSWORD "$(openssl rand -hex 16)"
    set_env MYSQL_PASSWORD      "$(openssl rand -hex 16)"
    set_env FLEET_SERVER_PRIVATE_KEY "$(openssl rand -base64 32)"
    set_env FLEET_SERVER_TLS true
    grep -q '^MYSQL_USER=.'        .env || set_env MYSQL_USER fleet
    grep -q '^MYSQL_DATABASE=.'    .env || set_env MYSQL_DATABASE fleet
    grep -q '^FLEET_SERVER_PORT=.' .env || set_env FLEET_SERVER_PORT 1337
    ok "Generated passwords and server key"
  fi

  if [[ -f certs/fleet.crt && -f certs/fleet.key ]]; then
    ok "Certificates exist"
    openssl x509 -in certs/fleet.crt -noout -text | grep -q "Subject Alternative Name" \
      || say "Note: this certificate has no SANs, so hosts can't verify it. Run: ./fleetlab.sh certs"
  else
    say "Creating self-signed certificate"
    make_cert
    show_cert
  fi

  say "Setup complete. Next: ./fleetlab.sh up"
}

# ---------------------------------------------------------------- deploy ---
run_fleet() {
  local fleet_address
  if [[ -n "${FLEET_SERVER_ADDRESS:-}" && "${FLEET_SERVER_ADDRESS}" =~ :[0-9]+$ ]]; then
    fleet_address="${FLEET_SERVER_ADDRESS}"
  else
    fleet_address="${FLEET_SERVER_ADDRESS:-0.0.0.0}:${FLEET_PORT}"
  fi

  container run -d --name fleet --network "$NET" \
    --platform linux/amd64 --rosetta -m 4G -c 2 \
    -p "${FLEET_PORT}:${FLEET_PORT}" \
    -v fleet-data:/fleet -v fleet-logs:/logs -v "fleet-vulndb:${VULNDB_PATH}" \
    -v "$PWD/certs:/certs:ro" \
    -e FLEET_REDIS_ADDRESS="fleet-redis.${DNS_DOMAIN}:6379" \
    -e FLEET_MYSQL_ADDRESS="fleet-mysql.${DNS_DOMAIN}:3306" \
    -e FLEET_MYSQL_DATABASE="$MYSQL_DATABASE" \
    -e FLEET_MYSQL_USERNAME="$MYSQL_USER" \
    -e FLEET_MYSQL_PASSWORD="$MYSQL_PASSWORD" \
    -e FLEET_SERVER_ADDRESS="$fleet_address" \
    -e FLEET_SERVER_TLS="${FLEET_SERVER_TLS:-true}" \
    -e FLEET_SERVER_CERT=/certs/fleet.crt -e FLEET_SERVER_KEY=/certs/fleet.key \
    -e FLEET_SERVER_PRIVATE_KEY="$FLEET_SERVER_PRIVATE_KEY" \
    -e FLEET_LICENSE_KEY="${FLEET_LICENSE_KEY:-}" \
    -e FLEET_LOGGING_JSON="${FLEET_LOGGING_JSON:-true}" \
    -e FLEET_OSQUERY_STATUS_LOG_PLUGIN="${FLEET_OSQUERY_STATUS_LOG_PLUGIN:-filesystem}" \
    -e FLEET_FILESYSTEM_STATUS_LOG_FILE="${FLEET_FILESYSTEM_STATUS_LOG_FILE:-/logs/osqueryd.status.log}" \
    -e FLEET_FILESYSTEM_RESULT_LOG_FILE="${FLEET_FILESYSTEM_RESULT_LOG_FILE:-/logs/osqueryd.results.log}" \
    -e FLEET_VULNERABILITIES_DATABASES_PATH="$VULNDB_PATH" \
    -e FLEET_VULNERABILITIES_CURRENT_INSTANCE_CHECKS="${FLEET_VULNERABILITIES_CURRENT_INSTANCE_CHECKS:-yes}" \
    "$FLEET_IMAGE" sh -c "/usr/bin/fleet prepare db --no-prompt && /usr/bin/fleet serve" >/dev/null
}

wait_for_dbs() {
  local i out
  say "Waiting for MySQL"
  for i in $(seq 1 120); do
    if out="$(container exec fleet-mysql mysql -h 127.0.0.1 -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" \
                -e 'SELECT 1' "$MYSQL_DATABASE" 2>&1)"; then break; fi
    if grep -q "Access denied" <<< "$out"; then
      die "MySQL rejected the password in .env. The fleet-mysql volume was probably created by an earlier deployment with different passwords. Use that deployment's .env, or run ./fleetlab.sh reset."
    fi
    (( i < 120 )) || die "MySQL didn't become ready. Check: ./fleetlab.sh logs mysql"
    sleep 3
  done

  say "Waiting for Redis"
  for i in $(seq 1 60); do
    container exec fleet-redis redis-cli ping 2>/dev/null | grep -q PONG && break
    (( i < 60 )) || die "Redis didn't become ready. Check: ./fleetlab.sh logs redis"
    sleep 2
  done
}

# Fleet finds MySQL and Redis by DNS name. Make sure container DNS answers before
# starting Fleet, because Fleet exits if its first lookup times out.
wait_for_dns() {
  local i
  running fleet-redis || die "MySQL and Redis aren't running. Run: ./fleetlab.sh start"
  say "Checking container DNS"
  for i in $(seq 1 30); do
    container exec fleet-redis getent hosts "fleet-mysql.${DNS_DOMAIN}" >/dev/null 2>&1 && return
    (( i < 30 )) || die "Containers can't resolve fleet-mysql.${DNS_DOMAIN}. Run: container system stop && container system start, then ./fleetlab.sh start"
    sleep 2
  done
}

fleet_healthy() {
  [[ "$(curl -sk --max-time 5 -o /dev/null -w '%{http_code}' "${SCHEME}://127.0.0.1:${FLEET_PORT}/healthz")" == "200" ]]
}

wait_for_fleet() {
  say "Waiting for Fleet (first start can take 5-10 minutes under Rosetta)"
  local i
  for i in $(seq 1 180); do
    if fleet_healthy; then
      ok "Fleet is ready"
      echo "    On this Mac:       ${SCHEME}://localhost:${FLEET_PORT}"
      echo "    Server URL to use: ${SCHEME}://$(lan_ip):${FLEET_PORT}"
      echo "    Fleet version:     $(container exec fleet /usr/bin/fleet version 2>/dev/null | head -1)"
      warn_no_tls
      return
    fi
    if ! running fleet; then
      echo "--- last 20 lines of Fleet's log ---" >&2
      container logs fleet 2>/dev/null | tail -20 >&2 || true
      die "Fleet stopped while starting. See the log above."
    fi
    sleep 5
  done
  die "Fleet didn't answer within 15 minutes. Check: ./fleetlab.sh logs"
}

cmd_up() {
  load_env; ensure_system
  exists fleet && die "Fleet is already deployed. Use ./fleetlab.sh start"
  if volume_exists fleet-mysql; then
    say "Reusing Fleet data from a previous deployment (its passwords must match .env)"
  fi

  say "Creating network and volumes"
  container network create "$NET" >/dev/null 2>&1 || true
  local v
  for v in "${VOLUMES[@]}"; do container volume create "$v" >/dev/null 2>&1 || true; done

  say "Starting MySQL and Redis"
  # An earlier `up` that failed before starting Fleet leaves these behind.
  local c
  for c in fleet-mysql fleet-redis; do
    container stop "$c" >/dev/null 2>&1 || true
    container rm "$c" >/dev/null 2>&1 || true
  done
  container run -d --name fleet-mysql --network "$NET" \
    -e MYSQL_ROOT_PASSWORD="$MYSQL_ROOT_PASSWORD" -e MYSQL_DATABASE="$MYSQL_DATABASE" \
    -e MYSQL_USER="$MYSQL_USER" -e MYSQL_PASSWORD="$MYSQL_PASSWORD" \
    -v fleet-mysql:/var/lib/mysql mysql:8 >/dev/null
  container run -d --name fleet-redis --network "$NET" \
    -v fleet-redis:/data redis:6 redis-server --appendonly yes >/dev/null

  say "Fixing volume permissions"
  container run --rm --network "$NET" \
    -v fleet-logs:/logs -v fleet-data:/data -v fleet-vulndb:/vulndb \
    alpine:latest sh -c "chown -R 100:101 /logs /data /vulndb"

  wait_for_dbs
  wait_for_dns
  say "Starting Fleet"
  run_fleet
  wait_for_fleet
}

# ------------------------------------------------------------ day-to-day ---
cmd_start() {
  load_env; ensure_system
  exists fleet || die "Fleet isn't deployed yet. Run: ./fleetlab.sh up"
  local c
  for c in fleet-mysql fleet-redis; do running "$c" || container start "$c" >/dev/null; done
  wait_for_dbs
  wait_for_dns
  running fleet || container start fleet >/dev/null
  wait_for_fleet
}

cmd_stop() {
  say "Stopping Fleet"
  local c
  for c in "${CONTAINERS[@]}"; do container stop "$c" >/dev/null 2>&1 || true; done
  ok "Stopped (data kept)"
}

cmd_status() {
  load_env
  container ls -a | awk 'NR==1 || /fleet/'
  if fleet_healthy; then ok "Fleet is answering at ${SCHEME}://localhost:${FLEET_PORT}"
  else echo "Fleet is not answering."; fi
}

cmd_logs() {
  case "${1:-fleet}" in
    mysql) container logs -f fleet-mysql ;;
    redis) container logs -f fleet-redis ;;
    *)     container logs -f fleet ;;
  esac
}

cmd_update() {
  load_env; ensure_system
  say "Pulling $FLEET_IMAGE"
  container image pull --platform linux/amd64 "$FLEET_IMAGE"
  wait_for_dns
  container stop fleet >/dev/null 2>&1 || true
  container rm fleet >/dev/null 2>&1 || true
  run_fleet
  wait_for_fleet
}

cmd_certs() {
  load_env
  if [[ -f certs/fleet.crt ]]; then
    mv -f certs/fleet.crt certs/fleet.crt.bak
    mv -f certs/fleet.key certs/fleet.key.bak 2>/dev/null || true
    say "Backed up the old certificate to certs/*.bak"
  fi
  say "Creating a new self-signed certificate"
  make_cert
  show_cert
  if exists fleet; then
    ensure_system
    wait_for_dns
    say "Recreating the Fleet container with the new certificate"
    container stop fleet >/dev/null 2>&1 || true
    container rm fleet >/dev/null 2>&1 || true
    run_fleet
    wait_for_fleet
  fi
  say "Rebuild and reinstall agent packages with --fleet-certificate=certs/fleet.crt"
}

cmd_down() {
  ensure_system
  cmd_stop
  local c
  for c in "${CONTAINERS[@]}"; do container rm "$c" >/dev/null 2>&1 || true; done
  ok "Containers removed (data kept). Redeploy with ./fleetlab.sh up"
}

cmd_reset() {
  read -r -p "Delete ALL Fleet data (database, logs, settings)? [y/N] " a
  [[ "$a" =~ ^[Yy]$ ]] || { echo "Cancelled."; exit 0; }
  cmd_down
  local v vols left=()
  for v in "${VOLUMES[@]}"; do container volume rm "$v" >/dev/null 2>&1 || true; done
  vols="$(container volume ls | awk 'NR>1 {print $1}')" || die "Couldn't list volumes to confirm the reset. Check: container volume ls"
  for v in "${VOLUMES[@]}"; do
    if grep -qx "$v" <<< "$vols"; then left+=("$v"); fi
  done
  (( ${#left[@]} == 0 )) || die "Couldn't delete ${left[*]}. A container may still be using it. Check: container ls -a"
  ok "All Fleet data deleted. Fresh deploy: ./fleetlab.sh up"
}

case "${1:-}" in
  setup)  cmd_setup ;;
  up)     cmd_up ;;
  start)  cmd_start ;;
  stop)   cmd_stop ;;
  status) cmd_status ;;
  logs)   shift; cmd_logs "${1:-fleet}" ;;
  update) cmd_update ;;
  certs)  cmd_certs ;;
  down)   cmd_down ;;
  reset)  cmd_reset ;;
  *)      sed -n '2,14p' "$0" | sed 's/^# \{0,1\}//'; exit 1 ;;
esac