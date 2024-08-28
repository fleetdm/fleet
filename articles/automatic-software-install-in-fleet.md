# Automatic conditional installation of software on hosts

TODO: Add general image here

Fleet has the ability to automatically and remotly install software on hosts upon a specific policy failure, programmed in advance. 
This guide will walk you through the process of configuring fleet for automatic installation of
software on hosts using pre uploded installation images and based on pre programmed policies. 
You'll learn how to configure and use this feature, as well as understand how the underlying
mechanism works.

Fleet allows its users to upload trusted software installation files to be installed and used on hosts.
This installation could be conditioned on a failure of a specific Fleet Policy.

A very simple example will be this: 
Install a patch on all MacOS hosts with version lower than 14.2.1.
You will create a policy that assures hosts are equal or higher than 14.2.1 
Like this: 
- ```SELECT 1 where exists (SELECT version FROM os_version WHERE version >= "14.2.1");```

Then all hosts failing this policy will have the patch programmed to be installed.

Of course this feature holds a strong and flexible way to install software based on any chosen policy.
See step by step section below.

## Step-by-Step Instructions

1. Add any software to be available for installation. Follow this document with instruction how to
   do it.
   TODO: Sharon - add link to SW install doc.

2. In Fleet, add a policy that failure to pass it will trigger the required installation.
  Go to Policies tab --> Press the top right "Add policy" button. --> Click "create your own policy"
  --> Save --> Fill details in the Save modal and Save.

![Add New Policy](../website/assets/images/articles/automatic-software-install-add-new-policy.png)

3. Open Manage Automations: Policies Tab --> top right "Manage automations" --> "Install software".

![Plocies Manage](../website/assets/images/articles/automatic-software-install-policies-manage.png)

4. Select (click th echeck box of) your newly created policy. To the right of it select from the
   drop-down list the software you would like to be installed upon failure of this policy.

![Install Software Modal](../website/assets/images/articles/automatic-software-install-install-software.png)




```
Current supported installation files, manual upload of these formats:
- Macos: .pkg
- Windows: .msi, .exe
- Linux: .deb

Coming soon:
- Ability to auto install from App store (VPP).
- Install on iOS and iPadOS
```

## Using fleet API/GitOps:
The same result can be achieved by using Fleet API, Fleetctl ot GitOps.

Example:
TODO - add API usage
TODO - Add Fleetctl usage
TODO - link to relevant GitOps 



## How does it work?

* After configuring Fleet to auto-install a specific software the rest will be done automatically.
* TODO Sharon: Describe the workflow 

![Flowchart](../website/assets/images/articles/automatic-software-install-workflow.png)
*Detailed flowchart*

## Prerequisites

* Fleet premium. 
* Admin permissions for all three services above.





## Additional Information

* Add Demo video if exists
* Add other docs

## Conclusion

Software deployment can be time consumng and risky if not done the proper way.
This guide presents Fleet's ability to mass deploy software to your fleet in a way that is both
simple and safe. Starting with uploading a trusted installer and ending with deploying it to the
proper set of machines.

Leveraging Fleet’s ability to install and upgrade software on your hosts, you can streamline the
process of controlling your hosts, replacing old versions of software and having the up-to-date info
on what's installed on your fleet.

By automating software deployment, you can gain better control on what's installed on your machines
and have a better ability to upgrade software versions with known issues.


<meta name="articleTitle" value="Automatic installation of software on hosts">
<meta name="authorFullName" value="Sharon Katz">
<meta name="authorGitHubUsername" value="sharon-fdm">
<meta name="category" value="guides">
<meta name="publishedOn" value="2024-08-15">
<meta name="articleImageUrl" value="../website/assets/images/articles/automatic-software-install-in-fleet-731x738@2x.png">
<meta name="description" value="A guide to workflows using automatic software installation in Fleet.">
