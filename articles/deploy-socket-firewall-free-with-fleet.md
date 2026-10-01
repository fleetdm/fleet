# Deploy Socket Firewall Free with Fleet

[Socket Firewall Free](https://docs.socket.dev/docs/socket-firewall-free) is a command-line wrapper that blocks known-malicious packages before npm, yarn, pnpm, pip, uv, or cargo can download them. It works with no API key and no configuration. This guide uses Fleet to install the `sfw` binary on macOS, Windows, and Linux hosts, verify the install with policies, reinstall it automatically when it goes missing or falls behind, and optionally route package manager commands through it by default.

Socket Firewall Free only runs in wrapper mode, so a developer has to run `sfw npm install` instead of `npm install`. Fleet can put the binary in place and add shell aliases, but it can't stop someone from bypassing the wrapper. If you need enforcement, see [Deploy Socket Firewall Enterprise in registry mode with Fleet](https://fleetdm.com/guides/deploy-socket-firewall-enterprise-registry-mode-with-fleet).

## Prerequisites

- Hosts enrolled in Fleet with scripts enabled. See the [scripts guide](https://fleetdm.com/guides/scripts) if you deployed `fleetd` without `--enable-scripts`.
- Fleet Premium, if you want the policy automations in step 4. Installing the scripts on demand or through self-service works on Fleet Free.
- Hosts that can reach `github.com`. Socket publishes the binaries as [GitHub release assets](https://github.com/SocketDev/sfw-free/releases), and the hosts need network access at install time and every time `sfw` checks a package.
- A GitOps repository connected to Fleet, if you want to manage this in Git. Every step also has a Fleet UI path.

> **Note:** Socket Firewall Free always sends anonymous telemetry to Socket: a non-reversible machine identifier, blocked and permitted package names and versions, latency, errors, and the GitHub organization name from configured remotes. You can't turn it off. Read Socket's [telemetry section](https://docs.socket.dev/docs/socket-firewall-free) before you roll it out, and use the Enterprise wrapper if your organization needs telemetry controls.

## Step 1: Pin a release and record its checksums and sizes

Socket's install instructions download the `latest` release. A fleet-wide install should pin a version instead, so every host runs the same build and Fleet can verify it. The install scripts below download a specific tag and refuse to install a binary whose SHA-256 doesn't match. The policies in step 4 then check the installed binary's size against the release asset, because osquery's `hash` table skips files larger than 50 MB by default and `sfw` is well past that.

Socket doesn't publish checksums for its release assets, so compute them yourself when you adopt a version. From any machine:

```bash
SFW_VERSION="1.15.2"
for asset in sfw-free-macos-arm64 sfw-free-macos-x86_64 sfw-free-linux-x86_64 sfw-free-linux-arm64 sfw-free-windows-x86_64.exe sfw-free-windows-arm64.exe; do
  curl -fsSL -o "$asset" "https://github.com/SocketDev/sfw-free/releases/download/v${SFW_VERSION}/${asset}"
done
shasum -a 256 sfw-free-*
ls -l sfw-free-*
```

These are the checksums and sizes for v1.15.2, published 2026-09-15:

| Asset | SHA-256 | Size (bytes) |
|-------|---------|--------------|
| `sfw-free-macos-arm64` | `28c4d14ed5db09e3a3e299c02036ddaa524c5c476cb28e32deac4f77091acacb` | 127758784 |
| `sfw-free-macos-x86_64` | `3abd6086098e6ad604a8814cd6a5d9e5dd8e64c2108583f9e85c12e08baa7a26` | 130079504 |
| `sfw-free-linux-x86_64` | `fea8171808f9d913635c8f55fa70f7e72451cd38b0fca3d694e12ee1a9d7fc68` | 139988160 |
| `sfw-free-linux-arm64` | `d3e5490e7a1315ff2ba9bcc903d9d57de042d8dcde83dde1d59536725a470946` | 136973440 |
| `sfw-free-windows-x86_64.exe` | `8802ace1584212ed0361db6f4f12447f9f426b52c24eada54f7625910c3d5292` | 100409920 |
| `sfw-free-windows-arm64.exe` | `d762eb85db7b39f0d14f3724514e1eb45f86955b6bc7a9978bc82917549bf4eb` | 89699392 |

When you move to a newer release, update the version and checksums in the install scripts (step 2) and the sizes in the policies (step 4) in the same change. A host with an older build fails the updated policy and gets reinstalled.

> **Note:** Socket also ships `sfw-free-musl-linux-*` assets for Alpine and other musl-based distributions. The Linux script below installs the glibc build. Add a musl branch if you manage Alpine hosts.

## Step 2: Write the install and uninstall scripts

Fleet runs shell scripts as root on macOS and Linux, and PowerShell scripts as SYSTEM on Windows. That lets you install `sfw` machine-wide for every user instead of into one user's profile as Socket's own instructions do.

### macOS and Linux

One script covers both platforms because Fleet runs `.sh` scripts on macOS and Linux hosts. Save this as `install-socket-firewall.sh`:

```bash
#!/bin/bash
# Installs Socket Firewall Free to /usr/local/bin/sfw.
# Pins a release and verifies its SHA-256 before installing.
set -euo pipefail

SFW_VERSION="1.15.2"
INSTALL_PATH="/usr/local/bin/sfw"

case "$(uname -s)-$(uname -m)" in
  Darwin-arm64)              ASSET="sfw-free-macos-arm64";  SHA256="28c4d14ed5db09e3a3e299c02036ddaa524c5c476cb28e32deac4f77091acacb" ;;
  Darwin-x86_64)             ASSET="sfw-free-macos-x86_64"; SHA256="3abd6086098e6ad604a8814cd6a5d9e5dd8e64c2108583f9e85c12e08baa7a26" ;;
  Linux-x86_64)              ASSET="sfw-free-linux-x86_64"; SHA256="fea8171808f9d913635c8f55fa70f7e72451cd38b0fca3d694e12ee1a9d7fc68" ;;
  Linux-aarch64|Linux-arm64) ASSET="sfw-free-linux-arm64";  SHA256="d3e5490e7a1315ff2ba9bcc903d9d57de042d8dcde83dde1d59536725a470946" ;;
  *) echo "Unsupported platform: $(uname -s) $(uname -m)" >&2; exit 1 ;;
esac

URL="https://github.com/SocketDev/sfw-free/releases/download/v${SFW_VERSION}/${ASSET}"
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

The last line prints `Socket Firewall Free, version 1.15.2` into the script output in Fleet, which is a quick way to confirm the binary runs on that host.

> **Note:** Socket's docs tell individual users to clear the `com.apple.quarantine` attribute on macOS. That attribute is set by browsers, not by `curl`, so the copy Fleet installs isn't quarantined and Gatekeeper doesn't block it. The binaries are ad-hoc signed, so a developer who downloads one manually with a browser still hits that prompt.

Save this as `uninstall-socket-firewall.sh`:

```bash
#!/bin/bash
# Removes Socket Firewall Free and the optional shell aliases.
set -euo pipefail

rm -f /usr/local/bin/sfw
rm -rf /etc/socket-firewall
echo "Socket Firewall Free removed."
```

### Windows

Socket's instructions place `sfw.exe` in the current user's `WindowsApps` folder. Fleet runs as SYSTEM, so install to **Program Files** and add that folder to the machine `Path` instead, which makes `sfw` available to every user on the host. Save this as `install-socket-firewall.ps1`:

```powershell
# Installs Socket Firewall Free to C:\Program Files\Socket Firewall\sfw.exe
# and adds that folder to the machine PATH.
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$SfwVersion = "1.15.2"
$InstallDir = "C:\Program Files\Socket Firewall"
$ExePath    = Join-Path $InstallDir "sfw.exe"

switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { $Asset = "sfw-free-windows-x86_64.exe"; $Sha256 = "8802ace1584212ed0361db6f4f12447f9f426b52c24eada54f7625910c3d5292" }
  "ARM64" { $Asset = "sfw-free-windows-arm64.exe";  $Sha256 = "d762eb85db7b39f0d14f3724514e1eb45f86955b6bc7a9978bc82917549bf4eb" }
  default { Write-Host "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"; exit 1 }
}

$Url = "https://github.com/SocketDev/sfw-free/releases/download/v$SfwVersion/$Asset"
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

> **Note:** A machine `Path` change reaches new processes only. Terminals and editors that were already open keep the old `Path` until the user restarts them or signs out and back in.

Save this as `uninstall-socket-firewall.ps1`:

```powershell
# Removes Socket Firewall Free and its PATH entry.
$ErrorActionPreference = "Stop"

$InstallDir = "C:\Program Files\Socket Firewall"

try {
  if (Test-Path $InstallDir) { Remove-Item -Path $InstallDir -Recurse -Force }

  $MachinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
  $NewPath = ($MachinePath -split ";" | Where-Object { $_ -and $_ -ne $InstallDir }) -join ";"
  if ($NewPath -ne $MachinePath) {
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "Machine")
  }

  Write-Host "Socket Firewall Free removed."
  exit 0
} catch {
  Write-Host "Error: $_"
  exit 1
}
```

## Step 3: Add the scripts to Fleet

You can run the install script on demand from a host's details page, offer it through self-service, or attach it to a policy so Fleet installs `sfw` wherever it's missing. Step 4 covers the policy route. This step gets the scripts into Fleet.

### GitOps

Save the four scripts under `lib/socket-firewall/` in your GitOps repository and register them in the fleet that contains your developer hosts, in `fleets/<name>.yml`:

```yaml
controls:
  scripts:
    - path: ../lib/socket-firewall/install-socket-firewall.sh
    - path: ../lib/socket-firewall/uninstall-socket-firewall.sh
    - path: ../lib/socket-firewall/install-socket-firewall.ps1
    - path: ../lib/socket-firewall/uninstall-socket-firewall.ps1
```

To also let developers install it themselves from **Fleet Desktop > Self-service**, add the install scripts as script-only packages. The file's contents become the install script, and the uninstall script attaches alongside it:

```yaml
software:
  packages:
    - path: ../lib/socket-firewall/install-socket-firewall.sh
      display_name: Socket Firewall Free
      self_service: true
      uninstall_script:
        path: ../lib/socket-firewall/uninstall-socket-firewall.sh
    - path: ../lib/socket-firewall/install-socket-firewall.ps1
      display_name: Socket Firewall Free
      self_service: true
      uninstall_script:
        path: ../lib/socket-firewall/uninstall-socket-firewall.ps1
```

> **Note:** A script-only package declared with `path` can't be the target of a policy's `install_software` automation in GitOps. Fleet resolves that target by URL or hash, and a script package has neither. Uploading one with `automatic_install` is also rejected, because Fleet can't generate a detection policy for a script. Step 4 uses the `controls.scripts` entries above with `run_script` instead, which has neither limit. If you add the package through the Fleet UI, you can attach it to a custom policy's install automation there.

### Fleet UI

1. Go to **Controls > Scripts**, select the fleet, and upload each script.
2. To run one immediately, open a host's details page, select **Actions > Run script**, and pick the install script.

## Step 4: Verify and self-heal with policies

The policies below pass only when `/usr/local/bin/sfw` (or `sfw.exe` on Windows) exists and its size matches one of the pinned release builds for that platform. A host with no binary or a different release fails. With the `run_script` automation, Fleet runs the install script on any host that newly fails.

Add to `fleets/<name>.yml`:

```yaml
policies:
  - name: Socket Firewall Free installed and current (macOS)
    platform: darwin
    description: Checks that the pinned Socket Firewall Free release is installed at /usr/local/bin/sfw.
    resolution: Fleet reinstalls Socket Firewall Free automatically. To install it yourself, run the install-socket-firewall.sh script from Fleet.
    query: "SELECT 1 FROM file WHERE path = '/usr/local/bin/sfw' AND size IN (127758784, 130079504);"
    run_script:
      path: ../lib/socket-firewall/install-socket-firewall.sh
  - name: Socket Firewall Free installed and current (Linux)
    platform: linux
    description: Checks that the pinned Socket Firewall Free release is installed at /usr/local/bin/sfw.
    resolution: Fleet reinstalls Socket Firewall Free automatically. To install it yourself, run the install-socket-firewall.sh script from Fleet.
    query: "SELECT 1 FROM file WHERE path = '/usr/local/bin/sfw' AND size IN (139988160, 136973440);"
    run_script:
      path: ../lib/socket-firewall/install-socket-firewall.sh
  - name: Socket Firewall Free installed and current (Windows)
    platform: windows
    description: Checks that the pinned Socket Firewall Free release is installed at C:\Program Files\Socket Firewall\sfw.exe.
    resolution: Fleet reinstalls Socket Firewall Free automatically. To install it yourself, run the install-socket-firewall.ps1 script from Fleet.
    query: "SELECT 1 FROM file WHERE path = 'C:\\Program Files\\Socket Firewall\\sfw.exe' AND size IN (100409920, 89699392);"
    run_script:
      path: ../lib/socket-firewall/install-socket-firewall.ps1
```

To do the same in the Fleet UI, go to **Policies**, select the fleet, click **Add policy**, and paste each query. Then click **Manage automations > Run script**, check each policy, and pick the matching install script.

> **Note:** The `run_script` automation fires when a policy is newly failing, and Fleet retries a failing script up to three times. If a script fails on several hosts, fix the script and use **Reset policy** on the policy's details page to try again. Policy automations require Fleet Premium and fleet-level policies.

A few things to know about these policies:

- Size tells you which release is installed, not whether the file is intact. Integrity is checked once, by the SHA-256 comparison in the install script. If you want the policy to verify the hash too, raise `read_max` above the binary's size in your fleet's [agent options](https://fleetdm.com/docs/configuration/agent-configuration) and switch the query to the `hash` table. That hashes roughly 130 MB on every policy run.
- Because the policy fails until the size matches, bumping the version in the scripts and policies together is what rolls a new release out: every host fails the updated policy once, and Fleet reinstalls.
- Scope the policies to developer hosts with `labels_include_any` if the fleet also contains hosts that don't need `sfw`.

## Step 5 (optional): Route package managers through sfw by default

Developers have to remember the `sfw` prefix. You can drop shell aliases that add it for them, so `npm install` runs `sfw npm install`. Socket documents this pattern for its [Enterprise wrapper](https://docs.socket.dev/docs/socket-firewall-enterprise-wrapper-mode), and it works the same way with the free binary for the package managers it supports.

This is a convenience, not a control. Aliases apply only to interactive shells, `command npm` or a full path to the binary bypasses them, and IDE-integrated tooling that calls package managers directly never sees them.

### macOS and Linux

Save this as `configure-socket-firewall-aliases.sh` and add it to `controls.scripts`. It writes one aliases file and sources it from each system-wide shell rc file that exists on the host:

```bash
#!/bin/bash
# Adds shell aliases that run supported package managers through sfw.
set -euo pipefail

ALIASES="/etc/socket-firewall/aliases.sh"
mkdir -p /etc/socket-firewall
cat > "$ALIASES" <<'EOF'
# Managed by Fleet. Routes package managers through Socket Firewall Free.
if command -v sfw >/dev/null 2>&1; then
  alias npm="sfw npm"
  alias yarn="sfw yarn"
  alias pnpm="sfw pnpm"
  alias pip="sfw pip"
  alias uv="sfw uv"
  alias cargo="sfw cargo"
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

macOS ships `/etc/zshrc` and `/etc/bashrc`. Debian-family distributions use `/etc/zsh/zshrc` and `/etc/bash.bashrc`, and RHEL-family distributions use `/etc/zshrc` and `/etc/bashrc`. The loop only touches files that exist, and the `grep` keeps the script idempotent.

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

# Managed by Fleet. Routes package managers through Socket Firewall Free.
if (Get-Command sfw -ErrorAction SilentlyContinue) {
  function npm   { sfw npm @args }
  function yarn  { sfw yarn @args }
  function pnpm  { sfw pnpm @args }
  function pip   { sfw pip @args }
  function uv    { sfw uv @args }
  function cargo { sfw cargo @args }
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

> **Note:** These functions apply to PowerShell only. Command Prompt and Git Bash are unaffected. If PowerShell refuses to load the profile with an execution policy error, Socket's docs suggest `Set-ExecutionPolicy RemoteSigned -Scope CurrentUser`.

## Verify

1. In Fleet, open a host's details page and check the **Policies** section. The Socket Firewall policy for that platform should show **Yes** within an hour of the install. Click **Refetch** to check sooner.
2. On the host, open a new terminal and run `sfw --version`. It prints `Socket Firewall Free, version 1.15.2`.
3. Run a package manager through it with `--verbose`, for example `sfw --verbose npm install left-pad` in a scratch project. The output starts with `Protected by Socket Firewall`.
4. If you deployed aliases, run `type npm` in bash or zsh. It prints `npm is an alias for sfw npm`. In PowerShell, `Get-Command npm` shows `CommandType: Function`.

## Troubleshoot

**The policy still fails right after a successful install.** Policies run on the host's reporting interval, hourly by default. Refetch the host from its details page to update it now. If it still fails, run `ls -l /usr/local/bin/sfw` (or `Get-Item 'C:\Program Files\Socket Firewall\sfw.exe'`) on the host and compare the size with the policy. The policy's sizes must match the release the script installs.

**Checksum mismatch in the script output.** Either the version in the script and the checksum table disagree, or Socket re-published the asset. Download the asset again, recompute its SHA-256, and update the script and policy together.

**`sfw` is not recognized on Windows.** The `Path` change applies to new processes. Have the user close and reopen their terminal, or sign out and back in. Run `[Environment]::GetEnvironmentVariable("Path", "Machine")` to confirm the folder is present.

**macOS says the binary "cannot be opened because the developer cannot be verified."** This happens to copies downloaded with a browser, not to the copy Fleet installed. Remove the quarantine attribute with `xattr -d com.apple.quarantine /path/to/sfw`, or have the user run the Fleet-installed `/usr/local/bin/sfw`.

**Linux host reports "No such file or directory" when running sfw.** The host uses musl (Alpine, for example) and needs the `sfw-free-musl-linux-*` asset. Add a musl branch to the install script.

**Packages install but nothing is blocked.** Socket Firewall Free needs network access to Socket's API on every run, blocks only confirmed malware, and warns rather than blocks on AI-detected threats and unscanned versions. It also only inspects traffic to public registries. See Socket's [limitations](https://docs.socket.dev/docs/socket-firewall-free) for the full list.

## Further reading

- [Socket Firewall Free documentation](https://docs.socket.dev/docs/socket-firewall-free)
- [Socket Firewall overview](https://docs.socket.dev/docs/socket-firewall-overview), which compares Free and Enterprise
- [Deploy Socket Firewall Enterprise in wrapper mode with Fleet](https://fleetdm.com/guides/deploy-socket-firewall-enterprise-wrapper-mode-with-fleet)
- [Deploy Socket Firewall Enterprise in registry mode with Fleet](https://fleetdm.com/guides/deploy-socket-firewall-enterprise-registry-mode-with-fleet)
- [Scripts](https://fleetdm.com/guides/scripts) and [Automations](https://fleetdm.com/guides/automations) guides
- [GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files)

<meta name="articleTitle" value="Deploy Socket Firewall Free with Fleet">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-09-17">
<meta name="description" value="Install Socket Firewall Free on macOS, Windows, and Linux with Fleet scripts, verify it with policies, and reinstall it automatically.">
