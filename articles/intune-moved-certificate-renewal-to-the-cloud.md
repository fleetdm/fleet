# Intune moved certificate renewal to the cloud. The failures are still on your devices.

*Every Intune-managed Windows PC renews its management certificate about once a year, and the way that happens changed in 2026. The devices that can't renew are the ones Intune reports as healthy.*

## Key takeaways

- **A renewal that fails looks exactly like one that hasn't happened yet.** A device with a stale enrollment record keeps checking in, keeps reporting compliant, and keeps receiving policy right up until the certificate expires. Then it's orphaned, and the fix is re-enrollment.
- **Renewal is no longer a device decision.** Windows used to decide on its own when to renew and send the request itself. On current builds the device prepares a request and Intune's enrollment service decides whether it goes through, so a server-side rejection is a new way to fail.
- **The two documented failures have nothing in common except their silence.** A UPN left over from a domain migration gets rejected as an unlicensed user. A third-party key storage provider made the older flow give up before it sent anything. Both leave the console green.
- **Intune's console shows you the date, not the outcome.** The management certificate expiration column tells you when a certificate ends. It can't tell you the renewal will fail, because that evidence lives in the registry, the event log, and the certificate store of each device.
- **Fleet's agent reads that evidence next to Intune, without replacing it.** One query across every Windows host returns the stored UPN, the last renewal error, and the days left on the certificate. A policy turns that into a list of failing hosts, and a script automation corrects the UPN before the next attempt.
- **If Fleet is your MDM, you own the certificate authority and its lifetime.** The CA is one you generated, with an expiry you chose. Automatic renewal of Fleet's own Windows MDM certificate is still on the roadmap, and this piece says so plainly.

<a purpose="cta-button" href="https://fleetdm.com/guides/find-windows-devices-that-cant-renew-their-intune-mdm-certificate">Run the queries</a>

Rudy Ooms at Patch My PC spent early 2026 pulling apart how a Windows device renews the certificate that ties it to Intune. He published two findings worth your attention. First, the [renewal flow itself changed](https://patchmypc.com/blog/intune-device-certificate-renewal-has-shifted-from-pull-to-push/), moving decisions off the device and into Intune's enrollment service. Second, an [undocumented condition](https://patchmypc.com/blog/intune-mdm-device-certificate-renewal-upn-issue/) was quietly blocking renewal on devices that looked fine from every angle Intune offers.

This piece covers what changed, why the failures stay invisible to the console, and how to find them from the device side while Intune stays in charge. It ends with what the same problem looks like when Fleet is the MDM.

## The certificate that keeps a device enrolled

When a Windows device enrolls in Intune, it receives a client certificate issued by the Microsoft Intune MDM Device CA. The certificate lives in the local machine's personal store, and the device presents it every time it talks to the management service. It's valid for about a year. Lose it, and Intune has no way to recognize the device. Policies stop arriving, apps sit in a loading state, and the only way back is to enroll again.

Renewal doesn't extend the existing certificate. Microsoft's [renewal documentation](https://learn.microsoft.com/en-us/windows/client-management/certificate-renewal-windows-mdm) describes the device signing a request with the old certificate and receiving a brand new one. Two details in that documentation matter for what follows. Microsoft recommends opening the renewal window 40 to 60 days before expiry so the device gets several tries. And once the certificate has expired, automatic renewal doesn't run at all.

The reason this got attention in 2026 is that the intermediate CA behind every one of those certificates reached the end of its own lifetime in the spring. Every device certificate had to be reissued under the replacement, which put a deadline on a mechanism most admins had never needed to look at. Whether or not your devices came through that window cleanly, each of them is at most a year away from its next renewal. The mechanics below are the ones it will use.

## How renewal used to work, and what replaced it

For years the process was self-contained. A scheduled task under the EnterpriseMgmt folder, created at enrollment and named for certificate renewal, ran the enrollment client on a weekly cadence. The client read the enrollment record from the registry and checked whether the renewal window was open. If it was, the client built a renewal request signed with the current certificate and sent it straight to Intune's enrollment endpoint. The service checked that the certificate was within its window, that it had issued the certificate, and that the request came from the same device. Then it returned a new one.

That design had a weakness. The device made every decision from local state. If the registry drifted, if the stored provider was wrong, or if the private key sat somewhere the client didn't expect, the device concluded there was nothing to do. It never asked anyone. Rudy's earlier [write-up of the task and the client](https://call4cloud.nl/intune-device-certificate-renewed-renewal/) documents the checks in detail.

By comparing the enrollment client across Windows 11 23H2 and 24H2, he found that the renewal logic had moved. The client now hands the job to the enrollment engine, a background component that queues the renewal and writes its state as the service responds. Timing and retry markers come back from Intune rather than being computed locally. Alongside that, Microsoft's [CertificateStore CSP reference](https://learn.microsoft.com/en-us/windows/client-management/mdm/certificatestore-csp) gained a RenewNow node whose stated purpose is to start a renewal immediately. In his testing, that trigger bypassed the scheduler and the local timing checks entirely. The scheduled task still exists, but the service is now the one deciding.

Rudy calls this a shift from pull to push, and the label fits. The upside is that a device with drifted local state can no longer talk itself out of renewing. The downside is new: the service can now say no, and the device-side signals you learned to trust mean less than they did.

## Two ways it fails without a sound

### A UPN that no longer exists

When a device enrolls, Windows stores the enrolling user's UPN in the enrollment record under the Enrollments key in the registry. It stays there. Now picture a merger. Users move to a new domain, their UPNs change in Entra, and months later someone removes the old domain from the tenant. The user is active and licensed under the new name. The device still carries the old one.

Under the old flow, nobody looked at that value during renewal. Under the new flow, the enrollment service validates it. On affected devices, Rudy found a renewal error of 0x80180018, which Microsoft's enrollment protocol documents as a problem with the user's license. Alongside it, the DeviceManagement-Enterprise-Diagnostics-Provider logged event 52 with a UserLicense fault and the reason "Failed to issue token: UserValidation". The device logged that on every attempt and kept syncing normally in between. He confirmed the cause by changing nothing but the stored UPN and running the renewal again. It succeeded immediately.

Nothing about this is exotic. Any organization that has renamed a domain, absorbed another company, or cleaned up a tenant has devices in this state. They'll keep working until the certificate runs out.

### A key storage provider that isn't Microsoft's

The second failure predates the UPN finding and hit a different set of devices. On Windows 11 23H2, the older renewal path initialized every registered key storage provider on the system before it built the request, regardless of which one held the MDM certificate's key. A third-party provider that came bundled with digital signature software returned an error during that initialization. The whole request failed before anything left the device, and the client logged the skip as if the renewal window hadn't opened yet.

An MSP described the result in a [Microsoft Q&A thread](https://learn.microsoft.com/en-us/answers/questions/5610335/intune-mdm-certificates-not-renewing-on-windows-de). Dozens of devices across several tenants had expired certificates, every one of them showing as compliant and recently synced. More were expiring every month, and Microsoft's support cases found no cause. Rudy traced it to the provider, and his [post on the bug](https://patchmypc.com/blog/the-intune-mdm-device-certificate-ksp-renewal-bug-why-23h2-devices-stopped-renewing/) explains the fix that shipped in the November 2025 cumulative updates. As reported in that thread, KB5068865 for 23H2 renewed even expired certificates after a reboot. KB5068861 for 24H2 and 25H2 carried the same fix behind a staged rollout that didn't switch on for every device at once.

The two failures share one trait. In both cases Intune's view of the device was wrong in the reassuring direction.

## Why the console can't warn you

Intune does give you a date. Since the 2309 service release, the device list can show a management certificate expiration column, and you can filter by date range and export the result. That's worth using. It answers "when does this certificate end," which is the right question to start with.

It can't answer "will the renewal succeed," because nothing the console displays is derived from the renewal attempt. Last check-in looks fine, since the old certificate still works. Compliance is green, since nothing about the device's configuration changed. The evidence that renewal is failing sits on the device: the UPN and the last renewal error in the enrollment record, the events the enrollment client writes, the certificate's actual expiry in the local store, and the list of registered key storage providers. Reading those on one machine is a remote session. Reading them on five thousand machines, every week, is the problem.

## Fleet next to Intune: read the device, keep the console

Fleet's agent, fleetd, runs on a Windows host whether or not Fleet is the MDM. It has its own enrollment with the Fleet server and its own TLS session, and neither depends on the Intune certificate. That independence is the whole point here. When the MDM channel is in trouble, the agent channel isn't.

Three things follow from that.

**The certificate is already in host vitals.** Fleet 4.90.0 added Windows certificates to the host details page, read from the local machine's personal store. The Intune certificate shows up with its expiry, turns yellow inside 30 days, and turns red once it's gone. The same data is available through the API, so a list of hosts approaching expiry is a call, not a spreadsheet.

**The renewal evidence is a query away.** Fleet's agent exposes the Windows registry, the event log, and the certificate store as tables. A live query returns results from every online host in seconds, and the simplest one is the days left on the Intune certificate itself:

```
SELECT
  common_name,
  datetime(not_valid_after, 'unixepoch') AS expires,
  CAST((not_valid_after - strftime('%s', 'now')) / 86400 AS INTEGER) AS days_left
FROM certificates
WHERE store = 'Personal'
  AND issuer2 LIKE '%Microsoft Intune MDM Device CA%';
```

A second query reads the enrollment record for the stored UPN and the last renewal error, and a third pulls event 52 and the skip events from the provider bug out of the event log. Both are in the companion guide. Save any of them as a scheduled report and the picture stays current without anyone remembering to look.

**Policies turn it into a workflow.** A Fleet policy is a yes-or-no question about device state. "Does the stored UPN use a domain we still own?" is one. "Is the last renewal error zero?" is another. Hosts that fail show up in a list, and policy automations can send a webhook, open a ticket, or run a script. For the UPN case, the script is short: back up the stored value, rewrite the domain, and ask the enrollment client to try again. Fleet runs it as SYSTEM, so the registry write needs no extra plumbing. The policy and the script can both live in Git as YAML, which means the fix is a reviewed pull request and not a change someone remembers making in a console.

Fleet can't make Intune's enrollment service accept a request, and this piece won't pretend otherwise. What Fleet does is make the failure visible weeks before the deadline, put the correction on the device, and keep a channel open to the device after Intune has lost it. That last part matters more than it sounds. When a certificate does expire, the community has worked out [re-enrollment procedures that avoid a wipe](https://admin365.blog/2026/06/17/intune-re-enrollment-fix-broken-enrollments-without-a-wipe/). Running one of those is a Fleet script on a list of hosts you already have, not a help desk ticket from a user whose apps stopped installing. Entra conditional access through Fleet also works without Fleet's MDM turned on, so none of this requires changing who manages the device.

## When Fleet is the MDM

If Fleet is the MDM authority for your Windows hosts, the problem has a different shape, and some of it is simpler.

The certificate authority is yours. Fleet's Windows MDM uses a WSTEP identity certificate and key that you generate, following the [Windows MDM setup guide](https://fleetdm.com/guides/windows-mdm-setup). Its lifetime is a parameter you pass to OpenSSL. There's no vendor-side intermediate reaching end of life on a schedule you didn't set. Client certificates are issued for 365 days, and Fleet tells Windows to open the renewal window 180 days before expiry with a four-day retry.

What Fleet doesn't do yet is service the renewal request. The work is tracked in [issue 52492](https://github.com/fleetdm/fleet/issues/52492), and until it ships a Fleet-enrolled Windows host's MDM certificate is good for a year from enrollment. Once the renewal window opens, the enrollment record will show a failed renewal status with an HTTP-facility error code. That's Fleet declining the request, not a problem with the device, and the host stays managed in the meantime. The same host vitals and the same queries from the previous section show you the expiry date on every Fleet-enrolled host, so nothing about the visibility story changes.

Two more things hold regardless of who issued the certificate. Fleet reads the Enrollments key as part of its standard host detail collection, so MDM enrollment state is on the host page no matter which MDM the device belongs to. And because Fleet's agent turns MDM on programmatically and retries enrollment on its own, a Windows host that loses its MDM session hasn't lost its connection to Fleet.

## The quiet part

An expired certificate is loud. Apps stop installing, policies stop landing, and someone opens a ticket. A failing renewal is silent, and the gap between the two is measured in weeks you could have used.

Intune changed where renewal decisions get made. It didn't change where the evidence lives, and it didn't give the console a way to show it. The devices that will be orphaned next quarter are reporting compliant today. Whether you find them first depends on whether anything in your stack is reading the device instead of the dashboard.

## See it live

- Follow the guide: [Find Windows devices that can't renew their Intune MDM certificate](https://fleetdm.com/guides/find-windows-devices-that-cant-renew-their-intune-mdm-certificate)
- **Get a demo** of Fleet alongside Intune: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Join a GitOps workshop** to see policies and scripts managed as code: [fleetdm.com/workshops](https://fleetdm.com/workshops)

## Sources

- Rudy Ooms, Patch My PC, [Intune device certificate renewal flow has shifted from pull to push](https://patchmypc.com/blog/intune-device-certificate-renewal-has-shifted-from-pull-to-push/)
- Rudy Ooms, Patch My PC, [Intune MDM certificate renewal failure: the hidden UPN issue](https://patchmypc.com/blog/intune-mdm-device-certificate-renewal-upn-issue/)
- Rudy Ooms, Patch My PC, [The Intune MDM device certificate KSP renewal bug](https://patchmypc.com/blog/the-intune-mdm-device-certificate-ksp-renewal-bug-why-23h2-devices-stopped-renewing/)
- Microsoft Learn, [Certificate renewal for Windows MDM](https://learn.microsoft.com/en-us/windows/client-management/certificate-renewal-windows-mdm)
- Microsoft Learn, [CertificateStore CSP](https://learn.microsoft.com/en-us/windows/client-management/mdm/certificatestore-csp)
- Microsoft Q&A, [Intune MDM certificates not renewing on Windows devices](https://learn.microsoft.com/en-us/answers/questions/5610335/intune-mdm-certificates-not-renewing-on-windows-de)

<meta name="articleTitle" value="Intune moved certificate renewal to the cloud. The failures are still on your devices.">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-10-26">
<meta name="description" value="Intune now decides Windows MDM certificate renewals server-side. See why failing devices still look healthy and how Fleet finds them first.">
