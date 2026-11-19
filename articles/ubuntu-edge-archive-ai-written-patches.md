# If a judge model approved your Ubuntu patch, your servers are part of the test

*Canonical's new Edge archive ships AI-written security fixes to Ubuntu Pro users before qualification ends. Opting in is one command, so knowing which hosts ran it falls to you.*

## Key takeaways

- **AI agents now write some Ubuntu security fixes.** Canonical's Surge system matches known CVEs to Ubuntu packages, has competing models write backports, and sends them to a panel of judge models that must agree before a patch is built and tested.
- **Edge patches arrive before qualification is finished.** Canonical calls the archive "an opt-in preview of fixes in development." Its stated goal is to use real-world feedback from Pro users to qualify fixes for general release.
- **Your regular Ubuntu updates don't change.** Standard security updates stay human-validated, Canonical won't push Surge fixes into stable releases automatically, and the kernel is out of scope.
- **Turning it on is a local decision.** Any admin with Ubuntu Pro on 18.04 through 26.04 LTS can run `sudo apt install pro-edge && sudo pro-enable-edge`. One more flag makes Edge updates install automatically.
- **Knowing which hosts opted in is a device question.** The package, the configured source, and the installed versions all live on the host, and Fleet's agent can report all three across every Ubuntu machine you manage.
- **A policy can keep the decision honest.** Whether your team adopts Edge or forbids it, a Fleet policy can flag every host that doesn't match the rule you picked.

<a purpose="cta-button" href="https://fleetdm.com/software-management">See how Fleet tracks Linux patch state</a>

On November 12, Canonical opened an [opt-in Edge archive](https://canonical.com/blog/new-edge-archive-offers-opt-in-ai-driven-security-remediation-and-backports) for Ubuntu Pro subscribers. It carries security fixes written by Surge, which Canonical describes as "a multi-agent orchestration framework for CVE remediation and backporting." The fixes reach subscribers before Canonical's own extended testing and qualification are done.

Canonical's reasoning is volume. Pierre Guillemin, Canonical's VP of Excellence Engineering, said at Ubuntu Summit 26.10 that October 2026 alone saw almost as many CVEs as all of 2025, according to [The Register](https://www.theregister.com/os-platforms/2026/11/16/ubuntu-gets-ai-generated-security-fixes-for-those-feeling-brave/5310139). Whether Edge is a good trade is your team's call. Either way, someone has to answer which of your servers made that trade.

## How Surge writes a patch

[Help Net Security](https://www.helpnetsecurity.com/2026/11/18/ubuntu-ai-security-patches-edge-archive/) lays out the pipeline. One agent reads vulnerability feeds and drops reports that don't affect Ubuntu. Another pulls the upstream fix and the code around it. Several competing models each write their own backport to the older package version in a stable Ubuntu release.

A separate panel of judge models then reviews the candidates and has to reach consensus. Deadlocks, low confidence scores, and unusual cases go to a human on the Ubuntu security team. Patches that pass are built and run through the same deterministic tests as other Ubuntu packages.

Surge works on the backlog of already-known vulnerabilities in userspace. Kernel fixes are excluded and stay on a separate two-week Stable Release Update cycle.

## What "before qualification" means for you

Canonical is clear about the boundaries. "We will not automatically upload Surge-generated fixes to our stable releases," the company says, and regular security updates still get human validation. Edge sits outside that path, and fixes there are, in Canonical's words, "machine-generated CVE patches before they pass qualification and get promoted to general availability."

That's a reasonable offer for a test server or a host where an unpatched CVE is the bigger risk. It also means the qualification step that normally happens before you install a patch now partly happens after. Canonical asks Edge users to file bugs on [Launchpad](https://bugs.launchpad.net/ubuntu-security-edge/+filebug) and frames that feedback as part of qualifying fixes for everyone else.

## Opting in takes one command

The [Ubuntu Discourse guide](https://discourse.ubuntu.com/t/how-to-enable-the-edge-archive/91771) covers LTS releases from 18.04 through 26.04 and needs Ubuntu Pro with ESM Infra or ESM Apps enabled:

```
sudo apt install pro-edge
sudo pro-enable-edge
```

By default, Edge updates don't install on their own. An admin lists them with `apt list --upgradable '?origin(^Ubuntu.*Edge$)'` and installs each one by version. Adding `--automatic-updates` to `pro-enable-edge` lets a regular `apt upgrade` pull them in.

That default is careful, and it also makes the result hard to predict from the outside. Two hosts with Edge enabled can hold different packages, depending on what someone chose to install on each. A host where someone added the automatic flag behaves like a third kind of machine. Your configuration management may record none of this if the commands were run by hand during an incident.

## Checking from the device

The useful questions are concrete: which hosts have Edge enabled, which of them install Edge updates automatically, and which package versions are actually on disk. Fleet's agent reports from the host itself, so each of those is a report you can run across every Ubuntu machine.

To find hosts with the Edge tooling installed:

```sql
SELECT name, version FROM deb_packages WHERE name = 'pro-edge';
```

To find hosts with an Edge source configured, filter `apt_sources` on the repository origin that the Discourse guide matches (`^Ubuntu.*Edge$`):

```sql
SELECT name, base_uri, release, maintainer FROM apt_sources WHERE maintainer LIKE 'Ubuntu%Edge';
```

Confirm both on a test host after enabling Edge before you rely on them, since the exact strings may differ from what the guide implies. For package versions, Fleet's software inventory already lists installed `deb` packages and their versions on every Linux host. When Canonical publishes a CVE fix, you can compare the version each host reports against the version in the advisory. A host running a different build than the advisory names is worth a closer look. To see exactly what's pending, Fleet can [run a script](https://fleetdm.com/guides/scripts) such as the `apt list` command above on Linux hosts and return the output.

## Turn the decision into a policy

Some teams will want Edge on a handful of exposed servers. Others will want it nowhere near production. Both choices are easy to state and easy to drift from.

A Fleet policy can encode either rule. For example, this one passes only on hosts where `pro-edge` isn't installed:

```sql
SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM deb_packages WHERE name = 'pro-edge');
```

Scope it to your production fleet, and a host that opts in quietly starts failing. Policy automations can then open a ticket or send a webhook so the right team sees it. If your Fleet configuration lives in Git as [YAML](https://fleetdm.com/docs/configuration/yaml-files), changing the rule later is a reviewed pull request, not a console edit.

## Faster patches still need a record

Canonical is betting that AI-written fixes, checked by other models and then by early adopters, close exposure faster than human review alone. That bet may pay off. It only helps your organization if you can say which machines took part, which builds they're running, and whether that matches what you decided.

Edge moves part of the testing onto your hosts. The record of which hosts carry that risk should come from those hosts too.

## See it live

- **Get a demo** to see Linux package and patch state across your own hosts: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read how Fleet runs scripts on Linux, macOS, and Windows:** [fleetdm.com/guides/scripts](https://fleetdm.com/guides/scripts)

## Sources

- Canonical, [New Edge archive offers opt-in AI-driven security remediation and backports](https://canonical.com/blog/new-edge-archive-offers-opt-in-ai-driven-security-remediation-and-backports).
- Ubuntu Discourse, [How to enable the Edge archive](https://discourse.ubuntu.com/t/how-to-enable-the-edge-archive/91771).
- Help Net Security, [Ubuntu AI security patches are here, if you're willing to test them yourself](https://www.helpnetsecurity.com/2026/11/18/ubuntu-ai-security-patches-edge-archive/).
- The Register, [Ubuntu gets AI-generated security fixes for those feeling brave](https://www.theregister.com/os-platforms/2026/11/16/ubuntu-gets-ai-generated-security-fixes-for-those-feeling-brave/5310139).
- OMG! Ubuntu, [Ubuntu Pro users can get early access to AI-written security fixes](https://www.omgubuntu.co.uk/2026/11/ubuntu-pro-ai-security-archive).

<meta name="articleTitle" value="If a judge model approved your Ubuntu patch, your servers are part of the test">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-11-19">
<meta name="description" value="Ubuntu Pro's Edge archive ships AI-written patches before qualification ends. Here's how to see which of your Ubuntu hosts opted in and what they run.">
