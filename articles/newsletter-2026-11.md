# Fleet news: November 2026

Our first virtual summit streams free on November 10, and [you're invited](https://www.linkedin.com/events/7518683538410459136/)! October also brought two releases, including a heads-up for Mac users before Fleet patches the apps they're working in. Let's dig in!

## 🚀 What shipped last month

- **Warn end users before patching macOS apps.** With Fleet Premium, patch policies can now notify end users when an open app is about to update. Fleet Desktop lists the apps, waits an hour with a reminder five minutes before, and then installs the update, so nobody loses unsaved work.
- **Zero-touch enrollment for company-owned Android.** With Fleet Premium, ship company-owned Android devices straight to end users, and they enroll in Fleet the first time they're turned on. As of Fleet 4.94.0, you choose which fleet each device lands in.
- **Self-service configuration profiles for macOS.** Offer configuration profiles that end users install themselves from the **My device** page, instead of pushing every profile to every host. Mark a profile hidden to keep it off the list until an end user turns on **Show hidden profiles**.

Also shipped in October: a dedicated configuration profile editor with names and descriptions, Windows local admin password rotation, on-demand FileVault recovery key rotation, and reports that keep a result for every host instead of stopping at 1,000. [See every release](https://fleetdm.com/releases).

## 🗺️ What we plan to ship this month

Fleet 4.95.0 is planned for November. These are planned, not promised, and scope changes as the release comes together.

- **AI tools in software inventory.** With Fleet Premium, see which AI tools are installed across your hosts on the **Software** page, **Host details**, and **My device**, without writing a custom report.
- **Patch deadlines for macOS apps.** With Fleet Premium, set a deadline for Fleet-maintained app patches, so updates land within your organization's patching rules.
- **Configuration profile validation.** Fleet checks the type, keys, and values in Apple and Windows configuration profiles, so you know a profile built with Fleet's AI-native builder, or any other tool, is valid before it reaches a host.

Also on deck: Okta conditional access for Windows hosts, scoping Android Play Store apps with labels, running a bootstrap package during manual macOS enrollment, and multiple versions of the same App Store app for iOS, iPadOS, and Android.

Fleet plans releases in the open. The [release planning board](https://github.com/orgs/fleetdm/projects/87/views/10) shows what's queued for the releases after this one, and the [October roadmap preview](https://fleetdm.com/announcements/roadmap-preview-october-2026) walks through the next 90 days, including Fleet 5.

## 📺 Fleet Virtual Summit

Fleet's first virtual summit streams free on LinkedIn Live on Tuesday, November 10, at 11 AM ET / 8 AM PT / 4 PM GMT. Four 30-minute panels of IT and security practitioners cover infrastructure as code for device management, open source in enterprise device management, AI agents as a device management problem, and device management in 2027. [Mike McNeil](https://www.linkedin.com/in/mikermcneil/), Fleet's cofounder and CEO, hosts all four. [See the sessions and panelists](https://fleetdm.com/announcements/fleet-virtual-summit-2026).

## 🎓 Upcoming workshops

Fleet workshops are free, run about four hours, and set you on course to becoming a Fleet-certified expert. Seats are limited, so register early.

- **GitOps, Chicago.** November 6, 1pm to 5pm CST. [Register](https://www.eventbrite.com/e/gitops-chicago-tickets-2003719976388)
- **GitOps, Tokyo.** November 12, 1pm to 5pm GMT+9. [Register](https://www.eventbrite.com/e/gitops-tokyo-tickets-2003716644422)
- **GitOps, Austin.** November 17, 1pm to 5pm CST. [Register](https://www.eventbrite.com/e/gitops-austin-tickets-2002266719658)
- **Apple administrator, San Antonio.** November 18, 1pm to 5pm CST. [Register](https://www.eventbrite.com/e/apple-administrator-workshop-san-antonio-tickets-2002266872114)
- **Apple administrator, Raleigh.** December 15, 1pm to 5pm EST. [Register](https://www.eventbrite.com/e/apple-administrator-workshop-raleigh-tickets-2002267524064)

Don't see your city? [Request a workshop](https://fleetdm.com/contact#gitops) to get hands-on experience and earn your certification.

## 📖 Worth reading

### Articles

- [New research: AI helps Mac admins ship changes quickly, but rollback often takes days](https://fleetdm.com/announcements/apple-in-the-enterprise-2026). Fleet surveyed more than 250 enterprise Mac admins: 86% let AI-written changes reach production devices, but 69% can't roll back a bad change within an hour.
- [Fleet brings NVIDIA AI hardware under enterprise IT control](https://fleetdm.com/announcements/nvidia-gtc-2026). Manage NVIDIA RTX PRO workstations and NVIDIA DGX Station alongside your macOS, Windows, and Linux devices.
- [Tailworm: what to do about Tailscale SSH before the fix ships](https://fleetdm.com/articles/tailworm-what-to-do-about-tailscale-ssh-before-the-fix-ships). Find the Linux servers and Macs where Tailscale SSH can forward as root, and close the path today with one environment variable and a Fleet policy.

### Guides

- [Revert a bad change in GitOps](https://fleetdm.com/guides/revert-a-bad-change-in-gitops). Undo a bad change with a revert commit instead of a reset and force-push, so your history keeps a record of what changed and why.
- Block browser extensions in [Chrome](https://fleetdm.com/guides/block-chrome-extensions), [Edge](https://fleetdm.com/guides/block-edge-extensions), [Firefox](https://fleetdm.com/guides/block-firefox-extensions), and [Safari](https://fleetdm.com/guides/block-safari-extensions). Block, allow, or force-install extensions across macOS, Windows, and Linux.
- [Use Claude Code with Fleet](https://fleetdm.com/guides/use-claude-code-with-fleet). Generate Fleet policies, reports, and configuration profiles with Claude Code, with or without GitOps.



## 🤝 Community contributions

Fleet is open source, and people outside Fleet helped build it in October.

- [Michael Alt](https://github.com/alt-Rational) added Azure Blob Storage support for software installers, bootstrap packages, and organization logos, so self-managed Fleet deployments on Azure can keep those files in their own storage.
- [Andrew Doering](https://github.com/delize) made `fleetctl gitops --dry-run` check that each App Store app is available on your VPP account, so a wrong or unlicensed app fails in CI instead of during the real apply.
- [Adriel Velazquez](https://github.com/AdrielVelazquez) extended Linux remote wipe to clear the Nix store on NixOS hosts.

Thanks also to everyone who made their first contribution to Fleet in October: [@416rehman](https://github.com/416rehman), [Aliaska Varieva](https://github.com/aliaska-varieva), [@AmanSarsenov-01](https://github.com/AmanSarsenov-01), [Ben Owens](https://github.com/browens1116), [Billy Bryant](https://github.com/billyjbryant), [Catherine Lu](https://github.com/cathyhulu), [Chris Durham](https://github.com/chrsdrhm), [Chris Kennard](https://github.com/chriskennard-kobold), [Dan Keller](https://github.com/K3ll3r), [Eloi Torrents](https://github.com/Eloitor), [Eric Beahan](https://github.com/ebeahan), [IOCRYPOSTO](https://github.com/Aaditya1273), [Jake P](https://github.com/Taraan03), [Jarryd Stanbrook](https://github.com/jarrydstan), [Jason Kurtz](https://github.com/JasonMKurtz), [@jdscreations](https://github.com/jdscreations), [Liu XingJian](https://github.com/tianxingjian2406), [Manik Singh](https://github.com/Manik-733), [Matt Anderson](https://github.com/Synse), [@mjsenatore](https://github.com/mjsenatore), [Om Lahore](https://github.com/omlahore), [Ovo](https://github.com/OvOhao), [@patrickrgr](https://github.com/patrickrgr), [Prajwal Janbandhu](https://github.com/Prajwal2246), [Pratik Singh](https://github.com/kitarp29), [Rayen Mabrouk](https://github.com/RayenMabrouk2), [Robbie Trencheny](https://github.com/robbiet480), [@shubhamkapoor01](https://github.com/shubhamkapoor01), [timball](https://github.com/timball), [Tobias Almén](https://github.com/almenscorner), and [Yanis](https://github.com/ynaamane).

Thank you all!

That's a wrap for November. [See you at the summit](https://www.linkedin.com/events/7518683538410459136/)!

<meta name="articleTitle" value="Fleet news: November 2026">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="publishedOn" value="2026-11-02">
<meta name="category" value="newsletter">
<meta name="description" value="What shipped in Fleet 4.93.0 and 4.94.0, what's planned for 4.95.0, the Fleet Virtual Summit, November workshops, and October's best reads.">
<meta name="newsletterIssue" value="2026-11">
<meta name="coversPeriod" value="2026-10">
