# ConnectWise SkyKick breach: How to check your Microsoft 365 tenant for rogue Global Administrator apps

*ConnectWise pulled three services offline after a Microsoft 365 secret was abused. Its own advisory tells admins to check their own environments, which is the right instinct and a useful template.*

## Key takeaways

- **A vendor's all clear can't cover your tenant.** ConnectWise has said it cannot see into MSP-managed Microsoft 365 tenants, so whether an abused secret touched your environment is a question only you can answer.
- **One secret was enough.** A stolen credential for the Cloud Backup app, formerly SkyKick, let an attacker request Microsoft Graph tokens with no user signed in. In one tenant, Huntress saw a second ConnectWise app add itself to Global Administrator, which turns an app registration into a tenant-wide risk.
- **Start with the grants, not the advisory.** Reviewing which applications hold privileged roles, and when those roles were granted, tells you more than waiting for indicators of compromise.
- **Evidence has to be collected before it is cleaned up.** Rotating secrets and removing grants is necessary, and it can also erase the trail you need, so capture logs and role assignments first. Entra keeps sign-in and audit logs for 7 days on the free tier and 30 days on P1 or P2.
- **Device data is the other half of the picture.** Tenant audit logs show what an app did in Microsoft 365, while endpoint inventory shows which machines run the affected vendor software.

<a purpose="cta-button" href="https://fleetdm.com/contact">Talk to Fleet about verifiable device data</a>

On October 17, 2026, ConnectWise took its Cloud Backup, SaaS Security, and Cloud Management services offline and disabled seven of its Microsoft 365 applications. A stolen Cloud Backup credential had been used to sign in to customer tenants. Huntress counted close to 1,000 Entra tenants with sign-ins from the attacker's IP address, and more than 300 succeeded. ConnectWise has also said that, by design, Cloud Backup "does not provide us visibility or access to downstream partner and customer environments," which means customers and partners have to look for signs of compromise themselves.

That statement is unusually plain, and it describes a limit every vendor has. Nobody outside your tenant can certify what happened inside it. The rest of this piece covers what that means in practice.

## Why a connected app with Global Administrator is a tenant-wide problem

Global Administrator is the most privileged role in a Microsoft 365 tenant. An application holding it can act with broad authority, and a secret is all it needs to authenticate. When a vendor's secret leaks, every tenant that granted the vendor's app that access inherits the exposure. In the tenant Huntress investigated, the app that made itself Global Administrator then added new secrets to two other applications. ConnectWise notes that disabling its apps "does not reverse changes previously made inside a tenant," so those secrets stay until you remove them.

This is why the incident reads differently from a typical vendor outage. The services going offline is the visible part. The open question for each customer is whether the abused access reached their tenant.

## Check your own tenant first

Because ConnectWise can't answer that for you, the first pass is yours to run:

- List the enterprise applications and app registrations in your tenant, and note which hold privileged directory roles.
- Look at when each privileged grant was made and by whom, and flag any you don't recognize. Check client secrets and certificates on every app for anything added since October 13.
- Review sign-in and audit logs for those applications from October 13 onward, including service principal sign-ins. ConnectWise lists 144.172.64[.]128 and the `Python-urllib/3.12` and `python-requests/2.31` user agents as indicators.

Microsoft's own tooling does this work. None of it depends on a vendor telling you the coast is clear.

## Collect evidence before you clean up

The instinct after an incident is to rotate secrets and strip permissions immediately. Do that, but export first. Audit logs have retention limits, and removing a grant removes the clearest record of it. Capture role assignments, app permissions, and the relevant log windows before changing anything, so you can answer questions later.

## Where endpoint data fits

Tenant checks answer what an application did in Microsoft 365. They don't tell you which of your machines run a vendor's agent or software, or what version. Fleet doesn't audit Microsoft 365 roles itself. It supplies the device-side facts that a tenant review can't:

- **Scope vendor software.** ConnectWise says it's also reviewing other partner-facing services. Search Fleet's [software inventory](https://fleetdm.com/guides/software-inventory-reference) for "ConnectWise" and "ScreenConnect" to see which macOS, Windows, and Linux hosts run those tools, and which versions, without relying on a vendor's customer list.
- **Review Fleet's own access.** If you use Fleet's [Entra conditional access integration](https://fleetdm.com/guides/entra-conditional-access-integration), Fleet's multi-tenant app is consented in your tenant, so include it in the review above. Inside Fleet, [API-only users](https://fleetdm.com/guides/fleetctl#using-fleetctl-with-an-api-only-user) hold tokens that don't expire. Confirm each one is expected and has the right role.
- **Keep an outside record.** The [activities webhook](https://fleetdm.com/docs/configuration/yaml-files#activities-webhook) posts each new entry in Fleet's activity log to your SIEM, so the record of admin changes lives outside Fleet.

## The through-line

An advisory is one party's account of an incident. Your tenant's role assignments and logs, and your devices' own inventory, are records you can read yourself. Build the habit of checking those first, and the next vendor all clear becomes one input among several.

## See it live

- **Get a demo** of how Fleet reports on software across your devices: [fleetdm.com/contact](https://fleetdm.com/contact)

## Sources

- ConnectWise, [Trust Center advisories: Cloud Backup (formerly known as SkyKick) service interruption](https://www.connectwise.com/company/trust/advisories)
- ConnectWise, [Product and services status: Cloud Backup incident](https://status.connectwise.com/pages/incident/619cf82551fec9053d612f09/6ad263692dc48705ab3da2cb)
- Huntress, [ConnectWise (SkyKick) vendor credentials abused in M365 attacks](https://www.huntress.com/blog/vendor-credential-enables-unauthorized-access)
- The Hacker News, [ConnectWise suspends Cloud Backup after attackers compromise Microsoft 365 tenants](https://thehackernews.com/2026/10/connectwise-suspends-cloud-backup-after.html)

<meta name="category" value="industry news">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="publishedOn" value="2026-10-19">
<meta name="articleTitle" value="ConnectWise SkyKick breach: How to check your Microsoft 365 tenant for rogue Global Administrator apps">
<meta name="description" value="ConnectWise says it can't see inside MSP tenants after a Microsoft 365 secret was abused. Here is how to check your own tenant and devices first.">
