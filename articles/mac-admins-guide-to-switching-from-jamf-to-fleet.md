### Everything you know about Jamf, mapped to how Fleet does it

This guide is for Mac admins with years of Jamf Pro experience who want a fast, accurate path to Fleet. It is not for teams new to MDM.

Inside, you will find a Jamf-to-Fleet terminology table, side-by-side workflow comparisons, and a phased migration plan. The plan runs Fleet alongside Jamf, so you can build and test before you move a single device.

### What you'll learn

You will learn how Jamf concepts map to Fleet, one at a time. PreStage Enrollment becomes Setup Experience. Smart Groups become labels and policies. Extension Attributes become osquery queries. Jamf policies split into Fleet scripts and Fleet compliance checks.

You will also get a five-phase migration plan. It runs Fleet next to Jamf, migrates devices in cohorts using Apple's native MDM migration, and retires Jamf only after a final validation period.

### Chapter list

- **Jamf to Fleet: the terminology translation**
  * A side-by-side table that maps core Jamf concepts to their Fleet equivalents.
- **Enrollment**
  * How Fleet's Setup Experience lines up with Jamf PreStage Enrollment, step by step, on top of Apple ADE and ABM.
- **Configuration profiles**
  * What carries over unchanged, what differs, and how to manage profiles with GitOps.
- **Grouping devices**
  * How Jamf Sites and Smart Groups map to Fleet's fleets and labels.
- **Compliance checks**
  * How a Fleet policy differs from a Jamf policy, and how osquery tests compliance directly with a pass or fail.
- **Software deployment: packages, Self Service, and patching**
  * Package formats, Fleet-maintained apps, Self Service, and patching across macOS, Windows, and Linux.
- **osquery for beginners**
  * The osquery mental model, an Extension Attribute translation, and your first 15 reports.
- **GitOps: your new version control workflow**
  * Version, review, and roll back your Fleet configuration in Git.
- **The migration plan**
  * A five-phase plan, how Apple's no-wipe MDM migration works, and a migration readiness checklist.



<meta name="articleTitle" value="The Mac admin's guide to switching from Jamf to Fleet">

<meta name="authorFullName" value="n/a">
<meta name="authorGitHubUsername" value="fleet-release">

<meta name="category" value="whitepaper">
<meta name="publishedOn" value="2026-05-29">
<meta name="description" value="A guide for Mac admins moving from Jamf to Fleet. Map Jamf terms to Fleet equivalents, then follow a phased, no-wipe migration plan.">

<meta name="articleImageUrl" value="../website/assets/images/articles/mac-admins-guide-to-switching-from-jamf-to-fleet-cover-image-504x336@2x.png">
<meta name="whitepaperFilename" value="mac-admins-guide-to-switching-from-jamf-to-fleet.pdf">
<meta name="ungated" value="true">

<meta name="introductionTextBlockOne" value="You know Jamf Pro. You know policies, Smart Groups, and Extension Attributes. Now you are evaluating Fleet, or you have already decided to migrate.">
<meta name="introductionTextBlockTwo" value="This guide does not start from zero. It assumes you know MDM, and it maps what you already do in Jamf to how Fleet does the same work.">
