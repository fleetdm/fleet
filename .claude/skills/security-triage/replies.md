# Reporter replies

Plain, short, specific. Say what we checked, what we found, and what happens next. No promises about CVEs, dates, or bounties. No details about other reporters or other reports. Every reply goes on the channel the reporter used (Bugbop comment, advisory comment, or email reply).

Replace `<...>` before sending. Read it once more after.

## Acknowledge (within one business day, before validation)

> Thanks for the report. We've received it and are reviewing it now. We'll update you here once we've confirmed the behavior and rated it.

## Needs more information

> Thanks for the report. We tried to reproduce this on current `main` and couldn't: <one sentence on what we saw instead>. Could you send the Fleet version you tested, the exact request and response (secrets redacted), and how the user or device in your test was set up? We'll pick this back up as soon as we have that.

If no answer in 7 days, one nudge. If no answer 7 days after that, close with the "not applicable" reply and say they can reopen with a working reproduction.

## Validated

> Thanks again for this report. We've confirmed it and rated it <Severity>: <one sentence on the precondition chain, e.g. "it requires a valid fleet observer account and discloses host IDs in other fleets">. It's tracked internally and assigned for a fix. We'll update you here when the fix merges and again when it ships, naming the release. If you'd like credit on the advisory, tell us the GitHub username to list; we also send swag to reporters at fleetdm.com/swag.

If Fleet's severity is lower than the reporter's, add one sentence: "We rated this <lower> rather than <theirs> because <precondition>."

## Not applicable

> Thanks for the report. We looked into it and this isn't a vulnerability in Fleet as described: <one or two sentences with the specific reason, e.g. "the endpoint requires a valid API token, and an unauthenticated request receives the same response whether or not the identifier exists">. We're closing this one. If you can show the behavior against a current release, reply with the request and response and we'll take another look.

## Duplicate

> Thanks for the report. This was first reported to us on <date> and is already tracked for a fix. We credit the first reporter on the advisory, so we're closing this as a duplicate. We appreciate the independent confirmation.

Across Bugbop programs the duplicate can't be linked. Use the same text, leave the report `open`, and record the canonical report in the internal note.

## Fix merged (not yet shipped)

> Update: the fix for this has merged and is scheduled for Fleet <version>. We'll confirm here when that release is out.

## Fix shipped

> This is fixed in Fleet <version>, released <date>. Thank you for the report. <If an advisory is planned: "We'll publish an advisory crediting you after our standard delay, and we'll link it here.">

Set the Bugbop status to `fixed` with this reply, not before.

## Advisory published

> The advisory for this report is now public: <GHSA URL>, with you credited as <login or name>. <If a CVE was requested: "A CVE has been requested from GitHub and will attach to the advisory when assigned.">

## Reporter asks for a CVE, a disclosure date, or a bounty

> We request CVEs when we publish the advisory, which happens after the fix has shipped and had time to roll out. We can't commit to a date before then. <VDP: "Our disclosure program doesn't pay cash bounties; we credit reporters on the advisory and send swag.">

## "Any update?" with nothing new to say

> Still tracked and assigned; no change since our last update. We'll post here the moment the fix merges.

Never leave an "any update?" unanswered for more than two business days.
