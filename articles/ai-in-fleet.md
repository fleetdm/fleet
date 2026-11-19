# AI in Fleet

<!--
This page is required evidence for an ISO 42001 control. Do not delete, rename, or move it, because its URL is part of the evidence. Run all edits past the CIO before merging.
-->

Fleet uses AI to help IT and security teams work faster. In every AI feature, people stay in control: AI drafts and answers, and your team decides what happens on your devices. This page lists each AI feature in Fleet, what data it sends and to whom, how to turn it off, and how to report a concern.

## Your control

Each AI feature has a documented default state, listed in [AI features](#ai-features) below. Features that send your data to a model provider engaged by Fleet run only when a user chooses to use them.

Turning off AI features does not remove any other Fleet functionality.

### Turn off AI features

Administrators can turn off AI-assisted features for the whole organization.

In the Fleet UI:

1. Go to **Settings** > **Organization settings** > **Advanced options**.
2. Under **Features**, clear **Generative AI**.
3. Select **Save**.

With [GitOps](https://fleetdm.com/docs/configuration/yaml-files#server-settings), set `ai_features_disabled` to `true` in `org_settings`:

```yaml
org_settings:
  server_settings:
    ai_features_disabled: true
```

## Data and privacy

Fleet does not use your data to train AI models, and our model provider does not use it to train its models.

Each feature below lists exactly what data is sent and to whom.

- Fleet's sub-processors are listed at [trust.fleetdm.com](https://trust.fleetdm.com).
- Read our [Privacy Policy](https://fleetdm.com/legal/privacy).

## AI features

### AI-assisted policy descriptions and resolutions

| Item | Detail |
| :--- | :--- |
| What it does | Drafts a description of what a policy checks, and a resolution telling device users how to fix a failing device, from the policy's SQL. |
| How to use it | When creating or editing a policy, select **Autofill** next to **Description** or **Resolution**, then review and edit the draft before saving. Only users who can write policies can use it. |
| How to override or reject it | Edit the drafts, clear them, or close the form without saving. Nothing is saved until you save the policy. |
| Default | Available, and runs only when you select **Autofill**. Turn it off for your organization with `server_settings.ai_features_disabled: true`. The **Autofill** buttons are then disabled. |
| Data sent | The policy's SQL only. Your Fleet server sends it to fleetdm.com, which sends it to Anthropic's Claude Haiku 4.5 model. This applies to Fleet Cloud and self-hosted Fleet. Do not put personal data or secrets in policy SQL. |
| Human oversight | You must review the drafts. Saved text can be shown to device users in Fleet Desktop and in calendar events, so check that the resolution steps are correct and safe for your environment. |
| Accuracy | Drafts are suggestions and can be wrong or incomplete. Fleet tests the feature against a set of reference policies before every model or prompt change. |
| Limitations | May misread complex SQL. Drafts are short. The resolution may describe what IT will do rather than steps the device user can take. Drafts may not fit your own processes or support channels. English only. Your Fleet server must reach fleetdm.com, so autofill does not work in air-gapped deployments. |
| Requirements | Fleet 4.50.0 or later, Fleet Free or Fleet Premium. Your Fleet server must be able to reach fleetdm.com over HTTPS. Calendar events require Fleet Premium. |
| Expected lifetime | Supported in current Fleet releases. Any plan to retire or replace the feature is announced in advance in release notes. |
| Impact assessment summary | Low impact. Main benefit: faster, clearer guidance for device users. Main risk: inaccurate or unsuitable resolution steps reaching device users, who are usually employees and can include students where schools manage devices. Administrator review before saving reduces this risk. The feature makes no decisions about individuals and treats no group differently. |

### Other AI features

Other AI features will be listed here, with the same information, before they are released.

## Updates

Changes to AI features, including model changes and changes to what each feature can do, are announced in Fleet's [release notes](https://fleetdm.com/releases). Fleet re-tests each AI feature after every model or prompt change before it reaches customers. This page is checked and updated at each release that changes an AI feature.

## Report an AI concern

If you believe a Fleet AI feature has produced harmful, unsafe, biased, or incorrect output, or has been used in a way that caused harm, tell us. Customers, device users, researchers, and members of the public can all report.

Email security at fleetdm.com with the subject line "AI concern". Include the feature, what happened, when, and any screenshots or examples. Do not send passwords or other secrets.

We acknowledge every report within 2 business days, investigate it, and tell you the outcome where we can identify you.

## Further reading

- [Fleet's AI-assisted policy descriptions and resolutions](https://fleetdm.com/guides/fleet-ai-assisted-policy-descriptions-and-resolutions)
- [Product support](https://fleetdm.com/support)

<meta name="articleTitle" value="AI in Fleet">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-11-04">
<meta name="category" value="guides">
<meta name="description" value="Which Fleet features use AI, what data they send and to whom, how to turn them off, and how to report a concern.">
