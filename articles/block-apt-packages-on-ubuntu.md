# Block apt packages on Ubuntu

Users with `sudo` rights can install any package from Ubuntu's repositories with apt. Use Fleet to block packages you don't trust, remove ones that are already installed, and keep the block in place.

## Prerequisites

- Hosts enrolled in Fleet.
- Scripts enabled. Linux hosts don't use Fleet's MDM features, so pass the `--enable-scripts` flag when you generate the `fleetd` package.
- Fleet Premium, to run the scripts automatically when a host fails a policy.

## How apt blocks packages

apt uses pin priorities, set in files in `/etc/apt/preferences.d/`, to decide which version of a package to install. Pinning can also prefer or force a version, but a priority of `-1` tells apt never to install the package. Each package needs its own block of three lines:

```text
Package: sshpass
Pin: release *
Pin-Priority: -1
```

A package with a priority of `-1` can't be installed or downloaded with apt. Any package that depends on it can't be installed either.

> **Note:** This is a guardrail, not a hard control. It stops installs through apt. Users with `sudo` rights can still edit or remove the pin file, or install software other ways, like Snap. Users without `sudo` rights can't install packages with apt at all.

## Set the package list

Store the blocked packages in a [custom variable](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles), so you don't need to edit the scripts when the list changes.

1. In Fleet, go to **Controls > Variables** and select **Add variable**.
2. Add `APT_BLOCKED_PACKAGES` with the packages to block, like `sshpass telnet inetutils-telnet`.

Separate names with spaces. To look up a package's exact name, run `apt-cache search <keyword>`. To see what's already installed on your hosts, run the report in [Verify](#verify).

> **Note:** A `-1` pin only blocks the exact package name. Transitional packages only install another package, so block both. For example, on Ubuntu, `telnet` installs `inetutils-telnet`. To check a package, run `apt-cache show <name>` and look at its `Depends` line.

Scripts reference the variable as `$FLEET_SECRET_APT_BLOCKED_PACKAGES`. Fleet fills in the value when it sends a script to a host.

> **Note:** Custom variables apply to all fleets, and Fleet hides their values in the UI. To block different packages on different fleets, create a variable for each list, like `APT_BLOCKED_PACKAGES_SERVERS`, and a copy of each script that uses it.

## Block packages

1. Save the following as `apt-blocked-packages.sh`.

```bash
#!/bin/bash

read -ra PACKAGES <<< "$FLEET_SECRET_APT_BLOCKED_PACKAGES"

PIN_FILE=/etc/apt/preferences.d/fleet-blocked-packages
LIST_FILE=/etc/apt/blocked-packages.txt

: > "$PIN_FILE"
: > "$LIST_FILE"

for pkg in "${PACKAGES[@]}"; do
  printf 'Package: %s\nPin: release *\nPin-Priority: -1\n\n' "$pkg" >> "$PIN_FILE"
  echo "$pkg" >> "$LIST_FILE"
done

chown root:root "$PIN_FILE" "$LIST_FILE"
chmod 644 "$PIN_FILE" "$LIST_FILE"
```

> **Note:** Fleet replaces `$FLEET_SECRET_APT_BLOCKED_PACKAGES` before the script reaches the host. To test the script outside Fleet, replace the variable with package names first.

2. In Fleet, go to **Controls > Scripts** and upload the script.
3. To run the script on a host, open the host's page and select **Actions > Run script**. To run it automatically on hosts that fail a policy, see [Run automatically with policies](#run-automatically-with-policies).

The script writes a pin for each package to `/etc/apt/preferences.d/fleet-blocked-packages`. It also writes the package names to `/etc/apt/blocked-packages.txt`, one per line, for the [policy](#run-automatically-with-policies) that checks for blocked packages. Both files are owned by root, so only users with `sudo` rights can change them. Each time the script runs, it rewrites both files, so packages you remove from the variable are unblocked.

Blocking a package doesn't remove copies that are already installed. To remove them, see [Remove installed packages](#remove-installed-packages).

## Remove installed packages

When apt removes a package, it also removes every package that depends on it. To avoid removing other software, the script first simulates each removal and skips packages that would take anything else with them.

1. Save the following as `apt-package-removal.sh`. It uses the same variable as the blocking script.

```bash
#!/bin/bash

read -ra PACKAGES <<< "$FLEET_SECRET_APT_BLOCKED_PACKAGES"

export DEBIAN_FRONTEND=noninteractive
STATUS=0

for pkg in "${PACKAGES[@]}"; do
  if ! dpkg-query -W -f='${Status}' "$pkg" 2>/dev/null | grep -q "install ok installed"; then
    echo "Not installed: $pkg"
    continue
  fi

  OTHERS=$(apt-get -s remove "$pkg" | awk '/^Remv /{print $2}' | grep -vx "$pkg" | tr '\n' ' ')
  if [ -n "$OTHERS" ]; then
    echo "Skipped: removing $pkg would also remove: $OTHERS"
    STATUS=1
    continue
  fi

  if apt-get -y -o DPkg::Lock::Timeout=120 remove "$pkg"; then
    echo "Removed: $pkg"
  else
    echo "Failed: $pkg"
    STATUS=1
  fi
done

exit $STATUS
```

2. In Fleet, go to **Controls > Scripts** and upload the script.
3. To run the script on a host, open the host's page and select **Actions > Run script**. To run it automatically on hosts that fail a policy, see [Run automatically with policies](#run-automatically-with-policies).

If removing a blocked package would also remove other packages, the script skips it and lists them, like `Skipped: removing libonig5 would also remove: jq libjq1`. The script run then shows as failed in Fleet.

> **Note:** When the script skips a package, it exits with an error, so the policy automation retries it, up to 3 attempts in total. If the package is still needed after that, the host keeps failing the policy until you remove or block the packages that depend on it.

apt leaves behind dependencies that were installed only for the removed package. The script doesn't run `apt-get autoremove`, because that removes every package apt no longer needs, not only the ones left by the script.

## Run automatically with policies

Add two policies that run the scripts when a host fails them. The first puts the blocking files back if either is missing. The second removes blocked packages, including ones installed before you added the block. It reads the names from `/etc/apt/blocked-packages.txt`, so you don't need to edit it when the list changes.

1. In Fleet, go to **Policies**, select your fleet, and select **Add policy**.
2. Enter the following query:

```sql
SELECT 1 WHERE (
  SELECT COUNT(*) FROM file
  WHERE path IN ('/etc/apt/preferences.d/fleet-blocked-packages', '/etc/apt/blocked-packages.txt')
) = 2;
```

3. Name the policy `Linux - apt blocklist is present` and set the platform to Linux.
4. Set the automation to run `apt-blocked-packages.sh`, then save the policy.
5. Repeat steps 1-4 for the second policy. Name it `Linux - Blocked apt packages are not installed`, set its automation to run `apt-package-removal.sh`, and use this query:

```sql
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM deb_packages
  WHERE status = 'install ok installed'
    AND name IN (SELECT line FROM file_lines WHERE path = '/etc/apt/blocked-packages.txt')
);
```

> **Note:** The first policy only checks that the files exist. After you change the variable, run `apt-blocked-packages.sh` again. To run it on many hosts at once, see [Batch execute scripts](https://fleetdm.com/guides/scripts#batch-execute-scripts).

To learn more, see [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script).

### Fleets with other Linux distributions

The Linux platform includes every Linux host in the fleet. On hosts that don't use apt, like Fedora, the first policy fails and runs the apt script. To limit both policies to hosts with apt, create a dynamic [label](https://fleetdm.com/guides/managing-labels-in-fleet) named `Hosts with apt` with this query, and target both policies to it:

```sql
SELECT 1 FROM file WHERE path = '/usr/bin/apt-get';
```

## Manage with GitOps

To manage these settings with [GitOps](https://fleetdm.com/docs/configuration/yaml-files):

1. Add the package list to your repository's secrets as `FLEET_SECRET_APT_BLOCKED_PACKAGES`.
2. Pass the secret to your GitOps workflow:

```yaml
env:
  FLEET_SECRET_APT_BLOCKED_PACKAGES: ${{ secrets.FLEET_SECRET_APT_BLOCKED_PACKAGES }}
```

> **Note:** If you run `fleetctl gitops` from your own computer, like for a local test, skip steps 1 and 2. Instead, set the variable in the same terminal window before you run `fleetctl gitops`:
>
> ```bash
> export FLEET_SECRET_APT_BLOCKED_PACKAGES="sshpass telnet inetutils-telnet"
> ```

3. Save the scripts in your GitOps repository and reference them in your fleet's YAML file:

```yaml
labels:
  - name: Hosts with apt
    description: Linux hosts that use apt to install packages.
    label_membership_type: dynamic
    platform: linux
    query: SELECT 1 FROM file WHERE path = '/usr/bin/apt-get';

controls:
  scripts:
    - path: ../lib/linux/scripts/apt-blocked-packages.sh
    - path: ../lib/linux/scripts/apt-package-removal.sh

policies:
  - name: Linux - apt blocklist is present
    query: |
      SELECT 1 WHERE (
        SELECT COUNT(*) FROM file
        WHERE path IN ('/etc/apt/preferences.d/fleet-blocked-packages', '/etc/apt/blocked-packages.txt')
      ) = 2;
    platform: linux
    labels_include_any:
      - Hosts with apt
    run_script:
      path: ../lib/linux/scripts/apt-blocked-packages.sh
  - name: Linux - Blocked apt packages are not installed
    query: |
      SELECT 1 WHERE NOT EXISTS (
        SELECT 1 FROM deb_packages
        WHERE status = 'install ok installed'
          AND name IN (SELECT line FROM file_lines WHERE path = '/etc/apt/blocked-packages.txt')
      );
    platform: linux
    labels_include_any:
      - Hosts with apt
    run_script:
      path: ../lib/linux/scripts/apt-package-removal.sh
```

> **Note:** Each `path` is relative to the YAML file it's in. Change the paths to match where the scripts are saved in your repository, like `../lib/servers/scripts/`.

> **Warning:** The `labels` key replaces all labels defined in that file. If your file already has a `labels` section, add `Hosts with apt` to it. If your organization manages labels outside GitOps, leave out the `labels` section and create the label in the Fleet UI instead.

To learn more, see [Custom variables](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles#gitops).

## What users see

| Situation | What happens |
| --------- | ------------ |
| Blocked package | apt stops the install: `E: Package 'sshpass' has no installation candidate`. |
| Blocked dependency | If a package depends on a blocked package, apt stops the install with `E: Unable to satisfy dependencies` and names the blocked dependency. |
| Download | `apt-get download` stops with `E: Can't select candidate version from package sshpass as it has no candidate`. |
| Removed package | The package is removed without a prompt. If users try to install it again, apt blocks it. |

## Verify

1. On a host, run `cat /etc/apt/preferences.d/fleet-blocked-packages /etc/apt/blocked-packages.txt` and confirm the files list the packages you blocked.
2. Run `sudo apt-get install <blocked package>` and confirm apt refuses.

To check across all your hosts, run this report to list installed packages:

```sql
SELECT name, version, status FROM deb_packages WHERE status = 'install ok installed';
```

To check for one package, add `AND name = '<name>'`. If it was removed, the report returns no results.

> **Note:** To see apt packages on a host's **Software** tab, select **Package (deb)** in the **Types** filter.

## Troubleshoot

**Users can still install blocked packages**

Check these in order:

1. Confirm the package name matches exactly. A `-1` pin doesn't block transitional or alternative packages that provide the same tool, like `inetutils-telnet` for `telnet`.
2. Confirm `/etc/apt/preferences.d/fleet-blocked-packages` has a separate block for each package. A single `Package:` line with several names doesn't block them.
3. Check whether another script or policy automation also writes to the pin file, which would overwrite your settings.
4. Users with `sudo` rights can edit or delete the blocking files. The [blocklist policy](#run-automatically-with-policies) puts deleted files back the next time the host checks in. To restore edited files, run `apt-blocked-packages.sh` again.

**A host doesn't block a package you added**

Changing the variable doesn't update hosts that already have the blocking files. Run `apt-blocked-packages.sh` again on those hosts. To run it on many hosts at once, see [Batch execute scripts](https://fleetdm.com/guides/scripts#batch-execute-scripts).

**The removal script skips a package**

Removing it would also remove packages that depend on it. Fleet retries the script up to 3 times, and the host keeps failing the policy until the dependent packages are gone. Remove the dependent packages first, or add them to your blocked packages.

**A blocked package is still installed**

Blocking doesn't remove packages that were already installed. Run `apt-package-removal.sh`, or set it to [run automatically](#run-automatically-with-policies).

**The script output shows `perl: warning: Setting locale failed`**

Some package tools print locale warnings when they run from a Fleet script. They don't affect the result. Check the `Removed`, `Skipped`, or `Failed` lines at the end of the output.

**The apt policies run on hosts that don't use apt**

Limit both policies to the `Hosts with apt` label. See [Fleets with other Linux distributions](#fleets-with-other-linux-distributions).

**`fleetctl gitops` fails with `script file does not exist`**

A script `path` in your YAML file doesn't match where the script is saved. Paths are relative to the YAML file, so check the folder names and the file name, then run `fleetctl gitops` again.

## Further reading

- [apt_preferences manual](https://manpages.ubuntu.com/manpages/noble/man5/apt_preferences.5.html)
- [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script)
- [Labels in Fleet](https://fleetdm.com/guides/managing-labels-in-fleet)
- [Block Homebrew packages on macOS](https://fleetdm.com/guides/block-homebrew-packages-on-macos)

<meta name="articleTitle" value="Block apt packages on Ubuntu">
<meta name="authorFullName" value="Matthew Thomas">
<meta name="authorGitHubUsername" value="adriant172">
<meta name="publishedOn" value="2026-11-13">
<meta name="category" value="guides">
<meta name="description" value="Block, remove, and keep apt packages off Ubuntu hosts with Fleet using scripts and policies.">
