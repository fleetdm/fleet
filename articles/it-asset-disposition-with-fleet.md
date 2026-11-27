# How to retire computers securely: IT asset disposition with Fleet

Every computer leaves the company eventually. IT asset disposition (ITAD) is how it leaves safely: the data is gone, the device is out of your systems, and you have proof. Fleet handles the digital side. An ITAD partner handles the physical side.

## What IT asset disposition is

ITAD covers what happens to a computer after its last day of work: pickup and shipping, wiping or destroying the drive, resale or recycling, and a certificate for each device that proves it was done.

The standard most partners follow is [NIST SP 800-88 Rev. 2](https://csrc.nist.gov/pubs/sp/800/88/r2/final), published in September 2025. It defines three ways to sanitize a drive: clear, purge, and destroy. It also allows [cryptographic erase](https://csrc.nist.gov/files/pubs/sp/800/88/r2/final/docs/sp800-88r2-faq.pdf) as a purge method under certain conditions. Destroy the key to an encrypted drive, and the data becomes unreadable.

Auditors care about this. SOC 2, HIPAA, and ISO 27001 reviews ask how you dispose of devices that held company data, and a forgotten drive in a closet is a breach waiting to happen.

## Who does it

**PC makers** offer ITAD alongside new purchases and lease returns:

- [Dell Asset Recovery Services](https://www.dell.com/en-us/lp/dt/ars) takes owned or leased equipment of any brand, including PC as a Service returns. Dell sanitizes data to its own data security policy and provides chain-of-custody reports.
- [HP IT Asset Disposition](https://www.hp.com/us-en/services/workforce-solutions/workforce-computing/it-asset-disposition.html) sends a Certificate of Data Sanitization that cites NIST 800-88. HP takes other brands too, but only alongside a purchase of new HP devices.
- [Lenovo Asset Recovery Services](https://lenovopress.lenovo.com/datasheet/ds0091-asset-recovery-services) provides a certificate of data destruction for each device.

**Apple** runs [Apple Business Trade In](https://www.apple.com/shop/trade-in) for organizations, with a five-device minimum. Apple asks you to erase each device, remove its profiles, and turn off Find My first. The Mac-specific steps below cover that.

**Independent ITAD companies** include [Iron Mountain](https://www.ironmountain.com/services/it-asset-lifecycle-management/secure-it-asset-disposition), [Sims Lifecycle Services](https://www.simslifecycle.com/), [SK tes](https://www.sktes.com/it-services/it-asset-disposition), [ERI](https://eridirect.com/services), and [Ingram Micro](https://na.ingrammicro.com/en-us/services/lifecycle). The market is fragmented. Global Market Insights [puts the leader at about 3%](https://www.gminsights.com/industry-analysis/IT-asset-disposition-market), though its own pages disagree on who that leader is.

**Resellers** like [CDW](https://www.cdw.com/content/cdw/en/services/renewals-and-warranty/it-asset-dispostition.html) and [SHI](https://www.shi.com/it-lifecycle-services/device-lifecycle-management/asset-recovery) also sell ITAD services, so you can often add it to an existing contract.

## Where Fleet fits

Your ITAD partner takes over once the computer is in a box. Fleet handles everything before that: knowing what you're retiring, confirming it's encrypted, locking it, wiping it, and removing it from your systems. Fleet doesn't handle the physical part.

## Before you begin

- Fleet Premium. Lock, wipe, and releasing devices from Apple Business are Premium features.
- MDM turned on for macOS, Windows, iOS, and iPadOS hosts. Linux hosts need [scripts enabled](https://fleetdm.com/docs/using-fleet/scripts).
- A global admin account, to release devices from Apple Business.
- Access to Apple Business, and to Microsoft Intune if your PCs are registered with Windows Autopilot.

## Retire a computer with Fleet

### 1. Know what you're retiring

On the **Hosts** page, search or filter for the devices you're retiring. Select **Edit columns** to show **Serial number**, **Hardware model**, and **User email**, then select **Export hosts**. This CSV is your manifest for the ITAD partner.

### 2. Confirm the disk is encrypted

Check each host's disk encryption status. Fleet [enforces encryption](https://fleetdm.com/guides/enforce-disk-encryption) on macOS (FileVault) and Windows (BitLocker), and escrows keys for macOS, Windows, and Linux (Ubuntu, Kubuntu, Fedora, and Omarchy). Encryption is your safety net if a device goes missing in transit.

### 3. Lock it if it's not back yet

If a departing employee still has the computer, [lock it](https://fleetdm.com/guides/lock-wipe-hosts) from **Actions > Lock** on the **Host details** page. On a Mac, Fleet shows a six-digit PIN you'll need to unlock it.

### 4. Clear the locks that block resale

Do these before you wipe, while you can still reach the device.

- **Mac: Activation Lock.** If Activation Lock is on, nobody can set up the Mac again. Fleet disables it by default on Macs enrolled through Apple Business. If one is on anyway, [turn it off in Apple Business](https://support.apple.com/guide/deployment/activation-lock-depf4ab94ef1/web). Fleet doesn't yet manage Activation Lock bypass codes.
- **Windows: Autopilot.** A PC still registered with Autopilot keeps getting your company's setup. [Deregister it in Intune](https://learn.microsoft.com/en-us/autopilot/registration-overview#deregister-a-device). Fleet reads Autopilot registrations but doesn't yet remove them.
- **Dell, HP, and Lenovo: BIOS passwords.** A forgotten BIOS password can mean a [Dell release code](https://www.dell.com/support/kbdoc/en-us/000131024/how-to-clear-the-bios-password) with proof of ownership, or [a new system board](https://download.lenovo.com/manual/thinkpad_t14g5_t16g3_p14sg5/ug/html_en/en/What_to_do_if_you_forget_your_supervisor_password.html) for a Lenovo. Clear them with your vendor's tools. Fleet doesn't yet manage BIOS passwords ([#59972](https://github.com/fleetdm/fleet/issues/59972)).

### 5. Wipe it

On the **Host details** page, select **Actions > Wipe**. What happens depends on the platform:

| Platform | What Fleet does |
|---|---|
| macOS | Erase All Content and Settings, which [destroys the encryption keys](https://support.apple.com/guide/deployment/dep0a819891e/web) on Macs with Apple silicon or a T2 chip |
| iOS and iPadOS | Erases the device |
| Windows | Sends Microsoft's `doWipeProtected` command. Microsoft warns this [can leave a PC unable to boot](https://learn.microsoft.com/en-us/windows/client-management/mdm/remotewipe-csp#dowipeprotected). |
| Linux | Runs a script that deletes user and system files and halts the computer. It's [not a secure erase](https://fleetdm.com/guides/lock-wipe-hosts#linux-wipe-behavior), so plan to reinstall. |
| Android | Erases company-owned devices. Deletes only the work profile on personal ones. |

> **Note:** A Fleet wipe doesn't replace your ITAD partner's certified sanitization. Ask for a certificate either way.

### 6. Release it and remove it from Fleet

Wait until the wipe finishes, then:

1. **Apple devices:** select **Actions > Release from Apple Business**. This can't be undone. [Turn off Activation Lock first](https://support.apple.com/guide/business/axmec4d28461/web), because Apple Business can't manage it after release.
2. Select **Actions > Delete** to remove the host from Fleet. Deleted hosts no longer count toward the hosts on your license.

> **Warning:** If you delete an Apple device that's still assigned to Fleet in Apple Business, it comes back as a **Pending** host. A computer that still has a valid enroll secret re-enrolls the next time it checks in. Release or wipe first.

Fleet keeps archived disk encryption keys, so a global admin can still [look one up by serial number](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-disk-encryption-key) after the host is deleted.

### 7. Hand it to your ITAD partner

Match the partner's pickup manifest against your Fleet export from step 1. Every serial number should appear on both.

### 8. Close the loop

Store each certificate of sanitization by serial number. Fleet's [audit log](https://fleetdm.com/docs/api/global-audit-logs) records when each device was locked (`locked_host`), wiped (`wiped_host`), released (`released_from_ab`), and deleted (`deleted_host`, with its serial number). Export those with the certificates, and you have the record an auditor will ask for.

## Automate it with the API

Steps 1, 3, 5, and 6 each have an API endpoint. Here's one computer, by serial number:

```sh
FLEET=https://fleet.example.com
AUTH="Authorization: Bearer $FLEET_API_TOKEN"
SERIAL=C02XXXXXXXXX

# Find the host
ID=$(curl -s -H "$AUTH" "$FLEET/api/v1/fleet/hosts/identifier/$SERIAL?exclude_software=true" | jq .host.id)

# Wipe it
curl -s -X POST -H "$AUTH" "$FLEET/api/v1/fleet/hosts/$ID/wipe"

# After the wipe finishes: release it from Apple Business, then delete it
curl -s -X POST -H "$AUTH" -d "{\"ids\": [$ID]}" "$FLEET/api/v1/fleet/hosts/release_ab"
curl -s -X DELETE -H "$AUTH" "$FLEET/api/v1/fleet/hosts/$ID"
```

To export the manifest, call [`GET /api/v1/fleet/hosts/report?format=csv`](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-report-in-csv). From the command line, `fleetctl mdm lock --host` and `fleetctl mdm wipe --host` accept a serial number. GitOps manages settings, not one-time actions like these. The closest setting is [`host_expiry_settings`](https://fleetdm.com/docs/configuration/yaml-files#host-expiry-settings), which removes hosts that stop checking in.

## Checklist

- [ ] Export serial numbers, models, and end users from Fleet
- [ ] Confirm disk encryption and escrowed keys
- [ ] Lock any device that hasn't come back
- [ ] Turn off Activation Lock, deregister from Autopilot, and clear BIOS passwords
- [ ] Wipe the device and wait for it to finish
- [ ] Release Apple devices from Apple Business, then delete the host from Fleet
- [ ] Match the ITAD partner's manifest against your export
- [ ] File each certificate with the Fleet audit log entries, by serial number

## Further reading

- [Lock and wipe hosts](https://fleetdm.com/guides/lock-wipe-hosts)
- [Enforce disk encryption](https://fleetdm.com/guides/enforce-disk-encryption)
- [Release a host from Apple Business](https://fleetdm.com/guides/apple-mdm-setup#release-a-host-from-apple-business)
- [NIST SP 800-88 Rev. 2](https://csrc.nist.gov/pubs/sp/800/88/r2/final)

<meta name="articleTitle" value="How to retire computers securely: IT asset disposition with Fleet">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-11-30">
<meta name="description" value="What IT asset disposition is, who does it, and how to retire computers securely with Fleet and an ITAD partner.">
