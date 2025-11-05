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

- Early Development (most categories Planned/Minimal)
- Core Capabilities Available (key platforms/categories Viable)
- Production Ready (majority Viable/Complete)
- Enterprise Ready (mostly Complete, competitive feature set)
- Market Competitive (Complete across all major use cases)
- Market Leading (Lovable in key areas, Complete elsewhere)

---

## Device lifecycle stages

### Enroll

**Stage lifecycle**: Enterprise Ready

**Analyst reports**

- [Add any relevant analyst reports]

| Category | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| [DEP/ABM enrollment](https://fleetdm.com/docs/using-fleet/mdm-macos-setup#dep) (Apple) | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| ASM enrollment (Apple) | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** |
| [Windows enrollment](https://fleetdm.com/docs/using-fleet/mdm-windows-setup) | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| Windows Autopilot | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Work Profile enrollment (Android) | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** |
| Automatic Device Enrollment (Android) | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** |
| [Linux enrollment](https://fleetdm.com/docs/using-fleet/adding-hosts) | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| [iOS/iPadOS profile-based enrollment](https://fleetdm.com/docs/using-fleet/mdm-ios-setup) | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Account Driven User Enrollment (Apple) | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Account Driven Device Enrollment (Apple) | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** |
| ChromeOS enrollment | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |

---

### Configure

**Stage lifecycle**: Production Ready

**Analyst reports**

- [Add any relevant analyst reports]

| Category | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| [Setup experience (macOS)](https://fleetdm.com/docs/using-fleet/macos-setup-experience) | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Setup experience (Windows) | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** |
| Setup experience (Linux) | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** |
| Configuration [Profiles (macOS)](https://fleetdm.com/docs/using-fleet/mdm-macos-profile) | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| Configuration [Profiles (iOS/iPadOS)](https://fleetdm.com/docs/using-fleet/mdm-ios-setup#configuration-profiles) | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| Configuration Profiles (tvOS/VisionOS/watchOS) | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** |
| Configuration Profiles (Windows) | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** |
| Configuration Profiles (Android) | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** |
| [Remote script execution](https://fleetdm.com/docs/using-fleet/run-scripts) | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| [Software deployment](https://fleetdm.com/docs/using-fleet/software) | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| [App Store app management](https://fleetdm.com/docs/using-fleet/mdm-app-deployment) | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| [Custom package deployment](https://fleetdm.com/docs/using-fleet/software#custom-packages) | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| Fleet-maintained apps | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** |
| [FileVault](https://fleetdm.com/docs/using-fleet/mdm-disk-encryption#macos-filevault) management | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| [BitLocker](https://fleetdm.com/docs/using-fleet/mdm-disk-encryption#windows-bitlocker) management | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| LUKS management | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| [Certificate management](https://fleetdm.com/guides/ndes-scep-proxy) | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |

---

### Secure

**Stage lifecycle**: Production Ready

**Analyst reports**

- [Add any relevant analyst reports]

| Category | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| [Vulnerability detection](https://fleetdm.com/docs/using-fleet/vulnerability-processing) | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| [Policy automation](https://fleetdm.com/docs/using-fleet/policies) | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Binary authorization | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** |
| [CIS Benchmark checks](https://fleetdm.com/docs/using-fleet/policies#cis-benchmarks) | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| [Custom security policies](https://fleetdm.com/docs/using-fleet/policies) | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Threat detection | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| Zero Trust integration | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** |
| Conditional Access | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** |

---

### Monitor

**Stage lifecycle**: Market Competitive

**Analyst reports**

- [Add any relevant analyst reports]

| Category | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| [Live query execution](https://fleetdm.com/docs/using-fleet/live-queries) | 🦢 **Lovable** | 🦢 **Lovable** | 🦢 **Lovable** | 🦢 **Lovable** | 🦢 **Lovable** |
| [Scheduled queries](https://fleetdm.com/docs/using-fleet/scheduled-queries) | 🦢 **Lovable** | 🦢 **Lovable** | 🦢 **Lovable** | 🦢 **Lovable** | 🦢 **Lovable** |
| [Software inventory](https://fleetdm.com/docs/using-fleet/software-inventory) | 🦢 **Lovable** | 🦢 **Lovable** | 🦢 **Lovable** | 🦢 **Lovable** | 🦢 **Lovable** |
| [Hardware inventory](https://fleetdm.com/docs/using-fleet/host-details) | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Device status monitoring | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Geolocation tracking | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** |
| Activity feed | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| [Audit logs](https://fleetdm.com/docs/using-fleet/audit-logging) | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| Custom dashboards | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** |
| Real-time alerts | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| Historical data analysis | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| Compliance reporting | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |

---

### Maintain

**Stage lifecycle**: Production Ready

**Analyst reports**

- [Add any relevant analyst reports]

| Category | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| OS update management (macOS) | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| OS update management (iPhone/iPadOS) | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| OS update management (tvOS/visionOS/watchOS) | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** |
| OS update management (Windows) | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** |
| OS update management (Linux) | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** |
| OS update management (Android) | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** |
| Patch management | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** |
| Remote lock | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Remote restart | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** |
| Remote support tools | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** |
| Self-service portal | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| Device health checks | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Maintenance windows | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** |
| Ticket integration | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| End user communications | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** |

---

### Offboard

**Stage lifecycle**: Core Capabilities Available

**Analyst reports**

- [Add any relevant analyst reports]

| Category | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| Remote wipe | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Device unenrollment | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| Offboarding audit trail | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** |
| Lock lost/stolen devices | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** |
| Activation lock management | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** |

---

## Cross-cutting stages

### Platform support

**Stage lifecycle**: Varies by platform (see individual platform rows)

| Platform | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| macOS | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Windows | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| Linux (Ubuntu) | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Linux (RHEL) | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Linux (Debian) | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| Linux (Arch) | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| Linux (SUSE) | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| Android | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** | 🐣 **Minimal** |
| tvOS/visionOS/watchOS | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** |
| iOS/iPadOS | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| ChromeOS | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |

---

### Integrate

**Stage lifecycle**: Market Competitive

| Category | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| REST API | 🦢 **Lovable** | 🦢 **Lovable** | 🦢 **Lovable** | 🦢 **Lovable** | 🦢 **Lovable** |
| Webhooks | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| SSO/SAML | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Google Workspace Calendar | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Slack integration | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| Jira integration | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| Zendesk integration | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Splunk integration | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** | 🐥 **Viable** |
| Terraform provider | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| GitOps support | 🦢 **Lovable** | 🦢 **Lovable** | 🦢 **Lovable** | 🦢 **Lovable** | 🦢 **Lovable** |
| ServiceNow integration | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** |

---

### Operate

**Stage lifecycle**: Market Competitive

| Category | Current | Q1 2026 | Q2 2026 | Q3 2026 | Q4 2026 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| Self-managed deployment | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Fleet cloud | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Docker deployment | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Kubernetes deployment | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** | 🦆 **Complete** |
| Multi-tenancy | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** | 🥚 **Planned** |

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

Choose the best description for the stage overall based on the mix of category maturities:

- Early Development (most categories Planned/Minimal)
- Core Capabilities Available (key platforms/categories Viable)
- Production Ready (majority Viable/Complete)
- Enterprise Ready (mostly Complete, competitive feature set)
- Market Competitive (Complete across all major use cases)
- Market Leading (Lovable in key areas, Complete elsewhere)

Replace placeholders with the current assessment. Look at the overall mix of category maturities in the stage to determine the appropriate lifecycle stage.

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


