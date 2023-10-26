# Fleet 4.39.0 | Sonoma support, script library, query reports.

![Fleet 4.39.0](../website/assets/images/articles/fleet-4.39.0-1600x900@2x.png)

Fleet 4.39.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.39.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.


## Highlights

* macOS Sonoma support
* Script library
* Scheduled query reports


### macOS Sonoma support

Fleet has incorporated support for Apple's latest macOS release, Sonoma. Recognizing the profound shifts in device management introduced with macOS Sonoma, Fleet is keenly working towards assimilating these changes in future updates. Upholding the values of Openness and Empathy, we're not just adapting to these innovations but are also inviting user feedback to ensure that we prioritize the features and enhancements most crucial to our users.

One of the standout features of macOS Sonoma is the introduction of declarative device management (DDM). DDM, an enhancement to the Mobile Device Management (MDM) protocol, is a paradigm shift, offering a more secure and efficient method for macOS device administration. For a deeper dive into DDM and its transformative potential, our dedicated article, "[Embracing the Future: Declarative Device Management](https://fleetdm.com/announcements/embracing-the-future-declarative-device-management)," provides comprehensive insights.


### Script library

_Available in Fleet Premium._

Fleet has enabled the storage and management of saved scripts for macOS. Within the Fleet web user interface, command-line interface(CLI), and API, administrators and maintainers can add, delete, and download a saved script, whether affiliated with a global team or a specific team. 

Beyond just management, authorized users have the capability to execute a saved script on designated hosts. Users can easily view the status of each script and access the most recent script's output, limited to the last 10,000 characters. Importantly, actions related to these scripts are monitored and documented in the activity feed. This enhancement embodies Fleet's values of Empathy and Ownership, empowering admins and users with streamlined script management and clear visibility into script outputs and actions, promoting a more efficient workflow.


### Scheduled query reports

The latest update to Fleet introduces the storage of results from scheduled queries as a report. Each scheduled query can now retain up to 1000 results. For any scheduled query that has results fewer than 1000, Fleet will update these results as new data is received from hosts. 

We’re excited about this powerful feature. Fleet’s query reports allow you to collect and store query data for your hosts, whether online or offline. Storing more data may mean more load on your Fleet database. As you turn on query reports, we recommend watching the DB to see if you need to scale it up. You can always turn off query reports later if you don’t want to scale up.

Users also have the option to disable reports for individual queries using the discard_data field. Saved query results can be disabled in the global configuration. This enhancement fosters Fleet's value of Ownership by providing administrators with greater control over their data and Objectivity by offering clear data storage options and enhanced validation of osquery result logs.




## New features, improvements, and bug fixes




## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.39.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-10-26">
<meta name="articleTitle" value="Fleet 4.39.0 | Sonoma support, script library, query reports.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.39.0-1600x900@2x.png">
