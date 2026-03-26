# Go-To-Market Architecture


## Automation


### LinkedIn comments from tracked posts

We track certian social posts from the [LinkedIn company page](https://www.linkedin.com/company/fleetdm/) using the following workflow:

![linkedin_clay_enrichment_flow](image.png)


- LinkedIn post URL provided to Clay
- Clay enriches data
  - Sends webhook to webhooks/receive-from-clay.js
  - fleetdm.com sends a webhook to Salesforce
    - Salesforce will create/update the contact and account and creates a "Historical event"
  - Clay then sends a webook to Zapier
  - Zapier posts a message to the [_linkedin-comments-from-tracked-posts](https://fleetdm.slack.com/archives/C0AP1FM3ES2)






<meta name="maintainedBy" value="sampfluger88">
<meta name="title" value="🚂 GTM architecture">