# Deploy Firefox in multiple languages with one package

Mozilla ships Firefox as a separate download for every language: en-US, en-GB, de, ja, pt-BR, and roughly a hundred more. There's no single, language-neutral installer, so standing up a multilingual workforce looks like a choice between maintaining a matrix of per-language packages or forcing everyone onto one language. You don't have to do either. This guide shows how to deploy one Firefox [Fleet-maintained app](https://fleetdm.com/guides/fleet-maintained-apps) (FMA) with a post-install script so each host gets the correct language automatically on macOS and Windows, while users stay free to switch.

## How Firefox handles languages

Three facts make this work:

- **The installer is single-locale, but the language packs are not.** Each Firefox release publishes its UI languages as separate `.xpi` files under a per-platform path (for example `.../releases/152.0.5/win64/xpi/`), but the packs themselves are platform-independent. A German pack pulled from the Windows directory is byte-identical to the one under macOS.
- **Language packs are version-specific.** A pack built for 152.0.5 only works with Firefox 152.0.5, so any deployment method has to keep packs in lockstep with the installed version. Pulling packs from Mozilla's release directory by the exact installed version handles this cleanly, and sidesteps the version skew you'll see in third-party catalogs where individual locale packages lag behind the base release.
- **A policy controls which language activates.** Firefox reads a `RequestedLocales` policy. Set it to an empty string and Firefox follows the machine's OS locale, activating a matching language pack if one is present, and still lets the user change languages manually. Set it to a fixed value like `en-US` and the choice is locked. For most fleets, the empty string is what you want.

The plan: install the standard en-US Firefox, drop in the language packs your workforce actually needs, and set `RequestedLocales` to empty. A Fleet post-install script does all of this automatically and re-runs on every update, so it stays correct as Firefox versions change.

## Prerequisites

- Fleet with the Firefox Fleet-maintained app available for your macOS and/or Windows hosts.
- The list of locales your workforce uses. Ship only those, not all hundred.
- Target hosts able to reach `releases.mozilla.org` at install time to download the packs.

> **Note:** A host that can't reach `releases.mozilla.org` simply stays en-US. The scripts below skip a failed download rather than failing the whole install.

## Step 1: Add the Firefox Fleet-maintained app

On the **Software** page, choose your fleet, select **Add software**, open the **Fleet-maintained** tab, and select **Firefox**. This is the standard base install; you'll layer language support on top of it in the next step.

## Step 2: Attach the post-install script

When adding the app, open **Advanced options** to reach the post-install script field. You can also add it later by editing the software item. The post-install script runs after the base install completes, with elevated privileges, so it can write into the Firefox install directory.

Both scripts do the same three things: read the version Firefox just installed, download matching language packs and rename them to the extension-ID format Firefox expects, and write a `policies.json` that tells Firefox to follow the OS locale.

Edit the locale list at the top of each script to match your organization before saving, then paste the script for the platform you're configuring.

> **Warning:** If the post-install script returns a non-zero exit code, Fleet treats the install as failed and attempts to uninstall. Both scripts skip an unavailable pack instead of erroring out so a single unreachable locale doesn't fail the whole install.

### macOS

```zsh
#!/bin/zsh
set -euo pipefail

APP="/Applications/Firefox.app"
DIST="$APP/Contents/Resources/distribution"
EXT="$DIST/extensions"

# Edit to the locales your workforce needs (Mozilla BCP-47 codes).
LOCALES=(de fr es-ES ja pt-BR zh-CN)

VER="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$APP/Contents/Info.plist")"
BASE="https://releases.mozilla.org/pub/firefox/releases/${VER}/mac/xpi"

mkdir -p "$EXT"

for loc in "${LOCALES[@]}"; do
  dest="$EXT/langpack-${loc}@firefox.mozilla.org.xpi"
  if curl -fsSL "${BASE}/${loc}.xpi" -o "$dest"; then
    echo "installed langpack: ${loc}"
  else
    echo "WARNING: could not fetch ${loc} for ${VER} (skipping)"
    rm -f "$dest"
  fi
done

# Empty string = follow the OS locale; users can still switch manually.
cat > "$DIST/policies.json" <<'JSON'
{
  "policies": {
    "RequestedLocales": ""
  }
}
JSON

chown -R root:wheel "$DIST"
chmod -R a+r "$DIST"
exit 0
```

### Windows

```powershell
$ErrorActionPreference = 'Stop'

$FirefoxDir = Join-Path $env:ProgramFiles 'Mozilla Firefox'
$Dist       = Join-Path $FirefoxDir 'distribution'
$Ext        = Join-Path $Dist 'extensions'

# Edit to the locales your workforce needs (Mozilla BCP-47 codes).
$Locales = @('de','fr','es-ES','ja','pt-BR','zh-CN')

$Ver  = (Get-Item (Join-Path $FirefoxDir 'firefox.exe')).VersionInfo.ProductVersion.Trim()
$Base = "https://releases.mozilla.org/pub/firefox/releases/$Ver/win64/xpi"

New-Item -ItemType Directory -Force -Path $Ext | Out-Null
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

foreach ($loc in $Locales) {
  $dest = Join-Path $Ext "langpack-$loc@firefox.mozilla.org.xpi"
  try {
    Invoke-WebRequest -Uri "$Base/$loc.xpi" -OutFile $dest -UseBasicParsing
    Write-Host "installed langpack: $loc"
  } catch {
    Write-Host "WARNING: could not fetch $loc for $Ver (skipping)"
    if (Test-Path $dest) { Remove-Item $dest -Force }
  }
}

# Empty string = follow the OS locale; users can still switch manually.
$policies = @{ policies = @{ RequestedLocales = '' } } | ConvertTo-Json -Depth 5
$policies | Set-Content -Path (Join-Path $Dist 'policies.json') -Encoding UTF8
exit 0
```

> **Note:** The packs download as `de.xpi`, `fr.xpi`, and so on, but Firefox won't load them unless they're renamed to the extension-ID form, `langpack-de@firefox.mozilla.org.xpi`. The scripts handle this. They also place the packs and policy in the right location per platform: inside `Firefox.app/Contents/Resources/distribution/` on macOS, and `C:\Program Files\Mozilla Firefox\distribution\` on Windows.

## Step 3: Target and deploy

Scope the software to the hosts you want under **Target**, either all hosts or a label-based subset. From there, deploy through any of Fleet's normal paths: install on demand from the host's **Software** tab, let end users install it from self-service, or install automatically with a policy.

## Keep language packs current

Language packs have to match the Firefox version, so the deployment has to refresh them on every upgrade. This is handled for you: add a [patch policy](https://fleetdm.com/guides/automatic-software-install-in-fleet) so Fleet keeps Firefox up to date. Because the patch reinstalls the app, your post-install script runs again and re-fetches packs matching the new version. The setup is self-healing: a wiped or stale distribution directory is repopulated on the next install.

## How language packs affect patch policies

If a user switches Firefox to another language, does that change the app's identity and break the patch policy that keeps it up to date? It does not.

On Windows, a patch policy identifies Firefox by the name and publisher in the `programs` table, for example:

```sql
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM programs
  WHERE name = 'Mozilla Firefox (x64 en-US)'
    AND publisher = 'Mozilla'
    AND version_compare(version, '152.0.5') < 0
);
```

That `(x64 en-US)` in the name is written by the installer at install time and reflects the installer's build locale, not the language Firefox is currently displaying. Because this guide installs the en-US base and layers runtime language packs on top, the installer never re-runs and never rewrites that entry. A user running Firefox in German changes only the UI language: the name stays `Mozilla Firefox (x64 en-US)`, the publisher stays the same, and only the version field moves as the app updates. The policy above keeps matching on every host, in every language.

This is a real advantage over deploying a separate installer per language. Per-locale installers each register a different name, `Mozilla Firefox (x64 de)`, `Mozilla Firefox (x64 fr)`, and so on, so an exact-match policy would miss every non-en-US host and force you into one policy per locale. A single base install keeps one stable identity fleet-wide, so one policy covers everyone.

Two things worth confirming against your own hosts:

- **Publisher value.** Check that `programs.publisher` reports `Mozilla` and not `Mozilla Corporation` on your fleet. If it's the longer string, the `AND publisher = ...` clause silently never matches. Mirror whatever value the Firefox Fleet-maintained app uses in its own detection.
- **Robustness option.** If mixed installer locales might ever appear (for example, a machine someone set up by hand), `name LIKE 'Mozilla Firefox (x64 %)'` tolerates any locale suffix while still pinning the architecture. A pure language-pack deployment doesn't need it, but it's inexpensive insurance.

## Things to know

- **GitOps.** These steps use the Fleet UI, where post-install scripts on Fleet-maintained apps are supported. Applying a post-install script to an FMA through GitOps is also possible.
- **Forcing a language.** To pin a specific language instead of following the OS, replace the empty string with an ordered list, for example `"RequestedLocales": ["de", "en-US"]`. This locks the UI language and users can't change it.
- **macOS configuration profile alternative.** You can manage the `RequestedLocales` policy with a configuration profile (managed preference domain `org.mozilla.firefox`) instead of writing `policies.json`. The language pack files still have to live in the app bundle, so you'd keep the download portion of the script and drop the policy-writing portion.
- **Which locales to ship.** Resist shipping every language. A short list keeps installs fast and downloads small; add locales as your workforce grows into them.

## Further reading

- [Fleet-maintained apps](https://fleetdm.com/guides/fleet-maintained-apps)
- [Automatic software install in Fleet](https://fleetdm.com/guides/automatic-software-install-in-fleet)

<meta name="articleTitle" value="Deploy Firefox in multiple languages with one package">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-07-10">
<meta name="category" value="guides">
<meta name="description" value="Deploy one Firefox Fleet-maintained app with a post-install script so each host gets the correct language automatically on macOS and Windows.">
