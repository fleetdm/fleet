# 🧭 Product maturity assessment

Fleet provides comprehensive device management across the entire device lifecycle. Some stages and features are more mature than others. To convey the state of our feature set and be transparent with our customers, we use a maturity framework for categories and stages.

## Maturity legend

**Category maturity**

- 🥚 **Planned**: Not yet implemented in Fleet, but on our roadmap
- 🐣 **Minimal**: A minimal foundation so people can see where we're going and to validate customer need
- 🐥 **Viable**: Used by customers to solve real production problems
- 🦆 **Complete**: Contains a competitive feature set sufficient to meet enterprise requirements and displace a device management competitor
- 🦢 **Lovable**: Provides an elevated experience that customers love

**Stage lifecycle**

- Not yet available
- Used internally at Fleet
- Majority of users are early adopters
- Majority of users are production customers
- Usable for most Fleet users
- Users of competing tools start to switch
- Best product in the market

---

## Device lifecycle stages

### Enroll

**Stage lifecycle**: Users of competing tools start to switch

**Analyst reports**

- [Add any relevant analyst reports]

| Category | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| [DEP/ABM enrollment](https://fleetdm.com/docs/using-fleet/mdm-macos-setup#dep) (Apple) | 🦆 **Complete** |  |  |  |  |
| ASM enrollment (Apple) | 🥚 **Planned** |  |  |  |  |
| [Windows enrollment](https://fleetdm.com/docs/using-fleet/mdm-windows-setup) | 🐥 **Viable** |  |  |  |  |
| Windows Autopilot | 🦆 **Complete** |  |  |  |  |
| Work Profile enrollment (Android) | 🐣 **Minimal** |  |  |  |  |
| Automatic Device Enrollment (Android) | 🥚 **Planned** |  |  |  |  |
| [Linux enrollment](https://fleetdm.com/docs/using-fleet/adding-hosts) | 🐥 **Viable** |  |  |  |  |
| [iOS/iPadOS profile-based enrollment](https://fleetdm.com/docs/using-fleet/mdm-ios-setup) | 🦆 **Complete** |  |  |  |  |
| Account Driven User Enrollment (Apple) | 🦆 **Complete** |  |  |  |  |
| Account Driven Device Enrollment (Apple) | 🥚 **Planned** |  |  |  |  |
| ChromeOS enrollment | 🐥 **Viable** |  |  |  |  |

---

### Configure

**Stage lifecycle**: [e.g., Majority of users are production customers (year 3)]

**Analyst reports**

- [Add any relevant analyst reports]

| Category | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| [Setup experience (macOS)](https://fleetdm.com/docs/using-fleet/macos-setup-experience) | 🦆 **Complete** |  |  |  |  |
| Setup experience (Windows) | 🐣 **Minimal** |  |  |  |  |
| Setup experience (Linux) | 🐣 **Minimal** |  |  |  |  |
| Configuration [Profiles (macOS)](https://fleetdm.com/docs/using-fleet/mdm-macos-profile) | 🐥 **Viable** |  |  |  |  |
| Configuration [Profiles (iOS/iPadOS)](https://fleetdm.com/docs/using-fleet/mdm-ios-setup#configuration-profiles) | 🐥 **Viable** |  |  |  |  |
| Configuration Profiles (tvOS/VisionOS/watchOS) | 🥚 **Planned** |  |  |  |  |
| Configuration Profiles (Windows) | 🐣 **Minimal** |  |  |  |  |
| Configuration Profiles (Android) | 🐣 **Minimal** |  |  |  |  |
| [Remote script execution](https://fleetdm.com/docs/using-fleet/run-scripts) | 🐥 **Viable** |  |  |  |  |
| [Software deployment](https://fleetdm.com/docs/using-fleet/software) | 🐥 **Viable** |  |  |  |  |
| [App Store app management](https://fleetdm.com/docs/using-fleet/mdm-app-deployment) | 🐥 **Viable** |  |  |  |  |
| [Custom package deployment](https://fleetdm.com/docs/using-fleet/software#custom-packages) | 🐥 **Viable** |  |  |  |  |
| Fleet-maintained apps | 🐣 **Minimal** |  |  |  |  |
| [FileVault](https://fleetdm.com/docs/using-fleet/mdm-disk-encryption#macos-filevault) management | 🐥 **Viable** |  |  |  |  |
| [BitLocker](https://fleetdm.com/docs/using-fleet/mdm-disk-encryption#windows-bitlocker) management | 🐥 **Viable** |  |  |  |  |
| LUKS management | 🐥 **Viable** |  |  |  |  |
| [Certificate management](https://fleetdm.com/guides/ndes-scep-proxy) | 🐥 **Viable** |  |  |  |  |

---

### Secure

**Stage lifecycle**: [e.g., Usable for most Fleet users (year 4)]

**Analyst reports**

- [Add any relevant analyst reports]

| Category | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| [Vulnerability detection](https://fleetdm.com/docs/using-fleet/vulnerability-processing) | 🐥 **Viable** |  |  |  |  |
| [Policy automation](https://fleetdm.com/docs/using-fleet/policies) | 🦆 **Complete** |  |  |  |  |
| Binary authorization | 🐣 **Minimal** |  |  |  |  |
| [CIS Benchmark checks](https://fleetdm.com/docs/using-fleet/policies#cis-benchmarks) | 🐥 **Viable** |  |  |  |  |
| [Custom security policies](https://fleetdm.com/docs/using-fleet/policies) | 🦆 **Complete** |  |  |  |  |
| [Threat detection](http://link) | 🐥 **Viable** |  |  |  |  |
| [Zero Trust integration](http://link) | 🐣 **Minimal** |  |  |  |  |
| [Conditional Access](http://link) | 🐣 **Minimal** |  |  |  |  |

---

### Monitor

**Stage lifecycle**: [e.g., Users of competing tools start to switch (year 5)]

**Analyst reports**

- [Add any relevant analyst reports]

| Category | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| [Live query execution](https://fleetdm.com/docs/using-fleet/live-queries) | 🦢 **Lovable** |  |  |  |  |
| [Scheduled queries](https://fleetdm.com/docs/using-fleet/scheduled-queries) | 🦢 **Lovable** |  |  |  |  |
| [Software inventory](https://fleetdm.com/docs/using-fleet/software-inventory) | 🦢 **Lovable** |  |  |  |  |
| [Hardware inventory](https://fleetdm.com/docs/using-fleet/host-details) | 🦆 **Complete** |  |  |  |  |
| [Device status monitoring](http://link) | 🦆 **Complete** |  |  |  |  |
| [Geolocation tracking](http://link) | 🐣 **Minimal** |  |  |  |  |
| [Activity feed](http://link) | 🐥 **Viable** |  |  |  |  |
| [Audit logs](https://fleetdm.com/docs/using-fleet/audit-logging) | 🐥 **Viable** |  |  |  |  |
| [Custom dashboards](http://link) | 🥚 **Planned** |  |  |  |  |
| [Real-time alerts](http://link) |  |  |  |  |  |
| [Historical data analysis](http://link) |  |  |  |  |  |
| [Compliance reporting](http://link) | 🦆 **Complete** |  |  |  |  |

---

### Maintain

**Stage lifecycle**: [e.g., Majority of users are early adopters (year 2)]

**Analyst reports**

- [Add any relevant analyst reports]

| Category | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| [OS update management](http://link) (macOS) | 🐥 **Viable** |  |  |  |  |
| [OS update management](http://link) (iPhone/iPadOS) | 🐥 **Viable** |  |  |  |  |
| [OS update management](http://link) (tvOS/visionOS/watchOS) | 🥚 **Planned** |  |  |  |  |
| [OS update management](http://link) (Windows) | 🐣 **Minimal** |  |  |  |  |
| [OS update management](http://link) (Linux) | 🥚 **Planned** |  |  |  |  |
| [OS update management](http://link) (Android) | 🥚 **Planned** |  |  |  |  |
| [Patch management](http://link) | 🐣 **Minimal** |  |  |  |  |
| [Remote lock](http://link) | 🦆 **Complete** |  |  |  |  |
| [Remote restart](http://link) | 🐣 **Minimal** |  |  |  |  |
| [Remote support tools](http://link) | 🥚 **Planned** |  |  |  |  |
| [Self-service portal](http://link) | 🐥 **Viable** |  |  |  |  |
| [Device health checks](http://link) | 🦆 **Complete** |  |  |  |  |
| [Maintenance windows](http://link) | 🐣 **Minimal** |  |  |  |  |
| [Ticket integration](http://link) | 🐥 **Viable** |  |  |  |  |
| [End user communications](http://link) | 🥚 **Planned** |  |  |  |  |

---

### Offboard

**Stage lifecycle**: [e.g., Used internally at Fleet (year 1)]

**Analyst reports**

- [Add any relevant analyst reports]

| Category | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| [Remote wipe](http://link) | 🦆 **Complete** |  |  |  |  |
| [Device unenrollment](http://link) |  |  |  |  |  |
| [Transfer ownership](http://link) |  |  |  |  |  |
| [Reassignment workflows](http://link) |  |  |  |  |  |
| [Offboarding audit trail](http://link) |  |  |  |  |  |
| [Lock lost/stolen devices](http://link) | 🐣 **Minimal** |  |  |  |  |
| [Activation lock management](http://link) |  |  |  |  |  |

---

## Cross-cutting stages

### Platform support

**Stage lifecycle**: [varies by platform]

| Platform | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| [macOS](http://link) | 🦆 **Complete** |  |  |  |  |
| [Windows](http://link) | 🐥 **Viable** |  |  |  |  |
| [Linux (Ubuntu)](http://link) | 🦆 **Complete** |  |  |  |  |
| [Linux (RHEL)](http://link) | 🦆 **Complete** |  |  |  |  |
| [Linux (Debian)](http://link) | 🐥 **Viable** |  |  |  |  |
| Linux (Arch) | 🐥 **Viable** |  |  |  |  |
| Linux (SUSE) | 🐥 **Viable** |  |  |  |  |
| Android | 🐣 **Minimal** |  |  |  |  |
| tvOS/visionOS/watchOS | 🥚 **Planned** |  |  |  |  |
| [iOS/iPadOS](http://link) | 🐥 **Viable** |  |  |  |  |
| [ChromeOS](http://link) | 🦆 **Complete** |  |  |  |  |

---

### Integrate

**Stage lifecycle**: [e.g., Usable for most Fleet users (year 4)]

| Category | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| [REST API](http://link) | 🦢 **Lovable** |  |  |  |  |
| [Webhooks](http://link) | 🐥 **Viable** |  |  |  |  |
| [SSO/SAML](http://link) | 🦆 **Complete** |  |  |  |  |
| [Google Workspace Calendar](http://link) | 🦆 **Complete** |  |  |  |  |
| [Slack integration](http://link) |  |  |  |  |  |
| [Jira integration](http://link) |  |  |  |  |  |
| [Zendesk integration](http://link) |  |  |  |  |  |
| [Splunk integration](http://link) |  |  |  |  |  |
| [Datadog integration](http://link) |  |  |  |  |  |
| [Terraform provider](http://link) |  |  |  |  |  |
| [Zapier](http://link) |  |  |  |  |  |
| [GitOps support](http://link) | 🦢 **Lovable** |  |  |  |  |
| ServiceNow integration |  |  |  |  |  |

---

### Operate

**Stage lifecycle**: [e.g., Usable for most Fleet users (year 4)]

| Category | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| [Self-managed deployment](http://link) | 🦆 **Complete** |  |  |  |  |
| [Fleet cloud](http://link) | 🦆 **Complete** |  |  |  |  |
| [Docker deployment](http://link) |  |  |  |  |  |
| [Kubernetes deployment](http://link) |  |  |  |  |  |
| [High availability](http://link) |  |  |  |  |  |
| [Auto-scaling](http://link) |  |  |  |  |  |
| [Performance monitoring](http://link) |  |  |  |  |  |
| [Disaster recovery](http://link) |  |  |  |  |  |
| [Backup & Restore](http://link) |  |  |  |  |  |
| [Multi-region support](http://link) |  |  |  |  |  |
| [Multi-tenancy](http://link) |  |  |  |  |  |

---

## Planned category maturity

The maturity framework makes it easy to visualize where Fleet is making investments and resulting category maturity improvements. As part of the planning process for each category, the set of features required and expected date to reach the next maturity is maintained.

---

## How to fill out and maintain this page

Use this guide to keep assessments consistent and up to date. Updates are typically made quarterly and when major features ship.

### Maturity levels (per category)

- 🥚 Planned: Not yet implemented, but on Fleet's roadmap
- 🐣 Minimal: Basic foundation, validating customer need
- 🐥 Viable: Used by customers in production to solve real problems
- 🦆 Complete: Competitive feature set that can replace competitors
- 🦢 Lovable: Elevated experience that customers love (e.g., NPS/surveys)

When deciding a category's maturity, ask:

1. Is it shipped? If not, it's Planned
2. Is it used in production? If yes, at least Viable
3. Does it match leading competitors? If yes, Complete
4. Do customers praise the experience? If yes, Lovable

### Stage lifecycle (per stage)

Choose the best description for the stage overall:

- Not yet available (year 0)
- Used internally at Fleet (year 1)
- Majority of users are early adopters (year 2)
- Majority of users are production customers (year 3)
- Usable for most Fleet users (year 4)
- Users of competing tools start to switch (year 5)
- Entry point for new customers (year 6)
- Best product in the market (year 7)

Replace placeholders like "[e.g., Users of competing tools start to switch]" with the current assessment and year.

### What to include in each stage section

1. Stage lifecycle: Replace the placeholder with the current stage-level assessment
2. Analyst reports: Add any relevant mentions (optional)
3. On our roadmap: List planned features that map to Planned categories (optional)
4. Category maturity table: For each category row, set Current and projections for future quarters (Q1–Q4)

Example row transformation:

- Before: `| [DEP/ABM enrollment](link) |  |  |  |  |  |`
- After:  `| 🐥 [DEP/ABM enrollment](https://fleetdm.com/docs/using-fleet/mdm-setup#dep) | 🦆 | 🦆 | 🦢 | 🦢 | 🦢 |`

This indicates: Current is Viable, targeting Complete then Lovable over time.

### Tips for projections

- Be realistic; don't overpromise
- Show progress; gradually advance maturity levels
- Consider dependencies; some categories need others to mature first
- Align with Fleet's public roadmap and release plans
- Not everything must advance each quarter

Common patterns:

- Rapid maturation: 🥚 → 🐣 → 🐥 → 🦆 → 🦢
- Steady improvement: 🐥 → 🐥 → 🦆 → 🦆 → 🦢
- Maintenance mode: 🦆 → 🦆 → 🦆 → 🦢 → 🦢
- Already excellent: 🦢 → 🦢 → 🦢 → 🦢 → 🦢

### Quarterly review checklist

1. Update velocity metrics (last 3 months)
2. Advance categories that met goals; adjust projections as needed
3. Move shipped features from "On our roadmap" to "Since YYYY Fleet added"
4. Update stage lifecycle if overall maturity improved
5. Update links and replace any `(link)` placeholders
6. Update the "Last updated" date in your PR description

### Governance

- Internal review: Product design and engineering to validate assessments
- Consistency: Ensure projections align with public roadmap and release plans
- Transparency: Avoid commitments that create legal obligations; treat projections as targets

---

<meta name="maintainedBy" value="allenhouchins">
<meta name="title" value="🧭 Product maturity assessment">


