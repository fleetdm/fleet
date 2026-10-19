# Which apps hold Global Administrator in your tenant? ConnectWise says it can't tell you

*ConnectWise pulled three services offline after a Microsoft 365 secret was abused. Its own advisory tells admins to check their own environments, which is the right instinct and a useful template.*

## Key takeaways

- **A vendor's all clear can't cover your tenant.** ConnectWise has said it cannot see into MSP-managed Microsoft 365 tenants, so whether an abused secret touched your environment is a question only you can answer.
- **One secret was enough.** A compromised Microsoft 365 API secret was used to grant a connected application Global Administrator rights, which turns an app registration into a tenant-wide risk.
- **Start with the grants, not the advisory.** Reviewing which applications hold privileged roles, and when those roles were granted, tells you more than waiting for indicators of compromise.
- **Evidence has to be collected before it is cleaned up.** Rotating secrets and removing grants is necessary, and it can also erase the trail you need, so capture logs and role assignments first.
- **Device data is the other half of the picture.** Tenant audit logs show what an app did in Microsoft 365, while endpoint inventory shows which machines run the affected vendor software.

<a purpose="cta-button" href="https://fleetdm.com/contact">Talk to Fleet about verifiable device data</a>

ConnectWise took its Cloud Backup, SaaS Security, and Cloud Management services offline after a Microsoft 365 API secret was compromised and used to give a connected application Global Administrator rights. The company has also said it cannot see into MSP-managed tenants, which means customers and partners have to look for signs of compromise themselves.

That statement is unusually plain, and it describes a limit every vendor has. Nobody outside your tenant can certify what happened inside it. The rest of this piece covers what that means in practice.

## Why a connected app with Global Administrator is a tenant-wide problem

Global Administrator is the most privileged role in a Microsoft 365 tenant. An application holding it can act with broad authority, and a secret is all it needs to authenticate. When a vendor's secret leaks, every tenant that granted the vendor's app that access inherits the exposure.

This is why the incident reads differently from a typical vendor outage. The services going offline is the visible part. The open question for each customer is whether the abused access reached their tenant.

## Check your own tenant first

Because ConnectWise can't answer that for you, the first pass is yours to run:

- List the enterprise applications and app registrations in your tenant, and note which hold privileged directory roles.
- Look at when each privileged grant was made and by whom, and flag any you don't recognize.
- Review sign-in and audit logs for those applications, focusing on the days around the incident.

Microsoft's own tooling does this work. None of it depends on a vendor telling you the coast is clear.

## Collect evidence before you clean up

The instinct after an incident is to rotate secrets and strip permissions immediately. Do that, but export first. Audit logs have retention limits, and removing a grant removes the clearest record of it. Capture role assignments, app permissions, and the relevant log windows before changing anything, so you can answer questions later.

## Where endpoint data fits

Tenant checks answer what an application did in Microsoft 365. They don't tell you which of your machines run a vendor's agent or software, or what version. Fleet's agent reports installed software across macOS, Windows, and Linux hosts, so you can scope which devices are involved without relying on a vendor's customer list. Fleet doesn't audit Microsoft 365 roles itself. It supplies the device-side facts that a tenant review can't.

## The through-line

An advisory is one party's account of an incident. Your tenant's role assignments and logs, and your devices' own inventory, are records you can read yourself. Build the habit of checking those first, and the next vendor all clear becomes one input among several.

## See it live

- **Get a demo** of how Fleet reports on software across your devices: [fleetdm.com/contact](https://fleetdm.com/contact)

<meta name="category" value="industry news">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="publishedOn" value="2026-10-19">
<meta name="articleTitle" value="Which apps hold Global Administrator in your tenant? ConnectWise says it can't tell you">
<meta name="description" value="ConnectWise says it can't see inside MSP tenants after a Microsoft 365 secret was abused. Here is how to check your own tenant and devices first.">
