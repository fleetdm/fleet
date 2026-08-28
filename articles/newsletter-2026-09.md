# Fleet newsletter, September 2026

Define a custom host vital once, then use it on every platform Fleet manages, from macOS to Android. Fleet 4.90.0 also opens up every Apple declarative profile and asset, and brings custom BitLocker profiles to Windows. September's release lands on the 11th.

## 🚀 What shipped last month

- **Custom host vitals for every platform.** Define your own vitals, like an asset tag or warranty expiration, across macOS, Windows, Linux, iOS, iPadOS, and Android. They show up on **Host details**, and you can build labels from them or use them as variables in scripts and configuration profiles, so values from another system drive automation everywhere in your fleet.
- **Support for all Apple declarative (DDM) profiles and assets.** Upload any DDM configuration or asset, on both the device and user channel, and reference assets from your profiles. When Apple ships a new DDM feature, you can use it on day one instead of waiting for Fleet to add support.
- **Custom BitLocker configuration profiles for Windows.** Upload your own BitLocker profile and customize Windows disk encryption beyond Fleet's built-in controls, matching the flexibility macOS already had. Requires a server configuration change, so it's available to self-managed deployments today.

Also shipped in August: Android OS versions and vulnerabilities in software inventory, renaming macOS, iOS, and iPadOS hosts, editing a configuration profile's labels or contents without deleting it, multiple custom packages under one software title, and Splunk as a log destination. [See every release](https://fleetdm.com/releases).

## 🗺️ What we plan to ship this month

Fleet 4.92.0 is scheduled for September 11. These are planned, not promised, and scope changes as the release comes together.

- **Windows: force standard account.** Keep end users off local administrator accounts on Windows hosts.
- **iOS and iPadOS vulnerabilities.** See vulnerabilities for iOS and iPadOS on **Software > OS**, alongside the Android coverage that landed last month.
- **QR code enrollment for mobile hosts.** Add a QR code to the **Add hosts** modal so people can enroll an Android, iOS, or iPadOS device by scanning it.

Also on deck: custom FileVault and escrow configurations, vulnerability exposure filtering by severity, single sign-on in front of Fleet Desktop's My device page, and more Android host vitals.

Fleet plans releases in the open. The [release planning board](https://github.com/orgs/fleetdm/projects/87/views/10) shows what's queued for the releases after this one.

## 🎓 Upcoming workshops

Fleet workshops are free, run about four hours, and cap at roughly seven people so everyone gets hands-on time. They lead to the Fleet level 1 certificate.

- **Apple administrator, Louisville.** September 8, 1pm to 5pm EDT. [Register](https://www.eventbrite.com/e/apple-administrator-workshop-tickets-1997986140330)
- **GitOps, Washington, DC.** September 10, 1pm to 5pm EDT. [Register](https://www.eventbrite.com/e/gitops-washington-dc-tickets-1992169659078)
- **GitOps speed run, Kansas City.** September 24, 8:30am to 10:30am CDT. [Register](https://www.eventbrite.com/e/speed-run-gitops-jnuc-tickets-1992170088362)
- **GitOps, Gothenburg.** September 28, 1pm to 5pm GMT+2. [Register](https://www.eventbrite.com/e/gitops-macsysadmins-tickets-1993054810590)
- **GitOps, Richmond.** October 13, 1pm to 5pm EDT. [Register](https://www.eventbrite.com/e/gitops-richmond-tickets-1993052268988)

More dates are on the way, and nothing near you yet is worth telling us about. You can [request a workshop](https://fleetdm.com/workshops) in your city.

## 📖 Worth reading

### Customer stories

- [How Reed College gained visibility into 300 unmanaged PCs](https://fleetdm.com/articles/reed-college). Reed brought its unmanaged Windows fleet into view and reclaimed about 20 hours a week.

### Articles

- [Three questions to answer before you renew your MDM](https://fleetdm.com/articles/three-questions-to-answer-before-you-renew-mdm). What to work out about coverage, cost, and lock-in before the renewal conversation starts.
- [What Apple's latest security update shows about patch lag across a fleet](https://fleetdm.com/articles/what-apples-latest-security-update-shows-about-patch-lag-across-a-fleet). How long real fleets take to land an Apple security update, and what that gap costs.
- [What ChatGPT's new Linux desktop app means for shadow AI on developer machines](https://fleetdm.com/articles/chatgpt-linux-desktop-app-shadow-ai). AI tools are arriving on developer laptops whether or not IT put them there.

### Guides

- [Manage Fleet during a GitOps outage](https://fleetdm.com/guides/manage-fleet-during-a-gitops-outage). What to do when GitHub Actions is down and your configuration pipeline stops.
- [Build and validate configuration profiles with AI instead of a GUI](https://fleetdm.com/guides/build-configuration-profiles-with-ai). Generate a profile, check it, and ship it without hunting through a settings screen.
- [Deploy printers with Fleet](https://fleetdm.com/guides/deploy-printers-with-fleet). Printer setup for macOS, Windows, Linux, iOS, iPadOS, and Android, in one place.
- [Add Microsoft Store apps to Windows self-service with winget](https://fleetdm.com/guides/build-your-own-windows-self-service-with-winget-and-script-only-packages-guide). Turn winget commands into a self-service catalog your Windows users can install from.

## 💬 From the community

- [Kitzy](https://www.linkedin.com/in/kitzy/) wrote up [what the extended GitHub Actions outage meant](https://www.linkedin.com/posts/kitzy_todays-extended-github-actions-outage-left-share-7491337261117181952-CEYC) for teams whose device configuration runs through CI, alongside the guide above.
- [Nicklas Holst Hansen](https://www.linkedin.com/in/nicklas-holst-hansen-9982aa268/) shared [a summer intern project at Sopra Steria Norway](https://www.linkedin.com/posts/nicklas-holst-hansen-9982aa268_kan-man-administrere-linux-laptoper-like-ugcPost-7489649285769289728-46Xf) that managed Linux laptops with Fleet, and [Kevin Maksevicius](https://www.linkedin.com/in/kevin-maksevicius-1a892724a/) followed up with [what the team built](https://www.linkedin.com/posts/kevin-maksevicius-1a892724a_this-summer-i-was-very-lucky-to-lead-my-own-share-7493951654325633024-5TKU).
- A practitioner comparing Jamf, Addigy, and Fleet [wrote about how Fleet handles GitOps and AI](https://www.linkedin.com/feed/update/urn:li:share:7498702644194983936/).

Thanks to everyone who shared what they are building.

<meta name="articleTitle" value="Fleet newsletter, September 2026">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-09-01">
<meta name="category" value="newsletters">
<meta name="description" value="What shipped in Fleet 4.90.0, what's planned for 4.92.0, upcoming Apple and GitOps workshops, and August's best guides and customer stories.">
<meta name="newsletterIssue" value="2026-09">
<meta name="coversPeriod" value="2026-08">
