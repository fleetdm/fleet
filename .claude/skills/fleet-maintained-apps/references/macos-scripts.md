# macOS install and uninstall scripts

Fleet runs these scripts as root through fleetd, with the installer at `$INSTALLER_PATH` and a scratch `$TMPDIR` that fleetd deletes afterward, usually while a user is signed in and may have the app open. The generated scripts (`ee/maintained-apps/ingesters/homebrew/scripts.go`) already handle most of this; read what they do before writing a custom one.

- [Generated scripts](#generated-scripts)
- [When you need a custom script](#when-you-need-a-custom-script)
- [Quit and relaunch](#quit-and-relaunch)
- [Unpacking dmg, zip, and pkg](#unpacking-dmg-zip-and-pkg)
- [Ownership and self-updating apps](#ownership-and-self-updating-apps)
- [Suite casks and nested apps](#suite-casks-and-nested-apps)
- [Rebrands and legacy copies](#rebrands-and-legacy-copies)
- [Uninstall scripts](#uninstall-scripts)
- [Script security](#script-security)

## Generated scripts

From the cask's artifacts and `zap`, the template produces:
- **dmg**: `yes | hdiutil attach` (the `yes` answers a license agreement), `sudo cp -R` of the volume into `$TMPDIR`, which detaches and exits 1 if the copy fails, then a detach that retries and finally forces (a busy volume left attached after fleetd deletes the dmg fills disks).
- **zip**: `ditto -x -k "$INSTALLER_PATH" "$TMPDIR" || exit 1`.
- **pkg**: `installer -pkg "$TMPDIR/<name>"`, where the name is the server's `Content-Disposition` filename, which is also what fleetd names the download.
- **app**: `quit_and_track_application '<id>' || exit 1`, move the old bundle aside, copy the new one, restore the old one if the copy fails, then `relaunch_application` if the app was running.
- **uninstall**: `quit_application '<id>' || exit 1` for each app it removes, `launchctl` removal in both the system and console-user domains, `pkgutil --forget`, and `trash` of `zap` paths as the console user.

A change to `scripts.go` regenerates every non-frozen generated output on the next nightly (about 540 outputs for a dmg change), and frozen and custom outputs need hand-patching in the same PR. Generated scripts must stay byte-identical for legitimate casks: the server's auto-update compares whole script strings to detect admin edits, and `normalizeInstallerFilename` pattern-matches the emitted `"$TMPDIR/<file>"`.

## When you need a custom script

Only when the generated script can't work: a format the template doesn't unpack (a dmg inside a zip, a pkg inside a dmg, an installer app), a `suite` or `artifact` stanza, a rebrand or bundle-id migration, ownership or permission fixes, an app that ignores the AppleScript quit, or vendor-specific steps. Then:

1. Start from the generated script for the same cask (regenerate once without `install_script_path` and copy it), not from another app's custom script.
2. Copy helpers **verbatim** from `scripts.go`. Guard tests in `ingesters/homebrew/scripts_test.go` fail on stale copies: `TestCustomInstallScriptsStopWhenAppWontQuit`, `TestCustomScriptsCountNoProcessAsQuit`, `TestCustomScriptsSkipRelaunchWhenAppWasntRunning`, `TestCustomInstallScriptsStopWhenExtractionFails`, `TestCustomUninstallScriptsUseCurrentQuitHelper`, `TestCustomUninstallScriptsUseCurrentTrash`, `TestCustomUninstallScriptsUseCurrentLaunchctlHelper`, `TestCustomInstallScriptsDoNotUseOldVarNameDerivation`.
3. **When a helper changes in `scripts.go`, sync every copy** in `inputs/homebrew/scripts/`. Grep for the old line as well as the function name: older scripts carry their own variants. Then hand-patch the outputs of custom and frozen apps ([validator-failures.md](validator-failures.md#editing-an-output-without-regenerating)).
4. Prefer `install_script_path` plus `pre_uninstall_scripts`/`post_uninstall_scripts` over a full custom uninstall when the generated uninstall is fine; the two can't be combined.
5. Keep comments admin-facing ([script-comments.md](script-comments.md)).

## Quit and relaunch

`quit_and_track_application` asks the app to quit, then confirms with AppleScript's `is running`, which must succeed *and* print `false` (an error doesn't count as quit). It gives up after about 10 seconds and returns 1, for example when the user cancels a save prompt. Call it as `quit_and_track_application '<id>' || exit 1` **before anything moves or replaces the app**: an app replaced while it's running loses its files when fleetd deletes `$TMPDIR`. A copied helper that confirms the quit with `pgrep -f "$bundle_id"` is stale and the guard test rejects it.

The one exception is a vendor installer that kills the app itself. Read its `preinstall`/`postinstall` (`inspect_mac_installer.sh` prints the relevant lines; [identity-verification.md](identity-verification.md#partial-download-probes) has the range-read recipe for big pkgs) and, only if it really does, keep going after a failed quit and add the script with the reason to `continuesAnyway` in `TestCustomInstallScriptsStopWhenAppWontQuit`. "The pkg's Distribution says `must-close`" doesn't count: that only applies in the Installer GUI.

**Apps that ignore the AppleScript quit.** Some apps defer or refuse `applicationShouldTerminate:` (Prism blocks up to 120 s; PureVPN shows a "disconnect?" alert while connected; Mozilla VPN answers "User canceled"). These scripts quit them with a per-app function that sends SIGTERM to the app's main executable, waits, then sends SIGKILL, called with `|| exit 1` (`purevpn`, `mozilla-vpn`, `tailscale`, `wezterm` scripts, each with a test in `scripts_test.go`). Where the graceful quit usually works, prefer chaining the fallback after it: `quit_and_track_application '<id>' || stop_<app> || exit 1` keeps the stock helper, so relaunch still works. Before relying on SIGTERM, check the app handles it (Mozilla VPN maps it to a clean quit that disconnects first; Prism doesn't catch it, but autosaves). Match processes precisely by executable path and arguments: browsers run Mozilla VPN's executable as a native-messaging host, which must be left alone. `pgrep -f` with a full-path anchor misses processes started as shell-script stubs (`/bin/bash <path>`).

**Relaunch** only after a successful install and only if the app was running (the stock helper records `APP_WAS_RUNNING_<var>`). A vendor postinstall that reopens the app (often as root) means you must not open it a second time. After a bundle-id change, `open -b <new id>` fails because the user's LaunchServices database has never seen the new id: register the bundle as the console user first (`launchctl asuser <uid> sudo -u <user> lsregister -f <app>`, the `box-tools` pattern), then relaunch. If `open -b` fails for a large bundle with restricted entitlements right after install, open it by path with retries and log `open`'s error (`affinity-install.sh`).

**Testing quit and relaunch locally.** LaunchServices won't resolve `application id` or `open -b` for bundles under `/private/tmp` (scratch dirs, dmg mounts), so the helper reports "did not quit" there. Copy the app to a path under your home directory, `lsregister -f` it, test, then `lsregister -u` and delete it and its `~/Library` data. For a stand-in app, `osacompile -s` makes a stay-open applet; add a `CFBundleIdentifier` with `PlistBuddy -c 'Add …'`. Run a copy of the script with `sudo` stripped and paths redirected into your scratch directory.

## Unpacking dmg, zip, and pkg

- **`installer_format` must match the real file**, or the generated preamble fails before anything installs (`hdiutil attach` on a pkg: "image not recognized").
- **Zips: `ditto -x -k`, never `unzip`.** Vendor zips made with `ditto -c -k` store extended attributes as AppleDouble `._*` files; `unzip` writes them into the bundle as real files and breaks the code signature ("sealed resource is missing or invalid"), and files whose signature lives in `com.apple.cs.*` xattrs lose it (Sublime Text). Check the `ditto` exit status.
- **Nested formats**: a zip containing a dmg, or a dmg containing a pkg or zip, needs a custom script that unpacks each layer (`pd-install.sh`). Discover the inner file and app names at run time, so the script survives version bumps; leave the uninstall generated where it can be.
- **Every path that mounts a dmg detaches it**, including failure paths, with the template's retry-then-force sequence. Check the staging copy's exit status.
- **Large payloads** (multi-GB suites): copy straight from the mounted volume with `ditto` or `cp -R` rather than staging twice through `$TMPDIR`.
- **A pkg whose `installer` never returns** (Mozilla VPN on one macOS release): run `installer` in the background with a bounded wait, quit any GUI its postinstall opened, then SIGTERM `installer` and exit 1, so fleetd doesn't hold the script for an hour.
- **Installers that modify their own signed bundle in postinstall** (Topaz, Omnissa Horizon, WiFiman) fail Gatekeeper after install. There's no script fix; freeze or remove the app.

## Ownership and self-updating apps

The generated `sudo cp -R` leaves the bundle `root:wheel` with mode 755/644. That breaks apps whose updater runs as the user:
- **Squirrel.Mac (ShipIt)** renames the bundle out of `/Applications` as the signed-in user, which fails on a root-owned bundle even for admins, so self-update breaks. Spot it by `ShipIt` in the cask's `zap` or `Contents/Frameworks/Squirrel.framework` in the app. The fix in `claude-install.sh`: `chown -R` the bundle to the console user (falling back to `loginwindow`'s `lastUserName`), warning rather than failing if `chown` fails. Only admins benefit: standard users can't write to `/Applications` anyway. A pkg that installs `root:admin 775` (Slack) is fine.
- **Mozilla's updater** accepts group admin with group write: `chown -R root:admin` and `chmod -R a+rX,a-st,ug+w,o-w` (`firefox-install.sh`), matching Mozilla's own pkg. Without it, users get a "Privileged Install Helper" prompt whenever an update is pending.

Use `sudo cp -R` (root-owned, keeps symlinks) rather than `ditto` when you want root ownership: `ditto` preserves the source's ownership.

## Suite casks and nested apps

The ingester only parses `app`, `pkg`, `binary`, `uninstall`, and `zap` stanzas. A `suite` (a folder like `/Applications/KiCad`) or `artifact` stanza generates a script that copies nothing into `/Applications`, so it needs custom scripts (`kicad-install.sh`):
- Copy the folder from the mount, then register the main app with `lsregister -f`. osquery's `apps` table doesn't recurse into folders in `/Applications`, so a nested app only appears once LaunchServices knows it (`openvpn-connect-install.sh` and `box-tools-install.sh` do this). Once the bundle is deleted, osquery drops it even if it's still registered.
- The validator detects the new top-level folder, so suites validate fine.
- Skip `$HOMEBREW_PREFIX` binary symlinks, as the ingester does.
- Never trash user data the suite created (KiCad's `~/Documents/KiCad/<ver>/projects`).
- If the user-facing windows run from nested bundles (KiCad's editors under `Contents/Applications/`), the open query needs a per-token override ([queries.md](queries.md#open-run-it-with-the-app-running)).

## Rebrands and legacy copies

When a new version installs under a new name or bundle id and leaves the old app in place, the patch policy that flags the old id stays red forever unless the install script removes the legacy app. Quit it first (`|| exit 1`), remove it only after the new app is in place, and check the vendor's pre/postinstall doesn't already handle it or deliberately keep the old app (when a vendor keeps it, the queries have to accept it instead). The query side is in [queries.md](queries.md#rebrands-renamed-ids-and-side-by-side-installs).

Don't recreate links or files an app manages itself. Docker Desktop validates its CLI symlinks byte for byte; links written as `"$APPDIR/Docker.app/…"` with `APPDIR="/Applications/"` contain `//`, so Docker sees them as misconfigured and shows a privileged-access prompt after every FMA update. Build paths without doubled slashes.

## Uninstall scripts

- A custom uninstall copies `quit_application` verbatim and calls it with `|| exit 1` for each app whose bundle it removes, as generated scripts do, even when the cask's `uninstall quit` doesn't list the app. Leave helpers, shared updaters (Microsoft AutoUpdate), and other vendors' apps without it: a failed quit there doesn't mean the bundle being removed is still running.
- Remove per-user data with the template's `trash` (as the console user), and only what the cask's `zap` names. Scope destructive sweeps tightly: Box Tools' uninstall removes only the Box Edit subdirectory, because the parent is shared with Box Drive.
- Unload `launchctl` services in both the system domain and the console user's GUI domain; root alone only reaches the system domain.
- Some uninstalls leave privileged helpers the cask also leaves (PureVPN's SMJobBless helper). Match the cask's behavior and note it in the PR, rather than inventing removal steps you can't test.

## Script security

These scripts run as root on every managed Mac that installs the app, and their inputs come from places an attacker could influence (cask metadata, bundle contents, the user's home directory).

- **Never interpolate untrusted text into code.** The template escapes cask values in place (`shellDoubleQuoteEscape` for double-quoted contexts; the pkg choices XML stays in a heredoc only when it's inert, and is single-quoted otherwise) because `$(…)` and backticks survive double quotes and unquoted heredocs. Real app names contain `'`, spaces, and parentheses ("Cycling '74", "Adobe Photoshop (2024).app"), so quoting, not character blocklists, is the control for paths.
- **Pass ids to AppleScript as data**: `osascript -e 'on run argv' -e '… application id (item 1 of argv) …' -e 'end run' -- "$bundle_id"`. The `--` is required: osascript keeps parsing options after the script, so an id like `-e<code>` would add a statement. Cask `quit` ids are also validated at ingest (`validateQuitIDs`).
- **Validate values read from disk** (a bundle id from an `Info.plist`, a path from a user's preferences) against a strict pattern before using them as root.
- **Don't follow user-writable paths as root** without checking them: resolve the console user, act in their home directory as that user (`sudo -u`/`launchctl asuser`), and don't let a symlink there redirect a root-owned `rm` or `chown`.
