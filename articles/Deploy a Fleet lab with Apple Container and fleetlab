# Deploy a Fleet lab with Apple Container and fleetlab


This guide walks you through deploying a Fleet lab on your Mac with `fleetlab`, a single script that automates the entire setup using Apple's open-source [container](https://github.com/apple/container) tool. You'll have Fleet running with MySQL and Redis in two commands and about 15 minutes, with no Docker required.

`fleetlab` is designed for local labs, testing, and learning Fleet. It deploys the same configuration as the official [Deploy Fleet with Docker Compose](https://fleetdm.com/guides/deploy-fleet-on-docker-compose) guide. If you'd rather run each step yourself, see the manual guide, Deploy Fleet with Apple container.

`fleetlab` isn't intended for production. For production deployments, see the [Docker Compose guide](https://fleetdm.com/guides/deploy-fleet-on-docker-compose) and Fleet's [Reference configuration strategies](https://fleetdm.com/docs/deploy/reference-architectures#reference-configuration-strategies).

## What you'll need

- A Mac with Apple silicon (M1 or later)
- macOS 26 or later
- Administrator access (you'll be asked for your password during setup)
- At least 8 GB of free memory and about 6 GB of free disk space
- Port 1337 available
- Basic command line familiarity

You don't need to install Rosetta or Apple container ahead of time. `fleetlab` checks for both and installs them if they're missing.

## Download the script

Create a new directory for your Fleet lab:

```
mkdir ~/fleet-lab
cd ~/fleet-lab
```

Save the script from The fleetlab script section at the end of this guide as `fleetlab.sh` in this directory, then make it executable:

```
chmod +x fleetlab.sh
```

Keep the script in this directory. It stores your configuration (`.env`) and certificates (`certs/`) alongside it.

## Set up your Mac

Run the one-time setup:

```
./fleetlab.sh setup
```

Setup checks your Mac, installs anything that's missing, and prepares your Fleet configuration. Enter your administrator password when prompted. If Apple container isn't installed, setup asks before downloading and installing it.

Setup is safe to run again. It skips anything that's already in place, and never overwrites an existing `.env` file.

## Optional: Add your license key

If you have a Fleet Premium license or a [30-day trial key](https://fleetdm.com/new-license), add it to the `.env` file setup created:

```
FLEET_LICENSE_KEY=your_license_key_here
```

Leave this blank to use Fleet's free tier.

## Start Fleet

Deploy Fleet:

```
./fleetlab.sh up
```

`fleetlab` downloads the images, starts MySQL and Redis, starts Fleet, and waits until Fleet is ready. The first run takes 5–10 minutes, because Fleet runs its database migrations under Rosetta.

When Fleet is ready, you'll see:

```
 ok Fleet is ready
    On this Mac:       https://localhost:1337
    Server URL to use: https://<Mac's LAN IP address>:1337
```

## Access Fleet

Open your browser and navigate to:

```
https://localhost:1337
```

Your browser will warn you about the self-signed certificate. This is expected - click "Advanced" and proceed. To remove the warning, see Manage the certificate.

You'll see the Fleet setup screen. Follow the prompts to:

1. Create your first admin account
2. Configure your organization name
3. Set the Fleet server URL

For the Fleet server URL, use the **Server URL to use** address `fleetlab` printed. This is your Mac's LAN IP address, which enrolled devices can reach. Don't use `localhost`.

Make sure the Fleet server URL is saved before you enroll hosts. Fleet uses it in the agent packages it generates, and a package built without it can't find your Fleet server.

That's it! Fleet is running.

## Enroll hosts

To enroll a host, you install fleetd, Fleet's agent, on it. Because `fleetlab` uses a self-signed certificate, you'll bundle the certificate with each agent package so hosts can verify your Fleet server.

Install fleetctl, Fleet's command-line tool, to generate agent packages:

```
npm install -g fleetctl
```

In Fleet, select **Hosts** > **Add hosts** and choose your host's platform. Copy the `fleetctl package` command, add the `--fleet-certificate` flag, and check that `--fleet-url` uses your Mac's LAN IP address. For example, for a macOS host:

```
fleetctl package --type=pkg --fleet-desktop \
  --fleet-url=https://<insert IP address>:1337 \
  --enroll-secret=YOUR_ENROLL_SECRET \
  --fleet-certificate=certs/fleet.crt
```

Run the command from your `~/fleet-lab` directory. With `--fleet-certificate`, fleetd trusts your certificate without changes to the host's certificate store. Learn more in [Certificates in fleetd](https://fleetdm.com/guides/certificates-in-fleetd).

Copy the package to your host and install it. On macOS, double-click the `.pkg` file, or run:

```
sudo installer -pkg fleet-osquery.pkg -target /
```

The host appears on the **Hosts** page within a few minutes.

> **Testing with a virtual machine on the same Mac?** Set the VM to bridged networking, so it reaches Fleet over your network like a separate computer. Also shut the VM down before restarting Apple container's services (`container system stop`), because the restart can break the VM's network connection.

## Manage your Fleet lab

Run every command from your `~/fleet-lab` directory.

| Command | What it does |
| --- | --- |
| `./fleetlab.sh setup` | One-time setup. Safe to run again |
| `./fleetlab.sh up` | Deploys Fleet for the first time and waits until it's ready |
| `./fleetlab.sh start` | Starts an existing deployment in the right order |
| `./fleetlab.sh stop` | Stops all services. Your data is kept |
| `./fleetlab.sh status` | Shows the containers and whether Fleet is answering |
| `./fleetlab.sh logs` | Follows Fleet's logs. Use `logs mysql` or `logs redis` for the other services |
| `./fleetlab.sh update` | Pulls the latest Fleet image and recreates the Fleet container |
| `./fleetlab.sh certs` | Creates a new TLS certificate and restarts Fleet with it. Use it when your IP address changes or to renew the certificate |
| `./fleetlab.sh down` | Removes the containers. Your data is kept |
| `./fleetlab.sh reset` | Removes the containers and all data (asks first) |

Apple container doesn't restart containers automatically. After you restart your Mac, run `./fleetlab.sh start`.

Fleet automatically runs database migrations when you run `update`. Your data is kept.

To start over with a fresh Fleet, run `reset`, then `up`:

```
./fleetlab.sh reset
./fleetlab.sh up
```

## Manage the certificate

`fleetlab` creates a self-signed certificate valid for 365 days. It covers your Mac's hostname, `localhost`, `fleet.test`, `127.0.0.1`, and your Mac's LAN IP address. Check what it covers and when it expires:

```
openssl x509 -in certs/fleet.crt -noout -text | grep -A1 "Subject Alternative Name"
openssl x509 -in certs/fleet.crt -noout -enddate
```

Run `./fleetlab.sh certs` to renew the certificate, or when your Mac's LAN IP address changes. Then generate new agent packages and reinstall them on your hosts, because fleetd only trusts the certificate bundled in its package. If your IP address changed, also update the Fleet server URL in **Settings** > **Organization settings** before generating packages. A DHCP reservation for your Mac avoids IP changes.

To use fleetctl with your Fleet lab, point it at your certificate, then log in:

```
fleetctl config set --address https://localhost:1337 --rootca certs/fleet.crt
fleetctl login
```

To stop your browser warning about the certificate, trust it in your Mac's System keychain:

```
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain certs/fleet.crt
```

This self-signed certificate works for enrolling hosts with fleetd. Fleet's MDM features, like turning on Apple MDM, have additional certificate requirements and aren't covered in this lab.

## What's running

Each service runs in its own lightweight Linux virtual machine on a private network called `fleet-net`. Containers find each other through a local DNS domain (`test`) that `setup` creates.

**fleet-mysql** stores all Fleet data (devices, policies, queries, users). It runs natively on Apple silicon.

**fleet-redis** handles background jobs and caching. It runs natively on Apple silicon.

**fleet** is the main application. It serves the web UI and API, and handles device connections. Fleet publishes only an amd64 image, so it runs under Rosetta with 4 GB of memory.

Fleet's data lives in five named volumes, which are kept when you stop or remove the containers. Devices connect to Fleet on port 1337. If needed, allow inbound connections on port 1337 in your Mac's firewall.

## Configuration options

All Fleet settings live in `.env`, and work the same as in the [Docker Compose guide](https://fleetdm.com/guides/deploy-fleet-on-docker-compose). After changing a Fleet setting, recreate the Fleet container with `./fleetlab.sh update`.

These additional settings control `fleetlab` itself. Add them to `.env`:

| Setting | Default | Description |
| --- | --- | --- |
| `FLEET_IMAGE` | `fleetdm/fleet:latest` | The Fleet image to run. Pin a version for repeatable labs |
| `FLEET_VULNERABILITIES_CURRENT_INSTANCE_CHECKS` | `yes` | Set to `no` to turn off vulnerability processing and reduce memory use |

## Troubleshooting

`fleetlab` stops with an explanation and a suggested fix when setup or startup fails. For anything else, start with Fleet's logs:

```
./fleetlab.sh logs
```

**Fleet logs `lookup fleet-mysql.test ... i/o timeout`**

Apple container's DNS service isn't answering. Restart the Apple container services, then start Fleet:

```
container system stop
container system start
./fleetlab.sh start
```

**Fleet stops responding after about an hour, with nothing in the logs**

Fleet's vulnerability processing ran out of memory. Set `FLEET_VULNERABILITIES_CURRENT_INSTANCE_CHECKS=no` in `.env`, then run `./fleetlab.sh update`.

**Browser can't connect, or curl reports `tlsv1 alert protocol version`**

`FLEET_SERVER_TLS` is set to `false` in `.env`, so Fleet is serving plain HTTP. Set `FLEET_SERVER_TLS=true` and run `./fleetlab.sh update`. fleetctl and host enrollment require HTTPS.

**Hosts don't appear after installing the agent package**

Check the connection from the host:

```
curl -vk --max-time 15 https://<Mac's LAN IP address>:1337/healthz
```

If it fails with `No route to host`, a connection reset, or a timeout, check that port 1337 is reachable from the host. A VM on the same Mac should use bridged networking. On macOS hosts, fleetd's logs are in `/var/log/orbit/orbit.stderr.log`.

**fleetd reports `http: no Host in request URL`**

The agent package was built without a Fleet server URL. Set the Fleet server URL in **Settings** > **Organization settings**, or pass `--fleet-url` to `fleetctl package`. Then generate a new package and reinstall it.

**fleetd reports `x509: certificate signed by unknown authority`**

The agent package was built without `--fleet-certificate`, or with an older certificate than the one Fleet is serving. Generate a new package with `--fleet-certificate=certs/fleet.crt` and reinstall it.

**fleetd reports `x509: certificate is valid for ..., not ...` or `doesn't contain any IP SANs`**

The Fleet server URL uses an address your certificate doesn't cover. Run `./fleetlab.sh certs`, then generate and reinstall agent packages.

## The fleetlab script

Save this script as `fleetlab.sh` in your `~/fleet-lab` directory.

```bash
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
    curl -fL "$url" -o /tmp/container-installer.pkg
    sudo installer -pkg /tmp/container-installer.pkg -target /
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
    -e FLEET_SERVER_ADDRESS="${FLEET_SERVER_ADDRESS:-0.0.0.0}:${FLEET_PORT}" \
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
  if container volume ls 2>/dev/null | awk 'NR>1 {print $1}' | grep -qx fleet-mysql; then
    say "Reusing Fleet data from a previous deployment (its passwords must match .env)"
  fi

  say "Creating network and volumes"
  container network create "$NET" >/dev/null 2>&1 || true
  local v
  for v in "${VOLUMES[@]}"; do container volume create "$v" >/dev/null 2>&1 || true; done

  say "Starting MySQL and Redis"
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
  cmd_stop
  local c
  for c in "${CONTAINERS[@]}"; do container rm "$c" >/dev/null 2>&1 || true; done
  ok "Containers removed (data kept). Redeploy with ./fleetlab.sh up"
}

cmd_reset() {
  read -r -p "Delete ALL Fleet data (database, logs, settings)? [y/N] " a
  [[ "$a" =~ ^[Yy]$ ]] || { echo "Cancelled."; exit 0; }
  cmd_down
  local v
  for v in "${VOLUMES[@]}"; do container volume rm "$v" >/dev/null 2>&1 || true; done
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
```







<meta name="articleTitle" value="Deploy Fleet with Apple Containers">
<meta name="authorGitHubUsername" value="adriant172">
<meta name="authorFullName" value="Matthew Thomas">
<meta name="publishedOn" value="2026-10-29">
<meta name="category" value="guides">
<meta name="description" value="Learn how to deploy Fleet using Apple Containerand fleetlab.">
