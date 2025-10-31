# Labels

![Managing labels in Fleet](../website/assets/images/articles/managing-labels-in-fleet-1600x900@2x.png)

In Fleet, you can use labels to scope [software](https://fleetdm.com/guides/deploy-software-packages), [policies](https://fleetdm.com/securing/what-are-fleet-policies), [queries](https://fleetdm.com/guides/queries), and [configuration profiles](https://fleetdm.com/guides/custom-os-settings) for specific hosts, and filter the hosts view.

Labels can be one of the following types:
- **Dynamic**: A query-based label applied to any host that returns a result for the label’s query.
- **Manual**: A list of selected hosts.
- **Host vitals**: All hosts that have a specific host vital get this label applied. Currently only supported for IdP host vitals (groups and department) on macOS, iOS, iPadOS, and Android hosts.

To add or edit a label in Fleet, select the avatar on the right side of the top navigation and select **Labels**.

You can also manage labels via [Fleet's API](https://fleetdm.com/docs/rest-api/rest-api#labels) or [best practice GitOps](https://fleetdm.com/docs/configuration/yaml-files#labels).

> For dynamic labels, if you want to change the query or platform, you must delete the existing label and create a new one.

<meta name="articleTitle" value="Labels in Fleet">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2025-10-24">
<meta name="articleImageUrl" value="../website/assets/images/articles/managing-labels-in-fleet-1600x900@2x.png">
<meta name="description" value="Using labels in the Fleet">
