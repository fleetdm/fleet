# Who installed DeepSeek? Under DHS Order 2026-01, ask the devices, not the employees

*DHS told civilian agencies to remove DeepSeek and anything built on it. The order assumes you already know where it runs, and most teams only have a guess.*

## Key takeaways

- **The order reaches past the DeepSeek app.** Exclusion and Removal Order #2026-01 covers DeepSeek, its affiliates, and third-party products built using DeepSeek implementations, so a search for one app name misses part of the scope.
- **Self-reporting is not an inventory.** A survey or an acceptable-use reminder tells you what people remember installing. It doesn't tell you what is on the device today.
- **Installed software is the first list to pull.** Fleet's software inventory records apps, packages, browser extensions, and IDE extensions on each host, so one name search can show where DeepSeek-branded software sits.
- **A name match has limits you should state up front.** A tool that embeds a DeepSeek model without saying so won't match on "deepseek." Treat the first pass as a floor, then work the wrappers through vendor answers and procurement records.
- **A policy turns the cleanup into a recheck.** A policy that fails on any host with matching software gives you a count to drive to zero, and the same count is your evidence afterward.

<a purpose="cta-button" href="https://fleetdm.com/guides/software-inventory-reference">See what Fleet's software inventory collects</a>

DHS has ordered civilian federal agencies to remove DeepSeek from their IT systems, and to stop buying it. The order is Exclusion and Removal Order #2026-01, issued under the Federal Acquisition Supply Chain Security Act of 2018. It was signed July 24, became public October 19, and was published in the Federal Register on October 20. DHS cited possible data transfer to China, possible intelligence gathering on US persons, and possible malware or vulnerabilities in the models themselves.

Most of the coverage is about the politics. The operational problem is simpler and harder: you can't remove what you can't find. If your fleet is a mix of macOS, Windows, and Linux hosts, the question for IT is where DeepSeek-related software actually is, today, on which devices.

## What the order covers

The order names DeepSeek, its affiliates, and third-party products and services developed using DeepSeek implementations. It also bars procurement, including through agency-approved product lists and e-commerce marketplaces.

That second category is the one that changes the work. A chatbot app with "DeepSeek" in its name is easy to spot. A coding assistant, note-taking tool, or browser extension that calls a DeepSeek model behind its own branding is not, and the order treats it the same way. You aren't only looking for one app. You're looking for a dependency.

Private companies aren't bound by a federal civilian order. Contractors, regulated industries, and security teams that follow federal guidance will still get the same question from a customer or auditor, so the same inventory work applies.

## Why asking people doesn't answer it

The usual first move is a message to staff asking them to say whether they use DeepSeek. It produces a short list of honest people and tells you nothing about everyone else. Someone who tried the app once, installed an extension in a personal browser profile, or pulled a model into a local runtime may not think of it as "using DeepSeek" at all.

A self-reported list also goes stale the day after you collect it. An inventory that updates on its own doesn't have that problem.

## Pulling the list from the devices

Fleet's software inventory collects installed apps, Linux packages, browser extensions, and IDE extensions across macOS, Windows, and Linux hosts. By default Fleet queries hosts for software inventory hourly. The [software inventory reference](https://fleetdm.com/guides/software-inventory-reference) lists exactly which software types and platforms are covered, including the ones that aren't.

The first pass is a name search. Search the software list for "deepseek" and you get every host reporting a matching app, package, or extension, along with the version and install path where the platform provides them. That list is your starting point for removal tickets.

Two caveats belong in the write-up. Fleet only sees what the inventory covers, so personal-side apps on BYOD devices and software inside containers or virtual machines can sit outside it. And a name search finds software that is named for DeepSeek. It does not tell you which models a tool calls at runtime.

## The wrappers need another source

For products built on DeepSeek, the device can only give you the inventory side. Fleet can list every AI assistant, extension, and developer tool installed, which is the set of candidates. Whether any of them embeds DeepSeek is a question for the vendor, the contract, and the procurement record.

A practical sequence: pull the full list of AI-related software from your hosts, send each vendor the same written question about which model providers they use, and record the answers next to the inventory. Where a vendor can't or won't answer, that is a decision for your security and procurement leads, not something a scan resolves.

## Making the removal measurable

Once you have a list of hosts and the software to remove, a policy keeps the work honest. A Fleet policy that passes only when no matching software is installed shows up as a failing-host count. Remove the software, let the next inventory check run, and the count falls.

That count is the useful artifact. "We told people to uninstall it" is a statement. "Fleet shows zero hosts failing the policy as of this morning" is something an auditor can read. If new software appears later, the same policy flags it without anyone remembering to look.

## Closing

The order tells agencies what to remove. The work for IT is finding it first, and that starts with the devices, which report what is installed whether or not anyone remembers to mention it. Take the inventory result as a floor, ask vendors about the wrappers, and keep the policy running so the count stays at zero.

## See it live

- **Get a demo.** See software inventory and policies across macOS, Windows, and Linux hosts: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read the reference.** Check what Fleet collects on each platform: [Software inventory reference](https://fleetdm.com/guides/software-inventory-reference)

## Sources

- Federal Register public inspection, [DHS Exclusion and Removal Order 2026-01, document 2026-21447](https://public-inspection.federalregister.gov/2026-21447.pdf).
- The Next Web, [US orders civilian federal agencies to remove DeepSeek from their systems](https://thenextweb.com/news/us-bars-deepseek-federal-agencies).

<meta name="articleTitle" value="Who installed DeepSeek? Under DHS Order 2026-01, ask the devices, not the employees">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-21">
<meta name="description" value="DHS Order 2026-01 removes DeepSeek and products built on it from federal systems. Find where it runs by asking devices, not employees.">
