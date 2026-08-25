# How Fleet uses GitHub

Fleet doesn't do things the same way as other companies.

How we use GitHub is different to newcomers, customers and employees alike. This article hopes to show folks the Fleet way.

We use a lot of GitHub features that may be familiar:

- [Labels](https://docs.github.com/en/issues/using-labels-and-milestones-to-track-work/managing-labels) are used to track what team works on an issue, its priority, and to add extra context.
- The current [status](https://docs.github.com/en/issues/planning-and-tracking-with-projects/sharing-project-updates) of the issue is best found by checking the [project](https://docs.github.com/en/issues/planning-and-tracking-with-projects/learning-about-projects/about-projects) board that it's currently assigned to.
- The [milestone](https://docs.github.com/en/issues/using-labels-and-milestones-to-track-work/about-milestones) is the proposed version that it will be included in.

All of the above information can be found at a glance in the sidebar that's located on the right side of the web page when [viewing an issue](https://docs.github.com/en/issues/tracking-your-work-with-issues/learning-about-issues/about-issues).

Issues generally fall into two categories: feature requests and bug reports.


## Feature requests

New customer requests go to the Drafting inbox where they are [triaged](https://fleetdm.com/handbook/product-design#triage-new-requests) by the Head of Product Design, and assigned to the appropriate [product group](https://fleetdm.com/handbook/company/product-groups#current-product-groups). The product group's Product Designer reviews these requests and determines if they qualify to be "unpacked" in a process called [Unpacking the why](https://fleetdm.com/handbook/product-design#unpacking-the-why), where the request is reviewed with a former IT admin to get a better understanding of it. This is the most common time that the original comment on the request will be updated, to clarify the intent of the request.

Features are then prioritized during [Feature fest](https://fleetdm.com/handbook/company/product-groups#feature-fest), which occurs approximately every three weeks. If a feature is selected for prioritization, it will move into the Product Design drafting process, where at least one [sub-issue](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/browsing-sub-issues) called a story will be created. This story will receive a milestone, and wireframes will be designed and reviewed before handing it off to Engineering. A milestone is a proposed release version along with a proposed release date, both of which are subject to change.

If a feature request is large and requires changes in multiple areas of the product, multiple sub-issues are created and attached to it. Check for any sub-issues under the request and its stories for milestones, as the parent issue sometimes doesn't have a milestone, but the sub-issues will; or the sub-issues will have a different milestone than the parent issue.

After wireframes have been approved, work will begin on implementing and testing the new feature. After the code has been reviewed and approved, [QA](https://fleetdm.com/handbook/company/product-groups#quality) confirms the change works as intended on a live test instance.

You can learn more about [the statuses](https://fleetdm.com/handbook/company/product-groups#board-columns) of our GitHub projects and [how issues move](https://fleetdm.com/handbook/company/product-groups#how-issues-move) through them in our handbook.

Feature requests from customers are left open until they have gone through [Confirm and celebrate](https://fleetdm.com/handbook/product-design#confirm-and-celebrate), and are confirmed as fulfilled by the Manager of Customer Support and Solutions Architecture. The account's CSM then informs the customer and closes out the request after verifying with the customer.

Sometimes feature requests get converted to [quick wins](https://fleetdm.com/handbook/company/product-groups#work-items), which go through a simplified, quicker process than full feature requests that don't get unpacked and don't go through Feature fest.


## Bug reports

Bugs follow a simpler process and do not (typically) have sub-issues. Bugs are [triaged during standup](https://fleetdm.com/handbook/company/product-groups#daily-standup-30-minutes), [assigned a priority](https://fleetdm.com/handbook/company/product-groups#high-priority-user-stories-and-bugs) if needed, worked on [in order](https://fleetdm.com/handbook/company/product-groups#bug-prioritization), and brought through [a process](https://fleetdm.com/handbook/company/product-groups#inbox) where they are reproduced if necessary, reviewed and resolved by Engineering with a pull request, and then verified by QA. If prioritized, they similarly receive a milestone.


## FAQ

**When will this feature be released? When will this bug be fixed?**

Check the milestone (in the sidebar) of the original issue, and any sub-issues. If there isn't a milestone, the issue does not have a planned release date. Please note that the release version and date are subject to change. You can check with your Customer Success Manager (CSM) to verify the most accurate status of your request.

**Are there any updates on this issue?**

New comments and status updates are public and show in the GitHub feed for the original issue, and any sub-issues. This is the best place to check for any updates on the issue. The status is best seen under the Projects section in the sidebar on the right.

**Why is the original post being edited?**

Issues are considered a living document while being worked on, and it's critical that the top post contains clear and up-to-date information for the issue. When designers, engineers, and QA review an issue, they often check the original post as the source of truth. Comments and discussion still happen under the issue, but any relevant updates should be moved up into the original post.


<meta name="articleTitle" value="How Fleet uses GitHub">
<meta name="authorFullName" value="Steven Palmesano">
<meta name="authorGitHubUsername" value="spalmesano0">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-08-25">
<meta name="description" value="Fleet doesn't do things the same way as other companies.">
