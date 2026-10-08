# Deploy Socket Firewall Enterprise in wrapper mode with Fleet

In wrapper mode, [Socket Firewall Enterprise](https://docs.socket.dev/docs/socket-firewall-enterprise-wrapper-mode) runs on each developer's machine as the `sfw` command, the same way Socket Firewall Free does, but with your organization's Socket API key, security policies, allow lists, dashboard reporting, and support for more ecosystems. This guide uses Fleet to install the Enterprise `sfw` binary on macOS, Windows, and Linux, deliver the API key from a Fleet secret, verify both with policies, and repair them automatically.

Wrapper mode still depends on developers running `sfw npm install` instead of `npm install`, or on the shell aliases in step 5. If you need protection that developers can't bypass, use [registry mode](https://fleetdm.com/guides/deploy-socket-firewall-enterprise-registry-mode-with-fleet) instead.

## Prerequisites

- A Socket Firewall Enterprise license. Socket's [firewall-release wiki](https://github.com/SocketDev/firewall-release/wiki) states that the software requires a paid license.
- A Socket API key with the `packages` and `entitlements:list` scopes, created in your Socket organization.
- Hosts enrolled in Fleet with scripts enabled. See the [scripts guide](https://fleetdm.com/guides/scripts) if you deployed `fleetd` without `--enable-scripts`.
- Fleet Premium, for the policy automations in step 4 and for masking the API key as a Fleet secret.
- Hosts that can reach `github.com`, where Socket publishes the binaries as [GitHub release assets](https://github.com/SocketDev/firewall-release/releases), and Socket's API at `api.socket.dev`.
- A GitOps repository connected to Fleet. The API key is stored as a `FLEET_SECRET_` variable, which Fleet masks in the UI and API.

> **Warning:** The API key ends up in plain text on each host, in a file readable by that user on macOS and Linux, and in a machine environment variable on Windows. Fleet masks the value on the server side, not on the host. Use a key scoped to only what the wrapper needs, and rotate it if a host is lost.

## Step 1: Pin a release and record its checksums and sizes

Pin one version fleet-wide rather than installing whatever is `latest`, so every host runs the same build and Fleet can verify it. The install scripts verify the download's SHA-256, and the policies in step 4 check the installed binary's size against the release asset, because osquery's `hash` table skips files larger than 50 MB by default and `sfw` is well past that. Socket doesn't publish checksums for the release assets, so compute them yourself when you adopt a version:

```bash
SFW_VERSION="1.15.2"
for asset in sfw-macos-arm64 sfw-macos-x86_64 sfw-linux-x86_64 sfw-linux-arm64 sfw-windows-x86_64.exe sfw-windows-arm64.exe; do
  curl -fsSL -o "$asset" "https://github.com/SocketDev/firewall-release/releases/download/v${SFW_VERSION}/${asset}"
done
shasum -a 256 sfw-*
ls -l sfw-*
```

These are the checksums and sizes for v1.15.2, published 2026-09-15:

| Asset | SHA-256 | Size (bytes) |
|-------|---------|--------------|
| `sfw-macos-arm64` | `7fea0f5dcf14a158f009ab2906eeed853e624965390d914fa733f03d7f4780d0` | 133504944 |
| `sfw-macos-x86_64` | `53691eba2c1b9098c3c1be07bd2fc73662bcf21be31f829822a29ae2b06a520a` | 135838048 |
| `sfw-linux-x86_64` | `48dad19367ca076ffdad0b3d1b9df7bd1c381ae9b222b2fe5e132d690d660288` | 145689792 |
| `sfw-linux-arm64` | `dacd379481777f7afada49f18f76a6beaf1323ca007e48d2bb8aac790ff058d9` | 142675072 |
| `sfw-windows-x86_64.exe` | `6d4ae4a450b2c596e8db08ab06215dedc6daa6d40e5ebc235b1c1b6a72080bae` | 106127424 |
| `sfw-windows-arm64.exe` | `a6d843238d048ffadbb00621f37ee1de9ba5964140b2f9392a14d5dce4d70966` | 95416896 |

When you adopt a newer release, update the version and checksums in the install scripts (step 2) and the sizes in the policies (step 4) in the same change. Hosts on the old build fail the updated policy and Fleet reinstalls them.

> **Note:** Socket also ships `sfw-musl-linux-*` assets for Alpine and other musl-based distributions. The Linux script below installs the glibc build.

## Step 2: Write the install and uninstall scripts

Fleet runs shell scripts as root on macOS and Linux and PowerShell scripts as SYSTEM on Windows, so these scripts install `sfw` machine-wide. If a host already has Socket Firewall Free at the same path, the Enterprise binary replaces it.

### macOS and Linux

Save this as `install-socket-firewall.sh`:

```bash
#!/bin/bash
# Installs Socket Firewall Enterprise (wrapper mode) to /usr/local/bin/sfw.
# Pins a release and verifies its SHA-256 before installing.
set -euo pipefail

SFW_VERSION="1.15.2"
INSTALL_PATH="/usr/local/bin/sfw"

case "$(uname -s)-$(uname -m)" in
  Darwin-arm64)              ASSET="sfw-macos-arm64";  SHA256="7fea0f5dcf14a158f009ab2906eeed853e624965390d914fa733f03d7f4780d0" ;;
  Darwin-x86_64)             ASSET="sfw-macos-x86_64"; SHA256="53691eba2c1b9098c3c1be07bd2fc73662bcf21be31f829822a29ae2b06a520a" ;;
  Linux-x86_64)              ASSET="sfw-linux-x86_64"; SHA256="48dad19367ca076ffdad0b3d1b9df7bd1c381ae9b222b2fe5e132d690d660288" ;;
  Linux-aarch64|Linux-arm64) ASSET="sfw-linux-arm64";  SHA256="dacd379481777f7afada49f18f76a6beaf1323ca007e48d2bb8aac790ff058d9" ;;
  *) echo "Unsupported platform: $(uname -s) $(uname -m)" >&2; exit 1 ;;
esac

URL="https://github.com/SocketDev/firewall-release/releases/download/v${SFW_VERSION}/${ASSET}"
TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

echo "Downloading ${ASSET} v${SFW_VERSION}..."
curl -fsSL --retry 3 -o "$TMP" "$URL"

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "$TMP" | awk '{print $1}')"
else
  ACTUAL="$(shasum -a 256 "$TMP" | awk '{print $1}')"
fi
if [ "$ACTUAL" != "$SHA256" ]; then
  echo "Checksum mismatch for ${ASSET}: expected ${SHA256}, got ${ACTUAL}" >&2
  exit 1
fi

mkdir -p "$(dirname "$INSTALL_PATH")"
install -m 0755 "$TMP" "$INSTALL_PATH"
"$INSTALL_PATH" --version
```

The last line prints `Socket Firewall Enterprise, version 1.15.2` into the script output in Fleet.

> **Note:** The binaries are ad-hoc signed. Gatekeeper only blocks them when a browser download sets the `com.apple.quarantine` attribute, which `curl` doesn't, so the copy Fleet installs runs without the `xattr` step in Socket's docs.

Save this as `uninstall-socket-firewall.sh`. It removes the binary, the aliases from step 5, and each user's configuration file:

```bash
#!/bin/bash
# Removes Socket Firewall Enterprise, its shell aliases, and per-user config.
set -euo pipefail

rm -f /usr/local/bin/sfw
rm -rf /etc/socket-firewall
for cfg in /Users/*/.sfw.config /home/*/.sfw.config; do
  [ -f "$cfg" ] && rm -f "$cfg"
done
echo "Socket Firewall Enterprise removed."
```

### Windows

Save this as `install-socket-firewall.ps1`. It installs to **Program Files** and adds that folder to the machine `Path`, so `sfw` is available to every user:

```powershell
# Installs Socket Firewall Enterprise (wrapper mode) to
# C:\Program Files\Socket Firewall\sfw.exe and adds that folder to the machine PATH.
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$SfwVersion = "1.15.2"
$InstallDir = "C:\Program Files\Socket Firewall"
$ExePath    = Join-Path $InstallDir "sfw.exe"

switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { $Asset = "sfw-windows-x86_64.exe"; $Sha256 = "6d4ae4a450b2c596e8db08ab06215dedc6daa6d40e5ebc235b1c1b6a72080bae" }
  "ARM64" { $Asset = "sfw-windows-arm64.exe";  $Sha256 = "a6d843238d048ffadbb00621f37ee1de9ba5964140b2f9392a14d5dce4d70966" }
  default { Write-Host "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"; exit 1 }
}

$Url = "https://github.com/SocketDev/firewall-release/releases/download/v$SfwVersion/$Asset"
$Tmp = Join-Path $env:TEMP "sfw-download.exe"

try {
  Write-Host "Downloading $Asset v$SfwVersion..."
  [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
  Invoke-WebRequest -Uri $Url -OutFile $Tmp -UseBasicParsing

  $Actual = (Get-FileHash -Path $Tmp -Algorithm SHA256).Hash.ToLower()
  if ($Actual -ne $Sha256) {
    throw "Checksum mismatch for ${Asset}: expected $Sha256, got $Actual"
  }

  New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
  Move-Item -Path $Tmp -Destination $ExePath -Force

  $MachinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
  if (($MachinePath -split ";") -notcontains $InstallDir) {
    [Environment]::SetEnvironmentVariable("Path", "$MachinePath;$InstallDir", "Machine")
  }

  & $ExePath --version
  exit 0
} catch {
  Write-Host "Error: $_"
  if (Test-Path $Tmp) { Remove-Item $Tmp -Force }
  exit 1
}
```

Save this as `uninstall-socket-firewall.ps1`:

```powershell
# Removes Socket Firewall Enterprise, its PATH entry, and its machine environment variables.
$ErrorActionPreference = "Stop"

$InstallDir = "C:\Program Files\Socket Firewall"

try {
  if (Test-Path $InstallDir) { Remove-Item -Path $InstallDir -Recurse -Force }

  $MachinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
  $NewPath = ($MachinePath -split ";" | Where-Object { $_ -and $_ -ne $InstallDir }) -join ";"
  if ($NewPath -ne $MachinePath) {
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "Machine")
  }

  foreach ($Name in @("SOCKET_API_KEY", "SFW_UNKNOWN_HOST_ACTION")) {
    [Environment]::SetEnvironmentVariable($Name, $null, "Machine")
  }

  Write-Host "Socket Firewall Enterprise removed."
  exit 0
} catch {
  Write-Host "Error: $_"
  exit 1
}
```

## Step 3: Deliver the API key and configuration

The wrapper reads its settings from the `SOCKET_API_KEY` environment variable or from a `.sfw.config` file in the user's home directory, in dotenv format. Store the key in Fleet as a secret named `FLEET_SECRET_SOCKET_API_KEY`. Fleet replaces `$FLEET_SECRET_SOCKET_API_KEY` in a script with the value when it sends the script to a host, and masks the value everywhere in the Fleet UI and API. See [Secrets in scripts and configuration profiles](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles) for how to define the secret in your GitOps workflow.

Both scripts below also set `SFW_UNKNOWN_HOST_ACTION=block`, which is Socket's default. Other settings from Socket's [configuration reference](https://docs.socket.dev/docs/socket-firewall-enterprise-wrapper-mode-configuration) go in the same place:

| Variable | Purpose |
|----------|---------|
| `SFW_UNKNOWN_HOST_ACTION` | `block`, `warn`, or `ignore` traffic to hosts the wrapper doesn't recognize. |
| `SFW_CUSTOM_REGISTRIES` | Comma-separated `kind:fqdn/prefix` entries for private mirrors, for example `npm:packages.example.com/npm-mirror`. |
| `SFW_TELEMETRY_DISABLED` | `true` to stop sending telemetry to Socket. |
| `SFW_JSON_REPORT_PATH` | Where to write a JSON report of blocked packages. |
| `SFW_REPORT_MESSAGE` | Text shown to developers when a package is blocked, such as how to request an exception. |
| `SFW_DEBUG` | `true` for verbose diagnostics. |

### macOS and Linux

Fleet runs this script as root, so it writes a `.sfw.config` for every local home directory and hands ownership to that user. Save it as `configure-socket-firewall.sh`:

```bash
#!/bin/bash
# Writes ~/.sfw.config for every local user so sfw can find the Socket API key.
# $FLEET_SECRET_SOCKET_API_KEY is replaced by Fleet before the script reaches the host.
set -euo pipefail

API_KEY="$FLEET_SECRET_SOCKET_API_KEY"

case "$(uname -s)" in
  Darwin) HOMES=(/Users/*) ;;
  Linux)  HOMES=(/home/*) ;;
  *) echo "Unsupported platform: $(uname -s)" >&2; exit 1 ;;
esac

umask 077
for home in "${HOMES[@]}"; do
  [ -d "$home" ] || continue
  case "$home" in /Users/Shared|/home/lost+found) continue ;; esac
  owner="$(stat -f %Su "$home" 2>/dev/null || stat -c %U "$home")"
  cfg="$home/.sfw.config"
  cat > "$cfg" <<EOF
# Managed by Fleet. Socket Firewall Enterprise configuration.
SOCKET_API_KEY=${API_KEY}
SFW_UNKNOWN_HOST_ACTION=block
EOF
  chown "$owner" "$cfg"
  chmod 0600 "$cfg"
  echo "Wrote ${cfg}"
done
```

> **Warning:** The script overwrites any existing `~/.sfw.config`. Developers who need per-project settings can set `SFW_CONFIG_RELATIVE_PATHS`, which makes `sfw` also look for a `.sfw.config` in the current and parent directories, instead of editing the managed file.

A user created after this script runs has no configuration until it runs again. The policy in step 4 fails in that case, and its automation re-runs the script.

### Windows

Windows has a machine-wide environment, so set the variables there instead of touching each profile. Save this as `configure-socket-firewall.ps1`:

```powershell
# Sets the Socket API key machine-wide so sfw can find it for every user.
# $FLEET_SECRET_SOCKET_API_KEY is replaced by Fleet before the script reaches the host.
$ErrorActionPreference = "Stop"

try {
  [Environment]::SetEnvironmentVariable("SOCKET_API_KEY", "$FLEET_SECRET_SOCKET_API_KEY", "Machine")
  [Environment]::SetEnvironmentVariable("SFW_UNKNOWN_HOST_ACTION", "block", "Machine")
  Write-Host "Socket Firewall configuration applied."
  exit 0
} catch {
  Write-Host "Error: $_"
  exit 1
}
```

> **Note:** Every account on the host can read a machine environment variable. On shared Windows hosts, write a `.sfw.config` into each profile under `C:\Users` instead and restrict it with `icacls`, following the shape of the macOS and Linux script.

### Register the scripts

Save all six scripts under `lib/socket-firewall/` in your GitOps repository and register them in `fleets/<name>.yml`:

```yaml
controls:
  scripts:
    - path: ../lib/socket-firewall/install-socket-firewall.sh
    - path: ../lib/socket-firewall/configure-socket-firewall.sh
    - path: ../lib/socket-firewall/uninstall-socket-firewall.sh
    - path: ../lib/socket-firewall/install-socket-firewall.ps1
    - path: ../lib/socket-firewall/configure-socket-firewall.ps1
    - path: ../lib/socket-firewall/uninstall-socket-firewall.ps1
```

In the Fleet UI, go to **Controls > Scripts**, select the fleet, and upload each script. To run one right away, open a host's details page and select **Actions > Run script**.

## Step 4: Verify and self-heal with policies

Two policies per platform: one confirms the pinned binary is installed by checking its size against the release asset, and one confirms the configuration is in place. Each has a `run_script` automation that repairs the host when it newly fails.

Add to `fleets/<name>.yml`:

```yaml
policies:
  - name: Socket Firewall Enterprise installed and current (macOS)
    platform: darwin
    description: Checks that the pinned Socket Firewall Enterprise release is installed at /usr/local/bin/sfw.
    resolution: Fleet reinstalls Socket Firewall automatically. To install it yourself, run the install-socket-firewall.sh script from Fleet.
    query: "SELECT 1 FROM file WHERE path = '/usr/local/bin/sfw' AND size IN (133504944, 135838048);"
    run_script:
      path: ../lib/socket-firewall/install-socket-firewall.sh
  - name: Socket Firewall Enterprise configured for every user (macOS)
    platform: darwin
    description: Checks that every home directory under /Users has a .sfw.config file.
    resolution: Fleet writes the configuration automatically. To apply it yourself, run the configure-socket-firewall.sh script from Fleet.
    query: "SELECT 1 WHERE (SELECT COUNT(*) FROM file WHERE path LIKE '/Users/%' AND type = 'directory' AND path NOT IN ('/Users/Shared', '/Users/Shared/')) = (SELECT COUNT(*) FROM file WHERE path LIKE '/Users/%/.sfw.config');"
    run_script:
      path: ../lib/socket-firewall/configure-socket-firewall.sh
  - name: Socket Firewall Enterprise installed and current (Linux)
    platform: linux
    description: Checks that the pinned Socket Firewall Enterprise release is installed at /usr/local/bin/sfw.
    resolution: Fleet reinstalls Socket Firewall automatically. To install it yourself, run the install-socket-firewall.sh script from Fleet.
    query: "SELECT 1 FROM file WHERE path = '/usr/local/bin/sfw' AND size IN (145689792, 142675072);"
    run_script:
      path: ../lib/socket-firewall/install-socket-firewall.sh
  - name: Socket Firewall Enterprise configured for every user (Linux)
    platform: linux
    description: Checks that every home directory under /home has a .sfw.config file.
    resolution: Fleet writes the configuration automatically. To apply it yourself, run the configure-socket-firewall.sh script from Fleet.
    query: "SELECT 1 WHERE (SELECT COUNT(*) FROM file WHERE path LIKE '/home/%' AND type = 'directory' AND path NOT IN ('/home/lost+found', '/home/lost+found/')) = (SELECT COUNT(*) FROM file WHERE path LIKE '/home/%/.sfw.config');"
    run_script:
      path: ../lib/socket-firewall/configure-socket-firewall.sh
  - name: Socket Firewall Enterprise installed and current (Windows)
    platform: windows
    description: Checks that the pinned Socket Firewall Enterprise release is installed at C:\Program Files\Socket Firewall\sfw.exe.
    resolution: Fleet reinstalls Socket Firewall automatically. To install it yourself, run the install-socket-firewall.ps1 script from Fleet.
    query: "SELECT 1 FROM file WHERE path = 'C:\\Program Files\\Socket Firewall\\sfw.exe' AND size IN (106127424, 95416896);"
    run_script:
      path: ../lib/socket-firewall/install-socket-firewall.ps1
  - name: Socket Firewall Enterprise configured (Windows)
    platform: windows
    description: Checks that the SOCKET_API_KEY machine environment variable is set.
    resolution: Fleet writes the configuration automatically. To apply it yourself, run the configure-socket-firewall.ps1 script from Fleet.
    query: "SELECT 1 FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment\\SOCKET_API_KEY';"
    run_script:
      path: ../lib/socket-firewall/configure-socket-firewall.ps1
```

To do the same in the Fleet UI, go to **Policies**, select the fleet, click **Add policy**, and paste each query. Then click **Manage automations > Run script**, check each policy, and pick the matching script.

A few things to know about these policies:

- The configuration policies compare the number of home directories with the number of `.sfw.config` files, since osquery can't read a file's contents. If your hosts keep home directories somewhere other than `/Users` or `/home`, adjust both sides of the comparison.
- The Windows configuration policy checks that the variable exists, not that it holds the current key. Rotating the key means re-running the configure script on every host, which you can do by editing the script and using **Reset policy** on the policy's details page.
- Size tells you which release is installed, not whether the file is intact. Integrity is checked once, by the SHA-256 comparison in the install script. To verify the hash in the policy too, raise `read_max` above the binary's size in your fleet's [agent options](https://fleetdm.com/docs/configuration/agent-configuration) and switch the query to the `hash` table, at the cost of hashing roughly 130 MB on every run.
- Bumping the pinned version in the scripts and policies together rolls a new release out. Every host fails the updated install policy once, and Fleet reinstalls.
- Policy automations require Fleet Premium and fleet-level policies. Fleet retries a failing script up to three times.

## Step 5 (optional): Route package managers through sfw by default

Socket documents shell aliases and PowerShell functions that add the `sfw` prefix automatically, so `npm install` runs `sfw npm install`. Deploy them with the scripts below. Aliases only apply to interactive shells, `command npm` or a full path bypasses them, and IDE tooling that calls package managers directly never sees them. Treat this as a convenience for developers, not as enforcement.

> **Note:** Socket's docs list Poetry and Go on macOS as unsupported in wrapper mode. Drop the `go` alias on macOS hosts if it causes problems.

### macOS and Linux

Save this as `configure-socket-firewall-aliases.sh` and add it to `controls.scripts`:

```bash
#!/bin/bash
# Adds shell aliases that run supported package managers through sfw.
set -euo pipefail

ALIASES="/etc/socket-firewall/aliases.sh"
mkdir -p /etc/socket-firewall
cat > "$ALIASES" <<'EOF'
# Managed by Fleet. Routes package managers through Socket Firewall Enterprise.
if command -v sfw >/dev/null 2>&1; then
  alias npm="sfw npm"
  alias yarn="sfw yarn"
  alias pnpm="sfw pnpm"
  alias pip="sfw pip"
  alias pip3="sfw pip3"
  alias uv="sfw uv"
  alias cargo="sfw cargo"
  alias go="sfw go"
  alias mvn="sfw mvn"
  alias gradle="sfw gradle"
  alias gem="sfw gem"
  alias bundle="sfw bundle"
  alias dotnet="sfw dotnet"
fi
EOF
chmod 0644 "$ALIASES"

LINE="[ -r ${ALIASES} ] && . ${ALIASES}"
for rc in /etc/zshrc /etc/bashrc /etc/zsh/zshrc /etc/bash.bashrc; do
  [ -f "$rc" ] || continue
  grep -qF "$ALIASES" "$rc" || printf '\n%s\n' "$LINE" >> "$rc"
done
echo "Socket Firewall aliases installed."
```

### Windows

Save this as `configure-socket-firewall-aliases.ps1`. It appends functions to the all-users profile for Windows PowerShell 5.1 and, if installed, PowerShell 7:

```powershell
# Adds PowerShell functions that run supported package managers through sfw.
$ErrorActionPreference = "Stop"

$Profiles = @(
  "$env:SystemRoot\System32\WindowsPowerShell\v1.0\Microsoft.PowerShell_profile.ps1",
  "$env:ProgramFiles\PowerShell\7\Microsoft.PowerShell_profile.ps1"
)

$Block = @'

# Managed by Fleet. Routes package managers through Socket Firewall Enterprise.
if (Get-Command sfw -ErrorAction SilentlyContinue) {
  function npm    { sfw npm @args }
  function yarn   { sfw yarn @args }
  function pnpm   { sfw pnpm @args }
  function pip    { sfw pip @args }
  function pip3   { sfw pip3 @args }
  function uv     { sfw uv @args }
  function cargo  { sfw cargo @args }
  function go     { sfw go @args }
  function mvn    { sfw mvn @args }
  function gradle { sfw gradle @args }
  function gem    { sfw gem @args }
  function bundle { sfw bundle @args }
  function dotnet { sfw dotnet @args }
}
'@

try {
  foreach ($Profile in $Profiles) {
    if (-not (Test-Path (Split-Path $Profile))) { continue }
    if ((Test-Path $Profile) -and (Select-String -Path $Profile -Pattern "Socket Firewall" -Quiet)) { continue }
    Add-Content -Path $Profile -Value $Block
  }
  Write-Host "Socket Firewall functions installed."
  exit 0
} catch {
  Write-Host "Error: $_"
  exit 1
}
```

## Verify

1. In Fleet, open a host's details page and check the **Policies** section. Both Socket Firewall policies for that platform should show **Yes** within an hour. Click **Refetch** to check sooner.
2. On the host, open a new terminal and run `sfw --version`. It prints `Socket Firewall Enterprise, version 1.15.2`.
3. Run `sfw npm install left-pad` in a scratch project. With a valid key, the install completes and the request appears in your Socket dashboard. Without a key, `sfw` stops with `Configuration Error` and `SOCKET_API_TOKEN must be a valid API token`.
4. If you deployed aliases, run `type npm` in bash or zsh. It prints `npm is an alias for sfw npm`. In PowerShell, `Get-Command npm` shows `CommandType: Function`.

## Troubleshoot

**The configuration policy fails on macOS even though every user has a `.sfw.config`.** A directory under `/Users` that isn't a home directory, such as a leftover from a deleted account, inflates the count. Remove it or exclude its path in the policy query the same way `/Users/Shared` is excluded.

**`sfw` reports `SOCKET_API_TOKEN must be a valid API token`.** The error names `SOCKET_API_TOKEN`, but the binary also accepts `SOCKET_API_KEY`, which is the name Socket's docs use and the scripts above set. On macOS and Linux, confirm `~/.sfw.config` exists for that user and contains `SOCKET_API_KEY=`. On Windows, open a new terminal, since machine environment variables only reach new processes, and check `$env:SOCKET_API_KEY`. Confirm the key has the `packages` and `entitlements:list` scopes in Socket.

**Checksum mismatch in the script output.** The version and checksum in the script disagree, or Socket re-published the asset. Recompute the SHA-256 and size and update the script and policy together.

**`sfw` is not recognized on Windows.** The `Path` change applies to new processes. Have the user reopen their terminal or sign out and back in.

**Requests to a private mirror are blocked as an unknown host.** Add the mirror with `SFW_CUSTOM_REGISTRIES` in the configuration, using the `kind:fqdn/prefix` format from Socket's docs, and don't include a trailing slash.

## Further reading

- [Socket Firewall Enterprise: wrapper mode](https://docs.socket.dev/docs/socket-firewall-enterprise-wrapper-mode) and [configuration reference](https://docs.socket.dev/docs/socket-firewall-enterprise-wrapper-mode-configuration)
- [Deploy Socket Firewall Enterprise in registry mode with Fleet](https://fleetdm.com/guides/deploy-socket-firewall-enterprise-registry-mode-with-fleet)
- [Deploy Socket Firewall Free with Fleet](https://fleetdm.com/guides/deploy-socket-firewall-free-with-fleet)
- [Secrets in scripts and configuration profiles](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles)
- [Scripts](https://fleetdm.com/guides/scripts) and [Automations](https://fleetdm.com/guides/automations) guides

<meta name="articleTitle" value="Deploy Socket Firewall Enterprise in wrapper mode with Fleet">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-09-17">
<meta name="description" value="Install the Socket Firewall Enterprise wrapper on macOS, Windows, and Linux with Fleet, deliver the API key as a secret, and verify with policies.">
