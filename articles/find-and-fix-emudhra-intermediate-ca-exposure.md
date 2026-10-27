# Find and fix exposure to the eMudhra intermediate CA compromise

On October 20, 2026, Mozilla [disclosed](https://emudhra.we-have-a-problem.com) that eMudhra generated the private keys of eight intermediate certificate authorities (CAs) incorrectly, so anyone can recover the keys from the certificates. Six of the CAs issue TLS certificates and two issue code-signing certificates, and each also has a Sectigo cross-sign, for 16 certificates in total. Anyone with a recovered key can issue a certificate that a vulnerable TLS client accepts for any domain. This guide shows you how to find the hosts whose TLS clients still trust these certificates, and how to fix them with Fleet policies, scripts, and a Windows configuration profile.

## Why OS and browser updates aren't enough

Browsers and operating system certificate verifiers already block the compromised intermediates. Firefox and Chrome have blocked them since October 17, Apple since October 19, and Microsoft since October 20 through its disallowed certificate list.

Those blocks don't reach software that verifies certificates against a static CA bundle, a PEM file of trusted roots. These clients don't check revocation. If the bundle includes one of the two roots the intermediates chain to, the client trusts certificates issued by the compromised intermediates:

- emSign ECC Root CA - G3 (SHA-1 `3043FA4FF257DCA0C380EE2E58EA78B23FE6BBC1`), trusted by all four major root programs.
- AAA Certificate Services (SHA-1 `D1EB23A46D17D68FD92564C2F1F1601764D8E349`), trusted by Apple and Microsoft.

On macOS, `/usr/bin/curl` and other built-in command-line tools use `/etc/ssl/cert.pem`. On Linux, most TLS clients use the system CA bundle. fleetd's osquery uses its own bundle, `certs.pem`, to verify your Fleet server, and that bundle includes emSign ECC Root CA - G3.

## Prerequisites

- A Fleet account with permission to add policies and run scripts.
- [Scripts enabled](https://fleetdm.com/guides/scripts) on fleetd for the hosts you want to fix.
- Windows MDM turned on, only if you plan to deploy the Windows configuration profile in step 4.

## Check a host

On a Mac, compare the built-in `curl` with `nscurl`, which uses Apple's TLS stack. Both connect to a test site whose certificate was issued by one of the compromised intermediates:

```bash
/usr/bin/curl -sS -o /dev/null -w '%{http_code}\n' https://dv-g3b-emudhra.we-have-a-problem.com/
nscurl https://dv-g3b-emudhra.we-have-a-problem.com/
```

On macOS 26.5.1, `curl` prints `200` and `nscurl` fails with "The certificate for this server is invalid." The `200` means `curl` trusts the compromised intermediate. After you run the macOS script in step 2, `curl` fails too.

On Windows 11, `Invoke-WebRequest` against the same URL fails once Windows has synced Microsoft's disallowed certificate list.

## Step 1: Fix fleetd's CA bundle on macOS and Linux

osquery checks your Fleet server's TLS certificate against `/opt/orbit/certs.pem`. While that file trusts emSign ECC Root CA - G3, anyone holding one of the compromised keys could impersonate your Fleet server to osquery.

Add this policy. It passes when the bundle contains neither root:

```yaml
- name: fleetd CA bundle excludes compromised eMudhra roots (macOS and Linux)
  query: |
    SELECT 1 WHERE NOT EXISTS (
      SELECT 1 FROM file f
      JOIN certificates c ON c.path = f.path
      WHERE f.path = '/opt/orbit/certs.pem'
        AND lower(c.sha1) IN (
          '3043fa4ff257dca0c380ee2e58ea78b23fe6bbc1',
          'd1eb23a46d17d68fd92564c2f1f1601764d8e349'
        )
    );
  critical: false
  description: >-
    osquery uses /opt/orbit/certs.pem to verify the Fleet server's TLS certificate. fleetd packages built before
    the fix ship a bundle that trusts emSign ECC Root CA - G3, and osquery doesn't check revocation. Anyone holding one of the
    eight compromised eMudhra intermediate keys could impersonate your Fleet server to osquery on this host.
  resolution: >-
    Run remove-emudhra-roots-fleetd.sh on this host, or reinstall fleetd with a package built by a fleetctl
    release that excludes the emSign roots.
  platform: darwin,linux
  calendar_events_enabled: false
```

For hosts that fail, run this script, or set it as the policy's [script automation](https://fleetdm.com/guides/policy-automation-run-script) so Fleet runs it on each failing host. It removes emSign ECC Root CA - G3, two other emSign roots, and the Comodo AAA Services root from the bundle, keeps a backup at `/opt/orbit/certs.pem.bak-emudhra`, and is safe to run more than once:

```sh
#!/bin/sh
# Removes the emSign roots excluded by curl's CA bundle (eMudhra intermediate CA compromise,
# https://emudhra.we-have-a-problem.com) and the legacy Comodo AAA root from fleetd's osquery CA bundle.
# Safe to run more than once.

BUNDLE="${1:-/opt/orbit/certs.pem}"

if [ ! -f "$BUNDLE" ]; then
  echo "No bundle at $BUNDLE, nothing to do."
  exit 0
fi

TMP="$BUNDLE.tmp.$$"
trap 'rm -f "$TMP"' EXIT

# certs.pem uses mk-ca-bundle's layout: a name line, an "=====" underline, then the PEM block.
awk '
  BEGIN {
    drop["emSign ECC Root CA - G3"] = 1
    drop["emSign Root CA - C1"] = 1
    drop["emSign ECC Root CA - C3"] = 1
    drop["Comodo AAA Services root"] = 1
  }
  { line[NR] = $0 }
  END {
    skip = 0
    for (i = 1; i <= NR; i++) {
      if (!skip && (line[i] in drop) && line[i + 1] ~ /^=+$/) skip = 1
      if (!skip) print line[i]
      if (skip && line[i] ~ /^-----END CERTIFICATE-----/) skip = 0
    }
  }
' "$BUNDLE" > "$TMP" || exit 1

before=$(grep -c -- '-----BEGIN CERTIFICATE-----' "$BUNDLE")
after=$(grep -c -- '-----BEGIN CERTIFICATE-----' "$TMP")
removed=$((before - after))

if [ "$removed" -eq 0 ]; then
  echo "$BUNDLE doesn't contain the affected roots ($before certificates)."
  exit 0
fi

# Guard against a malformed bundle stripping more than the four named roots.
if [ "$removed" -gt 4 ] || [ "$after" -lt 50 ]; then
  echo "Unexpected result ($before -> $after certificates), leaving $BUNDLE unchanged." >&2
  exit 1
fi

[ -f "$BUNDLE.bak-emudhra" ] || cp -p "$BUNDLE" "$BUNDLE.bak-emudhra" || exit 1
# Overwrite in place to keep the file's owner and mode.
cat "$TMP" > "$BUNDLE" || exit 1

echo "Removed $removed certificate(s) from $BUNDLE ($before -> $after). Backup: $BUNDLE.bak-emudhra"
```

> **Note:** [fleetdm/fleet#56173](https://github.com/fleetdm/fleet/pull/56173) removes these roots from `certs.pem` in new fleetd packages. Until you build packages with that fix, hosts you enroll get the affected bundle and need the script too.

## Step 2: Fix the command-line trust store on macOS

macOS blocks the compromised intermediates for Safari and apps that use Apple's TLS stack. The built-in `curl`, `git`, `python3`, `perl`, `ruby`, and `openssl` use `/etc/ssl/cert.pem` instead, so they still trust the compromised intermediates.

Add this policy:

```yaml
- name: macOS command-line TLS trust store excludes compromised eMudhra roots
  query: |
    SELECT 1 WHERE NOT EXISTS (
      SELECT 1 FROM file f
      JOIN certificates c ON c.path = f.path
      WHERE f.path = '/etc/ssl/cert.pem'
        AND lower(c.sha1) IN (
          '3043fa4ff257dca0c380ee2e58ea78b23fe6bbc1',
          'd1eb23a46d17d68fd92564c2f1f1601764d8e349'
        )
    );
  critical: false
  description: >-
    macOS blocks the compromised eMudhra intermediates for Safari and apps that use Apple's TLS stack. The built-in
    curl, git, python3, perl, ruby, and openssl use /etc/ssl/cert.pem instead and don't check revocation, so they
    still trust certificates issued by the compromised intermediates.
  resolution: >-
    Run remove-emudhra-roots-macos-system.sh. A macOS update can restore /etc/ssl/cert.pem, so keep this policy
    in place until Apple ships an updated file.
  platform: darwin
  calendar_events_enabled: false
```

For hosts that fail, run this script. It removes AAA Certificate Services and emSign ECC Root CA - G3 from `/etc/ssl/cert.pem` and keeps a backup at `/etc/ssl/cert.pem.bak-emudhra`:

```sh
#!/bin/sh
# Removes AAA Certificate Services and emSign ECC Root CA - G3 from /etc/ssl/cert.pem, the bundle macOS's
# built-in curl, git, python3, perl, ruby, and openssl use without revocation checks
# (eMudhra intermediate CA compromise, https://emudhra.we-have-a-problem.com/#macos-curl).
# Apple's own TLS stack (Safari, URLSession) already blocks the compromised intermediates.
# Safe to run more than once. A macOS update can restore the original file.

BUNDLE="${1:-/etc/ssl/cert.pem}"

if [ ! -f "$BUNDLE" ]; then
  echo "No bundle at $BUNDLE, nothing to do."
  exit 0
fi

TMP="$BUNDLE.tmp.$$"
trap 'rm -f "$TMP"' EXIT

# cert.pem groups certificates under "### <Organization>" lines with one "=== <subject>" line per certificate.
awk '
  /^(###|===) / { skip = 0 }
  /^=== .*\/CN=(AAA Certificate Services|emSign ECC Root CA - G3)$/ { skip = 1 }
  !skip
' "$BUNDLE" > "$TMP" || exit 1

before=$(grep -c -- '-----BEGIN CERTIFICATE-----' "$BUNDLE")
after=$(grep -c -- '-----BEGIN CERTIFICATE-----' "$TMP")
removed=$((before - after))

if [ "$removed" -eq 0 ]; then
  echo "$BUNDLE doesn't contain the affected roots ($before certificates)."
  exit 0
fi

if [ "$removed" -gt 2 ] || [ "$after" -lt 50 ]; then
  echo "Unexpected result ($before -> $after certificates), leaving $BUNDLE unchanged." >&2
  exit 1
fi

[ -f "$BUNDLE.bak-emudhra" ] || cp -p "$BUNDLE" "$BUNDLE.bak-emudhra" || exit 1
cat "$TMP" > "$BUNDLE" || exit 1

echo "Removed $removed certificate(s) from $BUNDLE ($before -> $after). Backup: $BUNDLE.bak-emudhra"
```

> **Warning:** A macOS update can restore the original `/etc/ssl/cert.pem`. Keep the policy in place so you see any host where that happens, and run the script again.

## Step 3: Update the system CA bundle on Linux

Most Linux TLS clients, like `curl`, `wget`, package managers, and Python, verify against the system CA bundle and don't check revocation. Add this policy. It checks the three common system bundle paths:

```yaml
- name: Linux system CA bundle excludes compromised eMudhra roots
  query: |
    SELECT 1 WHERE NOT EXISTS (
      SELECT 1 FROM file f
      JOIN certificates c ON c.path = f.path
      WHERE f.path IN (
          '/etc/ssl/certs/ca-certificates.crt',
          '/etc/pki/tls/certs/ca-bundle.crt',
          '/etc/ssl/ca-bundle.pem'
        )
        AND lower(c.sha1) IN (
          '3043fa4ff257dca0c380ee2e58ea78b23fe6bbc1',
          'd1eb23a46d17d68fd92564c2f1f1601764d8e349'
        )
    );
  critical: false
  description: >-
    Most Linux TLS clients (curl, wget, package managers, Python, Go binaries older than 1.27.3) verify against the
    system CA bundle and don't check revocation. While the bundle includes emSign ECC Root CA - G3 or AAA
    Certificate Services, they trust certificates issued by the compromised eMudhra intermediates.
  resolution: >-
    Update the ca-certificates package (Ubuntu 20260816ubuntu0.*, Debian 20250419~deb12u2 / 20250419+deb13u1,
    Fedora 2026.2.92_v9.0.319, Alpine 20261020-r0 or later). If your distribution hasn't shipped an update yet,
    see the distrust steps at https://emudhra.we-have-a-problem.com/#linux.
  platform: linux
  calendar_events_enabled: false
```

To fix a failing host, update its `ca-certificates` package. If your distribution hasn't shipped an update yet, follow the [Linux distrust steps](https://emudhra.we-have-a-problem.com/#linux) on the disclosure site.

## Step 4: Confirm Windows has Microsoft's blocklist

Windows downloads Microsoft's disallowed certificate list automatically, and Microsoft added the 16 certificates on October 20. Add this policy. It passes when the list was last synced on or after October 21, 2026 (UTC), or when all 16 certificates are in the Untrusted Certificates store:

```yaml
- name: Windows has Microsoft's blocklist for compromised eMudhra intermediates
  query: |
    SELECT 1 FROM registry
    WHERE key = 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\SystemCertificates\AuthRoot\AutoUpdate'
      AND name = 'DisallowedCertLastSyncTime'
      AND substr(data, 15, 2) || substr(data, 13, 2) || substr(data, 11, 2) || substr(data, 9, 2) ||
          substr(data, 7, 2) || substr(data, 5, 2) || substr(data, 3, 2) || substr(data, 1, 2) >= '01DD60EF1D89C000'
    UNION
    SELECT 1 WHERE (
      SELECT COUNT(DISTINCT upper(sha1)) FROM certificates
      WHERE store IN ('Untrusted Certificates', 'Disallowed')
        AND upper(sha1) IN (
          'A99B289464F3B44343270A7A1E910F02370483FE', '1396BE1372BC93DFA2A433CA964DBA91548A6E57',
          '5F014E0DB3BB8F950E0598AB280846D383FC57BE', '5DFA19197D1C1CF629AF7C0CD865543C5050227C',
          'F33ACD1F5D31AA81AC18C72C928446226D3D5914', '19069BD6533BFF93FB67C71FB3376B4EA7402C3B',
          '8AD23BC78845D6FAAE1D534BAC89F0E36BA55A11', '4B0944C14F8C2CCBA4E4F32104AD920FA15EE331',
          '7FCC31AAB874CE6C8EE8584F54CFD08646B54117', '305FC1E253DCAB71B95D4995DAE3CB09D562F22A',
          'C988F3D0CC1629FFC43165D6464C9B6EF73D6E48', '8588708E875A1F8AA83524F3559080F7E17068C7',
          'D33195A244046A3263FB5E0E83C6803A5B2E7BC3', 'EE26D7AE3CC71718F6E07D7D0095076F0C158F63',
          '1F4A5D178F8B6A8EF0F6784155BD5FEA91158B12', '7FDE64753A957B98F172FD2AB0D922551909C7BA'
        )
    ) = 16;
  critical: false
  description: >-
    Microsoft blocked the compromised eMudhra intermediates (TLS and code signing) on 2026-10-20 through the
    disallowed certificate list that Windows downloads automatically. The policy passes when that list was last
    synced on or after 2026-10-21 (UTC), or when all 16 certificates were pushed to the Untrusted Certificates
    store with windows-distrust-emudhra.xml.
  resolution: >-
    Make sure the host can reach Windows Update and that the "Turn off Automatic Root Certificates Update" Group
    Policy (DisableRootAutoUpdate) isn't enabled. For hosts that can't sync, deploy windows-distrust-emudhra.xml.
  platform: windows
  calendar_events_enabled: false
```

For hosts that fail:

1. Make sure the host can reach Windows Update, and that the **Turn off Automatic Root Certificates Update** Group Policy setting (`DisableRootAutoUpdate`) isn't enabled.
2. Run this script. It removes the same four roots from fleetd's `C:\Program Files\Orbit\certs.pem`, then reports when Windows last synced the disallowed certificate list and whether automatic root updates are turned off:

   ```powershell
   # Removes the emSign roots excluded by curl's CA bundle (eMudhra intermediate CA compromise,
   # https://emudhra.we-have-a-problem.com) and the legacy Comodo AAA root from fleetd's osquery CA bundle,
   # then reports whether Windows has Microsoft's blocklist for the compromised intermediates.
   # Safe to run more than once.

   param([string]$Bundle = "$env:ProgramFiles\Orbit\certs.pem")

   $drop = @('emSign ECC Root CA - G3', 'emSign Root CA - C1', 'emSign ECC Root CA - C3', 'Comodo AAA Services root')

   if (Test-Path -LiteralPath $Bundle) {
       $lines = [IO.File]::ReadAllLines($Bundle)
       $out = New-Object System.Collections.Generic.List[string]
       $skip = $false
       # certs.pem uses mk-ca-bundle's layout: a name line, an "=====" underline, then the PEM block.
       for ($i = 0; $i -lt $lines.Length; $i++) {
           if (-not $skip -and $drop -contains $lines[$i] -and $i + 1 -lt $lines.Length -and $lines[$i + 1] -match '^=+$') { $skip = $true }
           if (-not $skip) { $out.Add($lines[$i]) }
           if ($skip -and $lines[$i] -like '-----END CERTIFICATE-----*') { $skip = $false }
       }

       $before = @($lines | Where-Object { $_ -like '-----BEGIN CERTIFICATE-----*' }).Count
       $after = @($out | Where-Object { $_ -like '-----BEGIN CERTIFICATE-----*' }).Count
       $removed = $before - $after

       if ($removed -eq 0) {
           Write-Output "$Bundle doesn't contain the affected roots ($before certificates)."
       } elseif ($removed -gt 4 -or $after -lt 50) {
           # Guard against a malformed bundle stripping more than the four named roots.
           Write-Error "Unexpected result ($before -> $after certificates), leaving $Bundle unchanged."
           exit 1
       } else {
           $backup = "$Bundle.bak-emudhra"
           if (-not (Test-Path -LiteralPath $backup)) { Copy-Item -LiteralPath $Bundle -Destination $backup }
           # Write LF endings without a BOM to match the file fleetd installs.
           [IO.File]::WriteAllText($Bundle, (($out -join "`n") + "`n"), (New-Object System.Text.UTF8Encoding($false)))
           Write-Output "Removed $removed certificate(s) from $Bundle ($before -> $after). Backup: $backup"
       }
   } else {
       Write-Output "No bundle at $Bundle, nothing to do."
   }

   $autoUpdate = Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\SystemCertificates\AuthRoot\AutoUpdate' -ErrorAction SilentlyContinue
   if ($autoUpdate.DisallowedCertLastSyncTime) {
       $synced = [DateTime]::FromFileTimeUtc([BitConverter]::ToInt64($autoUpdate.DisallowedCertLastSyncTime, 0))
       Write-Output ("Disallowed certificate list last synced {0:u}" -f $synced)
       if ($synced -lt [DateTime]::new(2026, 10, 21, 0, 0, 0, [DateTimeKind]::Utc)) {
           Write-Output "That sync predates Microsoft's 2026-10-20 block of the eMudhra intermediates."
       }
   } else {
       Write-Output "Windows hasn't synced the disallowed certificate list."
   }
   if ((Get-ItemProperty 'HKLM:\SOFTWARE\Policies\Microsoft\SystemCertificates\AuthRoot' -ErrorAction SilentlyContinue).DisableRootAutoUpdate -eq 1) {
       Write-Output "Automatic root certificate updates are turned off by policy. Deploy windows-distrust-emudhra.xml."
   }
   ```

3. For hosts that can't sync, like hosts with automatic root updates turned off by policy, add the [`windows-distrust-emudhra.xml`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/configuration-profiles/windows-distrust-emudhra.xml) configuration profile. It uses the [RootCATrustedCertificates CSP](https://learn.microsoft.com/en-us/windows/client-management/mdm/rootcacertificates-csp) to add all 16 certificates to the Untrusted Certificates store. Go to **Controls > OS settings > Configuration profiles** and click **Add profile**, or add it to `controls.windows_settings.configuration_profiles` in GitOps.

> **Warning:** The profile is well-formed XML, but it hasn't been tested through MDM yet. Deploy it to a test fleet first, and confirm the Windows policy passes on those hosts before you deploy it more widely.

## Update Fleet and fleetd

Go 1.27.3 blocks the eight compromised intermediates in `crypto/x509`, so Go binaries built with it reject these chains. Fleet 4.93.1 and fleetd's Orbit 1.62.0 were built with Go 1.27.1. When Fleet and fleetd releases built with Go 1.27.3 ship, update your Fleet server and fleetd. Until then, use the policies and scripts in this guide.

## Further reading

- [eMudhra intermediate CA compromise disclosure](https://emudhra.we-have-a-problem.com)
- [Mozilla Bugzilla 2083449](https://bugzilla.mozilla.org/show_bug.cgi?id=2083449)
- [What's a policy?](https://fleetdm.com/securing/what-are-fleet-policies)
- [Scripts in Fleet](https://fleetdm.com/guides/scripts)
- [Automatically run scripts on failing policies](https://fleetdm.com/guides/policy-automation-run-script)
- [Custom OS settings](https://fleetdm.com/guides/custom-os-settings)

<meta name="articleTitle" value="Find and fix exposure to the eMudhra intermediate CA compromise">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-10-21">
<meta name="category" value="guides">
<meta name="description" value="Use Fleet policies and scripts to find and fix hosts that still trust the compromised eMudhra intermediate CAs.">
