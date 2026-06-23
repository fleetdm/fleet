# Self-hosting fleetd for automatic enrollment

You can configure Fleet to point towards a self-hosted server for fleetd installation during enrollment.

This can be configured to ensure that host enrollments are successful in environments with strict network access

## Prerequisites

* Web server e.g. Apache, Nginx

> To ensure host enrollment is successful for out-of-the-box Fleet deployments using the default download URL, Fleet is configured to fetch from the `/stable/` directory for any given URL.

## Directory structure

The files used for fleetd installation during automatic enrollment are:

* [fleetd-base.msi](https://download.fleetdm.com/stable/fleetd-base.msi) - Used for Windows enrollments
* [fleetd-base.pkg](https://download.fleetdm.com/stable/fleetd-base.pkg) - Used for MacOS enrollments
* [fleetd-base-manifest.plist](https://download.fleetdm.com/stable/fleetd-base-manifest.plist) - Used for MacOS enrollments
    * *Contains sha256 hash & URL for fleetd-base.pkg.*
* [meta.json](https://download.fleetdm.com/stable/meta.json) - Metadata file used for Windows enrollments
    * *Contains URLs for all files listed above, sha256 hashes for the .msi & .pkg installers, and a version timestamp.*

> Certain values inside these files will require customization for your environment e.g. URLs, hashes


## Configure

You can define the fleetd base URL through the Fleet UI:

1. Navigate to **Settings > Organization Settings > Advanced Options**
2. Find **Fleet Agent base URL**, enter your desired server URL and select **Save**

<meta name="articleTitle" value="Self-hosting fleetd for automatic enrollment">
<meta name="authorFullName" value="William Bowman">
<meta name="authorGitHubUsername" value="William-TecNQ">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-06-23">
<meta name="description" value="A guide to self-hosting fleetd for automatic enrollments into Fleet.">

