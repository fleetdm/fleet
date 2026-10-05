# Which of your Ubuntu hosts run radosgw or EDK II? Two October 5 notices set the checklist

*Ubuntu patched Ceph's object gateway and the EDK II firmware framework on the same day. Whether your fleet is covered depends on which hosts run either package, and which took the update.*

## Key takeaways

- **Two notices landed on October 5.** USN-8867-1 fixes a Ceph flaw on four Ubuntu releases, and USN-8865-1 fixes several EDK II issues on six, so one patch day can touch very different hosts.
- **The Ceph bug is about presigned URLs.** The Ceph Object Gateway (RGW) honored extra headers that weren't part of the signature, which could let a URL holder do more than its signer intended.
- **EDK II reaches further back than most patching.** The notice covers releases from 26.04 LTS down to 16.04 LTS, including releases that only get fixes through Ubuntu Pro.
- **Package inventory answers "who has it."** Fleet's agent inventories installed packages on each Linux host, so you can list hosts running `radosgw`, `librgw2`, or EDK II packages instead of assuming.
- **Vulnerability status shows who's still exposed.** Fleet reports vulnerability status per host, so a host that skipped the update stays visible after the notice leaves the news cycle.

<a purpose="cta-button" href="https://fleetdm.com/software-catalog">See package inventory in Fleet</a>

On October 5, 2026, Ubuntu published two security notices that don't have much in common, apart from the day. One covers Ceph, the storage platform behind many Linux infrastructure deployments. The other covers EDK II, the open-source firmware framework used in virtual machines and some physical hardware.

Neither is the kind of package everyone installs on purpose, which is the point. You can't assume unattended upgrades reached the hosts that matter if you don't know which hosts have the packages at all.

## The Ceph notice: USN-8867-1

[USN-8867-1](https://ubuntu.com/security/notices/USN-8867-1) addresses CVE-2026-54330 in Ceph's Object Gateway. Ubuntu says the SigV4 handler didn't reject requests carrying `x-amz-*` headers that weren't in the signed header set. Someone holding a presigned URL could attach unsigned headers that RGW would honor, escalating beyond what the URL's signer intended.

The affected packages are `librgw2` and `radosgw`, with fixes for Ubuntu 26.04 LTS, 22.04 LTS, 20.04 LTS, and 18.04 LTS. Fixes for 20.04 and 18.04 come through Ubuntu Pro.

## The EDK II notice: USN-8865-1

[USN-8865-1](https://ubuntu.com/security/notices/USN-8865-1) says several security issues were fixed in EDK II, including CVE-2026-63072, CVE-2026-63076, and CVE-2026-54874. It covers Ubuntu 26.04 LTS, 24.04 LTS, 22.04 LTS, 20.04 LTS, 18.04 LTS, and 16.04 LTS.

Read the notice for the full CVE list and package names before building a search around it.

## Finding the hosts that matter

Fleet's agent inventories installed packages on each Linux host. Search the software inventory for `radosgw` or `librgw2` and you get every host running the Ceph gateway, with the version each one reports. Do the same for the EDK II packages named in the notice, then compare against the fixed versions Ubuntu lists for each release.

Fleet also reports vulnerability status per host. A host that took the update drops off the list of affected machines, and one that didn't stays on it. Because policies can live in Git as YAML in a GitOps workflow, raising a minimum package version is a reviewed pull request instead of a console change.

## Old releases need their own check

Both notices reach releases that are past standard support. For 20.04, 18.04, and, in EDK II's case, 16.04, fixes depend on Ubuntu Pro coverage. A host on one of those releases without that coverage may not have a fix to install, and a per-host version list is the quickest way to see which ones those are.

## See it live

- **Get a demo** to see package versions and vulnerability status across your own hosts: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Explore the software catalog** Fleet builds from every host: [fleetdm.com/software-catalog](https://fleetdm.com/software-catalog)

<meta name="articleTitle" value="Which of your Ubuntu hosts run radosgw or EDK II? Two October 5 notices set the checklist">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-05">
<meta name="description" value="Ubuntu patched Ceph's object gateway and EDK II on October 5. See how to find which hosts run either package and which took the fix.">
