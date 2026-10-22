# DeepSeek ban for federal agencies: How to prove every device is clean under DHS Order 2026-01

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

Civilian executive branch agencies are in scope. Their national security and secure compartmented information systems are not. Two requirements take effect immediately:

- **Exclude** DeepSeek and its affiliates from procurement, including agency-approved product lists and e-commerce marketplaces.
- **Remove** DeepSeek and affiliate products, plus "third-party products and services developed using DeepSeek implementations," from agency IT.

The second half is what changes the work. A chatbot named DeepSeek is easy to spot. A coding assistant that calls a DeepSeek model under its own brand is not, and the order treats both the same. You aren't looking for an app. You're looking for a dependency.

An agency that can't comply right away has 60 days from issuance to file a written exception request. The request has to name the covered product and describe the mitigations in place. Until removal is finished, DHS encourages traffic monitoring, enhanced logging, and exfiltration controls.

Private companies aren't bound by the order. Contractors and regulated teams will hear the same question from customers and auditors anyway.

The order is a United States federal measure. FASCSA orders cover federal executive agencies, so state and local governments and organizations outside the US aren't in scope. Federal contractors can be pulled in through their contracts: [FAR 52.204-30](https://www.acquisition.gov/far/52.204-30) bars a contractor from providing or using, in contract performance, a covered article named in an applicable FASCSA order. Applicable orders are the ones listed in SAM.gov or in the solicitation. A contractor that finds one must report it to the contracting officer within 3 business days. The inventory steps below work the same either way.

## Why asking people doesn't answer it

An email asking who uses DeepSeek gets you a short list of honest people. Someone who tried the app once, added an extension to a personal browser profile, or pulled a DeepSeek model into Ollama for a weekend project may not count that as "using DeepSeek." The list is also stale the day you collect it.

## Where to look on each device

Fleet's agent reports installed software hourly by default. Search the **Software** page for "deepseek" to see every host with a match, its version, and, where the platform reports it, the install path. The [software inventory reference](https://fleetdm.com/guides/software-inventory-reference) lists what's covered, gaps included.

- **Desktop apps:** macOS and Windows apps, and Linux deb and rpm packages.
- **Mobile apps:** iOS, iPadOS, and Android apps on devices enrolled in Fleet's MDM. Fully managed devices report every app. [BYOD iPhones and iPads](https://fleetdm.com/guides/enroll-byod-ios-ipados-hosts) report only apps Fleet installed, and Android BYOD reports the work profile only.
- **Browser extensions:** Chrome, Edge, Brave, Firefox, Safari, and more on macOS, Windows, and Linux.
- **IDE extensions:** VS Code, Cursor, Windsurf, and JetBrains plugins, where AI coding assistants tend to live.
- **Python packages:** system, per-user, pipx, uv, and conda installs.

Two things inventory can't show you. Model weights are files, not software: a DeepSeek model pulled into [Ollama](https://docs.ollama.com/faq) appears as Ollama, not as DeepSeek. And the web chatbot at chat.deepseek.com installs nothing. That belongs to DNS and proxy logs, which is the monitoring the order already asks for.

## Keeping watch with osquery

Removal isn't a one-time sweep. Someone reinstalls the app, a new hire brings an extension over from a personal profile, or a developer tries a new model. The inventory has to keep looking after the cleanup is done.

[Fleet's agent](https://fleetdm.com/docs/get-started/anatomy#fleetd) is built on [osquery](https://www.osquery.io/), the open-source agent that exposes a device's apps, packages, extensions, and files as tables you can ask questions of. That's what keeps the software list current on its own. When DeepSeek comes back on a host, it shows up at the next refresh, with no new survey and no new scan.

osquery reaches past the software list too. Its [tables](https://fleetdm.com/tables) cover the file system, so the places inventory can't see, such as model folders on disk, are still within reach. Fleet [reports](https://fleetdm.com/docs/get-started/anatomy#report) save those questions and run them across macOS, Windows, and Linux hosts. If you need help shaping one for DeepSeek, [Fleet support](https://fleetdm.com/support) or your customer success team can help. If you aren't a Fleet customer yet, [talk to Fleet](https://fleetdm.com/contact).

"We told people to uninstall it" is a statement. "Zero hosts report DeepSeek as of this morning" is evidence. Fleet counts how many hosts have each software title, so the number you drive to zero is already on the **Software** page.

## The wrappers need another source

For products built on DeepSeek, the device gives you the candidate list: every AI assistant, extension, and developer tool installed. Send each vendor the same written question about which model providers they use, and record the answers next to the inventory. A vendor that won't answer is a call for security and procurement, or the subject of an exception request. A scan won't settle it.

## The count is the compliance

The order says what to remove. Finding it is the work, and devices report what's installed whether anyone mentions it or not. Treat the first result as a floor, chase the wrappers through vendors, and keep watching the inventory until the count stays at zero.

## See it live

- **Get a demo.** See software inventory across macOS, Windows, Linux, iOS, and Android: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Manage every platform in one place.** See how Fleet handles [device management](https://fleetdm.com/device-management) across your fleet.
- **Read the reference.** Check what Fleet collects on each platform: [Software inventory reference](https://fleetdm.com/guides/software-inventory-reference)
- **Look past one vendor.** Find every AI tool on your hosts: [Shadow AI is already on your fleet](https://fleetdm.com/articles/shadow-ai-is-already-on-your-fleet)

## Sources

- Federal Register, [Notice of Order Regarding Exclusion and Removal of DeepSeek Products from Federal Civilian Executive Branch Information Technology Systems and Procurements, document 2026-21447](https://www.federalregister.gov/d/2026-21447) ([public inspection PDF](https://public-inspection.federalregister.gov/2026-21447.pdf)). Signed order in docket [DHS-2026-0628](https://www.regulations.gov/docket/DHS-2026-0628).
- Bloomberg Government, [US Moves to Bar China's DeepSeek From Federal Agency Systems](https://news.bgov.com/california-brief/us-moves-to-bar-deepseek-from-federal-agencies-and-contracts).
- The Next Web, [US orders civilian federal agencies to remove DeepSeek from their systems](https://thenextweb.com/news/us-bars-deepseek-federal-agencies).
- NowSecure, [NowSecure Uncovers Multiple Security and Privacy Flaws in DeepSeek iOS Mobile App](https://www.nowsecure.com/blog/2025/02/06/nowsecure-uncovers-multiple-security-and-privacy-flaws-in-deepseek-ios-mobile-app/).
- Ollama, [FAQ: Where are models stored?](https://docs.ollama.com/faq)
- Acquisition.gov, [FAR 52.204-30, Federal Acquisition Supply Chain Security Act Orders—Prohibition (Dec 2023)](https://www.acquisition.gov/far/52.204-30).

<meta name="articleTitle" value="DeepSeek ban for federal agencies: How to prove every device is clean under DHS Order 2026-01">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-10-21">
<meta name="description" value="DHS Order 2026-01 removes DeepSeek and tools built on it from federal systems. Find its apps, extensions, and local models on every device.">
