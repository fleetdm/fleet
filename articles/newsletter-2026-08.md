# Fleet newsletter, August 2026

Pin a Fleet-maintained app to the version you trust, or roll one back whenever you need to. Fleet 4.89.0 also adds Google Workspace as a source of IdP host vitals, and gives end users a clear path through Windows enrollment. Two more releases land this month.

## What shipped last month

- **Auto-update, pin, and roll back Fleet-maintained apps.** Pin an app to a specific version to stop it from auto-updating, or roll back to the previous version if a new release breaks something. Fleet checks for new versions hourly, so hosts you leave on auto-update stay current without you re-adding the app. Available in Fleet Premium.
- **Windows setup experience: continue past a failed install.** When required setup software fails during Windows automatic enrollment, end users see exactly which app failed. If you haven't checked **Cancel setup if software fails**, they can continue and install it later from self-service. Either way they get a next step instead of a stuck screen. Available in Fleet Premium.
- **IdP host vitals from Google Workspace.** Populate group, department, username, email, and full name straight from Google Workspace. Google Workspace doesn't support SCIM, so Fleet pulls directory data from Google's API on a schedule. Scope profiles, software, and policies with IdP labels the same way you would with Okta or Entra. Available in Fleet Premium.

Three of eight highlights. For the rest, plus everything in 4.88.0, 4.89.1, and 4.89.2, see the [Fleet 4.89.0 release notes](https://fleetdm.com/releases/fleet-4-89-0).

## What we plan to ship this month

Two releases are scheduled for August: 4.90.0 and 4.91.0. These are planned, not promised, and scope changes as the releases come together.

- **Custom host vitals.** Define your own host fields, add, edit, and delete them, and use them in host name templates.
- **Full Apple declarative (DDM) profile support.** Support for all Apple declaration profiles and assets, plus custom activations.
- **macOS local accounts from any IdP.** Create the initial local account and sync its password from any identity provider that supports OAuth ROPG, moving out of experimental.
- **Certificate visibility.** View certificates in **Controls > OS settings**, and see certificates on the host details page for Windows hosts.
- **Android software inventory.** Show Android OS versions and their vulnerabilities in **Software > OS**, alongside more Android host vitals.
- **Python script-only packages.** Add Python to the script-only package options for software deployment.

Want the wider view? The [July roadmap preview](https://fleetdm.com/announcements/roadmap-preview-july-2026) covers the next three months, including AI governance, patch policies, Windows local admin account rotation, and tvOS.

## Upcoming workshops

Fleet workshops are free, run about four hours, and cap at roughly seven people so everyone gets hands-on time. They lead to the Fleet level 1 certificate.

- **GitOps, Atlanta.** August 11, 1pm to 5pm EDT. [Register](https://www.eventbrite.com/e/gitops-atlanta-tickets-1990879889342)
- **Apple administrator, New York.** August 12, 1pm to 5pm EDT. [Register](https://www.eventbrite.com/e/apple-administrator-workshop-tickets-1993125028614)
- **GitOps, Melbourne.** August 24, 1pm to 5pm GMT+10. [Register](https://www.eventbrite.com/e/gitops-x-world-tickets-1992169896789)
- **GitOps, Kansas City.** September 24, 8:30am to 10:30am EDT. [Register](https://www.eventbrite.com/e/speed-run-gitops-jnuc-tickets-1992170088362)
- **GitOps, Gothenburg.** September 28, 1pm to 5pm GMT+2. [Register](https://www.eventbrite.com/e/gitops-macsysadmins-tickets-1993054810590)

More dates are on the way, and nothing near you yet is worth telling us about. You can [request a workshop](https://fleetdm.com/workshops) in your city.

## Worth reading

### Customer stories

- [How Hawx gets field technicians productive on hour one](https://fleetdm.com/articles/hawx). Hawx automated seasonal iOS onboarding and offboarding with Fleet, Tines, and Okta, ending start-of-season helpdesk floods in one month.
- [How Primo built an HR-driven IT platform, powered by Fleet](https://fleetdm.com/articles/primo). Primo chose Fleet for the MDM layer inside its orchestration platform, scaling to 400 customers and 30,000 devices.

### Articles

- [Intune isn't free: what the Microsoft 365 bundle really costs in 2026](https://fleetdm.com/articles/intune-isnt-free-what-the-microsoft-365-bundle-really-costs). Microsoft's E7 tier and 2026 price increases expose the bundle fallacy. Price E3, E5, and E7 individually and rightsize device management.
- [How Fleet keeps Fleet-maintained apps safe and up to date](https://fleetdm.com/articles/inside-fleet-maintained-apps). Vendor-direct downloads, pinned hashes, validation on real hardware, and human review, behind every app in the catalog.
- [Take control of Apple beta programs with declarative device management](https://fleetdm.com/articles/control-apple-beta-programs-with-ddm). Use DDM software update settings to control beta enrollment, and automate fetching AppleSeed tokens from Apple Business Manager.

### Guides

- [Manage Windows updates with the Windows Update CSP](https://fleetdm.com/guides/custom-windows-updates). Enforce deadlines, build rollout rings, pin releases, and verify all of it.
- [Use custom host vitals in scripts and configuration profiles](https://fleetdm.com/guides/custom-host-vitals). Define a custom field, set a value per host, and use it as a variable.
- [Build a Linux self-service catalog with script-only packages](https://fleetdm.com/guides/build-your-own-linux-self-service-with-script-only-packages-guide). Turn apt and dnf commands into a self-service catalog your Linux users can install from.
- [Detect and remove unwanted software installed by peripherals](https://fleetdm.com/guides/detect-and-remove-peripheral-software). Find and clean up the software docks and peripherals install without asking.

## From the community

- [Fleet](https://www.linkedin.com/company/fleetdm/) published [three ways to find Intel-only Mac apps](https://www.linkedin.com/feed/update/urn:li:activity:7485034063301627904) still running in your fleet, ahead of Apple ending Rosetta support.
- [Zay Hanlon](https://www.linkedin.com/in/zayhanlon/) wrote about [taking the case study program in-house](https://www.linkedin.com/posts/zayhanlon_i-hijacked-the-case-study-program-from-marketing-share-7488595098134265856-WuPG) and what came out of the Hawx story.
- [Allen Houchins](https://www.linkedin.com/in/allenhouchins/) asked what ["it's already included"](https://www.linkedin.com/posts/allenhouchins_its-already-included-might-be-the-most-share-7483936371141693440-IFIH) really costs, and shared a [sneak peek at patch policies](https://www.linkedin.com/posts/allenhouchins_who-doesnt-love-a-sneak-peek-the-patch-ugcPost-7486499772229615616-SM1r).
- [Josh Roskos](https://www.linkedin.com/in/jroskos/) asked [an AI agent a compliance question](https://www.linkedin.com/posts/jroskos_i-asked-my-ai-agent-a-compliance-question-share-7486480039249747968-YJiZ) and posted what came back.

Workshop attendees in Sydney, Toronto, Boston, and Sweden also posted about their sessions. Thanks to everyone who shared.

<meta name="articleTitle" value="Fleet newsletter, August 2026">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-08-03">
<meta name="category" value="newsletters">
<meta name="description" value="What shipped in Fleet 4.89.0, what's planned for 4.90.0 and 4.91.0, upcoming GitOps and Apple workshops, and July's best guides and customer stories.">
<meta name="newsletterIssue" value="2026-08">
<meta name="coversPeriod" value="2026-07">
