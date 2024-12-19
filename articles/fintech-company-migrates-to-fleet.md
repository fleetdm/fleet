# American financial services company migrates to Fleet for MDM and next-gen change management

<div purpose="attribution-quote">

“I don't want one bad actor to brick my fleet, I want them to make a pull request first.”
**— Client Platform Engineering Manager**
</div>

## Challenge

A prominent financial services company encountered substantial challenges with its existing device management solution. The platform demanded excessive resources and time for maintenance while limiting its ability to implement automated GitOps workflows, a focus of its operational strategy. The previous migration experience had been arduous, needing to support a fleet comprising 2,700 devices across macOS, Windows, and iOS devices. They required a scalable and secure platform that could support its configuration-as-code philosophy and reallocate resources to more strategic initiatives.

## Solution

They selected Fleet as its new device management platform to unify its device ecosystem. Fleet's next-gen GitOps capabilities aligned with their commitment to configuration as code, enabling seamless integration with their existing automation workflows. Fleet's support and direct database schema [migration tool](https://github.com/fleetdm/fleet/blob/d563d09baca642d5e4f910759b71619333b500b9/tools/mdm/migration/micromdm/touchless/README.md) facilitated the migration process, ensuring a smooth transition. Additionally, [Fleet's open API](https://fleetdm.com/docs/rest-api/rest-api) and advanced features, such as live query execution, real-time insights, and configurable logging pipelines, provide their teams with real-time visibility and control over their endpoints.

## Results

<div purpose="checklist">

Unified device management platform across macOS, Windows, and iOS.

Introduced new infrastructure as code workflows.

Fewer resources are spent configuring device management.

Smooth and seamless migration.
</div>


## Their story

The leading financial services company dedicated to democratizing finance for all. By leveraging cutting-edge technology, they empower millions of users to invest and manage their finances with ease and confidence. The company sought a new device management solution that wouldn’t require strenuous time or resources to manage. They were looking for a more full-featured MDM that enabled customization through configuration as code.

Specifically, they were looking for:

- Next-gen change management and open-source flexibility
- Increased efficiency
- An easy migration path
- Improved support and feature access

### Next-gen change management and open-source flexibility

Fleet is [open-source](https://fleetdm.com/handbook/company/why-this-way#why-open-source), allowing engineering teams to audit, customize, and extend the platform as needed alongside [infrastructure-as-code](https://github.com/fleetdm/fleet-gitops) workflows. This makes device management more agile and automated while reducing errors through peer review.

### Eliminate tool overlap and increase efficiency

They were able to replace multiple legacy tools and consolidate the management of thousands of macOS, Windows, and iOS devices into a single platform. This led to a significant reduction in resources and time spent maintaining their previous tools, allowing efforts to be reallocated towards innovation and development.


### Easy migration path

Fleet ensures a smooth transition with minimal disruption. Migrations are directly assisted by Fleet’s [best-in-class support](https://fleetdm.com/support) teams and built-in [migration tools](https://github.com/fleetdm/fleet/tree/main/tools/mdm/migration).

### Improved support and feature access

Fleet has a three-week release schedule that quickly rolls out new features like automated software updates, VPP app support, and [policy-based scripts](https://fleetdm.com/guides/policy-automation-run-script). Faster rollouts and best-in-class support from Fleet enable them to stay ahead of their device management needs.


## Conclusion

The migration to [Fleet Device Management](https://fleetdm.com/device-management) exemplifies the fintech company’s dedication to leveraging advanced, flexible, and secure tools to support its expansive infrastructure. Fleet’s comprehensive feature set, combined with its commitment to security and scalability, not only addressed the limitations of legacy tools but also empowered them to increase efficiency and capabilities. This strategic move positions them to continue delivering exceptional financial services while maintaining forward-thinking device management practices.

<call-to-action></call-to-action>

<meta name="category" value="announcements">
<meta name="authorGitHubUsername" value="Drew-P-drawers">
<meta name="authorFullName" value="Andrew Baker">
<meta name="publishedOn" value="2024-12-19">
<meta name="articleTitle" value="American financial services company migrates to Fleet for MDM and next-gen change management">
<meta name="description" value="American financial services company migrates to Fleet for MDM and next-gen change management">
<meta name="showOnTestimonialsPageWithEmoji" value="🪟">
