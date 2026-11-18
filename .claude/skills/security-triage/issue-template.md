# fleetdm/security issue body

Mirrors `.github/ISSUE_TEMPLATE/security-report.yml` in fleetdm/security. Keep every heading, in this order, even when the answer is "N/A". Fetch the live template if you suspect it changed:

```bash
gh api repos/fleetdm/security/contents/.github/ISSUE_TEMPLATE/security-report.yml -q .content | base64 -d
```

Title: `Report [Critical|High|Medium|Low]: <what the attacker gets, via what>`. For hardening: `Report [Medium]: Hardening — <...>`.

```markdown
### Report source

<one of: GitHub security advisory (GHSA) | Pen test finding (Aikido or annual pen test) | Bug bounty / bugbop | Responsible disclosure (email or other direct contact) | Dependency vulnerability (Dependabot/GHSA/SCA) | Automated scan (Aikido, Claude scan, etc.) | Internal finding (code review, engineering) | Other>

### Link to original report

<Bugbop report URL, advisory URL, or "email from <reporter handle>, <date>, in the security inbox". List every report of the same bug, first reporter first.>

### Reported severity

<Critical | High | Medium | Low | Unknown, as the reporter rated it. Fleet's rating is in the title; if they differ, say why in one line.>

### Summary

<What can an attacker do, what do they need (role, token, device, network position), and what do they get. Two paragraphs at most. No customer names.>

### Attack path / reproduction

1. <Numbered steps from the attacker's starting point to impact. Exact endpoints, parameters, payloads where the report gives them.>

### Affected components

- `<repo-relative file:line>` on `main` @ `<short sha>`: <what it does wrong>
- Affects Fleet <first affected version or "all versions"> through <current release>.

### Verification and duplicate check

- Verified on `main` @ `<short sha>` (<date>): <what you reproduced or read; what you did not>.
- Dedup: searched fleetdm/security, fleetdm/confidential, advisories, and Bugbop for `<terms>`. <"Not a duplicate" | "Same class as #N; new path because ..." | prior fix #N covered X but not Y.>

### Suggested fix

<Remediation options with a recommendation. If it may be by design, state the product question the CTO needs to decide.>

### Original report (verbatim)

<details>
<summary><reporter handle>, <date></summary>

<Full original text, customer-identifying details redacted.>

</details>

### Intake checklist

- [x] Priority label applied (`p<n>`) and product group label (`#g-<group>`)
- [x] Assigned to the group's EM and added to the group board
- [x] Confirmed no real customer names or identifying details appear in this issue (codenames are OK)
- [x] Reporter acknowledged on <channel>
```
