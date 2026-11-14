# Find Windows devices that can't renew their Intune MDM certificate

An Intune-managed Windows device renews its MDM certificate about once a year. When the renewal fails, the device keeps syncing and reporting compliant until the certificate expires, and then it has to be enrolled again. This guide uses Fleet's agent, running next to Intune, to find the devices where renewal is failing or about to fail. It then corrects the most common cause, a stored UPN with an old domain, and confirms the new certificate arrived. It covers Windows hosts enrolled in Intune with fleetd installed. Fleet's own MDM doesn't need to be on. A short section at the end covers hosts where Fleet is the MDM.

For the background on why renewal fails silently, read [Intune moved certificate renewal to the cloud. The failures are still on your devices.](https://fleetdm.com/articles/intune-moved-certificate-renewal-to-the-cloud)

## Prerequisites

- Windows hosts enrolled in Intune with [Fleet's agent installed](https://fleetdm.com/guides/enroll-hosts). Scripts must be enabled on the agent for Step 5. See the [scripts guide](https://fleetdm.com/guides/scripts) if you deployed it without `--enable-scripts`.
- Fleet 4.90.0 or later to see the certificate in host vitals. The queries in this guide work on earlier versions.
- The domain, or domains, that your users' UPNs should carry today. For example, `contoso.com`.
- Fleet Premium for the run-script policy automation in Step 5. Steps 1 through 4 work on Fleet Free.

> **Note:** Fleet reads the device. It can't make Intune's enrollment service accept a renewal request. What you get is the list of devices that need attention, a fix for the stored UPN, and a way to confirm the renewal went through.

## Step 1: Find the certificate and its expiry

For a single host, navigate to **Hosts**, select the host, and scroll to **Certificates** on the **Details** tab. The Intune certificate is the one issued by Microsoft Intune MDM Device CA. Fleet shows its expiry and turns the indicator yellow inside 30 days.

For the whole fleet, navigate to **Reports**, select **Add report**, and run this query against your Windows hosts:

```
SELECT
  common_name,
  store_location,
  datetime(not_valid_after, 'unixepoch') AS expires_utc,
  CAST((not_valid_after - strftime('%s', 'now')) / 86400 AS INTEGER) AS days_left
FROM certificates
WHERE store = 'Personal'
  AND issuer2 LIKE '%Microsoft Intune MDM Device CA%';
```

Sort by `days_left`. Microsoft recommends a renewal window of 40 to 60 days, so a device inside that window that still shows its original expiry is one to check in Step 2.

> **Note:** On Entra-joined and Autopilot-enrolled devices the certificate is in the local machine's personal store, and `store_location` reads `LocalMachine`. If a host returns no row, the certificate may already be gone. Check the registry record in Step 2 and see [Troubleshoot](#troubleshoot).

## Step 2: Read the renewal state from the enrollment record

Windows keeps one key per enrollment under `HKLM\SOFTWARE\Microsoft\Enrollments`. The Intune enrollment is the one whose `ProviderID` is `MS DM Server`. This query collapses each enrollment into a single row and shows the values that matter for renewal:

```
WITH enrollment AS (
  SELECT
    key,
    MAX(CASE WHEN name = 'ProviderID' THEN data END) AS provider_id,
    MAX(CASE WHEN name = 'UPN' THEN data END) AS upn,
    MAX(CASE WHEN name = 'EnrollmentState' THEN data END) AS enrollment_state,
    MAX(CASE WHEN name = 'RenewStatus' THEN data END) AS renew_status,
    MAX(CASE WHEN name = 'RenewErrorCode' THEN data END) AS renew_error,
    MAX(CASE WHEN name = 'DMPCertThumbprint' THEN data END) AS cert_thumbprint
  FROM registry
  WHERE path LIKE 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Enrollments\%%'
  GROUP BY key
)
SELECT
  upn,
  LOWER(SUBSTR(upn, INSTR(upn, '@') + 1)) AS upn_domain,
  enrollment_state,
  renew_status,
  printf('0x%08X', CAST(renew_error AS INTEGER) & 4294967295) AS renew_error_hex,
  cert_thumbprint
FROM enrollment
WHERE provider_id = 'MS DM Server';
```

How to read the result:

- `upn_domain` is the domain Windows will send with the renewal request. If it isn't one your tenant still owns, the renewal will be rejected. This is the case Step 5 fixes.
- `renew_status` follows the CertificateStore CSP: `0` not started, `1` in progress, `2` succeeded, `3` failed.
- `renew_error_hex` is the last error from a renewal attempt. `0x00000000` means the last attempt didn't fail. `0x80180018` is the user license error that a stale UPN produces. `0x80180002` is an authentication failure. `0x80180016` means the service decided the certificate wasn't eligible to renew.
- `enrollment_state` is `1` for a healthy enrollment and `3` when Windows has recorded an enrollment failure.
- `cert_thumbprint` matches the `sha1` column of the `certificates` table, so you can join the two queries if you want expiry and renewal state in one row.

> **Note:** Hosts that have used Fleet's `mdm_bridge` table carry a second enrollment whose `ProviderID` is `Local_Management`. The `WHERE` clause above excludes it.

## Step 3: Pull the renewal events

The enrollment client logs every renewal attempt. This query returns the three event IDs that explain a failure:

```
SELECT datetime, eventid, data
FROM windows_eventlog
WHERE channel = 'Microsoft-Windows-DeviceManagement-Enterprise-Diagnostics-Provider/Admin'
  AND eventid IN (52, 3000, 3006)
ORDER BY datetime DESC
LIMIT 20;
```

- **Event 52** with `UserLicense` in the fault and the reason `Failed to issue token: UserValidation` is the stale UPN case. The device logs it on every attempt.
- **Event 3000** says the current time isn't yet past the expected renewal attempt time. On its own it's normal.
- **Event 3006** says the current time is earlier than the last renew time plus the wait period. Seeing it repeatedly on a device whose window has been open for weeks, with `renew_error_hex` still at zero, is the signature of the key storage provider bug covered in [Troubleshoot](#troubleshoot).

The `data` column is JSON. If you don't see event 52 in the Admin channel on your builds, change the channel name's suffix to `/Operational` and run the query again.

## Step 4: Turn the checks into policies

A policy passes when its query returns at least one row. Navigate to **Policies**, select **Add policy**, and create these three with the platform set to Windows. Replace the domain list in the first one with your own.

**Intune enrollment UPN uses a current domain**

```
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM (
    SELECT
      key,
      MAX(CASE WHEN name = 'ProviderID' THEN data END) AS provider_id,
      MAX(CASE WHEN name = 'UPN' THEN data END) AS upn
    FROM registry
    WHERE path LIKE 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Enrollments\%%'
    GROUP BY key
  )
  WHERE provider_id = 'MS DM Server'
    AND LOWER(SUBSTR(upn, INSTR(upn, '@') + 1)) NOT IN ('contoso.com', 'contoso.co.uk')
);
```

**Intune MDM certificate renewal has not failed**

```
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM registry
  WHERE path LIKE 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Enrollments\%%'
    AND name = 'RenewErrorCode'
    AND CAST(data AS INTEGER) != 0
);
```

**Intune MDM certificate is valid for more than 30 days**

```
SELECT 1 FROM certificates
WHERE store = 'Personal'
  AND issuer2 LIKE '%Microsoft Intune MDM Device CA%'
  AND not_valid_after > strftime('%s', 'now') + (30 * 86400);
```

Failing hosts appear on each policy's page. To get notified instead of checking, select **Manage automations** on the **Policies** page and turn on a webhook or ticket integration for these policies. See the [automations guide](https://fleetdm.com/guides/automations) for the options.

If you manage Fleet with GitOps, the same policies go in your fleet's YAML:

```yaml
policies:
  - name: Intune enrollment UPN uses a current domain
    platform: windows
    description: The UPN stored in the Intune enrollment record uses a domain the tenant no longer owns, so certificate renewal will be rejected.
    resolution: Correct the stored UPN. The run-script automation on this policy does it automatically.
    query: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM (SELECT key, MAX(CASE WHEN name = 'ProviderID' THEN data END) AS provider_id, MAX(CASE WHEN name = 'UPN' THEN data END) AS upn FROM registry WHERE path LIKE 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Enrollments\\%%' GROUP BY key) WHERE provider_id = 'MS DM Server' AND LOWER(SUBSTR(upn, INSTR(upn, '@') + 1)) NOT IN ('contoso.com'));"
    run_script:
      path: ../lib/scripts/fix-intune-enrollment-upn.ps1
```

## Step 5: Correct a stale UPN automatically

When the stored UPN carries a domain you've retired, the next renewal attempt succeeds once the value is corrected. This script rewrites the domain, keeps the original value in the same registry key for reference, and asks the enrollment client to try again. Fleet runs Windows scripts as SYSTEM, which is enough to write under `HKLM`.

> **Warning:** The script keeps the part of the UPN before the `@` and replaces only the domain. If users also changed their sign-in names during a migration (for example `jdoe@old.example` became `john.doe@new.example`), the rewritten value will still be wrong and the renewal will still fail. Scope the policy with a [label](https://fleetdm.com/guides/managing-labels-in-fleet) to the hosts where a domain swap is the whole change, and test on one host before you automate it.

Save this as `fix-intune-enrollment-upn.ps1` and set `$TargetDomain` before you upload it:

```powershell
# Rewrites the domain of the UPN stored in the Intune enrollment record.
# Set $TargetDomain to the domain your users' UPNs carry today.
$TargetDomain = 'contoso.com'

$enrollmentsPath = 'HKLM:\SOFTWARE\Microsoft\Enrollments'
$changed = 0

foreach ($enrollment in Get-ChildItem -Path $enrollmentsPath -ErrorAction SilentlyContinue) {
    $record = Get-ItemProperty -Path $enrollment.PSPath -ErrorAction SilentlyContinue
    if ($record.ProviderID -ne 'MS DM Server') { continue }
    if ([string]::IsNullOrWhiteSpace($record.UPN) -or $record.UPN -notlike '*@*') { continue }

    $localPart, $currentDomain = $record.UPN -split '@', 2
    if ($currentDomain -ieq $TargetDomain) { continue }

    $newUpn = "$localPart@$TargetDomain"

    if (-not $record.PSObject.Properties['UPN_PreviousValue']) {
        New-ItemProperty -Path $enrollment.PSPath -Name 'UPN_PreviousValue' -Value $record.UPN -PropertyType String -Force | Out-Null
        New-ItemProperty -Path $enrollment.PSPath -Name 'UPN_ChangedOn' -Value (Get-Date -Format 'o') -PropertyType String -Force | Out-Null
    }

    Set-ItemProperty -Path $enrollment.PSPath -Name 'UPN' -Value $newUpn
    Write-Output "$($enrollment.PSChildName): $($record.UPN) -> $newUpn"
    $changed++

    # Trigger the renewal task now instead of waiting for its next scheduled run.
    $taskPath = "\Microsoft\Windows\EnterpriseMgmt\$($enrollment.PSChildName)\"
    $task = Get-ScheduledTask -TaskPath $taskPath -ErrorAction SilentlyContinue |
        Where-Object { $_.TaskName -like '*renewal of certificate*' }
    if ($task) { Start-ScheduledTask -InputObject $task }
}

if ($changed -eq 0) { Write-Output 'No enrollment record needed a change.' }
exit 0
```

To attach it to the policy from Step 4:

1. Navigate to **Controls** > **Scripts**, select the fleet, and upload `fix-intune-enrollment-upn.ps1`.
2. Navigate to **Policies**, select the same fleet, and select **Manage automations**.
3. Select **Run script**, check **Intune enrollment UPN uses a current domain**, choose the script, and select **Save**.

Fleet runs the script the first time a host fails the policy and retries up to three times if it exits non-zero. The host passes the policy on its next report once the stored UPN matches.

> **Note:** On current Windows builds the scheduled task hands the request to the enrollment engine, and Intune's service decides when the renewal happens. If nothing changes within a day of the fix, leave the host alone and check again after the next scheduled attempt. The default retry interval is seven days.

## Verify

1. Run the Step 2 query again. The host should show `renew_status` of `2` and `renew_error_hex` of `0x00000000`.
2. Run the Step 1 query. The expiry should have moved about a year out, and `cert_thumbprint` from Step 2 should match the new certificate's `sha1`.
3. Open the host in Fleet, select **Refetch**, and confirm the certificate under **Certificates** is green.
4. In the Intune admin center, confirm the **Management certificate expiration date** column for the device updates after its next sync.

## Troubleshoot

**Event 52 keeps logging a UserLicense fault after the UPN was corrected**

The rewritten UPN has to match an active, licensed user in Entra exactly. Check the user's current UPN in the Entra admin center, their Intune license, and that they're in the MDM user scope. If the sign-in name changed along with the domain, correct the stored value by hand on that host.

**Event 3006 "skip renew" repeats with a zero error code and the window long open**

This is the behavior of the renewal bug on Windows 11 23H2, where a third-party key storage provider made the older renewal path fail before it sent a request. List the providers with a Fleet script that runs `certutil -csplist`, and look for anything not published by Microsoft. The fix shipped in the November 2025 cumulative updates. KB5068865 for 23H2 was reported to renew even expired certificates after a reboot. KB5068861 for 24H2 and 25H2 carries the same fix, but Microsoft staged its rollout. Install the update, reboot, and rerun the Step 2 query.

**The certificate has already expired**

Microsoft's [renewal documentation](https://learn.microsoft.com/en-us/windows/client-management/certificate-renewal-windows-mdm) says automatic renewal doesn't run once the certificate has expired, though the 23H2 update above was reported to recover expired certificates on affected devices. Otherwise the device needs to enroll again. Fleet's agent still has a working channel to the host, so you can run a re-enrollment script from Fleet rather than waiting for a ticket. Test the procedure on one host first. The community has documented [re-enrollment steps that avoid a wipe](https://admin365.blog/2026/06/17/intune-re-enrollment-fix-broken-enrollments-without-a-wipe/).

**A host returns a second enrollment with no UPN**

That's the `Local_Management` provider Fleet's `mdm_bridge` table registers. It isn't an MDM enrollment and doesn't renew. Filter on `ProviderID` as the queries above do.

**Hosts enrolled in Fleet's MDM fail the renewal-error policy**

See the next section.

## If Fleet is your MDM

The same queries work on hosts where Fleet is the MDM. Their enrollment record has a `ProviderID` of `Fleet`, and the certificate's issuer is the one you generated when you turned on Windows MDM, which the [setup guide](https://fleetdm.com/guides/windows-mdm-setup) suggests naming `Fleet Root CA`.

Fleet issues the certificate for 365 days and tells Windows to open the renewal window 180 days before expiry. Fleet doesn't yet complete the renewal itself. That work is tracked in [issue 52492](https://github.com/fleetdm/fleet/issues/52492). Until it ships, a Fleet-enrolled host will show `renew_status` of `3`, a `renew_error_hex` of `0x801900C8`, and an `enrollment_state` of `3` once its window opens. That's Fleet declining the request, not a problem with the host, and the host stays managed. Exclude Fleet-enrolled hosts from the second policy in Step 4 by adding `AND provider_id != 'Fleet'` to its inner query, or scope the policy with a label.

## Further reading

- [View certificates in host vitals](https://fleetdm.com/guides/view-certificates-in-host-vitals)
- [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script)
- [Policy automations](https://fleetdm.com/guides/automations)
- [Windows MDM setup](https://fleetdm.com/guides/windows-mdm-setup)
- Rudy Ooms, [Intune MDM certificate renewal failure: the hidden UPN issue](https://patchmypc.com/blog/intune-mdm-device-certificate-renewal-upn-issue/)
- Microsoft Learn, [CertificateStore CSP](https://learn.microsoft.com/en-us/windows/client-management/mdm/certificatestore-csp)

<meta name="articleTitle" value="Find Windows devices that can't renew their Intune MDM certificate">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-26">
<meta name="description" value="Use Fleet's agent next to Intune to find Windows devices whose MDM certificate renewal is failing, fix a stale UPN, and confirm the new certificate.">
