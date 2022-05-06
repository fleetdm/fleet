# Fleet 4.14.0 adds beta support for automatic ticket creation and improves the live query experience.

![Fleet 4.14.0](/images/articles/4-14-0-cover-1600x900@2x.png)

Fleet 4.14.0 has arrived. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.14.0) or continue reading to get the highlights.

For update instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights
- Jira and Zendesk integrations
- Improved live query experience
- Postman Collection

## Jira and Zendesk integrations
**Available to all Fleet users.**

![Jira and Zendesk integrations](/images/articles/jira-integration-1600x900@2x.png)

You can now configure Fleet to automatically create a Jira or Zendesk ticket when a new vulnerability (CVE) is detected on your hosts. No need to create tickets or spend time configuring a webhook manually. 

## Improved live query experience
**Available to all Fleet users.**

![Improved live query experience](/images/articles/show-query-1600x900@2x.png)

We added a “Show query” option to the live query results view. You can now double-check the syntax you used and compare that to your results without leaving the current view.

## Postman Collection
**Available to all Fleet users.**

![Postman Collection](/images/articles/postman-collection-1600x900@2x.png)

Fleet users can easily interact with Fleet's API routes using the new Postman Collection. Build and test integrations for running live queries, carving files, managing policies, and more!

## More new features, improvements, and bug fixes

In 4.14.0, we also:

- fixed deprecation warning message on `fleetctl package` for deb/rpm.
- added support for using a custom TUF server with `fleetctl preview`.
- made the duration values returned by `fleetctl` more human-friendly to read.
- improved error messaging for `fleetctl query`.
- improved the “Organizational settings” flow in the Fleet UI.
- improved “empty state” messaging in the Fleet UI.
- added "last opened at" information for software to the “host details” API endpoint (macOS only).
- added “optional” to hints for appropriate fields when creating new queries and policies in the Fleet UI. 
- fixed a bug with SAML SSO authentication.
- fixed a bug affecting “name” display for scheduled queries in the Fleet UI. 
- fixed a bug that caused panic errors when running `fleet –debug`. 
- fixed a bug affecting queries containing the `@` symbol.
- removed use of JSON_ARRAYAGG in SQL queries to support newer versions of MySQL and AWS RDS Aurora.
- added `osquery.min_software_last_opened_at_diff` configuration option.

---

### Ready to update?

Visit our [Update guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.14.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Kathy Satterlee">
<meta name="authorGitHubUsername" value="ksatter">
<meta name="publishedOn" value="2022-05-06">
<meta name="articleTitle" value="Fleet 4.14.0 adds beta support for automatic ticket creation and improves the live query experience.">
<meta name="articleImageUrl" value="/images/articles/4-14-0-cover-1600x900@2x.png">