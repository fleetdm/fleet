### You know Jamf. You're evaluating Fleet.

* Jamf admins are looking at Fleet for real-time device data, GitOps workflows, and cross-platform management from one console.
* Your existing knowledge transfers. Smart Groups, Extension Attributes, policies, and configuration profiles all have a Fleet equivalent.
* Apple's native MDM migration moves devices from Jamf to Fleet without wiping them.

### What changes, and what stays the same.

This guide maps what you already know in Jamf to how Fleet does it, then gives you a phased migration plan.

### What you'll learn

You'll learn how Jamf and Fleet terminology line up, where configuration profiles, Smart Groups, Extension Attributes, and policies translate one-to-one, and where they don't. You'll see what GitOps adds to your workflow, and you'll get a five-phase migration plan built around Apple's native MDM migration, so devices move without a wipe and without losing data.

### Chapter list

1. **Jamf to Fleet: the terminology translation**
    * A side-by-side mapping of every core concept, from agents and sites to policies, patches, and APIs.
2. **Enrollment**
    * How Fleet's Setup Experience compares to Jamf PreStage Enrollment, step by step.
3. **Configuration profiles**
    * What carries over directly, what changes, and how Jamf payload variables map to Fleet variables.
4. **Grouping devices**
    * How Fleet's fleets and labels replace Jamf Sites and Smart Groups, with osquery-powered real-time membership.
5. **Compliance checks**
    * Why Fleet policies are direct pass/fail tests, not group membership, and how automatic remediation works when a check fails.
6. **Software deployment: packages, Self Service, and patching**
    * Fleet-maintained apps, custom packages, AutoPkg support, and how Fleet Desktop compares to Jamf Self Service.
7. **osquery for beginners**
    * The osquery mental model for Jamf admins, Extension Attribute translations, and 15 starter queries.
8. **GitOps: your new version control workflow**
    * What moves into Git, how rollbacks work, and when to use GitOps versus the Fleet UI.
9. **The migration plan**
    * A five-phase rollout built on Apple's native MDM migration, plus a readiness checklist.

<meta name="articleTitle" value="The Mac admin's guide to switching from Jamf to Fleet">

<meta name="authorFullName" value="n/a">
<meta name="authorGitHubUsername" value="fleet-release">

<meta name="category" value="whitepaper">
<meta name="publishedOn" value="2026-06-11">
<meta name="description" value="Everything you know about Jamf, mapped to how Fleet does it. A guide for Mac admins evaluating or migrating from Jamf Pro to Fleet.">

<meta name="articleImageUrl" value="../website/assets/images/articles/mac-admins-guide-to-switching-from-jamf-to-fleet-cover-image-504x336@2x.png">
<meta name="whitepaperFilename" value="fleet-jamf-migration-guide.pdf">
<meta name="formHeadline" value="Learn how to switch from Jamf to Fleet">

<meta name="introductionTextBlockOne" value="If you've spent years mastering Jamf Pro, your knowledge transfers. Smart Groups, Extension Attributes, policies, and configuration profiles all have a Fleet equivalent.">
<meta name="introductionTextBlockTwo" value="This guide doesn't start from zero. It maps what you already know in Jamf's language to how Fleet does it, then gives you a phased migration plan.">
