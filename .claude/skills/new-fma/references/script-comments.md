# Comments in shipped scripts

FMA install/uninstall scripts are not internal code. They're returned verbatim by `GET /fleet/software/fleet_maintained_apps/:id` and by the software title endpoint, and rendered in the "Install script" / "Uninstall script" editors of the Edit software modal ([AdvancedOptionsFields.tsx](../../../../frontend/pages/SoftwarePage/components/forms/AdvancedOptionsFields/AdvancedOptionsFields.tsx)), where an admin reads them and can edit them. Every comment you leave is product copy — treat it like the app description, not like a commit message.

Budget: the Fleet template header (`# Learn more about .exe install scripts:` + URL) if the script started from a template, then **at most ~4 lines** of app-specific comment. Of the 580 scripts in `inputs/*/scripts/`, only 51 open with a longer block than that — a big header is the exception you have to justify, not the norm.

**Keep** a comment only if an admin who edits this script would break something without it, or would be surprised at install time:
- Host-visible side effects: the app is force-quit, users are logged out, a reboot happens, existing config is preserved or deleted.
- Scope and destructiveness decisions — e.g. [box-tools-uninstall.sh](../../../../ee/maintained-apps/inputs/homebrew/scripts/box-tools-uninstall.sh): removal sweeps every local user's home, and only the Box Edit subdirectory goes because the parent is shared with Box Drive.
- Constraints that must survive an edit: the required switch and why the obvious one is wrong (`/VERYSILENT` — this is Inno Setup, `/S` opens the GUI), removal ordering, "must run as the logged-in user."
- Exit-code meanings (`1605` = not installed, `3010` = reboot required).

**Cut** — this belongs in the PR description, not the shipped script:
- Fleet's own tooling: "the validator's 10-minute timeout", "hangs in CI", "the ingester", "osquery's programs table". A customer has no validator.
- Catalog archaeology: what winget/Homebrew metadata claimed vs. reality, `silentinstallhq.com` links, PR/issue numbers.
- Debugging narrative: what you tried first and why it failed ("a plain `Start-Process -Wait` would block until killed").
- First person ("we", "our", "ourselves") — describe what the script does, in present tense and sentence case.
- Restating the next line (`# Prints the exit code` above a `Write-Host`).

If the fact matters at run time rather than at edit time, `Write-Host`/`echo` it instead of commenting it — script output lands in the host's software install details, which is where an admin debugging a failure actually looks.

Before/after — [darktable_install.ps1](../../../../ee/maintained-apps/inputs/winget/scripts/darktable_install.ps1)'s 20-line header carries three admin-relevant facts and 16 lines of internal history:
```powershell
# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# darktable uses an Inno Setup installer: it needs /VERYSILENT (the NSIS /S
# switch winget's metadata implies does nothing) and installs machine-wide when
# elevated. Its installer stays running after a silent install, so this script
# waits for darktable to register in Programs and Features, then stops it.
```
Dropped: that winget mislabeled the installer type, that `PrivilegesRequiredOverridesAllowed=dialog` rules out `/ALLUSERS`, that the lingering process holds the installer file lock, what a plain `-Wait` did. All of it goes in the PR body, where reviewers need it and customers don't see it.

Body comments follow the same rule — keep the one above a non-obvious registry match or a load-bearing helper, drop the rest. **When you touch an existing script for any reason, prune its comments in the same edit.**

