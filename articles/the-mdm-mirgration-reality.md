# The MDM migration reality: easier, but not easy

Recent macOS and iOS/iPadOS updates, along with improvements in Apple Business Manager (ABM), have made MDM migration less disruptive. Previously, migrating iOS and iPadOS devices to a new MDM server required a complete factory reset, disrupting end users. Now, these mobile devices can be reassigned to a new MDM server without wiping, bringing them in line with macOS devices which already supported non-destructive migration.

While macOS avoided the factory reset requirement, migration prior to macOS 26 still required careful coordination and communication with end users and, in some cases, custom workflows to ease the transition.

Even with these improvements, it's crucial to understand the reality: migration remains complex. The device reassignment primarily moves the device enrollment from one MDM server to another. It does **not** automatically transfer your entire management schema.

---

## The hidden hurdles

It’s easy to think that device reassignment equals *“done.”* It doesn’t. After enrolling into the new MDM, the device has none of your previous configurations. Key elements must be rebuilt:

- **Configurations and settings:** Your thousands of granular settings, configuration profiles, and device restrictions from the old MDM are not moved over. They must all be recreated and applied in the new environment.  
- **App deployment and scoping:** Application deployments, including custom settings and volume purchase program (VPP) licenses, need to be re-established. The scoping must be replicated in the new system.  
- **Reporting and integrations:** Existing custom reports, compliance checks, or integrations with other business systems (like your Identity Provider or ticketing system) will be broken and need to be reconfigured.  
- **Technical debt:** You’ll want to go through your old policies, remove outdated settings, and start with a clean, efficient deployment. This requires auditing the old system before migrating.

---

## Key steps for a successful migration

To navigate this complexity and ensure a smooth, low-disruption transition, follow a structured **brick-by-brick** approach:

1. **Audit and rationalize:**  
   Document every setting, policy, app, and scope within your old MDM. Use this opportunity to shed unnecessary policies and define your desired end state for the new MDM.

2. **Staging and pre-configuration:**  
   Your new MDM must be fully configured. Recreate all necessary profiles, policies, applications, and integration links. Use test devices to validate that a fresh enrollment achieves your desired state.

3. **Controlled rollout & communication:**  
   Start with a pilot group and communicate clearly with end users about what’s changing, what to expect, and any required actions.

4. **Verification and post-migration:**  
   After the device re-enrolls, confirm that the new policies and applications are successfully applied.

---

## Support when you need it

Migration gets easier with Apple’s new tools, but it still takes real work. If you’re planning a move and want help working through the details, we’ve done this many times and are happy to share what we’ve learned.

Our **brick-by-brick** approach focuses on auditing your current setup, carefully staging the new one, and keeping the transition clean and predictable.

If you’re considering a migration and want to avoid surprises, let us know. We can share more about how the brick-by-brick method works.

<meta name="articleTitle" value="The MDM migration reality: easier, but not easy">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="articles">
<meta name="publishedOn" value="2025-11-26">
<meta name="description" value="MDM migration is less disruptive than before, but still demands careful planning.">
