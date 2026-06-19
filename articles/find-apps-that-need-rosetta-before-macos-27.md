# Find apps that need Rosetta before macOS 27

Apple is closing the door on Rosetta 2\. macOS 27, due in fall 2026, runs only on Apple silicon and removes Rosetta 2 during the upgrade if it was already installed. Users can reinstall it on demand, but macOS 27 is the last release with full Rosetta support. macOS 28, expected in fall 2027, drops it for almost every app. Apple committed to this timeline at WWDC 2025 and confirmed the macOS 27 specifics at WWDC 2026. See [Apple's Rosetta documentation](https://developer.apple.com/documentation/apple-silicon/about-the-rosetta-translation-environment) for the details.

If your fleet runs any Intel-only Mac apps, find them now. You have a full release cycle to test replacements before the apps stop working. This post covers three ways to find Rosetta-dependent apps with osquery and Fleet, and two mistakes that will give you wrong answers.

## The signal you want

An app needs Rosetta when its executable is Intel-only. On an Apple silicon Mac, a binary with an `x86_64` slice and no `arm64` slice runs under Rosetta. A universal binary carries both slices and runs native. An Apple silicon-native binary carries only `arm64`.

So the question for each app is simple: does its executable include an `arm64` slice? If not, it needs Rosetta.

One edge case to keep in mind: a universal app can still need Rosetta if it loads Intel-only frameworks, plugins, or helper binaries. The methods below check the main executable, so apps with Intel-only plugins inside a universal host (common in audio production tools) may pass the audit but still fail under macOS 28.

## What does not work: the app bundle

Each macOS app ships an `Info.plist` with metadata, so the architecture seems like it should live there too. It does not. We queried the `plist` table for an `NSExecutableArchitectures` key across `/Applications` and got nothing back. The join worked and the plist read fine, but the key is not there.

macOS reads the executable architecture from the Mach-O header at launch, not from `Info.plist`. The architecture has to come from the binary itself, or from an index that already read it. Both options follow.

## Method 1: what is running under Rosetta right now

The fastest signal is to ask which processes are running translated at this moment. The `processes` table has a `translated` column on macOS: `1` means the process runs under Rosetta, `0` means native, and `-1` means osquery could not tell.

```sql
SELECT pid, name, path FROM processes WHERE translated = 1;
```

This shows active usage, which is the best signal for what your users depend on. One caveat: a translated osquery process cannot detect that it is itself translated. Fleet runs fleetd and osquery natively on Apple silicon, so this does not affect your results, and every other translated process reports correctly.

## Method 2: what is installed, through Spotlight

To find Intel-only apps that are installed but not running, use Spotlight through the `mdfind` table. Spotlight indexes each app's executable architecture under `kMDItemExecutableArchitectures`. An Intel-only app lists `x86_64` with no `arm64`. A universal app lists both.

The reliable approach is a set difference: find every app bundle with an `x86_64` slice, then remove the ones that also carry an `arm64` slice.

```sql
SELECT path FROM mdfind
WHERE query = 'kMDItemExecutableArchitectures == "x86_64" && kMDItemContentType == "com.apple.application-bundle"'
  AND path NOT IN (
    SELECT path FROM mdfind
    WHERE query = 'kMDItemExecutableArchitectures == "arm64" && kMDItemContentType == "com.apple.application-bundle"'
  );
```

Sample output:

```
/Applications/Wine Stable.app
/Applications/YubiKey Manager.app
/Library/Printers/RICOH/Filters/pstopsRV2.app
/Library/Application Support/Adobe/Adobe Desktop Common/DEBox/Setup.app
```

Do the set difference in SQL, not inside the Spotlight query. `kMDItemExecutableArchitectures` is an array, and matching `!= "arm64"` against an array gives unreliable results. The `NOT IN` approach is predictable.

Spotlight searches the whole disk, so this finds Rosetta-dependent bundles outside `/Applications`, including printer filters and installer helpers. That is good for coverage. To focus on user-facing apps, scope the results to `/Applications`.

To verify a single app, read its value with the `mdls` table:

```sql
SELECT key, value FROM mdls
WHERE path = '/Applications/Wine Stable.app'
  AND key = 'kMDItemExecutableArchitectures';
```

## The arm64e gotcha

Apple's own apps will fool a careless architecture check. Run `lipo -archs` on Safari and you get:

```
x86_64 arm64e
```

Safari is universal and runs native on Apple silicon. It does not need Rosetta. But `arm64e` is not the same string as `arm64`. A check that matches the exact word `arm64` skips `arm64e` and flags Safari, along with every other Apple system app, as Intel-only.

`arm64e` is the Apple silicon variant that adds pointer authentication. Apple compiles its system binaries this way. Third-party apps almost always ship plain `arm64`. Any architecture check has to treat both `arm64` and `arm64e` as native.

Spotlight handles this for you. Read `kMDItemExecutableArchitectures` for Safari and Spotlight reports:

```
arm64,x86_64
```

Spotlight normalizes `arm64e` to `arm64`, so the `mdfind` method already gets Safari right. A hand-written binary scan does not get that help, which leads to the next point.

## Method 3: a fallback for hosts without Spotlight

The `mdfind` method depends on Spotlight. If indexing is off, or `/Applications` is excluded from the index, `mdfind` returns fewer results than the truth. Some managed fleets disable Spotlight, so plan for it.

For those hosts, read the Mach-O header with `lipo` in a script and deploy it with Fleet. Match on the `arm64` substring so `arm64e` counts as native:

```shell
for app in /Applications/*.app; do
  exe="$app/Contents/MacOS/$(defaults read "$app/Contents/Info.plist" CFBundleExecutable 2>/dev/null)"
  if [ -f "$exe" ]; then
    archs=$(lipo -archs "$exe" 2>/dev/null)
    if [ -n "$archs" ] && ! echo "$archs" | grep -q 'arm64'; then
      echo "NEEDS ROSETTA: $(basename "$app") [$archs]"
    fi
  fi
done
```

`grep -q 'arm64'` matches both `arm64` and `arm64e`, so universal apps clear the check. Run this against the same Mac as the `mdfind` query and compare the two lists. If the script finds apps that `mdfind` missed, that host has a Spotlight indexing gap, and the script is your source of truth there.

## Turn this into a fleet-wide check

Save the `processes` and `mdfind` queries in Fleet to inventory Intel-only apps across your Apple silicon hosts. If you also want a yes/no compliance signal you can automate against, add a policy that inverts the question: for example, a policy that fails when Rosetta is missing, or when any Intel-only apps remain in `/Applications`. Once you know which apps need Rosetta, you have two jobs before macOS 27 reaches your fleet:

1. Decide which Intel-only apps to keep. Replace the rest with native or universal builds.
2. For the apps you keep, you'll need to reinstall Rosetta after the macOS 27 upgrade. Use Fleet's policy automation to handle this without manual steps: write a policy that fails when Rosetta is missing on macOS 27, then attach the reinstall script (`softwareupdate --install-rosetta --agree-to-license`) as the policy's automated remediation. Fleet runs the script on every host that fails the policy, and stops once the host passes.

Start the audit while macOS 26 and macOS 27 still run Rosetta. That gives you a full release cycle to test replacements before macOS 28 removes it.

<meta name="articleTitle" value="Find apps that need Rosetta before macOS 27">
<meta name="authorFullName" value="Josh Roskos">
<meta name="authorGitHubUsername" value="kc9wwh">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-06-18">
<meta name="articleImageUrl" value="../website/assets/images/find-rosetta-apps-before-macos-27-1200x627@2x.jpg">
<meta name="description" value="Find which Mac apps need Rosetta across your fleet with osquery and Fleet, before macOS 27 removes it on upgrade.">
