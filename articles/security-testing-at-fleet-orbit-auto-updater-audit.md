# Security testing at Fleet/Orbit auto-updater audit

![Security testing at Fleet/Orbit auto-updater audit](../website/assets/images/articles/security-testing-at-fleet-orbit-auto-updater-audit-cover-1600x900@2x.jpg)

At Fleet, openness is one of our core [values](https://fleetdm.com/handbook/company#values). We believe a rising tide lifts all boats and that almost everything we do regarding security should be public.

[Orbit](https://blog.fleetdm.com/introducing-orbit-for-osquery-751da494d617) is an [osquery](https://github.com/osquery/osquery) runtime and auto-updater. It leverages [The Update Framework](https://theupdateframework.io/) to create a secure update mechanism using a hierarchy of cryptographic keys and operations.

About a year ago, while Orbit was still brand new, not “production-ready,” and in use by almost nobody, we had an external vendor ([Trail of Bits](https://www.trailofbits.com/)) perform a [security audit](https://fleetdm.com/docs/using-fleet/security-audits) on the Orbit auto-updater functionality.

We then handled the issues surfaced by the audit publicly in the Fleet repository and the old Orbit repository.

### Testing in the future

Fleet will regularly perform security tests. These tests will target Fleet, Orbit, our company, and many other components.

We will:

1. Resolve issues that expose Fleet users to risk.
2. Share the results of tests as rapidly as possible once we have addressed issues.
3. Comment when necessary and valuable.

If external testers find significant vulnerabilities, we will generate GitHub [security advisories](https://github.com/fleetdm/fleet/security/advisories) on a case-by-case basis. We can share important information about vulnerabilities before releasing the full report.

### Auto-updater security
We believe the security of auto-updates is critical in a world where supply chain attacks are common. When automatic updaters are trusted, systems receive essential security updates quicker. It is up to the software industry to make these updates trustworthy so everyone can benefit from more secure systems around the globe.

We will continue improving our software and processes for packaging and delivering Orbit updates by expanding security mechanisms to cover more and more threat scenarios. You can always peek at our [security project](https://github.com/orgs/fleetdm/projects/33), where public issues are visible to everyone.

Future improvements will appear there, and we are always thankful when researchers discover and [disclose](https://github.com/fleetdm/fleet/security/policy) vulnerabilities.

If you have questions about the Orbit audit or Fleet security, please join us on [Slack](https://osquery.fleetdm.com/c/fleet)!

<meta name="category" value="security">
<meta name="authorGitHubUsername" value="GuillaumeRoss">
<meta name="authorFullName" value="Guillaume Ross">
<meta name="publishedOn" value="2022-03-30">
<meta name="articleTitle" value="Security testing at Fleet/Orbit auto-updater audit">
<meta name="articleImageUrl" value="../website/assets/images/articles/security-testing-at-fleet-orbit-auto-updater-audit-cover-1600x900@2x.jpg">