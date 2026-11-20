# Canonical's AI agents now write Ubuntu security patches. Find the servers that opted in to test them

*Canonical's new Edge archive delivers AI-written security fixes to Ubuntu Pro users before Canonical finishes qualifying them. Opting in takes two commands on each host, so the record of which servers opted in has to come from the hosts themselves.*

## Key takeaways

- **AI agents now write some Ubuntu security fixes.** Canonical's Surge system maps known CVEs to Ubuntu packages, uses competing models to write backports, and sends the candidates to a panel of judge models that must reach consensus before a patch is built and tested.
- **Edge fixes arrive before qualification is complete.** Canonical calls the archive "an opt-in preview of fixes in development," while standard Ubuntu security updates remain human-validated and kernel fixes stay out of scope.
- **Opting in is a decision made on each host.** Any Ubuntu Pro host on 18.04 through 26.04 LTS can enable Edge with two commands, and one additional flag lets unattended-upgrades install Edge fixes automatically.
- **Fleet can show which hosts opted in.** Fleet's agent reports installed packages and configured APT sources from every Ubuntu host, and a Fleet policy can flag any host that does not match your team's decision.

<a purpose="cta-button" href="https://fleetdm.com/software-management">See how Fleet tracks Linux patch state</a>

On November 12, Canonical opened an [opt-in Edge archive](https://canonical.com/blog/new-edge-archive-offers-opt-in-ai-driven-security-remediation-and-backports) for Ubuntu Pro subscribers. It carries security fixes produced by Surge, which Canonical describes as "a multi-agent orchestration framework for CVE remediation and backporting." These fixes reach subscribers before Canonical's own extended testing and qualification are finished.

Canonical's stated reason is the volume of new vulnerabilities. At Ubuntu Summit 26.10, Pierre Guillemin, Canonical's VP of Excellence Engineering, said "October 2026 saw almost as many CVEs as all of 2025," [The Register](https://www.theregister.com/os-platforms/2026/11/16/ubuntu-gets-ai-generated-security-fixes-for-those-feeling-brave/5310139) reports. Whether Edge is a sensible trade is a decision for each team, but every team needs to know which of its servers made that trade.

## How Surge writes a patch

According to Canonical, Surge divides the work among AI agents with narrow roles. One stage reads vulnerability feeds and discards any report for a package Ubuntu does not ship. Another retrieves the upstream fix and its source context, and a third runs several competing models to write candidate backports for the older package versions in stable releases.

A separate panel of judge models then has to agree on a candidate. Disagreements, low-confidence results, and unusual cases are escalated to Canonical's security engineers. Approved patches go through Ubuntu's standard build and deterministic test process before publication. Surge covers only already-known vulnerabilities in userspace, and kernel fixes follow their own two-week Stable Release Update cycle.

## What "before qualification" means

Canonical draws a clear line around the archive: "We will not automatically upload Surge-generated fixes to our stable releases," the company writes, and updates through the traditional security channels remain human-validated. The Edge archive, in Canonical's words, provides "machine-generated CVE patches before they pass qualification and get promoted to general availability."

That trade can make sense for a test server or for a host where an unpatched CVE is the greater risk. It also means that part of the qualification normally completed before installation now happens afterward, on the subscriber's hosts. Canonical asks Edge users to report problems on [Launchpad](https://bugs.launchpad.net/ubuntu-security-edge/+filebug) and describes that feedback as part of validating fixes for everyone else.

## Opting in takes two commands

The [Ubuntu Discourse guide](https://discourse.ubuntu.com/t/how-to-enable-the-edge-archive/91771) covers LTS releases from 18.04 through 26.04 and requires Ubuntu Pro with ESM Infra or ESM Apps enabled. An administrator installs the `pro-edge` package and then runs `pro-enable-edge`. By default, APT ranks the Edge archive below the standard Ubuntu and ESM archives, so an Edge update is installed only when an administrator selects a specific package version.

Adding the `--automatic-updates` flag to `pro-enable-edge` removes that safeguard. Edge fixes then install with a regular `apt upgrade`, and unattended-upgrades, which Ubuntu enables by default, installs them without anyone logging in. As a result, two hosts with Edge enabled can carry different packages, and if the commands were run by hand during an incident, configuration management may have no record of either.

## Checking from the device

The questions that matter are which hosts have Edge enabled, which install Edge fixes automatically, and which package versions are on disk, and Fleet's agent answers them from the host itself. Fleet's [software inventory](https://fleetdm.com/guides/software-inventory) lists every installed `deb` package and its version on each Linux host, so the `pro-edge` package shows which hosts installed the Edge tooling. [OMG! Ubuntu](https://www.omgubuntu.co.uk/2026/11/ubuntu-pro-ai-security-archive) notes that removing `pro-edge` disables the archive, which makes the package a reasonable marker for opted-in hosts.

Fleet's agent can also read the APT sources configured on each host, and Fleet can [run a script](https://fleetdm.com/guides/scripts) such as the `apt list` command from Canonical's guide to show which Edge updates are pending. Whether a host installs Edge fixes automatically depends on its APT configuration, and [Fleet support](https://fleetdm.com/support) can help build a check for that setting.

## Turn the decision into a policy

Some teams will enable Edge on a few exposed servers, and others will keep it away from production entirely. Either decision is easy to state and easy to drift from. A Fleet [policy](https://fleetdm.com/securing/what-are-fleet-policies) can encode the rule, for example by failing on any production host where `pro-edge` is installed. A [policy automation](https://fleetdm.com/guides/automations) can then open a ticket or send a webhook so the responsible team sees the change, and teams that manage Fleet through [YAML in Git](https://fleetdm.com/docs/configuration/yaml-files) can change the rule later through a reviewed pull request.

## Faster patches still need a record

Canonical is betting that AI-written fixes, checked by other models and then by early adopters, will close exposure faster than human review alone. Because Edge moves part of that testing onto your hosts, the record of which hosts carry the risk should come from those hosts as well.

## See it live

- **Get a demo:** [Talk to the Fleet team](https://fleetdm.com/contact) about tracking Linux package and patch state across your hosts.
- **Rather try it yourself?** Send your IT team to a [free, hands-on Fleet workshop](https://fleetdm.com/workshops) to learn how to write policies like the Edge check and keep them in Git.

## Sources

- Canonical, [New Edge archive offers opt-in AI-driven security remediation and backports](https://canonical.com/blog/new-edge-archive-offers-opt-in-ai-driven-security-remediation-and-backports), November 12, 2026.
- Ubuntu Discourse, [How to enable the Edge archive](https://discourse.ubuntu.com/t/how-to-enable-the-edge-archive/91771), November 12, 2026.
- The Register, [Ubuntu gets AI-generated security fixes for those feeling brave](https://www.theregister.com/os-platforms/2026/11/16/ubuntu-gets-ai-generated-security-fixes-for-those-feeling-brave/5310139), November 16, 2026.
- OMG! Ubuntu, [Ubuntu Pro users can get early access to AI-written security fixes](https://www.omgubuntu.co.uk/2026/11/ubuntu-pro-ai-security-archive).

<meta name="articleTitle" value="Canonical's AI agents now write Ubuntu security patches. Find the servers that opted in to test them">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-11-19">
<meta name="description" value="Ubuntu Pro's Edge archive ships AI-written patches before qualification ends. See which of your Ubuntu hosts opted in and what they run.">
