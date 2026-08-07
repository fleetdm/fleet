# Build a Linux self-service catalog with script-only packages

Fleet 4.89.0 gave script-only packages an uninstall script, a pre-install query, and a post-install script. That is enough to turn `apt-get install` and `dnf install` into a self-service software catalog for Linux, defined entirely in Git, with no `.deb` or `.rpm` files to build or host. This guide walks through building one app end to end, then generating the rest from a package name. It covers apt (Debian family) and dnf (RHEL family) hosts driven through self-service, not policy-based automatic installs.

## Prerequisites

- Fleet 4.89.0 or later. The uninstall script, pre-install query, and post-install script on script-only packages were added in this release.
- A GitOps repository connected to your Fleet instance. Software is defined per fleet in `fleets/<name>.yml`.
- Linux hosts enrolled in Fleet with `fleetd`, running a distribution that uses `apt-get` or `dnf`.
- Fleet Desktop available to end users, so the self-service page is reachable.

> **Note:** Script-only packages do not support an `install_script` key (the file's contents already are the install script) or automatic install through a policy. Drive installs through self-service or the setup experience, not `install_software`.

## Step 1: Write the install script

Fleet runs shell scripts in the host's root shell (`/bin/sh`) by default, so there is no `sudo` and no password prompt. Add a `#!/bin/bash` shebang if you want bash features.

Use `apt-get`, not `apt`: `apt` warns that its command-line interface is not stable for scripting. The install verb is `apt-get install`; removal is `apt-get remove` (or `apt-get purge` to also drop config files).

A single script covers both Debian- and RHEL-family hosts by checking which package manager exists. Save this as `install-htop.sh`:

```bash
#!/bin/bash
set -euo pipefail

PKG="htop"

echo "Installing ${PKG}..."
if command -v apt-get >/dev/null 2>&1; then
  export DEBIAN_FRONTEND=noninteractive
  apt-get update
  apt-get install -y "${PKG}"
elif command -v dnf >/dev/null 2>&1; then
  dnf install -y "${PKG}"
else
  echo "No supported package manager found (apt-get or dnf)." >&2
  exit 1
fi
echo "${PKG} installed."
```

The `-y` flags and `DEBIAN_FRONTEND=noninteractive` keep the run unattended. `set -euo pipefail` makes the script exit non-zero the moment something fails, which is what Fleet reads to mark the install as failed.

## Step 2: Write the uninstall script

Write the mirror-image script so the same tile can remove the app. Save this as `uninstall-htop.sh`:

```bash
#!/bin/bash
set -euo pipefail

PKG="htop"

echo "Removing ${PKG}..."
if command -v apt-get >/dev/null 2>&1; then
  export DEBIAN_FRONTEND=noninteractive
  apt-get remove -y "${PKG}"
elif command -v dnf >/dev/null 2>&1; then
  dnf remove -y "${PKG}"
else
  echo "No supported package manager found (apt-get or dnf)." >&2
  exit 1
fi
echo "${PKG} removed."
```

## Step 3: Register the package in your fleet file

Save both scripts under `lib/linux/scripts/` in your GitOps repo. Then register the package in the fleet where you want it available, in `fleets/<name>.yml` or `fleets/unassigned.yml`. Point the entry at the install script and attach the uninstall script:

```yaml
software:
  packages:
    - path: ../lib/linux/scripts/install-htop.sh
      display_name: htop
      self_service: true
      categories:
        - "🛟 Support"
      uninstall_script:
        path: ../lib/linux/scripts/uninstall-htop.sh
```

With `self_service: true`, the app appears as a tile on the end user's **Fleet Desktop > Self-service** page, filed under the Support category (added as a default self-service category in 4.89.0). When the user clicks install, `fleetd` runs the install script as root. When they remove it, the uninstall script runs.

## Step 4: Target the right hosts with labels

The combined apt/dnf script degrades gracefully on a host with neither package manager, but you usually want tighter control over which hosts see a tile. Define a dynamic label, then reference it on the package.

```yaml
labels:
  - name: Linux (apt)
    query: "SELECT 1 FROM os_version WHERE platform_like = 'debian';"
    label_membership_type: dynamic
```

```yaml
software:
  packages:
    - path: ../lib/linux/scripts/install-htop.sh
      display_name: htop
      self_service: true
      labels_include_any:
        - Linux (apt)
      uninstall_script:
        path: ../lib/linux/scripts/uninstall-htop.sh
```

> **Note:** Any label you reference on a package must be defined in the `labels` section first. Use `labels_include_any`, `labels_include_all`, or `labels_exclude_any`, but only one per package.

## Step 5: Verify the install actually landed

A pre-install query gates whether the install runs at all (it proceeds only if the query returns a row). A post-install script confirms the result afterward. The post-install check matters more here: a non-zero exit fails the install and triggers your uninstall script, so a package that silently did not install does not sit there reporting success.

Save this as `verify-htop.sh`:

```bash
#!/bin/bash
# post-install: confirm the package is actually present
set -euo pipefail

PKG="htop"

if command -v dpkg >/dev/null 2>&1; then
  dpkg -s "${PKG}" >/dev/null 2>&1
elif command -v rpm >/dev/null 2>&1; then
  rpm -q "${PKG}" >/dev/null 2>&1
fi
```

Reference it on the package:

```yaml
      post_install_script:
        path: ../lib/linux/scripts/verify-htop.sh
```

## Step 6: Wire it into GitOps

Every self-service app is now a set of files in your repository:

```
lib/
  linux/
    scripts/
      install-htop.sh
      uninstall-htop.sh
      verify-htop.sh
fleets/
  workstations.yml   # references the scripts under software.packages
```

Adding, changing, or removing an app is a pull request. Reviewers see the exact commands that will run as root on every targeted host, the change ships through CI when it merges, and the catalog is auditable and reversible.

To make the Fleet UI reflect that these are code-managed, [turn on GitOps mode](https://fleetdm.com/learn-more-about/ui-gitops-mode) so the relevant sections are read-only in the app and point back at your repo. Fleet manages its own software this way; the [it-and-security configuration](https://github.com/fleetdm/fleet/tree/main/it-and-security/fleets) is public if you want a real-world layout to borrow from.

## Step 7: Generate new apps from a package name

Once the pattern is settled, the per-app work is mechanical. The only real input is the package name. This generator writes both scripts and prints the YAML block to paste into your fleet file. Save it as `make-self-service-app.sh`:

```bash
#!/usr/bin/env bash
# make-self-service-app.sh <package-name> [display name]
set -euo pipefail

APP="${1:?Usage: make-self-service-app.sh <package-name> [display name]}"
DISPLAY="${2:-$APP}"
DIR="lib/linux/scripts"
mkdir -p "$DIR"

cat > "$DIR/install-$APP.sh" <<EOF
#!/bin/bash
set -euo pipefail
echo "Installing $DISPLAY..."
if command -v apt-get >/dev/null 2>&1; then
  export DEBIAN_FRONTEND=noninteractive
  apt-get update
  apt-get install -y "$APP"
elif command -v dnf >/dev/null 2>&1; then
  dnf install -y "$APP"
else
  echo "No supported package manager found." >&2
  exit 1
fi
echo "$DISPLAY installed."
EOF

cat > "$DIR/uninstall-$APP.sh" <<EOF
#!/bin/bash
set -euo pipefail
echo "Removing $DISPLAY..."
if command -v apt-get >/dev/null 2>&1; then
  export DEBIAN_FRONTEND=noninteractive
  apt-get remove -y "$APP"
elif command -v dnf >/dev/null 2>&1; then
  dnf remove -y "$APP"
else
  echo "No supported package manager found." >&2
  exit 1
fi
echo "$DISPLAY removed."
EOF

chmod +x "$DIR/install-$APP.sh" "$DIR/uninstall-$APP.sh"

cat <<EOF

# Add to your fleet's software.packages:
    - path: ../$DIR/install-$APP.sh
      display_name: $DISPLAY
      self_service: true
      categories:
        - "🛟 Support"
      uninstall_script:
        path: ../$DIR/uninstall-$APP.sh
EOF
```

To onboard an app: find the package name (`apt-cache search`, `dnf search`, or knowing it already), run `./make-self-service-app.sh htop`, paste the printed block into the fleet file, and open a pull request.

## Troubleshoot

**The tile does not appear on a host's self-service page.** Check that the host matches the package's label. If you set `labels_include_any: Linux (apt)`, a dnf-only host will not see the tile. Confirm the label query returns the host in the Fleet UI under the label's host list.

**The install reports success but the app is not present.** Add the post-install verification script from Step 5. Without it, a package manager that exits zero on a no-op leaves the tile claiming success. The `dpkg -s` / `rpm -q` check exits non-zero when the package is missing, which fails the install and triggers the uninstall.

**The install hangs.** A prompt is waiting for input. Confirm `-y` is on the install command and `DEBIAN_FRONTEND=noninteractive` is exported for apt-get.

## Further reading

- [Deploy software guide](https://fleetdm.com/guides/deploy-software-packages) for full detail on script-only packages, pre-install queries, and uninstall scripts.
- [Fleet 4.89.0 release notes](https://fleetdm.com/releases/fleet-4.89.0).
- [GitOps YAML reference](https://fleetdm.com/docs/configuration/yaml-files).

<meta name="articleTitle" value="Build a Linux self-service catalog with script-only packages">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-07-20">
<meta name="description" value="Use Fleet 4.89.0 script-only packages to turn apt and dnf commands into a Linux self-service catalog.">
