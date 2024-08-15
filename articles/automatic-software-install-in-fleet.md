# Automatic installation of software on hosts

![Automatic installation of software on hosts](../website/assets/images/articles/automatic-software-install-in-fleet-731x738@2x.png)

Fleet has the ability to automatically and remotly install software on hosts based on labels applied to them. 
This guide will walk you through the process of configuring fleet for automatic installation of
software on specific hosts using pre uploded installation images and using label based install
policy. You'll learn how to configure and use this feature, as well as understand how the underlying
mechanism works.

Fleet allows its users to upload trusted software installation files to be installed and used on hosts.
When adding a new software, it is possible to configure this specific software to be automatically
installed on:
- All hosts
- All hosts excluding specific label/s
- Only on hosts with specific label/s


## Step-by-Step Instructions

- In Fleet click the upper menu `Software` tab.
- On top left corner drop down menu, select a team (or `No team`)
- On top right corner click `Add software`.
- In the newly openned modal add a new trusted binary.
- In the `Install` dropdown menu select "Automatic".
- Under the `Target` section select `All hosts` or `Custom`.
- If `Custom` was selected, right below it will appear a dropdown that will allow `Include` or `Exclude` of specific label/s. A list of labels will be shown to configure the proper operation


In the example below Install on all hosts EXCEPT those with 'Okta - Engineering' label

![Install on all hosts EXCEPT those with 'Okta - Engineering' label](../website/assets/images/articles/automatic-software-install-in-fleet-729x924@2x.png)


Current supported installation files, manual upload of these formats:
- Macos: .pkg
- Windows: .msi, .exe
- Linux: .deb

Coming soon:
- Ability to auto install from Apple VPP.
- Install on iOS and iPadOS

## Using fleet API/GitOps:
The same result can be achieved by using Fleet API, Fleetctl ot GitOps.

Example:
TODO - add API usage
TODO - Add Fleetctl usage
TODO - link to relevant GitOps 



## How does it work?

* After configuring Fleet to auto-install a specific software the rest will be done automatically.
* TODO Sharon: Describe the workflow 

### Detailed flowchart
![Flowchart](../website/assets/images/articles/automatic-software-install-in-fleet-13762x2946@2x.png)


## Prerequisites

* Fleet premium. 
* Admin permissions for all three services above.





## Additional Information

* Add Demo video if exists
* Add other docs

## Conclusion

By following this guide, you’ve learned how to effectively automate the mitigation of failing policies on hosts using Fleet's scheduled maintenance window feature. This approach not only ensures that your fleet remains compliant and secure but also respects your users' time by scheduling maintenance at convenient intervals.

Leveraging Fleet’s integration with Google Calendar and webhooks, you can streamline the process of addressing policy failures, reducing manual oversight, and minimizing disruptions to your users' workflows. Whether it's updating critical software, ensuring security measures like disk encryption, or maintaining compliance with organizational policies, this feature provides a powerful tool to maintain the health and security of your infrastructure.

By automating these tasks, you can focus on more strategic initiatives, confident that Fleet is helping to keep your systems secure and your users satisfied. For more detailed insights into how this feature works or to explore additional customization options, refer to the additional resources provided.

Implementing scheduled mitigations not only enhances operational efficiency but also contributes to a proactive security posture, helping your organization stay ahead of potential risks.


<meta name="articleTitle" value="Automatic installation of software on hosts">
<meta name="authorFullName" value="Sharon Katz">
<meta name="authorGitHubUsername" value="sharon-fdm">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-08-15">
<meta name="articleImageUrl" value="../website/assets/images/articles/automatic-software-install-in-fleet-731x738@2x.png">
<meta name="description" value="A guide to workflows using automatic software installation in Fleet.">
