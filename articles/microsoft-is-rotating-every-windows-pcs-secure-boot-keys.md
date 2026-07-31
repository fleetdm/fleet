# Microsoft is rotating every Windows PC's Secure Boot keys. Is your fleet ready?

*Every Windows PC built since 2012 carries the same Secure Boot certificates baked into its firmware, and they start expiring in June 2026. Here's what's quietly happening underneath, and how to see exactly where your devices stand.*

## Key takeaways

- **The deadline is real, but nothing breaks on day one.** Three 2011-era certificates begin expiring in June 2026. Affected devices keep booting and running normally; what they lose is the ability to receive new signed boot updates, dbx revocations of vulnerable components, and anti-bootkit lists.
- **Microsoft's rollout is gradual and silent.** Updates arrive per hardware "bucket" based on telemetry confidence, so plenty of devices sit in limbo with no error, no progress, and no obvious signal that anything is waiting.
- **Three failure modes stall the rollout for good.** Secure Boot turned off, an OEM that never shipped the required key update, and known blocking firmware issues each leave a device stuck in a state the automatic process can't resolve on its own.
- **One query tells you where every device stands.** The extension returns a derived state and a needs-action flag per host, so you can triage a whole fleet with a single grouped query instead of reading registry keys and event logs by hand.
- **You have three ways to push it through.** Microsoft Intune's Secure Boot certificate settings, a registry value, or a configuration profile (CSP). The profile is the cleanest option for a managed fleet.
- **Looking now beats scrambling in June.** Deploy the extension today and you can count blocked devices, group them by bucket, and validate a fix on one machine before rolling it out to the rest.

Grab the [secure boot certificate extension](https://github.com/allenhouchins/fleet-extensions/tree/main/secureboot_cert_update).

<a purpose="cta-button" href="/articles/deploying-custom-osquery-extensions-in-fleet">Deploy the extension</a>

If you manage Windows devices, this one is going to sneak up on you. Per Microsoft's guidance, an affected device keeps starting normally, Windows keeps installing most updates, and everyday app use, networking, and browsing stay unchanged, so there's no alarm to trip.

What you quietly lose is the ability to receive new signed updates, new boot manager versions, new dbx revocations of vulnerable boot components, and new Defender anti-bootkit lists. That's a security gap that widens in the background. Here's what's actually expiring.

## What's expiring, and what's replacing it

Secure Boot's trust chain on a Windows PC is anchored by three certificates Microsoft issued around 2011:

- **Microsoft Corporation KEK CA 2011**: the Key Exchange Key that signs updates to the firmware's allowed (`db`) and revoked (`dbx`) signature databases. Expires June 24, 2026.
- **Microsoft Corporation UEFI CA 2011**: signs third-party UEFI components, including Linux shim binaries. Expires June 27, 2026.
- **Windows Production PCA 2011**: signs the Windows boot manager itself. Expires October 19, 2026.

The replacement is a new "2023" certificate family: KEK CA 2023, Windows UEFI CA 2023, and Microsoft UEFI CA 2023. Microsoft has been delivering these via Windows Update since the January 13, 2026, cumulative update.

## How the rollout works

The finer details of the mechanism aren't essential here (if you want to go deeper, message me on LinkedIn). The catch is that Microsoft hasn't turned the rollout on for every device at once. Each device reports a `BucketId`, a hash of its firmware and hardware identity, to Windows Update telemetry. Once enough devices in a given bucket have updated cleanly, all devices matching it get promoted to "high confidence" and receive the update automatically. Until then, the device sits at `Confidence: Under Observation - More Data Needed` indefinitely.

For mainstream hardware this works well, and those buckets get promoted quickly. For everything else it's less predictable. It was a cheap GMKtec I had lying around that led me further down this wormhole.

There are also three failure modes that admins need to see at a glance:

- **Secure Boot is disabled.** The rollout can't run on a device with Secure Boot turned off. Until an admin enables it in UEFI, the device makes no progress.
- **The OEM never shipped a PK-signed KEK update.** Some firmware vendors haven't provisioned the slot Windows needs to write the new KEK 2023 into. The device throws Event 1803 and waits for an OEM firmware update that may never arrive.
- **A known firmware issue is blocking the update.** Microsoft identifies these as `KI_<number>` codes and blocks the rollout on affected firmware versions. The device throws Event 1802.

For a managed fleet, none of these resolve on their own. You need to see them, count them, and decide whether to apply the manual override or chase the OEM for firmware.

## Surfacing all of this in Fleet

The `secureboot_cert_update` osquery extension ([grab it on GitHub](https://github.com/allenhouchins/fleet-extensions/tree/main/secureboot_cert_update)) produces one row per device with a derived state field that does the triage work for you. The full schema is in the [README](https://github.com/allenhouchins/fleet-extensions/blob/main/secureboot_cert_update/README.md), but the most useful columns are:

```
state              -- Updated, InProgress, RebootPending,
                   -- WaitingOnRollout, SecureBootDisabled,
                   -- BlockedOEMMissingKEK, BlockedKnownIssue,
                   -- BlockedFirmwareError, ServicingBroken,
                   -- OptedOut, Unknown
state_reason       -- Short human-readable explanation
needs_action       -- 1 if an admin should look at this device
action             -- none, wait, reboot, enable_secure_boot,
                   -- apply_manual_override, oem_firmware_required,
                   -- investigate
days_until_cert_expiry
```

Underneath that, the table preserves the raw registry and event-log signals (`uefica2023_status`, `available_updates`, `bucket_id`, `confidence`, `known_issue_id`, and more) so you can drill into forensics when the derived state isn't enough.

### Some queries worth running

The first query worth running is the fleet health summary:

```
SELECT state, COUNT(*) AS hosts
FROM secureboot_cert_update
GROUP BY state
ORDER BY hosts DESC;
```

This tells you in one shot what you're dealing with. In most environments you'll see a big block of `Updated`, a long tail of `WaitingOnRollout`, and a smaller set of states where `needs_action = 1`.

For triage, this is the next one to run:

```
SELECT
  hostname, state, state_reason, action, oem_manufacturer_name, oem_model_number, firmware_version
FROM secureboot_cert_update
WHERE needs_action = 1
ORDER BY state, oem_manufacturer_name;
```

A `bucket_id` with dozens of devices behind it is a group that will move together. Once you've validated the manual override on one of them, you can deploy it to the whole bucket with confidence.

This is by no means exhaustive. Run it across your devices, see what comes back, and check the README for more examples.

## Great, how do I fix this?

Microsoft provides a few ways to handle the upgrade. The first is Microsoft Intune, via the Enable Secure Boot Certificate Updates settings, which also offers a few other options. You can also modify the registry directly, under `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\SecureBoot`: set the `AvailableUpdates` DWORD to `0x5944`, then watch `UEFICA2023Status` and `UEFICA2023Error` to confirm the device is making progress. (The extension surfaces this too.)

The way I'd recommend, though, is a CSP. There's an [example profile](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/configuration-profiles/secureboot-update.xml) in the Fleet repo. Setting the CSP value to `22852` (decimal for `0x5944`) writes `AvailableUpdatesPolicy` in the registry. The next time the `Microsoft\Windows\PI\Secure-Boot-Update` scheduled task runs (roughly every 12 hours), Windows copies that policy value into the active `AvailableUpdates` and starts applying the stages in order.

## What's next?

If you've never thought about Secure Boot certificate expiration before, welcome to the party. Deploy the extension with this [guide](https://fleetdm.com/articles/deploying-custom-osquery-extensions-in-fleet), then start querying your devices to understand where you need to focus as we march toward June.

*See where your fleet stands before the deadline forces the question: deploy the extension, run the fleet health summary, and start with the devices that need a nudge.*

<meta name="articleTitle" value="Microsoft is rotating every Windows PC's Secure Boot keys. Is your fleet ready?">
<meta name="authorFullName" value="Harry Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-05-22">
<meta name="description" value="Microsoft's Secure Boot certs start expiring June 2026. See which Windows devices in your fleet are ready, and which need a nudge.">
