# Classification and severity

Rate in this order. Each step narrows the next.

## 1. Vulnerability or hardening?

Count the preconditions an attacker needs that they don't already have.

| Situation | Classification |
|---|---|
| Unauthenticated, or any role, and the finding lets them do something their access doesn't allow | Vulnerability |
| Needs an enrolled device or an enroll secret, and reaches other hosts or fleets | Vulnerability (often High) |
| Needs a prior primitive first: database read, token theft, log access, a different bug | Hardening. Medium at most, `p3` |
| Needs global admin (or equivalent) and doesn't exceed what that role already can do | Not a vulnerability. Explain and close (precedent: security#19, `contact_url` XSS, closed 2026-08-24) |
| Availability only, or an existence oracle | Hardening unless unauthenticated, trivial, and total |
| Expiry or lifecycle gap (tokens that never expire, sessions that survive a reset) | Hardening, but rate at the top of the band; cheap to fix and compliance-relevant |

A report that doesn't reproduce on `main` is closed as not applicable even when a classification rule would also close it; give the reporter both reasons.

Hardening still gets filed with the same template. Title it `Report [Medium]: Hardening — ...` so engineers can tell at a glance.

## 2. Severity

Fleet uses CVSS 3.1 as the vocabulary and the table below as the decision. When the vector and the table disagree, the table wins and the issue says why.

| Severity | Typical shape | Examples from Fleet's history |
|---|---|---|
| Critical | Unauthenticated, or any user, to global admin or to code execution on hosts | SAML session store takeover; SQL injection in an unauthenticated endpoint |
| High | Cross-fleet or cross-host takeover of identity, commands, or secrets with a credential the attacker can plausibly hold | Enrollment takeover via a device-reported identifier; team admin escalating to global admin; observer reading enroll secrets |
| Medium | Information disclosure beyond role, authenticated denial of service, secrets in places lower roles can read | Command results exposing enroll secrets; crafted installer crashing the server |
| Low | Existence oracles, client-side CSV formula injection, logging of data the caller could already see | 404-vs-403 host existence; CSV export cells |

Set the CVSS vector on the advisory to match the row. Don't inherit the reporter's.

## 3. Priority label

| Severity | Label | Handbook target |
|---|---|---|
| Emergency | `p0` | Active exploit patch immediately |
| Critical | `p1` | Patch release as soon as the fix is ready |
| High | `p2` | Fix merged within 2 to 3 weeks, next patch release |
| Medium | `p3` | Next minor release |
| Low | `p4` | Next minor release, or accepted risk with a written reason |


The label and the assignee are what compliance tracking reads. An issue missing either is invisible to the SLA clock and to the daily digest.

## 4. Things that change the rating

- **Customer-reported, or affecting a named customer**: same severity, but tell the CTO the same day and keep customer context in `fleetdm/confidential`.
- **Already mitigated by an opt-in setting**: still a vulnerability on the default path. Say so in the issue; the fix decision is the CTO's.
- **Fixed and released before the report arrived**: a duplicate if a tracker exists (security issue, advisory, or running list row). If nothing tracks it, file it and close it in the same step with the fixing PR and first release, so the running list picks it up. Neither case starts a new SLA clock.
- **Fixed on `main` but unreleased**: file it anyway if no tracker exists, milestone it to the release that carries the fix, and close it at the same time with the PR link so the advisory running list picks it up.
- **Reporter disagrees with the rating**: explain once, with the precondition chain. If they push back again, hand it to the CTO rather than re-arguing.
