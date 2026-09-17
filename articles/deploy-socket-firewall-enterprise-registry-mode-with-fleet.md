# Deploy Socket Firewall Enterprise in registry mode with Fleet

In registry mode, [Socket Firewall Enterprise](https://docs.socket.dev/docs/socket-firewall-enterprise-registry-mode) runs as a Docker container that package managers use as their registry. Every download passes through it and is checked against Socket's security API, and developers don't have to remember a command prefix. This guide uses Fleet to install Docker where the firewall runs, deploy the firewall container on a Linux host, trust the firewall's certificate on macOS, Windows, and Linux workstations, point npm, pip, Cargo, and Go at the firewall machine-wide, and verify all of it with policies that self-heal.

The guide covers Socket's downstream topology, where workstations talk to the firewall directly. If you route through Artifactory or Nexus instead, follow Socket's [upstream deployment guide](https://docs.socket.dev/docs/socket-firewall-enterprise-registry-mode-upstream-deployment-guide) for the firewall itself. Steps 3 through 5 here still apply to the workstations.

## Prerequisites

- A Socket Firewall Enterprise license and a Socket API key with the `packages:list` and `entitlements:list` scopes.
- A Linux host to run the firewall, enrolled in Fleet with scripts enabled, with Docker 20.10 or later and Docker Compose 2.0 or later. Step 1 installs Docker if you don't have it.
- A DNS name for the firewall, such as `sfw.example.com`, that resolves from every workstation, and a TLS certificate for that name. A certificate from a public CA skips step 3. A certificate from your internal CA needs step 3 so package managers trust it.
- Workstations enrolled in Fleet with scripts enabled. See the [scripts guide](https://fleetdm.com/guides/scripts) if you deployed `fleetd` without `--enable-scripts`.
- Fleet Premium, for Fleet-maintained apps, Fleet secrets, and policy automations.
- A GitOps repository connected to Fleet. Every step also has a Fleet UI path.

> **Warning:** Socket's quick start shows a self-signed certificate and tells clients to set `strict-ssl false` for npm and `trusted-host` for pip. Both disable certificate verification, which makes the firewall trivial to impersonate on the network. This guide distributes your CA certificate instead and keeps verification on.

## Step 1: Install Docker where the firewall runs

Fleet maintains Docker Desktop for macOS and Windows, which is the quickest way to get Docker onto the machines of developers who want to run or test the firewall locally. Add it in `fleets/<name>.yml`:

```yaml
software:
  fleet_maintained_apps:
    - slug: docker-desktop/darwin
      self_service: true
    - slug: docker/windows
      self_service: true
```

In the Fleet UI, go to **Software**, select the fleet, click **Add software**, choose the **Fleet-maintained apps** tab, and add **Docker Desktop**. See the [Fleet-maintained apps guide](https://fleetdm.com/guides/fleet-maintained-apps) for details.

> **Note:** Docker Desktop runs inside a signed-in user's session, so a firewall container on a laptop stops when the user signs out and can't be started by a Fleet script, which runs as root or SYSTEM. Run the always-on firewall on a Linux host with Docker Engine, and use Docker Desktop for local testing.

For the Linux host, install Docker Engine and the Compose plugin from the distribution's repositories. Save this as `install-docker.sh` and run it on the host from Fleet, or add it as a script-only package:

```bash
#!/bin/bash
# Installs Docker Engine and the Compose plugin from distribution repositories.
set -euo pipefail

if command -v apt-get >/dev/null 2>&1; then
  export DEBIAN_FRONTEND=noninteractive
  apt-get update
  apt-get install -y docker.io docker-compose-v2 curl
elif command -v dnf >/dev/null 2>&1; then
  dnf install -y moby-engine docker-compose curl
else
  echo "No supported package manager found (apt-get or dnf)." >&2
  exit 1
fi

systemctl enable --now docker
docker compose version
```

The package names above are for Ubuntu 24.04 and later, Debian 13 and later, and Fedora. On RHEL and its derivatives, use Docker's own repository following [Docker's install docs](https://docs.docker.com/engine/install/), since those distributions ship Podman instead.

## Step 2: Deploy the firewall container

Socket's [installation guide](https://docs.socket.dev/docs/socket-firewall-enterprise-registry-mode-installations) deploys the pre-built image with Docker Compose. The script below does the same thing from Fleet: it writes the `.env`, `socket.yml`, and `compose.yaml` files under `/opt/socket-firewall`, starts the container, and waits for the health check. The API key comes from a Fleet secret named `FLEET_SECRET_SOCKET_SECURITY_API_TOKEN`, which Fleet substitutes when it sends the script and masks in the UI and API. See [Secrets in scripts and configuration profiles](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles) for how to define it.

Put the certificate chain and private key at `/opt/socket-firewall/ssl/fullchain.pem` and `/opt/socket-firewall/ssl/privkey.pem` before running the script, for example from an ACME client on that host. The script refuses to start without them.

Save this as `deploy-socket-registry-firewall.sh` and change `SFW_HOST`:

```bash
#!/bin/bash
# Deploys Socket Registry Firewall with Docker Compose on this host.
# $FLEET_SECRET_SOCKET_SECURITY_API_TOKEN is replaced by Fleet before the script reaches the host.
set -euo pipefail

SFW_HOST="sfw.example.com"
DIR="/opt/socket-firewall"

mkdir -p "$DIR/ssl"
cd "$DIR"

for f in ssl/fullchain.pem ssl/privkey.pem; do
  [ -f "$f" ] || { echo "Missing ${DIR}/${f}. Install the TLS certificate first." >&2; exit 1; }
done
chmod 644 ssl/fullchain.pem ssl/privkey.pem

umask 077
cat > .env <<EOF
SOCKET_SECURITY_API_TOKEN=$FLEET_SECRET_SOCKET_SECURITY_API_TOKEN
EOF
umask 022

cat > socket.yml <<EOF
socket:
  api_url: https://api.socket.dev
ports:
  http: 8080
  https: 8443
path_routing:
  enabled: true
  domain: ${SFW_HOST}
  routes:
    - path: /npm
      upstream: https://registry.npmjs.org
      registry: npm
    - path: /pypi
      upstream: https://pypi.org
      registry: pypi
    - path: /cargo
      upstream: https://index.crates.io
      registry: cargo
    - path: /go
      upstream: https://proxy.golang.org
      registry: go
nginx:
  worker_processes: 2
  worker_connections: 4096
EOF

cat > compose.yaml <<'EOF'
services:
  socket-firewall:
    image: socketdev/socket-registry-firewall:latest
    ports:
      - "8080:8080"
      - "8443:8443"
    environment:
      - SOCKET_SECURITY_API_TOKEN=${SOCKET_SECURITY_API_TOKEN}
    volumes:
      - ./socket.yml:/app/socket.yml:ro
      - ./ssl:/etc/nginx/ssl
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-fk", "https://localhost:8443/health"]
      interval: 30s
      timeout: 10s
      retries: 3
EOF

docker compose pull
docker compose up -d

for _ in $(seq 1 30); do
  if curl -fsk https://localhost:8443/health >/dev/null 2>&1; then
    echo "Socket Registry Firewall is healthy."
    exit 0
  fi
  sleep 2
done
echo "Firewall did not become healthy. Run: docker compose -f ${DIR}/compose.yaml logs" >&2
exit 1
```

The `socket.yml` routes npm, PyPI, crates.io, and the Go module proxy. Socket also supports `maven`, `rubygems`, `nuget`, `conda`, and `openvsx` routes, and has options for fail-open behavior, caching, metadata filtering, and Splunk export in its [configuration reference](https://docs.socket.dev/docs/socket-firewall-enterprise-registry-mode-configuration-reference). Add what you need to the heredoc.

> **Note:** Socket's example pins nothing and pulls `latest`. For a production firewall, pin an image tag and bump it deliberately, the same way you would any other service.

Give the firewall host a manual label so the deploy script and its policy only target that host. Then register the script and a policy that checks the container is running:

```yaml
labels:
  - name: Socket Firewall host
    description: Linux hosts that run the Socket Registry Firewall container.
    label_membership_type: manual
    hosts:
      - sfw-01

controls:
  scripts:
    - path: ../lib/socket-firewall/deploy-socket-registry-firewall.sh

policies:
  - name: Socket Registry Firewall container running
    platform: linux
    description: Checks that the socketdev/socket-registry-firewall container is running on the firewall host.
    resolution: Fleet re-runs the deploy script automatically. To run it yourself, run deploy-socket-registry-firewall.sh from Fleet.
    query: "SELECT 1 FROM docker_containers WHERE image LIKE 'socketdev/socket-registry-firewall%' AND state = 'running';"
    labels_include_any:
      - Socket Firewall host
    run_script:
      path: ../lib/socket-firewall/deploy-socket-registry-firewall.sh
```

Run the script once from the host's details page with **Actions > Run script** to bring the firewall up the first time. Then from the workstation network:

```bash
curl -f https://sfw.example.com:8443/health
```

The response is `{"status":"healthy","version":"..."}`. If that only works with `curl -k`, your certificate isn't trusted yet, which step 3 fixes.

## Step 3: Trust the firewall's certificate on workstations

Skip this step if the firewall's certificate comes from a public CA. Otherwise, the operating system and two package managers each need to learn about your CA:

- macOS trusts a root certificate installed by a configuration profile from MDM.
- Windows and Linux trust a root certificate added to the system store by a script.
- npm and pip don't use the operating system store. npm reads `NPM_CONFIG_CAFILE` and pip reads `cert` in its config, so the scripts also save the CA as a plain file that step 4 points them at.

Export your CA certificate in PEM form. Then, for macOS, convert it to DER and base64 for the profile:

```bash
openssl x509 -in ca.pem -outform der -out ca.der
base64 -i ca.der
```

### macOS configuration profile

Save this as `socket-firewall-ca.mobileconfig`, paste the base64 output into the `PayloadContent` data element, and generate two UUIDs with `uuidgen`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key>
      <string>com.apple.security.root</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
      <key>PayloadIdentifier</key>
      <string>com.example.socket-firewall-ca.root</string>
      <key>PayloadUUID</key>
      <string>REPLACE-WITH-UUID-1</string>
      <key>PayloadDisplayName</key>
      <string>Socket Firewall CA</string>
      <key>PayloadCertificateFileName</key>
      <string>socket-firewall-ca.der</string>
      <key>PayloadContent</key>
      <data>
      PASTE-BASE64-DER-HERE
      </data>
    </dict>
  </array>
  <key>PayloadDisplayName</key>
  <string>Socket Firewall CA</string>
  <key>PayloadIdentifier</key>
  <string>com.example.socket-firewall-ca</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>REPLACE-WITH-UUID-2</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
  <key>PayloadScope</key>
  <string>System</string>
</dict>
</plist>
```

Register it under the fleet's macOS settings:

```yaml
controls:
  macos_settings:
    custom_settings:
      - path: ../lib/socket-firewall/socket-firewall-ca.mobileconfig
```

In the Fleet UI, go to **Controls > OS settings > Custom settings** and upload the profile. Socket's docs also show `security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain` for a one-off machine. The profile is the right tool for a managed fleet because it needs no interaction and is removed cleanly if you delete it.

### macOS and Linux script

Save this as `trust-socket-firewall-ca.sh` and paste your PEM certificate into the heredoc. On both platforms it writes `/etc/socket-firewall/ca.pem` for npm and pip. On Linux it also adds the CA to the system store:

```bash
#!/bin/bash
# Saves the Socket Registry Firewall CA for npm and pip, and adds it to the
# Linux system trust store. macOS system trust comes from the configuration profile.
set -euo pipefail

mkdir -p /etc/socket-firewall
cat > /etc/socket-firewall/ca.pem <<'EOF'
-----BEGIN CERTIFICATE-----
PASTE-YOUR-CA-CERTIFICATE-HERE
-----END CERTIFICATE-----
EOF
chmod 0644 /etc/socket-firewall/ca.pem

if [ "$(uname -s)" = "Linux" ]; then
  if command -v update-ca-certificates >/dev/null 2>&1; then
    cp /etc/socket-firewall/ca.pem /usr/local/share/ca-certificates/socket-firewall.crt
    update-ca-certificates
  elif command -v update-ca-trust >/dev/null 2>&1; then
    cp /etc/socket-firewall/ca.pem /etc/pki/ca-trust/source/anchors/socket-firewall.crt
    update-ca-trust extract
  else
    echo "No known CA trust tool found." >&2
    exit 1
  fi
fi
echo "Socket Firewall CA installed."
```

### Windows

Save this as `trust-socket-firewall-ca.ps1` and paste your PEM certificate into the here-string. It writes the file for npm and pip and imports the CA into the machine's **Trusted Root Certification Authorities** store:

```powershell
# Saves the Socket Registry Firewall CA for npm and pip, and adds it to the
# machine's Trusted Root Certification Authorities store.
$ErrorActionPreference = "Stop"

$Dir    = "C:\ProgramData\Socket Firewall"
$CaPath = Join-Path $Dir "ca.pem"

$Pem = @'
-----BEGIN CERTIFICATE-----
PASTE-YOUR-CA-CERTIFICATE-HERE
-----END CERTIFICATE-----
'@

try {
  New-Item -ItemType Directory -Path $Dir -Force | Out-Null
  Set-Content -Path $CaPath -Value $Pem -Encoding ASCII
  Import-Certificate -FilePath $CaPath -CertStoreLocation Cert:\LocalMachine\Root | Out-Null
  Write-Host "Socket Firewall CA installed."
  exit 0
} catch {
  Write-Host "Error: $_"
  exit 1
}
```

## Step 4: Point package managers at the firewall

Socket's [downstream guide](https://docs.socket.dev/docs/socket-firewall-enterprise-registry-mode-setup-downstream) shows the per-user commands for each package manager. The scripts below apply the same settings machine-wide, so they cover every user on the host and can be verified by a policy:

| Tool | Setting | Where the script puts it |
|------|---------|--------------------------|
| npm | `NPM_CONFIG_REGISTRY`, `NPM_CONFIG_CAFILE` | Environment variables |
| pip | `index-url`, `cert` | pip's global config file |
| Cargo | `CARGO_REGISTRIES_SOCKET_INDEX` | Environment variable |
| Go | `GOPROXY` | Environment variable |

On macOS and Linux, environment variables live in `/etc/socket-firewall/env.sh`, sourced from the system-wide shell startup files. On Windows they're machine environment variables. Terminals opened after the script runs pick them up. Applications launched from the Dock, Start menu, or a desktop session that started earlier don't see them until the user signs in again.

Two things to know about the settings themselves:

- Socket's Cargo example defines a named registry called `socket`. Cargo keeps using crates.io directly unless a project passes `--registry socket` or adds source replacement in its `.cargo/config.toml`. See [Cargo's source replacement docs](https://doc.rust-lang.org/cargo/reference/source-replacement.html) if you want the firewall to be the default.
- Socket's Go example also sets `GOSUMDB=off` and `GOINSECURE`. Both weaken Go's own verification, so this guide leaves them out. Add them only if module downloads fail through the firewall and you've weighed the trade-off.

### macOS and Linux

Save this as `configure-socket-firewall-clients.sh`. Change `SFW_HOST`, and set `CA_PATH` to an empty string if the firewall's certificate is publicly trusted:

```bash
#!/bin/bash
# Points npm, pip, Cargo, and Go at the Socket Registry Firewall for every user.
set -euo pipefail

SFW_HOST="sfw.example.com"
SFW_PORT="8443"
CA_PATH="/etc/socket-firewall/ca.pem"   # set to "" if the certificate is publicly trusted
BASE="https://${SFW_HOST}:${SFW_PORT}"

mkdir -p /etc/socket-firewall

ENV_FILE="/etc/socket-firewall/env.sh"
{
  echo "# Managed by Fleet. Routes package managers through Socket Registry Firewall."
  echo "export NPM_CONFIG_REGISTRY=\"${BASE}/npm/\""
  if [ -n "$CA_PATH" ]; then
    echo "export NPM_CONFIG_CAFILE=\"${CA_PATH}\""
  fi
  echo "export GOPROXY=\"${BASE}/go,direct\""
  echo "export CARGO_REGISTRIES_SOCKET_INDEX=\"${BASE}/cargo\""
} > "$ENV_FILE"
chmod 0644 "$ENV_FILE"

LINE="[ -r ${ENV_FILE} ] && . ${ENV_FILE}"
case "$(uname -s)" in
  Darwin)
    PIP_CONF="/Library/Application Support/pip/pip.conf"
    for rc in /etc/zshenv /etc/bashrc; do
      [ -f "$rc" ] || touch "$rc"
      grep -qF "$ENV_FILE" "$rc" || printf '\n%s\n' "$LINE" >> "$rc"
    done
    ;;
  Linux)
    PIP_CONF="/etc/pip.conf"
    ln -sf "$ENV_FILE" /etc/profile.d/socket-firewall.sh
    for rc in /etc/zsh/zshenv /etc/zshenv; do
      [ -f "$rc" ] || continue
      grep -qF "$ENV_FILE" "$rc" || printf '\n%s\n' "$LINE" >> "$rc"
    done
    ;;
  *) echo "Unsupported platform: $(uname -s)" >&2; exit 1 ;;
esac

mkdir -p "$(dirname "$PIP_CONF")"
{
  echo "# Managed by Fleet. Routes pip through Socket Registry Firewall."
  echo "[global]"
  echo "index-url = ${BASE}/pypi/simple"
  if [ -n "$CA_PATH" ]; then
    echo "cert = ${CA_PATH}"
  fi
} > "$PIP_CONF"
chmod 0644 "$PIP_CONF"

echo "Package managers now use ${BASE}."
```

### Windows

Save this as `configure-socket-firewall-clients.ps1`. Change `$SfwHost`, and set `$CaPath` to an empty string if the certificate is publicly trusted:

```powershell
# Points npm, pip, Cargo, and Go at the Socket Registry Firewall for every user.
$ErrorActionPreference = "Stop"

$SfwHost = "sfw.example.com"
$SfwPort = "8443"
$CaPath  = "C:\ProgramData\Socket Firewall\ca.pem"   # set to "" if the certificate is publicly trusted
$Base    = "https://${SfwHost}:${SfwPort}"

try {
  $Vars = @{
    "NPM_CONFIG_REGISTRY"           = "$Base/npm/"
    "GOPROXY"                       = "$Base/go,direct"
    "CARGO_REGISTRIES_SOCKET_INDEX" = "$Base/cargo"
  }
  if ($CaPath) { $Vars["NPM_CONFIG_CAFILE"] = $CaPath }
  foreach ($Name in $Vars.Keys) {
    [Environment]::SetEnvironmentVariable($Name, $Vars[$Name], "Machine")
  }

  $PipDir = "C:\ProgramData\pip"
  New-Item -ItemType Directory -Path $PipDir -Force | Out-Null
  $PipIni = @(
    "# Managed by Fleet. Routes pip through Socket Registry Firewall.",
    "[global]",
    "index-url = $Base/pypi/simple"
  )
  if ($CaPath) { $PipIni += "cert = $CaPath" }
  Set-Content -Path (Join-Path $PipDir "pip.ini") -Value $PipIni -Encoding ASCII

  Write-Host "Package managers now use $Base."
  exit 0
} catch {
  Write-Host "Error: $_"
  exit 1
}
```

### Register the scripts

Save the scripts under `lib/socket-firewall/` and register them in the workstation fleet's `fleets/<name>.yml`:

```yaml
controls:
  scripts:
    - path: ../lib/socket-firewall/trust-socket-firewall-ca.sh
    - path: ../lib/socket-firewall/configure-socket-firewall-clients.sh
    - path: ../lib/socket-firewall/trust-socket-firewall-ca.ps1
    - path: ../lib/socket-firewall/configure-socket-firewall-clients.ps1
```

In the Fleet UI, go to **Controls > Scripts**, select the fleet, and upload each script.

## Step 5: Verify and self-heal with policies

The client configuration files are identical on every host, so a policy can compare their SHA-256 against the known-good value and fail on any drift, including a developer pointing pip back at PyPI. Run the configure script on one host of each platform, then record the hashes:

```bash
shasum -a 256 /etc/socket-firewall/env.sh "/Library/Application Support/pip/pip.conf"
```

```powershell
Get-FileHash C:\ProgramData\pip\pip.ini
```

The `env.sh` and `pip.conf` contents are the same on macOS and Linux, so those two hashes are shared. Replace the placeholders below with your values and your CA's common name, then add to `fleets/<name>.yml`:

```yaml
policies:
  - name: Socket Registry Firewall configured (macOS)
    platform: darwin
    description: Checks that npm, pip, Cargo, and Go are configured to use the Socket Registry Firewall.
    resolution: Fleet reapplies the configuration automatically. To apply it yourself, run configure-socket-firewall-clients.sh from Fleet.
    query: "SELECT 1 WHERE EXISTS (SELECT 1 FROM hash WHERE path = '/etc/socket-firewall/env.sh' AND sha256 = '<env.sh sha256>') AND EXISTS (SELECT 1 FROM hash WHERE path = '/Library/Application Support/pip/pip.conf' AND sha256 = '<pip.conf sha256>');"
    run_script:
      path: ../lib/socket-firewall/configure-socket-firewall-clients.sh
  - name: Socket Registry Firewall CA trusted (macOS)
    platform: darwin
    description: Checks that the Socket Firewall CA is in the System keychain and saved for npm and pip.
    resolution: Fleet reinstalls the CA automatically. Confirm the socket-firewall-ca profile is delivered and run trust-socket-firewall-ca.sh from Fleet.
    query: "SELECT 1 WHERE EXISTS (SELECT 1 FROM certificates WHERE path = '/Library/Keychains/System.keychain' AND common_name = '<your CA common name>') AND EXISTS (SELECT 1 FROM file WHERE path = '/etc/socket-firewall/ca.pem');"
    run_script:
      path: ../lib/socket-firewall/trust-socket-firewall-ca.sh
  - name: Socket Registry Firewall configured (Linux)
    platform: linux
    description: Checks that npm, pip, Cargo, and Go are configured to use the Socket Registry Firewall.
    resolution: Fleet reapplies the configuration automatically. To apply it yourself, run configure-socket-firewall-clients.sh from Fleet.
    query: "SELECT 1 WHERE EXISTS (SELECT 1 FROM hash WHERE path = '/etc/socket-firewall/env.sh' AND sha256 = '<env.sh sha256>') AND EXISTS (SELECT 1 FROM hash WHERE path = '/etc/pip.conf' AND sha256 = '<pip.conf sha256>');"
    run_script:
      path: ../lib/socket-firewall/configure-socket-firewall-clients.sh
  - name: Socket Registry Firewall CA trusted (Linux)
    platform: linux
    description: Checks that the Socket Firewall CA is saved for npm and pip and added to the system trust store.
    resolution: Fleet reinstalls the CA automatically. To install it yourself, run trust-socket-firewall-ca.sh from Fleet.
    query: "SELECT 1 WHERE EXISTS (SELECT 1 FROM file WHERE path = '/etc/socket-firewall/ca.pem') AND EXISTS (SELECT 1 FROM file WHERE path IN ('/usr/local/share/ca-certificates/socket-firewall.crt', '/etc/pki/ca-trust/source/anchors/socket-firewall.crt'));"
    run_script:
      path: ../lib/socket-firewall/trust-socket-firewall-ca.sh
  - name: Socket Registry Firewall configured (Windows)
    platform: windows
    description: Checks that npm, pip, Cargo, and Go are configured to use the Socket Registry Firewall.
    resolution: Fleet reapplies the configuration automatically. To apply it yourself, run configure-socket-firewall-clients.ps1 from Fleet.
    query: "SELECT 1 WHERE EXISTS (SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment\\NPM_CONFIG_REGISTRY' AND data = 'https://sfw.example.com:8443/npm/') AND EXISTS (SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment\\GOPROXY' AND data = 'https://sfw.example.com:8443/go,direct') AND EXISTS (SELECT 1 FROM hash WHERE path = 'C:\\ProgramData\\pip\\pip.ini' AND sha256 = '<pip.ini sha256>');"
    run_script:
      path: ../lib/socket-firewall/configure-socket-firewall-clients.ps1
  - name: Socket Registry Firewall CA trusted (Windows)
    platform: windows
    description: Checks that the Socket Firewall CA is in the machine's Trusted Root store and saved for npm and pip.
    resolution: Fleet reinstalls the CA automatically. To install it yourself, run trust-socket-firewall-ca.ps1 from Fleet.
    query: "SELECT 1 WHERE EXISTS (SELECT 1 FROM certificates WHERE store_location = 'LocalMachine' AND store = 'Root' AND common_name = '<your CA common name>') AND EXISTS (SELECT 1 FROM file WHERE path = 'C:\\ProgramData\\Socket Firewall\\ca.pem');"
    run_script:
      path: ../lib/socket-firewall/trust-socket-firewall-ca.ps1
```

To do the same in the Fleet UI, go to **Policies**, select the fleet, click **Add policy**, and paste each query. Then click **Manage automations > Run script**, check each policy, and pick the matching script.

A few things to know about these policies:

- If the firewall uses a publicly trusted certificate, drop the two CA policies and the trust scripts.
- Changing `SFW_HOST`, the port, or the CA path changes the file contents. Recompute the hashes and update the policies in the same change as the script.
- Policy automations require Fleet Premium and fleet-level policies. They fire when a policy newly fails, and Fleet retries a failing script up to three times. Use **Reset policy** on a policy's details page to re-run it on hosts that already failed.
- Scope the workstation policies to developer hosts with `labels_include_any` if the fleet also contains hosts that never install packages.

## Verify

1. In Fleet, open a workstation's details page and check the **Policies** section. The Socket Registry Firewall policies for that platform should show **Yes** within an hour. Click **Refetch** to check sooner.
2. On the workstation, open a new terminal and confirm the settings landed:

   ```bash
   npm config get registry
   pip config list
   go env GOPROXY
   ```

   npm prints `https://sfw.example.com:8443/npm/`, pip prints the `global.index-url` and `global.cert` values, and Go prints `https://sfw.example.com:8443/go,direct`.
3. Install something. `npm install left-pad` in a scratch project completes and shows up in the Socket dashboard. Every response from the firewall carries an `X-Socket-Decision` header, which you can see with `curl -sI https://sfw.example.com:8443/npm/left-pad`.

## Troubleshoot

**npm fails with `UNABLE_TO_GET_ISSUER_CERT_LOCALLY` or `SELF_SIGNED_CERT_IN_CHAIN`.** npm doesn't use the operating system trust store. Confirm `NPM_CONFIG_CAFILE` is set in a fresh terminal and that the file it points to contains your CA. Don't fall back to `strict-ssl false`.

**pip fails with a certificate verify error.** pip also ignores the system store. Confirm `pip config list` shows `global.cert` pointing at the saved CA file. If a user's own `pip.conf` overrides the global one, the per-user file wins.

**Settings work in a terminal but not in an IDE.** Applications launched from a desktop session started before the script ran don't have the new environment. Have the user sign out and back in. On macOS, GUI apps don't read `/etc/zshenv` at all, so IDE-integrated package managers may need the registry set in the IDE's terminal settings or in the project's own config.

**The container policy fails on the firewall host.** Run `docker compose -f /opt/socket-firewall/compose.yaml logs` on the host. Socket's [installation guide](https://docs.socket.dev/docs/socket-firewall-enterprise-registry-mode-installations) covers failed health checks, missing certificates, and architecture mismatches.

**Cargo still downloads from crates.io.** The `socket` registry is defined but not the default. Use `--registry socket` or configure source replacement, as described in step 4.

**Go module downloads fail with checksum database errors.** The firewall doesn't proxy `sum.golang.org`. Socket's docs handle this with `GOSUMDB=off`. Weigh that trade-off before adding it to the environment file.

## Further reading

- [Socket Firewall Enterprise: registry mode](https://docs.socket.dev/docs/socket-firewall-enterprise-registry-mode), the [installation guide](https://docs.socket.dev/docs/socket-firewall-enterprise-registry-mode-installations), the [downstream deployment guide](https://docs.socket.dev/docs/socket-firewall-enterprise-registry-mode-setup-downstream), and the [configuration reference](https://docs.socket.dev/docs/socket-firewall-enterprise-registry-mode-configuration-reference)
- [Deploy Socket Firewall Enterprise in wrapper mode with Fleet](https://fleetdm.com/guides/deploy-socket-firewall-enterprise-wrapper-mode-with-fleet)
- [Deploy Socket Firewall Free with Fleet](https://fleetdm.com/guides/deploy-socket-firewall-free-with-fleet)
- [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps)
- [Secrets in scripts and configuration profiles](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles)
- [Scripts](https://fleetdm.com/guides/scripts) and [Automations](https://fleetdm.com/guides/automations) guides

<meta name="articleTitle" value="Deploy Socket Firewall Enterprise in registry mode with Fleet">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-09-17">
<meta name="description" value="Run Socket Registry Firewall with Docker, trust its certificate, and point npm, pip, Cargo, and Go at it on macOS, Windows, and Linux with Fleet.">
