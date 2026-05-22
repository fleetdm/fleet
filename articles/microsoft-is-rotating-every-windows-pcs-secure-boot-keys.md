# Microsoft is rotating every Windows PC's Secure Boot keys. Is your fleet ready?

Every Windows PC built since 2012 has the same set of Secure Boot certificates baked into its firmware. They start expiring in June 2026. If you manage a fleet, here's what's quietly happening underneath, and how to see where your devices stand. As we head towards the deadline, it's important to note what will continue to work as usual, per Microsoft's documentation:

- The device continues to start normally.  
- Windows updates continue to install, except for boot‑related security components that require the updated certificates.  
- Everyday app use, networking, browsing, and most OS features remain unchanged.

But what you lose matters: the ability to receive new signed updates, new boot manager versions, new dbx revocations of vulnerable boot components 💀, and new Defender anti-bootkit lists. This extension can help you understand the state of the (Windows) union.

Grab the extension here.

### What's expiring, and what's replacing it

Secure Boot's trust chain on a Windows PC is anchored by three certificates Microsoft issued around 2011:

- **Microsoft Corporation KEK CA 2011** — the Key Exchange Key that signs updates to the firmware's allowed (`db`) and revoked (`dbx`) signature databases. Expires **June 24, 2026**.  
- **Microsoft Corporation UEFI CA 2011** — signs third-party UEFI components, including Linux shim binaries. Expires **June 27, 2026**.  
- **Windows Production PCA 2011** — signs the Windows boot manager itself. Expires **October 19, 2026**.

The replacement is a new "2023" certificate family: **KEK CA 2023, Windows UEFI CA 2023**, and **Microsoft UEFI CA 2023**. Microsoft has been delivering these via Windows Update since the January 13, 2026, cumulative update.

### How the rollout works

The finer details of all that's happening aren't super important (if you want to know more, message me on LinkedIn!). But the catch is that Microsoft hasn't turned the rollout on for every device at once. Each device reports a BucketId (a hash of its firmware + hardware identity) to Windows Update telemetry. Microsoft watches the rollout succeed on a given bucket, and once enough devices in that bucket have updated cleanly, all devices matching it get promoted to "high confidence" and receive the update automatically. Until then, the device sits in **Confidence: Under Observation - More Data Needed** indefinitely.

For mainstream hardware, this works well, and those buckets get promoted quickly. For everything else, it's less predictable. It was actually a cheap GMKtec I had lying around that led me down even further into a wormhole here.

There are also three failure modes that admins need to be able to see at a glance:

- **Secure Boot is disabled.** The rollout cannot run on a device with Secure Boot turned off. Until an admin enables it in UEFI, the device makes no progress.  
- **The OEM never shipped a PK-signed KEK update.** Some firmware vendors haven't provisioned the slot Windows needs to write the new KEK 2023 into. The device throws Event 1803 and waits for an OEM firmware update that may never arrive.  
- **A known firmware issue is blocking the update.** Microsoft identifies these as KI_<number> codes and blocks the rollout on affected firmware versions. The device throws Event 1802.

For a managed fleet, all three of these are conditions that the natural rollout can't resolve on its own. You need to see them, count them, and decide whether to apply the manual AvailableUpdates override or chase the OEM for firmware.

### Surfacing all of this in Fleet

The **secureboot_cert_update** osquery extension produces one row per device with a derived state field that does the triage work for you. The full schema is in the [README](https://github.com/allenhouchins/fleet-extensions/blob/main/secureboot_cert_update/README.md), but the most useful columns are:

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

Underneath that, the table preserves the raw registry and event-log signals such as uefica2023_status, available_updates, bucket_id, confidence, known_issue_id, etc. so you can drill into forensics when the derived state isn't enough.

### Some interesting queries

The first query worth running is the fleet health summary:

```
SELECT state, COUNT(*) AS hosts
FROM secureboot_cert_update
GROUP BY state
ORDER BY hosts DESC;
```

This tells you in one shot what you're dealing with. In most environments, you'll see a big block of **Updated**, a long tail of **WaitingOnRollout**, and a smaller set of states with needs_action \= 1.

For triage, this is the work queue:

```
SELECT
  hostname, state, state_reason, action, oem_manufacturer_name,  oem_model_number, firmware_version
FROM secureboot_cert_update
WHERE needs_action = 1
ORDER BY state, oem_manufacturer_name;
```

A bucket_id with dozens of devices behind it is a group that will move together. Once you've validated the manual override on one of them, you can deploy it to the whole bucket with confidence.

This is by no means an exhaustive list of things you can query with this table. I encourage you to run it across your devices and see what data you get back, and have a play. Check out the README in the repo for some more examples.

### Great, how do I fix this?

Microsoft provides a couple of ways to handle this upgrade. The first is Intune via the **Enable Secure Boot Certificate** Updates settings, which also offers a few other options. You can always go the way of modifying registry keys, especially the **HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\SecureBoot** key. Set the AvailableUpdates DWORD to 0x5944. You'll want to monitor the UEFICA2023Status and UEFICA2023Error to see that the devices are making progress. (The osquery extension surfaces this information).

Lastly, and the way that I would recommend, is via a CSP. You can find an [example profile](https://github.com/fleetdm/fleet/blob/main/docs/solutions/windows/configuration-profiles/secureboot-update.xml) here. Setting the CSP value to ***22852** (decimal for 0x5944) writes **AvailableUpdatesPolicy** in the registry. The next time the **Microsoft\Windows\PI\Secure-Boot-Update** scheduled task runs (every ~12 hours), Windows copies that policy value into the active **AvailableUpdates** and starts applying the stages in order.

### What's next?

If you've never thought about Secure Boot certificate expiration before, welcome to the party. You can easily deploy the extension using this [guide](https://fleetdm.com/articles/deploying-custom-osquery-extensions-in-fleet), then get started querying your devices and understanding where you need to focus your efforts as we march towards June.

<meta name="articleTitle" value="Microsoft is rotating every Windows PC's Secure Boot keys. Is your fleet ready?">
<meta name="authorFullName" value="Harry Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-05-22">
<meta name="description" value="Microsoft's Secure Boot certs start expiring June 2026. See which Windows devices in your fleet are ready, and which need a nudge.">
