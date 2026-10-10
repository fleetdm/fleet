# 463 kernel CVEs cite Google's Sashiko this year. Which of your Linux hosts do they touch?

*AI tools now find kernel bugs faster than most teams can read the advisories. The question that keeps its value is which of your hosts run the affected code.*

## Key takeaways

- **The reports are real now.** Linux kernel security lead Greg Kroah-Hartman told The Register that AI bug reports went from "slop" to "real reports" in about a month, and that the increase "is not slowing down."
- **One AI tool is already behind hundreds of CVEs.** At the Linux Plumbers Conference, Google's Roman Gushchin said 463 kernel CVEs assigned this year cite Sashiko, which has reviewed more than 191,000 patches.
- **The kernel can absorb the load, and you may not.** Kroah-Hartman says the kernel team can handle the volume. Your patching process is a different system, and a higher CVE count lands there too.
- **A fixed upstream is not a fixed host.** A patch existing in the kernel tree says nothing about which of your machines run a build that includes it.
- **Per-host version data answers the question.** Knowing the kernel and package version on every Linux host turns "are we affected?" from a debate into a lookup.
- **Run the check before the next batch arrives.** The teams that cope are the ones who can already list their hosts by kernel version.

<a purpose="cta-button" href="https://fleetdm.com/software-catalog">See software inventory in Fleet</a>

In March, Kroah-Hartman [told The Register](https://www.theregister.com/software/2026/03/26/linux-kernel-czar-says-ai-bug-reports-arent-slop-anymore/5226256) that something had changed. Months earlier the kernel team was receiving what it called AI slop. Then, in his words, "the world switched," and the reports started pointing at real problems.

The numbers since then are large. [Phoronix](https://www.phoronix.com/news/Sashiko-Linux-AI-Metrics) reports that Sashiko, a review tool that began at Google and now sits under the Linux Foundation, has completed more than 191,000 patch reviews, and that 463 kernel CVEs assigned this year cite it. Each CVE is a question for anyone who runs Linux: does this one apply to us?

## What changed in the kernel's workload

Kroah-Hartman said the kernel team can handle the extra volume, because it has a large group of maintainers to share the work. He also said "all open source security teams are hitting this right now," and that smaller projects have far less room to absorb it. The increase, he said, "is real, and it's not slowing down."

He described AI mostly as a reviewer, not an author. Reviews catch many obvious issues but not everything, and he noted that about one-third of one batch of AI findings were wrong, though they still pointed at a relatively real problem. That is useful context. More findings means more CVEs, and not every CVE carries the same weight for your hosts.

I could not find a source tying Kroah-Hartman to the 463 figure directly. It comes from Gushchin's conference update as reported by Phoronix, and Phoronix's headline rounds it to "nearly 500." The number counts CVEs that cite Sashiko, not a measure of how many were found by it alone.

## Where the volume lands on your team

The kernel's maintainers handle the intake. Your team handles the other end: reading each advisory, working out which distributions and kernel versions are affected, and confirming that your hosts took the fix. Distributions backport fixes into their own kernel builds, so the version string on a host often doesn't match the upstream version named in a CVE. That makes the manual work slower than it looks.

More CVEs does not mean more risk in equal measure. It does mean more triage, and triage without data on your own hosts turns into guessing.

## Answering the question per host

The useful question is concrete. Which Linux hosts do you run, which kernel and package versions is each on, and which of those versions fall in an affected range for the CVEs you care about?

Fleet's agent collects installed deb and rpm packages on Linux hosts, with name and version, so kernel packages appear in the **Software** tab alongside everything else. The [software inventory reference](https://fleetdm.com/guides/software-inventory-reference) lists what Fleet collects on each platform. For a question the inventory doesn't answer directly, such as the running kernel version on each host, a [report](https://fleetdm.com/guides/queries) can ask the host itself. Test any check on a host where you already know the answer before you trust an empty result.

Fleet also reports known vulnerabilities per host. Coverage differs by distribution and by data source, so confirm how your distributions are handled in the [vulnerability documentation](https://fleetdm.com/guides/vulnerability-processing) instead of assuming every CVE will appear.

## Where this leaves you

Nobody can slow the pace of AI-found bugs, and the kernel team says it isn't slowing. What a team controls is how quickly it can say which hosts run an affected build and which have taken the fix. Build that list now, while the volume is manageable.

## See it live

- **Get a demo:** [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read the argument for ground truth:** [fleetdm.com/articles/agentic-security-needs-ground-truth](https://fleetdm.com/articles/agentic-security-needs-ground-truth)

## Sources

- The Register, [Linux kernel czar says AI bug reports aren't slop anymore](https://www.theregister.com/software/2026/03/26/linux-kernel-czar-says-ai-bug-reports-arent-slop-anymore/5226256).
- Phoronix, [Google's Sashiko AI has completed 191k patch reviews, cited on nearly 500 kernel CVEs](https://www.phoronix.com/news/Sashiko-Linux-AI-Metrics).
<meta name="articleTitle" value="463 kernel CVEs cite Google's Sashiko this year. Which of your Linux hosts do they touch?">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-09">
<meta name="description" value="Sashiko is cited in 463 kernel CVEs this year. Per-host kernel and package versions tell you which of your Linux machines are affected.">
