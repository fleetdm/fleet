# Social media

**Fleet accounts:** 

## Fleet social media directory

| Social media platform | Fleet UserID |
| :--- | :--- |
| [LinkedIn](https://www.linkedin.com/company/fleetdm/) | FleetDM |
| [YouTube](https://www.youtube.com/channel/UCZyoqZ4exJvoibmTKJrQ-dQ) | FleetDM |
| [Twitter / X](https://x.com/fleetctl) | FleetCTL |
| Reddit | FleetDM |
| [Facebook](https://www.facebook.com/fleetdm/) | FleetDM |
| [Instagram](https://www.instagram.com/fleetctl/) | FleetCTL |
| [Threads](https://www.threads.com/@fleetctl) | FleetCTL |
| TikTok | FleetDM |
| [Bluesky](https://bsky.app/profile/fleetdm.bsky.social) | FleetDM |
| [Mastodon](https://mastodon.social/@Fleet@discuss.systems) | Fleet |

[Internal table](https://docs.google.com/spreadsheets/d/1WMiVzHQ-QV_llgDCBRX0P7uxuDNJaooYKPX1XKW6SSc/edit?gid=0#gid=0)

# LinkedIn (Campaign Manager) ads

This page describes how Fleet runs paid LinkedIn advertising, including strategy, campaign structure, budget, rituals, and decision rules through Campaign Manager.

## Strategy

LinkedIn is Fleet's primary paid social channel and is treated as a **brand awareness and engagement play**, not a direct-attribution pipeline channel.

- **Primary metric:** Click-through rate (CTR). CTR is the CEO's proxy for creative resonance.  
- **Vanity metrics:** Impressions and engagement. No one is held accountable for LinkedIn ROI.  
- **Accountability:** The CEO holds people accountable for failing to execute things he considers logically sound or has seen work before, and for failing to act ASAP on post suggestions he drops in \#help-marketing.   
- **Creative freshness:** Ads must reflect Fleet's current positioning and messaging. Exception: an ad older than 6 months may stay if it still has strong CTR/engagement, and still reflects Fleet’s messaging.  
- **Prioritization:** Fix known issues before experimenting with new channels or tactics, e.g., new campaigns being created, different audiences being used

### Target personas

|  | Santas (decision-makers) | Elves (practitioners) |
| :---- | :---- | :---- |
| Example titles | IT Directors, VPs of IT, Head of I&O, Head of Digital Workplace | Sysadmins, Client Platform Engineers |
| Care about | ROI, consolidation, risk reduction, compliance | Workflow, automation, cross-platform, open source |
| LinkedIn audience | santa.it.major.mdm.strict | elf.it.major.mdm.strict |

### Messaging pillars

All creatives must map to one of the 2 active campaign themes as indicated in the [horizons tab]([url](https://docs.google.com/spreadsheets/d/1Hso0LxqwrRVINCyW_n436bNHmoqhoLhC8bcbvLPOs9A/edit?pli=1&gid=2104067010#gid=2104067010)) of the OKRs document.

## Responsibilities

| Role | DRI of... |
| :---- | :---- |
| Content Specialist (Marketing DRI) | Campaign Manager day-to-day, daily health checks, promote/pause/retest decisions, budget pacing, \#help-marketing monitoring, Social Media Manager coordination. |
| Social Media Manager | Daily posting, boosting per calendar or direction, content calendar updates, tagging Fleeties for engagement in \#help-marketing, LinkedIn comment response. |
| CMO | Ad strategy and creative direction, monthly performance review, final call on budget allocation, and major strategy changes. |
| CEO | Saved audience definitions |
| Sales | Triggers ABM campaigns via Salesforce. |
| Ad agency | Executes daily/weekly operations per this page, produces creative, provides weekly reports, and flags anomalies to CMO. |
| Head of Design | Approves new images/designs on creatives posted by the company page before they receive significant spending |

## Campaign structure

Fleet runs three campaign types in LinkedIn Campaign Manager. 

### Evergreen (engagement)

Always-on. Only proven creatives. Engagement objective.

- **Entry requirement:** Every creative must first earn 1%+ CTR (4%+ for non-company page posts) in "Targeting experiments".  
- **Refresh:** quarterly — remove stale, add new proven, align to current messaging.  
- **Budget split:** unaware.elf.it-major-mdm.strict 60%,  unaware.santa.it-major-mdm.strict, unaware.elf.it-gapfiller 20%.

### Targeting experiments

Where everything is tested first.

- **Run length:** 2–3 days.  
- **Default audience:** elf.it.major.mdm.strict. Use santa.it.major.mdm.strict for executive-focused content.  
- **Budget flex:** Borrow from Evergreen if needed. The CEO may direct pushes for high-conviction posts.

**CTR decision thresholds:**

| CTR | Verdict | Action |
| :---- | :---- | :---- |
| < 0.65% | Not performing | Pause. Do not reuse. Never delete — keep the data. |
| 0.65%–2% | Gray zone | Decide with CMO: retest, move to Evergreen, or drop. |
| 2%+ | Strong | Promote to Evergreen (or extend run spend). |
| 10%+ | Exceptional | Keep running as-is in Evergreen until performance stabilizes or the messaging is no longer relevant. |

### Automated campaigns (ABM from Salesforce)

Sales-triggered.

- Generated automatically when sales clicks "ads running" on a Salesforce account.   
- Named with creation date (e.g., elf.it-major-mdm - 2025-10-15 @ 7:51:16 PM CT).  
- Every ABM campaign must have **at least 5 creatives** added manually, selected from current top performers and matched to the account's companies/titles.  
- Keep only the latest 3 ABM campaigns active. Never touch the audience settings, as they're set in Salesforce.

## Budget

| Campaign | % | Flex |
| :---- | :---- | :---- |
| Evergreen | ~50% | Can lend to Experiments |
| Targeting Experiments | ~25–33% | Can borrow from Evergreen |
| ABM | ~17–25% | Fixed |

**Rules:** Experiment overflow comes from Evergreen, never from ABM. Each boosted post should consume 2–3% of the monthly budget over its 2–3 day run. The CEO can override with a larger push.

## Rituals

### Daily (Content Specialist, before 10 AM PST)

1. **ABM check.** Look for new ABM campaigns in Campaign Manager. Add 5+ creatives to any empty one immediately. Confirm only the latest 3 are active and each has daily budget. Pause older ones.  
2. **Evergreen verify.** All three audience segments active and pacing. Flag stopped delivery or abnormal spend. Turn off anything with low CTR.  
3. **Experiments monitor.** Log CTR for any experiments completed in the last 24 hours. Confirm active experiments are pacing and delivering to the right audience.  
4. **Slack #help-marketing.** Process boost requests. Create a marketing board issue, then follow the boosting process. CEO boost or RT requests are executed immediately.  
5. **Scheduled content.** Confirm today's calendar posts went live. Update the calendar with the LinkedIn post link. Remind Fleeties to post about workshops, events, and articles if they haven't.

**Afternoon scan (10–15 min):** Early signals on active experiments (expect at least 24 hours before LinkedIn shows reviewed-and-launched performance data), abnormal spend spikes, and any mid-day ABM triggers.

**Social Media Manager daily:** Post per calendar; update calendar with post links; boost per calendar or direction; tag Fleeties in \#help-marketing for reposts and engagement; respond to LinkedIn comments.

### Weekly (Content Specialist, by Friday EOD)

Pull CTR data for Evergreen (by segment, including audience penetration. If a creative has been shown 10+ times and CTR is declining, pause or rebalance the budget. Targeting Experiments (per experiment), and ABM (spend, delivery, and empty campaigns). Then answer and document:

| # | Question | Action |
| :---- | :---- | :---- |
| 1 | What's working? | Identify 2%+ CTR creatives — candidates for Evergreen. |
| 2 | What's not working? | Pause creatives below 0.65% CTR. |
| 3 | What should we stop? | Pause declining Evergreen or Experiment creatives (never delete). |
| 4 | What should we scale? | Promote high-CTR experiments to Evergreen or increase the budget on strong Evergreen ads. |
| 5 | What should we retest? | Schedule gray zone (0.65–2%) experiments for next week with tweaked audience, copy, or timing. |

Execute the decisions, then share a short Slack or doc summary of what changed, and verify monthly budget pacing.

### Monthly (CMO and Content Specialist; CEO briefed through end of Q2 2026)

Reconcile total spend against the budget. Compare performance by campaign type and by audience segment (elf vs. santa vs. gapfiller). Analyze which creative formats win consistently. Plan a budget for next month's events, launches, and content drops.

### Quarterly (Content Specialist + CMO + CEO for audience review)

Audit every Evergreen creative; remove anything out of date or declining in CTRs, keep what still works if messaging is aligned. Add proven creatives from last quarter's experiments (if haven’t already). Align all messaging to the current Fleet positioning. Request new visuals from the Head of Design as needed. 

## How-to

### How to boost a post

1. Identify the source (content calendar, #help-marketing, CEO directive, or a high-performing organic post).  
2. Select audience: Default to elf.it.major.mdm.strict; use santa.it.major.mdm.strict only for executive-focused content.  
3. Set duration: 2–3 days, starting Tuesday or Thursday. Events, happy hours, and workshops may run longer at the CMO's direction.  
4. Set budget: 2–3% of the monthly budget over the run, or follow CEO's direction for high-conviction.  
5. Launch inside the **Targeting Experiments** campaign. Never boost directly from the company page \- always route through Campaign Manager so performance is tracked.  
6. After the run, apply the CTR decision thresholds. Log the result. Promote to Evergreen or pause, never delete.

### How to promote a creative to Evergreen

1. Confirm the creative hit 2%+ CTR (or 10%+ for exceptional cases) in Targeting Experiments.  
2. Pick the right segment: practitioner content → unaware.elf.it-major-mdm.strict; executive → unaware.santa.it-major-mdm.strict; broad → unaware.elf.it-gapfiller.  
3. Add the creative to that segment. Leave the original paused in Targeting Experiments as a record.  
4. Monitor over the first week (2–3 check-ins) to confirm it holds up at scale.

## Creative types

All visual creatives must be approved by the Head of Design. The CEO-requested creatives route through the CMO for prioritization.

- **Quote cards**: Customer or employee face + Fleet proof point. Lowest effort, highest signal; always in rotation. Sourced from fleetdm.com customer quotes and headshots.  
- **Comparison ads**: Drive to the Fleet vs. Jamf / Intune / Workspace ONE landing pages on fleetdm.com.   
- **Document ads**: White papers and guides delivered as LinkedIn document ads with in-feed lead capture. Each needs a matching landing page.  
- **Organic post boosts**: Strong-performing posts from Fleet or individual Fleeties, boosted via Targeting Experiments.  
- **Thought leader ads**: Text-heavy, face-based leadership ads. Must reflect current positioning; refresh text-heavy ads that don't stand out with visual elements.  
- **Event and workshop promotions**: GitOps workshops, happy hours, conferences. Post-event summaries come from attending Fleeties and from the Social Media Manager (organic post from Fleet's page, then boosted). May run longer than the standard 2–3 day window.

## Guardrails

| Always | Never |
| :---- | :---- |
| Test everything in Targeting Experiments before Evergreen | Put untested creative directly into Evergreen |
| Check ABM daily for empty creative slots | Let ABM campaigns run without creatives |
| Pause creatives below 0.65% CTR quickly | Delete past experiments, campaigns, or data |
| Keep Evergreen aligned to current Fleet messaging | Change saved audiences without CEO design review |
| Boost through Campaign Manager for tracking | Boost directly from the LinkedIn company page |
| Default to elf.it.major.mdm.strict when unsure | Run experiments over weekends |
| Log every experiment result, even failures | Overspend without pulling from Evergreen first |

<meta name="maintainedBy" value="akuthiala">
<meta name="title" value="🫧 Fleet social media">
