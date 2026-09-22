<meta name="category" value="industry news">
<meta name="articleTitle" value="BigDiskBuster turns off Windows Defender's updates without turning off Windows Defender">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-09-22">
<meta name="description" value="A new zero-day blocks Defender's signature updates while leaving it looking healthy. See how to check real update status, not just on/off, with Fleet.">

# BigDiskBuster turns off Windows Defender's updates without turning off Windows Defender

*A new zero-day leaves Windows Defender running and reporting healthy while quietly blocking every signature and platform update behind it. Here's how to tell which of your Windows hosts are actually current.*

## Key takeaways

- **You can check real update status instead of trusting that Defender is running.** Fleet's agent already reports Defender's product state and a dedicated up-to-date flag for its signatures on every Windows host, so you can separate "installed and enabled" from "actually current."
- **BigDiskBuster doesn't disable Defender, it starves it.** The antivirus keeps running and reporting itself as active; the exploit just blocks the update pipeline underneath it, so a status check that only asks "is Defender on" won't catch this.
- **This is the same researcher's second Defender update-blocking exploit this year.** Abdelhamid Naceri disclosed a similar flaw, UnDefend, in April 2026. BigDiskBuster works through a different mechanism but produces the same outcome: an endpoint stuck on old signatures.
- **Microsoft hasn't shipped a fix, and the technique already works broadly.** Naceri says the current proof of concept is still buggy, but he says it works across supported Windows versions, so a patch isn't something to wait on before checking exposure.
- **Whether signatures are current is a fact you can query, not a guess you make.** Fleet's agent surfaces a dedicated up-to-date flag for Defender, so "did this month's update actually reach every endpoint" becomes an answer instead of an assumption.
- **A one-time check misses the hosts that go stale next week.** Save the query as a Fleet policy and any host that falls behind, from this exploit, a stuck update service, or anything else, gets flagged the moment it happens.

<a purpose="cta-button" href="https://fleetdm.com/security-and-control">See continuous compliance in Fleet</a>

Security researcher Abdelhamid Naceri released a proof of concept called BigDiskBuster that blocks Windows Defender from downloading new platform and signature updates, while leaving the antivirus itself running. He describes it as working across every supported version of Windows, and it follows a similar exploit, UnDefend, that he disclosed in April. Microsoft has not announced a patch.

That combination, no fix and no visible symptom, is what makes this worth checking for now rather than after the next incident. A host running BigDiskBuster still shows Defender as installed and active. What it stops showing you is whether Defender has actually seen this week's threat signatures, and that's the number that determines whether it can catch anything new.

## Why "Defender is on" isn't the same question as "Defender is current"

Most antivirus status checks stop at a single question: is real-time protection enabled? BigDiskBuster is built around the gap right behind that question. It doesn't touch Defender's on/off state or its ability to scan with whatever signatures it already has. It targets the update mechanism itself, so the antivirus keeps running, keeps reporting healthy, and keeps quietly falling further behind on what it can detect.

Naceri has now demonstrated this twice in one year. UnDefend, disclosed in April, let a standard user block definition updates. BigDiskBuster gets to the same result through a different path. Two independent routes to the same outcome, in the same year, from the same researcher, is a signal that "block the updates, leave the antivirus visible" is a durable technique, not a one-off bug that a single patch closes for good.

## Checking signature freshness across your Windows fleet

Fleet's agent already reports Defender's status on every Windows host it manages, which means the check here isn't "wait for Microsoft," it's "look at what you already have." The `windows_security_products` table Fleet queries carries a `signatures_up_to_date` flag alongside the product's `state`, so a live query answers the exact question BigDiskBuster is built to hide the answer to:

```sql
SELECT name, state, signatures_up_to_date, state_timestamp
FROM windows_security_products
WHERE type = 'Antivirus';
```

A host where `state` still reads healthy but `signatures_up_to_date` has flipped to 0 is exactly the pattern BigDiskBuster produces. It won't show up as "Defender disabled" in a simple health check, because Defender isn't disabled. It shows up as a signatures flag going stale underneath a status that still looks fine, which is why checking that flag directly matters more than checking whether real-time protection is on.

## Make it a standing check, not a one-time sweep

A single query answers "who's stale today." It doesn't answer "who goes stale next week," and that's the more useful question for an exploit that runs quietly in the background for as long as an attacker wants it to. Saving the signature-freshness check as a Fleet policy means every host that falls behind, whether from this specific exploit, a stuck update service, or a misconfiguration, gets flagged automatically as soon as it happens, without anyone re-running the hunt.

Because Fleet policies live in Git as YAML and deploy through the same GitOps workflow as everything else, tightening the freshness threshold once Microsoft ships a fix, or once you have a clearer picture of what "current" should mean for your fleet, is a reviewable pull request rather than an undocumented console change.

## Don't wait on the patch to check exposure

BigDiskBuster is still a rough proof of concept, and Microsoft may close it with a routine update. But the pattern it demonstrates, an antivirus that looks fine while it quietly stops learning about new threats, doesn't go away with one patch. The way to stay ahead of it is the same either way: stop asking whether Defender is on, and start asking whether it's current, across every Windows host, on a schedule that doesn't depend on someone remembering to check.

## See it live

- **Enforce and verify OS and security updates** across your fleet: [fleetdm.com/guides/enforce-os-updates](https://fleetdm.com/guides/enforce-os-updates)
- **Get a demo** to see Defender status and signature freshness checks against your own fleet: [fleetdm.com/contact](https://fleetdm.com/contact)

## Sources

- BleepingComputer, [New Windows Defender zero-day blocks Microsoft antivirus updates](https://www.bleepingcomputer.com/news/security/new-windows-defender-zero-day-blocks-microsoft-antivirus-updates/).
- SecurityWeek, [Nightmare Eclipse Drops New Microsoft Defender Exploit After Revealing Identity](https://www.securityweek.com/nightmare-eclipse-drops-new-microsoft-defender-exploit-after-revealing-identity/).
