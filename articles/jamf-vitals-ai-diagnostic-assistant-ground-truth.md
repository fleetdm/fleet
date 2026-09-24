# Vitals can tell an employee what's wrong. It can't show them why.

*Jamf's new AI diagnostic assistant answers a laptop's symptoms in plain language and offers a one-click fix. The question worth asking before comparing it to what's already in Fleet is what it read to get there.*

## Key takeaways

- **Jamf is putting a chat interface between employees and their own device data.** Vitals, announced at JNUC 2026 and targeted for public beta in Q1 2027, lets an employee ask something like "why are my video calls lagging" inside Self Service+ and get an explanation plus a fix they can approve themselves.
- **The new AI governance suite audits the tools, not the answers.** AI Policies, Model Insights, and Usage Intelligence track which AI tools run on managed Macs and what they cost. None of the three audits what a diagnostic feature like Vitals reads before it answers.
- **Fleet already ships a grounded version of the same idea, on the admin side.** Fleet's AI-assisted policy descriptions and resolutions are written by reading a policy's actual SQL, so the explanation an end user sees matches the check that's actually running, not a generic template.
- **A conversational front end is not the same claim as a grounded one.** An assistant can sound confident about a laptop's overheating fan or a full disk without reading anything more current than a cached diagnostic snapshot. Whether it's current, cross-platform, and inspectable is a separate question from whether it's fluent.
- **The comparison buyers should make is about the data underneath, not the chat window.** As MDM vendors ship some version of an AI assistant this year, the differentiator is whether you can check what it's reading, not how polished the reply sounds.

<a purpose="cta-button" href="https://fleetdm.com/policies">Explore Fleet policies</a>

At JNUC 2026 in Kansas City, Jamf introduced Vitals, an AI-powered diagnostic assistant built into Self Service+. An employee can ask a plain-language question about a problem on their Mac, "why is this fan so loud" or "why did my battery die overnight," and get back an explanation along with a recommended fix they can approve with one click. Jamf paired it with an expanded AI governance suite: AI Policies for compliance guardrails, Model Insights for token and cost reporting by device and provider, and Usage Intelligence for categorizing what AI work is actually happening across the fleet.

That's a real, useful shift for IT teams tired of routine tickets. It's also the second or third MDM vendor to ship an AI assistant this year, which makes the interesting question no longer "does it have AI" but "what is it reading when it answers."

## What the governance suite governs, and what it doesn't

AI Policies, Model Insights, and Usage Intelligence are all aimed at the AI tools running on a fleet: which ones are approved, what they cost per device and provider, and what kind of work they're doing. That's governance over the AI layer itself, a real and growing need as employees adopt AI tools faster than IT can approve them.

None of the three, based on what Jamf has published, audits the diagnostic assistant's own inputs. Whether Vitals answers "why are my video calls lagging" from a live read of the device or from a cached snapshot isn't a question the governance suite is built to answer, because it's watching AI usage, not grounding a specific feature's output in specific device state.

## Fleet already grounds this exact kind of answer

Fleet's AI-assisted policy descriptions and resolutions solve a narrower version of the same problem today. When an admin creates a policy, Fleet's AI drafts the user-facing description and resolution by reading the policy's actual SQL rather than writing generic text, so the explanation an end user sees in Fleet Desktop, or in a maintenance window calendar event, describes the specific check that's failing on their machine. An admin can edit the wording, but the AI's starting point is the real query, not a template pulled from a knowledge base.

That's not the same shape of product as an employee-facing chat assistant. It's the same underlying discipline: an AI-generated answer is only as good as what it's allowed to read, and Fleet ties the generated text to the actual check running against the actual device.

## The comparison that actually matters

As diagnostic assistants show up across MDM vendors, judging them by how conversational they sound skips the part that determines whether you can trust the output. An answer grounded in a live, cross-platform, inspectable read of the device is a different product than an answer grounded in whatever the model was trained on or whatever got cached at the last sync, even when both come back as a friendly paragraph of text.

That's true whether the AI is explaining a failing policy or diagnosing a laggy video call. The fix on either side might be the same one-click action, but only one of them can show you the exact data it read to get there.

## See it live

- **Get a demo** to see Fleet's AI-assisted policies grounded in your own device data: [fleetdm.com/contact](https://fleetdm.com/contact)
- **Read how it works**: Fleet's AI-assisted policy descriptions and resolutions: [fleetdm.com/articles/fleet-ai-assisted-policy-descriptions-and-resolutions](https://fleetdm.com/articles/fleet-ai-assisted-policy-descriptions-and-resolutions)

## Sources

- Jamf, [JNUC 2026 keynote recap: Power Up with Jamf](https://www.jamf.com/blog/jnuc-2026-keynote/).
- Business Wire (via FinancialContent), [Jamf Takes Apple Management into the AI Era at JNUC 2026](https://www.financialcontent.com/article/bizwire-2026-9-23-jamf-takes-apple-management-into-the-ai-era-at-jnuc-2026).

<meta name="articleTitle" value="Vitals can tell an employee what's wrong. It can't show them why.">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-09-24">
<meta name="description" value="Jamf's Vitals AI assistant explains device problems in Self Service+. Fleet already grounds AI-generated answers in the real check behind them.">
