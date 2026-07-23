# Build your own Linux self-service with script-only packages

_Fleet 4.89.0 gave script-only packages an uninstall option and install verification. That is enough to turn `apt-get install` and `dnf install` into a real, self-service software catalog for Linux, defined entirely in Git._

## Key takeaways

- **A script-only package is now a full lifecycle, not a one-shot.** As of Fleet 4.89.0, a `.sh` package can carry an uninstall script, a pre-install query, and a post-install script, so it behaves like a proper installer instead of a fire-and-forget command.
- **Any apt or dnf package can become a self-service tile.** The script's contents run in the host's root shell, so a two-line script wrapping `apt-get install` or `dnf install` is all it takes to put an app on the end user's self-service page.
- **The uninstall makes it a toggle.** Attach a matching removal script and the same tile that installed the app can now cleanly remove it, which is the piece script-only packages were missing before 4.89.0.
- **Labels and a verification step keep it honest across distros.** Target the right hosts with labels, and let a post-install check fail loudly, and roll back, when an install does not actually land.
- **The whole thing is generatable from a name.** Given a package name, a small generator emits the install script, the uninstall script, and the YAML block, so adding an app is one command and a pull request.

<a purpose="cta-button" href="/infrastructure-as-code">See it managed as code</a>

If you manage Linux with Fleet, you already have a fast way to run a script on a host. What you may not have noticed is that Fleet 4.89.0 turned the script-only package into something closer to a package manager front end. The release added a pre-install query, a post-install script, and an uninstall script to `.sh` and `.ps1` script-only packages, matching what custom packages already had.

That last item, the uninstall, is the unlock. With it, a script that runs `apt-get install` on the way in and `apt-get remove` on the way out becomes a self-service tile your end users can flip on and off, without you building or hosting a single `.deb`. Below is how to build it, ending with a generator that produces new catalog entries from nothing but a package name.

## The building block: script-only packages

A script-only package is a `.sh` file (for Linux) or `.ps1` file (for Windows) that you add to Fleet as software. There is no installer binary and no metadata extraction. The file's contents are the install script, and Fleet runs them on the host when someone installs the "package."

Before 4.89.0, that was the whole story: one script, run once, no way to undo it and no way to verify it worked. [Fleet 4.89.0](https://fleetdm.com/releases/fleet-4.89.0) changed that by letting a script-only package also carry:

- a pre-install query, a check that must return at least one row before the install runs,
- a post-install script, which runs after the install and, if it exits non-zero, fails the install and triggers the uninstall,
- an uninstall script, which runs when an admin or end user removes the software.

For a Linux self-service catalog, that maps cleanly to the two things a package manager does: put an app on, take it off, and check your work.

## The install script

The install script is the whole package. Fleet runs shell scripts in the host's root shell (`/bin/sh`) by default, so there is no `sudo` and no password prompt to worry about. Add a `#!/bin/bash` shebang if you want bash features.

One practical note first: there is no `apt uninstall`. The install verb is `apt-get install`, and removal is `apt-get remove` (or `apt-get purge` to also drop config files). Using `apt-get` rather than `apt` matters here, because `apt` itself warns that its interface is not stable for scripting.

A single script can cover both Debian- and RHEL-family hosts by checking which package manager exists:

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

The `-y` flags and `DEBIAN_FRONTEND=noninteractive` keep the run unattended, and `set -euo pipefail` makes the script exit non-zero the moment something fails, which is what Fleet reads to mark the install as failed. That non-zero exit is the difference between a self-service tile that reports the truth and one that always claims success.

## Add the uninstall

This is the part that was not possible before 4.89.0. Write the mirror-image script, `uninstall-htop.sh`:

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

Save both scripts under `lib/linux/scripts/` in your GitOps repo, then register the package from the fleet where you want it available. Software is defined per fleet, so this goes in `fleets/fleet-name.yml` or `fleets/unassigned.yml`. Point the entry at the install script and attach the uninstall script so the same tile can both install and remove:

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

With `self_service: true`, the app shows up as a tile on the end user's **Fleet Desktop > Self-service** page, filed under the Support category (added as a default self-service category in 4.89.0). When they click install, `fleetd` runs the install script as root. When they remove it, the uninstall script runs.

This is documented behavior, not a workaround. Both the [4.89.0 release notes](https://fleetdm.com/releases/fleet-4.89.0) and the [deploy software guide](https://fleetdm.com/guides/deploy-software-packages) confirm that script-only packages support `uninstall_script`, `post_install_script`, and `pre_install_query`. The one thing they still do not support is `install_script` (the file's contents already are the install script) and automatic install through a policy, so drive installs through self-service or the setup experience, not `install_software`.

## Guardrails: targeting and verification

Two more pieces keep a growing catalog from misbehaving.

### Target the right hosts with labels

The combined apt/dnf script degrades gracefully on a host with neither package manager, but you usually want tighter control over which hosts even see a tile. Labels handle that. Define a dynamic label for your Debian-family hosts and reference it on the package with `labels_include_any`:

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

Any label you reference on a package has to be defined in the `labels` section first. Use `labels_include_any`, `labels_include_all`, or `labels_exclude_any`, but only one per package.

### Verify the install actually landed

A pre-install query can gate whether the install runs at all (it proceeds only if the query returns a row), and a post-install script confirms the result afterward. The post-install check is the more valuable of the two here, because a non-zero exit fails the install and triggers your uninstall script, so a package that silently did not install does not sit there pretending it did:

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

```yaml
      post_install_script:
        path: ../lib/linux/scripts/verify-htop.sh
```

## Wire it into GitOps

Nothing above is a click in a console. Every self-service app is a set of files in your repository:

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

Adding, changing, or removing an app is a pull request. Reviewers see the exact commands that will run as root on every targeted host, the change ships through CI when it merges, and the catalog is auditable and reversible by design. If you want the Fleet UI to reflect that these are code-managed, [turn on GitOps mode](https://fleetdm.com/learn-more-about/ui-gitops-mode) so the relevant sections are read-only in the app and point back at your repo. Fleet manages its own software this way, and the [it-and-security configuration](https://github.com/fleetdm/fleet/tree/main/it-and-security/fleets) is public if you want a real-world layout to borrow from.

## Automate it: from app name to self-service tile

Once the pattern is settled, the per-app work is mechanical, which means it can be generated. The only real input is the package name. Here is a small generator that writes both scripts and prints the YAML block to paste into your fleet file:

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

Now onboarding an app is: find the package name (`apt-cache search`, `dnf search`, or knowing it already), run `./make-self-service-app.sh htop`, paste the printed block into the fleet file, and open a pull request. The generator is the honest expression of the whole idea: a self-service Linux catalog is just a naming convention plus a package manager, and both of those are things a script can produce.

## The point

Fleet did not ship a "Linux self-service store" in 4.89.0. It shipped three small additions to script-only packages, and those additions are enough to build one yourself, on top of the package managers your hosts already trust, with no installers to host and no per-app UI work. Because it all lives in Git, the catalog stays reviewable and reversible, and because the per-app work is mechanical, you can generate it from a name. That is the difference between a feature you consume and a primitive you build on.

## See it live

- Read the [deploy software guide](https://fleetdm.com/guides/deploy-software-packages) for the full detail on script-only packages, pre-install queries, and uninstall scripts.
- Get a demo: [fleetdm.com/contact](https://fleetdm.com/contact).
- Join a free GitOps workshop: [fleetdm.com/workshops](https://fleetdm.com/workshops).

_Managing devices as code, one pull request at a time. Start with the [GitOps reference](https://fleetdm.com/docs/configuration/yaml-files) or [talk to us](https://fleetdm.com/contact)._

<meta name="articleTitle" value="Build your own Linux self-service with script-only packages">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-07-20">
<meta name="articleImageUrl" value="../website/assets/images/articles/build-your-own-linux-self-service-with-script-only-packages-1200x627@2x.png">
<meta name="description" value="Use Fleet 4.89.0 script-only packages to turn apt and dnf commands into a Linux self-service catalog.">
