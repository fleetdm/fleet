# Block Homebrew packages on macOS

Homebrew lets users install command-line tools and apps without an admin. Use Fleet to block formulae and casks you don't trust, remove ones that are already installed, and keep the block in place.

## Prerequisites

- Hosts enrolled in Fleet.
- Scripts enabled. If you use Fleet's MDM features, scripts are enabled by default. If you deploy `fleetd` without MDM, pass the `--enable-scripts` flag during installation.
- Fleet Premium, to run the scripts automatically when a host fails a policy.

## Homebrew settings

Homebrew reads its settings from `/etc/homebrew/brew.env`. These settings control what users can install:

| Setting | What it does |
| ------- | ------------ |
| `HOMEBREW_FORBIDDEN_FORMULAE` | Blocks the listed formulae. Also blocks any formula that depends on one of them. |
| `HOMEBREW_FORBIDDEN_CASKS` | Blocks the listed casks. |
| `HOMEBREW_FORBIDDEN_OWNER` | The name shown in the error message, like `your IT team`. Defaults to `you`. |
| `HOMEBREW_FORBIDDEN_OWNER_CONTACT` | Contact details added to the error message, like a link to request an exception. |
| `HOMEBREW_SYSTEM_ENV_TAKES_PRIORITY` | Makes `/etc/homebrew/brew.env` override settings in the user's own `~/.homebrew/brew.env`. Without it, users can unblock packages themselves. |

## Set the package lists

Store the blocked packages in [custom variables](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles), so you don't need to edit the scripts when the lists change.

1. In Fleet, go to **Controls > Variables** and select **Add variable**.
2. Add `HOMEBREW_FORBIDDEN_FORMULAE` with the formulae to block, like `telnet sshpass`.
3. Add `HOMEBREW_FORBIDDEN_CASKS` with the casks to block, like `discord telegram`.

Separate names with spaces and don't use quotes, because Homebrew reads quotes as part of the name. To look up a package's exact name, search [formulae.brew.sh](https://formulae.brew.sh) or run `brew search <keyword>`. To see what's already installed on your hosts, run the report in [Verify](#verify).

Scripts reference the variables as `$FLEET_SECRET_HOMEBREW_FORBIDDEN_FORMULAE` and `$FLEET_SECRET_HOMEBREW_FORBIDDEN_CASKS`. Fleet fills in the values when it sends a script to a host.

> **Note:** Custom variables apply to all fleets, and Fleet hides their values in the UI. To block different packages on different fleets, create a variable for each list, like `HOMEBREW_FORBIDDEN_FORMULAE_SERVERS`, and a copy of each script that uses it.

## Block packages

1. Save the following as `homebrew-forbidden-packages.sh`.

```bash
#!/bin/bash

mkdir -p /etc/homebrew

cat > /etc/homebrew/brew.env <<'EOF'
HOMEBREW_FORBIDDEN_FORMULAE=$FLEET_SECRET_HOMEBREW_FORBIDDEN_FORMULAE
HOMEBREW_FORBIDDEN_CASKS=$FLEET_SECRET_HOMEBREW_FORBIDDEN_CASKS
HOMEBREW_FORBIDDEN_OWNER=your IT team
HOMEBREW_FORBIDDEN_OWNER_CONTACT=To request an exception, contact it@example.com.
HOMEBREW_SYSTEM_ENV_TAKES_PRIORITY=1
EOF

echo "$FLEET_SECRET_HOMEBREW_FORBIDDEN_FORMULAE $FLEET_SECRET_HOMEBREW_FORBIDDEN_CASKS" | tr -s ' ' '\n' > /etc/homebrew/blocked-packages.txt

chown root:wheel /etc/homebrew /etc/homebrew/brew.env /etc/homebrew/blocked-packages.txt
chmod 755 /etc/homebrew
chmod 644 /etc/homebrew/brew.env /etc/homebrew/blocked-packages.txt
```

> **Warning:** This script replaces `/etc/homebrew/brew.env`. If you manage other Homebrew settings in this file, add them to the script before you run it.

2. In Fleet, go to **Controls > Scripts** and upload the script.
3. To run the script on a host, open the host's page and select **Actions > Run script**. To run it automatically on hosts that fail a policy, see [Run automatically with policies](#run-automatically-with-policies).

The script also writes every blocked name to `/etc/homebrew/blocked-packages.txt`, one per line, for the [policy](#run-automatically-with-policies) that checks for blocked packages. Both files are owned by root, so only admins can change them.

Blocking a package doesn't remove copies that are already installed. To remove them, see [Remove installed packages](#remove-installed-packages).

## Remove installed packages

Homebrew doesn't run as root, so the script runs `brew` as the user who owns the Homebrew installation.

1. Save the following as `homebrew-package-removal.sh`. It uses the same variables as the blocking script.

```bash
#!/bin/bash

read -ra FORMULAE <<< "$FLEET_SECRET_HOMEBREW_FORBIDDEN_FORMULAE"
read -ra CASKS <<< "$FLEET_SECRET_HOMEBREW_FORBIDDEN_CASKS"

if [ -x /opt/homebrew/bin/brew ]; then
  BREW=/opt/homebrew/bin/brew
elif [ -x /usr/local/bin/brew ]; then
  BREW=/usr/local/bin/brew
else
  echo "Homebrew not found"
  exit 0
fi

BREW_USER=$(stat -f%Su "$BREW")
run_brew() { sudo -u "$BREW_USER" -H "$BREW" "$@"; }
cd /tmp
STATUS=0

for pkg in "${FORMULAE[@]}"; do
  if ! run_brew list --formula "$pkg" >/dev/null 2>&1; then
    echo "Not installed: $pkg"
    continue
  fi

  DEPENDENTS=$(run_brew uses --installed "$pkg" | tr '\n' ' ')
  if [ -n "$DEPENDENTS" ]; then
    echo "Skipped: $pkg is required by: $DEPENDENTS"
    STATUS=1
    continue
  fi

  if run_brew uninstall --formula "$pkg"; then
    echo "Removed: $pkg"
  else
    echo "Failed: $pkg"
    STATUS=1
  fi
done

for cask in "${CASKS[@]}"; do
  if ! run_brew list --cask "$cask" >/dev/null 2>&1; then
    echo "Not installed: $cask"
    continue
  fi

  if run_brew uninstall --cask "$cask"; then
    echo "Removed: $cask"
  else
    echo "Failed: $cask"
    STATUS=1
  fi
done

exit $STATUS
```

2. In Fleet, go to **Controls > Scripts** and upload the script.
3. To run the script on a host, open the host's page and select **Actions > Run script**. To run it automatically on hosts that fail a policy, see [Run automatically with policies](#run-automatically-with-policies).

If another installed formula depends on a blocked formula, the script skips it and lists the formulae that need it. The script run then shows as failed in Fleet.

When the script removes a formula, Homebrew also removes dependencies that were installed only for it. To turn this off, add `HOMEBREW_NO_AUTOREMOVE=1` to the `brew.env` lines in `homebrew-forbidden-packages.sh`.

> **Note:** Don't force the removal with `brew uninstall --ignore-dependencies`. It breaks the formulae that depend on the package.

## Run automatically with policies

Add two policies that run the scripts when a host fails them. The first puts the blocking files back if either is missing. The second removes blocked packages, including ones installed before you added the block. It reads the names from `/etc/homebrew/blocked-packages.txt`, so you don't need to edit it when the lists change.

1. In Fleet, go to **Policies**, select your fleet, and select **Add policy**.
2. Enter the following query:

```sql
SELECT 1 WHERE (
  SELECT COUNT(*) FROM file
  WHERE path IN ('/etc/homebrew/brew.env', '/etc/homebrew/blocked-packages.txt')
) = 2;
```

3. Name the policy `macOS - Homebrew blocklist is present` and set the platform to macOS.
4. Set the automation to run `homebrew-forbidden-packages.sh`, then save the policy.
5. Repeat steps 1-4 for the second policy. Name it `macOS - Blocked Homebrew packages are not installed`, set its automation to run `homebrew-package-removal.sh`, and use this query:

```sql
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM homebrew_packages
  WHERE name IN (SELECT line FROM file_lines WHERE path = '/etc/homebrew/blocked-packages.txt')
);
```

> **Note:** The first policy only checks that the files exist. After you change a variable, run `homebrew-forbidden-packages.sh` again on each host.

To learn more, see [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script).

## Manage with GitOps

To manage these settings with [GitOps](https://fleetdm.com/docs/configuration/yaml-files):

1. Add the package lists to your repository's secrets as `FLEET_SECRET_HOMEBREW_FORBIDDEN_FORMULAE` and `FLEET_SECRET_HOMEBREW_FORBIDDEN_CASKS`.
2. Pass the secrets to your GitOps workflow:

```yaml
env:
  FLEET_SECRET_HOMEBREW_FORBIDDEN_FORMULAE: ${{ secrets.FLEET_SECRET_HOMEBREW_FORBIDDEN_FORMULAE }}
  FLEET_SECRET_HOMEBREW_FORBIDDEN_CASKS: ${{ secrets.FLEET_SECRET_HOMEBREW_FORBIDDEN_CASKS }}
```

To run `fleetctl gitops` from your own computer instead, set the same variables in your terminal first:

```bash
export FLEET_SECRET_HOMEBREW_FORBIDDEN_FORMULAE="telnet sshpass"
export FLEET_SECRET_HOMEBREW_FORBIDDEN_CASKS="discord telegram"
```

3. Save the scripts in your GitOps repository and reference them in your fleet's YAML file:

```yaml
controls:
  scripts:
    - path: ../lib/macos/scripts/homebrew-forbidden-packages.sh
    - path: ../lib/macos/scripts/homebrew-package-removal.sh

policies:
  - name: macOS - Homebrew blocklist is present
    query: |
      SELECT 1 WHERE (
        SELECT COUNT(*) FROM file
        WHERE path IN ('/etc/homebrew/brew.env', '/etc/homebrew/blocked-packages.txt')
      ) = 2;
    platform: darwin
    run_script:
      path: ../lib/macos/scripts/homebrew-forbidden-packages.sh
  - name: macOS - Blocked Homebrew packages are not installed
    query: |
      SELECT 1 WHERE NOT EXISTS (
        SELECT 1 FROM homebrew_packages
        WHERE name IN (SELECT line FROM file_lines WHERE path = '/etc/homebrew/blocked-packages.txt')
      );
    platform: darwin
    run_script:
      path: ../lib/macos/scripts/homebrew-package-removal.sh
```

> **Note:** Each `path` is relative to the YAML file it's in. Change the paths to match where the scripts are saved in your repository, like `../lib/workstations/scripts/`.

To learn more, see [Custom variables](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles#gitops).

## What users see

| Situation | What happens |
| --------- | ------------ |
| Blocked formula | Homebrew stops the install: `Error: The installation of telnet was forbidden by your IT team`, followed by your contact details if you set them. |
| Blocked dependency | If a formula depends on a blocked formula, Homebrew stops the install and names the blocked dependency. |
| Blocked cask | Homebrew stops the install with a similar error. |
| Removed package | The package is uninstalled without a prompt. If users try to install it again, Homebrew blocks it. |

## Verify

1. On a host, run `cat /etc/homebrew/brew.env /etc/homebrew/blocked-packages.txt` and confirm both files list the packages you blocked.
2. Run `brew install <blocked formula>` and confirm Homebrew refuses.

To check across all your hosts, run this report to list installed Homebrew packages:

```sql
SELECT name, version, type, path FROM homebrew_packages;
```

To check for one package, add `WHERE name = '<name>'`. If it was removed, the report returns no results.

> **Note:** To see Homebrew formulae on a host's **Software** tab, select **Package (Homebrew)** in the **Types** filter. Casks that install an app, like `Discord.app`, are listed as apps instead, so they aren't counted twice.

## Troubleshoot

**Users can still install blocked packages**

Check these in order:

1. Confirm the names in `/etc/homebrew/brew.env` aren't in quotes.
2. Confirm `/etc/homebrew/brew.env` includes `HOMEBREW_SYSTEM_ENV_TAKES_PRIORITY=1`. Without it, users can unblock packages in their own `~/.homebrew/brew.env`.
3. Check whether another script or policy automation also writes to the file, which would overwrite your settings.
4. Admin users can edit or delete the blocking files. The [blocklist policy](#run-automatically-with-policies) puts deleted files back the next time the host checks in. To restore edited files, run `homebrew-forbidden-packages.sh` again.

**A host doesn't block a package you added**

Changing a variable doesn't update hosts that already have `/etc/homebrew/brew.env`. Run `homebrew-forbidden-packages.sh` again on those hosts.

**The removal script skips a formula**

Another installed formula depends on it. Remove the dependent formula first, or add it to your blocked formulae.

**A blocked package is still installed**

Blocking doesn't remove packages that were already installed. Run `homebrew-package-removal.sh`, or set it to [run automatically](#run-automatically-with-policies).

**The blocked tool still runs after you remove it**

The tool may also be installed outside Homebrew. For example, macOS includes its own copy of `jq` in `/usr/bin`. Run `which -a <tool>` to see every copy on the host.

**`fleetctl gitops` fails with `script file does not exist`**

A script `path` in your YAML file doesn't match where the script is saved. Paths are relative to the YAML file, so check the folder names and the file name, then run `fleetctl gitops` again.

## Further reading

- [Homebrew manual: Environment variables](https://docs.brew.sh/Manpage#environment)
- [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script)
- [Block Chrome extensions on macOS, Windows, and Linux](https://fleetdm.com/guides/block-chrome-extensions)

<meta name="articleTitle" value="Block Homebrew packages on macOS">
<meta name="authorFullName" value="Matthew Thomas">
<meta name="authorGitHubUsername" value="adriant172">
<meta name="publishedOn" value="2026-11-04">
<meta name="category" value="guides">
<meta name="description" value="Block, remove, and keep Homebrew formulae and casks off macOS hosts with Fleet using scripts and policies.">
