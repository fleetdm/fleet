# Fleet news: October 2026

Leaves are starting to fall in the US, but Windows hosts aren't falling into "Unassigned" anymore. September brought Fleet 4.91.0 and 4.92.0, a customer story from Abridge, and a guide to binary allow lists on macOS 27. Two more releases are planned for October, so let's right dive in!


## 🚀 What shipped last month

- **Windows Autopilot hosts land in the right fleet.** Choose a default fleet for Autopilot-enrolled hosts, so new Windows hosts skip "Unassigned." With Fleet Premium, Autopilot hosts also show up as pending hosts, with their group tag, before they enroll, so you can move them to the right fleet ahead of time.
- **Run any Android command.** Send any Android Management API command with `fleetctl mdm run-command` or Fleet's API, then see the results on **Host details**, the same way you already can for Apple and Windows hosts.
- **OS updates that follow the latest version.** Set macOS, iOS, and iPadOS updates to always target the latest OS version, with a deadline counted in days after each release. Fleet moves the target as Apple ships updates, so you no longer bump the minimum version by hand.

Also shipped in September: a hidden local admin account with an escrowed password on every Windows host, policy automations that resend a configuration profile when a host fails, and SSO in front of Fleet Desktop self-service. [See every release](https://fleetdm.com/releases).

## 🗺️ What we plan to ship this month

Fleet 4.93.0 and 4.94.0 are both planned for October. These are planned, not promised, and scope changes as each release comes together.

- **Prompt end users before app updates on macOS.** Patch policies that ask end users before updating an app, instead of updating it mid-task.
- **Zero-touch enrollment for company-owned Android.** Enroll company-owned Android devices in Fleet without anyone touching them first.
- **Windows: force standard account.** Keep end users off local administrator accounts on Windows hosts.

Also on deck: ACME certificates for iOS and iPadOS hosts enrolled through Apple Business, an option to allow only automated enrollment for Apple devices, on-demand FileVault key rotation, and opt-in configuration profiles in macOS self-service.

Want to see how these are coming along? We plan everything in the open. Bookmark our [release planning board](https://github.com/orgs/fleetdm/projects/87/views/10) to keep an eye on the queue and join the discussion.

## 🎓 Upcoming workshops

Fleet workshops are free, run about four hours, and set you on course to becoming a Fleet-certified expert. Seats are limited, so register early.

- **GitOps, Richmond.** October 13, 1pm to 5pm EDT. [Register](https://www.eventbrite.com/e/gitops-richmond-tickets-1993052268988)
- **Apple administrator, Denver.** October 20, 1pm to 5pm MDT. [Register](https://www.eventbrite.com/e/apple-administrator-workshop-denver-tickets-2002264951369)
- **Apple administrator, Houston.** October 22, 1pm to 5pm CDT. [Register](https://www.eventbrite.com/e/apple-administrator-workshop-houston-tickets-2002265456881)
- **Apple administrator, New York City.** October 27, 1pm to 5pm EDT. [Register](https://www.eventbrite.com/e/apple-administrator-workshop-nyc-tickets-2002265545145)
- **GitOps, Philadelphia.** October 29, 1pm to 5pm EDT. [Register](https://www.eventbrite.com/e/gitops-philadelphia-tickets-2002266067708)

Don't see your city? [Request a workshop](https://fleetdm.com/contact#gitops) to get hands-on experience and earn your certification.

## 📖 Worth reading

### Customer stories

- [How Abridge gets new hires productive in 10 minutes with Fleet](https://fleetdm.com/articles/abridge). The healthcare AI company uses GitOps and real-time visibility to onboard new hires fast and answer security questions in minutes.

### Articles

- [Autopilot without Autopilot: zero-touch Windows deployment with Fleet](https://fleetdm.com/articles/autopilot-without-autopilot). Ship Windows laptops straight to end users, enrolled from the first boot, without a per-user Entra ID P1 license.
- [The power of collaboration: an admin-rights audit trail for every Mac](https://fleetdm.com/articles/the-power-of-collaboration). How three open-source projects shipped a queryable record of every admin grant on macOS in about a week.

### Guides

- [Binary authorization on macOS 27](https://fleetdm.com/guides/binary-authorization-on-macos-27). Use the new native allow and deny lists to control which apps and command-line tools can run on your Macs.
- [Enroll Linux hosts on first boot with cloud-init](https://fleetdm.com/guides/enroll-linux-hosts-on-first-boot-with-cloud-init). New Linux VMs and Ubuntu installs enroll in Fleet automatically, with nobody touching them.
- [Deploy Socket Firewall Free with Fleet](https://fleetdm.com/guides/deploy-socket-firewall-free-with-fleet). Block known-malicious npm, pip, and cargo packages on developer machines, and reinstall the firewall automatically when it goes missing.

## 💬 From the community

- [Fleet](https://www.linkedin.com/company/fleetdm/) [announced it has joined Anthropic's Project Glasswing](https://www.linkedin.com/feed/update/urn:li:activity:7506085598974312448/) to research how frontier AI can strengthen the security of device management systems and MDM protocols, and to share what we learn in the open.

Thank you to everyone who contributes to Fleet and uses it every day.

<meta name="articleTitle" value="Fleet news: October 2026">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="publishedOn" value="2026-10-01">
<meta name="category" value="newsletter">
<meta name="description" value="What shipped in Fleet 4.91.0 and 4.92.0, what's planned for 4.93.0 and 4.94.0, October workshops, and September's best guides and customer stories.">
<meta name="newsletterIssue" value="2026-10">
<meta name="coversPeriod" value="2026-09">
